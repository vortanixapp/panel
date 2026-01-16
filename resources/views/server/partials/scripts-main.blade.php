<script>
    document.addEventListener('DOMContentLoaded', function () {
        const metricsUrl = "{{ route('server.metrics', $server) }}";
        const statusUrl = "{{ route('server.status', $server) }}";
        const initialRuntime = "{{ strtolower((string) ($server->runtime_status ?? '')) }}";

        function showToast(message, type) {
            const container = document.getElementById('serverToastContainer');
            if (!container) return;

            const t = (type || 'info').toLowerCase();
            let cls = 'border-sky-500/30 bg-sky-600';
            if (t === 'success') {
                cls = 'border-emerald-500/30 bg-emerald-600';
            } else if (t === 'error') {
                cls = 'border-rose-500/30 bg-rose-600';
            } else if (t === 'warning') {
                cls = 'border-amber-500/30 bg-amber-600';
            }

            const el = document.createElement('div');
            el.className = `rounded-lg border ${cls} shadow-lg p-4 text-sm text-white flex items-start gap-2`;
            el.textContent = message;
            container.appendChild(el);

            setTimeout(() => {
                el.classList.add('opacity-0', 'transition', 'duration-300');
                setTimeout(() => {
                    try { el.remove(); } catch (e) {}
                }, 300);
            }, 3000);
        }

        try {
            const pending = sessionStorage.getItem('vtx_restart_pending');
            if (pending === '1') {
                sessionStorage.removeItem('vtx_restart_pending');
                showToast('Сервер ставится на перезапуск…', 'info');
            }
        } catch (e) {
        }

        const restartingWatch = initialRuntime === 'restarting';
        const restartStartedAt = Date.now();
        let restartWarned = false;
        let statusFailures = 0;
        let statusFailuresWarned = false;

        const labels = [];
        const cpuValues = [];
        const ramValues = [];
        const maxPoints = 30;

        function pushPoint(label, cpu, ram) {
            labels.push(label);
            cpuValues.push(cpu);
            ramValues.push(ram);

            while (labels.length > maxPoints) {
                labels.shift();
                cpuValues.shift();
                ramValues.shift();
            }
        }

        const statusBadge = document.getElementById('runtimeStatusBadge');

        function applyBadge(state) {
            if (!statusBadge) return;
            const s = (state || '').toLowerCase();

            let label = 'Неизвестно';
            let cls = 'bg-slate-100 text-slate-700';
            if (s === 'running') {
                label = 'Работает';
                cls = 'bg-green-100 text-green-700';
            } else if (s === 'restarting') {
                label = 'Перезапуск';
                cls = 'bg-amber-100 text-amber-700';
            } else if (s === 'offline' || s === 'stopped') {
                label = 'Выключен';
                cls = 'bg-gray-100 text-gray-700';
            } else if (s === 'missing') {
                label = 'Не установлен';
                cls = 'bg-rose-100 text-rose-700';
            }

            statusBadge.className = `inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${cls}`;
            statusBadge.textContent = label;
        }

        async function fetchStatus() {
            try {
                const resp = await fetch(statusUrl, { headers: { 'Accept': 'application/json' } });
                const data = await resp.json().catch(() => null);
                if (resp.ok && data && data.ok === true) {
                    statusFailures = 0;
                    const runtime = String(data.runtime_status || '').toLowerCase();
                    applyBadge(runtime);

                    if (initialRuntime === 'restarting' && runtime === 'running') {
                        showToast('Сервер перезапущен', 'success');
                        setTimeout(() => window.location.reload(), 800);
                        return;
                    }

                    if (restartingWatch && !restartWarned && runtime !== 'running' && (Date.now() - restartStartedAt) > 90000) {
                        restartWarned = true;
                        showToast('Перезапуск занимает больше обычного. Статус обновится автоматически.', 'warning');
                    }
                } else {
                    statusFailures += 1;
                    if (!statusFailuresWarned && statusFailures >= 3) {
                        statusFailuresWarned = true;
                        showToast('Не удалось получить статус сервера. Попробуйте обновить страницу позже.', 'warning');
                    }
                }
            } catch (e) {
                statusFailures += 1;
                if (!statusFailuresWarned && statusFailures >= 3) {
                    statusFailuresWarned = true;
                    showToast('Не удалось получить статус сервера. Попробуйте обновить страницу позже.', 'warning');
                }
            }
        }

        function timeLabel() {
            const d = new Date();
            const hh = String(d.getHours()).padStart(2, '0');
            const mm = String(d.getMinutes()).padStart(2, '0');
            const ss = String(d.getSeconds()).padStart(2, '0');
            return hh + ':' + mm + ':' + ss;
        }

        let chart = null;
        const canvas = document.getElementById('serverLoadChart');
        if (window.Chart && canvas) {
            const ctx = canvas.getContext('2d');
            chart = new window.Chart(ctx, {
                type: 'line',
                data: {
                    labels,
                    datasets: [
                        {
                            label: 'CPU usage, %',
                            data: cpuValues,
                            borderColor: '#3b82f6',
                            backgroundColor: 'rgba(59, 130, 246, 0.12)',
                            pointRadius: 0,
                            borderWidth: 2,
                            tension: 0.25,
                            fill: true,
                        },
                        {
                            label: 'RAM usage, %',
                            data: ramValues,
                            borderColor: '#22c55e',
                            backgroundColor: 'rgba(34, 197, 94, 0.12)',
                            pointRadius: 0,
                            borderWidth: 2,
                            tension: 0.25,
                            fill: true,
                        },
                    ],
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    animation: false,
                    plugins: {
                        legend: {
                            display: true,
                            position: 'top',
                            labels: {
                                boxWidth: 30,
                                boxHeight: 10,
                                color: '#cbd5e1',
                            },
                        },
                        tooltip: {
                            enabled: true,
                            backgroundColor: 'rgba(2, 6, 23, 0.92)',
                            titleColor: '#e2e8f0',
                            bodyColor: '#e2e8f0',
                            borderColor: 'rgba(255, 255, 255, 0.10)',
                            borderWidth: 1,
                        },
                    },
                    scales: {
                        y: {
                            beginAtZero: true,
                            min: 0,
                            max: 100,
                            ticks: {
                                color: 'rgba(226, 232, 240, 0.70)',
                                callback: function (value) {
                                    return value + '%';
                                },
                            },
                            grid: {
                                color: 'rgba(255, 255, 255, 0.08)',
                            },
                            border: {
                                color: 'rgba(255, 255, 255, 0.10)',
                            },
                        },
                        x: {
                            ticks: {
                                color: 'rgba(226, 232, 240, 0.70)',
                            },
                            grid: {
                                color: 'rgba(255, 255, 255, 0.06)',
                            },
                            border: {
                                color: 'rgba(255, 255, 255, 0.10)',
                            },
                        },
                    },
                },
            });
        }

        async function loadMetrics() {
            try {
                const resp = await fetch(metricsUrl, { headers: { 'Accept': 'application/json' } });
                const data = await resp.json();

                if (!resp.ok || !data || data.ok !== true) {
                    return;
                }

                const cpu = typeof data.cpu_percent === 'number' ? data.cpu_percent : 0;
                const ram = typeof data.mem_percent === 'number' ? data.mem_percent : 0;

                const cpuEl = document.getElementById('mainCpuText');
                const cpuBar = document.getElementById('mainCpuBar');
                const ramEl = document.getElementById('mainRamText');
                const ramBar = document.getElementById('mainRamBar');
                const diskEl = document.getElementById('mainDiskText');
                const diskBar = document.getElementById('mainDiskBar');
                const updatedAt = document.getElementById('mainResourcesUpdatedAt');

                if (cpuEl && cpuBar) {
                    const cpuPct = Math.max(0, Math.min(100, cpu));
                    cpuEl.textContent = cpuPct.toFixed(1) + '%';
                    cpuBar.style.width = cpuPct + '%';
                }

                if (ramEl && ramBar) {
                    const ramPct = Math.max(0, Math.min(100, ram));
                    const memLimitMb = (typeof data.mem_limit_mb === 'number') ? data.mem_limit_mb : null;
                    if (memLimitMb !== null && memLimitMb > 0) {
                        const usedMb = (ramPct / 100.0) * memLimitMb;
                        if (memLimitMb < 1024) {
                            ramEl.textContent = ramPct.toFixed(1) + '% (' + usedMb.toFixed(0) + ' / ' + memLimitMb.toFixed(0) + ' MB)';
                        } else {
                            const usedGb = usedMb / 1024.0;
                            const totalGb = memLimitMb / 1024.0;
                            if (totalGb < 1) {
                                ramEl.textContent = ramPct.toFixed(1) + '% (' + usedMb.toFixed(0) + ' MB / ' + (totalGb * 1024.0).toFixed(0) + ' MB)';
                            } else {
                                ramEl.textContent = ramPct.toFixed(1) + '% (' + usedGb.toFixed(2) + ' / ' + totalGb.toFixed(2) + ' GB)';
                            }
                        }
                    } else {
                        ramEl.textContent = ramPct.toFixed(1) + '%';
                    }
                    ramBar.style.width = ramPct + '%';
                }

                if (diskEl && diskBar) {
                    const usedMb = (typeof data.disk_used_mb === 'number') ? data.disk_used_mb : null;
                    const totalMb = (typeof data.disk_total_mb === 'number') ? data.disk_total_mb : null;

                    let diskPct = null;
                    if (usedMb !== null && totalMb !== null && totalMb > 0) {
                        diskPct = Math.max(0, Math.min(100, (usedMb / totalMb) * 100.0));
                    } else {
                        const diskPctRaw = (typeof data.disk_percent === 'number') ? data.disk_percent : null;
                        diskPct = (diskPctRaw === null) ? null : Math.max(0, Math.min(100, diskPctRaw));
                    }

                    if (diskPct !== null) {
                        if (usedMb !== null && totalMb !== null && totalMb > 0) {
                            const totalGb = totalMb / 1024.0;

                            if (usedMb < 1024) {
                                diskEl.textContent = usedMb.toFixed(0) + ' MB / ' + totalGb.toFixed(2) + ' GB (' + diskPct.toFixed(1) + '%)';
                            } else {
                                const usedGb = usedMb / 1024.0;
                                diskEl.textContent = usedGb.toFixed(2) + ' / ' + totalGb.toFixed(2) + ' GB (' + diskPct.toFixed(1) + '%)';
                            }
                        } else {
                            diskEl.textContent = diskPct.toFixed(1) + '%';
                        }
                        diskBar.style.width = diskPct + '%';
                    } else {
                        diskEl.textContent = '—';
                        diskBar.style.width = '0%';
                    }
                }

                if (updatedAt) {
                    const ts = (typeof data.ts === 'string') ? data.ts : '';

                    function formatTs(raw) {
                        if (typeof raw !== 'string') {
                            return '';
                        }

                        const s = raw.trim();
                        if (!s) {
                            return '';
                        }

                        const m = s.match(/^(\d{4})-(\d{2})-(\d{2})[T\s](\d{2}):(\d{2}):(\d{2})(?:\.(\d+))?(?:Z)?$/);
                        if (m) {
                            const yyyy = m[1];
                            const mm = m[2];
                            const dd = m[3];
                            const HH = m[4];
                            const MM = m[5];
                            const SS = m[6];
                            return dd + '.' + mm + '.' + yyyy + ' ' + HH + ':' + MM + ':' + SS;
                        }

                        return s.replace('T', ' ').replace('Z', '').replace(/\.\d+/, '');
                    }

                    updatedAt.textContent = ts ? formatTs(ts) : timeLabel();
                }

                if (chart) {
                    pushPoint(timeLabel(), Math.max(0, Math.min(100, cpu)), Math.max(0, Math.min(100, ram)));
                    chart.update('none');
                }
            } catch (e) {
                // ignore
            }
        }

        loadMetrics();
        setInterval(loadMetrics, 3000);

        fetchStatus();
        setInterval(fetchStatus, initialRuntime === 'restarting' ? 3000 : 10000);
    });
</script>
