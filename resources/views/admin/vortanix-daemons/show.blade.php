@extends('layouts.app-admin')

@section('page_title', 'Демон: ' . $location->name)

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">
                        Демон: {{ $location->name }} ({{ $location->code }})
                    </h1>
                    <p class="mt-1 text-sm text-slate-300/80">
                        Подробная информация о Vortanix Daemon, нагрузке и логах для этой локации.
                    </p>
                </div>
                <div class="flex flex-wrap items-center gap-3 text-xs">
                    <a
                        href="{{ route('admin.vortanix-daemons.index') }}"
                        class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                    >
                        ← Ко всем демонам
                    </a>
                    <a
                        href="{{ route('admin.locations.show', $location) }}"
                        class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                    >
                        К локации
                    </a>
                    <button
                        type="button"
                        onclick="refreshDaemon({{ $location->id }})"
                        class="inline-flex items-center rounded-md bg-sky-600 px-4 py-2 text-[11px] font-semibold text-white shadow-sm hover:bg-sky-500"
                    >
                        Обновить состояние демона
                    </button>
                    <button
                        type="button"
                        onclick="restartDaemon({{ $location->id }})"
                        class="inline-flex items-center rounded-md bg-amber-600 px-4 py-2 text-[11px] font-semibold text-white shadow-sm hover:bg-amber-500"
                    >
                        Перезапустить демон
                    </button>
                </div>
            </div>

            <div class="grid gap-4 md:grid-cols-3 text-sm">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <p class="text-xs font-semibold text-slate-300/70">Статус демона</p>
                    @php
                        $status = optional($daemon)->status;
                    @endphp
                    <div class="mt-2 flex items-center gap-2">
                        <span class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px]
                            @if($status === \App\Models\LocationDaemon::STATUS_ONLINE) bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20
                            @elseif($status === \App\Models\LocationDaemon::STATUS_OFFLINE) bg-red-500/10 text-red-200 ring-1 ring-red-500/20
                            @else bg-white/5 text-slate-200 ring-1 ring-white/10
                            @endif">
                            <span class="inline-block h-1.5 w-1.5 rounded-full
                                @if($status === \App\Models\LocationDaemon::STATUS_ONLINE) bg-emerald-400
                                @elseif($status === \App\Models\LocationDaemon::STATUS_OFFLINE) bg-red-400
                                @else bg-slate-400
                                @endif"></span>
                            @switch($status)
                                @case(\App\Models\LocationDaemon::STATUS_ONLINE)
                                    Онлайн
                                    @break
                                @case(\App\Models\LocationDaemon::STATUS_OFFLINE)
                                    Оффлайн
                                    @break
                                @case(\App\Models\LocationDaemon::STATUS_UNKNOWN)
                                    Неизвестно
                                    @break
                                @default
                                    Нет данных
                            @endswitch
                        </span>
                    </div>
                    <p class="mt-2 text-[11px] text-slate-300/70">
                        Последний контакт: {{ optional($daemon->last_seen)->diffForHumans() ?? '—' }}
                    </p>
                    <p class="mt-1 text-[11px] text-slate-300/70">
                        Версия демона: {{ $daemon->version ?? '—' }}
                    </p>
                </div>

                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <p class="text-xs font-semibold text-slate-300/70">Сервер</p>
                    <dl class="mt-2 space-y-1 text-[13px] text-slate-200">
                        <div class="flex gap-2">
                            <dt class="w-28 text-slate-300/70">Хост</dt>
                            <dd class="flex-1">{{ $location->ssh_host ?? '—' }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-28 text-slate-300/70">Порт демона</dt>
                            <dd class="flex-1">{{ config('services.location_daemon.port', 9201) }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-28 text-slate-300/70">PID</dt>
                            <dd class="flex-1">{{ $daemon->pid ?? '—' }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-28 text-slate-300/70">Платформа</dt>
                            <dd class="flex-1">{{ $daemon->platform ?? '—' }}</dd>
                        </div>
                    </dl>
                </div>

                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <p class="text-xs font-semibold text-slate-300/70">Время работы</p>
                    <p class="mt-2 text-sm font-semibold text-slate-100">
                        @if(optional($daemon)->uptime_sec)
                            {{ round($daemon->uptime_sec / 3600, 1) }} ч
                        @else
                            —
                        @endif
                    </p>
                    <p class="mt-1 text-[11px] text-slate-300/70">
                        Значение uptime берётся из демона и обновляется при каждом запросе /info.
                    </p>
                </div>
            </div>

            <div class="grid gap-4 lg:grid-cols-2 text-sm">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                    <h2 class="text-sm font-semibold text-slate-100">Статистика и графики</h2>
                    @php
                        $cpuMetrics = ($metrics['daemon_cpu_usage'] ?? collect())->sortBy('measured_at')->values();
                        $ramMetrics = ($metrics['daemon_ram_usage'] ?? collect())->sortBy('measured_at')->values();
                        $hasData = !$cpuMetrics->isEmpty() || !$ramMetrics->isEmpty();
                    @endphp
                    @if (!$hasData)
                        <div class="mt-3 h-40 md:h-40 rounded-xl border border-white/10 bg-black/10 text-[11px] text-slate-300/80 flex items-center justify-center px-4 text-center">
                            <span>Для этой локации пока нет загруженных метрик. Выполните сбор метрик из демона, чтобы увидеть графики.</span>
                        </div>
                    @else
                        <div class="mt-3">
                            <canvas id="daemon-metrics-chart" class="w-full h-64 md:h-40"></canvas>
                        </div>
                    @endif
                </div>

                <div class="rounded-2xl border border-white/10 bg-black/20 p-4 shadow-sm text-[11px] text-slate-200">
                    <div class="flex items-center justify-between mb-2">
                        <h2 class="text-sm font-semibold text-slate-100">Real-time консоль демона</h2>
                        <p id="daemon-log-status" class="text-[11px] text-slate-400">Подключение к логам...</p>
                    </div>
                    <div id="daemon-log-output" class="h-56 rounded-md border border-white/10 bg-[#0b1220] px-3 py-2 font-mono text-[11px] text-slate-200 overflow-y-auto whitespace-pre-wrap mb-2"></div>
                    <div class="flex items-center gap-2">
                        <input
                            id="daemon-cmd-input"
                            type="text"
                            placeholder="Введите команду для выполнения на сервере"
                            class="flex-1 rounded-md border border-white/10 bg-black/10 px-2 py-1 text-[11px] text-slate-100 placeholder-slate-400 focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            autocomplete="off"
                        >
                        <button
                            id="daemon-cmd-send"
                            type="button"
                            class="inline-flex items-center rounded-md bg-emerald-600 px-3 py-1.5 text-[11px] font-medium text-white hover:bg-emerald-500 disabled:opacity-50"
                        >
                            Выполнить
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </section>
@endsection

@push('scripts')
    <script>
        function refreshDaemon(locationId) {
            fetch(`/admin/locations/${locationId}/daemon/refresh`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-CSRF-TOKEN': document.querySelector('meta[name="csrf-token"]').getAttribute('content')
                }
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    location.reload();
                } else {
                    alert('Ошибка обновления: ' + (data.error || 'Неизвестная ошибка'));
                }
            })
            .catch(error => {
                alert('Ошибка сети: ' + error.message);
            });
        }

        function restartDaemon(locationId) {
            if (!confirm('Перезапустить Vortanix Daemon на этой локации?')) return;

            fetch(`/admin/locations/${locationId}/daemon/restart`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-CSRF-TOKEN': document.querySelector('meta[name="csrf-token"]').getAttribute('content')
                }
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    alert('Демон перезапущен');
                    location.reload();
                } else {
                    alert('Ошибка перезапуска: ' + (data.error || 'Неизвестная ошибка'));
                }
            })
            .catch(error => {
                alert('Ошибка сети: ' + error.message);
            });
        }

        document.addEventListener('DOMContentLoaded', () => {
            // Графики CPU/RAM
            const rawCpu = @json(
                ($metrics['daemon_cpu_usage'] ?? collect())->sortBy('measured_at')->values()->map(fn($m) => [
                    't' => optional($m->measured_at)->toIso8601String(),
                    'v' => $m->value,
                ])
            );
            const rawRam = @json(
                ($metrics['daemon_ram_usage'] ?? collect())->sortBy('measured_at')->values()->map(fn($m) => [
                    't' => optional($m->measured_at)->toIso8601String(),
                    'v' => $m->value,
                ])
            );

            if (rawCpu.length || rawRam.length) {
                const allTimestamps = [...rawCpu.map(p => p.t), ...rawRam.map(p => p.t)].filter(Boolean).sort();
                const uniqueTimestamps = [...new Set(allTimestamps)];

                const labels = uniqueTimestamps.map(t => {
                    const d = new Date(t);
                    return d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
                });

                const ctx = document.getElementById('daemon-metrics-chart');
                if (ctx && window.Chart) {
                    const chartCtx = ctx.getContext('2d');
                    const isMobile = window.innerWidth < 768;
                    new Chart(chartCtx, {
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
                                legend: { display: !isMobile, position: 'top' },
                            },
                            scales: {
                                x: {
                                    ticks: { maxTicksLimit: isMobile ? 4 : 6, font: { size: isMobile ? 9 : 10 } },
                                    grid: { display: false },
                                },
                                y: {
                                    ticks: { font: { size: isMobile ? 9 : 10 } },
                                    grid: { color: 'rgba(148, 163, 184, 0.2)' },
                                },
                            },
                        },
                    });
                }
            }

            // Real-time логи демона
            const logBox = document.getElementById('daemon-log-output');
            const statusEl = document.getElementById('daemon-log-status');
            const logsUrl = @json(route('admin.locations.daemon.logs', $location));
            const execUrl = @json(route('admin.locations.daemon.exec', $location));
            const cmdInput = document.getElementById('daemon-cmd-input');
            const cmdSend = document.getElementById('daemon-cmd-send');

            if (!logBox || !logsUrl) return;

            let polling = true;
            let lastLogsSignature = '';

            function updateLogs() {
                if (!polling || document.hidden) return;

                fetch(logsUrl, { headers: { 'Accept': 'application/json' } })
                    .then(response => response.json())
                    .then(data => {
                        if (data.error) {
                            statusEl.textContent = 'Ошибка: ' + data.error;
                            return;
                        }
                        const lines = Array.isArray(data.logs) ? data.logs : [];

                        // Если содержимое логов не поменялось — не трогаем DOM, чтобы не лагало
                        const signature = lines.join('\n');
                        if (signature === lastLogsSignature) {
                            return;
                        }
                        lastLogsSignature = signature;

                        // Преобразуем ISO-дату в начале строки в локальное время [HH:MM:SS]
                        const formatted = lines.map(line => {
                            const idx = line.indexOf(' | ');
                            if (idx === -1) return line;
                            const ts = line.slice(0, idx);
                            const msg = line.slice(idx + 3);
                            const d = new Date(ts);
                            if (isNaN(d.getTime())) return line;
                            const t = d.toLocaleTimeString('ru-RU', {
                                hour: '2-digit',
                                minute: '2-digit',
                                second: '2-digit',
                            });
                            return `[${t}] ${msg}`;
                        });

                        logBox.textContent = formatted.join('\n');
                        logBox.scrollTop = logBox.scrollHeight;
                        statusEl.textContent = `Получено ${lines.length} строк логов`;
                    })
                    .catch(error => {
                        statusEl.textContent = 'Ошибка загрузки логов: ' + error.message;
                    });
            }

            updateLogs();
            const intervalId = setInterval(updateLogs, 5000);

            function sendCommand() {
                if (!execUrl || !cmdInput || !cmdSend) return;
                const cmd = cmdInput.value.trim();
                if (!cmd) return;

                cmdSend.disabled = true;
                statusEl.textContent = 'Выполнение команды...';

                fetch(execUrl, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-CSRF-TOKEN': document.querySelector('meta[name="csrf-token"]').getAttribute('content'),
                        'Accept': 'application/json',
                    },
                    body: JSON.stringify({ cmd }),
                })
                    .then(response => response.json())
                    .then(data => {
                        if (data.error) {
                            statusEl.textContent = 'Ошибка exec: ' + data.error;
                        } else if (data.ok === false && data.error) {
                            statusEl.textContent = 'Ошибка команды: ' + data.error;
                        } else {
                            statusEl.textContent = 'Команда выполнена';
                        }
                        cmdInput.value = '';
                        updateLogs();
                    })
                    .catch(error => {
                        statusEl.textContent = 'Ошибка запроса exec: ' + error.message;
                    })
                    .finally(() => {
                        cmdSend.disabled = false;
                    });
            }

            if (cmdSend && cmdInput) {
                cmdSend.addEventListener('click', sendCommand);
                cmdInput.addEventListener('keydown', (e) => {
                    if (e.key === 'Enter') {
                        e.preventDefault();
                        sendCommand();
                    }
                });
            }

            window.addEventListener('beforeunload', () => {
                polling = false;
                clearInterval(intervalId);
            });
        });
    </script>
@endpush
