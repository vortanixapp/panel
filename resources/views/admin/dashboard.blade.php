@extends('layouts.app-admin')

@section('page_title', 'Админ‑панель')

@push('styles')
<style>
    .chart-container { position: relative; height: 200px; width: 100%; }
</style>
@endpush

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div>
                <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Админ‑панель</h1>
                <p class="mt-1 text-sm text-slate-300/80">Обзор метрик проекта и управление пользователями и серверами.</p>
            </div>

            <div class="grid gap-3 md:grid-cols-4 text-xs">
                <a
                    href="{{ route('admin.users') }}"
                    class="flex items-center gap-2 rounded-2xl border border-white/10 bg-[#242f3d] px-3 py-2 shadow-sm shadow-black/20 hover:bg-black/10 transition-colors"
                >
                    <span class="inline-flex h-7 w-7 items-center justify-center rounded-lg bg-slate-900 text-white text-[11px]">U</span>
                    <span class="flex flex-col">
                        <span class="font-semibold text-slate-100">Пользователи</span>
                        <span class="text-[11px] text-slate-300/70">Список и управление доступом</span>
                    </span>
                </a>
                <a
                    href="{{ route('admin.locations.index') }}"
                    class="flex items-center gap-2 rounded-2xl border border-white/10 bg-[#242f3d] px-3 py-2 shadow-sm shadow-black/20 hover:bg-black/10 transition-colors"
                >
                    <span class="inline-flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-600 text-white text-[11px]">DC</span>
                    <span class="flex flex-col">
                        <span class="font-semibold text-slate-100">Локации</span>
                        <span class="text-[11px] text-slate-300/70">Дата‑центры и SSH‑доступ</span>
                    </span>
                </a>
                <a
                    href="{{ route('admin.vortanix-daemons.index') }}"
                    class="flex items-center gap-2 rounded-2xl border border-white/10 bg-[#242f3d] px-3 py-2 shadow-sm shadow-black/20 hover:bg-black/10 transition-colors"
                >
                    <span class="inline-flex h-7 w-7 items-center justify-center rounded-lg bg-sky-600 text-white text-[11px]">VD</span>
                    <span class="flex flex-col">
                        <span class="font-semibold text-slate-100">Vortanix Daemons</span>
                        <span class="text-[11px] text-slate-300/70">Статусы и логи демонов</span>
                    </span>
                </a>
                <a
                    href="{{ route('admin.support.index') }}"
                    class="flex items-center gap-2 rounded-2xl border border-white/10 bg-[#242f3d] px-3 py-2 shadow-sm shadow-black/20 hover:bg-black/10 transition-colors"
                >
                    <span class="inline-flex h-7 w-7 items-center justify-center rounded-lg bg-sky-600 text-white text-[11px]">TS</span>
                    <span class="flex flex-col">
                        <span class="font-semibold text-slate-100">Тех. поддержка</span>
                        <span class="text-[11px] text-slate-300/70">
                            Открытых: {{ (int) ($openSupportTicketsCount ?? 0) }}
                        </span>
                    </span>
                </a>
            </div>

            <div class="grid gap-4 md:grid-cols-3 text-sm">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Пользователи</p>
                    <p class="mt-2 text-2xl font-semibold text-slate-100">{{ $totalUsers }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Всего зарегистрированных аккаунтов.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Локации</p>
                    <p class="mt-2 text-2xl font-semibold text-slate-100">{{ $activeLocations }} / {{ $totalLocations }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Активных / Всего локаций.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Нагрузка</p>
                    <p class="mt-2 text-2xl font-semibold text-slate-100">
                        @if($metrics->has('cpu_usage') && $metrics['cpu_usage']->count() > 0)
                            {{ round($metrics['cpu_usage']->avg('value'), 1) }}%
                        @else
                            —
                        @endif
                    </p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Средний CPU по активным локациям.</p>
                </div>
            </div>

            <div class="grid gap-4 md:grid-cols-4 text-sm">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Сервера</p>
                    <p class="mt-2 text-2xl font-semibold text-slate-100">{{ (int) ($serversTotal ?? 0) }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Активных: {{ (int) ($serversActive ?? 0) }} · Истекло: {{ (int) ($serversExpired ?? 0) }} · Suspended: {{ (int) ($serversSuspended ?? 0) }}</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Провижининг</p>
                    <p class="mt-2 text-2xl font-semibold text-slate-100">{{ (int) ($provisioningInstalling ?? 0) }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">В работе · Ошибок: {{ (int) ($provisioningFailed ?? 0) }}</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Новые пользователи</p>
                    <p class="mt-2 text-2xl font-semibold text-slate-100">{{ (int) ($newUsers24h ?? 0) }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">За 24ч · За 7д: {{ (int) ($newUsers7d ?? 0) }}</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Выручка</p>
                    <p class="mt-2 text-2xl font-semibold text-slate-100">{{ number_format((float) ($revenue24h ?? 0), 2) }} ₽</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">7д: {{ number_format((float) ($revenue7d ?? 0), 2) }} ₽ · 30д: {{ number_format((float) ($revenue30d ?? 0), 2) }} ₽</p>
                </div>
            </div>

            <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                <div class="mb-3 flex items-center justify-between gap-2">
                    <h2 class="text-sm font-semibold text-slate-100">Состояние персонала</h2>
                    <span class="text-[11px] text-slate-300/70">Администраторы проекта</span>
                </div>

                @if(isset($staffUsers) && $staffUsers->isNotEmpty())
                    <div class="grid gap-3 md:grid-cols-3 text-[13px]">
                        @foreach($staffUsers as $user)
                            <div class="rounded-xl border border-white/10 bg-black/10 p-3 flex flex-col gap-1">
                                <div class="flex items-center justify-between gap-2">
                                    <div class="text-sm font-semibold text-slate-100 truncate">
                                        {{ $user->name ?: 'Без имени' }}
                                    </div>
                                    @if($user->is_admin)
                                        <span class="inline-flex items-center rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-medium text-emerald-200 ring-1 ring-emerald-500/20">Админ</span>
                                    @else
                                        <span class="inline-flex items-center rounded-full bg-black/10 px-2 py-0.5 text-[10px] font-medium text-slate-200 ring-1 ring-white/10">Пользователь</span>
                                    @endif
                                </div>
                                <p class="text-[11px] text-slate-300/70 truncate">{{ $user->email }}</p>
                                <p class="text-[10px] text-slate-300/60 mt-1">
                                    Зарегистрирован: {{ optional($user->created_at)->diffForHumans() ?? '—' }}
                                </p>
                            </div>
                        @endforeach
                    </div>
                @else
                    <p class="text-[12px] text-slate-300/80">Пока нет данных о пользователях.</p>
                @endif
            </div>

            <div class="grid gap-4 md:grid-cols-2">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="mb-3">
                        <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                            <div>
                                <h2 class="text-sm font-semibold text-slate-100">CPU Нагрузка</h2>
                                <span class="text-[11px] text-slate-300/70">
                                    @if(($selectedLocationId ?? 0) > 0)
                                        {{ (string) (($selectedLocation->name ?? '') ?: ($selectedLocation->code ?? '')) }}
                                    @else
                                        Последние измерения по активным локациям
                                    @endif
                                </span>
                            </div>

                            <form method="GET" action="{{ route('admin.dashboard') }}" class="flex items-center gap-2">
                                <label class="text-[11px] text-slate-300/70" for="location_id">Локация</label>
                                <select id="location_id" name="location_id" class="rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100" onchange="this.form.submit()">
                                    <option value="0" {{ ((int) ($selectedLocationId ?? 0) === 0) ? 'selected' : '' }}>Все</option>
                                    @foreach(($locationsForFilter ?? []) as $loc)
                                        <option value="{{ (int) $loc->id }}" {{ ((int) ($selectedLocationId ?? 0) === (int) $loc->id) ? 'selected' : '' }}>
                                            {{ (string) ($loc->name ?: $loc->code) }}{{ $loc->is_active ? '' : ' (off)' }}
                                        </option>
                                    @endforeach
                                </select>
                            </form>
                        </div>
                    </div>
                    <div class="chart-container">
                        @if(empty($cpuSeries) || count($cpuSeries) === 0)
                            <div class="flex h-full items-center justify-center rounded-xl border border-dashed border-white/10 bg-black/10 text-[12px] text-slate-300/80">
                                Нет данных по CPU
                            </div>
                        @else
                            <canvas id="cpuChart"></canvas>
                        @endif
                    </div>
                </div>

                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="mb-3">
                        <h2 class="text-sm font-semibold text-slate-100">RAM Нагрузка</h2>
                        <span class="text-[11px] text-slate-300/70">
                            @if(($selectedLocationId ?? 0) > 0)
                                {{ (string) (($selectedLocation->name ?? '') ?: ($selectedLocation->code ?? '')) }}
                            @else
                                Последние измерения по активным локациям
                            @endif
                        </span>
                    </div>
                    <div class="chart-container">
                        @if(empty($ramSeries) || count($ramSeries) === 0)
                            <div class="flex h-full items-center justify-center rounded-xl border border-dashed border-white/10 bg-black/10 text-[12px] text-slate-300/80">
                                Нет данных по RAM
                            </div>
                        @else
                            <canvas id="ramChart"></canvas>
                        @endif
                    </div>
                </div>
            </div>
        </div>
    </section>
@endsection

@push('scripts')
<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
<script>
    const cpuData = @json($cpuSeries ?? []);
    const ramData = @json($ramSeries ?? []);

    const chartTheme = {
        tickColor: 'rgba(226, 232, 240, 0.8)',
        gridColor: 'rgba(255, 255, 255, 0.08)',
    };

    const cpuCanvas = document.getElementById('cpuChart');
    if (cpuCanvas) {
    new Chart(cpuCanvas, {
        type: 'line',
        data: {
            datasets: [{
                label: 'CPU %',
                data: cpuData,
                borderColor: 'rgb(59, 130, 246)',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                tension: 0.4,
                pointRadius: 2,
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    type: 'category',
                    title: { display: true, text: 'Время', color: chartTheme.tickColor },
                    ticks: { color: chartTheme.tickColor },
                    grid: { color: chartTheme.gridColor },
                },
                y: {
                    beginAtZero: true,
                    max: 100,
                    title: { display: true, text: '%', color: chartTheme.tickColor },
                    ticks: { color: chartTheme.tickColor },
                    grid: { color: chartTheme.gridColor },
                }
            },
            plugins: {
                legend: { display: false },
                tooltip: {
                    titleColor: 'rgba(255,255,255,0.95)',
                    bodyColor: 'rgba(255,255,255,0.85)',
                    backgroundColor: 'rgba(15, 23, 42, 0.9)',
                    borderColor: 'rgba(255,255,255,0.10)',
                    borderWidth: 1,
                },
            }
        }
    });
    }

    const ramCanvas = document.getElementById('ramChart');
    if (ramCanvas) {
    new Chart(ramCanvas, {
        type: 'line',
        data: {
            datasets: [{
                label: 'RAM %',
                data: ramData,
                borderColor: 'rgb(16, 185, 129)',
                backgroundColor: 'rgba(16, 185, 129, 0.1)',
                tension: 0.4,
                pointRadius: 2,
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    type: 'category',
                    title: { display: true, text: 'Время', color: chartTheme.tickColor },
                    ticks: { color: chartTheme.tickColor },
                    grid: { color: chartTheme.gridColor },
                },
                y: {
                    beginAtZero: true,
                    max: 100,
                    title: { display: true, text: '%', color: chartTheme.tickColor },
                    ticks: { color: chartTheme.tickColor },
                    grid: { color: chartTheme.gridColor },
                }
            },
            plugins: {
                legend: { display: false },
                tooltip: {
                    titleColor: 'rgba(255,255,255,0.95)',
                    bodyColor: 'rgba(255,255,255,0.85)',
                    backgroundColor: 'rgba(15, 23, 42, 0.9)',
                    borderColor: 'rgba(255,255,255,0.10)',
                    borderWidth: 1,
                },
            }
        }
    });
    }
</script>
@endpush
