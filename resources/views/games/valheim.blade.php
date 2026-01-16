@extends('layouts.landing')

@section('content')
    <section class="border-b border-white/10 bg-gradient-to-b from-[#17212b] to-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-14 md:py-18">
            <div class="grid gap-10 lg:grid-cols-2 items-center">
                <div class="space-y-4">
                    <p class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-black/10 px-3 py-1 text-[11px] font-medium text-indigo-300">
                        Valheim серверы
                    </p>
                    <h1 class="text-3xl md:text-4xl lg:text-5xl font-semibold tracking-tight text-slate-100">
                        Надёжные кооп‑серверы Valheim для друзей и сообществ
                    </h1>
                    <p class="text-sm md:text-base text-slate-300/80 max-w-xl">
                        Запускайте приватные и публичные миры с автоматическими сохранениями, удобной настройкой слотов и поддержкой
                        crossplay.
                    </p>
                    <div class="flex flex-wrap items-center gap-3 text-xs">
                        <a href="{{ route('dashboard') }}" class="inline-flex items-center rounded-md bg-indigo-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-indigo-500">
                            Запустить Valheim сервер
                        </a>
                        <a href="{{ route('games') }}" class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-medium text-slate-200 hover:bg-black/15 hover:text-white">
                            ← Ко всем играм
                        </a>
                    </div>
                    <dl class="mt-4 grid grid-cols-3 gap-4 text-[11px] text-slate-300/80 max-w-md">
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Слоты</dt>
                            <dd class="mt-1 text-slate-100">от 2 до 20 игроков</dd>
                        </div>
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Режим</dt>
                            <dd class="mt-1 text-slate-100">Приватный и публичный</dd>
                        </div>
                        <div>
                            <dt class="uppercase tracking-wide text-slate-300/70">Crossplay</dt>
                            <dd class="mt-1 text-slate-100">Поддерживается</dd>
                        </div>
                    </dl>
                </div>
                <div class="lg:pl-6">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-4 shadow-lg shadow-black/20 flex flex-col gap-3">
                        <p class="text-[11px] uppercase tracking-wide text-slate-300/70">Кадры из вашего мира</p>
                        <div class="relative overflow-hidden rounded-2xl border border-white/10 bg-black/10">
                            <img
                                src="/images/games/valheim-main.jpg"
                                alt="Valheim — вид на лагерь в тумане"
                                class="h-44 w-full object-cover object-center md:h-52"
                            >
                            <div class="pointer-events-none absolute inset-0 bg-gradient-to-t from-slate-900/45 via-slate-900/0"></div>
                            <div class="absolute bottom-2 left-2 right-2 flex items-center justify-between text-[11px] text-slate-50">
                                <div>
                                    <p class="font-semibold">Тихий лагерь у костра</p>
                                    <p class="text-[10px] text-slate-200">Идеально для совместных приключений</p>
                                </div>
                                <span class="inline-flex items-center rounded-full bg-indigo-500/80 px-2 py-0.5 text-[10px] font-medium">
                                    2–10 игроков
                                </span>
                            </div>
                        </div>

                        <div class="grid grid-cols-2 gap-2 text-[11px] text-slate-300/80">
                            <div class="flex flex-col gap-1">
                                <div class="overflow-hidden rounded-xl border border-white/10 bg-black/10">
                                    <img
                                        src="/images/games/valheim-base.jpg"
                                        alt="Valheim база игроков"
                                        class="h-20 w-full object-cover object-center hover:scale-[1.03] transition-transform"
                                    >
                                </div>
                                <p class="font-medium text-slate-100">Дом и пристань</p>
                                <p class="text-[10px]">Стабильные сохранения и быстрый доступ для всей компании.</p>
                            </div>
                            <div class="flex flex-col gap-1">
                                <div class="overflow-hidden rounded-xl border border-white/10 bg-black/10">
                                    <img
                                        src="/images/games/valheim-boss.jpg"
                                        alt="Valheim бой с боссом"
                                        class="h-20 w-full object-cover object-center hover:scale-[1.03] transition-transform"
                                    >
                                </div>
                                <p class="font-medium text-slate-100">Бои с боссами</p>
                                <p class="text-[10px]">Стабильный сервер во время самых напряжённых сражений.</p>
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
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Удобно для коопа</h2>
                <p class="mt-1 text-sm text-slate-300/80">Максимум удобства для совместного приключения.</p>
            </div>
            <div class="grid gap-5 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Простой доступ</h3>
                    <p class="text-xs text-slate-300/80">IP и порт сервера доступны сразу после развёртывания, есть подсказки по подключению.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Надёжные сохранения</h3>
                    <p class="text-xs text-slate-300/80">Регулярные бэкапы мира и возможность отката к стабильной точке.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Гибкие ресурсы</h3>
                    <p class="text-xs text-slate-300/80">Лёгкое масштабирование ресурсов при росте онлайна или числа друзей.</p>
                </div>
            </div>
        </div>
    </section>

    <section class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14 text-sm">
            <div class="mb-6 max-w-2xl">
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Детали Valheim‑хостинга</h2>
                <p class="mt-1 text-sm text-slate-300/80">Сфокусированы на удобстве для небольшой компании друзей.</p>
            </div>
            <div class="grid gap-5 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Слоты и доступ</h3>
                    <p class="text-xs text-slate-300/80">Быстрая смена пароля, управление whitelists и настройка числа игроков без перезапуска мира.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Сохранения и откат</h3>
                    <p class="text-xs text-slate-300/80">Периодические бэкапы мира и возможность отката к стабильной точке перед крупными приключениями.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Ресурсы под рост</h3>
                    <p class="text-xs text-slate-300/80">Увеличивайте ресурсы сервера по мере роста компании и прогресса без сложных миграций.</p>
                </div>
            </div>
        </div>
    </section>
@endsection
