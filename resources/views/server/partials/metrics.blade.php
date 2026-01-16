<div class="mt-6 space-y-6">
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 text-sm">
        <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm transition-shadow duration-300 hover:shadow-md">
            <p class="text-xs font-semibold text-slate-300/70">Состояние</p>
            <p id="serverMetricsState" class="mt-2 text-lg font-semibold text-slate-100">—</p>
            <p id="serverMetricsMeta" class="mt-1 text-[11px] text-slate-300/70">—</p>
        </div>

        <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm transition-shadow duration-300 hover:shadow-md">
            <div class="flex items-start justify-between gap-3">
                <div>
                    <p class="text-xs font-semibold text-slate-300/70">CPU</p>
                    <p class="mt-2 text-2xl font-semibold text-slate-100"><span id="serverMetricsCpu">—</span><span class="text-sm text-slate-300/70">%</span></p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Текущая загрузка контейнера.</p>
                </div>
                <div class="flex h-10 w-10 items-center justify-center rounded-2xl bg-gradient-to-tr from-sky-900/30 to-indigo-900/30 ring-1 ring-sky-500/30">
                    <svg class="h-5 w-5 text-sky-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M8 8h8v8H8V8z" />
                    </svg>
                </div>
            </div>
        </div>

        <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm transition-shadow duration-300 hover:shadow-md">
            <div class="flex items-start justify-between gap-3">
                <div>
                    <p class="text-xs font-semibold text-slate-300/70">RAM</p>
                    <p class="mt-2 text-2xl font-semibold text-slate-100"><span id="serverMetricsMem">—</span><span class="text-sm text-slate-300/70">%</span></p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Использование памяти контейнера.</p>
                </div>
                <div class="flex h-10 w-10 items-center justify-center rounded-2xl bg-emerald-900/30 ring-1 ring-emerald-500/30">
                    <svg class="h-5 w-5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17v-2a4 4 0 118 0v2" />
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 7h.01M12 11h.01M12 15h.01" />
                    </svg>
                </div>
            </div>
        </div>
    </div>

    <div class="overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm transition-shadow duration-300 hover:shadow-md">
        <div class="border-b border-white/10 bg-black/10 px-5 py-4">
            <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <h2 class="text-sm font-semibold text-slate-100">График нагрузки</h2>
                    <p class="mt-1 text-[12px] text-slate-300/70">CPU / RAM (%). Автообновление каждые 3 секунды.</p>
                </div>
                <div id="serverMetricsUpdated" class="text-[12px] text-slate-300/70">—</div>
            </div>
        </div>

        <div class="p-5">
            <div class="h-[280px] rounded-2xl border border-white/10 bg-black/10 p-3">
                <canvas id="serverMetricsChart"></canvas>
            </div>

            <details class="mt-4 rounded-2xl border border-white/10 bg-black/10 px-4 py-3">
                <summary class="cursor-pointer text-[12px] font-semibold text-slate-200">Raw JSON (для диагностики)</summary>
                <pre id="serverMetricsRaw" class="mt-3 overflow-auto text-[12px] leading-5 text-slate-200"></pre>
            </details>
        </div>
    </div>
</div>
