<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Setting;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\View\View;

class SettingsController extends Controller
{
    public function edit(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $values = Setting::allCached();

        return view('admin.settings.edit', [
            'values' => $values,
        ]);
    }

    public function update(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'app_name' => ['nullable', 'string', 'max:128'],
            'site_description' => ['nullable', 'string', 'max:10000'],
            'site_domain' => ['nullable', 'string', 'max:255'],
            'site_ip' => ['nullable', 'string', 'max:64'],
            'site_subnet' => ['nullable', 'string', 'max:64'],
            'default_template' => ['nullable', 'string', 'max:64'],

            'recaptcha_site_key' => ['nullable', 'string', 'max:255'],
            'recaptcha_secret_key' => ['nullable', 'string', 'max:255'],

            'telegram_url' => ['nullable', 'string', 'max:255'],
            'discord_url' => ['nullable', 'string', 'max:255'],
            'support_url' => ['nullable', 'string', 'max:255'],

            'logo' => ['nullable', 'file', 'max:5120'],
            'icon' => ['nullable', 'file', 'max:2048'],

            'mail_mailer' => ['nullable', 'string', 'max:32'],
            'mail_host' => ['nullable', 'string', 'max:255'],
            'mail_port' => ['nullable', 'integer', 'min:1', 'max:65535'],
            'mail_username' => ['nullable', 'string', 'max:255'],
            'mail_password' => ['nullable', 'string', 'max:255'],
            'mail_scheme' => ['nullable', 'string', 'max:32'],
            'mail_from_address' => ['nullable', 'string', 'max:255'],
            'mail_from_name' => ['nullable', 'string', 'max:255'],

            'freekassa_merchant_id' => ['nullable', 'string', 'max:64'],
            'freekassa_secret1' => ['nullable', 'string', 'max:255'],
            'freekassa_secret2' => ['nullable', 'string', 'max:255'],
            'freekassa_currency' => ['nullable', 'string', 'max:16'],
            'freekassa_pay_url' => ['nullable', 'string', 'max:255'],
            'freekassa_methods_json' => ['nullable', 'string', 'max:100000'],

            'nowpayments_api_key' => ['nullable', 'string', 'max:255'],
            'nowpayments_ipn_secret' => ['nullable', 'string', 'max:255'],
            'nowpayments_api_url' => ['nullable', 'string', 'max:255'],
            'nowpayments_price_currency' => ['nullable', 'string', 'max:16'],
            'nowpayments_pay_currency' => ['nullable', 'string', 'max:16'],
        ]);

        $validated = array_merge([
            'app_name' => null,
            'site_description' => null,
            'site_domain' => null,
            'site_ip' => null,
            'site_subnet' => null,
            'default_template' => null,
            'recaptcha_site_key' => null,
            'recaptcha_secret_key' => null,
            'telegram_url' => null,
            'discord_url' => null,
            'support_url' => null,
            'mail_mailer' => null,
            'mail_host' => null,
            'mail_port' => null,
            'mail_username' => null,
            'mail_password' => null,
            'mail_scheme' => null,
            'mail_from_address' => null,
            'mail_from_name' => null,
            'freekassa_merchant_id' => null,
            'freekassa_secret1' => null,
            'freekassa_secret2' => null,
            'freekassa_currency' => null,
            'freekassa_pay_url' => null,
            'freekassa_methods_json' => null,
            'nowpayments_api_key' => null,
            'nowpayments_ipn_secret' => null,
            'nowpayments_api_url' => null,
            'nowpayments_price_currency' => null,
            'nowpayments_pay_currency' => null,
        ], (array) $validated);

        Setting::setValue('app.name', (string) $validated['app_name']);
        Setting::setValue('app.site.description', (string) $validated['site_description']);
        Setting::setValue('app.site.domain', (string) $validated['site_domain']);
        Setting::setValue('app.site.ip', (string) $validated['site_ip']);
        Setting::setValue('app.site.subnet', (string) $validated['site_subnet']);
        Setting::setValue('app.site.default_template', (string) $validated['default_template']);

        Setting::setValue('services.recaptcha.site_key', (string) $validated['recaptcha_site_key']);
        Setting::setValue('services.recaptcha.secret_key', (string) $validated['recaptcha_secret_key']);

        Setting::setValue('app.links.telegram', (string) $validated['telegram_url']);
        Setting::setValue('app.links.discord', (string) $validated['discord_url']);
        Setting::setValue('app.links.support', (string) $validated['support_url']);

        $logo = $request->file('logo');
        if ($logo && $logo->isValid()) {
            $ext = strtolower((string) $logo->getClientOriginalExtension());
            if (! in_array($ext, ['png', 'jpg', 'jpeg', 'webp', 'svg'], true)) {
                return back()->withErrors(['logo' => 'Поддерживаются: png, jpg, jpeg, webp, svg'])->withInput();
            }
            $path = $logo->storeAs('branding', 'logo.' . $ext, 'public');
            Setting::setValue('app.branding.logo', (string) $path);
        }

        $icon = $request->file('icon');
        if ($icon && $icon->isValid()) {
            $ext = strtolower((string) $icon->getClientOriginalExtension());
            if (! in_array($ext, ['png', 'ico', 'jpg', 'jpeg', 'webp', 'svg'], true)) {
                return back()->withErrors(['icon' => 'Поддерживаются: png, ico, jpg, jpeg, webp, svg'])->withInput();
            }
            $path = $icon->storeAs('branding', 'icon.' . $ext, 'public');
            Setting::setValue('app.branding.icon', (string) $path);
        }

        Setting::setValue('mail.default', (string) $validated['mail_mailer']);
        Setting::setValue('mail.mailers.smtp.host', (string) $validated['mail_host']);
        Setting::setValue('mail.mailers.smtp.port', (string) $validated['mail_port']);
        Setting::setValue('mail.mailers.smtp.username', (string) $validated['mail_username']);
        Setting::setValue('mail.mailers.smtp.password', (string) $validated['mail_password']);
        Setting::setValue('mail.mailers.smtp.scheme', (string) $validated['mail_scheme']);
        Setting::setValue('mail.from.address', (string) $validated['mail_from_address']);
        Setting::setValue('mail.from.name', (string) $validated['mail_from_name']);

        Setting::setValue('services.freekassa.merchant_id', (string) $validated['freekassa_merchant_id']);
        Setting::setValue('services.freekassa.secret1', (string) $validated['freekassa_secret1']);
        Setting::setValue('services.freekassa.secret2', (string) $validated['freekassa_secret2']);
        Setting::setValue('services.freekassa.currency', (string) $validated['freekassa_currency']);
        Setting::setValue('services.freekassa.pay_url', (string) $validated['freekassa_pay_url']);
        Setting::setValue('services.freekassa.methods', (string) $validated['freekassa_methods_json']);

        Setting::setValue('services.nowpayments.api_key', (string) $validated['nowpayments_api_key']);
        Setting::setValue('services.nowpayments.ipn_secret', (string) $validated['nowpayments_ipn_secret']);
        Setting::setValue('services.nowpayments.api_url', (string) $validated['nowpayments_api_url']);
        Setting::setValue('services.nowpayments.price_currency', (string) $validated['nowpayments_price_currency']);
        Setting::setValue('services.nowpayments.pay_currency', (string) $validated['nowpayments_pay_currency']);

        return redirect()->route('admin.settings.edit')->with('success', 'Настройки сохранены');
    }
}
