<div class="mt-6 overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] text-slate-100 shadow-sm transition-shadow duration-300 hover:shadow-md">
    <div id="serverToastContainer" class="fixed top-4 right-4 z-50 space-y-2"></div>
    <div class="border-b border-white/10 bg-black/10 px-5 py-4">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
                <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Игровой сервер: #{{ $server->id }}</div>
                <div class="mt-1 flex flex-wrap items-center gap-2">
                    <div class="text-base font-semibold text-slate-100">{{ $server->name }}</div>
                </div>
                <div class="mt-1 text-[12px] text-slate-300/70">
                    {{ $server->game->name }} (id: {{ $server->game_id }})
                    <span class="mx-1 text-slate-300">•</span>
                    {{ $server->location->name }}
                </div>
            </div>

            <div class="flex flex-col items-end gap-2">
                <span id="runtimeStatusBadge" class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium {{ $badge }}">{{ $label }}</span>
                <div class="inline-flex items-center rounded-2xl border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-semibold text-slate-100 shadow-sm">
                    {{ $server->ip_address }}:{{ $server->port }}
                </div>
            </div>
        </div>
    </div>

    @php
        $isExpired = $server->expires_at && now()->greaterThan($server->expires_at);
        $isInstallFailed = strtolower((string) ($server->provisioning_status ?? '')) === 'failed';
    @endphp

    @if ($isExpired)
        <div class="px-5 pt-5">
            <div class="rounded-2xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100">
                Срок аренды сервера истёк. Управление временно недоступно.
            </div>
        </div>
    @endif

    <div class="p-5">
        <div class="flex flex-col gap-6 md:flex-row md:items-start">
            <div class="flex-1 grid gap-4 text-sm">
                <div class="grid gap-3 sm:grid-cols-2">
                    <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                        <div class="text-[12px] text-slate-300/70">Адрес</div>
                        <div class="mt-1 font-semibold text-slate-100">{{ $server->ip_address }}:{{ $server->port }}</div>
                    </div>
                    <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                        <div class="text-[12px] text-slate-300/70">Локация</div>
                        <div class="mt-1 font-semibold text-slate-100">{{ $server->location->name }}</div>
                    </div>
                </div>

                <div class="grid gap-3 md:grid-cols-2">
                    <div class="grid h-full gap-2 rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                        <div class="flex items-center justify-between border-b border-white/10 pb-2">
                            <span class="text-slate-300/70">Создан</span>
                            <span class="font-medium text-slate-100">{{ $server->created_at->format('d.m.Y H:i') }}</span>
                        </div>
                        <div class="flex items-center justify-between border-b border-white/10 pb-2">
                            <span class="text-slate-300/70">Арендован до</span>
                            <span class="font-medium text-slate-100">{{ $server->expires_at->format('d.m.Y H:i') }}</span>
                        </div>
                        <div class="flex items-center justify-between">
                            <span class="text-slate-300/70">Осталось</span>
                            <span class="font-medium text-slate-100">{{ $daysLeft }} дней</span>
                        </div>
                    </div>

                    <div class="grid h-full gap-3 rounded-2xl border border-white/10 bg-[#242f3d] p-4">
                        <div class="flex items-center justify-between border-b border-white/10 pb-2">
                            <span class="text-slate-300/70">Ресурсы</span>
                            <span id="mainResourcesUpdatedAt" class="text-[11px] text-slate-400">—</span>
                        </div>

                        <div class="grid gap-1">
                            <div class="flex items-center justify-between text-sm">
                                <span class="text-slate-200">CPU</span>
                                <span id="mainCpuText" class="font-medium text-slate-100">—</span>
                            </div>
                            <div class="h-2 overflow-hidden rounded-full bg-black/20">
                                <div id="mainCpuBar" class="h-full rounded-full bg-sky-500" style="width: 0%"></div>
                            </div>
                        </div>

                        <div class="grid gap-1">
                            <div class="flex items-center justify-between text-sm">
                                <span class="text-slate-200">RAM</span>
                                <span id="mainRamText" class="font-medium text-slate-100">—</span>
                            </div>
                            <div class="h-2 overflow-hidden rounded-full bg-black/20">
                                <div id="mainRamBar" class="h-full rounded-full bg-emerald-500" style="width: 0%"></div>
                            </div>
                        </div>

                        <div class="grid gap-1">
                            <div class="flex items-center justify-between text-sm">
                                <span class="text-slate-200">Disk</span>
                                <span id="mainDiskText" class="font-medium text-slate-100">—</span>
                            </div>
                            <div class="h-2 overflow-hidden rounded-full bg-black/20">
                                <div id="mainDiskBar" class="h-full rounded-full bg-amber-500" style="width: 0%"></div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div class="w-full md:w-64 flex flex-col gap-3">
                @if ($showRestart)
                    <form method="POST" action="{{ route('server.restart', $server) }}" onsubmit="try { sessionStorage.setItem('vtx_restart_pending', '1'); } catch (e) {} this.querySelector('button[type=submit]')?.setAttribute('disabled','disabled'); this.querySelector('button[type=submit]')?.classList.add('opacity-60','cursor-not-allowed');">
                        @csrf
                        <button type="submit" class="w-full inline-flex items-center justify-center rounded-xl bg-emerald-600 px-4 py-2.5 text-xs font-semibold text-white shadow-sm transition hover:bg-emerald-700 active:scale-[0.99]">
                            Перезапустить
                        </button>
                    </form>
                @endif

                @if ($showStop)
                    <form method="POST" action="{{ route('server.stop', $server) }}" onsubmit="this.querySelector('button[type=submit]')?.setAttribute('disabled','disabled'); this.querySelector('button[type=submit]')?.classList.add('opacity-60','cursor-not-allowed');">
                        @csrf
                        <button type="submit" class="w-full inline-flex items-center justify-center rounded-xl bg-rose-600 px-4 py-2.5 text-xs font-semibold text-white shadow-sm transition hover:bg-rose-700 active:scale-[0.99]">
                            Выключить
                        </button>
                    </form>
                @endif

                @if ($showStart)
                    <form method="POST" action="{{ route('server.start', $server) }}" onsubmit="this.querySelector('button[type=submit]')?.setAttribute('disabled','disabled'); this.querySelector('button[type=submit]')?.classList.add('opacity-60','cursor-not-allowed');">
                        @csrf
                        <button type="submit" class="w-full inline-flex items-center justify-center rounded-xl bg-sky-600 px-4 py-2.5 text-xs font-semibold text-white shadow-sm transition hover:bg-sky-700 active:scale-[0.99]">
                            Включить
                        </button>
                    </form>
                @endif

                @if ($showReinstall)
                    <form method="POST" action="{{ route('server.reinstall', $server) }}">
                        @csrf
                        <button type="submit" class="w-full inline-flex items-center justify-center rounded-xl bg-amber-600 px-4 py-2.5 text-xs font-semibold text-white shadow-sm transition hover:bg-amber-700 active:scale-[0.99]">
                            Переустановить
                        </button>
                    </form>
                @endif

                @if ($isReinstalling)
                    <button type="button" disabled class="w-full inline-flex items-center justify-center rounded-xl bg-amber-100 px-4 py-2.5 text-xs font-semibold text-amber-700 shadow-sm opacity-80 cursor-not-allowed">
                        Переустановка...
                    </button>
                @endif

                @if (! $isInstallFailed)
                    <div class="pt-1">
                        @php
                            $renewalPeriods = $server->tariff && is_array($server->tariff->renewal_periods) && count($server->tariff->renewal_periods) > 0
                                ? array_values(array_map('intval', $server->tariff->renewal_periods))
                                : [15, 30, 60, 180];
                        @endphp
                        <form id="renewForm" method="POST" action="{{ route('server.renew', $server) }}" class="hidden">
                            @csrf
                            <input id="renewPeriodInput" type="hidden" name="period" value="">
                            <input id="renewPromoInput" type="hidden" name="promo_code" value="">
                        </form>

                        <button
                            type="button"
                            {{ $isExpired ? 'disabled' : '' }}
                            class="w-full inline-flex items-center justify-center rounded-xl border border-white/10 bg-black/10 px-4 py-2.5 text-xs font-semibold text-slate-200 shadow-sm transition hover:bg-black/15 hover:ring-1 hover:ring-white/10 active:scale-[0.99] {{ $isExpired ? 'opacity-60 cursor-not-allowed pointer-events-none' : '' }}"
                            onclick="document.getElementById('renewModal')?.classList.remove('hidden')"
                        >
                            Продлить аренду
                        </button>

                        <div id="renewModal" class="hidden fixed inset-0 z-50" onkeydown="if (event.key === 'Escape') { this.classList.add('hidden'); }" tabindex="-1">
                            <div class="absolute inset-0 bg-black/60" onclick="document.getElementById('renewModal')?.classList.add('hidden')"></div>
                            <div class="absolute inset-0 flex items-center justify-center p-4">
                                <div class="w-full max-w-sm rounded-2xl border border-white/10 bg-[#242f3d] shadow-lg">
                                    <div class="flex items-center justify-between border-b border-white/10 px-4 py-3">
                                        <div class="text-sm font-semibold text-slate-100">Продление аренды</div>
                                        <button type="button" class="text-slate-300/80 hover:text-slate-100" onclick="document.getElementById('renewModal')?.classList.add('hidden')">✕</button>
                                    </div>
                                    <div class="p-4">
                                        <div class="text-xs text-slate-300/80">Выберите период продления:</div>
                                        <div class="mt-3 grid grid-cols-2 gap-2">
                                            @foreach($renewalPeriods as $p)
                                                <button
                                                    type="button"
                                                    class="inline-flex items-center justify-center rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-xs font-semibold text-slate-200 shadow-sm transition hover:bg-black/15 hover:ring-1 hover:ring-white/10 active:scale-[0.99]"
                                                    onclick="if (window.__vtxRenewSubmitting) return; window.__vtxRenewSubmitting = true; document.querySelectorAll('#renewModal button').forEach(b => b.setAttribute('disabled','disabled')); document.getElementById('renewPeriodInput').value='{{ $p }}'; document.getElementById('renewPromoInput').value=(document.getElementById('renewPromoCodeField')?.value || ''); document.getElementById('renewForm').submit();"
                                                >
                                                    {{ $p }} дней
                                                </button>
                                            @endforeach
                                        </div>
                                        <div class="mt-3">
                                            <label for="renewPromoCodeField" class="block text-[11px] font-semibold text-slate-300/80">Промокод (опционально)</label>
                                            <input
                                                id="renewPromoCodeField"
                                                type="text"
                                                class="mt-1 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                            >
                                        </div>
                                        <div class="mt-3">
                                            <button type="button" class="w-full inline-flex items-center justify-center rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-xs font-semibold text-slate-200 shadow-sm transition hover:bg-black/15 hover:ring-1 hover:ring-white/10" onclick="document.getElementById('renewModal')?.classList.add('hidden')">
                                                Отмена
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <form method="POST" action="{{ route('server.auto-start', $server) }}" class="pt-2" onsubmit="this.querySelector('button[type=submit]')?.setAttribute('disabled','disabled');">
                        @csrf
                        <input type="hidden" name="enabled" value="0">
                        <label class="flex items-center gap-2 text-xs text-slate-300/80 {{ $isExpired ? 'opacity-60 cursor-not-allowed pointer-events-none' : '' }}">
                            <input
                                type="checkbox"
                                name="enabled"
                                value="1"
                                {{ $server->auto_start_enabled ? 'checked' : '' }}
                                {{ $isExpired ? 'disabled' : '' }}
                                class="h-4 w-4 rounded border-white/10 bg-black/10 text-sky-500 focus:ring-sky-500"
                                onchange="this.form.submit()"
                            >
                            Автоподнятие сервера
                        </label>
                    </form>
                @endif
            </div>
        </div>
    </div>
</div>


<div class="mt-6 grid gap-6 lg:grid-cols-2">
    <div class="overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] text-slate-100 shadow-sm transition-shadow duration-300 hover:shadow-md">
        <div class="border-b border-white/10 bg-black/10 px-5 py-3 text-[11px] uppercase tracking-wide text-slate-300/70">
            Статистика и графики
        </div>
        <div class="p-5">
            <div class="h-44 w-full">
                <canvas id="serverLoadChart" class="h-full w-full"></canvas>
            </div>
        </div>
    </div>

    <div class="overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] text-slate-100 shadow-sm transition-shadow duration-300 hover:shadow-md">
        <div class="border-b border-white/10 bg-black/10 px-5 py-3 text-[11px] uppercase tracking-wide text-slate-300/70">
            Быстрый доступ
        </div>
        <div class="p-5">
            <div class="grid gap-2 text-sm">
                <div class="group flex items-center justify-between gap-3 rounded-2xl border border-white/10 bg-black/10 px-4 py-3 transition hover:bg-black/15 hover:ring-1 hover:ring-white/10">
                    <div class="text-slate-200">Основные настройки (server.cfg)</div>
                    <a href="{{ route('server.show', $server) }}?tab=settings" class="inline-flex items-center rounded-xl border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-semibold text-slate-200 shadow-sm transition hover:bg-black/15">
                        Перейти
                    </a>
                </div>
                <div class="group flex items-center justify-between gap-3 rounded-2xl border border-white/10 bg-black/10 px-4 py-3 transition hover:bg-black/15 hover:ring-1 hover:ring-white/10">
                    <div class="text-slate-200">Снимки консоли (StartLogs)</div>
                    <a href="#" class="inline-flex items-center rounded-xl border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-semibold text-slate-200 shadow-sm transition hover:bg-black/15">
                        Перейти
                    </a>
                </div>
                <div class="group flex items-center justify-between gap-3 rounded-2xl border border-white/10 bg-black/10 px-4 py-3 transition hover:bg-black/15 hover:ring-1 hover:ring-white/10">
                    <div class="text-slate-200">Блокировка по оборудованию (Firewall)</div>
                    <a href="#" class="inline-flex items-center rounded-xl border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-semibold text-slate-200 shadow-sm transition hover:bg-black/15">
                        Перейти
                    </a>
                </div>
                <div class="group flex items-center justify-between gap-3 rounded-2xl border border-white/10 bg-black/10 px-4 py-3 transition hover:bg-black/15 hover:ring-1 hover:ring-white/10">
                    <div class="text-slate-200">Планировщик задач (CronTab)</div>
                    <a href="#" class="inline-flex items-center rounded-xl border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-semibold text-slate-200 shadow-sm transition hover:bg-black/15">
                        Перейти
                    </a>
                </div>
            </div>
        </div>
    </div>
</div>

<div class="mt-6 overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] text-slate-100 shadow-sm transition-shadow duration-300 hover:shadow-md">
    <div class="border-b border-white/10 bg-black/10 px-5 py-3 text-[11px] uppercase tracking-wide text-slate-300/70">
        Игроки онлайн
    </div>
    <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-white/10 text-sm">
            <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                <tr>
                    <th class="px-4 py-3 text-left">#</th>
                    <th class="px-4 py-3 text-left">Ник</th>
                    <th class="px-4 py-3 text-left">Фраги</th>
                    <th class="px-4 py-3 text-left">Пинг</th>
                </tr>
            </thead>
            <tbody class="divide-y divide-white/10">
                <tr>
                    <td colspan="4" class="px-4 py-6 text-center text-slate-300/70">
                        Нет данных
                    </td>
                </tr>
            </tbody>
        </table>
    </div>
</div>
