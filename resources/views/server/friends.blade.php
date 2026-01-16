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
                $tab = 'friends';
                $isExpired = $server->expires_at && now()->greaterThan($server->expires_at);
                $isReinstalling = strtolower((string) ($server->provisioning_status ?? '')) === 'reinstalling';
                $isOwner = (int) ($server->user_id ?? 0) === (int) (auth()->id() ?? 0);
            @endphp

            <div class="overflow-hidden rounded-3xl bg-[#242f3d] text-slate-100 shadow-sm transition-shadow duration-300 hover:shadow-md">
                <div>
                    @include('server.partials.tabs')
                </div>
            </div>

            <div class="mt-6 overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] text-slate-100 shadow-sm transition-shadow duration-300 hover:shadow-md">
                <div class="border-b border-white/10 bg-black/10 px-5 py-3">
                    <div class="flex items-center justify-between gap-2">
                        <h2 class="text-sm font-semibold text-slate-100">Друзья</h2>
                        <span class="text-[11px] text-slate-300/70">Выдача доступа к управлению сервером</span>
                    </div>
                </div>
                <div class="p-5">
                    @if ($isExpired)
                        <div class="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100">
                            Срок аренды сервера истёк. Управление доступами недоступно.
                        </div>
                    @elseif ($isReinstalling)
                        <div class="rounded-2xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
                            Идёт переустановка. Управление доступами временно недоступно.
                        </div>
                    @elseif (! $isOwner)
                        <div class="rounded-2xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-200">
                            Эта вкладка доступна только владельцу сервера.
                        </div>
                    @else
                        <div class="grid gap-4">
                            <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                                <div class="text-sm font-semibold text-slate-100">Добавить друга</div>
                                <div class="mt-1 text-[12px] text-slate-300/70">Введите email пользователя, зарегистрированного в системе.</div>
                                <div class="mt-3 grid gap-3 md:grid-cols-3">
                                    <div class="md:col-span-2">
                                        <label class="block text-[11px] text-slate-300/70">Email</label>
                                        <input id="frEmail" type="email" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500" placeholder="user@example.com">
                                    </div>
                                    <div class="md:col-span-1 flex items-end">
                                        <button id="frAddBtn" type="button" class="w-full inline-flex items-center justify-center rounded-xl bg-emerald-600 px-4 py-2 text-xs font-semibold text-white shadow-sm transition hover:bg-emerald-700 active:scale-[0.99]">Добавить</button>
                                    </div>
                                </div>
                                <div id="frTopError" class="mt-3 hidden rounded-xl border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-xs text-rose-100"></div>
                            </div>

                            <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                                <div class="flex items-center justify-between">
                                    <div class="text-sm font-semibold text-slate-100">Доступы</div>
                                    <button id="frRefreshBtn" type="button" class="text-xs font-semibold text-slate-200 hover:text-white">Обновить</button>
                                </div>
                                <div class="mt-1 text-[12px] text-slate-300/70">Права применяются сразу после сохранения. Запрещённые действия будут возвращать 403.</div>
                                <div id="frListError" class="mt-3 hidden rounded-xl border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-xs text-rose-100"></div>

                                <div class="mt-3 overflow-x-auto">
                                    <table class="min-w-full text-left text-sm">
                                        <thead class="text-[11px] text-slate-300/70">
                                            <tr>
                                                <th class="py-2 pr-4">Пользователь</th>
                                                <th class="py-2 pr-4">Вкладки</th>
                                                <th class="py-2 pr-4">Действия</th>
                                                <th class="py-2 pr-0 text-right">Удалить</th>
                                            </tr>
                                        </thead>
                                        <tbody id="frTbody" class="text-slate-100"></tbody>
                                    </table>
                                </div>

                                <div id="frEmpty" class="mt-4 rounded-xl border border-dashed border-white/10 bg-black/10 p-4 text-center text-[12px] text-slate-300/70 hidden">
                                    Нет добавленных друзей.
                                </div>
                            </div>
                        </div>

                        <script>
                            (function () {
                                const csrf = document.querySelector('meta[name="csrf-token"]')?.getAttribute('content') || '';
                                const listUrl = "{{ route('server.friends.list', $server) }}";
                                const addUrl = "{{ route('server.friends.add', $server) }}";
                                const updateUrl = "{{ route('server.friends.update', $server) }}";
                                const delUrl = "{{ route('server.friends.delete', $server) }}";

                                const tbody = document.getElementById('frTbody');
                                const emptyEl = document.getElementById('frEmpty');

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

                                function toggle(field, label, checked) {
                                    return `
                                        <label class="inline-flex items-center gap-2 rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[12px] text-slate-200">
                                            <input type="checkbox" data-field="${esc(field)}" class="h-4 w-4 rounded border-white/10 bg-black/10" ${checked ? 'checked' : ''}>
                                            <span>${esc(label)}</span>
                                        </label>
                                    `;
                                }

                                function render(items) {
                                    const arr = Array.isArray(items) ? items : [];
                                    tbody.innerHTML = '';

                                    if (arr.length === 0) {
                                        emptyEl.classList.remove('hidden');
                                        return;
                                    }
                                    emptyEl.classList.add('hidden');

                                    for (const it of arr) {
                                        const uid = Number(it.user_id || 0);
                                        const email = String(it.user_email || '');
                                        const name = String((it.user_name || '') + ' ' + (it.user_last_name || '')).trim();

                                        const tabs = [
                                            ['can_view_console', 'Консоль', !!it.can_view_console],
                                            ['can_view_logs', 'Логи', !!it.can_view_logs],
                                            ['can_view_metrics', 'Метрики', !!it.can_view_metrics],
                                            ['can_view_ftp', 'FTP', !!it.can_view_ftp],
                                            ['can_view_mysql', 'MySQL', !!it.can_view_mysql],
                                            ['can_view_cron', 'Планировщик', !!it.can_view_cron],
                                            ['can_view_firewall', 'Firewall', !!it.can_view_firewall],
                                            ['can_view_settings', 'Настройки', !!it.can_view_settings],
                                        ];
                                        const actions = [
                                            ['can_start', 'Запуск', !!it.can_start],
                                            ['can_stop', 'Остановка', !!it.can_stop],
                                            ['can_restart', 'Рестарт', !!it.can_restart],
                                            ['can_reinstall', 'Переустановка', !!it.can_reinstall],
                                            ['can_console_command', 'Команды в консоль', !!it.can_console_command],
                                            ['can_files', 'Файловый менеджер', !!it.can_files],
                                            ['can_cron_manage', 'Управлять планировщиком', !!it.can_cron_manage],
                                            ['can_firewall_manage', 'Управлять firewall', !!it.can_firewall_manage],
                                            ['can_settings_edit', 'Редактировать настройки', !!it.can_settings_edit],
                                        ];

                                        const tabsHtml = `
                                            <div class="grid gap-2 md:grid-cols-2">
                                                ${tabs.map(([f, l, v]) => toggle(f, l, v)).join('')}
                                            </div>
                                        `;
                                        const actionsHtml = `
                                            <div class="grid gap-2 md:grid-cols-2">
                                                ${actions.map(([f, l, v]) => toggle(f, l, v)).join('')}
                                            </div>
                                        `;

                                        const tr = document.createElement('tr');
                                        tr.className = 'border-t border-white/10';
                                        tr.innerHTML = `
                                            <td class="py-3 pr-4 align-top">
                                                <div class="text-sm font-semibold text-slate-100">${esc(name || email)}</div>
                                                <div class="text-[12px] text-slate-300/70">${esc(email)}</div>
                                            </td>
                                            <td class="py-3 pr-4 align-top">
                                                <div data-uid="${uid}" class="frPerms">${tabsHtml}</div>
                                            </td>
                                            <td class="py-3 pr-4 align-top">
                                                <div data-uid="${uid}" class="frPerms">${actionsHtml}</div>
                                                <div class="mt-2">
                                                    <button type="button" class="frSave inline-flex items-center justify-center rounded-xl bg-sky-600 px-3 py-2 text-xs font-semibold text-white shadow-sm transition hover:bg-sky-700 active:scale-[0.99]" data-uid="${uid}">Сохранить</button>
                                                </div>
                                            </td>
                                            <td class="py-3 pr-0 align-top text-right">
                                                <button type="button" class="frDelete text-xs font-semibold text-rose-200 hover:text-rose-100" data-uid="${uid}">Удалить</button>
                                            </td>
                                        `;
                                        tbody.appendChild(tr);
                                    }

                                    tbody.querySelectorAll('.frSave').forEach((btn) => {
                                        btn.addEventListener('click', async (e) => {
                                            setErr(document.getElementById('frListError'), '');
                                            const uid = Number(e.target.getAttribute('data-uid') || 0);
                                            const payload = { user_id: uid };
                                            const blocks = tbody.querySelectorAll(`.frPerms[data-uid="${uid}"]`);
                                            blocks.forEach((b) => {
                                                b.querySelectorAll('input[type="checkbox"]').forEach((ch) => {
                                                    const field = ch.getAttribute('data-field');
                                                    if (field) payload[field] = !!ch.checked;
                                                });
                                            });
                                            try {
                                                const data = await post(updateUrl, payload);
                                                render(data.items);
                                            } catch (err) {
                                                setErr(document.getElementById('frListError'), err.message || String(err));
                                            }
                                        });
                                    });

                                    tbody.querySelectorAll('.frDelete').forEach((btn) => {
                                        btn.addEventListener('click', async (e) => {
                                            setErr(document.getElementById('frListError'), '');
                                            const uid = Number(e.target.getAttribute('data-uid') || 0);
                                            try {
                                                const data = await post(delUrl, { user_id: uid });
                                                render(data.items);
                                            } catch (err) {
                                                setErr(document.getElementById('frListError'), err.message || String(err));
                                            }
                                        });
                                    });
                                }

                                async function refresh() {
                                    setErr(document.getElementById('frListError'), '');
                                    try {
                                        const data = await post(listUrl, {});
                                        render(data.items);
                                    } catch (err) {
                                        setErr(document.getElementById('frListError'), err.message || String(err));
                                    }
                                }

                                document.getElementById('frRefreshBtn')?.addEventListener('click', refresh);

                                document.getElementById('frAddBtn')?.addEventListener('click', async () => {
                                    setErr(document.getElementById('frTopError'), '');
                                    const email = document.getElementById('frEmail')?.value || '';
                                    try {
                                        const data = await post(addUrl, { email });
                                        render(data.items);
                                        if (document.getElementById('frEmail')) document.getElementById('frEmail').value = '';
                                    } catch (err) {
                                        setErr(document.getElementById('frTopError'), err.message || String(err));
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
