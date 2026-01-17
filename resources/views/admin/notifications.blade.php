@extends('layouts.app-admin')

@section('page_title', 'Уведомления')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div>
                <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Уведомления</h1>
                <p class="mt-1 text-sm text-slate-300/80">Список уведомлений из биллинга.</p>
            </div>

            <div class="rounded-2xl border border-white/10 bg-[#242f3d] shadow-sm shadow-black/20 overflow-hidden">
                <div class="flex items-center justify-between gap-2 px-4 py-3 border-b border-white/10 bg-black/10">
                    <div class="text-sm font-semibold text-slate-100">Все уведомления</div>
                    <button
                        type="button"
                        class="inline-flex h-9 items-center justify-center rounded-xl border border-white/10 bg-black/10 px-4 text-xs font-semibold text-slate-200 hover:bg-black/15 hover:text-white"
                        onclick="window.vtxNotifications && window.vtxNotifications.reloadAll && window.vtxNotifications.reloadAll()"
                    >
                        Обновить
                    </button>
                </div>

                <div class="overflow-x-auto">
                    <table class="min-w-[1000px] w-full table-fixed divide-y divide-white/10 text-sm">
                        <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                            <tr>
                                <th class="w-20 px-4 py-3 text-left">ID</th>
                                <th class="w-28 px-4 py-3 text-left">Уровень</th>
                                <th class="w-72 px-4 py-3 text-left">Заголовок</th>
                                <th class="w-[520px] px-4 py-3 text-left">Текст</th>
                                <th class="w-44 px-4 py-3 text-left">Создано</th>
                                <th class="w-28 px-4 py-3 text-right">Действие</th>
                            </tr>
                        </thead>
                        <tbody id="notificationsTableBody" class="divide-y divide-white/10 bg-[#242f3d] text-[13px]">
                            <tr>
                                <td colspan="6" class="px-4 py-8 text-center text-slate-300/70">Загрузка…</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </section>

    <div id="vtxNotifModal" class="fixed inset-0 z-[60] hidden">
        <div id="vtxNotifModalBackdrop" class="absolute inset-0 bg-black/60"></div>
        <div class="relative mx-auto mt-16 w-[calc(100%-2rem)] max-w-3xl">
            <div class="rounded-2xl border border-white/10 bg-[#242f3d] shadow-xl shadow-black/40">
                <div class="flex items-center justify-between gap-3 border-b border-white/10 px-4 py-3">
                    <div id="vtxNotifModalTitle" class="text-sm font-semibold text-slate-100">Уведомление</div>
                    <button id="vtxNotifModalClose" type="button" class="inline-flex h-9 items-center justify-center rounded-xl border border-white/10 bg-black/10 px-4 text-xs font-semibold text-slate-200 hover:bg-black/15 hover:text-white">
                        Закрыть
                    </button>
                </div>
                <div class="px-4 py-4">
                    <div id="vtxNotifModalBody" class="max-h-[70vh] overflow-auto whitespace-pre-wrap break-words text-sm text-slate-200"></div>
                </div>
            </div>
        </div>
    </div>

    <style>
        .vtx-clamp-3 {
            display: -webkit-box;
            -webkit-line-clamp: 3;
            -webkit-box-orient: vertical;
            overflow: hidden;
        }
        .vtx-wrap-anywhere {
            overflow-wrap: anywhere;
            word-break: break-word;
        }
    </style>

    @push('scripts')
        <script>
            (function () {
                const csrf = document.querySelector('meta[name="csrf-token"]')?.getAttribute('content') || '';
                const tbody = document.getElementById('notificationsTableBody');

                const modal = document.getElementById('vtxNotifModal');
                const modalBackdrop = document.getElementById('vtxNotifModalBackdrop');
                const modalClose = document.getElementById('vtxNotifModalClose');
                const modalTitle = document.getElementById('vtxNotifModalTitle');
                const modalBody = document.getElementById('vtxNotifModalBody');

                function openModal(title, body) {
                    if (!modal || !modalTitle || !modalBody) return;
                    modalTitle.textContent = title || 'Уведомление';
                    modalBody.textContent = body || '';
                    modal.classList.remove('hidden');
                }

                function closeModal() {
                    if (!modal) return;
                    modal.classList.add('hidden');
                }

                if (modalBackdrop) modalBackdrop.addEventListener('click', closeModal);
                if (modalClose) modalClose.addEventListener('click', closeModal);
                window.addEventListener('keydown', (e) => {
                    if (e.key === 'Escape') closeModal();
                });

                function esc(s) {
                    return String(s || '').replace(/[&<>"']/g, (c) => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
                }

                function badge(level) {
                    const l = String(level || 'info');
                    if (l === 'critical') return '<span class="inline-flex items-center rounded-full border border-rose-500/30 bg-rose-500/10 px-2 py-0.5 text-[11px] font-medium text-rose-200">critical</span>';
                    if (l === 'warning') return '<span class="inline-flex items-center rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-[11px] font-medium text-amber-200">warning</span>';
                    return '<span class="inline-flex items-center rounded-full border border-white/10 bg-black/10 px-2 py-0.5 text-[11px] font-medium text-slate-200">info</span>';
                }

                async function loadAll() {
                    if (!tbody) return;
                    tbody.innerHTML = '<tr><td colspan="6" class="px-4 py-8 text-center text-slate-300/70">Загрузка…</td></tr>';

                    let data;
                    try {
                        const r = await fetch(@json(route('admin.notifications.api.list')) + '?limit=100', {
                            headers: { 'Accept': 'application/json' },
                            credentials: 'same-origin',
                        });
                        data = await r.json();
                    } catch (e) {
                        tbody.innerHTML = '<tr><td colspan="6" class="px-4 py-8 text-center text-rose-200">Ошибка загрузки</td></tr>';
                        return;
                    }

                    const items = (data && data.data && Array.isArray(data.data.items)) ? data.data.items : [];
                    if (items.length === 0) {
                        tbody.innerHTML = '<tr><td colspan="6" class="px-4 py-8 text-center text-slate-300/70">Нет уведомлений</td></tr>';
                        return;
                    }

                    const itemsById = new Map(items.map((n) => [Number(n.id || 0), n]));

                    tbody.innerHTML = items.map((n) => {
                        const id = Number(n.id || 0);
                        const isRead = !!n.is_read;
                        const btn = isRead
                            ? '<span class="text-[11px] text-slate-300/70">Прочитано</span>'
                            : `<button type="button" data-id="${id}" class="markReadBtn inline-flex h-8 items-center justify-center rounded-xl border border-white/10 bg-black/10 px-3 text-xs font-semibold text-slate-200 hover:bg-black/15 hover:text-white">Прочитать</button>`;

                        return `
                            <tr class="hover:bg-black/10">
                                <td class="px-4 py-3 align-top font-mono text-xs text-slate-300/80 whitespace-nowrap">#${id}</td>
                                <td class="px-4 py-3 align-top whitespace-nowrap">${badge(n.level)}</td>
                                <td class="px-4 py-3 align-top text-slate-100 font-medium break-words">${esc(n.title)}</td>
                                <td class="px-4 py-3 align-top text-slate-200 w-[520px]">
                                    <button type="button" data-id="${id}" class="openBodyBtn block w-full text-left">
                                        <div class="vtx-clamp-3 vtx-wrap-anywhere whitespace-pre-wrap text-[13px] text-slate-200 hover:text-white">${esc(n.body)}</div>
                                        <div class="mt-1 text-[11px] text-slate-300/70">Нажмите, чтобы открыть полностью</div>
                                    </button>
                                </td>
                                <td class="px-4 py-3 align-top text-slate-300/80 whitespace-nowrap">${esc(n.created_at)}</td>
                                <td class="px-4 py-3 align-top text-right whitespace-nowrap">${btn}</td>
                            </tr>
                        `;
                    }).join('');

                    for (const el of Array.from(tbody.querySelectorAll('.openBodyBtn'))) {
                        el.addEventListener('click', (e) => {
                            const id = Number(e.currentTarget?.getAttribute('data-id') || 0);
                            const n = itemsById.get(id);
                            openModal(n?.title || 'Уведомление', n?.body || '');
                        });
                    }

                    for (const el of Array.from(tbody.querySelectorAll('.markReadBtn'))) {
                        el.addEventListener('click', async (e) => {
                            const id = e.currentTarget?.getAttribute('data-id');
                            if (!id) return;
                            try {
                                await fetch(@json(route('admin.notifications.api.read', ['id' => 0])).replace('/0/read', '/' + encodeURIComponent(id) + '/read'), {
                                    method: 'POST',
                                    headers: {
                                        'Accept': 'application/json',
                                        'Content-Type': 'application/json',
                                        'X-CSRF-TOKEN': csrf,
                                    },
                                    credentials: 'same-origin',
                                    body: '{}',
                                });
                            } catch (err) {}
                            await loadAll();
                        });
                    }
                }

                window.vtxNotifications = window.vtxNotifications || {};
                window.vtxNotifications.reloadAll = loadAll;

                loadAll();
            })();
        </script>
    @endpush
@endsection
