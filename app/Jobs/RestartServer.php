<?php

namespace App\Jobs;

use App\Models\Server;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

class RestartServer implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public int $serverId;

    public function __construct(int $serverId)
    {
        $this->serverId = $serverId;
        $this->onQueue('default');
    }

    public function handle(): void
    {
        $server = Server::with(['location', 'game', 'tariff'])->find($this->serverId);
        if (! $server) {
            return;
        }

        $host = (string) $server->location->ssh_host;
        if ($host === '') {
            Log::warning('RestartServer: missing daemon host', ['server_id' => $server->id]);
            return;
        }

        $daemonPort = (int) config('services.location_daemon.port', 9201);
        $daemonToken = (string) config('services.location_daemon.token', '');
        $url = sprintf('http://%s:%d/servers/restart', $host, $daemonPort);

        $gameCode = strtolower((string) $server->game->code);

        $payload = [
            'server_id' => $server->id,
            'game' => $gameCode,
            'cpu_cores' => (float) $server->cpu_cores,
            'cpu_shares' => $server->cpu_shares,
            'ram_gb' => (float) $server->ram_gb,
            'disk_gb' => (float) $server->disk_gb,
            'slots' => (int) $server->slots,
            'server_fps' => $server->server_fps,
            'server_tickrate' => $server->server_tickrate,
            'antiddos_enabled' => $server->antiddos_enabled,
        ];

        try {
            $http = Http::timeout(5);
            if ($daemonToken !== '') {
                $http = $http->withHeaders([
                    'X-Location-Daemon-Token' => $daemonToken,
                ]);
            }
            $response = $http->post($url, $payload);
            if (! $response->successful()) {
                Log::warning('RestartServer daemon error', [
                    'server_id' => $server->id,
                    'status' => $response->status(),
                    'body' => $response->body(),
                ]);
            }
        } catch (\Throwable $e) {
            Log::warning('RestartServer exception', [
                'server_id' => $server->id,
                'error' => $e->getMessage(),
            ]);
        }
    }
}
