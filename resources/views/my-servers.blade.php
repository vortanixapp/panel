@extends('layouts.app-user')

@section('title', 'Мои серверы')
@section('page_title', 'Мои серверы')

@section('content')
    <section class="py-8">
        <div class="w-full px-4 sm:px-6 lg:px-8">
            <div class="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
                <div>
                    <h1 class="text-2xl font-bold text-slate-100">Мои серверы</h1>
                    <p class="mt-1 text-sm text-slate-300">Управление вашими арендованными серверами</p>
                </div>

                <a
                    href="{{ route('rent-server') }}"
                    class="inline-flex items-center justify-center rounded-xl bg-slate-900 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-slate-800"
                >
                    Арендовать сервер
                </a>
            </div>

            @if($servers->isEmpty())
                <div class="overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm">
                    <div class="p-10 text-center">
                        <div class="mx-auto mb-4 inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-black/10 text-slate-300/80 ring-1 ring-white/10">
                            <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7h16M4 12h16M4 17h16"></path>
                            </svg>
                        </div>
                        <h3 class="text-lg font-semibold text-slate-100">У вас нет активных серверов</h3>
                        <p class="mt-1 text-sm text-slate-300/70">Начните с аренды вашего первого сервера</p>
                    </div>
                </div>
            @else
                @php
                    $today = now()->startOfDay();
                    $total = $servers->count();
                    $activeCount = $servers->where('runtime_status', 'running')->count();
                    $expiringSoon = $servers->filter(function ($s) use ($today) {
                        if (! $s->expires_at) {
                            return false;
                        }

                        $expiresAt = $s->expires_at->copy()->startOfDay();
                        $daysLeft = $today->diffInDays($expiresAt, false);

                        return $daysLeft >= 0 && $daysLeft < 7;
                    })->count();
                @endphp

                <div class="grid gap-3 sm:grid-cols-3">
                    <div class="rounded-2xl border border-white/10 bg-[#242f3d] px-4 py-3 shadow-sm">
                        <div class="text-xs uppercase tracking-wide text-slate-300/70">Всего</div>
                        <div class="mt-1 text-xl font-semibold text-slate-100">{{ $total }}</div>
                    </div>
                    <div class="rounded-2xl border border-white/10 bg-[#242f3d] px-4 py-3 shadow-sm">
                        <div class="text-xs uppercase tracking-wide text-slate-300/70">Активных</div>
                        <div class="mt-1 text-xl font-semibold text-slate-100">{{ $activeCount }}</div>
                    </div>
                    <div class="rounded-2xl border border-white/10 bg-[#242f3d] px-4 py-3 shadow-sm">
                        <div class="text-xs uppercase tracking-wide text-slate-300/70">Истекают ≤ 7 дней</div>
                        <div class="mt-1 text-xl font-semibold text-slate-100">{{ $expiringSoon }}</div>
                    </div>
                </div>

                <div class="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                    @foreach($servers as $server)
                        <!-- Server card -->
                        @php
                            $runtime = strtolower((string) ($server->runtime_status ?? ''));
                            $prov = strtolower((string) ($server->provisioning_status ?? ''));

                            $isProvisioning = in_array($prov, ['pending', 'installing', 'reinstalling'], true);
                            $isInstallFailed = $prov === 'failed';

                            $statusLabel = 'Неизвестно';
                            $statusClass = 'bg-slate-50 text-slate-700 ring-slate-600/20';

                            if (in_array($prov, ['pending', 'installing'], true)) {
                                $statusLabel = 'Установка';
                                $statusClass = 'bg-amber-50 text-amber-700 ring-amber-600/20';
                            } elseif ($prov === 'reinstalling') {
                                $statusLabel = 'Переустановка';
                                $statusClass = 'bg-amber-50 text-amber-700 ring-amber-600/20';
                            } elseif ($prov === 'failed') {
                                $statusLabel = 'Ошибка установки';
                                $statusClass = 'bg-rose-50 text-rose-700 ring-rose-600/20';
                            } elseif ($runtime === 'running') {
                                $statusLabel = 'Работает';
                                $statusClass = 'bg-green-50 text-green-700 ring-green-600/20';
                            } elseif (in_array($runtime, ['offline', 'stopped'], true)) {
                                $statusLabel = 'Выключен';
                                $statusClass = 'bg-slate-50 text-slate-700 ring-slate-600/20';
                            } elseif ($runtime === 'missing') {
                                $statusLabel = 'Не установлен';
                                $statusClass = 'bg-rose-50 text-rose-700 ring-rose-600/20';
                            }

                            $showRestart = ! $isProvisioning && ! $isInstallFailed && $runtime === 'running';
                            $showStop = ! $isProvisioning && ! $isInstallFailed && $runtime === 'running';
                            $showStart = ! $isProvisioning && ! $isInstallFailed && in_array($runtime, ['offline', 'stopped'], true);
                            $showReinstall = ! $isProvisioning && (in_array($runtime, ['offline', 'stopped', 'missing'], true) || $isInstallFailed);

                            $expiresAt = $server->expires_at;
                            $daysLeft = $expiresAt ? $today->diffInDays($expiresAt->copy()->startOfDay(), false) : null;
                            $expiresSoon = $daysLeft !== null && $daysLeft >= 0 && $daysLeft < 7;
                        @endphp

                        <div class="group overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm transition-all duration-300 hover:shadow-md">
                            <div class="p-5">
                                <div class="flex items-start justify-between gap-3">
                                    <div class="min-w-0">
                                        <div class="flex items-center gap-3">
                                            <span class="inline-flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-2xl bg-gradient-to-tr from-sky-500 to-indigo-500 text-white shadow-sm shadow-sky-500/30">
                                                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                                    <path d="M4 4h12v4H4V4zM4 12h12v4H4v-4z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                    <path d="M7 6h.01M7 14h.01" stroke-width="1.6" stroke-linecap="round" />
                                                </svg>
                                            </span>
                                            <div class="min-w-0">
                                                <div class="truncate text-sm font-semibold text-slate-100">{{ $server->name }}</div>
                                                <div class="mt-0.5 truncate text-xs text-slate-300/70">{{ $server->game->name ?? 'Игра' }} • {{ $server->location->name ?? 'Локация' }}</div>
                                            </div>
                                        </div>

                                        <div class="mt-3 flex flex-wrap items-center gap-2 text-xs">
                                            <span class="inline-flex items-center gap-1.5 rounded-xl bg-black/10 px-2 py-1 text-slate-200 ring-1 ring-white/10">
                                                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                                    <path d="M3 6h14v8H3V6z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                    <path d="M6 9h2" stroke-width="1.4" stroke-linecap="round" />
                                                </svg>
                                                <span class="font-medium">{{ $server->ip_address }}:{{ $server->port }}</span>
                                            </span>
                                            @if($expiresAt)
                                                <span class="inline-flex items-center gap-1.5 rounded-xl bg-black/10 px-2 py-1 text-slate-200 ring-1 ring-white/10">
                                                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                                        <circle cx="10" cy="10" r="7" stroke-width="1.4" />
                                                        <path d="M10 6v4l3 2" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                    </svg>
                                                    <span>до {{ $expiresAt->format('d.m.Y') }}</span>
                                                </span>
                                            @endif
                                        </div>
                                    </div>

                                    <div class="flex flex-col items-end gap-2">
                                        <span class="inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold ring-1 {{ $statusClass }}">{{ $statusLabel }}</span>
                                        @if($expiresSoon)
                                            <span class="text-[11px] font-medium text-amber-700">Скоро истекает</span>
                                        @endif
                                    </div>
                                </div>

                                <div class="mt-4 flex flex-wrap items-center gap-2">
                                    <a
                                        href="{{ route('server.show', $server) }}"
                                        class="inline-flex items-center justify-center rounded-xl bg-slate-900 px-3 py-2 text-xs font-semibold text-white shadow-sm hover:bg-slate-800"
                                    >
                                        Управление
                                    </a>

                                    @if($showRestart)
                                        <form method="POST" action="{{ route('server.restart', $server) }}">
                                            @csrf
                                            <button
                                                type="submit"
                                                class="inline-flex items-center justify-center rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-xs font-semibold text-slate-200 shadow-sm hover:bg-black/15"
                                            >
                                                Перезапуск
                                            </button>
                                        </form>

                                        <form method="POST" action="{{ route('server.stop', $server) }}">
                                            @csrf
                                            <button
                                                type="submit"
                                                class="inline-flex items-center justify-center rounded-xl border border-rose-200 bg-rose-50 px-3 py-2 text-xs font-semibold text-rose-700 shadow-sm hover:bg-rose-100"
                                            >
                                                Остановить
                                            </button>
                                        </form>

                                    @elseif($showStart)
                                        <form method="POST" action="{{ route('server.start', $server) }}">
                                            @csrf
                                            <button
                                                type="submit"
                                                class="inline-flex items-center justify-center rounded-xl border border-emerald-200 bg-emerald-50 px-3 py-2 text-xs font-semibold text-emerald-700 shadow-sm hover:bg-emerald-100"
                                            >
                                                Запустить
                                            </button>
                                        </form>

                                    @endif

                                    @if($showReinstall)
                                        <form method="POST" action="{{ route('server.reinstall', $server) }}">
                                            @csrf
                                            <button
                                                type="submit"
                                                class="inline-flex items-center justify-center rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-xs font-semibold text-amber-800 shadow-sm hover:bg-amber-100"
                                            >
                                                Переустановить
                                            </button>
                                        </form>
                                    @endif
                                </div>
                            </div>
                        </div>
                    @endforeach
                </div>
            @endif
        </div>
    </section>
@endsection
