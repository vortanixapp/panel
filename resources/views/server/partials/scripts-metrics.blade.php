<script>
    document.addEventListener('DOMContentLoaded', function () {
        const stateEl = document.getElementById('serverMetricsState');
        if (!stateEl) {
            return;
        }

        const metaEl = document.getElementById('serverMetricsMeta');
        const cpuEl = document.getElementById('serverMetricsCpu');
        const memEl = document.getElementById('serverMetricsMem');
        const updatedEl = document.getElementById('serverMetricsUpdated');
        const rawEl = document.getElementById('serverMetricsRaw');

        const chartCanvas = document.getElementById('serverMetricsChart');

        const url = "{{ route('server.metrics', $server) }}";

        const labels = [];
        const cpuPoints = [];
        const memPoints = [];
        const maxPoints = 60;

        const ChartCtor = (window && window.Chart) ? window.Chart : null;
        let chart = null;

        if (!ChartCtor) {
            if (metaEl) {
                metaEl.textContent = 'Chart.js не подключен (window.Chart отсутствует)';
            }
        } else if (chartCanvas) {
            const ctx = chartCanvas.getContext('2d');
            chart = new ChartCtor(ctx, {
                type: 'line',
                data: {
                    labels: labels,
                    datasets: [
                        {
                            label: 'CPU %',
                            data: cpuPoints,
                            borderColor: 'rgb(59, 130, 246)',
                            backgroundColor: 'rgba(59, 130, 246, 0.12)',
                            fill: true,
                            pointRadius: 0,
                            borderWidth: 2,
                            tension: 0.35,
                        },
                        {
                            label: 'RAM %',
                            data: memPoints,
                            borderColor: 'rgb(16, 185, 129)',
                            backgroundColor: 'rgba(16, 185, 129, 0.10)',
                            fill: true,
                            pointRadius: 0,
                            borderWidth: 2,
                            tension: 0.35,
                        },
                    ],
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    animation: false,
                    normalized: true,
                    interaction: {
                        mode: 'index',
                        intersect: false,
                    },
                    scales: {
                        x: {
                            grid: {
                                color: 'rgba(148, 163, 184, 0.18)',
                            },
                            ticks: {
                                maxTicksLimit: 8,
                                color: '#64748b',
                            },
                        },
                        y: {
                            min: 0,
                            max: 100,
                            grid: {
                                color: 'rgba(148, 163, 184, 0.18)',
                            },
                            ticks: {
                                color: '#64748b',
                                callback: (v) => v + '%',
                            },
                        },
                    },
                    plugins: {
                        legend: {
                            position: 'bottom',
                            labels: {
                                boxWidth: 10,
                                boxHeight: 10,
                                color: '#0f172a',
                                usePointStyle: true,
                            },
                        },
                        tooltip: {
                            callbacks: {
                                label: (ctx) => {
                                    const val = ctx && ctx.parsed ? ctx.parsed.y : null;
                                    const name = ctx && ctx.dataset ? ctx.dataset.label : '';
                                    if (typeof val !== 'number' || !Number.isFinite(val)) {
                                        return name + ': —';
                                    }
                                    return name + ': ' + val.toFixed(1) + '%';
                                },
                            },
                        },
                    },
                },
            });
        }

        function fmtPercent(v) {
            if (v === null || v === undefined) return '—';
            const n = Number(v);
            if (!Number.isFinite(n)) return '—';
            return n.toFixed(1);
        }

        function fmtTime(ts) {
            try {
                const d = ts ? new Date(String(ts)) : new Date();
                if (Number.isNaN(d.getTime())) {
                    return new Date().toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
                }
                return d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
            } catch (_) {
                return '';
            }
        }

        function pushPoint(arr, v) {
            const n = Number(v);
            if (!Number.isFinite(n)) {
                arr.push(null);
            } else {
                arr.push(Math.max(0, Math.min(n, 100)));
            }
            while (arr.length > maxPoints) {
                arr.shift();
            }
        }

        function pushLabel(label) {
            labels.push(label);
            while (labels.length > maxPoints) {
                labels.shift();
            }
        }

        function setText(el, text) {
            if (el) el.textContent = text;
        }

        async function loadMetrics() {
            try {
                const resp = await fetch(url, { headers: { 'Accept': 'application/json' } });
                const data = await resp.json().catch(() => null);

                if (!resp.ok || !data || data.ok !== true) {
                    const err = (data && data.error) ? data.error : ('HTTP ' + resp.status);
                    setText(stateEl, 'Ошибка');
                    setText(metaEl, err);
                    return;
                }

                const state = data && data.state ? String(data.state) : '—';
                const exists = data && typeof data.exists === 'boolean' ? data.exists : null;
                const port = data && data.port ? String(data.port) : null;

                setText(stateEl, state);
                setText(metaEl, (exists === null ? '—' : (exists ? 'Контейнер найден' : 'Контейнер не найден')) + (port ? (' • порт: ' + port) : ''));

                setText(cpuEl, fmtPercent(data.cpu_percent));
                setText(memEl, fmtPercent(data.mem_percent));

                if (data.ts) {
                    setText(updatedEl, 'Обновлено: ' + fmtTime(data.ts));
                }

                if (rawEl) {
                    rawEl.textContent = JSON.stringify(data, null, 2);
                }

                pushLabel(fmtTime(data.ts));
                pushPoint(cpuPoints, data.cpu_percent);
                pushPoint(memPoints, data.mem_percent);
                if (chart) {
                    chart.update('none');
                }
            } catch (e) {
                setText(stateEl, 'Ошибка');
                setText(metaEl, e && e.message ? e.message : String(e));
            }
        }

        loadMetrics();
        setInterval(loadMetrics, 3000);
    });
</script>
