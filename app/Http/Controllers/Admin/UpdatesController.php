<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\File;
use Illuminate\Support\Facades\Http;
use Illuminate\View\View;
use ZipArchive;

class UpdatesController extends Controller
{
    private function upsertEnvValues(array $values): void
    {
        $envPath = base_path('.env');
        $contents = file_exists($envPath) ? (string) file_get_contents($envPath) : '';

        foreach ($values as $key => $value) {
            $key = (string) $key;
            $value = (string) $value;

            $pattern = '/^' . preg_quote($key, '/') . '=.*/m';
            $line = $key . '=' . $value;

            if (preg_match($pattern, $contents) === 1) {
                $contents = preg_replace($pattern, $line, $contents);
            } else {
                $contents = rtrim($contents, "\n") . "\n" . $line . "\n";
            }
        }

        $tmp = $envPath . '.' . bin2hex(random_bytes(6)) . '.tmp';
        file_put_contents($tmp, $contents);
        rename($tmp, $envPath);
    }

    public function index(Request $request): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $currentVersion = (string) config('app.version', '0.0.0');

        $cached = Cache::get('license:cloud:manifest');
        $manifest = is_array($cached) && is_array($cached['manifest']) ? $cached['manifest'] : null;

        $latest = is_array($manifest) ? (string) $manifest['latest_version'] : '';
        $min = is_array($manifest) ? (string) $manifest['min_version'] : '';
        $blocked = is_array($manifest) && is_array($manifest['blocked_versions']) ? $manifest['blocked_versions'] : [];

        $blocked = array_values(array_filter(array_map(fn ($v) => (string) $v, $blocked), fn ($v) => $v !== ''));

        $needsUpdate = false;
        if ($latest !== '' && version_compare($currentVersion, $latest, '<')) {
            $needsUpdate = true;
        }

        $mandatoryUpdate = false;
        if ($min !== '' && version_compare($currentVersion, $min, '<')) {
            $mandatoryUpdate = true;
        }

        $isBlocked = in_array($currentVersion, $blocked, true);

        return view('admin.updates', [
            'currentVersion' => $currentVersion,
            'manifestCache' => is_array($cached) ? $cached : null,
            'manifest' => $manifest,
            'latest' => $latest,
            'min' => $min,
            'blocked' => $blocked,
            'needsUpdate' => $needsUpdate,
            'mandatoryUpdate' => $mandatoryUpdate,
            'isBlocked' => $isBlocked,
        ]);
    }

    public function apply(Request $request): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['message' => 'Forbidden'], 403);
        }

        $lock = Cache::lock('panel:update:apply', 600);
        if (! $lock->get()) {
            return response()->json(['message' => 'Update is already running'], 409);
        }

        try {
            $cached = Cache::get('license:cloud:manifest');
            $ok = is_array($cached) ? (bool) $cached['ok'] : false;
            $manifest = is_array($cached) && is_array($cached['manifest']) ? $cached['manifest'] : null;
            $downloadUrl = is_array($manifest) ? (string) $manifest['download_url'] : '';
            $latestVersion = is_array($manifest) ? (string) $manifest['latest_version'] : '';

            if (! $ok) {
                return response()->json(['message' => 'Manifest signature is not valid'], 422);
            }

            if ($downloadUrl === '') {
                return response()->json(['message' => 'download_url is not set in manifest'], 422);
            }

            $parts = parse_url($downloadUrl);
            $scheme = is_array($parts) ? (string) $parts['scheme'] : '';
            if (! in_array($scheme, ['https', 'http'], true)) {
                return response()->json(['message' => 'download_url must be http/https'], 422);
            }

            $updatesDir = storage_path('app/updates');
            $zipPath = $updatesDir . '/panel_update.zip';
            $extractDir = $updatesDir . '/extract';

            File::ensureDirectoryExists($updatesDir);
            if (File::exists($extractDir)) {
                File::deleteDirectory($extractDir);
            }
            File::ensureDirectoryExists($extractDir);

            $resp = Http::timeout(120)->withOptions(['verify' => false])->get($downloadUrl);
            if (! $resp->successful()) {
                return response()->json(['message' => 'Failed to download update: HTTP ' . $resp->status()], 502);
            }

            File::put($zipPath, $resp->body());

            $zip = new ZipArchive();
            if ($zip->open($zipPath) !== true) {
                return response()->json(['message' => 'Failed to open ZIP archive'], 422);
            }
            $zip->extractTo($extractDir);
            $zip->close();

            $root = base_path();
            $entries = File::directories($extractDir);
            $sourceRoot = count($entries) === 1 ? $entries[0] : $extractDir;

            $this->syncDir($sourceRoot, $root, [
                '.env',
                'storage',
                'bootstrap/cache',
            ]);

            if ($latestVersion !== '') {
                $this->upsertEnvValues([
                    'APP_VERSION' => $latestVersion,
                ]);
                putenv('APP_VERSION=' . $latestVersion);
                $_ENV['APP_VERSION'] = $latestVersion;
                $_SERVER['APP_VERSION'] = $latestVersion;
            }

            Artisan::call('optimize:clear');

            if ((string) env('INTEGRITY_AUTO_BASELINE', '1') === '1') {
                $upload = (string) env('INTEGRITY_UPLOAD_BASELINE', '') === '1';
                Artisan::call('integrity:baseline', [
                    '--force' => true,
                    '--upload' => $upload,
                ]);
            }

            return response()->json([
                'message' => 'Updated successfully',
            ]);
        } finally {
            optional($lock)->release();
        }
    }

    private function syncDir(string $from, string $to, array $exclude): void
    {
        $items = File::allFiles($from, true);
        foreach ($items as $item) {
            $rel = str_replace('\\', '/', $item->getRelativePathname());
            $parts = explode('/', $rel);
            $top = (string) $parts[0];

            if (in_array($top, $exclude, true) || in_array($rel, $exclude, true)) {
                continue;
            }

            $dest = rtrim($to, DIRECTORY_SEPARATOR) . DIRECTORY_SEPARATOR . $rel;
            File::ensureDirectoryExists(dirname($dest));
            File::copy($item->getPathname(), $dest);
        }
    }
}
