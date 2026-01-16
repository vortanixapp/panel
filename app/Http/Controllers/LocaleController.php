<?php

namespace App\Http\Controllers;

use App\Models\Setting;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;

class LocaleController extends Controller
{
    public function switch(Request $request, string $locale): RedirectResponse
    {
        $enabled = (string) Setting::getValue('app.locale.enabled', '');
        $enabledList = array_values(array_filter(array_map('trim', explode(',', $enabled)), fn ($v) => $v !== ''));
        if (count($enabledList) === 0) {
            $enabledList = ['en', 'ru'];
        }

        $locale = trim((string) $locale);
        if (! in_array($locale, $enabledList, true)) {
            $locale = (string) Setting::getValue('app.locale.default', config('app.locale', 'en'));
            $locale = $locale !== '' ? $locale : 'en';
        }

        $request->session()->put('locale', $locale);

        $user = Auth::user();
        if ($user) {
            $user->locale = $locale;
            $user->save();
        }

        return redirect()->back();
    }
}
