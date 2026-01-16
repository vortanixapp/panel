<?php

namespace App\Console\Commands;

use App\Models\Location;
use App\Models\Server;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Http;

class StopExpiredServers extends Command
{
    protected $signature = 'servers:stop-expired {--location= : Location code filter}';

    protected $description = 'Остановка серверов, срок аренды которых истек.';

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

        $totalStopped = 0;
        $totalErrors = 0;

        foreach ($locations as $location) {
            [$stopped, $errors] = $this->stopForLocation($location);
            $totalStopped += $stopped;
            $totalErrors += $errors;
            sleep(1);
        }

        $this->info("Expired servers stop completed. stopped={$totalStopped}, errors={$totalErrors}");

        return $totalErrors > 0 ? self::FAILURE : self::SUCCESS;
    }

    private function stopForLocation(Location $location): array
    {
        $servers = Server::query()
            ->where('location_id', $location->id)
            ->whereNotNull('expires_at')
            ->where('expires_at', '<', now())
            ->orderBy('id')
            ->get();

        if ($servers->isEmpty()) {
            $this->info("Location {$location->code}: no expired servers");
            return [0, 0];
        }

        $port = (int) config('services.location_daemon.port', 9201);
        $token = (string) config('services.location_daemon.token', '');

        $request = Http::timeout(15);
        if ($token !== '') {
            $request = $request->withHeaders([
                'X-Location-Daemon-Token' => $token,
            ]);
        }

        $stopped = 0;
        $errors = 0;

        foreach ($servers as $server) {
            try {
                $gameCode = strtolower((string) $server->game->code);
                if ($gameCode === '') {
                    $errors++;
                    continue;
                }

                $url = sprintf('http://%s:%d/servers/stop', $location->ssh_host, $port);
                $response = $request->post($url, [
                    'server_id' => (int) $server->id,
                    'game' => $gameCode,
                ]);

                if (! $response->successful()) {
                    $errors++;
                    continue;
                }

                $data = $response->json();
                if (! is_array($data) || ! $data['ok']) {
                    $errors++;
                    continue;
                }

                $server->update([
                    'runtime_status' => 'offline',
                    'status' => 'suspended',
                ]);

                $stopped++;
            } catch (\Throwable $e) {
                $errors++;
            }
        }

        $this->info("Location {$location->code}: stopped {$stopped}, errors {$errors}");

        return [$stopped, $errors];
    }
}
