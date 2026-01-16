@extends('layouts.landing')

@section('content')
    <section class="border-b border-white/10 bg-gradient-to-b from-[#17212b] to-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-14 md:py-18">
            <div class="grid gap-10 lg:grid-cols-2 items-center">
                <div class="space-y-4">
                    <p class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-black/10 px-3 py-1 text-[11px] font-medium text-sky-300">
                        CS2 / CS:GO серверы
                    </p>
                    <h1 class="text-3xl md:text-4xl lg:text-5xl font-semibold tracking-tight text-slate-100">
                        Стабильные сервера CS2 и CS:GO для пабликов и лиг
                    </h1>
                    <p class="text-sm md:text-base text-slate-300/80 max-w-xl">
                        Высокий tickrate, стабильный FPS, предсказуемый пинг и удобная панель — всё, что нужно для паблика, микса или
                        соревновательного сервера.
                    </p>
                    <div class="flex flex-wrap items-center gap-3 text-xs">
                        <a href="{{ route('dashboard') }}" class="inline-flex items-center rounded-md bg-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">
                            Запустить CS2 / CS:GO сервер
                        </a>
                        <a href="{{ route('games') }}" class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-medium text-slate-200 hover:bg-black/15 hover:text-white">
                            ← Ко всем играм
                        </a>
                    </div>
                    <dl class="mt-4 grid grid-cols-3 gap-4 text-[11px] text-slate-300/80 max-w-md">
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Режимы</dt>
                            <dd class="mt-1 text-slate-100">Паблик, ретейки, миксы</dd>
                        </div>
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Tickrate</dt>
                            <dd class="mt-1 text-slate-100">до 128 ticks</dd>
                        </div>
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Онлайн</dt>
                            <dd class="mt-1 text-slate-100">от 10 до 64+ слотов</dd>
                        </div>
                    </dl>
                </div>
                <div class="lg:pl-6">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-4 shadow-lg shadow-black/20 flex flex-col gap-3">
                        <p class="text-[11px] uppercase tracking-wide text-slate-300/70">Атмосфера вашего сервера</p>
                        <div class="relative overflow-hidden rounded-2xl border border-white/10 bg-black/10">
                            <img
                                src="/images/games/cs2-main.jpg"
                                alt="CS2 • раунд на соревновательной карте"
                                class="h-44 w-full object-cover object-center md:h-52"
                            >
                            <div class="pointer-events-none absolute inset-0 bg-gradient-to-t from-slate-900/40 via-slate-900/0"></div>
                            <div class="absolute bottom-2 left-2 right-2 flex items-center justify-between text-[11px] text-slate-50">
                                <div>
                                    <p class="font-semibold">Клатч‑момент на вашем сервере</p>
                                    <p class="text-[10px] text-slate-200">Паблик, ретейки или миксы — вы выбираете формат</p>
                                </div>
                                <span class="inline-flex items-center rounded-full bg-sky-500/80 px-2 py-0.5 text-[10px] font-medium">
                                    128 tick
                                </span>
                            </div>
                        </div>

                        <div class="grid grid-cols-2 gap-2 text-[11px] text-slate-300/80">
                            <div class="flex flex-col gap-1">
                                <div class="overflow-hidden rounded-xl border border-white/10 bg-black/10">
                                    <img
                                        src="/images/games/cs2-retake.jpg"
                                        alt="CS2 ретейк сервер"
                                        class="h-20 w-full object-cover object-center hover:scale-[1.03] transition-transform"
                                    >
                                </div>
                                <p class="font-medium text-slate-100">Ретейки и тренировки</p>
                                <p class="text-[10px]">Идеально для отработки раундов и игровых ситуаций.</p>
                            </div>
                            <div class="flex flex-col gap-1">
                                <div class="overflow-hidden rounded-xl border border-white/10 bg-black/10">
                                    <img
                                        src="/images/games/cs2-pub.jpg"
                                        alt="CS2 паблик сервер"
                                        class="h-20 w-full object-cover object-center hover:scale-[1.03] transition-transform"
                                    >
                                </div>
                                <p class="font-medium text-slate-100">Паблик и сообщество</p>
                                <p class="text-[10px]">Красивое лобби, ротация карт и удобная админ‑панель.</p>
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
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Создавайте свой идеальный сервер</h2>
                <p class="mt-1 text-sm text-slate-300/80">От небольшого сервера для друзей до крупного проекта с несколькими режимами.</p>
            </div>
            <div class="grid gap-5 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Паблики и ретейки</h3>
                    <p class="text-xs text-slate-300/80">Гибкое управление картами, RTV, админ‑система и логирование событий на сервере.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Соревновательные сервера</h3>
                    <p class="text-xs text-slate-300/80">Стабильный tickrate, мониторинг нагрузок и алерты, интеграция с античит‑решениями.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Сообщественные проекты</h3>
                    <p class="text-xs text-slate-300/80">Панель для команды, доступ к логам, быстрые рестарты и развёртывание новых режимов.</p>
                </div>
            </div>
        </div>
    </section>

    <section class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14 text-sm">
            <div class="mb-6 max-w-2xl">
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Тонкости CS2 / CS:GO‑хостинга</h2>
                <p class="mt-1 text-sm text-slate-300/80">Мы учитываем особенности движка и соревновательной сцены.</p>
            </div>
            <div class="grid gap-5 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Tickrate и FPS</h3>
                    <p class="text-xs text-slate-300/80">Оптимизированные конфиги и производительные CPU позволяют держать стабильный tickrate и FPS даже при полном сервере.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Демки и логи</h3>
                    <p class="text-xs text-slate-300/80">Поддержка автоматической записи демок, логирование событий и интеграция с античит‑решениями для турниров и лиг.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Удобство администрирования</h3>
                    <p class="text-xs text-slate-300/80">Смена карт и конфигов, управление слотами и рестарты доступны из панели, без сложных скриптов.</p>
                </div>
            </div>
        </div>
    </section>
@endsection
