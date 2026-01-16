<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Location;
use App\Models\LocationSetupStatus;
use App\Jobs\InstallComponent;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\RedirectResponse;
use Illuminate\Support\Facades\Auth;
use Illuminate\View\View;

class LocationSetupController extends Controller
{
    private const SETUP_CACHE_TTL_SEC = 21600;

    public function index(Location $location): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        if (! $location->ssh_host) {
            return redirect()
                ->route('admin.locations.show', $location)
                ->with('error', 'Для локации не настроен SSH-хост.');
        }

        $statuses = LocationSetupStatus::where('location_id', $location->id)
            ->pluck('status', 'component')
            ->toArray();
            
        $cacheKey = "setup_status_{$location->id}_" . Auth::id();
        $cacheState = \Cache::get($cacheKey);

        if ($cacheState && (($cacheState['completed']) === false) && !empty($cacheState['component'])) {
            $statuses[$cacheState['component']] = LocationSetupStatus::STATUS_INSTALLING;
        }

        $checks = [
            [
                'key' => 'ssh',
                'label' => 'SSH доступ (host/user/password)',
                'ok' => (bool) ($location->ssh_host && $location->ssh_user && $location->ssh_password),
                'hint' => 'Заполни SSH host/user/password в настройках локации.',
            ],
            [
                'key' => 'daemon_token',
                'label' => 'LOCATION_DAEMON_TOKEN (services.location_daemon.token)',
                'ok' => (string) config('services.location_daemon.token', '') !== '',
                'hint' => 'Укажи LOCATION_DAEMON_TOKEN в .env (или в config/services.php).',
            ],
            [
                'key' => 'panel_url',
                'label' => 'APP_URL (config(app.url))',
                'ok' => (string) config('app.url', '') !== '',
                'hint' => 'Укажи корректный APP_URL в .env (должен быть доступен с ноды).',
            ],
            [
                'key' => 'daemon_port',
                'label' => 'LOCATION_DAEMON_PORT (services.location_daemon.port)',
                'ok' => (int) config('services.location_daemon.port', 0) > 0,
                'hint' => 'Проверь services.location_daemon.port (обычно 9201) и доступность порта на ноде.',
            ],
        ];

        return view('admin.locations.setup', [
            'location' => $location,
            'statuses' => $statuses,
            'checks' => $checks,
        ]);
    }

    private function ensureSshConfigured(Location $location): ?JsonResponse
    {
        if (! $location->ssh_host || ! $location->ssh_user || ! $location->ssh_password) {
            return response()->json([
                'success' => false,
                'error' => 'Для локации не настроены SSH host/user/password.',
            ], 422);
        }

        return null;
    }

    public function getStatus(Location $location): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            \Log::error('Unauthorized access to getStatus');
            return response()->json(['error' => 'Unauthorized'], 403);
        }

        $cacheKey = "setup_status_{$location->id}_" . Auth::id();
        $status = \Cache::get($cacheKey, ['log' => '', 'completed' => false]);

        return response()->json($status);
    }

    public function installPackages(Location $location): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['error' => 'Unauthorized'], 403);
        }

        if ($resp = $this->ensureSshConfigured($location)) {
            return $resp;
        }

        $cacheKey = "setup_status_{$location->id}_" . Auth::id();
        \Cache::forget($cacheKey);
        \Cache::put($cacheKey, ['log' => 'Начинаем установку основных пакетов...', 'completed' => false], self::SETUP_CACHE_TTL_SEC);

        dispatch(new InstallComponent($location, 'packages', Auth::id()));

        return response()->json([
            'success' => true,
            'message' => 'Установка основных пакетов запущена.'
        ]);
    }

    public function installDocker(Location $location): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['error' => 'Unauthorized'], 403);
        }

        if ($resp = $this->ensureSshConfigured($location)) {
            return $resp;
        }

        $cacheKey = "setup_status_{$location->id}_" . Auth::id();
        \Cache::forget($cacheKey); 
        \Cache::put($cacheKey, ['log' => 'Начинаем установку Docker...', 'completed' => false], self::SETUP_CACHE_TTL_SEC);

        dispatch(new InstallComponent($location, 'docker', Auth::id()));

        return response()->json([
            'success' => true,
            'message' => 'Установка Docker запущена.'
        ]);
    }

    public function installMySQL(Location $location): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['error' => 'Unauthorized'], 403);
        }

        if ($resp = $this->ensureSshConfigured($location)) {
            return $resp;
        }

        $cacheKey = "setup_status_{$location->id}_" . Auth::id();
        \Cache::forget($cacheKey); 
        \Cache::put($cacheKey, ['log' => 'Начинаем установку MySQL...', 'completed' => false], self::SETUP_CACHE_TTL_SEC);

        dispatch(new InstallComponent($location, 'mysql', Auth::id()));

        return response()->json([
            'success' => true,
            'message' => 'Установка MySQL запущена.'
        ]);
    }

    public function installPhpMyAdmin(Location $location): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['error' => 'Unauthorized'], 403);
        }

        if ($resp = $this->ensureSshConfigured($location)) {
            return $resp;
        }

        $cacheKey = "setup_status_{$location->id}_" . Auth::id();
        \Cache::forget($cacheKey); 
        \Cache::put($cacheKey, ['log' => 'Начинаем установку phpMyAdmin...', 'completed' => false], self::SETUP_CACHE_TTL_SEC);

        dispatch(new InstallComponent($location, 'phpmyadmin', Auth::id()));

        return response()->json([
            'success' => true,
            'message' => 'Установка phpMyAdmin запущена.'
        ]);
    }

    public function installFTP(Location $location): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['error' => 'Unauthorized'], 403);
        }

        if ($resp = $this->ensureSshConfigured($location)) {
            return $resp;
        }

        $cacheKey = "setup_status_{$location->id}_" . Auth::id();
        \Cache::forget($cacheKey); 
        \Cache::put($cacheKey, ['log' => 'Начинаем установку FTP...', 'completed' => false], self::SETUP_CACHE_TTL_SEC);

        dispatch(new InstallComponent($location, 'ftp', Auth::id()));

        return response()->json([
            'success' => true,
            'message' => 'Установка FTP запущена.'
        ]);
    }

    public function installDaemon(Location $location): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['error' => 'Unauthorized'], 403);
        }

        if ($resp = $this->ensureSshConfigured($location)) {
            return $resp;
        }

        $cacheKey = "setup_status_{$location->id}_" . Auth::id();
        \Cache::forget($cacheKey);
        \Cache::put($cacheKey, ['log' => 'Начинаем установку Vortanix Daemon...', 'completed' => false], self::SETUP_CACHE_TTL_SEC);

        dispatch(new InstallComponent($location, 'daemon', Auth::id()));

        return response()->json([
            'success' => true,
            'message' => 'Установка Vortanix Daemon запущена.'
        ]);
    }

    public function buildImages(Location $location): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['error' => 'Unauthorized'], 403);
        }

        if ($resp = $this->ensureSshConfigured($location)) {
            return $resp;
        }

        $cacheKey = "setup_status_{$location->id}_" . Auth::id();
        \Cache::forget($cacheKey);
        \Cache::put($cacheKey, ['log' => 'Начинаем сборку игровых Docker образов...', 'completed' => false], self::SETUP_CACHE_TTL_SEC);

        dispatch(new InstallComponent($location, 'images', Auth::id()));

        return response()->json([
            'success' => true,
            'message' => 'Сборка образов запущена.'
        ]);
    }
}
