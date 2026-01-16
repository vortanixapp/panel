@extends('layouts.app-admin')

@section('page_title', 'Логи')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Логи</h1>
                    <p class="mt-1 text-sm text-slate-300/80">{{ (string) (($sources[$sourceKey]['label'] ?? '') ?: $sourceKey) }}</p>
                </div>
            </div>

            <div class="grid gap-6 xl:grid-cols-[340px,1fr]">
                <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
                    <div class="border-b border-white/10 bg-black/10 px-5 py-4">
                        <div class="space-y-2">
                            <div class="flex items-center justify-between gap-2">
                                <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Источник</div>
                            </div>
                            <select
                                class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100"
                                onchange="window.location.href=this.value"
                            >
                                @foreach(($sources ?? []) as $k => $s)
                                    <option value="{{ route('admin.logs.index', ['source' => $k]) }}" {{ $k === $sourceKey ? 'selected' : '' }}>
                                        {{ (string) (($s['label'] ?? '') ?: $k) }}
                                    </option>
                                @endforeach
                            </select>
                        </div>
                    </div>
                    <div class="p-3 space-y-2 max-h-[70vh] overflow-auto">
                        @forelse(($items ?? []) as $f)
                            <a
                                href="{{ route('admin.logs.index', ['source' => $sourceKey, 'item' => $f['key']]) }}"
                                class="block rounded-2xl border border-white/10 px-3 py-2 text-xs transition {{ $selected === $f['key'] ? 'bg-black/20 text-white ring-1 ring-white/10' : 'bg-black/10 text-slate-200 hover:bg-black/15' }}"
                            >
                                <div class="flex items-center justify-between gap-3">
                                    <div class="truncate font-semibold">{{ $f['label'] ?? $f['key'] }}</div>
                                    <div class="flex-shrink-0 text-[11px] text-slate-300/70">
                                        @if(($f['size'] ?? null) !== null)
                                            {{ number_format(((float) ($f['size'] ?? 0)) / 1024, 0) }} KB
                                        @else
                                            —
                                        @endif
                                    </div>
                                </div>
                            </a>
                        @empty
                            <div class="p-3 text-xs text-slate-300/70">Нет доступных элементов</div>
                        @endforelse
                    </div>
                </div>

                <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
                    <div class="border-b border-white/10 bg-black/10 px-5 py-4">
                        <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                            <div class="min-w-0">
                                <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Просмотр</div>
                                <div class="mt-1 truncate text-sm font-semibold text-slate-100">{{ $selected ?: '—' }}</div>
                            </div>
                            <div class="flex flex-wrap items-center gap-2">
                                <div class="flex items-center gap-2 rounded-xl border border-white/10 bg-black/10 px-3 py-2">
                                    <label class="text-[11px] text-slate-300/80" for="lines">Строк</label>
                                    <input id="lines" type="number" min="10" max="2000" value="250" class="w-24 rounded-md border border-white/10 bg-black/10 px-2 py-1 text-xs text-slate-100">
                                </div>
                                <label class="inline-flex items-center gap-2 rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-200">
                                    <input id="autorefresh" type="checkbox" checked>
                                    <span>Авто (3с)</span>
                                </label>
                                @if($selected && (string) (($sources[$sourceKey]['type'] ?? '') ?: '') !== 'command')
                                    <a href="{{ route('admin.logs.download', ['source' => $sourceKey, 'item' => $selected]) }}" class="rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 hover:bg-black/15">Скачать</a>
                                @endif
                                <button type="button" id="refreshBtn" class="rounded-xl bg-sky-600 px-3 py-2 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">Обновить</button>
                            </div>
                        </div>
                    </div>

                    <div class="p-4">
                        <div class="rounded-2xl border border-white/10 bg-black/10 overflow-hidden">
                            <pre id="logBox" class="h-[72vh] overflow-auto p-4 whitespace-pre font-mono text-[11px] leading-[1.35] text-slate-100"></pre>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </section>

    <script>
        (function () {
            const selected = @json($selected);
            const sourceKey = @json($sourceKey);
            const tailUrl = @json(route('admin.logs.tail'));

            const logBox = document.getElementById('logBox');
            const refreshBtn = document.getElementById('refreshBtn');
            const linesEl = document.getElementById('lines');
            const autoEl = document.getElementById('autorefresh');

            if (!selected || !logBox) return;

            let inFlight = false;

            async function loadTail() {
                if (inFlight) return;
                inFlight = true;

                try {
                    const url = new URL(tailUrl, window.location.origin);
                    url.searchParams.set('source', sourceKey);
                    url.searchParams.set('item', selected);
                    url.searchParams.set('lines', String(parseInt(linesEl.value || '250', 10) || 250));

                    const res = await fetch(url.toString(), {
                        headers: { 'Accept': 'application/json' },
                        credentials: 'same-origin'
                    });

                    if (!res.ok) return;
                    const data = await res.json();
                    if (!data || !data.ok) return;

                    logBox.textContent = data.content || '';
                } finally {
                    inFlight = false;
                }
            }

            refreshBtn?.addEventListener('click', loadTail);
            linesEl?.addEventListener('change', loadTail);

            loadTail();
            setInterval(function () {
                if (autoEl && !autoEl.checked) return;
                loadTail();
            }, 3000);
        })();
    </script>
@endsection
