@extends('layouts.landing')

@section('content')
    <section class="border-b border-white/10 bg-gradient-to-b from-[#17212b] to-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-14 md:py-18">
            <div class="grid gap-10 lg:grid-cols-2 items-center">
                <div class="space-y-4">
                    <p class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-black/10 px-3 py-1 text-[11px] font-medium text-emerald-300">
                        Minecraft серверы
                    </p>
                    <h1 class="text-3xl md:text-4xl lg:text-5xl font-semibold tracking-tight text-slate-100">
                        Идеальная площадка для ваших миров Minecraft
                    </h1>
                    <p class="text-sm md:text-base text-slate-300/80 max-w-xl">
                        Выживание с друзьями, мод‑сборки или сеть мини‑игр — мы подберём конфигурацию под ваш проект: быстрые NVMe,
                        стабильный TPS и автоматические бэкапы.
                    </p>
                    <div class="flex flex-wrap items-center gap-3 text-xs">
                        <a href="{{ route('dashboard') }}" class="inline-flex items-center rounded-md bg-emerald-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-emerald-500">
                            Запустить Minecraft сервер
                        </a>
                        <a href="{{ route('games') }}" class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-medium text-slate-200 hover:bg-black/15 hover:text-white">
                            ← Ко всем играм
                        </a>
                    </div>
                    <dl class="mt-4 grid grid-cols-3 gap-4 text-[11px] text-slate-300/80 max-w-md">
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Типы серверов</dt>
                            <dd class="mt-1 text-slate-100">Vanilla, Paper, Purpur, Forge</dd>
                        </div>
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Онлайн</dt>
                            <dd class="mt-1 text-slate-100">от 2 до 300+ игроков</dd>
                        </div>
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Бэкапы</dt>
                            <dd class="mt-1 text-slate-100">Авто каждый день</dd>
                        </div>
                    </dl>
                </div>
                <div class="lg:pl-6">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-4 shadow-lg shadow-black/20 flex flex-col gap-3">
                        <p class="text-[11px] uppercase tracking-wide text-slate-300/70">Как может выглядеть ваш мир</p>
                        <div class="relative overflow-hidden rounded-2xl border border-white/10 bg-black/10">
                            <img
                                src="/images/games/minecraft-main.jpg"
                                alt="Minecraft выживание — вид на базу игрока"
                                class="h-44 w-full object-cover object-center md:h-52"
                            >
                            <div class="pointer-events-none absolute inset-0 bg-gradient-to-t from-slate-900/40 via-slate-900/0"></div>
                            <div class="absolute bottom-2 left-2 right-2 flex items-center justify-between text-[11px] text-slate-50">
                                <div>
                                    <p class="font-semibold">Тёплый ламповый спавн</p>
                                    <p class="text-[10px] text-slate-200">Приватный сервер для друзей или паблик — выбор за вами</p>
                                </div>
                                <span class="inline-flex items-center rounded-full bg-emerald-500/80 px-2 py-0.5 text-[10px] font-medium">
                                    Online 20–40
                                </span>
                            </div>
                        </div>

                        <div class="grid grid-cols-2 gap-2 text-[11px] text-slate-300/80">
                            <div class="flex flex-col gap-1">
                                <div class="overflow-hidden rounded-xl border border-white/10 bg-black/10">
                                    <img
                                        src="/images/games/minecraft-base.jpg"
                                        alt="База игрока Minecraft"
                                        class="h-20 w-full object-cover object-center hover:scale-[1.03] transition-transform"
                                    >
                                </div>
                                <p class="font-medium text-slate-100">База и фермы</p>
                                <p class="text-[10px]">Стабильный TPS для автоматизации и редстоун‑механизмов.</p>
                            </div>
                            <div class="flex flex-col gap-1">
                                <div class="overflow-hidden rounded-xl border border-white/10 bg-black/10">
                                    <img
                                        src="/images/games/minecraft-minigames.jpg"
                                        alt="Мини‑игры на сервере Minecraft"
                                        class="h-20 w-full object-cover object-center hover:scale-[1.03] transition-transform"
                                    >
                                </div>
                                <p class="font-medium text-slate-100">Мини‑игры и спавн</p>
                                <p class="text-[10px]">Подходит для хабов, лобби и сетей мини‑игр.</p>
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
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Для каких проектов подойдёт</h2>
                <p class="mt-1 text-sm text-slate-300/80">От небольших приватных серверов до крупных сетей с мини‑играми и модами.</p>
            </div>
            <div class="grid gap-5 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Приватное выживание</h3>
                    <p class="text-xs text-slate-300/80">Идеально для игры с друзьями: стабильный TPS, бэкапы и простая настройка доступа.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Публичный проект</h3>
                    <p class="text-xs text-slate-300/80">Плагины, античит, мониторинг нагрузки и защита от DDoS для серверов с высоким онлайном.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Мод‑сборки</h3>
                    <p class="text-xs text-slate-300/80">Поддержка Forge/Fabric, гибкая настройка памяти и удобный откат к стабильным снапшотам.</p>
                </div>
            </div>
        </div>
    </section>

    <section class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14 text-sm">
            <div class="mb-6 max-w-2xl">
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Технические детали Minecraft‑хостинга</h2>
                <p class="mt-1 text-sm text-slate-300/80">Построено так, чтобы и ваниль, и тяжёлые мод‑сборки работали стабильно.</p>
            </div>
            <div class="grid gap-5 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Хранилище и бэкапы</h3>
                    <p class="text-xs text-slate-300/80">NVMe‑диски обеспечивают быструю генерацию чанков и сохранение миров. Автоматические бэкапы позволяют откатываться к стабильным снапшотам.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Оптимизация ядра</h3>
                    <p class="text-xs text-slate-300/80">Paper/Purpur, оптимальные GC‑настройки и лимиты по энтити помогают держать высокий TPS даже с фермами и автоматизацией.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Управление через панель</h3>
                    <p class="text-xs text-slate-300/80">Аптайм, графики TPS и RAM, перезапуск и загрузка новых сборок — всё доступно из одного интерфейса без SSH.</p>
                </div>
            </div>
        </div>
    </section>
@endsection
