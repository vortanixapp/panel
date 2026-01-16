<?php

namespace App\Http\Controllers;

use App\Models\Server;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Http;
use Illuminate\View\View;

class MonitoringController extends Controller
{
    public function index(): View
    {
        return view('monitoring');
    }

    public function data(Request $request): JsonResponse
    {
        $servers = Server::query()
            ->with(['game', 'location'])
            ->where('status', 'active')
            ->where('runtime_status', 'running')
            ->where(function ($q) {
                $q->whereNull('provisioning_status')
                    ->orWhere('provisioning_status', 'running');
            })
            ->orderBy('id', 'desc')
            ->get();

        $byLocation = $servers->groupBy(fn ($s) => (int) $s->location_id);

        $onlineMap = [];

        foreach ($byLocation as $locationId => $group) {
            $location = $group->first()?->location;
            if (! $location || ! $location->ssh_host) {
                continue;
            }

            $items = $group
                ->filter(fn ($s) => (int) $s->port > 0)
                ->map(function ($s) {
                    $gameCode = strtolower((string) $s->game->code);
                    return [
                        'server_id' => (int) $s->id,
                        'game' => $gameCode,
                        'port' => (int) $s->port,
                    ];
                })
                ->values()
                ->all();

            if (empty($items)) {
                continue;
            }

            $key = 'monitoring:online:' . (int) $locationId . ':' . md5(json_encode($items));

            $resp = Cache::remember($key, 12, function () use ($location, $items) {
                $daemonPort = (int) config('services.location_daemon.port', 9201);
                $daemonToken = (string) config('services.location_daemon.token', '');

                $url = sprintf('http://%s:%d/servers/online-batch', $location->ssh_host, $daemonPort);

                try {
                    $http = Http::timeout(3);
                    if ($daemonToken !== '') {
                        $http = $http->withHeaders([
                            'X-Location-Daemon-Token' => $daemonToken,
                        ]);
                    }

                    $response = $http->post($url, [
                        'host' => '127.0.0.1',
                        'items' => $items,
                    ]);

                    if (! $response->successful()) {
                        return ['ok' => false, 'items' => []];
                    }

                    $data = $response->json();
                    if (! is_array($data) || ! $data['ok'] || ! is_array($data['items'])) {
                        return ['ok' => false, 'items' => []];
                    }

                    return $data;
                } catch (\Throwable $e) {
                    return ['ok' => false, 'items' => []];
                }
            });

            if (! is_array($resp) || ! is_array($resp['items'])) {
                continue;
            }

            foreach ($resp['items'] as $it) {
                if (! is_array($it) || ! array_key_exists('server_id', $it)) {
                    continue;
                }
                $sid = (int) $it['server_id'];
                $onlineMap[$sid] = [
                    'online' => array_key_exists('online', $it) ? $it['online'] : null,
                    'max' => array_key_exists('max', $it) ? $it['max'] : null,
                    'ok' => (bool) $it['ok'],
                ];
            }
        }

        $out = $servers->map(function ($s) use ($onlineMap) {
            $gameCode = strtolower((string) $s->game->code);
            $locIp = (string) $s->location->ip_address;
            $sid = (int) $s->id;
            $om = $onlineMap[$sid];

            return [
                'server_id' => $sid,
                'name' => (string) $s->name,
                'game' => $gameCode,
                'game_name' => (string) $s->game->name,
                'ip' => $locIp,
                'port' => (int) $s->port,
                'runtime_status' => (string) $s->runtime_status,
                'online' => $om ? $om['online'] : null,
                'max' => $om ? $om['max'] : null,
                'online_ok' => $om ? (bool) $om['ok'] : false,
            ];
        })->values();

        return response()->json([
            'ok' => true,
            'items' => $out,
        ]);
    }
}
