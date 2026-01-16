@extends('layouts.app-admin')

@section('page_title', 'Язык')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex items-center justify-between gap-3">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Язык</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Настройки языков</p>
                </div>
            </div>

            @if(session('success'))
                <div class="rounded-xl bg-emerald-500/10 px-4 py-3 text-xs text-emerald-200 ring-1 ring-emerald-500/20">{{ session('success') }}</div>
            @endif

            <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
                <div class="border-b border-white/10 bg-black/10 px-5 py-4">
                    <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Настройки</div>
                </div>
                <div class="p-5">
                    <form method="POST" action="{{ route('admin.language.update') }}" class="space-y-5">
                        @csrf

                        <div class="grid gap-4 md:grid-cols-2">
                            <div class="space-y-1">
                                <div class="text-xs text-slate-300/70">Язык по умолчанию</div>
                                <input name="default_locale" value="{{ old('default_locale', (string) ($values['app.locale.default'] ?? config('app.locale', 'en'))) }}" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100" placeholder="en" required>
                                @error('default_locale')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                            </div>

                            <div class="space-y-1">
                                <div class="text-xs text-slate-300/70">Запасной язык (fallback)</div>
                                <input name="fallback_locale" value="{{ old('fallback_locale', (string) ($values['app.locale.fallback'] ?? config('app.fallback_locale', 'en'))) }}" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100" placeholder="en" required>
                                @error('fallback_locale')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                            </div>
                        </div>

                        <div class="space-y-1">
                            <div class="text-xs text-slate-300/70">Включённые языки (через запятую)</div>
                            <input name="enabled_locales" value="{{ old('enabled_locales', (string) ($values['app.locale.enabled'] ?? 'en,ru')) }}" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100" placeholder="en,ru,de" required>
                            @error('enabled_locales')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                        </div>

                        <div class="flex items-center gap-3">
                            <button type="submit" class="rounded-xl bg-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">Сохранить</button>
                        </div>
                    </form>
                </div>
            </div>

            <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
                <div class="border-b border-white/10 bg-black/10 px-5 py-4">
                    <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Тестирование</div>
                </div>
                <div class="p-5">
                    <p class="text-sm text-slate-300/80 mb-3">Текущий язык: <strong>{{ app()->getLocale() }}</strong></p>
                    <div class="flex gap-2">
                        @foreach(explode(',', (string) ($values['app.locale.enabled'] ?? 'en,ru')) as $l)
                            @if(trim($l) !== '')
                                <form method="POST" action="{{ route('locale.switch', trim($l)) }}">
                                    @csrf
                                    <button type="submit" class="rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 hover:bg-black/15">{{ trim($l) }}</button>
                                </form>
                            @endif
                        @endforeach
                    </div>
                </div>
            </div>
        </div>
    </section>
@endsection
