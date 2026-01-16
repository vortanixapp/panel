<?php

namespace App\Console\Commands;

use Illuminate\Console\Command;
use Illuminate\Support\Facades\File;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;
use App\Services\LicenseCloudClient;

class IntegrityCheck extends Command
{
    protected $signature = 'integrity:check {--json : Output JSON} {--fail : Exit with failure when changes detected}';

    protected $description = 'Проверка хеша SHA-256 критически важных файлов на соответствие базовому уровню.';

    public function handle(): int
    {
        $baselinePath = storage_path('app/integrity/baseline.json');
        if (!File::exists($baselinePath)) {
            $msg = 'Baseline not found. Run: php artisan integrity:baseline';
            $this->error($msg);
            Log::warning($msg);
            return $this->option('fail') ? self::FAILURE : self::SUCCESS;
        }

        $baseline = json_decode((string) File::get($baselinePath), true);
        $files = is_array($baseline) && is_array($baseline['files']) ? $baseline['files'] : [];

        $secret = (string) env('INTEGRITY_HMAC_SECRET', '');
        $baselineSigOk = null;
        if ($secret !== '') {
            $expectedSig = is_array($baseline) ? (string) $baseline['hmac_sha256'] : '';
            $payload = is_array($baseline) ? $baseline : [];
            unset($payload['hmac_sha256']);
            if (array_key_exists('files', $payload) && is_array($payload['files'])) {
                ksort($payload['files']);
            }
            $payloadJson = (string) json_encode($payload, JSON_UNESCAPED_SLASHES);
            $actualSig = hash_hmac('sha256', $payloadJson, $secret);
            $baselineSigOk = ($expectedSig !== '') && hash_equals($expectedSig, $actualSig);
        }

        $changes = [];

        foreach ($files as $rel => $meta) {
            $abs = base_path((string) $rel);
            $expected = is_array($meta) ? (string) $meta['sha256'] : '';

            if (!File::exists($abs) || !File::isFile($abs)) {
                $changes[] = [
                    'file' => (string) $rel,
                    'type' => 'missing',
                ];
                continue;
            }

            $actual = hash_file('sha256', $abs);
            if ($expected !== '' && !hash_equals($expected, $actual)) {
                $changes[] = [
                    'file' => (string) $rel,
                    'type' => 'modified',
                    'expected' => $expected,
                    'actual' => $actual,
                ];
            }
        }

        $payload = [
            'checked_at' => now()->toIso8601String(),
            'baseline_sig_ok' => $baselineSigOk,
            'changed' => count($changes) > 0,
            'changes' => $changes,
        ];

        if ($baselineSigOk === false) {
            Log::warning('Integrity baseline signature mismatch', $payload);
            $this->warn('Integrity baseline signature mismatch');
            $this->notify($payload);

            if ($this->option('json')) {
                $this->line(json_encode($payload, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES));
            }

            return $this->option('fail') ? self::FAILURE : self::SUCCESS;
        }

        if ($payload['changed']) {
            Log::warning('Integrity check: CHANGED', $payload);
            $this->warn('Integrity check: CHANGED (' . count($changes) . ')');
            $this->notify($payload);

            if ($this->option('json')) {
                $this->line(json_encode($payload, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES));
            }

            return $this->option('fail') ? self::FAILURE : self::SUCCESS;
        }

        Log::info('Integrity check: OK', $payload);
        $this->info('Integrity check: OK');

        if ($this->option('json')) {
            $this->line(json_encode($payload, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES));
        }

        return self::SUCCESS;
    }

    private function notify(array $payload): void
    {
        $url = (string) env('INTEGRITY_ALERT_WEBHOOK', '');

        if ($url !== '') {
            try {
                Http::timeout(5)->post($url, $payload);
            } catch (\Throwable $e) {
                Log::warning('Integrity alert webhook failed: ' . $e->getMessage());
            }
        }

        try {
            app(LicenseCloudClient::class)->postIntegrityAlert([
                'type' => (string) (($payload['baseline_sig_ok'] === false) ? 'baseline_sig_mismatch' : (($payload['changed']) ? 'changed' : 'changed')),
                'baseline_sig_ok' => $payload['baseline_sig_ok'],
                'changes_count' => is_array($payload['changes']) ? count($payload['changes']) : 0,
                'payload' => $payload,
                'app_version' => (string) config('app.version', ''),
            ]);
        } catch (\Throwable $e) {
            Log::warning('Integrity alert billing upload failed: ' . $e->getMessage());
        }
    }
}
