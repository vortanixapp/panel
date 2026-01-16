@extends('layouts.landing')

@section('content')
    <section class="relative border-b border-white/10">
        <div class="relative mx-auto max-w-6xl px-4 py-10 sm:py-14 md:py-20">
            <div class="grid items-center gap-10 lg:grid-cols-2">
                <div class="min-w-0 space-y-6" data-reveal>
                    <p class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-black/10 px-3 py-1 text-[11px] font-medium text-slate-200">
                        <span class="relative flex h-1.5 w-1.5">
                            <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400/60"></span>
                            <span class="relative inline-flex h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
                        </span>
                        Облачный игровой хостинг
                    </p>
                    <h1 class="text-3xl md:text-4xl lg:text-5xl font-semibold tracking-tight text-slate-100">
                        Инфраструктура для игровых серверов,
                        <span class="bg-gradient-to-r from-sky-500 via-emerald-400 to-indigo-500 bg-clip-text text-transparent">как у облачного провайдера</span>
                    </h1>
                    <p class="max-w-xl text-sm md:text-base text-slate-300/80">
                        Поднимайте сервера для CS2, Minecraft, Rust и других игр на высокопроизводительных NVMe‑узлах с быстрой
                        масштабируемостью, прозрачным биллингом и встроенной защитой от DDoS.
                    </p>
                    <div class="flex flex-col sm:flex-row sm:flex-wrap items-stretch sm:items-center gap-3 sm:gap-4">
                        <button class="vtx-btn-glow inline-flex w-full sm:w-auto items-center justify-center rounded-md bg-slate-900 px-4 py-2 text-xs md:text-sm font-semibold text-white shadow-sm hover:bg-slate-800 transition">
                            Создать сервер
                        </button>
                        <button class="vtx-btn-glow inline-flex w-full sm:w-auto items-center justify-center gap-2 rounded-md border border-white/10 bg-black/10 px-4 py-2 text-xs md:text-sm font-medium text-slate-200 hover:bg-black/15 hover:text-white transition" onclick="document.getElementById('pricing')?.scrollIntoView({ behavior: 'smooth' })">
                            Смотреть тарифы
                            <span class="text-slate-400">→</span>
                        </button>
                    </div>
                    <dl class="grid max-w-md grid-cols-1 sm:grid-cols-3 gap-4 pt-4 text-xs border-t border-white/10">
                        <div>
                            <dt class="text-[11px] uppercase tracking-wide text-slate-300/70">Пинг</dt>
                            <dd class="mt-1 text-sm font-semibold text-slate-100">от 5 мс</dd>
                        </div>
                        <div>
                            <dt class="text-[11px] uppercase tracking-wide text-slate-300/70">Регионы</dt>
                            <dd class="mt-1 text-sm font-semibold text-slate-100">Европа, РФ, Азия</dd>
                        </div>
                        <div>
                            <dt class="text-[11px] uppercase tracking-wide text-slate-300/70">SLA</dt>
                            <dd class="mt-1 text-sm font-semibold text-slate-100">до 99.95%</dd>
                        </div>
                    </dl>

                    <div class="pt-6" data-reveal data-delay="220">
                        <div class="vtx-marquee rounded-2xl border border-white/10 bg-black/10 px-3 py-2">
                            <div class="vtx-marquee__track">
                                <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-[#242f3d] px-3 py-1 text-[11px] text-slate-200">DDoS‑защита</div>
                                <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-[#242f3d] px-3 py-1 text-[11px] text-slate-200">NVMe</div>
                                <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-[#242f3d] px-3 py-1 text-[11px] text-slate-200">Панель управления</div>
                                <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-[#242f3d] px-3 py-1 text-[11px] text-slate-200">Авто‑бэкапы</div>
                                <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-[#242f3d] px-3 py-1 text-[11px] text-slate-200">Мониторинг</div>
                                <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-[#242f3d] px-3 py-1 text-[11px] text-slate-200">Поддержка 24/7</div>
                            </div>
                            <div class="vtx-marquee__track" aria-hidden="true">
                                <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-[#242f3d] px-3 py-1 text-[11px] text-slate-200">DDoS‑защита</div>
                                <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-[#242f3d] px-3 py-1 text-[11px] text-slate-200">NVMe</div>
                                <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-[#242f3d] px-3 py-1 text-[11px] text-slate-200">Панель управления</div>
                                <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-[#242f3d] px-3 py-1 text-[11px] text-slate-200">Авто‑бэкапы</div>
                                <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-[#242f3d] px-3 py-1 text-[11px] text-slate-200">Мониторинг</div>
                                <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-[#242f3d] px-3 py-1 text-[11px] text-slate-200">Поддержка 24/7</div>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="min-w-0 lg:pl-6">
                    <div class="relative" data-reveal data-delay="120">
                        <div class="pointer-events-none absolute inset-0 sm:-inset-4 rounded-3xl bg-[radial-gradient(circle_at_top,_rgba(56,189,248,0.28),_transparent_60%),_radial-gradient(circle_at_bottom_right,_rgba(129,140,248,0.25),_transparent_55%)] opacity-70"></div>
                        <div class="relative rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-xl shadow-black/20">
                            <div class="mb-4 flex items-center justify-between gap-2">
                                <div>
                                    <p class="text-[11px] uppercase tracking-wide text-slate-300/70">Конфигурация сервера</p>
                                    <p class="text-sm font-semibold text-slate-100">Подойдёт для CS2 / Minecraft</p>
                                </div>
                                <span class="inline-flex items-center rounded-full bg-emerald-500/10 px-2 py-1 text-[10px] font-medium text-emerald-300 ring-1 ring-emerald-500/20">
                                    DDoS‑защита включена
                                </span>
                            </div>

                            @livewire('pricing-calculator')
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </section>

    <section id="advantages" class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14">
            <div class="mb-8 flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div data-reveal>
                    <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Быстрый и стабильный хостинг</h2>
                    <p class="mt-1 max-w-2xl text-sm text-slate-300/80">Короткий путь от идеи до запуска: быстрые узлы, DDoS‑защита и панель управления в одном месте.</p>
                </div>
            </div>
            <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4 text-sm">
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm" data-reveal data-delay="0">
                    <p class="mb-2 text-xs font-semibold text-slate-300/70">Производительность</p>
                    <h3 class="mb-2 text-sm font-semibold text-slate-100">Высокая скорость</h3>
                    <p class="text-xs text-slate-300/80">NVMe‑хранилище, современные CPU и стабильные частоты для низкого пинга и высокого FPS.</p>
                </div>
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm" data-reveal data-delay="80">
                    <p class="mb-2 text-xs font-semibold text-slate-300/70">Поддержка</p>
                    <h3 class="mb-2 text-sm font-semibold text-slate-100">Техподдержка 24/7</h3>
                    <p class="text-xs text-slate-300/80">Быстро реагируем на инциденты и помогаем с настройками, обновлениями и миграциями.</p>
                </div>
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm" data-reveal data-delay="160">
                    <p class="mb-2 text-xs font-semibold text-slate-300/70">Безопасность</p>
                    <h3 class="mb-2 text-sm font-semibold text-slate-100">Защита от DDoS</h3>
                    <p class="text-xs text-slate-300/80">Фильтрация на периметре и профилирование протоколов для типовых игровых атак.</p>
                </div>
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm" data-reveal data-delay="240">
                    <p class="mb-2 text-xs font-semibold text-slate-300/70">Управление</p>
                    <h3 class="mb-2 text-sm font-semibold text-slate-100">Своя панель</h3>
                    <p class="text-xs text-slate-300/80">Логи, консоль, файлы, мониторинг и доступы — всё в едином интерфейсе.</p>
                </div>
            </div>
        </div>
    </section>

    <section id="games" class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14">
            <div class="mb-6 flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div data-reveal>
                    <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Игровые сервера под любые
                        проекты</h2>
                    <p class="mt-1 max-w-2xl text-sm text-slate-300/80">
                        Готовые пресеты для популярных игр и гибкая конфигурация ресурсов для собственных мод‑проектов и
                        сообществ.
                    </p>
                </div>
            </div>
            <div class="grid gap-4 md:grid-cols-3 text-sm">
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="0">
                    <div class="vtx-card-media mb-3 overflow-hidden rounded-xl border border-white/10 bg-black/10">
                        <div class="relative aspect-[16/10]">
                            <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top,_rgba(34,197,94,0.12),_transparent_55%),_radial-gradient(circle_at_bottom_right,_rgba(56,189,248,0.12),_transparent_55%)]"></div>
                            <div class="absolute inset-0 flex items-center justify-center">
                                <div class="rounded-lg border border-white/10 bg-[#17212b]/60 px-2.5 py-1.5 text-[11px] text-slate-200">Minecraft cover</div>
                            </div>
                        </div>
                    </div>
                    <p class="text-xs font-semibold text-slate-300/70">Minecraft</p>
                    <p class="mt-1 text-sm font-semibold text-slate-100">Выживание, мод‑сборки, мини‑игры</p>
                    <p class="mt-2 text-xs text-slate-300/80">Оптимизированные пресеты под Paper, Purpur и modded‑ядра, быстрая
                        загрузка миров и резервные копии.</p>
                </div>
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="90">
                    <div class="vtx-card-media mb-3 overflow-hidden rounded-xl border border-white/10 bg-black/10">
                        <div class="relative aspect-[16/10]">
                            <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top,_rgba(56,189,248,0.14),_transparent_55%),_radial-gradient(circle_at_bottom_right,_rgba(129,140,248,0.12),_transparent_55%)]"></div>
                            <div class="absolute inset-0 flex items-center justify-center">
                                <div class="rounded-lg border border-white/10 bg-[#17212b]/60 px-2.5 py-1.5 text-[11px] text-slate-200">CS2 cover</div>
                            </div>
                        </div>
                    </div>
                    <p class="text-xs font-semibold text-slate-300/70">CS2 / CS:GO</p>
                    <p class="mt-1 text-sm font-semibold text-slate-100">Паблики, миксы, FACEIT‑серверы</p>
                    <p class="mt-2 text-xs text-slate-300/80">Высокая частота тиков, стабильный FPS и низкий пинг для соревновательных
                        серверов.</p>
                </div>
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="180">
                    <div class="vtx-card-media mb-3 overflow-hidden rounded-xl border border-white/10 bg-black/10">
                        <div class="relative aspect-[16/10]">
                            <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top,_rgba(245,158,11,0.10),_transparent_55%),_radial-gradient(circle_at_bottom_right,_rgba(56,189,248,0.10),_transparent_55%)]"></div>
                            <div class="absolute inset-0 flex items-center justify-center">
                                <div class="rounded-lg border border-white/10 bg-[#17212b]/60 px-2.5 py-1.5 text-[11px] text-slate-200">Rust cover</div>
                            </div>
                        </div>
                    </div>
                    <p class="text-xs font-semibold text-slate-300/70">Rust и другие</p>
                    <p class="mt-1 text-sm font-semibold text-slate-100">Вайп‑серверы и крупные проекты</p>
                    <p class="mt-2 text-xs text-slate-300/80">Горизонтальное масштабирование, отдельные узлы и приватная сеть для
                        сложной инфраструктуры.</p>
                </div>
            </div>
        </div>
    </section>

    <section id="panel" class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14">
            <div class="grid gap-8 lg:grid-cols-2 lg:items-center">
                <div data-reveal>
                    <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Удобная панель управления</h2>
                    <p class="mt-1 max-w-2xl text-sm text-slate-300/80">Управляйте сервером с компьютера или телефона: консоль, файлы, логи, бэкапы, пользователи и доступы — всё под рукой.</p>

                    <div class="mt-5 grid gap-3 sm:grid-cols-2 text-sm">
                        <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="0">
                            <h3 class="mb-1 text-sm font-semibold text-slate-100">Удобный интерфейс</h3>
                            <p class="text-xs text-slate-300/80">Основные действия — в один клик, понятная навигация по вкладкам.</p>
                        </div>
                        <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="80">
                            <h3 class="mb-1 text-sm font-semibold text-slate-100">Бэкапы</h3>
                            <p class="text-xs text-slate-300/80">Резервные копии и восстановление без сложных процедур.</p>
                        </div>
                        <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="160">
                            <h3 class="mb-1 text-sm font-semibold text-slate-100">Доступы</h3>
                            <p class="text-xs text-slate-300/80">Добавляйте сотрудников и настраивайте права управления.</p>
                        </div>
                        <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="240">
                            <h3 class="mb-1 text-sm font-semibold text-slate-100">Поддержка 24/7</h3>
                            <p class="text-xs text-slate-300/80">Помогаем с настройкой, обновлениями и переносами.</p>
                        </div>
                    </div>
                </div>

                <div class="min-w-0 relative lg:pl-6" data-reveal data-delay="120">
                    <div class="pointer-events-none absolute inset-0 sm:-inset-4 rounded-3xl bg-[radial-gradient(circle_at_top,_rgba(56,189,248,0.22),_transparent_60%),_radial-gradient(circle_at_bottom_right,_rgba(129,140,248,0.2),_transparent_55%)] opacity-70"></div>
                    <div class="relative rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-xl shadow-black/20">
                        <div class="vtx-card-media mb-4 overflow-hidden rounded-2xl border border-white/10 bg-black/10">
                            <div class="relative aspect-[16/10]">
                                <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top,_rgba(56,189,248,0.16),_transparent_55%),_radial-gradient(circle_at_bottom_right,_rgba(129,140,248,0.12),_transparent_55%)]"></div>
                                <div class="absolute inset-0 flex items-center justify-center">
                                    <div class="rounded-xl border border-white/10 bg-[#17212b]/60 px-3 py-2 text-[11px] text-slate-200">Скрин панели / графики (заменишь позже)</div>
                                </div>
                            </div>
                        </div>
                        <p class="text-[11px] uppercase tracking-wide text-slate-300/70">Панель управления</p>
                        <p class="mt-1 text-sm font-semibold text-slate-100">Полный контроль над сервером</p>
                        <div class="mt-4 grid gap-3 text-xs text-slate-300/80">
                            <div class="rounded-xl border border-white/10 bg-black/10 p-3">Консоль и команды • Логи • Файлы</div>
                            <div class="rounded-xl border border-white/10 bg-black/10 p-3">Бэкапы • Cron • Firewall</div>
                            <div class="rounded-xl border border-white/10 bg-black/10 p-3">Мониторинг • Доступы друзей</div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </section>

    <section id="features" class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14">
            <div class="mb-6 flex items-end justify-between gap-4">
                <div data-reveal>
                    <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Облачная архитектура под игры</h2>
                    <p class="mt-1 max-w-2xl text-sm text-slate-300/80">
                        Используем подходы из классического облака: распределённые кластеры, снапшоты и наблюдаемость, адаптированные
                        под игровой трафик.
                    </p>
                </div>
            </div>
            <div class="grid gap-5 md:grid-cols-3 text-sm">
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm" data-reveal data-delay="0">
                    <h3 class="mb-2 text-sm font-semibold text-slate-100">NVMe‑хранилище и быстрые CPU</h3>
                    <p class="text-xs text-slate-300/80">Современные процессоры и NVMe обеспечивают стабильный TPS и быструю загрузку
                        карт даже при пиковых онлайнах.</p>
                </div>
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm" data-reveal data-delay="110">
                    <h3 class="mb-2 text-sm font-semibold text-slate-100">Умная защита от DDoS</h3>
                    <p class="text-xs text-slate-300/80">Фильтрация трафика на периметре, профилирование игровых протоколов и автоматическое
                        переключение маршрутов.</p>
                </div>
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm" data-reveal data-delay="220">
                    <h3 class="mb-2 text-sm font-semibold text-slate-100">Панель управления как у облака</h3>
                    <p class="text-xs text-slate-300/80">Графики нагрузки, логи, снапшоты и быстрое масштабирование — всё в одном личном
                        кабинете.</p>
                </div>
            </div>
        </div>
    </section>

    <section id="pricing" class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14">
            <div class="mb-8 flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div data-reveal>
                    <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Тарифы для разных масштабов</h2>
                    <p class="mt-1 max-w-2xl text-sm text-slate-300/80">
                        Начните с небольшого сервера для друзей и масштабируйтесь до крупных игровых проектов без миграций и
                        простоев.
                    </p>
                </div>
            </div>
            <div class="grid gap-6 md:grid-cols-3 text-sm">
                <div class="vtx-card relative rounded-2xl border border-white/10 bg-[#242f3d] p-5" data-reveal data-delay="0">
                    <p class="mb-1 text-xs font-semibold uppercase tracking-wide text-slate-300/70">Старт</p>
                    <p class="mb-4 text-2xl font-semibold text-slate-100">от 190 ₽/мес</p>
                    <ul class="mb-5 space-y-1.5 text-xs text-slate-300/80">
                        <li>До 20 слотов</li>
                        <li>1 vCPU • 2 ГБ RAM</li>
                        <li>NVMe 20 ГБ</li>
                        <li>DDoS‑защита уровня L3/L4</li>
                    </ul>
                    <button class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-medium text-slate-200 hover:bg-black/15 hover:text-white">
                        Выбрать тариф
                    </button>
                </div>
                <div class="vtx-card relative rounded-2xl border border-white/10 bg-[#242f3d] p-5 shadow-[0_18px_45px_rgba(0,0,0,0.35)]" data-reveal data-delay="110">
                    <div class="absolute -top-3 right-4 inline-flex items-center rounded-full bg-sky-600 px-2 py-0.5 text-[10px] font-semibold text-white shadow-sm">
                        Популярный выбор
                    </div>
                    <p class="mb-1 text-xs font-semibold uppercase tracking-wide text-sky-300">Про</p>
                    <p class="mb-4 text-2xl font-semibold text-slate-100">от 420 ₽/мес</p>
                    <ul class="mb-5 space-y-1.5 text-xs text-slate-200">
                        <li>До 64 слотов</li>
                        <li>2 vCPU • 4 ГБ RAM</li>
                        <li>NVMe 40 ГБ</li>
                        <li>Авто‑бэкапы и мониторинг</li>
                    </ul>
                    <button class="w-full rounded-md bg-sky-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-sky-500">
                        Развернуть за 40 секунд
                    </button>
                </div>
                <div class="vtx-card relative rounded-2xl border border-white/10 bg-[#242f3d] p-5" data-reveal data-delay="220">
                    <p class="mb-1 text-xs font-semibold uppercase tracking-wide text-slate-300/70">Кластер</p>
                    <p class="mb-4 text-2xl font-semibold text-slate-100">индивидуально</p>
                    <ul class="mb-5 space-y-1.5 text-xs text-slate-300/80">
                        <li>Неограниченное число серверов</li>
                        <li>Выделенные узлы и приватная сеть</li>
                        <li>Персональный менеджер</li>
                        <li>SLA от 99.95%</li>
                    </ul>
                    <button class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-medium text-slate-200 hover:bg-black/15 hover:text-white">
                        Запросить предложение
                    </button>
                </div>
            </div>
        </div>
    </section>

    <section id="facts" class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14">
            <div class="mb-6" data-reveal>
                <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Пару фактов о нашем хостинге</h2>
                <p class="mt-1 max-w-2xl text-sm text-slate-300/80">Коротко о том, что вы получаете в Vortanix GameCloud.</p>
            </div>
            <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4 text-sm">
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-5" data-reveal data-delay="0">
                    <p class="text-3xl font-semibold text-slate-100" data-count="600" data-suffix="+">0</p>
                    <p class="mt-1 text-xs text-slate-300/80">пользователей в панели</p>
                </div>
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-5" data-reveal data-delay="90">
                    <p class="text-3xl font-semibold text-slate-100" data-count="100" data-suffix="%">0</p>
                    <p class="mt-1 text-xs text-slate-300/80">целевой SLA</p>
                </div>
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-5" data-reveal data-delay="180">
                    <p class="text-3xl font-semibold text-slate-100" data-count="200" data-suffix="+">0</p>
                    <p class="mt-1 text-xs text-slate-300/80">серверов запущено</p>
                </div>
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-5" data-reveal data-delay="270">
                    <p class="text-3xl font-semibold text-slate-100" data-count="49" data-suffix="+">0</p>
                    <p class="mt-1 text-xs text-slate-300/80">оценка сервиса</p>
                </div>
            </div>
        </div>
    </section>

    <section id="reviews" class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14">
            <div class="mb-6 flex flex-col gap-2 md:flex-row md:items-end md:justify-between" data-reveal>
                <div>
                    <h2 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Отзывы</h2>
                    <p class="mt-1 max-w-2xl text-sm text-slate-300/80">Реальные кейсы сообществ: стабильность, скорость и удобное управление.</p>
                </div>
            </div>
            <div class="grid gap-4 md:grid-cols-3 text-sm">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="0">
                    <p class="text-xs text-slate-300/80">«Стабильный FPS и низкий пинг. Переехали без простоя — поддержка помогла со всем.»</p>
                    <p class="mt-3 text-xs font-semibold text-slate-100">Администратор проекта</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="90">
                    <p class="text-xs text-slate-300/80">«Панель управления реально удобная: бэкапы, файлы, логи — не нужно ковыряться в консоли.»</p>
                    <p class="mt-3 text-xs font-semibold text-slate-100">Владелец сервера</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="180">
                    <p class="text-xs text-slate-300/80">«Защита от DDoS держит, онлайн не проседает. Для наших вайпов это критично.»</p>
                    <p class="mt-3 text-xs font-semibold text-slate-100">Комьюнити менеджер</p>
                </div>
            </div>
        </div>
    </section>

    <section id="trial" class="border-b border-white/10 bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14">
            <div class="relative overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] p-6 md:p-10 shadow-xl shadow-black/20" data-reveal>
                <div class="pointer-events-none absolute -top-24 right-0 h-72 w-72 rounded-full bg-sky-500/10 blur-3xl"></div>
                <div class="pointer-events-none absolute -bottom-28 left-0 h-72 w-72 rounded-full bg-indigo-500/10 blur-3xl"></div>
                <div class="relative grid gap-6 lg:grid-cols-2 lg:items-center">
                    <div>
                        <h2 class="text-2xl md:text-3xl font-semibold tracking-tight text-slate-100">Попробуйте наш хостинг в деле</h2>
                        <p class="mt-2 text-sm text-slate-300/80">Запустите сервер, посмотрите панель и оцените производительность — без лишней рутины.</p>
                    </div>
                    <div class="flex flex-col sm:flex-row gap-3 sm:justify-end">
                        <a href="{{ route('register') }}" class="inline-flex items-center justify-center rounded-md bg-sky-600 px-4 py-2 text-sm font-semibold text-white hover:bg-sky-500">
                            Создать аккаунт
                        </a>
                        <a href="{{ route('login') }}" class="inline-flex items-center justify-center rounded-md border border-white/10 bg-black/10 px-4 py-2 text-sm font-medium text-slate-200 hover:bg-black/15 hover:text-white">
                            Войти в панель
                        </a>
                    </div>
                </div>
            </div>
        </div>
    </section>

    <section id="faq" class="bg-[#17212b]">
        <div class="mx-auto max-w-6xl px-4 py-10 md:py-14 text-sm">
            <div class="mb-6 max-w-2xl">
                <h2 class="text-xl font-semibold tracking-tight text-slate-100">Частые вопросы</h2>
                <p class="mt-1 text-sm text-slate-300/80">Кратко о запуске и управлении игровыми серверами в Vortanix GameCloud.</p>
            </div>
            <div class="grid gap-4 md:grid-cols-2">
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="0">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Сколько времени занимает развёртывание сервера?</h3>
                    <p class="text-xs text-slate-300/80">Обычно от 30 до 40 секунд после выбора тарифа. Конфигурация и установка нужной
                        игры происходят автоматически.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="90">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Можно ли позже изменить игру или конфигурацию?</h3>
                    <p class="text-xs text-slate-300/80">Да, можно сменить игру, увеличить число слотов или ресурсы сервера без потери
                        данных, используя снапшоты и миграции.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="180">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Какие есть способы оплаты?</h3>
                    <p class="text-xs text-slate-300/80">Поддерживаем банковские карты, электронные кошельки и пополнение баланса в
                        личном кабинете. Списание по модели pay‑as‑you‑go.</p>
                </div>
                <div class="vtx-card rounded-2xl border border-white/10 bg-[#242f3d] p-4" data-reveal data-delay="270">
                    <h3 class="mb-1 text-sm font-semibold text-slate-100">Поможете ли вы перенести существующий сервер?</h3>
                    <p class="text-xs text-slate-300/80">Да, команда миграции бесплатно перенесёт ваш сервер от другого провайдера с
                        минимальным простоем.</p>
                </div>
            </div>
        </div>
    </section>
@endsection
