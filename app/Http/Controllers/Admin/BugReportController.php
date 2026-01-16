<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Setting;
use App\Services\LicenseCloudClient;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Schema;
use Illuminate\View\View;

class BugReportController extends Controller
{
    public function index(Request $request): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $licenseKey = '';
        if (Schema::hasTable('settings')) {
            $licenseKey = (string) Setting::getValue('license.key', '');
        }

        $panelId = (string) config('services.license_cloud.panel_id', '');
        $serverIp = (string) (config('services.license_cloud.server_ip', '') ?: (string) config('app.site.ip'));
        $appVersion = (string) config('app.version', '');

        $reports = [];
        try {
            $resp = app(LicenseCloudClient::class)->listBugReports(50);
            $reports = is_array($resp['data']['items']) ? $resp['data']['items'] : [];
        } catch (\Throwable $e) {
            $reports = [];
        }

        return view('admin.bug-report', [
            'panelId' => $panelId,
            'serverIp' => $serverIp,
            'appVersion' => $appVersion,
            'licenseKey' => $licenseKey,
            'reports' => $reports,
        ]);
    }

    public function store(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $data = $request->validate([
            'title' => ['required', 'string', 'max:255'],
            'description' => ['required', 'string', 'max:20000'],
            'steps' => ['nullable', 'string', 'max:20000'],
            'expected' => ['nullable', 'string', 'max:20000'],
            'actual' => ['nullable', 'string', 'max:20000'],
            'severity' => ['nullable', 'string', 'max:32'],
        ]);

        $data = array_merge([
            'steps' => null,
            'expected' => null,
            'actual' => null,
            'severity' => null,
        ], (array) $data);

        $licenseKey = '';
        if (Schema::hasTable('settings')) {
            $licenseKey = (string) Setting::getValue('license.key', '');
        }

        $serverIp = (string) (config('services.license_cloud.server_ip', '') ?: (string) config('app.site.ip'));

        try {
            app(LicenseCloudClient::class)->postBugReport([
                'title' => (string) $data['title'],
                'description' => (string) $data['description'],
                'steps' => $data['steps'] !== null ? (string) $data['steps'] : null,
                'expected' => $data['expected'] !== null ? (string) $data['expected'] : null,
                'actual' => $data['actual'] !== null ? (string) $data['actual'] : null,
                'severity' => (string) $data['severity'],
                'license_key' => $licenseKey !== '' ? $licenseKey : null,
                'server_ip' => $serverIp !== '' ? $serverIp : null,
                'app_version' => (string) config('app.version', ''),
            ]);
        } catch (\Throwable $e) {
            return redirect()->route('admin.bug-report')->with('error', 'Не удалось отправить баг-репорт: ' . $e->getMessage());
        }

        return redirect()->route('admin.bug-report')->with('success', 'Баг-репорт отправлен');
    }
}
