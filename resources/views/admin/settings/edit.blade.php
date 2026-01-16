@extends('layouts.app-admin')

@section('page_title', 'Настройки')

@section('content')
    <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
        <div class="border-b border-white/10 bg-black/10 px-4 py-3 text-[11px] uppercase tracking-wide text-slate-300/70">
            Настройки панели
        </div>
        <div class="p-4" x-data="{ tab: 'main' }">
            @if(session('success'))
                <div class="mb-4 rounded-md bg-emerald-50 p-3 text-xs text-emerald-800">
                    {{ session('success') }}
                </div>
            @endif

            <div class="mb-4 flex flex-wrap gap-1 rounded-2xl border border-white/10 bg-black/10 p-1 text-[12px] text-slate-200 shadow-sm">
                <button
                    type="button"
                    class="inline-flex items-center rounded-xl px-3 py-2 transition-all duration-200"
                    :class="tab === 'main' ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white'"
                    @click="tab = 'main'"
                >
                    Основное
                </button>
                <button
                    type="button"
                    class="inline-flex items-center rounded-xl px-3 py-2 transition-all duration-200"
                    :class="tab === 'payments' ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white'"
                    @click="tab = 'payments'"
                >
                    Платёжные системы
                </button>
                <button
                    type="button"
                    class="inline-flex items-center rounded-xl px-3 py-2 transition-all duration-200"
                    :class="tab === 'mail' ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white'"
                    @click="tab = 'mail'"
                >
                    Почта
                </button>
            </div>

            <form method="POST" action="{{ route('admin.settings.update') }}" enctype="multipart/form-data" class="space-y-6">
                @csrf

                <div x-show="tab === 'main'" x-transition:enter="transition ease-out duration-200" x-transition:enter-start="opacity-0 translate-y-1" x-transition:enter-end="opacity-100 translate-y-0" class="space-y-6">
                    <div class="grid gap-4 md:grid-cols-2">
                        <div class="space-y-1">
                            <div class="text-slate-500 text-xs">Название проекта</div>
                            <input name="app_name" value="{{ old('app_name', (string) ($values['app.name'] ?? config('app.name', '')) ) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400">
                            @error('app_name')
                                <div class="text-xs text-red-400">{{ $message }}</div>
                            @enderror
                        </div>

                        <div class="space-y-1 md:col-span-2">
                            <div class="text-slate-500 text-xs">Описание сайта</div>
                            <textarea name="site_description" rows="3" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400">{{ old('site_description', (string) ($values['app.site.description'] ?? '')) }}</textarea>
                            @error('site_description')
                                <div class="text-xs text-red-400">{{ $message }}</div>
                            @enderror
                        </div>

                        <div class="space-y-1">
                            <div class="text-slate-500 text-xs">Домен</div>
                            <input name="site_domain" value="{{ old('site_domain', (string) ($values['app.site.domain'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="example.com">
                            @error('site_domain')
                                <div class="text-xs text-red-400">{{ $message }}</div>
                            @enderror
                        </div>

                        <div class="space-y-1">
                            <div class="text-slate-500 text-xs">Шаблон по умолчанию</div>
                            <input name="default_template" value="{{ old('default_template', (string) ($values['app.site.default_template'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="default">
                            @error('default_template')
                                <div class="text-xs text-red-400">{{ $message }}</div>
                            @enderror
                        </div>

                        <div class="space-y-1">
                            <div class="text-slate-500 text-xs">IP-Адрес сайта</div>
                            <input name="site_ip" value="{{ old('site_ip', (string) ($values['app.site.ip'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="1.2.3.4">
                            @error('site_ip')
                                <div class="text-xs text-red-400">{{ $message }}</div>
                            @enderror
                        </div>

                        <div class="space-y-1">
                            <div class="text-slate-500 text-xs">Подсеть сайта</div>
                            <input name="site_subnet" value="{{ old('site_subnet', (string) ($values['app.site.subnet'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="1.2.3.0/24">
                            @error('site_subnet')
                                <div class="text-xs text-red-400">{{ $message }}</div>
                            @enderror
                        </div>

                        <div class="space-y-1">
                            <div class="text-slate-500 text-xs">Telegram</div>
                            <input name="telegram_url" value="{{ old('telegram_url', (string) ($values['app.links.telegram'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="https://t.me/...">
                            @error('telegram_url')
                                <div class="text-xs text-red-400">{{ $message }}</div>
                            @enderror
                        </div>

                        <div class="space-y-1">
                            <div class="text-slate-500 text-xs">Discord</div>
                            <input name="discord_url" value="{{ old('discord_url', (string) ($values['app.links.discord'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="https://discord.gg/...">
                            @error('discord_url')
                                <div class="text-xs text-red-400">{{ $message }}</div>
                            @enderror
                        </div>

                        <div class="space-y-1">
                            <div class="text-slate-500 text-xs">Support</div>
                            <input name="support_url" value="{{ old('support_url', (string) ($values['app.links.support'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="https://...">
                            @error('support_url')
                                <div class="text-xs text-red-400">{{ $message }}</div>
                            @enderror
                        </div>

                        <div class="space-y-1">
                            <div class="text-slate-500 text-xs">Лого</div>
                            <input type="file" name="logo" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                            @error('logo')
                                <div class="text-xs text-red-400">{{ $message }}</div>
                            @enderror
                            @php $logo = (string) ($values['app.branding.logo'] ?? ''); @endphp
                            @if($logo !== '')
                                <div class="text-xs text-slate-300/70">Текущее: <a class="underline" href="{{ \Illuminate\Support\Facades\Storage::disk('public')->url($logo) }}" target="_blank">{{ $logo }}</a></div>
                            @endif
                        </div>

                        <div class="space-y-1">
                            <div class="text-slate-500 text-xs">Иконка</div>
                            <input type="file" name="icon" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                            @error('icon')
                                <div class="text-xs text-red-400">{{ $message }}</div>
                            @enderror
                            @php $icon = (string) ($values['app.branding.icon'] ?? ''); @endphp
                            @if($icon !== '')
                                <div class="text-xs text-slate-300/70">Текущая: <a class="underline" href="{{ \Illuminate\Support\Facades\Storage::disk('public')->url($icon) }}" target="_blank">{{ $icon }}</a></div>
                            @endif
                        </div>

                        <div class="space-y-1">
                            <div class="text-slate-500 text-xs">ReCaptcha Site Key</div>
                            <input name="recaptcha_site_key" value="{{ old('recaptcha_site_key', (string) ($values['services.recaptcha.site_key'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400">
                            @error('recaptcha_site_key')
                                <div class="text-xs text-red-400">{{ $message }}</div>
                            @enderror
                        </div>

                        <div class="space-y-1">
                            <div class="text-slate-500 text-xs">ReCaptcha Secret Key</div>
                            <input name="recaptcha_secret_key" value="{{ old('recaptcha_secret_key', (string) ($values['services.recaptcha.secret_key'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400">
                            @error('recaptcha_secret_key')
                                <div class="text-xs text-red-400">{{ $message }}</div>
                            @enderror
                        </div>
                    </div>
                </div>

                <div x-show="tab === 'payments'" x-transition:enter="transition ease-out duration-200" x-transition:enter-start="opacity-0 translate-y-1" x-transition:enter-end="opacity-100 translate-y-0" class="space-y-6">
                    <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                        <div class="text-xs font-semibold text-slate-200">Платёжка (FreeKassa)</div>
                        <div class="mt-3 grid gap-4 md:grid-cols-2">
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">Merchant ID</div>
                                <input name="freekassa_merchant_id" value="{{ old('freekassa_merchant_id', (string) ($values['services.freekassa.merchant_id'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                            </div>
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">Currency</div>
                                <input name="freekassa_currency" value="{{ old('freekassa_currency', (string) ($values['services.freekassa.currency'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="RUB">
                            </div>
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">Secret 1</div>
                                <input name="freekassa_secret1" value="{{ old('freekassa_secret1', (string) ($values['services.freekassa.secret1'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                            </div>
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">Secret 2</div>
                                <input name="freekassa_secret2" value="{{ old('freekassa_secret2', (string) ($values['services.freekassa.secret2'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                            </div>
                            <div class="space-y-1 md:col-span-2">
                                <div class="text-slate-500 text-xs">Pay URL</div>
                                <input name="freekassa_pay_url" value="{{ old('freekassa_pay_url', (string) ($values['services.freekassa.pay_url'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="https://pay.fk.money/">
                            </div>
                            <div class="space-y-1 md:col-span-2">
                                <div class="text-slate-500 text-xs">Methods JSON</div>
                                <textarea name="freekassa_methods_json" rows="6" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400">{{ old('freekassa_methods_json', (string) ($values['services.freekassa.methods'] ?? '')) }}</textarea>
                            </div>
                        </div>
                    </div>

                    <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                        <div class="text-xs font-semibold text-slate-200">Платёжка (NowPayments)</div>
                        <div class="mt-3 grid gap-4 md:grid-cols-2">
                            <div class="space-y-1 md:col-span-2">
                                <div class="text-slate-500 text-xs">API URL</div>
                                <input name="nowpayments_api_url" value="{{ old('nowpayments_api_url', (string) ($values['services.nowpayments.api_url'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="https://api.nowpayments.io">
                                @error('nowpayments_api_url')
                                    <div class="text-xs text-red-400">{{ $message }}</div>
                                @enderror
                            </div>
                            <div class="space-y-1 md:col-span-2">
                                <div class="text-slate-500 text-xs">API Key</div>
                                <input name="nowpayments_api_key" value="{{ old('nowpayments_api_key', (string) ($values['services.nowpayments.api_key'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                @error('nowpayments_api_key')
                                    <div class="text-xs text-red-400">{{ $message }}</div>
                                @enderror
                            </div>
                            <div class="space-y-1 md:col-span-2">
                                <div class="text-slate-500 text-xs">IPN Secret</div>
                                <input name="nowpayments_ipn_secret" value="{{ old('nowpayments_ipn_secret', (string) ($values['services.nowpayments.ipn_secret'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                @error('nowpayments_ipn_secret')
                                    <div class="text-xs text-red-400">{{ $message }}</div>
                                @enderror
                            </div>
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">Price currency</div>
                                <input name="nowpayments_price_currency" value="{{ old('nowpayments_price_currency', (string) ($values['services.nowpayments.price_currency'] ?? 'RUB')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="RUB">
                                @error('nowpayments_price_currency')
                                    <div class="text-xs text-red-400">{{ $message }}</div>
                                @enderror
                            </div>
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">Pay currency</div>
                                <input name="nowpayments_pay_currency" value="{{ old('nowpayments_pay_currency', (string) ($values['services.nowpayments.pay_currency'] ?? 'usdt')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="usdt">
                                @error('nowpayments_pay_currency')
                                    <div class="text-xs text-red-400">{{ $message }}</div>
                                @enderror
                            </div>
                        </div>
                    </div>
                </div>

                <div x-show="tab === 'mail'" x-transition:enter="transition ease-out duration-200" x-transition:enter-start="opacity-0 translate-y-1" x-transition:enter-end="opacity-100 translate-y-0" class="space-y-6">
                    <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                        <div class="text-xs font-semibold text-slate-200">Почта (SMTP)</div>
                        <div class="mt-3 grid gap-4 md:grid-cols-2">
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">Mailer</div>
                                <input name="mail_mailer" value="{{ old('mail_mailer', (string) ($values['mail.default'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="smtp">
                                @error('mail_mailer')
                                    <div class="text-xs text-red-400">{{ $message }}</div>
                                @enderror
                            </div>
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">Scheme</div>
                                <input name="mail_scheme" value="{{ old('mail_scheme', (string) ($values['mail.mailers.smtp.scheme'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="tls">
                                @error('mail_scheme')
                                    <div class="text-xs text-red-400">{{ $message }}</div>
                                @enderror
                            </div>
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">Host</div>
                                <input name="mail_host" value="{{ old('mail_host', (string) ($values['mail.mailers.smtp.host'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="smtp.example.com">
                                @error('mail_host')
                                    <div class="text-xs text-red-400">{{ $message }}</div>
                                @enderror
                            </div>
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">Port</div>
                                <input name="mail_port" value="{{ old('mail_port', (string) ($values['mail.mailers.smtp.port'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="587">
                                @error('mail_port')
                                    <div class="text-xs text-red-400">{{ $message }}</div>
                                @enderror
                            </div>
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">Username</div>
                                <input name="mail_username" value="{{ old('mail_username', (string) ($values['mail.mailers.smtp.username'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                @error('mail_username')
                                    <div class="text-xs text-red-400">{{ $message }}</div>
                                @enderror
                            </div>
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">Password</div>
                                <input name="mail_password" value="{{ old('mail_password', (string) ($values['mail.mailers.smtp.password'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                @error('mail_password')
                                    <div class="text-xs text-red-400">{{ $message }}</div>
                                @enderror
                            </div>
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">From address</div>
                                <input name="mail_from_address" value="{{ old('mail_from_address', (string) ($values['mail.from.address'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="no-reply@example.com">
                                @error('mail_from_address')
                                    <div class="text-xs text-red-400">{{ $message }}</div>
                                @enderror
                            </div>
                            <div class="space-y-1">
                                <div class="text-slate-500 text-xs">From name</div>
                                <input name="mail_from_name" value="{{ old('mail_from_name', (string) ($values['mail.from.name'] ?? '')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="{{ config('app.name') }}">
                                @error('mail_from_name')
                                    <div class="text-xs text-red-400">{{ $message }}</div>
                                @enderror
                            </div>
                        </div>
                    </div>
                </div>

                <div>
                    <button type="submit" class="inline-flex items-center rounded-md bg-sky-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-sky-500">
                        Сохранить
                    </button>
                </div>
            </form>
        </div>
    </div>
@endsection
