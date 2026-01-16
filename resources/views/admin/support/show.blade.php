@extends('layouts.app-admin')

@section('page_title', 'Тикет #' . $ticket->id)

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Тикет #{{ $ticket->id }}</h1>
                    <p class="mt-1 text-sm text-slate-300/80">{{ $ticket->subject }}</p>
                    <p class="mt-1 text-[12px] text-slate-300/70">Пользователь: <span class="text-slate-100">{{ $ticket->user?->email ?? '—' }}</span></p>
                </div>
                <div class="flex items-center gap-2">
                    <a href="{{ route('admin.support.index') }}" class="inline-flex h-10 items-center justify-center rounded-xl border border-white/10 bg-black/10 px-4 text-xs font-semibold text-slate-200 hover:bg-black/15 hover:text-white">Назад</a>
                    @if((string) $ticket->status === 'open')
                        <form method="POST" action="{{ route('admin.support.close', $ticket) }}" onsubmit="return confirm('Закрыть тикет?');">
                            @csrf
                            <button type="submit" class="inline-flex h-10 items-center justify-center rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 text-xs font-semibold text-rose-200 hover:bg-rose-500/15">Закрыть</button>
                        </form>
                    @else
                        <form method="POST" action="{{ route('admin.support.reopen', $ticket) }}" onsubmit="return confirm('Открыть тикет снова?');">
                            @csrf
                            <button type="submit" class="inline-flex h-10 items-center justify-center rounded-xl border border-sky-500/30 bg-sky-500/10 px-4 text-xs font-semibold text-sky-200 hover:bg-sky-500/15">Открыть</button>
                        </form>
                    @endif
                </div>
            </div>

            @if (session('success'))
                <div class="rounded-2xl border border-emerald-500/20 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">{{ session('success') }}</div>
            @endif
            @if (session('error'))
                <div class="rounded-2xl border border-rose-500/20 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">{{ session('error') }}</div>
            @endif

            <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
                <div class="border-b border-white/10 bg-black/10 px-5 py-4 flex items-center justify-between gap-2">
                    <div class="text-sm font-semibold text-slate-100">Переписка</div>
                    @if((string) $ticket->status === 'closed')
                        <span class="inline-flex items-center rounded-full bg-black/10 px-2 py-0.5 text-[11px] font-medium text-slate-200 ring-1 ring-white/10">Закрыт</span>
                    @else
                        <span class="inline-flex items-center rounded-full border border-sky-500/30 bg-sky-500/10 px-2 py-0.5 text-[11px] font-medium text-sky-200">Открыт</span>
                    @endif
                </div>

                <div id="chatMessages" class="p-5 space-y-3" data-last-id="{{ (int) ($ticket->messages->last()?->id ?? 0) }}">
                    @forelse($ticket->messages as $m)
                        <div class="rounded-2xl border border-white/10 {{ $m->is_admin ? 'bg-slate-900/40' : 'bg-black/10' }} p-4" data-message-id="{{ (int) $m->id }}">
                            <div class="flex items-center justify-between gap-3">
                                <div class="text-[12px] font-semibold {{ $m->is_admin ? 'text-emerald-200' : 'text-slate-100' }}">
                                    {{ $m->is_admin ? ('Админ: ' . ($m->user?->email ?? '')) : 'Пользователь' }}
                                </div>
                                <div class="text-[11px] text-slate-300/70">{{ $m->created_at?->format('d.m.Y H:i') ?? '—' }}</div>
                            </div>
                            <div class="mt-2 text-[13px] text-slate-200 whitespace-pre-line">{{ $m->message }}</div>
                        </div>
                    @empty
                        <div class="text-sm text-slate-300/70">Сообщений пока нет</div>
                    @endforelse
                </div>

                @if((string) $ticket->status === 'open')
                    <div class="border-t border-white/10 bg-black/10 p-5">
                        <form method="POST" action="{{ route('admin.support.reply', $ticket) }}" class="grid gap-3">
                            @csrf
                            <div>
                                <textarea name="message" rows="4" required class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500" placeholder="Ответ админа..."></textarea>
                                @error('message')
                                    <div class="mt-2 text-xs text-rose-300">{{ $message }}</div>
                                @enderror
                            </div>
                            <div class="flex justify-end">
                                <button type="submit" class="inline-flex h-10 items-center justify-center rounded-xl bg-sky-600 px-5 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">Отправить</button>
                            </div>
                        </form>
                    </div>
                @endif
            </div>
        </div>
    </section>

    <script>
        (function () {
            const messagesEl = document.getElementById('chatMessages');
            if (!messagesEl) return;

            const ticketStatus = @json((string) $ticket->status);
            if (ticketStatus !== 'open') return;

            const endpoint = @json(route('admin.support.messages', $ticket));

            let afterId = parseInt(messagesEl.dataset.lastId || '0', 10);
            let inFlight = false;

            function isNearBottom() {
                const threshold = 120;
                const distance = messagesEl.scrollHeight - messagesEl.scrollTop - messagesEl.clientHeight;
                return distance <= threshold;
            }

            function scrollToBottom() {
                messagesEl.scrollTop = messagesEl.scrollHeight;
            }

            function buildMessageNode(m) {
                const root = document.createElement('div');
                root.className = 'rounded-2xl border border-white/10 ' + (m.is_admin ? 'bg-slate-900/40' : 'bg-black/10') + ' p-4';
                root.dataset.messageId = String(m.id);

                const header = document.createElement('div');
                header.className = 'flex items-center justify-between gap-3';

                const who = document.createElement('div');
                who.className = 'text-[12px] font-semibold ' + (m.is_admin ? 'text-emerald-200' : 'text-slate-100');
                who.textContent = m.is_admin ? ('Админ: ' + (m.user_email || '')) : 'Пользователь';

                const time = document.createElement('div');
                time.className = 'text-[11px] text-slate-300/70';
                time.textContent = m.created_at || '—';

                header.appendChild(who);
                header.appendChild(time);

                const body = document.createElement('div');
                body.className = 'mt-2 text-[13px] text-slate-200 whitespace-pre-line';
                body.textContent = m.message || '';

                root.appendChild(header);
                root.appendChild(body);

                return root;
            }

            async function poll() {
                if (inFlight) return;
                inFlight = true;

                const shouldStickToBottom = isNearBottom();

                try {
                    const url = new URL(endpoint, window.location.origin);
                    url.searchParams.set('after_id', String(afterId));

                    const res = await fetch(url.toString(), {
                        headers: {
                            'Accept': 'application/json'
                        },
                        credentials: 'same-origin'
                    });

                    if (!res.ok) return;

                    const data = await res.json();
                    const msgs = Array.isArray(data.messages) ? data.messages : [];

                    for (const m of msgs) {
                        if (!m || typeof m.id !== 'number') continue;
                        if (m.id <= afterId) continue;

                        messagesEl.appendChild(buildMessageNode(m));
                        afterId = m.id;
                        messagesEl.dataset.lastId = String(afterId);
                    }

                    if (msgs.length > 0 && shouldStickToBottom) {
                        scrollToBottom();
                    }
                } catch (e) {
                } finally {
                    inFlight = false;
                }
            }

            setTimeout(scrollToBottom, 0);
            setInterval(poll, 3000);
        })();
    </script>
@endsection
