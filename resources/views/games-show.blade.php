@extends('layouts.landing')

@section('content')
    {{-- Hero конкретной игры --}}
    <section class="border-b border-white/10 bg-gradient-to-b from-[#17212b] to-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-12 md:py-16">
            <div class="mb-6 flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                <div>
                    <p class="text-[11px] font-medium uppercase tracking-wide text-sky-300">Игровой сервер</p>
                    <h1 class="mt-1 text-2xl md:text-3xl lg:text-4xl font-semibold tracking-tight text-slate-100">
                        {{ $game['title'] }}
                    </h1>
                    <p class="mt-2 text-sm text-slate-300/80 max-w-xl">
                        {{ $game['description'] }}
                    </p>
                    <dl class="mt-4 grid grid-cols-2 gap-4 text-[12px] text-slate-300/80 max-w-md">
                        <div>
                            <dt class="text-[11px] uppercase tracking-wide text-slate-300/70">Жанр</dt>
                            <dd class="mt-1 text-slate-100">{{ $game['genre'] }}</dd>
                        </div>
                        <div>
                            <dt class="text-[11px] uppercase tracking-wide text-slate-300/70">Платформы</dt>
                            <dd class="mt-1 text-slate-100">{{ $game['platforms'] }}</dd>
                        </div>
                    </dl>
                    <div class="mt-5 flex flex-wrap items-center gap-3 text-xs">
                        <a href="{{ route('dashboard') }}" class="inline-flex items-center rounded-md bg-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">
                            Развернуть сервер {{ $game['title'] }}
                        </a>
                        <a href="{{ route('games') }}" class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-medium text-slate-200 hover:bg-black/15 hover:text-white">
                            ← Ко всем играм
                        </a>
                    </div>
                </div>
                <div class="mt-6 md:mt-0 md:w-80 lg:w-96">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-4 shadow-md shadow-black/20">
                        <p class="text-[11px] uppercase tracking-wide text-slate-300/70 mb-1">Быстрый старт</p>
                        <p class="text-sm font-semibold text-slate-100 mb-3">Пример конфигурации для {{ $game['title'] }}</p>
                        <ul class="space-y-1.5 text-[12px] text-slate-300/80">
                            <li>2 vCPU • 4 ГБ RAM • NVMe 40 ГБ</li>
                            <li>DDoS‑защита и авто‑перезапуск сервера</li>
                            <li>Развёртывание ~ 40 секунд</li>
                        </ul>
                        <button class="mt-4 w-full rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white">
                            Настроить свой пресет
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </section>

    {{-- Секция: другие игры --}}
    <section class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14">
            <div class="mb-4 flex items-center justify-between gap-2">
                <h2 class="text-sm font-semibold tracking-tight text-slate-100">Другие игры</h2>
                <a href="{{ route('games') }}" class="text-[11px] text-sky-300 hover:text-sky-200">Смотреть все</a>
            </div>
            <div class="grid gap-4 md:grid-cols-3 text-[12px] text-slate-300/80">
                @foreach($games as $other)
                    @if($other['slug'] === $game['slug'])
                        @continue
                    @endif
                    <a href="{{ route('games.show', $other['slug']) }}" class="group rounded-2xl border border-white/10 bg-[#242f3d] p-3 hover:border-sky-500/50 hover:bg-black/10 transition flex flex-col gap-1.5">
                        <div class="flex items-center justify-between gap-2">
                            <div>
                                <p class="text-xs font-semibold text-slate-100">{{ $other['title'] }}</p>
                                <p class="text-[11px] text-slate-300/70">{{ $other['genre'] }}</p>
                            </div>
                            <span class="inline-flex items-center rounded-full bg-sky-500/10 px-2 py-0.5 text-[10px] font-medium text-sky-300 ring-1 ring-sky-500/20">
                                {{ $other['platforms'] }}
                            </span>
                        </div>
                        <p class="text-[11px] text-slate-300/80">{{ $other['short'] }}</p>
                        <span class="mt-1 inline-flex items-center text-[11px] font-medium text-sky-300 group-hover:text-sky-200">
                            Открыть страницу игры →
                        </span>
                    </a>
                @endforeach
            </div>
        </div>
    </section>

    {{-- Можно позже добавить секцию FAQ / гайды для конкретной игры --}}
@endsection
