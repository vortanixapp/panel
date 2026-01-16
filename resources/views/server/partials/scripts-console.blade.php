<script>
    document.addEventListener('DOMContentLoaded', function () {
        const el = document.getElementById('serverConsole');
        if (!el) {
            return;
        }

        const serverId = {{ (int) ($server->id ?? 0) }};
        const themeKey = 'vortanix_console_theme_' + String(serverId || 'global');

        const consoleWrap = document.getElementById('serverConsoleWrap');
        const responseEl = document.getElementById('serverConsoleResponse');
        const presetSelect = document.getElementById('consoleThemePreset');
        const bgInput = document.getElementById('consoleThemeBg');
        const fgInput = document.getElementById('consoleThemeFg');

        const PRESETS = {
            default: { bg: '#020617', fg: '#e2e8f0' },
            dark: { bg: '#0b1220', fg: '#e2e8f0' },
            black: { bg: '#000000', fg: '#e5e7eb' },
            light: { bg: '#f8fafc', fg: '#0f172a' },
        };

        function normalizeColor(v, fallback) {
            if (typeof v !== 'string') {
                return fallback;
            }
            const s = v.trim();
            return /^#[0-9a-fA-F]{6}$/.test(s) ? s : fallback;
        }

        function applyConsoleTheme(bg, fg) {
            if (!consoleWrap) {
                return;
            }
            const safeBg = normalizeColor(bg, PRESETS.default.bg);
            const safeFg = normalizeColor(fg, PRESETS.default.fg);
            consoleWrap.style.setProperty('--console-bg', safeBg);
            consoleWrap.style.setProperty('--console-fg', safeFg);
            if (responseEl) {
                responseEl.style.color = safeFg;
            }
        }

        function saveTheme(state) {
            try {
                localStorage.setItem(themeKey, JSON.stringify(state));
            } catch (e) {
                // ignore
            }
        }

        function loadTheme() {
            try {
                const raw = localStorage.getItem(themeKey);
                if (!raw) {
                    return null;
                }
                const parsed = JSON.parse(raw);
                if (!parsed || typeof parsed !== 'object') {
                    return null;
                }
                return {
                    preset: typeof parsed.preset === 'string' ? parsed.preset : 'default',
                    bg: typeof parsed.bg === 'string' ? parsed.bg : PRESETS.default.bg,
                    fg: typeof parsed.fg === 'string' ? parsed.fg : PRESETS.default.fg,
                };
            } catch (e) {
                return null;
            }
        }

        function setInputs(bg, fg) {
            if (bgInput) {
                bgInput.value = normalizeColor(bg, PRESETS.default.bg);
            }
            if (fgInput) {
                fgInput.value = normalizeColor(fg, PRESETS.default.fg);
            }
        }

        function setPreset(preset) {
            const p = PRESETS[preset] || PRESETS.default;
            applyConsoleTheme(p.bg, p.fg);
            setInputs(p.bg, p.fg);
            if (presetSelect) {
                presetSelect.value = preset;
            }
            saveTheme({ preset, bg: p.bg, fg: p.fg });
        }

        function initTheme() {
            const saved = loadTheme();
            if (saved) {
                const preset = saved.preset;
                if (preset && preset !== 'custom' && PRESETS[preset]) {
                    setPreset(preset);
                    return;
                }

                if (presetSelect) {
                    presetSelect.value = 'custom';
                }
                applyConsoleTheme(saved.bg, saved.fg);
                setInputs(saved.bg, saved.fg);
                saveTheme({ preset: 'custom', bg: normalizeColor(saved.bg, PRESETS.default.bg), fg: normalizeColor(saved.fg, PRESETS.default.fg) });
                return;
            }

            setPreset('default');
        }

        initTheme();

        if (presetSelect) {
            presetSelect.addEventListener('change', function () {
                const preset = presetSelect.value;
                if (preset === 'custom') {
                    const bg = bgInput ? bgInput.value : PRESETS.default.bg;
                    const fg = fgInput ? fgInput.value : PRESETS.default.fg;
                    applyConsoleTheme(bg, fg);
                    saveTheme({ preset: 'custom', bg: normalizeColor(bg, PRESETS.default.bg), fg: normalizeColor(fg, PRESETS.default.fg) });
                    return;
                }
                setPreset(preset);
            });
        }

        if (bgInput) {
            bgInput.addEventListener('input', function () {
                const bg = bgInput.value;
                const fg = fgInput ? fgInput.value : PRESETS.default.fg;
                applyConsoleTheme(bg, fg);
                if (presetSelect) {
                    presetSelect.value = 'custom';
                }
                saveTheme({ preset: 'custom', bg: normalizeColor(bg, PRESETS.default.bg), fg: normalizeColor(fg, PRESETS.default.fg) });
            });
        }

        if (fgInput) {
            fgInput.addEventListener('input', function () {
                const bg = bgInput ? bgInput.value : PRESETS.default.bg;
                const fg = fgInput.value;
                applyConsoleTheme(bg, fg);
                if (presetSelect) {
                    presetSelect.value = 'custom';
                }
                saveTheme({ preset: 'custom', bg: normalizeColor(bg, PRESETS.default.bg), fg: normalizeColor(fg, PRESETS.default.fg) });
            });
        }

        const url = "{{ route('server.console.logs', $server) }}";
        const cmdUrl = "{{ route('server.console.command', $server) }}";

        const form = document.getElementById('serverConsoleForm');
        const input = document.getElementById('serverConsoleCommand');
        const sendBtn = document.getElementById('serverConsoleSend');
        const statusEl = document.getElementById('serverConsoleCommandStatus');
        const pauseBtn = document.getElementById('consolePauseBtn');
        const clearBtn = document.getElementById('consoleClearBtn');

        const csrfMeta = document.querySelector('meta[name="csrf-token"]');
        const csrfToken = csrfMeta ? csrfMeta.getAttribute('content') : '';

        let paused = false;
        let sending = false;
        let history = [];
        let historyIndex = -1;

        async function loadLogs() {
            if (paused) {
                return;
            }
            const shouldStickToBottom = (el.scrollTop + el.clientHeight) >= (el.scrollHeight - 20);

            try {
                const resp = await fetch(url + '?tail=400', { headers: { 'Accept': 'application/json' } });
                const data = await resp.json();

                if (!resp.ok || !data || data.ok !== true) {
                    const err = (data && data.error) ? data.error : ('HTTP ' + resp.status);
                    el.textContent = 'Ошибка загрузки: ' + err;
                    return;
                }

                const logs = Array.isArray(data.logs) ? data.logs : [];
                el.textContent = logs.join('\n');

                if (shouldStickToBottom) {
                    el.scrollTop = el.scrollHeight;
                }
            } catch (e) {
                el.textContent = 'Ошибка загрузки: ' + (e && e.message ? e.message : String(e));
            }
        }

        async function sendCommand(command) {
            if (!command || sending) {
                return;
            }

            sending = true;
            if (statusEl) {
                statusEl.textContent = 'Отправка команды...';
            }
            if (sendBtn) {
                sendBtn.disabled = true;
                sendBtn.classList.add('opacity-70', 'cursor-not-allowed');
            }

            try {
                const resp = await fetch(cmdUrl, {
                    method: 'POST',
                    headers: {
                        'Accept': 'application/json',
                        'Content-Type': 'application/json',
                        ...(csrfToken ? { 'X-CSRF-TOKEN': csrfToken } : {}),
                    },
                    body: JSON.stringify({ command }),
                });

                const data = await resp.json().catch(() => null);
                if (!resp.ok || !data || data.ok !== true) {
                    const err = (data && (data.error || data.message)) ? (data.error || data.message) : ('HTTP ' + resp.status);
                    if (statusEl) {
                        statusEl.textContent = 'Ошибка: ' + err;
                    }
                    if (responseEl) {
                        responseEl.textContent = (data && data.errors) ? JSON.stringify(data.errors, null, 2) : '—';
                    }
                    return;
                }

                const out = (typeof data.output === 'string') ? data.output : '';
                if (statusEl) {
                    statusEl.textContent = out ? 'Команда выполнена.' : 'Команда отправлена.';
                }
                if (responseEl) {
                    responseEl.textContent = out && out.trim() ? out : '—';
                    responseEl.scrollTop = 0;
                }
            } catch (e) {
                if (statusEl) {
                    statusEl.textContent = 'Ошибка: ' + (e && e.message ? e.message : String(e));
                }
            } finally {
                sending = false;
                if (sendBtn) {
                    sendBtn.disabled = false;
                    sendBtn.classList.remove('opacity-70', 'cursor-not-allowed');
                }
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
                el.textContent = '';
            });
        }

        if (form && input) {
            form.addEventListener('submit', function (e) {
                e.preventDefault();
                const cmd = (input.value || '').trim();
                if (!cmd) {
                    return;
                }

                history = history.filter((x) => x !== cmd);
                history.push(cmd);
                historyIndex = history.length;

                input.value = '';
                sendCommand(cmd);
            });

            input.addEventListener('keydown', function (e) {
                if (e.key === 'ArrowUp') {
                    if (!history.length) return;
                    e.preventDefault();
                    historyIndex = Math.max(0, (historyIndex < 0 ? history.length - 1 : historyIndex - 1));
                    input.value = history[historyIndex] || '';
                    input.setSelectionRange(input.value.length, input.value.length);
                }
                if (e.key === 'ArrowDown') {
                    if (!history.length) return;
                    e.preventDefault();
                    historyIndex = Math.min(history.length, historyIndex + 1);
                    input.value = historyIndex >= history.length ? '' : (history[historyIndex] || '');
                    input.setSelectionRange(input.value.length, input.value.length);
                }
            });
        }

        loadLogs();
        setInterval(loadLogs, 3000);
    });
</script>
