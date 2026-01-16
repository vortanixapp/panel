<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Dto\Daemon\MetricsResponse;
use App\Models\Location;
use App\Models\LocationMetric;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Http;
use Illuminate\View\View;
use phpseclib3\Net\SSH2;

class LocationController extends Controller
{
    public function index(Request $request): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $status = $request->query('status', 'active');
        $selectedRegion = $request->query('region');

        $query = Location::query();

        if ($status === 'active') {
            $query->where('is_active', true);
        } elseif ($status === 'inactive') {
            $query->where('is_active', false);
        }

        if ($selectedRegion) {
            $query->where('region', $selectedRegion);
        }

        $locations = $query
            ->orderBy('sort_order')
            ->orderBy('name')
            ->get();

        $regions = Location::query()
            ->whereNotNull('region')
            ->where('region', '!=', '')
            ->distinct()
            ->orderBy('region')
            ->pluck('region');

        return view('admin.locations.index', [
            'locations' => $locations,
            'status' => $status,
            'regions' => $regions,
            'selectedRegion' => $selectedRegion,
        ]);
    }

    public function create(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return view('admin.locations.create');
    }

    public function show(Location $location): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $metrics = $location->metrics()
            ->whereIn('metric_type', ['cpu_usage', 'ram_usage', 'online_players'])
            ->orderByDesc('measured_at')
            ->limit(50)
            ->get()
            ->groupBy('metric_type');

        $serverMetrics = $location->metrics()
            ->whereIn('metric_type', ['os_info', 'cpu_model', 'ram_total', 'disk_total', 'disk_used', 'disk_available', 'uptime'])
            ->orderByDesc('measured_at')
            ->get()
            ->keyBy('metric_type');

        $serviceStatuses = $this->getServiceStatuses($location);

        return view('admin.locations.show', [
            'location' => $location,
            'metrics' => $metrics,
            'serverMetrics' => $serverMetrics,
            'serviceStatuses' => $serviceStatuses,
        ]);
    }

    private function getServiceStatuses(Location $location): array
    {
        $services = [
            'docker' => 'Docker',
            'mysql' => 'MySQL',
            'vsftpd' => 'FTP',
            'vortanix-daemon' => 'Vortanix Daemon',
        ];

        $result = [];
        foreach ($services as $unit => $label) {
            $result[$unit] = [
                'label' => $label,
                'state' => 'unknown',
                'error' => null,
            ];
        }

        if (! $location->ssh_host || ! $location->ssh_user || ! $location->ssh_password) {
            foreach ($result as &$service) {
                $service['state'] = 'unconfigured';
            }
            return $result;
        }

        try {
            $ssh = new SSH2($location->ssh_host, (int) $location->ssh_port > 0 ? (int) $location->ssh_port : 22);

            if (! $ssh->login($location->ssh_user, $location->ssh_password)) {
                foreach ($result as &$service) {
                    $service['state'] = 'ssh_failed';
                }
                return $result;
            }

            $cmd = 'for s in docker mysql vsftpd vortanix-daemon; do '
                . 'state=$(systemctl is-active $s 2>/dev/null || echo "unknown"); '
                . 'echo "$s $state"; '
                . 'done';

            $output = $ssh->exec($cmd);
            $ssh->disconnect();

            if ($output !== false) {
                foreach (preg_split("/\r?\n/", trim($output)) as $line) {
                    if (! $line) {
                        continue;
                    }
                    [$name, $state] = array_pad(explode(' ', trim($line), 2), 2, 'unknown');
                    if (array_key_exists($name, $result)) {
                        $result[$name]['state'] = $state ?: 'unknown';
                    }
                }
            }
        } catch (\Throwable $e) {
            foreach ($result as &$service) {
                $service['state'] = 'error';
                $service['error'] = $e->getMessage();
            }
        }

        return $result;
    }

    public function pullFromDaemon(Location $location): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        if (! $location->ssh_host) {
            return redirect()
                ->route('admin.locations.show', $location)
                ->with('error', 'Для локации не настроен SSH‑хост, адрес демона неизвестен.');
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
                return redirect()
                    ->route('admin.locations.show', $location)
                    ->with('error', 'Демон вернул статус ' . $response->status());
            }

            $data = $response->json();
            $metricsResponse = MetricsResponse::fromArray(is_array($data) ? $data : []);
            if (count($metricsResponse->items) === 0) {
                return redirect()
                    ->route('admin.locations.show', $location)
                    ->with('error', 'Некорректный ответ демона.');
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
                if ($measuredAt === null) {
                    $measuredAt = now();
                }

                $isText = in_array($type, $textTypes, true);

                LocationMetric::create([
                    'location_id' => $location->id,
                    'metric_type' => $type,
                    'value' => $isText ? 0 : (float) $value,
                    'text_value' => $isText ? (string) $value : null,
                    'measured_at' => $measuredAt,
                ]);
            }

            return redirect()
                ->route('admin.locations.show', $location)
                ->with('status', 'Метрики успешно обновлены с демона.');
        } catch (\Throwable $e) {
            return redirect()
                ->route('admin.locations.show', $location)
                ->with('error', 'Ошибка запроса к демону: ' . $e->getMessage());
        }
    }

    public function store(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'code' => ['required', 'string', 'max:100', 'unique:locations,code'],
            'name' => ['required', 'string', 'max:255'],
            'country' => ['nullable', 'string', 'max:255'],
            'city' => ['nullable', 'string', 'max:255'],
            'region' => ['nullable', 'string', 'max:255'],
            'description' => ['nullable', 'string'],
            'ip_address' => ['nullable', 'string', 'max:255'],
            'ip_pool' => ['nullable', 'string'],
            'sort_order' => ['nullable', 'integer', 'min:0'],
            'is_active' => ['sometimes', 'boolean'],
            'ssh_host' => ['nullable', 'string', 'max:255'],
            'ssh_user' => ['nullable', 'string', 'max:255'],
            'ssh_password' => ['nullable', 'string', 'max:255'],
            'ssh_port' => ['nullable', 'integer', 'min:1', 'max:65535'],
        ]);

        $validated = array_merge([
            'country' => null,
            'city' => null,
            'region' => null,
            'description' => null,
            'ip_address' => null,
            'ip_pool' => null,
            'sort_order' => 0,
            'is_active' => true,
            'ssh_host' => null,
            'ssh_user' => null,
            'ssh_password' => null,
            'ssh_port' => 22,
        ], (array) $validated);

        $ipPool = $this->parseIpPool((string) $validated['ip_pool']);

        Location::create([
            'code' => $validated['code'],
            'name' => $validated['name'],
            'country' => $validated['country'],
            'city' => $validated['city'],
            'region' => $validated['region'],
            'description' => $validated['description'],
            'ip_address' => $validated['ip_address'],
            'ip_pool' => $ipPool,
            'sort_order' => (int) $validated['sort_order'],
            'is_active' => (bool) $validated['is_active'],
            'ssh_host' => $validated['ssh_host'],
            'ssh_user' => $validated['ssh_user'],
            'ssh_password' => $validated['ssh_password'],
            'ssh_port' => (int) $validated['ssh_port'],
        ]);

        return redirect()->route('admin.locations.index');
    }

    private function parseIpPool(?string $raw): ?array
    {
        $raw = trim((string) $raw);
        if ($raw === '') {
            return null;
        }

        $decoded = json_decode($raw, true);
        if (is_array($decoded)) {
            $items = $decoded;
        } else {
            $items = preg_split('/[\s,;]+/', $raw) ?: [];
        }

        $out = [];
        foreach ($items as $item) {
            $ip = trim((string) $item);
            if ($ip === '') {
                continue;
            }
            if (! preg_match('/^\d{1,3}(?:\.\d{1,3}){3}$/', $ip)) {
                continue;
            }
            $out[$ip] = $ip;
        }

        return empty($out) ? null : array_values($out);
    }

    public function edit(Location $location): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return view('admin.locations.edit', [
            'location' => $location,
        ]);
    }

    public function update(Request $request, Location $location): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'code' => ['required', 'string', 'max:100', 'unique:locations,code,' . $location->id],
            'name' => ['required', 'string', 'max:255'],
            'country' => ['nullable', 'string', 'max:255'],
            'city' => ['nullable', 'string', 'max:255'],
            'region' => ['nullable', 'string', 'max:255'],
            'description' => ['nullable', 'string'],
            'ip_address' => ['nullable', 'string', 'max:255'],
            'ip_pool' => ['nullable', 'string'],
            'sort_order' => ['nullable', 'integer', 'min:0'],
            'is_active' => ['sometimes', 'boolean'],
            'ssh_host' => ['nullable', 'string', 'max:255'],
            'ssh_user' => ['nullable', 'string', 'max:255'],
            'ssh_password' => ['nullable', 'string', 'max:255'],
            'ssh_port' => ['nullable', 'integer', 'min:1', 'max:65535'],
        ]);

        $validated = array_merge([
            'country' => null,
            'city' => null,
            'region' => null,
            'description' => null,
            'ip_address' => null,
            'ip_pool' => null,
            'sort_order' => 0,
            'is_active' => false,
            'ssh_host' => null,
            'ssh_user' => null,
            'ssh_password' => null,
            'ssh_port' => ((int) $location->ssh_port > 0 ? (int) $location->ssh_port : 22),
        ], (array) $validated);

        $ipPool = $this->parseIpPool((string) $validated['ip_pool']);

        $data = [
            'code' => $validated['code'],
            'name' => $validated['name'],
            'country' => $validated['country'],
            'city' => $validated['city'],
            'region' => $validated['region'],
            'description' => $validated['description'],
            'ip_address' => $validated['ip_address'],
            'ip_pool' => $ipPool,
            'sort_order' => (int) $validated['sort_order'],
            'is_active' => (bool) $validated['is_active'],
            'ssh_host' => $validated['ssh_host'],
            'ssh_user' => $validated['ssh_user'],
            'ssh_port' => (int) $validated['ssh_port'],
        ];

        if ((string) $validated['ssh_password'] !== '') {
            $data['ssh_password'] = $validated['ssh_password'];
        }

        $location->update($data);

        return redirect()->route('admin.locations.index');
    }

    public function toggle(Location $location): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $location->update([
            'is_active' => ! $location->is_active,
        ]);

        return redirect()->route('admin.locations.index');
    }

    public function destroy(Location $location): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $location->delete();

        return redirect()->route('admin.locations.index')->with('success', 'Локация удалена успешно.');
    }
}
