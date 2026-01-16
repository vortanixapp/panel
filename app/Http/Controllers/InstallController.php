<?php

namespace App\Http\Controllers;

use App\Models\Setting;
use App\Models\User;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;
use Illuminate\Support\Str;
use Illuminate\View\View;

class InstallController extends Controller
{
    public function show(Request $request): View|RedirectResponse
    {
        if ($this->isInstalled()) {
            return redirect()->route('dashboard');
        }

        $checks = [
            ['key' => 'php', 'label' => 'PHP >= 8.2', 'ok' => version_compare(PHP_VERSION, '8.2.0', '>=')],
            ['key' => 'ext_pdo', 'label' => 'ext-pdo', 'ok' => extension_loaded('pdo')],
            ['key' => 'ext_pdo_mysql', 'label' => 'ext-pdo_mysql', 'ok' => extension_loaded('pdo_mysql')],
            ['key' => 'ext_mbstring', 'label' => 'ext-mbstring', 'ok' => extension_loaded('mbstring')],
            ['key' => 'ext_openssl', 'label' => 'ext-openssl', 'ok' => extension_loaded('openssl')],
            ['key' => 'ext_json', 'label' => 'ext-json', 'ok' => extension_loaded('json')],
            ['key' => 'storage_writable', 'label' => 'storage/ writable', 'ok' => is_writable(storage_path())],
            ['key' => 'bootstrap_cache_writable', 'label' => 'bootstrap/cache writable', 'ok' => is_writable(base_path('bootstrap/cache'))],
        ];

        $dbOk = false;
        $dbError = '';
        try {
            $defaultConnection = (string) config('database.default', '');
            if ($defaultConnection === '') {
                throw new \RuntimeException('DB_CONNECTION is empty');
            }

            DB::connection($defaultConnection)->getPdo();
            $dbOk = true;
        } catch (\Throwable $e) {
            $dbOk = false;
            $dbError = $e->getMessage();
        }

        $envWritable = is_file(base_path('.env')) ? is_writable(base_path('.env')) : is_writable(base_path());
        $appKeyPresent = (string) config('app.key', '') !== '';

        $migrationsApplied = false;
        try {
            $migrationsApplied = Schema::hasTable('users') && Schema::hasTable('settings');
        } catch (\Throwable $e) {
            $migrationsApplied = false;
        }

        return view('install', [
            'checks' => $checks,
            'dbOk' => $dbOk,
            'dbError' => $dbError,
            'envWritable' => $envWritable,
            'appKeyPresent' => $appKeyPresent,
            'migrationsApplied' => $migrationsApplied,
            'db' => [
                'connection' => (string) env('DB_CONNECTION', 'mysql'),
                'host' => (string) env('DB_HOST', '127.0.0.1'),
                'port' => (string) env('DB_PORT', '3306'),
                'database' => (string) env('DB_DATABASE', ''),
                'username' => (string) env('DB_USERNAME', ''),
                'password' => (string) env('DB_PASSWORD', ''),
            ],
            'license' => [
                'url' => (string) env('LICENSE_CLOUD_URL', ''),
                'panel_id' => (string) env('LICENSE_CLOUD_PANEL_ID', ''),
                'hmac_secret' => (string) env('LICENSE_CLOUD_HMAC_SECRET', ''),
                'server_ip' => (string) env('LICENSE_CLOUD_SERVER_IP', ''),
            ],
        ]);
    }

    public function submit(Request $request): RedirectResponse
    {
        if ($this->isInstalled()) {
            return redirect()->route('dashboard');
        }

        $data = $request->validate([
            'db_host' => ['required', 'string', 'max:255'],
            'db_port' => ['required', 'string', 'max:10'],
            'db_database' => ['required', 'string', 'max:255'],
            'db_username' => ['required', 'string', 'max:255'],
            'db_password' => ['nullable', 'string', 'max:255'],
            'license_cloud_url' => ['required', 'string', 'max:255'],
            'license_cloud_panel_id' => ['required', 'string', 'max:255'],
            'license_cloud_hmac_secret' => ['required', 'string', 'max:512'],
            'license_cloud_server_ip' => ['nullable', 'string', 'max:255'],
            'admin_name' => ['required', 'string', 'max:255'],
            'admin_email' => ['required', 'email', 'max:255'],
            'admin_password' => ['required', 'string', 'min:8', 'confirmed'],
        ]);

        $this->tryWriteEnvValue('DB_CONNECTION', 'mysql');
        $this->tryWriteEnvValue('DB_HOST', $data['db_host']);
        $this->tryWriteEnvValue('DB_PORT', $data['db_port']);
        $this->tryWriteEnvValue('DB_DATABASE', $data['db_database']);
        $this->tryWriteEnvValue('DB_USERNAME', $data['db_username']);
        $this->tryWriteEnvValue('DB_PASSWORD', (string) $data['db_password']);

        $this->tryWriteEnvValue('LICENSE_CLOUD_URL', $data['license_cloud_url']);
        $this->tryWriteEnvValue('LICENSE_CLOUD_PANEL_ID', $data['license_cloud_panel_id']);
        $this->tryWriteEnvValue('LICENSE_CLOUD_HMAC_SECRET', $data['license_cloud_hmac_secret']);
        $this->tryWriteEnvValue('LICENSE_CLOUD_SERVER_IP', (string) $data['license_cloud_server_ip']);

        config()->set('database.default', 'mysql');
        config()->set('database.connections.mysql.host', $data['db_host']);
        config()->set('database.connections.mysql.port', $data['db_port']);
        config()->set('database.connections.mysql.database', $data['db_database']);
        config()->set('database.connections.mysql.username', $data['db_username']);
        config()->set('database.connections.mysql.password', (string) $data['db_password']);

        DB::purge('mysql');
        DB::setDefaultConnection('mysql');

        try {
            DB::connection('mysql')->getPdo();
        } catch (\Throwable $e) {
            return back()->withErrors(['db' => 'Database connection failed: ' . $e->getMessage()])->withInput();
        }

        try {
            Artisan::call('migrate', ['--force' => true]);
        } catch (\Throwable $e) {
            return back()->withErrors(['migrate' => 'Migration failed: ' . $e->getMessage()])->withInput();
        }

        if ((string) config('app.key', '') === '') {
            $key = 'base64:' . base64_encode(random_bytes(32));
            $this->tryWriteEnvValue('APP_KEY', $key);
            config()->set('app.key', $key);
        }

        $user = User::query()->where('email', $data['admin_email'])->first();
        if (! $user) {
            $user = User::query()->create([
                'name' => $data['admin_name'],
                'email' => $data['admin_email'],
                'password' => $data['admin_password'],
                'is_admin' => true,
                'public_id' => Str::lower(Str::random(12)),
                'locale' => (string) config('app.locale', 'en'),
            ]);
        } else {
            $user->is_admin = true;
            $user->save();
        }

        Setting::setValue('app.installed', '1');

        $this->tryWriteEnvValue('SESSION_DRIVER', 'database');
        $this->tryWriteEnvValue('CACHE_STORE', 'database');
        $this->tryWriteEnvValue('QUEUE_CONNECTION', 'database');

        Auth::login($user);

        return redirect()->route('admin.dashboard');
    }

    private function isInstalled(): bool
    {
        try {
            if (! Schema::hasTable('settings')) {
                return false;
            }

            return (string) Setting::getValue('app.installed', '') === '1';
        } catch (\Throwable $e) {
            return false;
        }
    }

    private function tryWriteEnvValue(string $key, string $value): void
    {
        $path = base_path('.env');

        if (! is_file($path)) {
            return;
        }

        if (! is_writable($path)) {
            return;
        }

        $contents = file_get_contents($path);
        if ($contents === false) {
            return;
        }

        $line = $key . '=' . $value;

        if (preg_match('/^' . preg_quote($key, '/') . '=.*/m', $contents) === 1) {
            $contents = preg_replace('/^' . preg_quote($key, '/') . '=.*/m', $line, $contents);
        } else {
            $contents = rtrim($contents) . "\n" . $line . "\n";
        }

        file_put_contents($path, $contents);
    }
}
