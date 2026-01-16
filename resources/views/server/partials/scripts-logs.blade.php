<script>
    document.addEventListener('DOMContentLoaded', function () {
        const outputEl = document.getElementById('serverLogsOutput');
        if (!outputEl) {
            return;
        }

        const statusEl = document.getElementById('serverLogsStatus');
        const tailEl = document.getElementById('serverLogsTail');
        const pauseBtn = document.getElementById('serverLogsPauseBtn');
        const clearBtn = document.getElementById('serverLogsClearBtn');
        const downloadBtn = document.getElementById('serverLogsDownloadBtn');

        const url = "{{ route('server.console.logs', $server) }}";

        let paused = false;
        let lastText = '';

        function currentTail() {
            const v = tailEl ? parseInt(tailEl.value || '400', 10) : 400;
            if (!Number.isFinite(v)) return 400;
            return Math.max(1, Math.min(v, 1000));
        }

        function setStatus(text) {
            if (statusEl) {
                statusEl.textContent = text;
            }
        }

        async function loadLogs() {
            if (paused) {
                return;
            }

            const shouldStick = (outputEl.scrollTop + outputEl.clientHeight) >= (outputEl.scrollHeight - 20);

            try {
                const tail = currentTail();
                const resp = await fetch(url + '?tail=' + encodeURIComponent(String(tail)), { headers: { 'Accept': 'application/json' } });
                const data = await resp.json().catch(() => null);

                if (!resp.ok || !data || data.ok !== true) {
                    const err = (data && data.error) ? data.error : ('HTTP ' + resp.status);
                    setStatus('Ошибка: ' + err);
                    return;
                }

                const logs = Array.isArray(data.logs) ? data.logs : [];
                const text = logs.join('\n');
                lastText = text;
                outputEl.textContent = text;

                const state = (data && data.state) ? String(data.state) : '—';
                setStatus('Статус: ' + state + ' • строк: ' + logs.length);

                if (shouldStick) {
                    outputEl.scrollTop = outputEl.scrollHeight;
                }
            } catch (e) {
                setStatus('Ошибка: ' + (e && e.message ? e.message : String(e)));
            }
        }

        if (pauseBtn) {
            pauseBtn.addEventListener('click', function () {
                paused = !paused;
                pauseBtn.textContent = paused ? 'Продолжить' : 'Пауза';
            });
        }

        if (clearBtn) {
            clearBtn.addEventListener('click', function () {
                outputEl.textContent = '';
                lastText = '';
                setStatus('Очищено');
            });
        }

        if (downloadBtn) {
            downloadBtn.addEventListener('click', function () {
                const blob = new Blob([lastText || ''], { type: 'text/plain;charset=utf-8' });
                const a = document.createElement('a');
                const ts = new Date();
                const stamp = ts.toISOString().replace(/[:.]/g, '-');
                a.href = URL.createObjectURL(blob);
                a.download = 'server-' + {{ (int) $server->id }} + '-logs-' + stamp + '.txt';
                document.body.appendChild(a);
                a.click();
                a.remove();
                URL.revokeObjectURL(a.href);
            });
        }

        if (tailEl) {
            tailEl.addEventListener('change', function () {
                loadLogs();
            });
        }

        loadLogs();
        setInterval(loadLogs, 3000);
    });
</script>
