<!DOCTYPE html>
<html lang="{{ str_replace('_', '-', app()->getLocale()) }}" class="h-full">
    <head>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1">

        <title>{{ config('app.name', 'GameCloud') }} — Личный кабинет</title>

        @php
            $vtxIcon = (string) (config('app.branding.icon') ?? '');
        @endphp
        @if($vtxIcon !== '')
            <link rel="icon" href="{{ \Illuminate\Support\Facades\Storage::disk('public')->url($vtxIcon) }}">
        @endif

        <meta name="csrf-token" content="{{ csrf_token() }}">

        <link rel="preconnect" href="https://fonts.googleapis.com">
        <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
        <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">

        @vite(['resources/css/app.css', 'resources/js/app.js'])
        @livewireStyles
        <style>[x-cloak] { display: none !important; }</style>
    </head>
    <body class="panel-dark font-sans min-h-screen bg-[#17212b] text-slate-100 subpixel-antialiased" x-data="{ mobileSidebarOpen: false, userMenuOpen: false }">
        <div class="min-h-screen flex gap-3 px-3 py-3">
            <div
                class="md:hidden fixed inset-0 z-40"
                x-show="mobileSidebarOpen"
                x-transition.opacity
                x-cloak
            >
                <div class="absolute inset-0 bg-black/50" @click="mobileSidebarOpen = false"></div>
                <div class="relative h-full max-w-xs w-full bg-[#242f3d] rounded-r-3xl shadow-xl border border-white/10 p-4">
                    <div class="mb-4 flex items-center justify-between gap-2">
                        <div class="flex items-center gap-2">
                            @php
                                $vtxLogo = (string) (config('app.branding.logo') ?? '');
                            @endphp
                            @if($vtxLogo !== '')
                                <img src="{{ \Illuminate\Support\Facades\Storage::disk('public')->url($vtxLogo) }}" alt="Logo" class="h-8 w-8 rounded-xl object-cover">
                            @else
                                <span class="inline-flex h-8 w-8 items-center justify-center rounded-xl bg-gradient-to-tr from-sky-500 to-indigo-500 shadow-sm shadow-sky-500/30">
                                    <span class="h-3 w-3 rounded-md bg-white/90"></span>
                                </span>
                            @endif
                            <div class="flex flex-col leading-tight">
                                <span class="text-[13px] font-semibold text-slate-100">{{ config('app.name', 'Vortanix GameCloud') }}</span>
                                <span class="text-[11px] text-slate-300/80">Личный кабинет</span>
                            </div>
                        </div>
                        <button
                            type="button"
                            class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-100"
                            @click="mobileSidebarOpen = false"
                        >
                            ✕
                        </button>
                    </div>

                    <nav class="space-y-1 text-[13px] text-slate-200">
                        <a
                            href="{{ route('dashboard') }}"
                            class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('dashboard') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('dashboard') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                            <span>Обзор</span>
                        </a>
                        <a
                            href="{{ route('account') }}"
                            class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('account') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('account') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                            <span>Профиль</span>
                        </a>
                        <a
                            href="{{ route('my-servers') }}"
                            class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('my-servers') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('my-servers') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                            <span>Мои серверы</span>
                        </a>
                        <a
                            href="{{ route('rent-server') }}"
                            class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('rent-server') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('rent-server') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                            <span>Аренда сервера</span>
                        </a>
                        <a
                            href="{{ route('billing') }}"
                            class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('billing') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('billing') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                            <span>Биллинг</span>
                        </a>
                        <a
                            href="{{ route('monitoring') }}"
                            class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('monitoring') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('monitoring') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                            <span>Мониторинг</span>
                        </a>
                        <a
                            href="{{ route('support.index') }}"
                            class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('support.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('support.*') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                            <span>Тех. поддержка</span>
                        </a>
                    </nav>
                </div>
            </div>
            <aside class="hidden md:flex md:w-64 flex-col rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm">
                <div class="flex items-center gap-2 px-4 py-4 border-b border-white/10">
                    @php
                        $vtxLogo = (string) (config('app.branding.logo') ?? '');
                    @endphp
                    @if($vtxLogo !== '')
                        <img src="{{ \Illuminate\Support\Facades\Storage::disk('public')->url($vtxLogo) }}" alt="Logo" class="h-8 w-8 rounded-xl object-cover">
                    @else
                        <span class="inline-flex h-8 w-8 items-center justify-center rounded-xl bg-gradient-to-tr from-sky-500 to-indigo-500 shadow-sm shadow-sky-500/30">
                            <span class="h-3 w-3 rounded-md bg-white/90"></span>
                        </span>
                    @endif
                    <div class="flex flex-col leading-tight">
                        <span class="text-[13px] font-semibold text-slate-100">{{ config('app.name', 'Vortanix GameCloud') }}</span>
                        <span class="text-[11px] text-slate-300/80">Личный кабинет</span>
                    </div>
                </div>

                <nav class="flex-1 px-3 py-4 text-[13px] text-slate-200 space-y-1">
                    <a
                        href="{{ route('dashboard') }}"
                        class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('dashboard') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                    >
                        <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('dashboard') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                        <span>Главная</span>
                    </a>
                    <a
                        href="{{ route('account') }}"
                        class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('account') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                    >
                        <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('account') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                        <span>Профиль</span>
                    </a>
                    <a
                        href="{{ route('my-servers') }}"
                        class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('my-servers') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                    >
                        <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('my-servers') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                        <span>Мои серверы</span>
                    </a>
                    <a
                        href="{{ route('rent-server') }}"
                        class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('rent-server') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                    >
                        <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('rent-server') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                        <span>Аренда сервера</span>
                    </a>
                    <a
                        href="{{ route('billing') }}"
                        class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('billing') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                    >
                        <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('billing') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                        <span>Биллинг</span>
                    </a>
                    <a
                        href="{{ route('monitoring') }}"
                        class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('monitoring') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                    >
                        <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('monitoring') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                        <span>Мониторинг</span>
                    </a>
                    <a
                        href="{{ route('support.index') }}"
                        class="group relative flex items-center rounded-xl px-3 py-2 transition-all duration-200 {{ request()->routeIs('support.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white' }}"
                    >
                        <span class="absolute left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-full {{ request()->routeIs('support.*') ? 'bg-gradient-to-b from-indigo-400 to-sky-400 opacity-100' : 'bg-white/30 opacity-0 group-hover:opacity-60' }}"></span>
                        <span>Тех. поддержка</span>
                    </a>
                </nav>

                <div class="border-t border-white/10 px-4 py-3 text-[11px] text-slate-300/80">
                    @auth
                        <div class="truncate">{{ auth()->user()->email }}</div>
                    @endauth
                </div>
            </aside>

            <div class="flex-1 flex flex-col min-w-0">
                <header class="mb-3 px-4 sm:px-6 lg:px-8">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm">
                        <div class="flex items-center justify-between gap-3 px-4 py-2">
                            <div class="flex items-center gap-3 text-[13px] text-slate-300">
                                <button
                                    type="button"
                                    class="md:hidden inline-flex h-8 w-8 items-center justify-center rounded-xl border border-white/10 bg-black/10 text-slate-100 shadow-sm transition-all duration-200 hover:bg-black/20 active:scale-95"
                                    @click="mobileSidebarOpen = true"
                                >
                                    ☰
                                </button>
                                @hasSection('breadcrumb')
                                    <div class="hidden sm:flex items-center gap-2 text-xs text-slate-200">
                                        @yield('breadcrumb')
                                    </div>
                                @else
                                    <div class="hidden sm:flex items-center gap-2">
                                        <span class="text-xs text-slate-200">{{ __('messages.cabinet') }}</span>
                                        <span class="h-1 w-1 rounded-full bg-white/25"></span>
                                        <span class="text-xs text-slate-200">
                                            @yield('page_title', __('messages.overview'))
                                        </span>
                                    </div>
                                @endif
                            </div>

                            <div class="flex items-center gap-3 text-[12px]">
                                @auth
                                <a
                                    href="{{ route('billing') }}"
                                    class="inline-flex items-center gap-3 rounded-xl bg-black/10 px-3 py-1.5 ring-1 ring-white/10 text-slate-100 transition hover:bg-black/15"
                                    title="Баланс"
                                >
                                    <span class="text-[11px] text-slate-300/80">{{ __('messages.balance') }}</span>
                                    <span class="text-xs font-semibold">{{ number_format((float) (auth()->user()->balance ?? 0), 2) }} ₽</span>
                                </a>

                                @php
                                    $enabledLocales = ['en', 'ru'];
                                    try {
                                        if (\Illuminate\Support\Facades\Schema::hasTable('settings')) {
                                            $csv = (string) (\App\Models\Setting::getValue('app.locale.enabled', '') ?? '');
                                            $list = array_values(array_filter(array_map('trim', explode(',', $csv)), fn ($v) => $v !== ''));
                                            if (count($list) > 0) {
                                                $enabledLocales = $list;
                                            }
                                        }
                                    } catch (\Throwable $e) {
                                        //
                                    }
                                @endphp

                                <div class="relative" x-data="{ langOpen: false }" @click.outside="langOpen = false">
                                    <button
                                        type="button"
                                        class="inline-flex items-center gap-2 rounded-xl px-2.5 py-1.5 text-xs text-slate-200 transition-all duration-200 hover:bg-black/10 hover:text-white"
                                        :class="{ 'bg-black/10 text-white ring-1 ring-white/10': langOpen }"
                                        @click="langOpen = !langOpen; userMenuOpen = false"
                                        title="Язык"
                                    >
                                        <span class="font-semibold">{{ strtoupper((string) app()->getLocale()) }}</span>
                                        <svg class="h-3 w-3 transition-transform" :class="{ 'rotate-180': langOpen }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                                        </svg>
                                    </button>

                                    <div
                                        x-show="langOpen"
                                        x-transition:enter="transition ease-out duration-150"
                                        x-transition:enter-start="opacity-0 scale-95 translate-y-1"
                                        x-transition:enter-end="opacity-100 scale-100 translate-y-0"
                                        x-transition:leave="transition ease-in duration-100"
                                        x-transition:leave-start="opacity-100 scale-100 translate-y-0"
                                        x-transition:leave-end="opacity-0 scale-95 translate-y-1"
                                        x-cloak
                                        class="absolute right-0 mt-2 w-28 origin-top-right rounded-xl border border-white/10 bg-[#242f3d] shadow-xl shadow-black/40 z-10"
                                    >
                                        <div class="py-1">
                                            @foreach($enabledLocales as $loc)
                                                <form method="POST" action="{{ route('locale.switch', $loc) }}">
                                                    @csrf
                                                    <button
                                                        type="submit"
                                                        class="block w-full text-left px-4 py-2 text-xs text-slate-200 transition-colors hover:bg-black/10 hover:text-white"
                                                        @click="langOpen = false"
                                                    >
                                                        {{ strtoupper((string) $loc) }}
                                                    </button>
                                                </form>
                                            @endforeach
                                        </div>
                                    </div>
                                </div>

                                <div class="relative" @click.outside="userMenuOpen = false">
                                    <button
                                        type="button"
                                        class="inline-flex items-center gap-2 rounded-xl px-2.5 py-1.5 text-xs text-slate-200 transition-all duration-200 hover:bg-black/10 hover:text-white"
                                        :class="{ 'bg-black/10 text-white ring-1 ring-white/10': userMenuOpen }"
                                        @click="userMenuOpen = !userMenuOpen"
                                    >
                                        <span class="inline-flex h-6 w-6 items-center justify-center rounded-full bg-black/15 text-[10px] font-medium text-slate-100 leading-none pt-px ring-1 ring-white/10">
                                            {{ mb_strtoupper(mb_substr(auth()->user()->name ?? auth()->user()->email ?? 'A', 0, 1)) }}
                                        </span>
                                        <span class="hidden sm:inline">{{ auth()->user()->name ?? 'Аккаунт' }}</span>
                                        <svg class="h-3 w-3 transition-transform" :class="{ 'rotate-180': userMenuOpen }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                                        </svg>
                                    </button>
                                    <div
                                        x-show="userMenuOpen"
                                        x-transition:enter="transition ease-out duration-150"
                                        x-transition:enter-start="opacity-0 scale-95 translate-y-1"
                                        x-transition:enter-end="opacity-100 scale-100 translate-y-0"
                                        x-transition:leave="transition ease-in duration-100"
                                        x-transition:leave-start="opacity-100 scale-100 translate-y-0"
                                        x-transition:leave-end="opacity-0 scale-95 translate-y-1"
                                        x-cloak
                                        class="absolute right-0 mt-2 w-48 origin-top-right rounded-xl border border-white/10 bg-[#242f3d] shadow-xl shadow-black/40 z-10"
                                    >
                                        <div class="py-1">
                                            <a
                                                href="{{ route('account') }}"
                                                class="block px-4 py-2 text-xs text-slate-200 transition-colors hover:bg-black/10 hover:text-white"
                                                @click="userMenuOpen = false"
                                            >
                                                {{ __('messages.profile') }}
                                            </a>
                                            <form method="POST" action="{{ route('logout') }}">
                                                @csrf
                                                <button
                                                    type="submit"
                                                    class="block w-full text-left px-4 py-2 text-xs text-slate-200 transition-colors hover:bg-black/10 hover:text-white"
                                                >
                                                    {{ __('messages.logout') }}
                                                </button>
                                            </form>
                                        </div>
                                    </div>
                                </div>
                                @endauth
                            </div>
                        </div>
                    </div>
                </header>

                <main class="flex-1 min-w-0">
                    @yield('content')
                </main>

                <footer class="mt-8 pb-6 px-4 sm:px-6 lg:px-8">
                    <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                        <div class="text-[11px] text-slate-300/70">
                            © {{ now()->format('Y') }} {{ config('app.name', 'Vortanix GameCloud') }} — панель управления игровыми серверами.
                        </div>
                        <div class="flex items-center gap-3">
                            <a
                                href="{{ config('app.links.telegram', '#') }}"
                                target="_blank"
                                rel="noopener noreferrer"
                                class="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-white/10 bg-black/10 text-slate-200 transition hover:bg-black/15 hover:text-white"
                                aria-label="Telegram"
                                title="Telegram"
                            >
                                <svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                                    <path d="M21.6 4.3c.3-1.3-1-2.4-2.2-1.9L3.5 9c-1.6.7-1.4 3 .3 3.4l4.1 1 1.6 4.9c.5 1.5 2.4 1.9 3.5.8l2.4-2.2 4.3 3.2c1.2.9 2.9.2 3.2-1.3L21.6 4.3zM9.7 13.1l8.6-5.1c.2-.1.4.2.2.3l-7.1 6.3-.3 3.6-1.6-4.2c-.1-.4 0-.7.2-.9z"/>
                                </svg>
                            </a>
                            <a
                                href="{{ config('app.links.discord', '#') }}"
                                target="_blank"
                                rel="noopener noreferrer"
                                class="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-white/10 bg-black/10 text-slate-200 transition hover:bg-black/15 hover:text-white"
                                aria-label="Discord"
                                title="Discord"
                            >
                                <svg class="h-4 w-4" viewBox="0 0 245 240" fill="currentColor" aria-hidden="true">
                                    <path d="M104.4 104.9c-5.7 0-10.2 5-10.2 11.1 0 6.1 4.6 11.1 10.2 11.1 5.7 0 10.3-5 10.2-11.1 0-6.1-4.5-11.1-10.2-11.1zm36.2 0c-5.7 0-10.2 5-10.2 11.1 0 6.1 4.6 11.1 10.2 11.1 5.7 0 10.3-5 10.2-11.1 0-6.1-4.5-11.1-10.2-11.1z"/>
                                    <path d="M189.5 20h-134C24.9 20 0 44.9 0 75.5v89c0 30.6 24.9 55.5 55.5 55.5h113.2l-5.3-18.5 12.8 11.9 12.1 11.2 21.5 19.4V75.5C245 44.9 220.1 20 189.5 20zm-39.8 135.9s-3.7-4.4-6.8-8.3c13.5-3.8 18.6-12.3 18.6-12.3-4.2 2.8-8.2 4.8-11.8 6.1-5.1 2.1-10 3.5-14.8 4.3-9.8 1.8-18.8 1.3-26.5-.1-5.8-1.1-10.8-2.6-15-4.3-2.4-.9-5-2-7.6-3.4-.3-.2-.6-.3-.9-.5-.2-.1-.3-.2-.5-.3-2.2-1.2-3.4-2-3.4-2s4.9 8.3 17.9 12.2c-3.1 3.9-6.9 8.5-6.9 8.5-22.8-.7-31.5-15.7-31.5-15.7 0-33.2 14.8-60.1 14.8-60.1 14.8-11.1 28.9-10.8 28.9-10.8l1 1.2c-18.5 5.3-27.1 13.3-27.1 13.3s2.3-1.3 6.2-3.1c11.3-5 20.3-6.4 24-6.7.6-.1 1.1-.2 1.7-.2 6.1-.8 13-1 20.3-.2 9.6 1.1 19.9 3.9 30.5 9.6 0 0-8.2-7.8-25.9-13.1l1.4-1.6s14.1-.3 28.9 10.8c0 0 14.8 26.9 14.8 60.1 0 .1-8.8 15-31.6 15.7z"/>
                                </svg>
                            </a>
                            <a
                                href="{{ config('app.links.docs', '#') }}"
                                target="_blank"
                                rel="noopener noreferrer"
                                class="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-white/10 bg-black/10 text-slate-200 transition hover:bg-black/15 hover:text-white"
                                aria-label="Документация"
                                title="Документация"
                            >
                                <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                                    <path d="M4 19.5A2.5 2.5 0 016.5 17H20" />
                                    <path d="M6.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15A2.5 2.5 0 016.5 2z" />
                                </svg>
                            </a>
                            <a
                                href="{{ config('app.links.support', '#') }}"
                                target="_blank"
                                rel="noopener noreferrer"
                                class="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-white/10 bg-black/10 text-slate-200 transition hover:bg-black/15 hover:text-white"
                                aria-label="Support"
                                title="Support"
                            >
                                <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                                    <path d="M18 8a6 6 0 00-12 0" />
                                    <path d="M6 8v8a2 2 0 002 2h2" />
                                    <path d="M18 8v8a2 2 0 01-2 2h-2" />
                                    <path d="M12 18a2 2 0 002-2v-1" />
                                </svg>
                            </a>
                        </div>
                    </div>
                </footer>
            </div>
        </div>

        @stack('scripts')
        @livewireScripts
    </body>
</html>
