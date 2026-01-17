@extends('layouts.app-admin')

@section('page_title', 'Обновления')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div>
                <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Обновления</h1>
                <p class="mt-1 text-sm text-slate-300/80">Статус версии панели и политика обновлений из облака (данные в реальном времени).</p>
            </div>

            <div class="grid gap-4 md:grid-cols-3 text-sm">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Текущая версия</p>
                    <p class="mt-2 text-2xl font-semibold text-slate-100">{{ $currentVersion }}</p>
                </div>

                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Проверка подписи manifest</p>
                    @php
                        $mOk = is_array($manifestCache) ? (bool) ($manifestCache['ok'] ?? false) : false;
                        $mErr = is_array($manifestCache) ? (string) ($manifestCache['error'] ?? '') : '';
                    @endphp
                    <p class="mt-2 text-sm font-semibold {{ $mOk ? 'text-emerald-300' : 'text-rose-200' }}">
                        {{ $mOk ? 'OK' : 'NO' }}
                    </p>
                    <p class="mt-1 text-[11px] text-slate-300/70 break-all">
                        {{ $mOk ? 'Подпись валидна' : ($mErr !== '' ? $mErr : 'manifest не получен') }}
                    </p>
                </div>

                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Статус</p>
                    @php
                        $statusLabel = 'OK';
                        $statusClass = 'text-emerald-300';
                        if ($isBlocked) { $statusLabel = 'BLOCKED'; $statusClass = 'text-rose-200'; }
                        elseif ($mandatoryUpdate) { $statusLabel = 'UPDATE REQUIRED'; $statusClass = 'text-rose-200'; }
                        elseif ($needsUpdate) { $statusLabel = 'UPDATE AVAILABLE'; $statusClass = 'text-amber-200'; }
                    @endphp
                    <p class="mt-2 text-sm font-semibold {{ $statusClass }}">{{ $statusLabel }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">
                        @if($isBlocked)
                            Текущая версия находится в списке заблокированных.
                        @elseif($mandatoryUpdate)
                            Версия ниже минимально разрешённой.
                        @elseif($needsUpdate)
                            Доступна более новая версия.
                        @else
                            Версия соответствует политике.
                        @endif
                    </p>
                </div>
            </div>

            <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                <div class="flex flex-wrap items-center justify-between gap-2">
                    <h2 class="text-sm font-semibold text-slate-100">Политика обновлений (manifest)</h2>
                    @php
                        $checkedAt = is_array($manifestCache) ? (int) ($manifestCache['checked_at'] ?? 0) : 0;
                    @endphp
                    <div class="text-[11px] text-slate-300/70">
                        Проверено: {{ $checkedAt > 0 ? \Carbon\Carbon::createFromTimestamp($checkedAt)->toDateTimeString() : '—' }}
                    </div>
                </div>

                <div class="mt-4 grid gap-3 md:grid-cols-3 text-sm">
                    <div class="rounded-xl border border-white/10 bg-black/10 p-3">
                        <div class="text-xs font-semibold text-slate-300/70">latest_version</div>
                        <div class="mt-1 font-mono text-xs text-slate-100">{{ $latest !== '' ? $latest : '—' }}</div>
                    </div>
                    <div class="rounded-xl border border-white/10 bg-black/10 p-3">
                        <div class="text-xs font-semibold text-slate-300/70">min_version</div>
                        <div class="mt-1 font-mono text-xs text-slate-100">{{ $min !== '' ? $min : '—' }}</div>
                    </div>
                    <div class="rounded-xl border border-white/10 bg-black/10 p-3">
                        <div class="text-xs font-semibold text-slate-300/70">blocked_versions</div>
                        <div class="mt-1 font-mono text-xs text-slate-100">
                            {{ count($blocked) > 0 ? implode(', ', $blocked) : '—' }}
                        </div>
                    </div>
                </div>

                @if(is_array($manifest))
                    @php
                        $notes = (string) ($manifest['release_notes'] ?? '');
                        $url = (string) ($manifest['download_url'] ?? '');
                    @endphp
                    @if($notes !== '')
                        <div class="mt-4 rounded-xl border border-white/10 bg-black/10 p-3">
                            <div class="text-xs font-semibold text-slate-300/70">release_notes</div>
                            <pre class="mt-2 whitespace-pre-wrap text-xs text-slate-100">{{ $notes }}</pre>
                        </div>
                    @endif
                    @if($url !== '')
                        <div class="mt-4">
                            @php
                                $canAutoUpdate = $mOk;
                            @endphp
                            <div class="flex flex-wrap items-center gap-2">
                                <a href="{{ $url }}" class="inline-flex items-center rounded-xl bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800 transition" target="_blank" rel="noreferrer">
                                    Открыть ссылку на обновление
                                </a>
                                <button
                                    type="button"
                                    id="apply-update-btn"
                                    class="inline-flex items-center rounded-xl bg-emerald-700 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-600 transition disabled:opacity-50 disabled:cursor-not-allowed"
                                    {{ $canAutoUpdate ? '' : 'disabled' }}
                                >
                                    Обновить
                                </button>
                            </div>
                        </div>
                    @endif
                @endif
            </div>

            <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                <h2 class="text-sm font-semibold text-slate-100">Как обновляться (вариант A)</h2>
                <div class="mt-2 text-sm text-slate-300/80 space-y-2">
                    <div>1) Скачай новую версию по ссылке (если она задана) или возьми релиз из репозитория.</div>
                    <div>2) Разверни обновление на сервере панели.</div>
                    <div>3) Выполни миграции (если есть) и перезапусти сервис.</div>
                </div>
            </div>
        </div>
    </section>

    <div id="update-modal" class="fixed inset-0 z-50 hidden">
        <div id="update-modal-overlay" class="absolute inset-0 bg-black/60 backdrop-blur-sm"></div>
        <div class="relative z-10 flex min-h-full items-center justify-center p-4">
            <div id="update-modal-panel" class="w-full max-w-md rounded-2xl border border-white/10 bg-[#242f3d] p-5 shadow-2xl shadow-black/50">
                <div class="flex items-start justify-between gap-4">
                    <div>
                        <div class="text-sm font-semibold text-slate-100" id="update-modal-title">Обновление</div>
                        <div class="mt-1 text-sm text-slate-300/80" id="update-modal-message">—</div>
                    </div>
                    <button type="button" class="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-white/10 bg-white/5 text-slate-200 hover:bg-white/10 transition" id="update-modal-close" aria-label="Закрыть">
                        <span class="text-base leading-none">✕</span>
                    </button>
                </div>

                <div class="mt-4 flex items-center justify-end gap-2">
                    <button type="button" class="inline-flex items-center rounded-xl bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800 transition" id="update-modal-ok">
                        OK
                    </button>
                </div>
            </div>
        </div>
    </div>

    <script>
        (function () {
            const btn = document.getElementById('apply-update-btn');
            const modal = document.getElementById('update-modal');
            const overlay = document.getElementById('update-modal-overlay');
            const titleEl = document.getElementById('update-modal-title');
            const msgEl = document.getElementById('update-modal-message');
            const closeBtn = document.getElementById('update-modal-close');
            const okBtn = document.getElementById('update-modal-ok');
            const panel = document.getElementById('update-modal-panel');
            let shouldReloadOnClose = false;

            function openModal(type, title, msg) {
                titleEl.textContent = title;
                msgEl.textContent = msg;

                panel.classList.remove('border-emerald-400/20', 'bg-emerald-500/10', 'border-rose-400/20', 'bg-rose-500/10');
                if (type === 'success') {
                    panel.classList.add('border-emerald-400/20', 'bg-emerald-500/10');
                }
                if (type === 'error') {
                    panel.classList.add('border-rose-400/20', 'bg-rose-500/10');
                }

                modal.classList.remove('hidden');
            }

            function closeModal() {
                modal.classList.add('hidden');
                if (shouldReloadOnClose) {
                    shouldReloadOnClose = false;
                    window.location.reload();
                }
            }

            closeBtn?.addEventListener('click', closeModal);
            okBtn?.addEventListener('click', closeModal);
            overlay?.addEventListener('click', closeModal);

            document.addEventListener('keydown', function (e) {
                if (e.key === 'Escape' && !modal.classList.contains('hidden')) {
                    closeModal();
                }
            });

            btn?.addEventListener('click', async function () {
                if (btn.disabled) return;
                btn.disabled = true;
                const prevText = btn.textContent;
                btn.textContent = 'Обновление…';

                try {
                    const res = await fetch('{{ route('admin.updates.apply') }}', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'Accept': 'application/json',
                            'X-CSRF-TOKEN': '{{ csrf_token() }}',
                        },
                        body: JSON.stringify({}),
                    });

                    const data = await res.json().catch(() => ({}));
                    if (!res.ok) {
                        throw new Error(data?.message || ('HTTP ' + res.status));
                    }

                    shouldReloadOnClose = true;
                    openModal('success', 'Успешно', data?.message || 'Обновление выполнено.');
                } catch (e) {
                    shouldReloadOnClose = false;
                    openModal('error', 'Ошибка', e?.message || 'Не удалось выполнить обновление');
                } finally {
                    btn.disabled = false;
                    btn.textContent = prevText;
                }
            });
        })();
    </script>
@endsection
