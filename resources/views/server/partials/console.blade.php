<div class="mt-6 overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] text-slate-100 shadow-sm transition-shadow duration-300 hover:shadow-md">
    <div class="border-b border-white/10 bg-black/10 px-5 py-3">
        <div class="flex items-center justify-between gap-3">
            <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Консоль</div>
            <div class="flex items-center gap-2">
                <div class="flex items-center gap-2">
                    <select id="consoleThemePreset" class="h-8 rounded-xl border border-white/10 bg-black/10 px-2.5 text-[11px] font-semibold text-slate-200 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                        <option value="default">Default</option>
                        <option value="dark">Dark</option>
                        <option value="black">Black</option>
                        <option value="light">Light</option>
                        <option value="custom">Custom</option>
                    </select>
                    <input id="consoleThemeBg" type="color" class="h-8 w-10 rounded-xl border border-white/10 bg-black/10 p-1" title="Цвет фона" />
                    <input id="consoleThemeFg" type="color" class="h-8 w-10 rounded-xl border border-white/10 bg-black/10 p-1" title="Цвет текста" />
                </div>
                <button id="consolePauseBtn" type="button" class="inline-flex items-center rounded-xl border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-semibold text-slate-200 shadow-sm transition hover:bg-black/15 hover:ring-1 hover:ring-white/10">
                    Пауза
                </button>
                <button id="consoleClearBtn" type="button" class="inline-flex items-center rounded-xl border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-semibold text-slate-200 shadow-sm transition hover:bg-black/15 hover:ring-1 hover:ring-white/10">
                    Очистить
                </button>
            </div>
        </div>
    </div>

    <div class="p-5 space-y-3">
        <div id="serverConsoleWrap" class="rounded-2xl border border-white/10 shadow-sm" style="--console-bg: #020617; --console-fg: #e2e8f0; background: var(--console-bg); color: var(--console-fg);">
            <pre id="serverConsole" class="max-h-[720px] overflow-auto p-4 text-xs whitespace-pre-wrap" style="background: transparent; color: inherit;">Загрузка...</pre>
        </div>

        <form id="serverConsoleForm" class="flex flex-col gap-2 sm:flex-row sm:items-center">
            <input
                id="serverConsoleCommand"
                name="command"
                type="text"
                placeholder="Введите команду RCON (например: players)"
                class="w-full flex-1 rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500 placeholder-slate-400"
            >
            <button id="serverConsoleSend" type="submit" class="inline-flex items-center justify-center rounded-xl bg-slate-900 px-4 py-2 text-[12px] font-semibold text-white shadow-sm transition hover:bg-slate-800 active:scale-[0.99]">
                Отправить
            </button>
        </form>

        <div id="serverConsoleCommandStatus" class="text-[11px] text-slate-300/70"></div>

        <div class="overflow-hidden rounded-2xl border border-white/10 bg-black/10">
            <div class="border-b border-white/10 px-4 py-2 text-[11px] uppercase tracking-wide text-slate-300/70">
                Ответ RCON
            </div>
            <pre id="serverConsoleResponse" class="max-h-64 overflow-auto px-4 py-3 text-xs whitespace-pre-wrap" style="color: var(--console-fg);">—</pre>
        </div>
    </div>
</div>
