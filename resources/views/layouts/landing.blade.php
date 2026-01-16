<!DOCTYPE html>
<html lang="{{ str_replace('_', '-', app()->getLocale()) }}" class="h-full">
    <head>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1">

        <title>{{ config('app.name', 'GameCloud') }} — Игровой хостинг</title>
        <meta name="description" content="Быстрые и надёжные игровые серверы для CS2, Minecraft, Rust и других игр.">

        @php
            $vtxIcon = (string) (config('app.branding.icon') ?? '');
        @endphp
        @if($vtxIcon !== '')
            <link rel="icon" href="{{ \Illuminate\Support\Facades\Storage::disk('public')->url($vtxIcon) }}">
        @endif

        <link rel="preconnect" href="https://fonts.googleapis.com">
        <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
        <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">

        @vite(['resources/css/app.css', 'resources/js/app.js'])
        @livewireStyles
    </head>
    <body class="min-h-screen overflow-x-clip bg-[#17212b] text-slate-100 antialiased">
        <div class="min-h-screen flex flex-col relative">
            @if (request()->routeIs('home'))
                @php
                    $heroBackgrounds = [
                        asset('img/minecraft.webp'),
                        asset('img/rust.webp'),
                        asset('img/bf4.webp'),
                    ];
                @endphp
                <div class="pointer-events-none absolute inset-x-0 top-0 h-[620px] sm:h-[720px] lg:h-[920px] overflow-hidden">
                    <div class="absolute inset-0" data-hero-carousel data-interval="6500">
                        @foreach ($heroBackgrounds as $i => $src)
                            <div
                                class="vtx-hero-bg absolute inset-0 bg-cover bg-center {{ $i === 0 ? 'is-active' : '' }}"
                                data-hero-slide
                                style="background-image: url('{{ $src }}');"
                            ></div>
                        @endforeach
                    </div>
                    <div class="absolute inset-0 bg-[radial-gradient(circle_at_top,_rgba(56,189,248,0.10),_transparent_55%),_radial-gradient(circle_at_bottom_right,_rgba(129,140,248,0.10),_transparent_55%)]"></div>
                    <div class="absolute inset-0 bg-gradient-to-b from-[#17212b]/35 via-[#17212b]/50 to-[#17212b]"></div>
                    <div class="absolute inset-x-0 -top-40 h-64 bg-[radial-gradient(circle_at_top,_rgba(56,189,248,0.18),_transparent_60%),_radial-gradient(circle_at_20%_20%,_rgba(129,140,248,0.16),_transparent_55%)]"></div>
                    <div class="absolute inset-x-0 bottom-0 h-48 bg-gradient-to-b from-transparent to-[#17212b]"></div>
                </div>
            @endif

            <header class="relative z-10 border-b border-white/10 bg-[#242f3d]/55 backdrop-blur">
                <div class="mx-auto max-w-6xl px-4 py-3 flex items-center justify-between gap-4">
                    <a href="{{ url('/') }}" class="flex items-center gap-2">
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
                            <span class="text-sm font-semibold tracking-tight text-slate-100">{{ config('app.name', 'Vortanix GameCloud') }}</span>
                            <span class="text-[11px] text-slate-300/80">Игровой хостинг</span>
                        </div>
                    </a>
                    <nav class="hidden md:flex items-center gap-6 text-xs font-medium text-slate-200">
                        <a href="{{ route('games') }}" class="hover:text-white transition">Игры</a>
                        <a href="{{ route('features') }}" class="hover:text-white transition">Возможности</a>
                        <a href="#pricing" class="hover:text-white transition">Тарифы</a>
                        <a href="#faq" class="hover:text-white transition">FAQ</a>
                    </nav>
                    <div class="flex items-center gap-3">
                        @auth
                            <a
                                href="{{ route('dashboard') }}"
                                class="hidden sm:inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-medium text-slate-200 hover:bg-black/15 hover:text-white transition"
                            >
                                Панель управления
                            </a>
                            <form method="POST" action="{{ route('logout') }}" class="hidden sm:block">
                                @csrf
                                <button
                                    type="submit"
                                    class="inline-flex items-center rounded-md bg-slate-900 px-3.5 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-slate-800 transition"
                                >
                                    Выйти
                                </button>
                            </form>
                        @else
                            <a href="{{ route('login') }}" class="text-xs font-medium text-slate-200 hover:text-white">
                                Войти
                            </a>
                            <a
                                href="{{ route('register') }}"
                                class="inline-flex items-center rounded-md bg-slate-900 px-3.5 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-slate-800 transition"
                            >
                                Регистрация
                            </a>
                        @endauth
                    </div>
                </div>
            </header>

            <main class="relative z-10 flex-1 flex flex-col">
                @yield('content')
            </main>

            <footer class="border-t border-white/10 bg-[#17212b]">
                <div class="mx-auto max-w-6xl px-4 py-6 flex flex-col md:flex-row items-center justify-between gap-4 text-[11px] text-slate-300/70">
                    <p>© {{ date('Y') }} Vortanix GameCloud. Игровой хостинг.</p>
                    <div class="flex flex-wrap items-center gap-4">
                        <span>Поддержка 24/7</span>
                        <span class="h-1 w-1 rounded-full bg-white/25"></span>
                        <span>Серверы в ЕС и РФ</span>
                        <span class="h-1 w-1 rounded-full bg-white/25"></span>
                        <span>Защита от DDoS</span>
                    </div>
                </div>
            </footer>
        </div>

        @livewireScripts
    </body>
</html>
