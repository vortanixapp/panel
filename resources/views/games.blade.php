@extends('layouts.landing')

@section('content')
    {{-- Hero / промо-блок --}}
    <section class="relative border-b border-white/10 bg-gradient-to-b from-[#17212b] to-[#17212b]">
        <div class="relative mx-auto max-w-6xl px-4 py-14 md:py-20">
            <div class="grid items-center gap-10 lg:grid-cols-2">
                <div class="space-y-6">
                    <p class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-black/10 px-3 py-1 text-[11px] font-medium text-slate-200">
                        Каталог поддерживаемых игр
                    </p>
                    <h1 class="text-3xl md:text-4xl lg:text-5xl font-semibold tracking-tight text-slate-100">
                        Разворачивайте игровые серверы
                        <span class="bg-gradient-to-r from-sky-500 via-emerald-400 to-indigo-500 bg-clip-text text-transparent">за считанные секунды</span>
                    </h1>
                    <p class="max-w-xl text-sm md:text-base text-slate-300/80">
                        Выберите игру, задайте конфигурацию и получите готовый сервер с DDoS‑защитой, резервными копиями и
                        удобной панелью управления.
                    </p>
                    <div class="flex flex-wrap items-center gap-4">
                        <a href="{{ route('dashboard') }}" class="inline-flex items-center rounded-md bg-sky-600 px-4 py-2 text-xs md:text-sm font-semibold text-white shadow-sm hover:bg-sky-500 transition">
                            Развернуть сервер
                        </a>
                        <a href="#games-list" class="inline-flex items-center gap-2 rounded-md border border-white/10 bg-black/10 px-4 py-2 text-xs md:text-sm font-medium text-slate-200 hover:bg-black/15 hover:text-white transition">
                            Смотреть список игр
                            <span class="text-slate-400">↓</span>
                        </a>
                    </div>
                </div>

                <div class="lg:pl-6">
                    <div class="relative rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-xl shadow-black/20">
                        <p class="text-[11px] uppercase tracking-wide text-slate-300/70">Пример пресета</p>
                        <p class="mt-1 text-sm font-semibold text-slate-100">CS2 • 32 слота • EU</p>
                        <ul class="mt-3 space-y-1.5 text-[12px] text-slate-300/80">
                            <li>2 vCPU • 4 ГБ RAM • NVMe 40 ГБ</li>
                            <li>DDoS‑защита, авто‑перезапуск, мониторинг загрузки</li>
                            <li>Развёртывание сервера ~ 40 секунд</li>
                        </ul>
                        <button class="mt-4 inline-flex items-center rounded-md bg-sky-600 px-4 py-2 text-[11px] font-semibold text-white hover:bg-sky-500">
                            Выбрать этот пресет
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </section>

    {{-- Список игр карточками --}}
    <section id="games-list" class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14">
            <div class="mb-6 flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Поддерживаемые игры</h2>
                    <p class="mt-1 max-w-2xl text-sm text-slate-300/80">
                        Готовые шаблоны конфигураций, оптимизированные под конкретные игровые движки и режимы.
                    </p>
                </div>
            </div>

            <div class="grid gap-5 md:grid-cols-3 text-sm">
                @foreach ([
                    [
                        'slug' => 'minecraft',
                        'title' => 'Minecraft',
                        'genre' => 'Песочница, выживание, мод‑сборки',
                        'platforms' => 'Java / Paper / Purpur / Modded',
                        'desc' => 'Готовые пресеты под ваниль, Bukkit‑ядра и тяжёлые мод‑сборки. Авто‑бэкапы миров, быстрый откат снапшотов.',
                    ],
                    [
                        'slug' => 'cs2-csgo',
                        'title' => 'CS2 / CS:GO',
                        'genre' => 'Шутер, соревновательные режимы',
                        'platforms' => 'Linux • 128 tickrate',
                        'desc' => 'Сервера для пабликов, миксов и лиг. Стабильный FPS, низкий пинг, удобный ротационный конфиг.',
                    ],
                    [
                        'slug' => 'rust',
                        'title' => 'Rust',
                        'genre' => 'Выживание, вайп‑проекты',
                        'platforms' => 'Linux • Oxide / uMod',
                        'desc' => 'Готово к большим онлайнам: высокий онлайн, периодические вайпы, моды и плагины.',
                    ],
                    [
                        'slug' => 'valheim',
                        'title' => 'Valheim',
                        'genre' => 'Кооперативное выживание',
                        'platforms' => 'Linux • Crossplay',
                        'desc' => 'Быстрый запуск приватных и публичных серверов, автоматическое сохранение прогресса.',
                    ],
                    [
                        'slug' => 'automation',
                        'title' => 'Satisfactory / Factorio',
                        'genre' => 'Автоматизация и заводы',
                        'platforms' => 'Linux',
                        'desc' => 'Оптимизировано под долгую работу и тяжёлые карты с большим количеством логики.',
                    ],
                    [
                        'slug' => 'custom',
                        'title' => 'Другая игра',
                        'genre' => 'Свой мод‑проект или кастомный сервер',
                        'platforms' => 'Docker / Bare metal',
                        'desc' => 'Разворачивайте любые серверы в Docker‑контейнерах или на выделенных узлах.',
                    ],
                ] as $game)
                    <a href="{{ route('games.show', $game['slug']) }}" class="group flex flex-col rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm hover:border-sky-500/50 hover:bg-black/10 transition">
                        <div class="mb-2 flex items-center justify-between gap-2">
                            <div>
                                <p class="text-xs font-semibold text-slate-100">{{ $game['title'] }}</p>
                                <p class="mt-0.5 text-[11px] text-slate-300/70">{{ $game['genre'] }}</p>
                            </div>
                            <span class="inline-flex items-center rounded-full bg-sky-500/10 px-2 py-0.5 text-[10px] font-medium text-sky-300 ring-1 ring-sky-500/20">
                                {{ $game['platforms'] }}
                            </span>
                        </div>
                        <p class="text-xs text-slate-300/80 flex-1">{{ $game['desc'] }}</p>
                        <div class="mt-3 flex flex-wrap items-center gap-2 text-[11px]">
                            <span class="inline-flex items-center rounded-md bg-sky-600 px-3 py-1.5 text-[11px] font-medium text-white group-hover:bg-sky-500">
                                Развернуть сервер
                            </span>
                            <span class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 group-hover:bg-black/15 group-hover:text-white">
                                Подробнее
                            </span>
                        </div>
                    </a>
                @endforeach
            </div>
        </div>
    </section>

    {{-- Преимущества сервиса --}}
    <section class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14">
            <div class="mb-6 flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Почему запускать сервер у нас</h2>
                    <p class="mt-1 max-w-2xl text-sm text-slate-300/80">
                        Строим платформу как облачный провайдер: высокая отказоустойчивость, наблюдаемость и удобная панель управления,
                        адаптированные под игровые нагрузки.
                    </p>
                </div>
            </div>

            <div class="grid gap-5 md:grid-cols-3 text-sm">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Производительность и стабильность</h3>
                    <p class="text-xs text-slate-300/80">NVMe‑хранилище, современные CPU и оптимизированные образы игр обеспечивают
                        стабильный FPS и низкий пинг даже при высоких онлайнах.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Безопасность и бэкапы</h3>
                    <p class="text-xs text-slate-300/80">DDoS‑защита на периметре, регулярные резервные копии и снапшоты позволяют
                        спокойно переживать пики трафика и эксперименты с модами.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Инструменты для команд и сообществ</h3>
                    <p class="text-xs text-slate-300/80">Управление несколькими серверами, роли для команды, метрики и логи в реальном
                        времени — всё в одном личном кабинете.</p>
                </div>
            </div>
        </div>
    </section>
@endsection
