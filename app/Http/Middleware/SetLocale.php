<?php

namespace App\Http\Middleware;

use App\Models\Setting;
use Closure;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Config;
use Illuminate\Support\Facades\Schema;
use Symfony\Component\HttpFoundation\Response;

class SetLocale
{
    public function handle(Request $request, Closure $next): Response
    {
        $enabledList = ['en', 'ru'];
        $default = (string) config('app.locale', 'en');

        $canUseDatabaseSettings = true;
        $defaultConnection = (string) Config::get('database.default', '');
        if ($defaultConnection === '') {
            $canUseDatabaseSettings = false;
        }

        if ($defaultConnection === 'mysql') {
            $mysqlHost = (string) Config::get('database.connections.mysql.host', '');
            $mysqlDatabase = (string) Config::get('database.connections.mysql.database', '');
            $mysqlUsername = (string) Config::get('database.connections.mysql.username', '');
            if ($mysqlHost === '' || $mysqlDatabase === '' || $mysqlUsername === '') {
                $canUseDatabaseSettings = false;
            }
        }

        if ($canUseDatabaseSettings) {
            try {
                if (Schema::hasTable('settings')) {
                    $enabled = (string) Setting::getValue('app.locale.enabled', '');
                    $fromSettings = array_values(array_filter(array_map('trim', explode(',', $enabled)), fn ($v) => $v !== ''));
                    if (count($fromSettings) > 0) {
                        $enabledList = $fromSettings;
                    }

                    $fromSettingsDefault = (string) Setting::getValue('app.locale.default', '');
                    if ($fromSettingsDefault !== '') {
                        $default = $fromSettingsDefault;
                    }
                }
            } catch (\Throwable $e) {
                //
            }
        }

        if ($default === '') {
            $default = 'en';
        }

        $preferred = null;
        $user = $request->user();
        if ($user && ! empty($user->locale)) {
            $preferred = (string) $user->locale;
        }

        if (! $preferred && $request->session()->has('locale')) {
            $preferred = (string) $request->session()->get('locale');
        }

        $preferred = $preferred ?: $default;

        if (! in_array($preferred, $enabledList, true)) {
            $preferred = $default;
        }

        app()->setLocale($preferred);

        return $next($request);
    }
}
