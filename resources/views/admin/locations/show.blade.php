@extends('layouts.app-admin')

@section('page_title', 'Локация: ' . $location->name)

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">
                        Локация: {{ $location->name }}
                    </h1>
                    <p class="mt-1 text-sm text-slate-300/80">
                        Детальная информация о дата‑центре и доступе по SSH.
                    </p>
                </div>
                <div class="flex flex-wrap items-center gap-3 text-xs">
                    <a
                        href="{{ route('admin.locations.index') }}"
                        class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                    >
                        Назад к списку
                    </a>
                    <a
                        href="{{ route('admin.locations.edit', $location) }}"
                        class="inline-flex items-center rounded-md bg-black/10 px-4 py-2 text-[11px] font-semibold text-slate-100 ring-1 ring-white/10 hover:bg-black/15"
                    >
                        Редактировать
                    </a>
                    @if($location->ssh_host)
                        <a
                            href="{{ route('admin.locations.setup', $location) }}"
                            class="inline-flex items-center rounded-md bg-blue-600 px-4 py-2 text-[11px] font-semibold text-white shadow-sm hover:bg-blue-500"
                        >
                            Настроить
                        </a>
                    @endif
                    @if($location->ssh_host)
                        <form method="POST" action="{{ route('admin.locations.pull-daemon', $location) }}">
                            @csrf
                            <button
                                type="submit"
                                class="inline-flex items-center rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-1.5 text-[11px] font-medium text-emerald-200 hover:bg-emerald-500/15"
                            >
                                Обновить из демона
                            </button>
                        </form>
                    @endif
                </div>
            </div>

            <div class="grid gap-4 md:grid-cols-3 text-sm">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Статус</p>
                    <p class="mt-2 text-sm font-semibold {{ $location->is_active ? 'text-emerald-300' : 'text-slate-300/80' }}">
                        {{ $location->is_active ? 'Активна' : 'Неактивна' }}
                    </p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Управляет доступностью локации для новых серверов.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Расположение</p>
                    <p class="mt-2 text-sm font-semibold text-slate-100">
                        {{ $location->city ?: '—' }}@if($location->city && $location->country), @endif{{ $location->country ?: '' }}
                    </p>
                    @if($location->region)
                        <p class="mt-1 text-[11px] text-slate-300/70">Регион: {{ $location->region }}</p>
                    @endif
                    <p class="mt-1 text-[11px] text-slate-300/70">Код: <code>{{ $location->code }}</code></p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">SSH‑хост</p>
                    <p class="mt-2 text-sm font-semibold text-slate-100">{{ $location->ssh_host ?: 'не задан' }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Порт: {{ $location->ssh_port ?? 22 }}</p>
                </div>
            </div>

            <div class="grid gap-4 md:grid-cols-3 text-sm">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">IP адрес (для игроков)</p>
                    <p class="mt-2 text-sm font-semibold text-slate-100">{{ $location->ip_address ?: 'не задан' }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Этот IP будет использоваться на странице сервера для подключения игроков.</p>
                </div>

                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">IP пул</p>
                    @if(is_array($location->ip_pool) && count($location->ip_pool) > 0)
                        <p class="mt-2 text-sm font-semibold text-slate-100">{{ implode(', ', $location->ip_pool) }}</p>
                    @else
                        <p class="mt-2 text-sm font-semibold text-slate-100">—</p>
                    @endif
                    <p class="mt-1 text-[11px] text-slate-300/70">Список IP, доступных для выбора при аренде (если задан).</p>
                </div>

                @php
                    $mysqlRootPass = '';
                    try {
                        if (! empty($location->mysql_root_password)) {
                            $mysqlRootPass = \Illuminate\Support\Facades\Crypt::decryptString($location->mysql_root_password);
                        }
                    } catch (\Throwable $e) {
                        $mysqlRootPass = '';
                    }
                @endphp
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">MySQL доступ (локация)</p>
                    <p class="mt-2 text-sm font-semibold text-slate-100">{{ $location->mysql_host ?: ($location->ip_address ?: 'не задан') }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Порт: {{ $location->mysql_port ?: 3306 }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Пользователь: {{ $location->mysql_root_username ?: '—' }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Пароль: {{ $mysqlRootPass !== '' ? $mysqlRootPass : '—' }}</p>

                    @php
                        $pmaHost = (string) (($location->ip_address ?: '') ?: ($location->ssh_host ?: ''));
                        $pmaPort = (int) ($location->phpmyadmin_port ?: 0);
                        $pmaUrl = ($pmaHost !== '' && $pmaPort > 0) ? ('http://' . $pmaHost . ':' . $pmaPort) : '';
                    @endphp
                    <p class="mt-2 text-[11px] text-slate-300/70">
                        phpMyAdmin:
                        @if($pmaUrl !== '')
                            <a href="{{ $pmaUrl }}" target="_blank" class="font-medium text-slate-100 underline hover:text-white">{{ $pmaUrl }}</a>
                        @else
                            —
                        @endif
                    </p>
                </div>
            </div>

            <div class="grid gap-4 md:grid-cols-3 text-sm">
                <div class="md:col-span-2 rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <h2 class="text-sm font-semibold text-slate-100">Информация о сервере</h2>
                    <dl class="mt-3 grid gap-y-1 text-[13px] text-slate-200">
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">ОС</dt>
                            <dd class="flex-1">{{ $serverMetrics['os_info']->text_value ?? '—' }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Процессор</dt>
                            <dd class="flex-1">{{ $serverMetrics['cpu_model']->text_value ?? '—' }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Память (total)</dt>
                            <dd class="flex-1">{{ $serverMetrics['ram_total']->text_value ?? '—' }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Диск (total)</dt>
                            <dd class="flex-1">{{ $serverMetrics['disk_total']->text_value ?? '—' }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Диск (использовано)</dt>
                            <dd class="flex-1">{{ $serverMetrics['disk_used']->text_value ?? '—' }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Диск (доступно)</dt>
                            <dd class="flex-1">{{ $serverMetrics['disk_available']->text_value ?? '—' }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Время работы</dt>
                            <dd class="flex-1">{{ $serverMetrics['uptime']->text_value ?? '—' }}</dd>
                        </div>
                    </dl>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <h2 class="text-sm font-semibold text-slate-100">SSH‑доступ</h2>
                    <div class="mt-3 space-y-1 text-[13px] text-slate-200">
                        <p><span class="text-slate-300/70">Хост:</span> {{ $location->ssh_host ?: '—' }}</p>
                        <p><span class="text-slate-300/70">Пользователь:</span> {{ $location->ssh_user ?: '—' }}</p>
                        <p><span class="text-slate-300/70">Порт:</span> {{ $location->ssh_port ?? 22 }}</p>
                        <p>
                            <span class="text-slate-300/70">Пароль:</span>
                            @if($location->ssh_password)
                                <span class="align-middle">••••••••</span>
                            @else
                                <span class="text-slate-300/70">не задан</span>
                            @endif
                        </p>
                    </div>
                    @if($location->ssh_host && $location->ssh_user)
                        <div class="mt-3">
                            <p class="text-[11px] uppercase tracking-wide text-slate-300/70">Команда подключения</p>
                            <pre class="mt-1 rounded-md border border-white/10 bg-black/20 px-3 py-2 text-[11px] text-slate-100 overflow-x-auto">ssh {{ $location->ssh_user }}@{{ $location->ssh_host }} -p {{ $location->ssh_port ?? 22 }}</pre>
                        </div>
                    @endif
                </div>
            </div>

            @php
                $serviceStatuses = $serviceStatuses ?? [];
            @endphp

            <div class="grid gap-4 md:grid-cols-4 text-sm">
                @foreach ([
                    'docker' => 'Docker',
                    'mysql' => 'MySQL',
                    'vsftpd' => 'FTP',
                    'vortanix-daemon' => 'Vortanix Daemon',
                ] as $unit => $label)
                    @php
                        $svc = $serviceStatuses[$unit] ?? ['state' => 'unknown', 'error' => null, 'label' => $label];
                        $state = $svc['state'];
                        $stateLabel = match ($state) {
                            'active' => 'Запущен',
                            'inactive', 'failed' => 'Остановлен',
                            'unconfigured' => 'SSH не настроен',
                            'ssh_failed' => 'Ошибка SSH',
                            'error' => 'Ошибка',
                            default => 'Неизвестно',
                        };
                        $badgeClasses = match ($state) {
                            'active' => 'bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20',
                            'inactive', 'failed' => 'bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20',
                            'unconfigured', 'ssh_failed', 'error' => 'bg-amber-500/10 text-amber-200 ring-1 ring-amber-500/20',
                            default => 'bg-black/10 text-slate-200 ring-1 ring-white/10',
                        };
                        $dotClasses = match ($state) {
                            'active' => 'bg-emerald-400',
                            'inactive', 'failed' => 'bg-rose-400',
                            'unconfigured', 'ssh_failed', 'error' => 'bg-amber-400',
                            default => 'bg-slate-400',
                        };
                    @endphp
                    <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20 flex flex-col gap-2">
                        <p class="text-xs font-semibold text-slate-300/70">{{ $label }}</p>
                        <span class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] {{ $badgeClasses }}">
                            <span class="inline-block h-1.5 w-1.5 rounded-full {{ $dotClasses }}"></span>
                            {{ $stateLabel }}
                        </span>
                        @if(!empty($svc['error']))
                            <p class="text-[10px] text-amber-200">
                                {{ \Illuminate\Support\Str::limit($svc['error'], 80) }}
                            </p>
                        @endif
                    </div>
                @endforeach
            </div>

            <div class="grid gap-4 md:grid-cols-3 text-sm">
                <div class="md:col-span-3 rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <h2 class="text-sm font-semibold text-slate-100">Статистика и графики</h2>
                    @php
                        $cpuMetrics = ($metrics['cpu_usage'] ?? collect())->sortBy('measured_at')->values();
                        $ramMetrics = ($metrics['ram_usage'] ?? collect())->sortBy('measured_at')->values();
                        $hasData = !$cpuMetrics->isEmpty() || !$ramMetrics->isEmpty();
                    @endphp
                    @if (!$hasData)
                        <div class="mt-3 h-40 md:h-40 rounded-xl border border-dashed border-white/10 bg-black/10 text-[11px] text-slate-300/80 flex items-center justify-center">
                            <span>Для этой локации пока нет загруженных метрик. Отправьте данные в /api/metrics, чтобы увидеть график.</span>
                        </div>
                    @else
                        <div class="mt-3">
                            <canvas id="location-metrics-chart" class="w-full h-64 md:h-40"></canvas>
                        </div>
                        <script>
                            document.addEventListener('DOMContentLoaded', () => {
                                const rawCpu = @json(
                                    $cpuMetrics->map(fn($m) => [
                                        't' => optional($m->measured_at)->toIso8601String(),
                                        'v' => $m->value,
                                    ])
                                );
                                const rawRam = @json(
                                    $ramMetrics->map(fn($m) => [
                                        't' => optional($m->measured_at)->toIso8601String(),
                                        'v' => $m->value,
                                    ])
                                );

                                if (!rawCpu.length && !rawRam.length) return;

                                // Объединяем временные метки для оси X
                                const allTimestamps = [...rawCpu.map(p => p.t), ...rawRam.map(p => p.t)].filter(Boolean).sort();
                                const uniqueTimestamps = [...new Set(allTimestamps)];

                                const labels = uniqueTimestamps.map(t => {
                                    const d = new Date(t);
                                    return d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
                                });

                                const ctx = document.getElementById('location-metrics-chart').getContext('2d');
                                const isMobile = window.innerWidth < 768; // md breakpoint

                                const chartTheme = {
                                    tickColor: 'rgba(226, 232, 240, 0.8)',
                                    gridColor: 'rgba(255, 255, 255, 0.08)',
                                };

                                new Chart(ctx, {
                                    type: 'line',
                                    data: {
                                        labels,
                                        datasets: [
                                            {
                                                label: 'CPU usage, %',
                                                data: uniqueTimestamps.map(t => {
                                                    const point = rawCpu.find(p => p.t === t);
                                                    return point ? point.v : null;
                                                }),
                                                borderColor: 'rgb(56, 189, 248)',
                                                backgroundColor: 'rgba(56, 189, 248, 0.15)',
                                                tension: 0.3,
                                                fill: true,
                                                pointRadius: 0,
                                                spanGaps: true,
                                            },
                                            {
                                                label: 'RAM usage, %',
                                                data: uniqueTimestamps.map(t => {
                                                    const point = rawRam.find(p => p.t === t);
                                                    return point ? point.v : null;
                                                }),
                                                borderColor: 'rgb(34, 197, 94)',
                                                backgroundColor: 'rgba(34, 197, 94, 0.15)',
                                                tension: 0.3,
                                                fill: true,
                                                pointRadius: 0,
                                                spanGaps: true,
                                            }
                                        ],
                                    },
                                    options: {
                                        responsive: true,
                                        maintainAspectRatio: false,
                                        plugins: {
                                            legend: {
                                                display: !isMobile,
                                                position: 'top',
                                                labels: { color: chartTheme.tickColor },
                                            }, // Скрываем легенду на мобильных
                                            tooltip: {
                                                titleColor: 'rgba(255,255,255,0.95)',
                                                bodyColor: 'rgba(255,255,255,0.85)',
                                                backgroundColor: 'rgba(15, 23, 42, 0.9)',
                                                borderColor: 'rgba(255,255,255,0.10)',
                                                borderWidth: 1,
                                            },
                                        },
                                        scales: {
                                            x: {
                                                ticks: { color: chartTheme.tickColor, maxTicksLimit: isMobile ? 4 : 6, font: { size: isMobile ? 9 : 10 } },
                                                grid: { display: false },
                                            },
                                            y: {
                                                ticks: { color: chartTheme.tickColor, font: { size: isMobile ? 9 : 10 } },
                                                grid: { color: chartTheme.gridColor },
                                            },
                                        },
                                    },
                                });
                            });
                        </script>
                    @endif
                </div>
                </div>
            </div>
        </div>
    </section>
@endsection
