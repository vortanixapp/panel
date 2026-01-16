@extends('layouts.app-user')

@section('title', 'Мониторинг')
@section('page_title', 'Мониторинг')

@section('content')
    <section class="py-8">
        <div class="w-full px-4 sm:px-6 lg:px-8">
            <div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
                <div>
                    <h1 class="text-2xl font-bold text-slate-100">Мониторинг</h1>
                    <p class="mt-1 text-sm text-slate-300">Список серверов и текущий онлайн</p>
                </div>
                <div class="flex items-center gap-2">
                    <button id="monRefresh" type="button" class="inline-flex h-10 items-center justify-center rounded-xl border border-white/10 bg-black/10 px-4 text-xs font-semibold text-slate-200 shadow-sm hover:bg-black/15">
                        Обновить
                    </button>
                </div>
            </div>

            <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm">
                <div class="border-b border-white/10 px-5 py-4">
                    <div class="grid gap-3 md:grid-cols-3">
                        <div>
                            <label class="block text-[11px] text-slate-300/70">Игра</label>
                            <select id="monGame" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                                <option value="">Все игры</option>
                            </select>
                        </div>
                        <div>
                            <label class="block text-[11px] text-slate-300/70">Сортировка</label>
                            <select id="monSort" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                                <option value="game">По игре</option>
                                <option value="online_desc">По онлайну (убыв.)</option>
                                <option value="online_asc">По онлайну (возр.)</option>
                                <option value="name">По названию</option>
                            </select>
                        </div>
                        <div>
                            <label class="block text-[11px] text-slate-300/70">Поиск</label>
                            <input id="monSearch" type="text" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500" placeholder="Название или IP">
                        </div>
                    </div>
                    <div id="monError" class="mt-3 hidden rounded-xl border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-xs text-rose-100"></div>
                </div>

                <div class="p-5">
                    <div class="overflow-x-auto">
                        <table class="min-w-full text-left text-sm">
                            <thead class="text-[11px] text-slate-300/70">
                                <tr>
                                    <th class="py-2 pr-4">Сервер</th>
                                    <th class="py-2 pr-4">Игра</th>
                                    <th class="py-2 pr-4">Адрес</th>
                                    <th class="py-2 pr-4">Онлайн</th>
                                    <th class="py-2 pr-0 text-right">Открыть</th>
                                </tr>
                            </thead>
                            <tbody id="monTbody" class="text-slate-100"></tbody>
                        </table>
                    </div>

                    <div id="monEmpty" class="mt-4 hidden rounded-xl border border-dashed border-white/10 bg-black/10 p-6 text-center text-sm text-slate-300/70">
                        Нет серверов.
                    </div>
                </div>
            </div>
        </div>
    </section>

    <script>
        (function () {
            const csrf = document.querySelector('meta[name="csrf-token"]')?.getAttribute('content') || '';
            const dataUrl = "{{ route('monitoring.data') }}";

            const els = {
                tbody: document.getElementById('monTbody'),
                empty: document.getElementById('monEmpty'),
                err: document.getElementById('monError'),
                game: document.getElementById('monGame'),
                sort: document.getElementById('monSort'),
                search: document.getElementById('monSearch'),
                refresh: document.getElementById('monRefresh'),
            };

            let rawItems = [];

            function setErr(msg) {
                const text = String(msg || '').trim();
                if (!text) {
                    els.err?.classList.add('hidden');
                    if (els.err) els.err.textContent = '';
                    return;
                }
                if (els.err) {
                    els.err.textContent = text;
                    els.err.classList.remove('hidden');
                }
            }

            function esc(s) {
                return String(s || '')
                    .replaceAll('&', '&amp;')
                    .replaceAll('<', '&lt;')
                    .replaceAll('>', '&gt;')
                    .replaceAll('"', '&quot;')
                    .replaceAll("'", '&#039;');
            }

            function rebuildGameOptions(items) {
                const set = new Map();
                for (const it of items) {
                    const code = String(it.game || '');
                    const name = String(it.game_name || code);
                    if (!code) continue;
                    if (!set.has(code)) set.set(code, name);
                }

                const current = els.game?.value || '';
                if (!els.game) return;

                els.game.innerHTML = '<option value="">Все игры</option>';
                Array.from(set.entries())
                    .sort((a, b) => a[1].localeCompare(b[1]))
                    .forEach(([code, name]) => {
                        const opt = document.createElement('option');
                        opt.value = code;
                        opt.textContent = name;
                        els.game.appendChild(opt);
                    });

                els.game.value = current;
            }

            function applyFilters() {
                const game = String(els.game?.value || '');
                const q = String(els.search?.value || '').trim().toLowerCase();

                let items = rawItems.slice();

                if (game) {
                    items = items.filter((it) => String(it.game || '') === game);
                }

                if (q) {
                    items = items.filter((it) => {
                        const name = String(it.name || '').toLowerCase();
                        const addr = (String(it.ip || '') + ':' + String(it.port || '')).toLowerCase();
                        return name.includes(q) || addr.includes(q);
                    });
                }

                const sort = String(els.sort?.value || 'game');
                if (sort === 'name') {
                    items.sort((a, b) => String(a.name || '').localeCompare(String(b.name || '')));
                } else if (sort === 'online_desc') {
                    items.sort((a, b) => (Number(b.online ?? -1) - Number(a.online ?? -1)));
                } else if (sort === 'online_asc') {
                    items.sort((a, b) => (Number(a.online ?? 999999) - Number(b.online ?? 999999)));
                } else {
                    items.sort((a, b) => {
                        const ag = String(a.game_name || a.game || '');
                        const bg = String(b.game_name || b.game || '');
                        const c = ag.localeCompare(bg);
                        if (c !== 0) return c;
                        return String(a.name || '').localeCompare(String(b.name || ''));
                    });
                }

                return items;
            }

            function render() {
                const items = applyFilters();

                if (!els.tbody || !els.empty) return;

                els.tbody.innerHTML = '';

                if (items.length === 0) {
                    els.empty.classList.remove('hidden');
                    return;
                }
                els.empty.classList.add('hidden');

                for (const it of items) {
                    const online = (it.online === null || it.online === undefined) ? null : Number(it.online);
                    const max = (it.max === null || it.max === undefined) ? null : Number(it.max);
                    const onlineText = (online === null)
                        ? '<span class="text-slate-300/70">—</span>'
                        : `<span class="font-semibold text-slate-100">${online}</span>` + (max !== null ? `<span class="text-slate-300/70"> / ${max}</span>` : '');

                    const addr = `${esc(it.ip || '')}:${esc(it.port || '')}`;
                    const openUrl = `/servers/${encodeURIComponent(String(it.server_id || ''))}`;

                    const tr = document.createElement('tr');
                    tr.className = 'border-t border-white/10';
                    tr.innerHTML = `
                        <td class="py-3 pr-4 align-top">
                            <div class="text-sm font-semibold text-slate-100">${esc(it.name || '')}</div>
                            <div class="text-[12px] text-slate-300/70">${esc(it.runtime_status || '')}</div>
                        </td>
                        <td class="py-3 pr-4 align-top">
                            <div class="text-sm text-slate-100">${esc(it.game_name || it.game || '')}</div>
                            <div class="text-[12px] text-slate-300/70">${esc(it.game || '')}</div>
                        </td>
                        <td class="py-3 pr-4 align-top font-mono text-[12px]">${addr}</td>
                        <td class="py-3 pr-4 align-top">${onlineText}</td>
                        <td class="py-3 pr-0 align-top text-right">
                            <a href="${openUrl}" class="inline-flex items-center justify-center rounded-xl bg-sky-600 px-3 py-2 text-xs font-semibold text-white shadow-sm transition hover:bg-sky-700 active:scale-[0.99]">Открыть</a>
                        </td>
                    `;

                    els.tbody.appendChild(tr);
                }
            }

            async function load() {
                setErr('');
                try {
                    const resp = await fetch(dataUrl, {
                        method: 'POST',
                        headers: {
                            'Accept': 'application/json',
                            'Content-Type': 'application/json',
                            'X-CSRF-TOKEN': csrf,
                        },
                        body: JSON.stringify({}),
                    });
                    const data = await resp.json().catch(() => null);
                    if (!resp.ok || !data || data.ok !== true) {
                        throw new Error((data && data.error) ? data.error : ('HTTP ' + resp.status));
                    }

                    rawItems = Array.isArray(data.items) ? data.items : [];
                    rebuildGameOptions(rawItems);
                    render();
                } catch (e) {
                    setErr(e.message || String(e));
                    rawItems = [];
                    rebuildGameOptions(rawItems);
                    render();
                }
            }

            els.refresh?.addEventListener('click', load);
            els.game?.addEventListener('change', render);
            els.sort?.addEventListener('change', render);
            els.search?.addEventListener('input', render);

            load();
        })();
    </script>
@endsection
