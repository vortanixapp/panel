@extends('layouts.landing')

@section('content')
    <section class="border-b border-white/10 bg-gradient-to-b from-[#17212b] to-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-14 md:py-18">
            <div class="grid gap-10 lg:grid-cols-2 items-center">
                <div class="space-y-4">
                    <p class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-black/10 px-3 py-1 text-[11px] font-medium text-yellow-300">
                        Satisfactory / Factorio серверы
                    </p>
                    <h1 class="text-3xl md:text-4xl lg:text-5xl font-semibold tracking-tight text-slate-100">
                        Серверы для заводов и автоматизации
                    </h1>
                    <p class="text-sm md:text-base text-slate-300/80 max-w-xl">
                        Долгоживущие миры с тяжёлой логикой, сложными базами и большим числом сущностей — наши конфигурации готовы к
                        такому сценарию.
                    </p>
                    <div class="flex flex-wrap items-center gap-3 text-xs">
                        <a href="{{ route('dashboard') }}" class="inline-flex items-center rounded-md bg-yellow-500 px-4 py-2 text-xs font-semibold text-slate-100 shadow-sm hover:bg-yellow-400">
                            Запустить сервер Satisfactory / Factorio
                        </a>
                        <a href="{{ route('games') }}" class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-medium text-slate-200 hover:bg-black/15 hover:text-white">
                            ← Ко всем играм
                        </a>
                    </div>
                    <dl class="mt-4 grid grid-cols-3 gap-4 text-[11px] text-slate-300/80 max-w-md">
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Тип проектов</dt>
                            <dd class="mt-1 text-slate-100">Кооператив, большие заводы</dd>
                        </div>
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Онлайн</dt>
                            <dd class="mt-1 text-slate-100">от 2 до 20+ игроков</dd>
                        </div>
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Работа</dt>
                            <dd class="mt-1 text-slate-100">Круглосуточно</dd>
                        </div>
                    </dl>
                </div>
                <div class="lg:pl-6">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-4 shadow-lg shadow-black/20 flex flex-col gap-3">
                        <p class="text-[11px] uppercase tracking-wide text-slate-300/70">Ваши заводы в действии</p>
                        <div class="relative overflow-hidden rounded-2xl border border-white/10 bg-black/10">
                            <img
                                src="/images/games/factorio-main.jpg"
                                alt="Factorio — крупная фабрика сверху"
                                class="h-44 w-full object-cover object-center md:h-52"
                            >
                            <div class="pointer-events-none absolute inset-0 bg-gradient-to-t from-slate-900/45 via-slate-900/0"></div>
                            <div class="absolute bottom-2 left-2 right-2 flex items-center justify-between text-[11px] text-slate-50">
                                <div>
                                    <p class="font-semibold">Фабрика, работающая 24/7</p>
                                    <p class="text-[10px] text-slate-200">Снапшоты перед крупными изменениями схемы</p>
                                </div>
                                <span class="inline-flex items-center rounded-full bg-yellow-500/80 px-2 py-0.5 text-[10px] font-medium text-slate-100">
                                    TPS стабилен
                                </span>
                            </div>
                        </div>

                        <div class="grid grid-cols-2 gap-2 text-[11px] text-slate-300/80">
                            <div class="flex flex-col gap-1">
                                <div class="overflow-hidden rounded-xl border border-white/10 bg-black/10">
                                    <img
                                        src="/images/games/factorio-belt.jpg"
                                        alt="Factorio ленты и логистика"
                                        class="h-20 w-full object-cover object-center hover:scale-[1.03] transition-transform"
                                    >
                                </div>
                                <p class="font-medium text-slate-100">Сложная логистика</p>
                                <p class="text-[10px]">Готово к миллионам предметов на лентах и поездах.</p>
                            </div>
                            <div class="flex flex-col gap-1">
                                <div class="overflow-hidden rounded-xl border border-white/10 bg-black/10">
                                    <img
                                        src="/images/games/satisfactory-main.jpg"
                                        alt="Satisfactory завод в 3D"
                                        class="h-20 w-full object-cover object-center hover:scale-[1.03] transition-transform"
                                    >
                                </div>
                                <p class="font-medium text-slate-100">3D‑заводы</p>
                                <p class="text-[10px]">Красивая картинка и плавный геймплей для кооп‑проектов.</p>
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
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Для любителей оптимизации</h2>
                <p class="mt-1 text-sm text-slate-300/80">Получайте максимум производительности от своих заводов.</p>
            </div>
            <div class="grid gap-5 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Стабильный TPS</h3>
                    <p class="text-xs text-slate-300/80">Производительные CPU и оптимизированные конфиги под тиковую нагрузку.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Бэкапы и снапшоты</h3>
                    <p class="text-xs text-slate-300/80">Сохраняйте ключевые этапы развития фабрики, легко откатываясь назад при ошибках.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Наблюдаемость</h3>
                    <p class="text-xs text-slate-300/80">Мониторинг ресурсов и логирование событий помогают вовремя замечать узкие места.</p>
                </div>
            </div>
        </div>
    </section>
@endsection
