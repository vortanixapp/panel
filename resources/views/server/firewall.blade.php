@extends($layout ?? 'layouts.app-user')

@section('title', $server->name)

@section('breadcrumb')
    <a href="{{ route('dashboard') }}" class="text-xs text-slate-200 hover:text-white">Кабинет</a>
    <span class="h-1 w-1 rounded-full bg-white/25"></span>
    <a href="{{ route('my-servers') }}" class="text-xs text-slate-200 hover:text-white">Мои серверы</a>
    <span class="h-1 w-1 rounded-full bg-white/25"></span>
    <span class="text-xs text-slate-100">{{ $server->name }}</span>
@endsection

@section('content')
    <section class="py-6">
        <div class="w-full px-4 sm:px-6 lg:px-8">
            @php
                $tab = 'firewall';
                $isExpired = $server->expires_at && now()->greaterThan($server->expires_at);
                $isReinstalling = strtolower((string) ($server->provisioning_status ?? '')) === 'reinstalling';
            @endphp

            <div class="overflow-hidden rounded-3xl bg-[#242f3d] text-slate-100 shadow-sm transition-shadow duration-300 hover:shadow-md">
                <div>
                    @include('server.partials.tabs')
                </div>
            </div>

            <div class="mt-6 overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] text-slate-100 shadow-sm transition-shadow duration-300 hover:shadow-md">
                <div class="border-b border-white/10 bg-black/10 px-5 py-3">
                    <div class="flex items-center justify-between gap-2">
                        <h2 class="text-sm font-semibold text-slate-100">Firewall</h2>
                        <span class="text-[11px] text-slate-300/70">Ограничение доступа по IP (на локации)</span>
                    </div>
                </div>
                <div class="p-5">
                    @if ($isExpired)
                        <div class="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100">
                            Срок аренды сервера истёк. Управление Firewall недоступно.
                        </div>
                    @elseif ($isReinstalling)
                        <div class="rounded-2xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
                            Идёт переустановка. Управление Firewall временно недоступно.
                        </div>
                    @else
                        <div class="grid gap-4">
                            <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                                <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
                                    <div class="grid gap-2">
                                        <div class="text-sm font-semibold text-slate-100">Режим</div>
                                        <div class="text-[12px] text-slate-300/70">allow = доступ только для указанных IP (остальные будут заблокированы). deny = блокировать указанные IP.</div>
                                    </div>
                                    <div class="flex items-center gap-3">
                                        <label class="inline-flex items-center gap-2 text-sm text-slate-200">
                                            <input id="fwEnabled" type="checkbox" class="h-4 w-4 rounded border-white/10 bg-black/10">
                                            <span>Включено</span>
                                        </label>
                                        <select id="fwMode" class="rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                                            <option value="allow">allow</option>
                                            <option value="deny">deny</option>
                                        </select>
                                        <button id="fwSaveBtn" type="button" class="inline-flex items-center justify-center rounded-xl bg-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-sm transition hover:bg-sky-700 active:scale-[0.99]">Сохранить</button>
                                    </div>
                                </div>
                                <div id="fwTopError" class="mt-3 hidden rounded-xl border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-xs text-rose-100"></div>
                            </div>

                            <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                                <div class="text-sm font-semibold text-slate-100">Добавить правило</div>
                                <div class="mt-3 grid gap-3 md:grid-cols-4">
                                    <div class="md:col-span-1">
                                        <label class="block text-[11px] text-slate-300/70">CIDR (IPv4)</label>
                                        <input id="fwCidr" type="text" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500" placeholder="1.2.3.4/32">
                                    </div>
                                    <div class="md:col-span-1">
                                        <label class="block text-[11px] text-slate-300/70">Протокол</label>
                                        <select id="fwProto" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                                            <option value="both">both</option>
                                            <option value="udp">udp</option>
                                            <option value="tcp">tcp</option>
                                        </select>
                                    </div>
                                    <div class="md:col-span-2">
                                        <label class="block text-[11px] text-slate-300/70">Комментарий (опционально)</label>
                                        <input id="fwNote" type="text" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500" placeholder="Напр. Админ">
                                    </div>
                                </div>
                                <div class="mt-3 flex items-center justify-between gap-3">
                                    <label class="inline-flex items-center gap-2 text-sm text-slate-200">
                                        <input id="fwRuleEnabled" type="checkbox" class="h-4 w-4 rounded border-white/10 bg-black/10" checked>
                                        <span>Включено</span>
                                    </label>
                                    <button id="fwAddBtn" type="button" class="inline-flex items-center justify-center rounded-xl bg-emerald-600 px-4 py-2 text-xs font-semibold text-white shadow-sm transition hover:bg-emerald-700 active:scale-[0.99]">Добавить</button>
                                </div>
                                <div id="fwAddError" class="mt-3 hidden rounded-xl border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-xs text-rose-100"></div>
                            </div>

                            <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                                <div class="flex items-center justify-between">
                                    <div class="text-sm font-semibold text-slate-100">Правила</div>
                                    <button id="fwRefreshBtn" type="button" class="text-xs font-semibold text-slate-200 hover:text-white">Обновить</button>
                                </div>
                                <div id="fwListError" class="mt-3 hidden rounded-xl border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-xs text-rose-100"></div>

                                <div class="mt-3 overflow-x-auto">
                                    <table class="min-w-full text-left text-sm">
                                        <thead class="text-[11px] text-slate-300/70">
                                            <tr>
                                                <th class="py-2 pr-4">Вкл</th>
                                                <th class="py-2 pr-4">CIDR</th>
                                                <th class="py-2 pr-4">Proto</th>
                                                <th class="py-2 pr-4">Комментарий</th>
                                                <th class="py-2 pr-0 text-right">Действия</th>
                                            </tr>
                                        </thead>
                                        <tbody id="fwTbody" class="text-slate-100"></tbody>
                                    </table>
                                </div>

                                <div id="fwEmpty" class="mt-4 rounded-xl border border-dashed border-white/10 bg-black/10 p-4 text-center text-[12px] text-slate-300/70 hidden">
                                    Нет правил.
                                </div>
                            </div>
                        </div>

                        <script>
                            (function () {
                                const csrf = document.querySelector('meta[name="csrf-token"]')?.getAttribute('content') || '';
                                const listUrl = "{{ route('server.firewall.list', $server) }}";
                                const setUrl = "{{ route('server.firewall.set', $server) }}";
                                const toggleUrl = "{{ route('server.firewall.toggle', $server) }}";
                                const addUrl = "{{ route('server.firewall.add-rule', $server) }}";
                                const delUrl = "{{ route('server.firewall.delete-rule', $server) }}";
                                const toggleRuleUrl = "{{ route('server.firewall.toggle-rule', $server) }}";

                                const tbody = document.getElementById('fwTbody');
                                const emptyEl = document.getElementById('fwEmpty');

                                function setErr(el, msg) {
                                    if (!el) return;
                                    const text = String(msg || '').trim();
                                    if (!text) {
                                        el.classList.add('hidden');
                                        el.textContent = '';
                                        return;
                                    }
                                    el.textContent = text;
                                    el.classList.remove('hidden');
                                }

                                async function post(url, payload) {
                                    const resp = await fetch(url, {
                                        method: 'POST',
                                        headers: {
                                            'Accept': 'application/json',
                                            'Content-Type': 'application/json',
                                            'X-CSRF-TOKEN': csrf,
                                        },
                                        body: JSON.stringify(payload || {}),
                                    });
                                    const data = await resp.json().catch(() => null);
                                    if (!resp.ok || !data) {
                                        throw new Error((data && data.error) ? data.error : ('HTTP ' + resp.status));
                                    }
                                    if (data.ok !== true) {
                                        throw new Error(String(data.error || 'Unknown error'));
                                    }
                                    return data;
                                }

                                function esc(s) {
                                    return String(s || '')
                                        .replaceAll('&', '&amp;')
                                        .replaceAll('<', '&lt;')
                                        .replaceAll('>', '&gt;')
                                        .replaceAll('"', '&quot;')
                                        .replaceAll("'", '&#039;');
                                }

                                function render(state) {
                                    const rules = Array.isArray(state?.rules) ? state.rules : [];
                                    const enabled = !!state?.enabled;
                                    const mode = String(state?.mode || 'allow');

                                    const enabledEl = document.getElementById('fwEnabled');
                                    const modeEl = document.getElementById('fwMode');
                                    if (enabledEl) enabledEl.checked = enabled;
                                    if (modeEl) modeEl.value = (mode === 'deny') ? 'deny' : 'allow';

                                    if (!tbody || !emptyEl) return;
                                    tbody.innerHTML = '';

                                    if (rules.length === 0) {
                                        emptyEl.classList.remove('hidden');
                                        return;
                                    }
                                    emptyEl.classList.add('hidden');

                                    for (const r of rules) {
                                        const id = String(r.id || '');
                                        const cidr = String(r.cidr || '');
                                        const proto = String(r.proto || 'both');
                                        const note = String(r.note || '');
                                        const ren = !!r.enabled;

                                        const tr = document.createElement('tr');
                                        tr.className = 'border-t border-white/10';
                                        tr.innerHTML = `
                                            <td class="py-2 pr-4 align-top">
                                                <input type="checkbox" class="fwToggleRule h-4 w-4 rounded border-white/10 bg-black/10" data-id="${esc(id)}" ${ren ? 'checked' : ''} />
                                            </td>
                                            <td class="py-2 pr-4 align-top font-mono text-[12px]">${esc(cidr)}</td>
                                            <td class="py-2 pr-4 align-top text-[12px]">${esc(proto)}</td>
                                            <td class="py-2 pr-4 align-top text-[12px]">${esc(note)}</td>
                                            <td class="py-2 pr-0 align-top text-right">
                                                <button type="button" class="fwDelete text-xs font-semibold text-rose-200 hover:text-rose-100" data-id="${esc(id)}">Удалить</button>
                                            </td>
                                        `;
                                        tbody.appendChild(tr);
                                    }

                                    tbody.querySelectorAll('.fwToggleRule').forEach((el) => {
                                        el.addEventListener('change', async (e) => {
                                            setErr(document.getElementById('fwListError'), '');
                                            const id = e.target.getAttribute('data-id') || '';
                                            const enabled = !!e.target.checked;
                                            try {
                                                const data = await post(toggleRuleUrl, { rule_id: id, enabled });
                                                render(data);
                                            } catch (err) {
                                                setErr(document.getElementById('fwListError'), err.message || String(err));
                                            }
                                        });
                                    });

                                    tbody.querySelectorAll('.fwDelete').forEach((el) => {
                                        el.addEventListener('click', async (e) => {
                                            setErr(document.getElementById('fwListError'), '');
                                            const id = e.target.getAttribute('data-id') || '';
                                            try {
                                                const data = await post(delUrl, { rule_id: id });
                                                render(data);
                                            } catch (err) {
                                                setErr(document.getElementById('fwListError'), err.message || String(err));
                                            }
                                        });
                                    });
                                }

                                async function refresh() {
                                    setErr(document.getElementById('fwListError'), '');
                                    try {
                                        const data = await post(listUrl, {});
                                        render(data);
                                    } catch (err) {
                                        setErr(document.getElementById('fwListError'), err.message || String(err));
                                    }
                                }

                                document.getElementById('fwRefreshBtn')?.addEventListener('click', refresh);

                                document.getElementById('fwSaveBtn')?.addEventListener('click', async () => {
                                    setErr(document.getElementById('fwTopError'), '');
                                    const enabled = !!document.getElementById('fwEnabled')?.checked;
                                    const mode = document.getElementById('fwMode')?.value || 'allow';
                                    try {
                                        const data = await post(setUrl, { enabled, mode });
                                        render(data);
                                    } catch (err) {
                                        setErr(document.getElementById('fwTopError'), err.message || String(err));
                                    }
                                });

                                document.getElementById('fwEnabled')?.addEventListener('change', async () => {
                                    setErr(document.getElementById('fwTopError'), '');
                                    const enabled = !!document.getElementById('fwEnabled')?.checked;
                                    try {
                                        const data = await post(toggleUrl, { enabled });
                                        render(data);
                                    } catch (err) {
                                        setErr(document.getElementById('fwTopError'), err.message || String(err));
                                    }
                                });

                                document.getElementById('fwAddBtn')?.addEventListener('click', async () => {
                                    setErr(document.getElementById('fwAddError'), '');
                                    const cidr = document.getElementById('fwCidr')?.value || '';
                                    const proto = document.getElementById('fwProto')?.value || 'both';
                                    const note = document.getElementById('fwNote')?.value || '';
                                    const enabled = !!document.getElementById('fwRuleEnabled')?.checked;
                                    try {
                                        const data = await post(addUrl, { cidr, proto, note, enabled });
                                        render(data);
                                        if (document.getElementById('fwCidr')) document.getElementById('fwCidr').value = '';
                                    } catch (err) {
                                        setErr(document.getElementById('fwAddError'), err.message || String(err));
                                    }
                                });

                                refresh();
                            })();
                        </script>
                    @endif
                </div>
            </div>
        </div>
    </section>
@endsection
