<?php

namespace App\Console\Commands;

use App\Models\Location;
use App\Models\Server;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Http;

class PullDaemonServerStatusesAll extends Command
{
    protected $signature = 'servers:pull-daemon-status-all {--location= : Location code filter}';

    protected $description = 'Получение статуса серверов от демонов.';

    public function handle(): int
    {
        $locationsQuery = Location::where('is_active', true)
            ->whereNotNull('ssh_host');

        $filter = (string) $this->option('location');
        $filter = trim($filter);
        if ($filter !== '') {
            $locationsQuery->where('code', $filter);
        }

        $locations = $locationsQuery->get();

        if ($locations->isEmpty()) {
            $this->info('No active locations with SSH hosts found.');
            return self::SUCCESS;
        }

        foreach ($locations as $location) {
            $this->pullForLocation($location);
            sleep(1);
        }

        $this->info('Server runtime statuses pulled for all active locations.');
        return self::SUCCESS;
    }

    private function pullForLocation(Location $location): void
    {
        $servers = Server::query()
            ->where('location_id', $location->id)
            ->orderBy('id')
            ->get();

        if ($servers->isEmpty()) {
            $this->info("Location {$location->code}: no servers");
            return;
        }

        $port = (int) config('services.location_daemon.port', 9201);
        $token = (string) config('services.location_daemon.token', '');

        $request = Http::timeout(10);
        if ($token !== '') {
            $request = $request->withHeaders([
                'X-Location-Daemon-Token' => $token,
            ]);
        }

        $updated = 0;
        $errors = 0;

        foreach ($servers as $server) {
            try {
                $gameCode = strtolower((string) $server->game->code);

                $url = sprintf('http://%s:%d/servers/status', $location->ssh_host, $port);
                $response = $request->post($url, [
                    'server_id' => (int) $server->id,
                    'game' => $gameCode,
                ]);

                if (! $response->successful()) {
                    $errors++;
                    $server->update([
                        'runtime_status' => 'unknown',
                    ]);
                    continue;
                }

                $data = $response->json();
                if (! is_array($data) || ! $data['ok']) {
                    $errors++;
                    $server->update([
                        'runtime_status' => 'unknown',
                    ]);
                    continue;
                }

                $daemonState = (string) $data['state'];
                $runtimeStatus = $this->mapRuntimeStatus($daemonState);

                if ((string) $server->provisioning_status === 'reinstalling') {
                    $runtimeStatus = 'reinstalling';
                }

                $updates = [
                    'runtime_status' => $runtimeStatus,
                    'container_id' => $data['container_id'],
                    'container_name' => $data['container_name'],
                ];

                $daemonStateLower = strtolower(trim($daemonState));
                if ($daemonStateLower === 'running') {
                    $prov = strtolower((string) $server->provisioning_status);
                    if (in_array($prov, ['pending', 'installing', 'reinstalling', 'failed'], true)) {
                        $updates['provisioning_status'] = 'running';
                        $updates['provisioning_error'] = null;
                    }

                    $currStatus = strtolower((string) $server->status);
                    if ($currStatus === 'suspended') {
                        $updates['status'] = 'active';
                    }
                }

                $server->update($updates);

                $updated++;
            } catch (\Throwable $e) {
                $errors++;
                $server->update([
                    'runtime_status' => 'unknown',
                ]);
            }
        }

        $this->info("Location {$location->code}: updated {$updated}, errors {$errors}");
    }

    private function mapRuntimeStatus(string $daemonState): string
    {
        $state = strtolower(trim($daemonState));

        return match ($state) {
            'running' => 'running',
            'stopped', 'exited', 'restarting', 'dead', 'paused', 'created', 'removing' => 'offline',
            'missing' => 'missing',
            default => 'unknown',
        };
    }
}
