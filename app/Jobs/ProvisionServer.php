<?php

namespace App\Jobs;

use App\Models\Server;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Http\Client\ConnectionException;
use Illuminate\Support\Facades\Crypt;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Str;

class ProvisionServer implements ShouldQueue
{
    use Queueable;

    public $timeout = 900;

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
            'provisioning_status' => 'installing',
            'provisioning_error' => null,
        ]);

        $host = (string) $server->location->ssh_host;
        if ($host === '') {
            $server->update([
                'provisioning_status' => 'failed',
                'provisioning_error' => 'Для локации не настроен хост демона',
                'status' => 'suspended',
            ]);
            return;
        }

        $daemonPort = (int) config('services.location_daemon.port', 9201);
        $daemonToken = (string) config('services.location_daemon.token', '');

        $healthUrl = sprintf('http://%s:%d/health', $host, $daemonPort);
        $createUrl = sprintf('http://%s:%d/servers/create', $host, $daemonPort);

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

        try {
            $http = Http::timeout(10);
            if ($daemonToken !== '') {
                $http = $http->withHeaders([
                    'X-Location-Daemon-Token' => $daemonToken,
                ]);
            }

            $healthResponse = $http->get($healthUrl);
            if (! $healthResponse->successful()) {
                $body = (string) $healthResponse->body();
                $body = mb_substr($body, 0, 400);
                throw new \RuntimeException('Daemon healthcheck failed: status ' . $healthResponse->status() . ' url ' . $healthUrl . ' body: ' . $body);
            }

            $response = $http->post($createUrl, [
                'server_id' => (int) $server->id,
                'game' => $gameCode,
                'port' => (int) $server->port,
                'cpu_cores' => (float) $server->cpu_cores,
                'cpu_shares' => $server->cpu_shares,
                'ram_gb' => (float) $server->ram_gb,
                'disk_gb' => (float) $server->disk_gb,
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
                $body = (string) $response->body();
                $body = mb_substr($body, 0, 400);
                throw new \RuntimeException('Daemon returned status ' . $response->status() . ' url ' . $createUrl . ' body: ' . $body);
            }

            $data = $response->json();
            $data = array_merge([
                'ok' => false,
                'error' => 'Unknown daemon error',
                'state' => '',
                'port' => $server->port,
                'container_id' => null,
                'container_name' => null,
            ], (array) $data);
            if (! (bool) $data['ok']) {
                throw new \RuntimeException((string) $data['error']);
            }

            $daemonState = strtolower((string) $data['state']);
            $runtimeStatus = match ($daemonState) {
                'running' => 'running',
                'missing' => 'missing',
                'stopped', 'exited', 'restarting', 'dead', 'paused', 'created', 'removing' => 'offline',
                default => 'unknown',
            };

            $provisioningStatus = $daemonState === 'running' ? 'running' : 'installing';

            $server->update([
                'port' => (int) $data['port'],
                'container_id' => $data['container_id'],
                'container_name' => $data['container_name'],
                'provisioning_status' => $provisioningStatus,
                'provisioning_error' => null,
                'runtime_status' => $runtimeStatus,
                'status' => 'active',
            ]);

            try {
                $ftpUser = 'srv' . $server->id;
                $ftpPass = Str::password(16, true, true, false, false);
                $ftpUrl = sprintf('http://%s:%d/servers/ftp/create-user', $host, $daemonPort);

                $ftpResp = $http->post($ftpUrl, [
                    'server_id' => (int) $server->id,
                    'username' => $ftpUser,
                    'password' => $ftpPass,
                ]);

                if ($ftpResp->successful()) {
                    $ftpData = $ftpResp->json();
                    $ftpData = array_merge([
                        'ok' => false,
                        'port' => 21,
                        'root' => '',
                    ], (array) $ftpData);
                    if ((bool) $ftpData['ok']) {
                        $publicIp = (string) (($server->location->ip_address ?: $server->location->ssh_host) ?: $host);
                        $server->update([
                            'ftp_host' => $publicIp,
                            'ftp_port' => (int) $ftpData['port'],
                            'ftp_username' => $ftpUser,
                            'ftp_password' => Crypt::encryptString($ftpPass),
                            'ftp_root' => (string) $ftpData['root'],
                        ]);
                    }
                }
            } catch (\Throwable $e) {
            }

            try {
                $mysqlDb = 'srv' . $server->id;
                $mysqlUser = 'srv' . $server->id;
                $mysqlPass = Str::password(16, true, true, false, false);
                $mysqlUrl = sprintf('http://%s:%d/servers/mysql/create-db', $host, $daemonPort);

                $mysqlResp = $http->post($mysqlUrl, [
                    'server_id' => (int) $server->id,
                    'database' => $mysqlDb,
                    'username' => $mysqlUser,
                    'password' => $mysqlPass,
                ]);

                if ($mysqlResp->successful()) {
                    $mysqlData = $mysqlResp->json();
                    $mysqlData = array_merge([
                        'ok' => false,
                        'port' => 3306,
                    ], (array) $mysqlData);
                    if ((bool) $mysqlData['ok']) {
                        $publicIp = (string) (($server->location->mysql_host ?: $server->location->ip_address) ?: $host);
                        $mysqlPort = (int) $server->location->mysql_port;
                        if ($mysqlPort <= 0) {
                            $mysqlPort = (int) $mysqlData['port'];
                        }
                        $server->update([
                            'mysql_host' => $publicIp,
                            'mysql_port' => $mysqlPort,
                            'mysql_database' => $mysqlDb,
                            'mysql_username' => $mysqlUser,
                            'mysql_password' => Crypt::encryptString($mysqlPass),
                        ]);
                    }
                }
            } catch (\Throwable $e) {
            }
        } catch (\Throwable $e) {
            if ($e instanceof ConnectionException) {
                $server->update([
                    'provisioning_status' => 'installing',
                    'provisioning_error' => null,
                ]);
                return;
            }
            $server->update([
                'provisioning_status' => 'failed',
                'provisioning_error' => $e->getMessage(),
                'status' => 'suspended',
            ]);
        }
    }
}
