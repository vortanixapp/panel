<!DOCTYPE html>
<html lang="{{ str_replace('_', '-', app()->getLocale()) }}" class="h-full">
    <head>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1">

        <title>{{ config('app.name', 'GameCloud') }} — Админ‑панель</title>

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
        @stack('styles')
        @push('styles')
<style>
    [x-cloak] { display: none !important; }
</style>
@endpush
    </head>
    <body class="panel-dark font-sans min-h-screen bg-[#17212b] text-slate-100 subpixel-antialiased" x-data="{ mobileSidebarOpen: false, userMenuOpen: false, notificationsOpen: false, sidebarCollapsed: false, notificationsUnread: 0 }" x-cloak>
        <div class="min-h-screen flex gap-3 px-3 py-3" @click.outside="userMenuOpen = false; notificationsOpen = false">
            <div
                class="md:hidden fixed inset-0 z-40"
                x-show="mobileSidebarOpen"
                x-transition.opacity
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
                                <span class="inline-flex h-8 w-8 items-center justify-center rounded-xl bg-gradient-to-tr from-sky-500 to-indigo-500 shadow-sm shadow-black/20">
                                    <span class="h-3 w-3 rounded-md bg-white/80"></span>
                                </span>
                            @endif
                            <div class="flex flex-col leading-tight">
                                <span class="text-[13px] font-semibold text-slate-100">{{ config('app.name', 'Vortanix GameCloud') }}</span>
                                <span class="text-[11px] text-slate-300/80">Админ‑панель</span>
                            </div>
                        </div>
                        <button
                            type="button"
                            class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-100"
                            @click="sidebarCollapsed = !sidebarCollapsed"
                        >
                            <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path>
                            </svg>
                        </button>
                        <button
                            type="button"
                            class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-100"
                            @click="mobileSidebarOpen = false"
                        >
                            ✕
                        </button>
                    </div>

                    <nav class="space-y-1 text-[13px] text-slate-200">
                        <div class="px-2.5 pt-1 text-[11px] uppercase tracking-wide text-slate-300/70">Общее</div>
                        <a
                            href="{{ route('admin.dashboard') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ request()->routeIs('admin.dashboard') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <!-- Домашняя страница -->
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M3 9.5L10 3l7 6.5V17a1 1 0 01-1 1h-4v-4H8v4H4a1 1 0 01-1-1V9.5z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            </svg>
                            <span>Главная</span>
                        </a>

                        <div class="my-2 border-t border-white/10"></div>
                        <div class="px-2.5 pt-1 text-[11px] uppercase tracking-wide text-slate-300/70">Управление</div>
                        <a
                            href="{{ route('admin.users') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ request()->routeIs('admin.users.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <!-- Пользователи -->
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M7 9a3 3 0 100-6 3 3 0 000 6zM15 9a3 3 0 100-6 3 3 0 000 6z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M3 17a4 4 0 018 0v1H3v-1zM11 17a4 4 0 018 0v1h-8v-1z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            </svg>
                            <span>Пользователи</span>
                        </a>
                        <a
                            href="{{ route('admin.servers.index') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ request()->routeIs('admin.servers.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <!-- Серверы -->
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M4 4h12v4H4V4zM4 12h12v4H4v-4z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M7 6h.01M7 14h.01" stroke-width="1.6" stroke-linecap="round" />
                            </svg>
                            <span>Серверы</span>
                        </a>
                        <a
                            href="{{ route('admin.support.index') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ request()->routeIs('admin.support.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M10 18a7 7 0 100-14 7 7 0 000 14z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M7.5 8.2a2.5 2.5 0 115 0c0 1.7-2.5 1.9-2.5 3.3" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M10 14h.01" stroke-width="2" stroke-linecap="round" />
                            </svg>
                            <span>Тех. поддержка</span>
                        </a>

                        <a
                            href="{{ route('admin.bug-report') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ request()->routeIs('admin.bug-report*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M7 7h6v6H7V7z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M4 11h3M13 11h3M5 5l2 2M15 5l-2 2M5 15l2-2M15 15l-2-2" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            </svg>
                            <span>Баг-репорт</span>
                        </a>

                        <div class="my-2 border-t border-white/10"></div>
                        <div class="px-2.5 pt-1 text-[11px] uppercase tracking-wide text-slate-300/70">Инфраструктура</div>
                        <a
                            href="{{ route('admin.locations.index') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ (request()->routeIs('admin.locations.*') && !request()->routeIs('admin.locations.daemon.*')) ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <!-- Локации -->
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M10 3a4 4 0 00-4 4c0 3 4 7 4 7s4-4 4-7a4 4 0 00-4-4z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <circle cx="10" cy="7" r="1.5" stroke-width="1.4" />
                            </svg>
                            <span>Локации</span>
                        </a>
                                                <a
                            href="{{ route('admin.games') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ request()->routeIs('admin.games.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <!-- Игры -->
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M11 4a2 2 0 114 0 2 2 0 01-4 0zM15 8a2 2 0 01-4 0 2 2 0 014 0zM7 8a2 2 0 11-4 0 2 2 0 014 0zM3 4a2 2 0 114 0 2 2 0 01-4 0z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M10 12v4" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M14 12H6" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            </svg>
                            <span>Игры</span>
                        </a>
                        <a
                            href="{{ route('admin.tariffs.index') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ request()->routeIs('admin.tariffs.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <!-- Тарифы -->
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            </svg>
                            <span>Тарифы</span>
                        </a>
                        <a
                            href="{{ route('admin.vortanix-daemons.index') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ (request()->routeIs('admin.vortanix-daemons.*') || request()->routeIs('admin.locations.daemon.*')) ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <!-- Vortanix Daemons (шестерёнка) -->
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <circle cx="10" cy="10" r="2.5" stroke-width="1.4" />
                                <path d="M10 3v2M10 15v2M4.22 4.22l1.42 1.42M14.36 14.36l1.42 1.42M3 10h2M15 10h2M4.22 15.78l1.42-1.42M14.36 5.64l1.42-1.42" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            </svg>
                            <span>Vortanix Daemons</span>
                        </a>

                        <div class="my-2 border-t border-white/10"></div>
                        <div class="px-2.5 pt-1 text-[11px] uppercase tracking-wide text-slate-300/70">Каталог</div>
                        <a
                            href="{{ route('admin.plugins.index') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ request()->routeIs('admin.plugins.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <!-- Плагины -->
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M7 3h6v4H7V3zM3 9h4v4H3V9zm10 0h4v4h-4V9zM7 13h6v4H7v-4z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            </svg>
                            <span>Плагины</span>
                        </a>

                        <a
                            href="{{ route('admin.maps.index') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ request()->routeIs('admin.maps.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <!-- Карты -->
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M3 4l5-2 4 2 5-2v14l-5 2-4-2-5 2V4z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M8 2v14" stroke-width="1.4" stroke-linecap="round" />
                                <path d="M12 4v14" stroke-width="1.4" stroke-linecap="round" />
                            </svg>
                            <span>Карты</span>
                        </a>

                        <div class="my-2 border-t border-white/10"></div>
                        <div class="px-2.5 pt-1 text-[11px] uppercase tracking-wide text-slate-300/70">Контент и маркетинг</div>

                        <a
                            href="{{ route('admin.news.index') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ request()->routeIs('admin.news.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <!-- Новости -->
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M5 4h10v12H5V4z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M7 7h6M7 10h6M7 13h4" stroke-width="1.4" stroke-linecap="round" />
                            </svg>
                            <span>Новости</span>
                        </a>
                        <a
                            href="#"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 hover:bg-black/10 hover:text-white"
                            @click="mobileSidebarOpen = false"
                        >
                            <!-- Биллинг / оплаты -->
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M3 6h14v8H3V6z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M3 9h14" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M6 12h2" stroke-width="1.4" stroke-linecap="round" />
                            </svg>
                            <span>Биллинг</span>
                        </a>

                        <a
                            href="{{ route('admin.settings.edit') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ request()->routeIs('admin.settings.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M10 2l1 2 2 .5-1 2 1 2-2 .5-1 2-1-2-2-.5 1-2-1-2 2-.5 1-2z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <circle cx="10" cy="10" r="1.5" stroke-width="1.4" />
                            </svg>
                            <span>Настройки</span>
                        </a>

                        <a
                            href="{{ route('admin.license') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ request()->routeIs('admin.license*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M6 9V7a4 4 0 118 0v2" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M5 9h10v8H5V9z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M10 13v2" stroke-width="1.4" stroke-linecap="round" />
                            </svg>
                            <span>Лицензия</span>
                        </a>

                        <a
                            href="{{ route('admin.updates') }}"
                            class="flex items-center gap-2 rounded-md px-2.5 py-2 {{ request()->routeIs('admin.updates') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                            @click="mobileSidebarOpen = false"
                        >
                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                                <path d="M10 3v9" stroke-width="1.4" stroke-linecap="round" />
                                <path d="M6.5 9.5L10 13l3.5-3.5" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                <path d="M4 17h12" stroke-width="1.4" stroke-linecap="round" />
                            </svg>
                            <span>Обновления</span>
                        </a>
                    </nav>
                </div>
            </div>
            <aside class="hidden md:flex flex-col rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm transition-all duration-300" :class="{ 'md:w-20': sidebarCollapsed, 'md:w-64': !sidebarCollapsed }">
                <div class="flex items-center gap-2 px-4 py-4 border-b border-white/10" :class="{ 'justify-center': sidebarCollapsed }">
                    @php
                        $vtxLogo = (string) (config('app.branding.logo') ?? '');
                    @endphp
                    @if($vtxLogo !== '')
                        <img src="{{ \Illuminate\Support\Facades\Storage::disk('public')->url($vtxLogo) }}" alt="Logo" class="h-8 w-8 rounded-xl object-cover">
                    @else
                        <span class="inline-flex h-8 w-8 items-center justify-center rounded-xl bg-gradient-to-tr from-sky-500 to-indigo-500 shadow-sm shadow-black/20">
                            <span class="h-3 w-3 rounded-md bg-white/80"></span>
                        </span>
                    @endif
                    <div class="flex flex-col leading-tight overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">
                        <span class="text-[13px] font-semibold text-slate-100">{{ config('app.name', 'Vortanix GameCloud') }}</span>
                        <span class="text-[11px] text-slate-300/80">Админ‑панель</span>
                    </div>
                </div>

                <nav class="flex-1 min-h-0 px-2 py-4 text-[13px] text-slate-200 space-y-1 overflow-y-auto">
                    <div class="px-2 pt-1 text-[11px] uppercase tracking-wide text-slate-300/70">Общее</div>
                    <a
                        href="{{ route('admin.dashboard') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.dashboard') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <!-- Домашняя страница -->
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M3 9.5L10 3l7 6.5V17a1 1 0 01-1 1h-4v-4H8v4H4a1 1 0 01-1-1V9.5z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Главная</span>
                    </a>

                    <div class="my-2 border-t border-white/10"></div>
                    <div class="px-2 pt-1 text-[11px] uppercase tracking-wide text-slate-300/70">Управление</div>
                    <a
                        href="{{ route('admin.users') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.users.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <!-- Пользователи -->
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M7 9a3 3 0 100-6 3 3 0 000 6zM15 9a3 3 0 100-6 3 3 0 000 6z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M3 17a4 4 0 018 0v1H3v-1zM11 17a4 4 0 018 0v1h-8v-1z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Пользователи</span>
                    </a>
                    <a
                        href="{{ route('admin.servers.index') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.servers.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <!-- Серверы -->
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M4 4h12v4H4V4zM4 12h12v4H4v-4z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M7 6h.01M7 14h.01" stroke-width="1.6" stroke-linecap="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Серверы</span>
                    </a>

                    <a
                        href="{{ route('admin.support.index') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.support.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M10 18a7 7 0 100-14 7 7 0 000 14z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M7.5 8.2a2.5 2.5 0 115 0c0 1.7-2.5 1.9-2.5 3.3" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M10 14h.01" stroke-width="2" stroke-linecap="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Тех. поддержка</span>
                    </a>

                    <a
                        href="{{ route('admin.bug-report') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.bug-report*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M7 7h6v6H7V7z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M4 11h3M13 11h3M5 5l2 2M15 5l-2 2M5 15l2-2M15 15l-2-2" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Баг-репорт</span>
                    </a>

                    <div class="my-2 border-t border-white/10"></div>
                    <div class="px-2 pt-1 text-[11px] uppercase tracking-wide text-slate-300/70">Инфраструктура</div>
                    <a
                        href="{{ route('admin.locations.index') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ (request()->routeIs('admin.locations.*') && !request()->routeIs('admin.locations.daemon.*')) ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <!-- Локации -->
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M10 3a4 4 0 00-4 4c0 3 4 7 4 7s4-4 4-7a4 4 0 00-4-4z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <circle cx="10" cy="7" r="1.5" stroke-width="1.4" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Локации</span>
                    </a>
                    <a
                        href="{{ route('admin.games') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.games.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <!-- Игры -->
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M11 4a2 2 0 114 0 2 2 0 01-4 0zM15 8a2 2 0 01-4 0 2 2 0 014 0zM7 8a2 2 0 11-4 0 2 2 0 014 0zM3 4a2 2 0 114 0 2 2 0 01-4 0z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M10 12v4" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M14 12H6" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Игры</span>
                    </a>
                    <a
                        href="{{ route('admin.tariffs.index') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.tariffs.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <!-- Тарифы -->
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Тарифы</span>
                    </a>
                    <a
                        href="{{ route('admin.vortanix-daemons.index') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ (request()->routeIs('admin.vortanix-daemons.*') || request()->routeIs('admin.locations.daemon.*')) ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <!-- Vortanix Daemons (шестерёнка) -->
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <circle cx="10" cy="10" r="2.5" stroke-width="1.4" />
                            <path d="M10 3v2M10 15v2M4.22 4.22l1.42 1.42M14.36 14.36l1.42 1.42M3 10h2M15 10h2M4.22 15.78l1.42-1.42M14.36 5.64l1.42-1.42" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Vortanix Daemons</span>
                    </a>

                    <div class="my-2 border-t border-white/10"></div>
                    <div class="px-2 pt-1 text-[11px] uppercase tracking-wide text-slate-300/70">Каталог</div>
                    <a
                        href="{{ route('admin.plugins.index') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.plugins.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <!-- Плагины -->
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M7 3h6v4H7V3zM3 9h4v4H3V9zm10 0h4v4h-4V9zM7 13h6v4H7v-4z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Плагины</span>
                    </a>

                    <a
                        href="{{ route('admin.maps.index') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.maps.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <!-- Карты -->
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M3 4l5-2 4 2 5-2v14l-5 2-4-2-5 2V4z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M8 2v14" stroke-width="1.4" stroke-linecap="round" />
                            <path d="M12 4v14" stroke-width="1.4" stroke-linecap="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Карты</span>
                    </a>

                    <div class="my-2 border-t border-white/10"></div>
                    <div class="px-2 pt-1 text-[11px] uppercase tracking-wide text-slate-300/70">Контент и маркетинг</div>

                    <a
                        href="{{ route('admin.news.index') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.news.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <!-- Новости -->
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M5 4h10v12H5V4z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M7 7h6M7 10h6M7 13h4" stroke-width="1.4" stroke-linecap="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Новости</span>
                    </a>

                    <a
                        href="{{ route('admin.promotions.index') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.promotions.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M10 2l2.5 5 5.5.8-4 3.9.9 5.5-4.9-2.6-4.9 2.6.9-5.5-4-3.9L7.5 7 10 2z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Акции</span>
                    </a>

                    <a
                        href="{{ route('admin.logs.index') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.logs.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M4 4h12v12H4V4z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M6 7h8M6 10h8M6 13h6" stroke-width="1.4" stroke-linecap="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Логи</span>
                    </a>

                    <a
                        href="{{ route('admin.mailings.index') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.mailings.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M3 5h14v10H3V5z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M3 6l7 5 7-5" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Рассылка</span>
                    </a>

                    <a
                        href="{{ route('admin.language.edit') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.language.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M4 15h12" stroke-width="1.4" stroke-linecap="round" />
                            <path d="M5 5h6" stroke-width="1.4" stroke-linecap="round" />
                            <path d="M6 13l4-8 4 8" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M7.5 10h5" stroke-width="1.4" stroke-linecap="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Язык</span>
                    </a>

                    <a
                        href="#"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 hover:bg-black/10 hover:text-white"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <!-- Биллинг / оплаты -->
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M3 6h14v8H3V6z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M3 9h14" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M6 12h2" stroke-width="1.4" stroke-linecap="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Биллинг</span>
                    </a>

                    <a
                        href="{{ route('admin.settings.edit') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.settings.*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M10 2l1 2 2 .5-1 2 1 2-2 .5-1 2-1-2-2-.5 1-2-1-2 2-.5 1-2z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <circle cx="10" cy="10" r="1.5" stroke-width="1.4" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Настройки</span>
                    </a>

                    <a
                        href="{{ route('admin.license') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.license*') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M6 9V7a4 4 0 118 0v2" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M5 9h10v8H5V9z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M10 13v2" stroke-width="1.4" stroke-linecap="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Лицензия</span>
                    </a>

                    <a
                        href="{{ route('admin.updates') }}"
                        class="flex items-center gap-2 rounded-md px-2 py-2 transition-all duration-300 {{ request()->routeIs('admin.updates') ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'hover:bg-black/10 hover:text-white' }}"
                        :class="{ 'justify-center': sidebarCollapsed }"
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5 flex-shrink-0">
                            <path d="M10 3v9" stroke-width="1.4" stroke-linecap="round" />
                            <path d="M6.5 9.5L10 13l3.5-3.5" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                            <path d="M4 17h12" stroke-width="1.4" stroke-linecap="round" />
                        </svg>
                        <span class="overflow-hidden transition-all duration-300" :class="{ 'w-0 opacity-0': sidebarCollapsed, 'w-auto opacity-100': !sidebarCollapsed }">Обновления</span>
                    </a>
                </nav>

                <div class="border-t border-white/10 px-4 py-3 text-[11px] text-slate-300/80 transition-all duration-300" :class="{ 'text-center': sidebarCollapsed }">
                    @auth
                        <div class="truncate">{{ auth()->user()->email }}</div>
                    @endauth
                </div>
            </aside>

            <div class="flex-1 flex flex-col min-w-0">
                <header class="mb-3 rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm">
                    <div class="flex items-center justify-between gap-3 px-4 py-3">
                        <div class="flex items-center gap-3 text-[13px] text-slate-300">
                            <button
                                type="button"
                                class="hidden md:inline-flex h-7 w-7 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-100 hover:bg-black/15"
                                @click="sidebarCollapsed = !sidebarCollapsed"
                            >
                                <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path>
                                </svg>
                            </button>
                            <button
                                type="button"
                                class="md:hidden inline-flex h-7 w-7 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-100"
                                @click="mobileSidebarOpen = true"
                            >
                                ☰
                            </button>
                            <div class="hidden sm:flex items-center gap-2">
                                <span class="text-xs uppercase tracking-wide text-slate-300/70">Админка</span>
                                <span class="h-1 w-1 rounded-full bg-white/25"></span>
                                <span class="text-xs text-slate-200">
                                    @yield('page_title', 'Обзор админки')
                                </span>
                            </div>
                        </div>
                        <div class="flex items-center gap-3 text-[12px]">
                            @auth
                                <div class="relative">
                                    <button
                                        type="button"
                                        class="inline-flex items-center justify-center h-7 w-7 rounded-md border border-white/10 bg-black/10 text-slate-100 hover:bg-black/15"
                                        title="Уведомления"
                                        @click="notificationsOpen = !notificationsOpen; userMenuOpen = false"
                                    >
                                        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-5-5V7a3 3 0 00-6 0v5l-5 5h5m0 0v1a3 3 0 006 0v-1m-6 0h6"></path>
                                        </svg>
                                    </button>

                                    <span
                                        id="vtxNotifBadge"
                                        x-show="notificationsUnread > 0"
                                        x-text="notificationsUnread"
                                        class="absolute right-0 top-0 translate-x-1/2 -translate-y-1/2 min-w-[16px] h-4 px-1 rounded-full bg-rose-500 text-[10px] leading-4 text-white text-center font-semibold"
                                    ></span>
                                    <div
                                        x-show="notificationsOpen"
                                        x-transition
                                        class="absolute right-0 mt-1 w-80 rounded-md border border-white/10 bg-[#242f3d] shadow-xl shadow-black/30 z-10"
                                    >
                                        <div class="px-4 py-3 border-b border-white/10">
                                            <h3 class="text-sm font-semibold text-slate-100">Уведомления</h3>
                                        </div>
                                        <div class="max-h-64 overflow-y-auto" id="vtxNotifDropdownBody">
                                            <div class="px-4 py-3 text-center text-xs text-slate-300/80">Загрузка…</div>
                                        </div>
                                        <div class="px-4 py-2 border-t border-white/10">
                                            <a href="{{ route('admin.notifications') }}" class="text-xs text-slate-200 hover:text-white">Просмотреть все</a>
                                        </div>
                                    </div>
                                </div>
                                <div class="relative">
                                    <button
                                        type="button"
                                        class="inline-flex items-center gap-2 text-xs text-slate-200 hover:text-white cursor-pointer bg-transparent border-0 p-0"
                                        @click="userMenuOpen = !userMenuOpen"
                                    >
                                        <span class="inline-flex h-6 w-6 items-center justify-center rounded-full bg-black/15 text-[10px] font-medium text-slate-100 leading-none pt-px ring-1 ring-white/10">
                                            {{ mb_strtoupper(mb_substr(auth()->user()->name ?? auth()->user()->email ?? 'A', 0, 1)) }}
                                        </span>
                                        <span class="hidden sm:inline">{{ auth()->user()->name ?? 'Админ' }}</span>
                                        <span class="inline sm:hidden">Админ</span>
                                        <svg class="h-3 w-3 transition-transform" :class="{ 'rotate-180': userMenuOpen }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                                        </svg>
                                    </button>
                                    <div
                                        x-show="userMenuOpen"
                                        x-transition
                                        class="absolute right-0 mt-1 w-48 rounded-md border border-white/10 bg-[#242f3d] shadow-xl shadow-black/30 z-10"
                                    >
                                        <div class="py-1">
                                            <a
                                                href="#"
                                                class="block px-4 py-2 text-xs text-slate-200 hover:bg-black/10 hover:text-white"
                                                @click="userMenuOpen = false"
                                            >
                                                Профиль
                                            </a>
                                            <form method="POST" action="{{ route('logout') }}">
                                                @csrf
                                                <button
                                                    type="submit"
                                                    class="block w-full text-left px-4 py-2 text-xs text-slate-200 hover:bg-black/10 hover:text-white"
                                                >
                                                    Выйти
                                                </button>
                                            </form>
                                        </div>
                                    </div>
                                </div>
                            @endauth
                        </div>
                    </div>
                </header>

                <main class="flex-1 min-w-0">
                    @yield('content')
                </main>
            </div>
        </div>

        @push('scripts')
            <script>
                (function () {
                    const body = document.getElementById('vtxNotifDropdownBody');
                    const badgeEl = document.getElementById('vtxNotifBadge');
                    const csrf = document.querySelector('meta[name="csrf-token"]')?.getAttribute('content') || '';

                    function esc(s) {
                        return String(s || '').replace(/[&<>"']/g, (c) => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
                    }

                    function badge(level) {
                        const l = String(level || 'info');
                        if (l === 'critical') return '<span class="inline-flex items-center rounded-full border border-rose-500/30 bg-rose-500/10 px-2 py-0.5 text-[10px] font-semibold text-rose-200">critical</span>';
                        if (l === 'warning') return '<span class="inline-flex items-center rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-[10px] font-semibold text-amber-200">warning</span>';
                        return '<span class="inline-flex items-center rounded-full border border-white/10 bg-black/10 px-2 py-0.5 text-[10px] font-semibold text-slate-200">info</span>';
                    }

                    async function apiGet(url) {
                        const r = await fetch(url, { headers: { 'Accept': 'application/json' }, credentials: 'same-origin' });
                        return r.json();
                    }

                    async function apiPost(url) {
                        const r = await fetch(url, {
                            method: 'POST',
                            headers: { 'Accept': 'application/json', 'Content-Type': 'application/json', 'X-CSRF-TOKEN': csrf },
                            credentials: 'same-origin',
                            body: '{}',
                        });
                        return r.json().catch(() => ({}));
                    }

                    async function loadDropdown() {
                        if (!body) return;
                        let data;
                        try {
                            data = await apiGet(@json(route('admin.notifications.api.list')) + '?limit=10');
                        } catch (e) {
                            body.innerHTML = '<div class="px-4 py-3 text-center text-xs text-rose-200">Ошибка загрузки</div>';
                            return;
                        }

                        const items = (data && data.data && Array.isArray(data.data.items)) ? data.data.items : [];
                        const unread = (data && data.data && Number.isFinite(Number(data.data.unread_count))) ? Number(data.data.unread_count) : 0;

                        if (badgeEl) {
                            badgeEl.textContent = String(unread);
                            badgeEl.style.display = unread > 0 ? '' : 'none';
                        }

                        const unreadItems = items.filter((n) => !n.is_read);
                        if (unreadItems.length === 0) {
                            body.innerHTML = '<div class="px-4 py-3 text-center text-xs text-slate-300/80">Нет новых уведомлений.</div>';
                            return;
                        }

                        body.innerHTML = unreadItems.map((n) => {
                            const id = Number(n.id || 0);
                            return `
                                <div class="px-4 py-3 border-b border-white/10 hover:bg-black/10">
                                    <div class="flex items-start justify-between gap-2">
                                        <div class="text-xs font-semibold text-slate-100">${esc(n.title)}</div>
                                        ${badge(n.level)}
                                    </div>
                                    <div class="mt-1 text-[11px] text-slate-300/80 whitespace-pre-line">${esc(n.body)}</div>
                                    <div class="mt-2 flex items-center justify-between gap-2">
                                        <div class="text-[10px] text-slate-400">${esc(n.created_at || '')}</div>
                                        <button type="button" data-id="${id}" class="vtxNotifReadBtn text-[11px] text-sky-200 hover:text-white">Прочитать</button>
                                    </div>
                                </div>
                            `;
                        }).join('');

                        for (const btn of Array.from(body.querySelectorAll('.vtxNotifReadBtn'))) {
                            btn.addEventListener('click', async (e) => {
                                const id = e.currentTarget?.getAttribute('data-id');
                                if (!id) return;
                                try {
                                    await apiPost(@json(route('admin.notifications.api.read', ['id' => 0])).replace('/0/read', '/' + encodeURIComponent(id) + '/read'));
                                } catch (err) {}
                                await loadDropdown();
                            });
                        }
                    }

                    loadDropdown();
                    window.setInterval(loadDropdown, 30000);
                })();
            </script>
        @endpush

        @livewireScripts
        @stack('scripts')
    </body>
</html>
