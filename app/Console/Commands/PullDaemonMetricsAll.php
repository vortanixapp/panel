<?php

namespace App\Console\Commands;

use App\Dto\Daemon\MetricsResponse;
use App\Models\Location;
use App\Models\LocationMetric;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Http;

class PullDaemonMetricsAll extends Command
{
    protected $signature = 'metrics:pull-daemon-all';

    protected $description = 'Получаем метрики от демонов всех активных локаций.';

    public function handle(): int
    {
        $locations = Location::where('is_active', true)->whereNotNull('ssh_host')->get();

        if ($locations->isEmpty()) {
            $this->info('No active locations with SSH hosts found.');
            return self::SUCCESS;
        }

        foreach ($locations as $location) {
            $this->pullForLocation($location);
            sleep(1);
        }

        $this->info('Daemon metrics pulled for all active locations.');
        return self::SUCCESS;
    }

    private function pullForLocation(Location $location)
    {
        try {
            $port = (int) config('services.location_daemon.port', 9201);
            $token = (string) config('services.location_daemon.token', '');
            $url = sprintf('http://%s:%d/metrics', $location->ssh_host, $port);

            $request = Http::timeout(10);

            if ($token !== '') {
                $request = $request->withHeaders([
                    'X-Location-Daemon-Token' => $token,
                ]);
            }

            $response = $request->get($url);

            if (!$response->successful()) {
                $this->error("Location {$location->code}: daemon returned {$response->status()}");
                return;
            }

            $data = $response->json();
            $metricsResponse = MetricsResponse::fromArray(is_array($data) ? $data : []);
            if (count($metricsResponse->items) === 0) {
                $this->error("Location {$location->code}: invalid response");
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

            $count = 0;
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
                $count++;
            }

            $this->info("Location {$location->code}: pulled {$count} metrics");
        } catch (\Throwable $e) {
            $this->error("Location {$location->code}: {$e->getMessage()}");
        }
    }
}
