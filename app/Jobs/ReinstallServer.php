<?php

namespace App\Jobs;

use App\Models\Server;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Http\Client\ConnectionException;
use Illuminate\Support\Facades\Crypt;
use Illuminate\Support\Facades\Http;

class ReinstallServer implements ShouldQueue
{
    use Queueable;

    public $timeout = 3600;

    protected int $serverId;

    public function __construct(int $serverId)
    {
        $this->serverId = $serverId;
    }

    public function handle(): void
    {
        $server = Server::query()->with(['location', 'game', 'tariff', 'gameVersion'])->find($this->serverId);
        if (! $server) {
            return;
        }

        if (in_array((string) $server->provisioning_status, ['running', 'failed'], true)) {
            return;
        }

        $server->update([
            'provisioning_status' => 'reinstalling',
            'provisioning_error' => null,
        ]);

        $gameCode = strtolower((string) $server->game->code);
        if ($gameCode === '') {
            $gameCode = strtolower((string) $server->game->slug);
        }
        if ($gameCode === '') {
            $gameCode = 'samp';
        }

        $archiveUrl = null;
        $steamAppId = null;
        $steamBranch = null;
        $versionName = null;
        if ($server->gameVersion) {
            $versionName = (string) $server->gameVersion->name;
            $src = (string) $server->gameVersion->source_type;
            $src = $src !== '' ? $src : 'archive';
            if ($src === 'steam') {
                $steamAppId = (int) $server->gameVersion->steam_app_id;
                $steamBranch = (string) $server->gameVersion->steam_branch;
                if ($steamAppId <= 0) {
                    $steamAppId = null;
                    $steamBranch = null;
                }
            } else {
                $archiveUrl = (string) $server->gameVersion->url;
                if (trim($archiveUrl) === '') {
                    $archiveUrl = null;
                }
            }

            if ($archiveUrl === null && $steamAppId === null) {
                $versionName = null;
            }
        }

        $host = (string) $server->location->ssh_host;
        if ($host === '') {
            $server->update([
                'provisioning_status' => 'failed',
                'provisioning_error' => 'Для локации не настроен хост демона',
                'runtime_status' => 'offline',
            ]);
            return;
        }

        $daemonPort = (int) config('services.location_daemon.port', 9201);
        $daemonToken = (string) config('services.location_daemon.token', '');
        $url = sprintf('http://%s:%d%s', $host, $daemonPort, '/servers/reinstall');

        try {
            $cpuCores = $server->cpu_cores;
            if ($cpuCores === null) {
                $cpuCores = $server->tariff->cpu_cores;
            }
            if ($cpuCores === null) {
                $cpuCores = 0;
            }

            $cpuShares = $server->cpu_shares;
            if ($cpuShares === null) {
                $cpuShares = $server->tariff->cpu_shares;
            }

            $ramGb = $server->ram_gb;
            if ($ramGb === null) {
                $ramGb = $server->tariff->ram_gb;
            }
            if ($ramGb === null) {
                $ramGb = 0;
            }

            $diskGb = $server->disk_gb;
            if ($diskGb === null) {
                $diskGb = $server->tariff->disk_gb;
            }
            if ($diskGb === null) {
                $diskGb = 0;
            }

            $http = Http::timeout(10);
            if ($daemonToken !== '') {
                $http = $http->withHeaders([
                    'X-Location-Daemon-Token' => $daemonToken,
                ]);
            }

            $response = $http->post($url, [
                'server_id' => $server->id,
                'game' => $gameCode,
                'port' => (int) $server->port,
                'cpu_cores' => (float) $cpuCores,
                'cpu_shares' => $cpuShares,
                'ram_gb' => (float) $ramGb,
                'disk_gb' => (float) $diskGb,
                'slots' => (int) $server->slots,
                'server_fps' => $server->server_fps,
                'server_tickrate' => $server->server_tickrate,
                'antiddos_enabled' => (bool) $server->antiddos_enabled,
                'archive_url' => $archiveUrl,
                'steam_app_id' => $steamAppId,
                'steam_branch' => $steamBranch,
                'version_name' => $versionName,
            ]);

            if (! $response->successful()) {
                $server->update([
                    'provisioning_status' => 'failed',
                    'provisioning_error' => 'Daemon returned status ' . $response->status(),
                    'runtime_status' => 'offline',
                ]);
                return;
            }

            $data = $response->json();
            $data = array_merge([
                'ok' => false,
                'error' => 'Unknown daemon error',
                'state' => '',
                'port' => $server->port,
                'container_id' => $server->container_id,
                'container_name' => $server->container_name,
            ], (array) $data);
            if (! (bool) $data['ok']) {
                $server->update([
                    'provisioning_status' => 'failed',
                    'provisioning_error' => (string) $data['error'],
                    'runtime_status' => 'offline',
                ]);
                return;
            }

            $daemonState = strtolower((string) $data['state']);
            $runtimeStatus = match ($daemonState) {
                'running' => 'running',
                'missing' => 'missing',
                'stopped', 'exited', 'restarting', 'dead', 'paused', 'created', 'removing' => 'offline',
                default => 'unknown',
            };

            $provisioningStatus = $daemonState === 'running' ? 'running' : 'reinstalling';

            $server->update([
                'port' => (int) $data['port'],
                'container_id' => $data['container_id'],
                'container_name' => $data['container_name'],
                'provisioning_status' => $provisioningStatus,
                'provisioning_error' => null,
                'runtime_status' => $runtimeStatus,
                'status' => 'active',
            ]);
        } catch (\Throwable $e) {
            if ($e instanceof ConnectionException) {
                $server->update([
                    'provisioning_status' => 'reinstalling',
                    'provisioning_error' => null,
                ]);
                return;
            }
            $server->update([
                'provisioning_status' => 'failed',
                'provisioning_error' => $e->getMessage(),
                'runtime_status' => 'offline',
            ]);
        }
    }
}
