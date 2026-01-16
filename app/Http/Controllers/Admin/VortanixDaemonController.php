<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Dto\Daemon\MetricsResponse;
use App\Models\LocationDaemon;
use App\Models\LocationMetric;
use Illuminate\Http\RedirectResponse;
use Illuminate\Support\Facades\Auth;
use Illuminate\View\View;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use App\Models\Location;
use Illuminate\Support\Facades\Http;
use phpseclib3\Net\SSH2;

class VortanixDaemonController extends Controller
{
    public function index(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $daemons = LocationDaemon::with('location')->get();

        return view('admin.vortanix-daemons.index', [
            'daemons' => $daemons,
        ]);
    }

    public function show(Location $location): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $this->fetchAndSaveDaemonInfo($location);

        $this->pullMetricsFromDaemon($location);

        $daemon = LocationDaemon::where('location_id', $location->id)->first();

        $metrics = $location->metrics()
            ->whereIn('metric_type', ['daemon_cpu_usage', 'daemon_ram_usage'])
            ->orderByDesc('measured_at')
            ->limit(50)
            ->get()
            ->groupBy('metric_type');

        return view('admin.vortanix-daemons.show', [
            'location' => $location,
            'daemon' => $daemon,
            'metrics' => $metrics,
        ]);
    }

    public function refreshDaemon(Location $location): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['error' => 'Unauthorized'], 403);
        }

        try {
            $this->fetchAndSaveDaemonInfo($location);
            return response()->json(['success' => true]);
        } catch (\Exception $e) {
            return response()->json(['success' => false, 'error' => $e->getMessage()]);
        }
    }

    public function exec(Location $location, Request $request): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['error' => 'Unauthorized'], 403);
        }

        if (! $location->ssh_host) {
            return response()->json(['error' => 'SSH host is not configured for this location'], 400);
        }

        $cmd = (string) $request->input('cmd', '');
        if (trim($cmd) === '') {
            return response()->json(['error' => 'Command is required'], 422);
        }

        try {
            $port = (int) config('services.location_daemon.port', 9201);
            $token = (string) config('services.location_daemon.token', '');
            $url = sprintf('http://%s:%d/exec', $location->ssh_host, $port);

            $http = Http::timeout(15);
            if ($token !== '') {
                $http = $http->withHeaders([
                    'X-Location-Daemon-Token' => $token,
                ]);
            }

            $response = $http->post($url, ['cmd' => $cmd]);

            if (! $response->successful()) {
                return response()->json([
                    'error' => 'Daemon returned status ' . $response->status(),
                ], $response->status());
            }

            return response()->json($response->json());
        } catch (\Throwable $e) {
            return response()->json([
                'error' => $e->getMessage(),
            ], 500);
        }
    }

    public function logs(Location $location): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['error' => 'Unauthorized'], 403);
        }

        if (! $location->ssh_host) {
            return response()->json(['error' => 'SSH host is not configured for this location'], 400);
        }

        try {
            $port = (int) config('services.location_daemon.port', 9201);
            $token = (string) config('services.location_daemon.token', '');
            $url = sprintf('http://%s:%d/logs?tail=300', $location->ssh_host, $port);

            $request = Http::timeout(5);

            if ($token !== '') {
                $request = $request->withHeaders([
                    'X-Location-Daemon-Token' => $token,
                ]);
            }

            $response = $request->get($url);

            if (! $response->successful()) {
                return response()->json([
                    'error' => 'Daemon returned status ' . $response->status(),
                ], $response->status());
            }

            return response()->json($response->json());
        } catch (\Throwable $e) {
            return response()->json([
                'error' => $e->getMessage(),
            ], 500);
        }
    }

    public function restartDaemon(Location $location): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['error' => 'Unauthorized'], 403);
        }

        try {
            $commands = [
                'sudo systemctl restart vortanix-daemon',
            ];

            $this->runSSHCommands($location, $commands, 'restart_' . Auth::id());

            sleep(2);
            $this->fetchAndSaveDaemonInfo($location);

            return response()->json(['success' => true]);
        } catch (\Exception $e) {
            return response()->json(['success' => false, 'error' => $e->getMessage()]);
        }
    }

    private function fetchAndSaveDaemonInfo(Location $location): void
    {
        try {
            $port = (int) config('services.location_daemon.port', 9201);
            $token = (string) config('services.location_daemon.token', '');
            $url = sprintf('http://%s:%d/info', $location->ssh_host, $port);

            $request = Http::timeout(5);

            if ($token !== '') {
                $request = $request->withHeaders([
                    'X-Location-Daemon-Token' => $token,
                ]);
            }

            $response = $request->get($url);

            if ($response->successful()) {
                $data = $response->json();

                LocationDaemon::updateOrCreate(
                    ['location_id' => $location->id],
                    [
                        'status' => LocationDaemon::STATUS_ONLINE,
                        'version' => '1.0',
                        'pid' => $data['pid'],
                        'uptime_sec' => $data['uptime_sec'],
                        'platform' => $data['platform'],
                        'last_seen' => now(),
                    ]
                );
            } else {
                LocationDaemon::updateOrCreate(
                    ['location_id' => $location->id],
                    [
                        'status' => LocationDaemon::STATUS_OFFLINE,
                        'last_seen' => now(),
                    ]
                );
            }
        } catch (\Exception $e) {
            LocationDaemon::updateOrCreate(
                ['location_id' => $location->id],
                [
                    'status' => LocationDaemon::STATUS_UNKNOWN,
                    'last_seen' => now(),
                ]
            );
        }
    }

    private function pullMetricsFromDaemon(Location $location): void
    {
        if (! $location->ssh_host) {
            return;
        }

        $port = (int) config('services.location_daemon.port', 9201);
        $token = (string) config('services.location_daemon.token', '');
        $url = sprintf('http://%s:%d/metrics', $location->ssh_host, $port);

        try {
            $request = Http::timeout(5);

            if ($token !== '') {
                $request = $request->withHeaders([
                    'X-Location-Daemon-Token' => $token,
                ]);
            }

            $response = $request->get($url);

            if (! $response->successful()) {
                return;
            }

            $data = $response->json();
            $metricsResponse = MetricsResponse::fromArray(is_array($data) ? $data : []);
            if (count($metricsResponse->items) === 0) {
                return;
            }

            $textTypes = [
                'os_info',
                'cpu_model',
                'ram_total',
                'disk_total',
                'disk_used',
                'disk_available',
                'uptime',
            ];

            foreach ($metricsResponse->items as $metric) {
                $type = $metric->type;
                $value = $metric->value;
                $measuredAt = $metric->measuredAt;

                $isText = in_array($type, $textTypes, true);

                LocationMetric::create([
                    'location_id' => $location->id,
                    'metric_type' => $type,
                    'value' => $isText ? 0 : (float) $value,
                    'text_value' => $isText ? (string) $value : null,
                    'measured_at' => $measuredAt,
                ]);
            }
        } catch (\Throwable $e) {
            // Тихо игнорируем ошибку, страница демона всё равно отобразится
        }
    }

    private function runSSHCommands(Location $location, array $commands, string $cacheKey): string
    {
        $host = $location->ssh_host;
        $user = (string) $location->ssh_user;
        $port = (int) $location->ssh_port;
        $password = $location->ssh_password;

        $ssh = new SSH2($host, $port);

        if (!$ssh->login($user, $password)) {
            throw new \Exception('SSH login failed');
        }

        $output = '';

        $combinedCommand = implode(' && ', array_map(function($cmd) {
            return "({$cmd})";
        }, $commands));

        $result = $ssh->exec($combinedCommand);
        if ($result === false) {
            throw new \Exception("SSH command failed: $combinedCommand");
        }
        $output .= $result . "\n";

        $ssh->disconnect();

        return $output;
    }
}
