<div class="mt-6 overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm transition-shadow duration-300 hover:shadow-md">
    <div class="border-b border-white/10 bg-black/10 px-5 py-4">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
                <h2 class="text-sm font-semibold text-slate-100">Логи</h2>
                <p class="mt-1 text-[12px] text-slate-300/70">Вывод docker logs для сервера (только чтение).</p>
            </div>

            <div class="flex flex-wrap items-center gap-2">
                <label class="text-[11px] font-semibold text-slate-300/70">
                    <span class="sr-only">Tail</span>
                    <select id="serverLogsTail" class="rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-xs font-semibold text-slate-200 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                        <option value="200">tail 200</option>
                        <option value="400" selected>tail 400</option>
                        <option value="700">tail 700</option>
                        <option value="1000">tail 1000</option>
                    </select>
                </label>

                <button id="serverLogsPauseBtn" type="button" class="inline-flex items-center justify-center rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-xs font-semibold text-slate-200 shadow-sm transition hover:bg-black/15">
                    Пауза
                </button>

                <button id="serverLogsClearBtn" type="button" class="inline-flex items-center justify-center rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-xs font-semibold text-slate-200 shadow-sm transition hover:bg-black/15">
                    Очистить
                </button>

                <button id="serverLogsDownloadBtn" type="button" class="inline-flex items-center justify-center rounded-xl bg-slate-900 px-3 py-2 text-xs font-semibold text-white shadow-sm transition hover:bg-slate-800">
                    Скачать
                </button>
            </div>
        </div>
    </div>

    <div class="p-5">
        <div class="mb-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div class="text-[12px] text-slate-200">Источник: <span class="font-semibold">docker logs</span></div>
            <div id="serverLogsStatus" class="text-[12px] text-slate-300/70">—</div>
        </div>

        <div class="rounded-2xl border border-white/10 bg-slate-950/95 p-4 shadow-inner">
            <pre id="serverLogsOutput" class="max-h-[520px] overflow-auto whitespace-pre-wrap break-words font-mono text-[12px] leading-5 text-slate-100"></pre>
        </div>
    </div>
</div>
