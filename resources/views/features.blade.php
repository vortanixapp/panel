@extends('layouts.landing')

@section('content')
    {{-- Hero: возможности панели --}}
    <section class="relative border-b border-white/10 bg-gradient-to-b from-[#17212b] to-[#17212b] overflow-hidden">
        <div class="pointer-events-none absolute inset-x-0 -top-40 h-64 bg-[radial-gradient(circle_at_top,_rgba(56,189,248,0.18),_transparent_60%),_radial-gradient(circle_at_20%_20%,_rgba(129,140,248,0.16),_transparent_55%)]"></div>
        <div class="relative mx-auto max-w-6xl px-4 py-14 md:py-20">
            <div class="grid gap-10 lg:grid-cols-2 items-center">
                <div class="space-y-5">
                    <p class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-black/10 px-3 py-1 text-[11px] font-medium text-slate-200">
                        Панель управления Vortanix GameCloud
                    </p>
                    <h1 class="text-3xl md:text-4xl lg:text-5xl font-semibold tracking-tight text-slate-100">
                        Вся инфраструктура игровых серверов под контролем
                    </h1>
                    <p class="max-w-xl text-sm md:text-base text-slate-300/80">
                        Наглядные графики нагрузки, логирование, управление серверами и безопасностью — всё в одной современной панели,
                        созданной специально под игровые проекты.
                    </p>
                    <div class="flex flex-wrap items-center gap-4 text-xs">
                        <a href="{{ route('dashboard') }}" class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-xs font-semibold text-white shadow-lg shadow-slate-700/30 hover:bg-slate-800 transition">
                            Перейти в панель
                        </a>
                        <a href="{{ route('register') }}" class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-4 py-2 text-xs font-medium text-slate-200 hover:bg-black/15 hover:text-white transition">
                            Зарегистрироваться
                        </a>
                    </div>
                    <div class="mt-4 flex flex-wrap gap-2 text-[10px] text-slate-300/70">
                        <span class="inline-flex items-center gap-1 rounded-full bg-black/10 px-2.5 py-1 border border-white/10">
                            <span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
                            Мониторинг в реальном времени
                        </span>
                        <span class="inline-flex items-center gap-1 rounded-full bg-black/10 px-2.5 py-1 border border-white/10">
                            <span class="h-1.5 w-1.5 rounded-full bg-sky-500"></span>
                            Управление демонами и локациями
                        </span>
                        <span class="inline-flex items-center gap-1 rounded-full bg-black/10 px-2.5 py-1 border border-white/10">
                            <span class="h-1.5 w-1.5 rounded-full bg-violet-500"></span>
                            Логи и безопасность
                        </span>
                    </div>
                </div>
                <div class="lg:pl-6">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-4 shadow-xl shadow-black/20 flex flex-col gap-3 backdrop-blur-sm">
                        <p class="text-[11px] uppercase tracking-wide text-slate-300/70">Обзор панели</p>
                        <div class="relative overflow-hidden rounded-2xl border border-white/10 bg-black/10">
                            <img
                                src="/images/panel/overview.jpg"
                                alt="Обзор панели управления Vortanix GameCloud"
                                class="h-52 w-full object-cover object-center md:h-60"
                            >
                            <div class="pointer-events-none absolute inset-0 bg-gradient-to-t from-slate-900/50 via-slate-900/0"></div>
                            <div class="absolute bottom-2 left-2 right-2 flex items-center justify-between text-[11px] text-slate-50">
                                <div>
                                    <p class="font-semibold">Графики, демоны и локации</p>
                                    <p class="text-[10px] text-slate-200">Всё, что важно администратору, на одном экране</p>
                                </div>
                                <span class="inline-flex items-center rounded-full bg-emerald-500/80 px-2 py-0.5 text-[10px] font-medium">
                                    Online
                                </span>
                            </div>
                        </div>
                        <p class="text-[11px] text-slate-300/80">
                            Панель строится поверх API, поэтому все действия доступны и программно: автоматизируйте развертывание,
                            бэкапы и управление через свои скрипты.
                        </p>
                    </div>
                </div>
            </div>
        </div>
    </section>

    {{-- Ключевые возможности панели --}}
    <section class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14 text-sm">
            <div class="mb-6 max-w-2xl">
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Что умеет панель управления</h2>
                <p class="mt-1 text-sm text-slate-300/80">Инструменты для ежедневной работы администраторов и команд.</p>
            </div>
            <div class="grid gap-5 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-transform">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Мониторинг нагрузки</h3>
                    <p class="text-xs text-slate-300/80">Графики CPU, RAM, диска и аптайма по локациям и демонам, история метрик для анализа пиков.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-transform">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Управление демонами и сервисами</h3>
                    <p class="text-xs text-slate-300/80">Статусы Vortanix Daemon, Docker, MySQL, FTP и других служб с возможностью перезапуска и установки.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-transform">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Логи в реальном времени</h3>
                    <p class="text-xs text-slate-300/80">Real‑time консоль демона, просмотр логов последних операций, удобный поиск по ошибкам.</p>
                </div>
            </div>
        </div>
    </section>

    {{-- Безопасность и доступы --}}
    <section class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14 text-sm">
            <div class="mb-6 max-w-2xl">
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Безопасность и управление доступом</h2>
                <p class="mt-1 text-sm text-slate-300/80">Контролируйте, кто и что может делать с инфраструктурой.</p>
            </div>
            <div class="grid gap-5 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-transform">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Профиль и 2FA</h3>
                    <p class="text-xs text-slate-300/80">Управление контактами, паролем, двухфакторной аутентификацией и активными сессиями прямо из панели.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-transform">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">SSH‑доступ и команды</h3>
                    <p class="text-xs text-slate-300/80">Централизованное хранение SSH‑доступов к локациям и безопасное выполнение команд через Vortanix Daemon.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow_sm hover:shadow-md hover:-translate-y-0.5 transition-transform">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Роли и разделение прав</h3>
                    <p class="text-xs text-slate-300/80">(Планируемо) Роли для команды и разграничение доступов к локациям, играм и биллингу.</p>
                </div>
            </div>
        </div>
    </section>

    {{-- UX панели: как выглядит работа --}}
    <section class="bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14 text-sm">
            <div class="mb-6 max-w-2xl">
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Как вы будете работать с панелью</h2>
                <p class="mt-1 text-sm text-slate-300/80">Минимум кликов до основных действий.</p>
            </div>
            <div class="grid gap-5 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 flex flex-col gap-2 shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-transform">
                    <span class="inline-flex h-7 w-7 items-center justify-center rounded-full bg-sky-600 text-[11px] font-semibold text-white">1</span>
                    <h3 class="text-sm font-semibold text-slate-100">Выбор локации и игры</h3>
                    <p class="text-xs text-slate-300/80">Зайдите в панель, выберите нужную локацию и игру из каталога.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 flex flex-col gap-2 shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-transform">
                    <span class="inline-flex h-7 w-7 items-center justify-center rounded-full bg-sky-600 text-[11px] font-semibold text-white">2</span>
                    <h3 class="text-sm font-semibold text-slate-100">Запуск и наблюдение</h3>
                    <p class="text-xs text-slate-300/80">Разверните сервер, следите за графиками нагрузки и логами демона.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 flex flex-col gap-2 shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-transform">
                    <span class="inline-flex h-7 w-7 items-center justify-center rounded-full bg-sky-600 text-[11px] font-semibold text-white">3</span>
                    <h3 class="text-sm font-semibold text-slate-100">Масштабирование и поддержка</h3>
                    <p class="text-xs text-slate-300/80">При росте онлайна увеличивайте ресурсы и подключайте новые локации без миграций.</p>
                </div>
            </div>
        </div>
    </section>
@endsection
