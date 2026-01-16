<?php

namespace App\Console\Commands;

use App\Services\LicenseCloudClient;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\File;

class IntegrityBaseline extends Command
{
    protected $signature = 'integrity:baseline {--force : Overwrite existing baseline} {--upload : Upload baseline to billing}';

    protected $description = 'Базовый набор хешей SHA-256 для критически важных файлов.';

    public function handle(): int
    {
        $baselinePath = storage_path('app/integrity/baseline.json');

        if (File::exists($baselinePath) && !$this->option('force')) {
            $this->error('Baseline already exists. Use --force to overwrite.');
            return self::FAILURE;
        }

        $files = $this->criticalFiles();
        $data = [
            'generated_at' => now()->toIso8601String(),
            'base_path' => base_path(),
            'files' => [],
        ];

        foreach ($files as $rel) {
            $abs = base_path($rel);
            if (!File::exists($abs) || !File::isFile($abs)) {
                $data['files'][$rel] = [
                    'status' => 'missing',
                ];
                continue;
            }

            $data['files'][$rel] = [
                'status' => 'ok',
                'sha256' => hash_file('sha256', $abs),
                'size' => File::size($abs),
                'mtime' => File::lastModified($abs),
            ];
        }

        ksort($data['files']);

        $secret = (string) env('INTEGRITY_HMAC_SECRET', '');
        if ($secret !== '') {
            $payload = $data;
            $payloadJson = $this->canonicalJson($payload);
            $data['hmac_sha256'] = hash_hmac('sha256', $payloadJson, $secret);
        }

        File::ensureDirectoryExists(dirname($baselinePath));
        File::put($baselinePath, json_encode($data, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES));

        $this->info('Baseline saved: ' . $baselinePath);

        $shouldUpload = (bool) $this->option('upload') || ((string) env('INTEGRITY_UPLOAD_BASELINE', '') === '1');
        if ($shouldUpload) {
            try {
                app(LicenseCloudClient::class)->uploadIntegrityBaseline($data);
                $this->info('Baseline uploaded to billing');
            } catch (\Throwable $e) {
                $this->error('Baseline upload failed: ' . $e->getMessage());
                return self::FAILURE;
            }
        }

        return self::SUCCESS;
    }

    private function criticalFiles(): array
    {
        $paths = [
            'bootstrap/app.php',
            'routes/web.php',
            'app/Http/Middleware/EnsureValidLicense.php',
            'app/Http/Controllers/Admin/LicenseController.php',
            'app/Http/Controllers/Admin/UpdatesController.php',
        ];

        foreach (File::glob(base_path('app/Services/License*.php')) as $abs) {
            $paths[] = ltrim(str_replace(base_path(), '', (string) $abs), DIRECTORY_SEPARATOR);
        }
        foreach (File::glob(base_path('app/Services/Manifest*.php')) as $abs) {
            $paths[] = ltrim(str_replace(base_path(), '', (string) $abs), DIRECTORY_SEPARATOR);
        }
        foreach (File::glob(base_path('app/Services/*Verifier*.php')) as $abs) {
            $paths[] = ltrim(str_replace(base_path(), '', (string) $abs), DIRECTORY_SEPARATOR);
        }

        $paths = array_values(array_unique(array_filter($paths)));
        sort($paths);
        return $paths;
    }

    private function canonicalJson(array $payload): string
    {
        if (array_key_exists('files', $payload) && is_array($payload['files'])) {
            ksort($payload['files']);
        }

        return (string) json_encode($payload, JSON_UNESCAPED_SLASHES);
    }
}
