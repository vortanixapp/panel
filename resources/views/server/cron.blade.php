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
                $tab = 'cron';
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
                        <h2 class="text-sm font-semibold text-slate-100">Крон</h2>
                        <span class="text-[11px] text-slate-300/70">Планировщик задач</span>
                    </div>
                </div>
                <div class="p-5">
                    @if ($isExpired)
                        <div class="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100">
                            Срок аренды сервера истёк. Управление планировщиком недоступно.
                        </div>
                    @elseif ($isReinstalling)
                        <div class="rounded-2xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100">
                            Идёт переустановка. Управление планировщиком временно недоступно.
                        </div>
                    @else
                        <div class="grid gap-4">
                            <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                                <div class="text-sm font-semibold text-slate-100">Добавить задачу</div>
                                <div class="mt-3 grid gap-3 md:grid-cols-4">
                                    <div class="md:col-span-1">
                                        <label class="block text-[11px] text-slate-300/70">Название (опционально)</label>
                                        <input id="cronName" type="text" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500" placeholder="Напр. Backup">
                                    </div>
                                    <div class="md:col-span-1">
                                        <label class="block text-[11px] text-slate-300/70">Расписание</label>
                                        <select id="cronPreset" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                                            <option value="every_minute">Каждую минуту</option>
                                            <option value="every_n_minutes">Каждые N минут</option>
                                            <option value="hourly">Каждый час</option>
                                            <option value="daily">Каждый день</option>
                                            <option value="weekly">Каждую неделю</option>
                                            <option value="monthly">Каждый месяц</option>
                                        </select>
                                        <input id="cronSchedule" type="hidden" value="">
                                    </div>
                                    <div class="md:col-span-2">
                                        <label class="block text-[11px] text-slate-300/70">Команда</label>
                                        <input id="cronCommand" type="text" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500" placeholder="echo hello > /data/cron.log">
                                    </div>
                                </div>

                                <div class="mt-3 grid gap-3 md:grid-cols-4">
                                    <div id="cronEveryNWrap" class="md:col-span-1 hidden">
                                        <label class="block text-[11px] text-slate-300/70">Каждые N минут</label>
                                        <input id="cronEveryN" type="number" min="1" max="59" value="5" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                                    </div>
                                    <div id="cronMinuteWrap" class="md:col-span-1 hidden">
                                        <label class="block text-[11px] text-slate-300/70">Минута</label>
                                        <input id="cronMinute" type="number" min="0" max="59" value="0" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                                    </div>
                                    <div id="cronTimeWrap" class="md:col-span-1 hidden">
                                        <label class="block text-[11px] text-slate-300/70">Время</label>
                                        <input id="cronTime" type="time" value="00:00" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                                    </div>
                                    <div id="cronWeekdayWrap" class="md:col-span-1 hidden">
                                        <label class="block text-[11px] text-slate-300/70">День недели</label>
                                        <select id="cronWeekday" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                                            <option value="1">Пн</option>
                                            <option value="2">Вт</option>
                                            <option value="3">Ср</option>
                                            <option value="4">Чт</option>
                                            <option value="5">Пт</option>
                                            <option value="6">Сб</option>
                                            <option value="0">Вс</option>
                                        </select>
                                    </div>
                                    <div id="cronMonthdayWrap" class="md:col-span-1 hidden">
                                        <label class="block text-[11px] text-slate-300/70">День месяца</label>
                                        <input id="cronMonthday" type="number" min="1" max="31" value="1" class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                                    </div>
                                    <div class="md:col-span-2">
                                        <label class="block text-[11px] text-slate-300/70">Cron (авто)</label>
                                        <input id="cronPreview" type="text" readonly class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 font-mono text-[12px] text-slate-100 shadow-sm opacity-90" value="* * * * *">
                                    </div>
                                </div>
                                <div class="mt-3 flex items-center justify-between gap-3">
                                    <label class="inline-flex items-center gap-2 text-sm text-slate-200">
                                        <input id="cronEnabled" type="checkbox" class="h-4 w-4 rounded border-white/10 bg-black/10" checked>
                                        <span>Включено</span>
                                    </label>

                                    <button id="cronCreateBtn" type="button" class="inline-flex items-center justify-center rounded-xl bg-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-sm transition hover:bg-sky-700 active:scale-[0.99]">
                                        Добавить
                                    </button>
                                </div>
                                <div id="cronCreateError" class="mt-3 hidden rounded-xl border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-xs text-rose-100"></div>
                            </div>

                            <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                                <div class="flex items-center justify-between">
                                    <div class="text-sm font-semibold text-slate-100">Список задач</div>
                                    <button id="cronRefreshBtn" type="button" class="text-xs font-semibold text-slate-200 hover:text-white">Обновить</button>
                                </div>
                                <div id="cronListError" class="mt-3 hidden rounded-xl border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-xs text-rose-100"></div>

                                <div class="mt-3 overflow-x-auto">
                                    <table class="min-w-full text-left text-sm">
                                        <thead class="text-[11px] text-slate-300/70">
                                            <tr>
                                                <th class="py-2 pr-4">Вкл</th>
                                                <th class="py-2 pr-4">Расписание</th>
                                                <th class="py-2 pr-4">Команда</th>
                                                <th class="py-2 pr-4">Название</th>
                                                <th class="py-2 pr-0 text-right">Действия</th>
                                            </tr>
                                        </thead>
                                        <tbody id="cronTbody" class="text-slate-100"></tbody>
                                    </table>
                                </div>

                                <div id="cronEmpty" class="mt-4 rounded-xl border border-dashed border-white/10 bg-black/10 p-4 text-center text-[12px] text-slate-300/70 hidden">
                                    Нет задач.
                                </div>
                            </div>
                        </div>

                        <script>
                            (function () {
                                const csrf = document.querySelector('meta[name="csrf-token"]')?.getAttribute('content') || '';
                                const listUrl = "{{ route('server.cron.list', $server) }}";
                                const createUrl = "{{ route('server.cron.create', $server) }}";
                                const delUrl = "{{ route('server.cron.delete', $server) }}";
                                const toggleUrl = "{{ route('server.cron.toggle', $server) }}";

                                const tbody = document.getElementById('cronTbody');
                                const emptyEl = document.getElementById('cronEmpty');
                                const errList = document.getElementById('cronListError');
                                const errCreate = document.getElementById('cronCreateError');

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

                                function renderJobs(jobs) {
                                    if (!tbody || !emptyEl) return;
                                    tbody.innerHTML = '';
                                    const arr = Array.isArray(jobs) ? jobs : [];
                                    if (arr.length === 0) {
                                        emptyEl.classList.remove('hidden');
                                        return;
                                    }
                                    emptyEl.classList.add('hidden');

                                    for (const j of arr) {
                                        const id = String(j.id || '');
                                        const enabled = !!j.enabled;
                                        const schedule = String(j.schedule || '');
                                        const command = String(j.command || '');
                                        const name = String(j.name || '');

                                        const tr = document.createElement('tr');
                                        tr.className = 'border-t border-white/10';
                                        tr.innerHTML = `
                                            <td class="py-2 pr-4 align-top">
                                                <input type="checkbox" class="cronToggle h-4 w-4 rounded border-white/10 bg-black/10" data-id="${esc(id)}" ${enabled ? 'checked' : ''} />
                                            </td>
                                            <td class="py-2 pr-4 align-top font-mono text-[12px]">${esc(schedule)}</td>
                                            <td class="py-2 pr-4 align-top font-mono text-[12px]">${esc(command)}</td>
                                            <td class="py-2 pr-4 align-top text-[12px]">${esc(name)}</td>
                                            <td class="py-2 pr-0 align-top text-right">
                                                <button type="button" class="cronDelete text-xs font-semibold text-rose-200 hover:text-rose-100" data-id="${esc(id)}">Удалить</button>
                                            </td>
                                        `;
                                        tbody.appendChild(tr);
                                    }

                                    tbody.querySelectorAll('.cronToggle').forEach((el) => {
                                        el.addEventListener('change', async (e) => {
                                            setErr(errList, '');
                                            const id = e.target.getAttribute('data-id') || '';
                                            const enabled = !!e.target.checked;
                                            try {
                                                const data = await post(toggleUrl, { job_id: id, enabled });
                                                renderJobs(data.jobs || []);
                                            } catch (err) {
                                                setErr(errList, err.message || String(err));
                                            }
                                        });
                                    });

                                    tbody.querySelectorAll('.cronDelete').forEach((el) => {
                                        el.addEventListener('click', async (e) => {
                                            setErr(errList, '');
                                            const id = e.target.getAttribute('data-id') || '';
                                            try {
                                                const data = await post(delUrl, { job_id: id });
                                                renderJobs(data.jobs || []);
                                            } catch (err) {
                                                setErr(errList, err.message || String(err));
                                            }
                                        });
                                    });
                                }

                                async function refresh() {
                                    setErr(errList, '');
                                    try {
                                        const data = await post(listUrl, {});
                                        renderJobs(data.jobs || []);
                                    } catch (err) {
                                        setErr(errList, err.message || String(err));
                                    }
                                }

                                document.getElementById('cronRefreshBtn')?.addEventListener('click', refresh);

                                function clampInt(v, min, max, defVal) {
                                    const n = parseInt(String(v || ''), 10);
                                    if (Number.isNaN(n)) return defVal;
                                    return Math.max(min, Math.min(max, n));
                                }

                                function getTimeParts() {
                                    const raw = document.getElementById('cronTime')?.value || '00:00';
                                    const m = String(raw).match(/^(\d{1,2}):(\d{1,2})$/);
                                    if (!m) return { hh: 0, mm: 0 };
                                    return {
                                        hh: clampInt(m[1], 0, 23, 0),
                                        mm: clampInt(m[2], 0, 59, 0),
                                    };
                                }

                                function computeSchedule() {
                                    const preset = document.getElementById('cronPreset')?.value || 'every_minute';

                                    const wrapEveryN = document.getElementById('cronEveryNWrap');
                                    const wrapMinute = document.getElementById('cronMinuteWrap');
                                    const wrapTime = document.getElementById('cronTimeWrap');
                                    const wrapWeekday = document.getElementById('cronWeekdayWrap');
                                    const wrapMonthday = document.getElementById('cronMonthdayWrap');

                                    function show(el, on) {
                                        if (!el) return;
                                        if (on) el.classList.remove('hidden');
                                        else el.classList.add('hidden');
                                    }

                                    show(wrapEveryN, preset === 'every_n_minutes');
                                    show(wrapMinute, preset === 'hourly');
                                    show(wrapTime, preset === 'daily' || preset === 'weekly' || preset === 'monthly');
                                    show(wrapWeekday, preset === 'weekly');
                                    show(wrapMonthday, preset === 'monthly');

                                    let schedule = '* * * * *';

                                    if (preset === 'every_minute') {
                                        schedule = '* * * * *';
                                    } else if (preset === 'every_n_minutes') {
                                        const n = clampInt(document.getElementById('cronEveryN')?.value, 1, 59, 5);
                                        schedule = '*/' + n + ' * * * *';
                                    } else if (preset === 'hourly') {
                                        const mm = clampInt(document.getElementById('cronMinute')?.value, 0, 59, 0);
                                        schedule = String(mm) + ' * * * *';
                                    } else if (preset === 'daily') {
                                        const t = getTimeParts();
                                        schedule = String(t.mm) + ' ' + String(t.hh) + ' * * *';
                                    } else if (preset === 'weekly') {
                                        const t = getTimeParts();
                                        const dow = clampInt(document.getElementById('cronWeekday')?.value, 0, 6, 1);
                                        schedule = String(t.mm) + ' ' + String(t.hh) + ' * * ' + String(dow);
                                    } else if (preset === 'monthly') {
                                        const t = getTimeParts();
                                        const dom = clampInt(document.getElementById('cronMonthday')?.value, 1, 31, 1);
                                        schedule = String(t.mm) + ' ' + String(t.hh) + ' ' + String(dom) + ' * *';
                                    }

                                    const hidden = document.getElementById('cronSchedule');
                                    if (hidden) hidden.value = schedule;
                                    const prev = document.getElementById('cronPreview');
                                    if (prev) prev.value = schedule;
                                }

                                ['cronPreset', 'cronEveryN', 'cronMinute', 'cronTime', 'cronWeekday', 'cronMonthday'].forEach((id) => {
                                    document.getElementById(id)?.addEventListener('change', computeSchedule);
                                    document.getElementById(id)?.addEventListener('input', computeSchedule);
                                });

                                computeSchedule();

                                document.getElementById('cronCreateBtn')?.addEventListener('click', async () => {
                                    setErr(errCreate, '');
                                    const name = document.getElementById('cronName')?.value || '';
                                    computeSchedule();
                                    const schedule = document.getElementById('cronSchedule')?.value || '';
                                    const command = document.getElementById('cronCommand')?.value || '';
                                    const enabled = !!document.getElementById('cronEnabled')?.checked;
                                    try {
                                        const data = await post(createUrl, { name, schedule, command, enabled });
                                        renderJobs(data.jobs || []);
                                        if (document.getElementById('cronCommand')) document.getElementById('cronCommand').value = '';
                                    } catch (err) {
                                        setErr(errCreate, err.message || String(err));
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
