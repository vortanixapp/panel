<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Setting;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\View\View;

class LanguageController extends Controller
{
    public function edit(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $values = Setting::allCached();

        return view('admin.language.edit', [
            'values' => $values,
        ]);
    }

    public function update(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'default_locale' => ['required', 'string', 'max:10'],
            'fallback_locale' => ['required', 'string', 'max:10'],
            'enabled_locales' => ['required', 'string', 'max:255'],
        ]);

        $enabled = array_values(array_filter(array_map('trim', explode(',', (string) $validated['enabled_locales'])), fn ($v) => $v !== ''));
        $enabled = array_values(array_unique($enabled));
        if (count($enabled) === 0) {
            $enabled = ['en'];
        }

        $default = (string) $validated['default_locale'];
        $fallback = (string) $validated['fallback_locale'];
        if (! in_array($default, $enabled, true)) {
            $enabled[] = $default;
        }
        if (! in_array($fallback, $enabled, true)) {
            $enabled[] = $fallback;
        }

        Setting::setValue('app.locale.default', $default);
        Setting::setValue('app.locale.fallback', $fallback);
        Setting::setValue('app.locale.enabled', implode(',', $enabled));

        return redirect()->route('admin.language.edit')->with('success', 'Настройки языка сохранены');
    }
}
