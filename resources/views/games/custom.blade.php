@extends('layouts.landing')

@section('content')
    <section class="border-b border-white/10 bg-gradient-to-b from-[#17212b] to-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-14 md:py-18">
            <div class="grid gap-10 lg:grid-cols-2 items-center">
                <div class="space-y-4">
                    <p class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-black/10 px-3 py-1 text-[11px] font-medium text-slate-200">
                        Кастомные игровые серверы
                    </p>
                    <h1 class="text-3xl md:text-4xl lg:text-5xl font-semibold tracking-tight text-slate-100">
                        Любая игра, любой стек — мы развёрнём
                    </h1>
                    <p class="text-sm md:text-base text-slate-300/80 max-w-xl">
                        Нестандартные игровые сервера, приватные билды, кастомные лаунчеры и внутриигровые сервисы — всё это можно
                        развернуть в нашей инфраструктуре.
                    </p>
                    <div class="flex flex-wrap items-center gap-3 text-xs">
                        <a href="{{ route('dashboard') }}" class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-slate-800">
                            Перейти в панель
                        </a>
                        <a href="{{ route('games') }}" class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-medium text-slate-200 hover:bg-black/15 hover:text-white">
                            ← Ко всем играм
                        </a>
                    </div>
                    <dl class="mt-4 grid grid-cols-3 gap-4 text-[11px] text-slate-300/80 max-w-md">
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Среды</dt>
                            <dd class="mt-1 text-slate-100">Docker, bare metal</dd>
                        </div>
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Языки</dt>
                            <dd class="mt-1 text-slate-100">C#, Java, Go, Node.js и др.</dd>
                        </div>
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Масштаб</dt>
                            <dd class="mt-1 text-slate-100">От одного узла до кластера</dd>
                        </div>
                    </dl>
                </div>
                <div class="lg:pl-6">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-4 shadow-lg shadow-black/20 flex flex-col gap-3">
                        <p class="text-[11px] uppercase tracking-wide text-slate-300/70">Примеры сценариев</p>
                        <div class="relative overflow-hidden rounded-2xl border border-white/10 bg-black/10">
                            <img
                                src="/images/games/custom-main.jpg"
                                alt="Кастомный игровой сервер и панель управления"
                                class="h-44 w-full object-cover object-center md:h-52"
                            >
                            <div class="pointer-events-none absolute inset-0 bg-gradient-to-t from-slate-900/45 via-slate-900/0"></div>
                            <div class="absolute bottom-2 left-2 right-2 flex items-center justify-between text-[11px] text-slate-50">
                                <div>
                                    <p class="font-semibold">Турнир или ивент под ключ</p>
                                    <p class="text-[10px] text-slate-200">Игровой сервер + панель + аналитика</p>
                                </div>
                            </div>
                        </div>

                        <div class="grid grid-cols-2 gap-2 text-[11px] text-slate-300/80">
                            <div class="flex flex-col gap-1">
                                <div class="overflow-hidden rounded-xl border border-white/10 bg-black/10">
                                    <img
                                        src="/images/games/custom-api.jpg"
                                        alt="API и веб‑панель рядом с игровым сервером"
                                        class="h-20 w-full object-cover object-center hover:scale-[1.03] transition-transform"
                                    >
                                </div>
                                <p class="font-medium text-slate-100">Многокомпонентные проекты</p>
                                <p class="text-[10px]">Игровой сервер, API, веб‑панель и фоновые сервисы в одном кластере.</p>
                            </div>
                            <div class="flex flex-col gap-1">
                                <div class="overflow-hidden rounded-xl border border-white/10 bg-black/10">
                                    <img
                                        src="/images/games/custom-staging.jpg"
                                        alt="Staging окружение для новых билдов"
                                        class="h-20 w-full object-cover object-center hover:scale-[1.03] transition-transform"
                                    >
                                </div>
                                <p class="font-medium text-slate-100">Staging‑окружения</p>
                                <p class="text-[10px]">Быстрая обкатка новых билдов и модов в отдельных средах.</p>
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
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Что мы предлагаем</h2>
                <p class="mt-1 text-sm text-slate-300/80">Гибкая платформа для нестандартных игровых проектов.</p>
            </div>
            <div class="grid gap-5 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Совместный дизайн архитектуры</h3>
                    <p class="text-xs text-slate-300/80">Поможем подобрать конфигурацию узлов, сетей и сервисов под ваш проект.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Наблюдаемость и SLA</h3>
                    <p class="text-xs text-slate-300/80">Мониторинг, логи, алерты и договорённости по SLA для продакшн‑проектов.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Помощь с миграцией</h3>
                    <p class="text-xs text-slate-300/80">Перенос с других площадок с минимальным простоем и сохранением данных.</p>
                </div>
            </div>
        </div>
    </section>
@endsection
