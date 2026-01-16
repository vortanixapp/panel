<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Setting;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Schema;
use Illuminate\View\View;

class LicenseController extends Controller
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
        $hb = Cache::get('license:cloud:status');
        $hbCheckedAt = is_array($hb) ? (int) $hb['checked_at'] : 0;
        $hbLastOkAt = is_array($hb) ? (int) $hb['last_ok_at'] : 0;
        $hbValid = is_array($hb) ? (bool) $hb['valid'] : false;

        return view('admin.license', [
            'licenseKey' => $licenseKey,
            'panelId' => $panelId,
            'serverIp' => $serverIp,
            'hbCheckedAt' => $hbCheckedAt,
            'hbLastOkAt' => $hbLastOkAt,
            'hbValid' => $hbValid,
        ]);
    }

    public function store(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        if (! Schema::hasTable('settings')) {
            return redirect()->route('admin.license')->with('error', 'Settings table not ready');
        }

        $data = $request->validate([
            'license_key' => ['required', 'string', 'min:8', 'max:128'],
        ]);

        Setting::setValue('license.key', trim((string) $data['license_key']));

        return redirect()->route('admin.license')->with('success', 'License saved');
    }

    public function clear(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        if (! Schema::hasTable('settings')) {
            return redirect()->route('admin.license')->with('error', 'Settings table not ready');
        }

        Setting::setValue('license.key', '');

        return redirect()->route('admin.license')->with('success', 'License cleared');
    }
}
