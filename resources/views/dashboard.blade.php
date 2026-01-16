@extends('layouts.app-user')

@section('page_title', 'Обзор')

@section('content')
    <section class="py-6 md:py-8">
        <div class="w-full px-4 sm:px-6 lg:px-8">
        @php
            $user = auth()->user();
            $balanceValue = $balance ?? (auth()->user()->balance ?? 0);
            $totalServersValue = $totalServers ?? 0;
            $activeServersValue = $activeServers ?? 0;

            $nextChargeText = '—';
            if (isset($nextExpiryServer) && $nextExpiryServer && $nextExpiryServer->expires_at) {
                $today = now()->startOfDay();
                $expiresAt = $nextExpiryServer->expires_at->copy()->startOfDay();
                $daysLeft = $today->diffInDays($expiresAt, false);
                $nextChargeText = $daysLeft >= 0 ? ('через ' . $daysLeft . ' дн.') : 'истёк';
            }
        @endphp
        <div class="w-full space-y-6">
            <div class="grid gap-4 md:grid-cols-[1.6fr_1fr]">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-5 shadow-sm">
                    <div class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
                        <div class="flex flex-col gap-1">
                            <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">
                                Привет, {{ $user->name ?? 'пользователь' }}
                            </h1>
                            <p class="text-sm text-slate-300">
                                Добро пожаловать в панель управления. Здесь ты можешь управлять серверами, балансом и настройками аккаунта.
                            </p>
                        </div>

                        <div class="flex flex-col gap-2 sm:flex-row md:flex-col md:items-end">
                            <a href="{{ route('my-servers') }}" class="inline-flex h-10 items-center justify-center rounded-xl bg-slate-900 px-4 text-xs font-semibold text-white shadow-sm hover:bg-slate-800 md:w-44">
                                Мои серверы
                            </a>
                            <a href="{{ route('rent-server') }}" class="inline-flex h-10 items-center justify-center rounded-xl border border-white/10 bg-black/10 px-4 text-xs font-semibold text-slate-200 shadow-sm hover:bg-black/15 md:w-44 text-center leading-tight">
                                Арендовать<br class="hidden md:block" />сервер
                            </a>
                            <a href="{{ route('billing') }}" class="inline-flex h-10 items-center justify-center rounded-xl border border-white/10 bg-black/10 px-4 text-xs font-semibold text-slate-200 shadow-sm hover:bg-black/15 md:w-44 text-center leading-tight">
                                Пополнить<br class="hidden md:block" />баланс
                            </a>
                            <a href="{{ route('support.index') }}" class="inline-flex h-10 items-center justify-center rounded-xl border border-white/10 bg-black/10 px-4 text-xs font-semibold text-slate-200 shadow-sm hover:bg-black/15 md:w-44 text-center leading-tight">
                                Тех. поддержка
                                @if(($openSupportTicketsCount ?? 0) > 0)
                                    <span class="ml-2 inline-flex h-5 min-w-[20px] items-center justify-center rounded-full bg-sky-500/15 px-2 text-[11px] font-semibold text-sky-200 ring-1 ring-sky-500/25">{{ (int) $openSupportTicketsCount }}</span>
                                @endif
                            </a>
                        </div>
                    </div>
                </div>

                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-5 shadow-sm">
                    <div class="text-xs font-semibold text-slate-300/80">Профиль</div>
                    <div class="mt-3 space-y-2 text-sm">
                        <div class="flex items-center justify-between gap-3">
                            <div class="text-[12px] text-slate-300/70">Email</div>
                            <div class="truncate font-semibold text-slate-100">{{ $user->email ?? '—' }}</div>
                        </div>
                        <div class="flex items-center justify-between gap-3">
                            <div class="text-[12px] text-slate-300/70">Баланс</div>
                            <div class="font-semibold text-slate-100">{{ number_format((float) $balanceValue, 2) }} ₽</div>
                        </div>
                        <div class="flex items-center justify-between gap-3">
                            <div class="text-[12px] text-slate-300/70">Дата регистрации</div>
                            <div class="font-semibold text-slate-100">{{ $user?->created_at?->format('d.m.Y') ?? '—' }}</div>
                        </div>
                    </div>

                    <div class="mt-4">
                        <a href="{{ route('account') }}" class="inline-flex w-full items-center justify-center rounded-xl border border-white/10 bg-black/10 px-4 py-2 text-xs font-semibold text-slate-200 shadow-sm hover:bg-black/15">
                            Открыть профиль
                        </a>
                    </div>
                </div>
            </div>

            <div class="overflow-hidden rounded-2xl border border-white/10 bg-[#242f3d] shadow-sm">

                @if(!isset($recentServers) || $recentServers->isEmpty())
                    <div class="p-6">
                        <div class="rounded-xl border border-dashed border-white/10 bg-black/10 p-4 text-center text-[12px] text-slate-300/70">
                            У вас пока нет серверов.
                            <a href="{{ route('rent-server') }}" class="font-semibold text-slate-200 hover:text-white">Арендовать сервер</a>
                        </div>
                    </div>
                @else
                    <div class="overflow-x-auto">
                        <table class="min-w-full divide-y divide-white/10 text-sm">
                            <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                                <tr>
                                    <th class="px-4 py-3 text-left">Сервер</th>
                                    <th class="px-4 py-3 text-left">Адрес</th>
                                    <th class="px-4 py-3 text-left">Аренда</th>
                                    <th class="px-4 py-3 text-left">Статус</th>
                                    <th class="px-4 py-3 text-right">Действия</th>
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-white/10">
                                @foreach($recentServers as $server)
                                    @php
                                        $prov = strtolower((string) ($server->provisioning_status ?? ''));
                                        $runtime = strtolower((string) ($server->runtime_status ?? ''));
                                        $status = strtolower((string) ($server->status ?? ''));

                                        $isProvisioning = in_array($prov, ['pending', 'installing', 'reinstalling'], true);
                                        $isInstallFailed = $prov === 'failed';

                                        $showRestart = ! $isProvisioning && ! $isInstallFailed && $runtime === 'running';
                                        $showStop = ! $isProvisioning && ! $isInstallFailed && $runtime === 'running';
                                        $showStart = ! $isProvisioning && ! $isInstallFailed && in_array($runtime, ['offline', 'stopped'], true);
                                        $showReinstall = ! $isProvisioning && (in_array($runtime, ['offline', 'stopped', 'missing'], true) || $isInstallFailed);

                                        $statusLabel = 'Неизвестно';
                                        $statusClass = 'bg-black/10 text-slate-200 ring-white/10';

                                        if (in_array($prov, ['pending', 'installing'], true)) {
                                            $statusLabel = 'Установка';
                                            $statusClass = 'bg-amber-500/10 text-amber-200 ring-amber-500/20';
                                        } elseif ($prov === 'reinstalling') {
                                            $statusLabel = 'Переустановка';
                                            $statusClass = 'bg-amber-500/10 text-amber-200 ring-amber-500/20';
                                        } elseif ($prov === 'failed') {
                                            $statusLabel = 'Ошибка установки';
                                            $statusClass = 'bg-rose-500/10 text-rose-200 ring-rose-500/20';
                                        } elseif ($runtime === 'running') {
                                            $statusLabel = 'Работает';
                                            $statusClass = 'bg-emerald-500/10 text-emerald-200 ring-emerald-500/20';
                                        } elseif ($runtime === 'missing') {
                                            $statusLabel = 'Не установлен';
                                            $statusClass = 'bg-rose-500/10 text-rose-200 ring-rose-500/20';
                                        } elseif (in_array($runtime, ['offline', 'stopped'], true)) {
                                            $statusLabel = 'Выключен';
                                            $statusClass = 'bg-black/10 text-slate-200 ring-white/10';
                                        } elseif ($status === 'suspended') {
                                            $statusLabel = 'Приостановлен';
                                            $statusClass = 'bg-amber-500/10 text-amber-200 ring-amber-500/20';
                                        } elseif ($status === 'active') {
                                            $statusLabel = 'Активен';
                                            $statusClass = 'bg-emerald-500/10 text-emerald-200 ring-emerald-500/20';
                                        } elseif ($status === 'offline') {
                                            $statusLabel = 'Оффлайн';
                                            $statusClass = 'bg-rose-500/10 text-rose-200 ring-rose-500/20';
                                        }
                                    @endphp
                                    <tr class="hover:bg-black/10">
                                        <td class="px-4 py-3">
                                            <div class="font-semibold text-slate-100">{{ $server->name }}</div>
                                            <div class="mt-0.5 text-[12px] text-slate-300/70">
                                                {{ $server->game->name ?? 'Игра' }} • {{ $server->location->name ?? 'Локация' }}
                                            </div>
                                        </td>
                                        <td class="px-4 py-3">
                                            <span class="inline-flex items-center rounded-xl bg-black/10 px-2 py-1 text-xs font-semibold text-slate-200 ring-1 ring-white/10">
                                                {{ $server->ip_address }}:{{ $server->port }}
                                            </span>
                                        </td>
                                        <td class="px-4 py-3 text-[12px] text-slate-200">
                                            @if($server->expires_at)
                                                до {{ $server->expires_at->format('d.m.Y') }}
                                            @else
                                                —
                                            @endif
                                        </td>
                                        <td class="px-4 py-3">
                                            <span class="inline-flex items-center rounded-full px-2.5 py-1 text-[11px] font-semibold ring-1 {{ $statusClass }}">{{ $statusLabel }}</span>
                                        </td>
                                        <td class="px-4 py-3">
                                            <div class="flex items-center justify-end gap-2">
                                                <a
                                                    href="{{ route('server.show', $server) }}"
                                                    title="Открыть управление"
                                                    class="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-white/10 bg-black/10 text-slate-200 shadow-sm transition hover:bg-black/15 hover:text-white"
                                                >
                                                    <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6h.01M12 12h.01M12 18h.01" />
                                                    </svg>
                                                </a>

                                                @if($showRestart)
                                                    <form method="POST" action="{{ route('server.restart', $server) }}">
                                                        @csrf
                                                        <button type="submit" title="Перезапуск" class="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-white/10 bg-black/10 text-slate-200 shadow-sm transition hover:bg-black/15 hover:text-white">
                                                            <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v6h6M20 20v-6h-6" />
                                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 9a9 9 0 00-15.5-3.5L4 10m0 5a9 9 0 0015.5 3.5L20 14" />
                                                            </svg>
                                                        </button>
                                                    </form>
                                                @endif

                                                @if($showStop)
                                                    <form method="POST" action="{{ route('server.stop', $server) }}">
                                                        @csrf
                                                        <button type="submit" title="Остановить" class="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-white/10 bg-rose-500/10 text-rose-200 shadow-sm transition hover:bg-rose-500/15">
                                                            <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                                <rect x="7" y="7" width="10" height="10" rx="2" ry="2" stroke-width="2" />
                                                            </svg>
                                                        </button>
                                                    </form>
                                                @endif

                                                @if($showStart)
                                                    <form method="POST" action="{{ route('server.start', $server) }}">
                                                        @csrf
                                                        <button type="submit" title="Запустить" class="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-white/10 bg-emerald-500/10 text-emerald-200 shadow-sm transition hover:bg-emerald-500/15">
                                                            <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5v14l11-7-11-7z" />
                                                            </svg>
                                                        </button>
                                                    </form>
                                                @endif

                                                @if($showReinstall)
                                                    <form method="POST" action="{{ route('server.reinstall', $server) }}">
                                                        @csrf
                                                        <button type="submit" title="Переустановить" class="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-white/10 bg-amber-500/10 text-amber-200 shadow-sm transition hover:bg-amber-500/15">
                                                            <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v6h6M20 20v-6h-6" />
                                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 9a9 9 0 00-15.5-3.5L4 10m0 5a9 9 0 0015.5 3.5L20 14" />
                                                            </svg>
                                                        </button>
                                                    </form>
                                                @endif
                                            </div>
                                        </td>
                                    </tr>
                                @endforeach
                            </tbody>
                        </table>
                    </div>
                @endif
            </div>

            <div>
                <h2 class="text-sm font-semibold text-slate-100">Новости</h2>
                <div class="mt-3 grid gap-4 md:grid-cols-3">
                    @if(!empty($news) && count($news) > 0)
                    @foreach($news as $item)
                        <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-5 shadow-sm">
                            @php
                                $img = null;
                                if (! empty($item->images) && count($item->images) > 0) {
                                    $img = $item->images[0];
                                }
                            @endphp
                            @if($img)
                                <div class="mb-3 aspect-video overflow-hidden rounded-xl border border-white/10 bg-black/10">
                                    <img src="{{ \Illuminate\Support\Facades\Storage::disk('public')->url((string) $img->path) }}" alt="" class="h-full w-full object-cover">
                                </div>
                            @endif
                            <div class="text-[11px] text-slate-300/70">{{ optional($item->published_at)->format('d.m.Y') ?? '—' }}</div>
                            <div class="mt-1 font-semibold text-slate-100">{{ $item->title }}</div>
                            <div class="mt-2 text-[12px] leading-relaxed text-slate-300/80">{{ $item->excerpt ?: ($item->body ?: '') }}</div>
                        </div>
                    @endforeach
                    @else
                        <div class="md:col-span-3 rounded-2xl border border-white/10 bg-[#242f3d] p-5 shadow-sm text-[12px] text-slate-300/80">
                            Пока нет новостей.
                        </div>
                    @endif
                </div>
            </div>
        </div>
        </div>
    </section>
@endsection
