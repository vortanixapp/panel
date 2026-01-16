@extends('layouts.landing')

@section('content')
    <section class="border-b border-white/10 bg-gradient-to-b from-[#17212b] to-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-14 md:py-18">
            <div class="grid gap-10 lg:grid-cols-2 items-center">
                <div class="space-y-4">
                    <p class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-black/10 px-3 py-1 text-[11px] font-medium text-orange-300">
                        Rust серверы
                    </p>
                    <h1 class="text-3xl md:text-4xl lg:text-5xl font-semibold tracking-tight text-slate-100">
                        Rust сервера для вайп‑проектов и больших онлайнов
                    </h1>
                    <p class="text-sm md:text-base text-slate-300/80 max-w-xl">
                        Готовая инфраструктура под частые вайпы, сложные карты и активный моддинг. Стабильная работа даже при
                        пиковых онлайнах.
                    </p>
                    <div class="flex flex-wrap items-center gap-3 text-xs">
                        <a href="{{ route('dashboard') }}" class="inline-flex items-center rounded-md bg-orange-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-orange-500">
                            Запустить Rust сервер
                        </a>
                        <a href="{{ route('games') }}" class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-medium text-slate-200 hover:bg-black/15 hover:text-white">
                            ← Ко всем играм
                        </a>
                    </div>
                    <dl class="mt-4 grid grid-cols-3 gap-4 text-[11px] text-slate-300/80 max-w-md">
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Моды</dt>
                            <dd class="mt-1 text-slate-100">Oxide / uMod</dd>
                        </div>
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Игроки</dt>
                            <dd class="mt-1 text-slate-100">от 50 до 300+ онлайна</dd>
                        </div>
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Вайпы</dt>
                            <dd class="mt-1 text-slate-100">гибкое расписание</dd>
                        </div>
                    </dl>
                </div>
                <div class="lg:pl-6">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-4 shadow-lg shadow-black/20 flex flex-col gap-3">
                        <p class="text-[11px] uppercase tracking-wide text-slate-300/70">Мир после вайпа</p>
                        <div class="relative overflow-hidden rounded-2xl border border-white/10 bg-black/10">
                            <img
                                src="/images/games/rust-main.jpg"
                                alt="Rust — вид на укреплённую базу после вайпа"
                                class="h-44 w-full object-cover object-center md:h-52"
                            >
                            <div class="pointer-events-none absolute inset-0 bg-gradient-to-t from-slate-900/45 via-slate-900/0"></div>
                            <div class="absolute bottom-2 left-2 right-2 flex items-center justify-between text-[11px] text-slate-50">
                                <div>
                                    <p class="font-semibold">Новый сезон — новые истории</p>
                                    <p class="text-[10px] text-slate-200">Стабильный онлайн даже к концу вайпа</p>
                                </div>
                                <span class="inline-flex items-center rounded-full bg-orange-500/80 px-2 py-0.5 text-[10px] font-medium">
                                    150+ онлайна
                                </span>
                            </div>
                        </div>

                        <div class="grid grid-cols-2 gap-2 text-[11px] text-slate-300/80">
                            <div class="flex flex-col gap-1">
                                <div class="overflow-hidden rounded-xl border border-white/10 bg-black/10">
                                    <img
                                        src="/images/games/rust-raid.jpg"
                                        alt="Rust рейд на базу"
                                        class="h-20 w-full object-cover object-center hover:scale-[1.03] transition-transform"
                                    >
                                </div>
                                <p class="font-medium text-slate-100">Рейды и оборона</p>
                                <p class="text-[10px]">Гладкий геймплей даже при большом количестве игроков и взрывов.</p>
                            </div>
                            <div class="flex flex-col gap-1">
                                <div class="overflow-hidden rounded-xl border border-white/10 bg-black/10">
                                    <img
                                        src="/images/games/rust-landscape.jpg"
                                        alt="Rust пейзаж"
                                        class="h-20 w-full object-cover object-center hover:scale-[1.03] transition-transform"
                                    >
                                </div>
                                <p class="font-medium text-slate-100">Большие карты</p>
                                <p class="text-[10px]">Комфортная игра на масштабных ландшафтах с множеством построек.</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </section>

    <section class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14 text-sm">
            <div class="mb-6 max-w-2xl">
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Удобства для админов и игроков</h2>
                <p class="mt-1 text-sm text-slate-300/80">Инструменты, которые экономят время команды и повышают стабильность проекта.</p>
            </div>
            <div class="grid gap-5 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Гибкое управление вайпами</h3>
                    <p class="text-xs text-slate-300/80">Настройка расписания вайпов, автоматические анонсы и бэкапы перед очисткой мира.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Мониторинг</h3>
                    <p class="text-xs text-slate-300/80">Графики нагрузки, алерты по CPU/RAM и диску, логи для быстрой диагностики.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Поддержка модов</h3>
                    <p class="text-xs text-slate-300/80">Удобная установка и обновление плагинов, отдельная среда для тестирования обновлений.</p>
                </div>
            </div>
        </div>
    </section>

    <section class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14 text-sm">
            <div class="mb-6 max-w-2xl">
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Технические особенности Rust‑серверов</h2>
                <p class="mt-1 text-sm text-slate-300/80">Проектируем инфраструктуру с учётом частых вайпов и тяжёлых плагинов.</p>
            </div>
            <div class="grid gap-5 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Вайпы без стресса</h3>
                    <p class="text-xs text-slate-300/80">Авто‑бэкапы перед вайпом, быстрый откат и удобное управление расписанием через панель.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Моды и плагины</h3>
                    <p class="text-xs text-slate-300/80">Поддержка Oxide/uMod, отдельные окружения для тестирования обновлений и отката проблемных плагинов.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Мониторинг и алерты</h3>
                    <p class="text-xs text-slate-300/80">Графики нагрузки, уведомления при пиковом онлайне и логирование для быстрой диагностики.</p>
                </div>
            </div>
        </div>
    </section>
@endsection
