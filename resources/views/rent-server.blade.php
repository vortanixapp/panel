@extends('layouts.app-user')

@section('title', 'Аренда сервера')
@section('page_title', 'Аренда сервера')

@section('content')
    <section class="py-6 md:py-8">
        <div class="w-full px-4 sm:px-6 lg:px-8">
            <div class="mb-6 flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Аренда сервера</h1>
                    <p class="mt-1 text-sm text-slate-300">Выберите параметры для аренды игрового сервера</p>
                </div>
                <div class="text-[12px] text-slate-300/70">
                    Сначала выберите локацию, игру и период — затем подтвердите заказ.
                </div>
            </div>

            @if (session('error'))
                <div class="mb-6 rounded-2xl border border-red-200 bg-red-50/70 px-4 py-3 text-sm text-red-800 shadow-sm">
                    {{ session('error') }}
                </div>
            @endif
            @if (session('success'))
                <div class="mb-6 rounded-2xl border border-green-200 bg-green-50/70 px-4 py-3 text-sm text-green-800 shadow-sm">
                    {{ session('success') }}
                </div>
            @endif

            <form id="rent-form" method="POST" action="{{ route('rent-server.post') }}" class="grid gap-6 lg:grid-cols-2">
                @csrf

                <!-- Левый блок -->
                <div class="flex flex-col gap-6">
                    <div class="overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm transition-shadow duration-300 hover:shadow-md">
                        <div class="border-b border-white/10 bg-black/10 px-5 py-4">
                            <h2 class="text-sm font-semibold text-slate-100">Параметры сервера</h2>
                            <p class="mt-1 text-[12px] text-slate-300/70">Настройте конфигурацию и рассчитайте стоимость.</p>
                        </div>

                        <div class="p-5">
                            <div class="grid gap-4">
                                <div class="grid gap-4 md:grid-cols-2">
                            <div>
                                <label for="location_id" class="block text-xs font-semibold text-slate-500 mb-1">Локация</label>
                                <select
                                    id="location_id"
                                    name="location_id"
                                    required
                                    class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    onchange="autoSubmit()"
                                >
                                    <option value="">Выберите локацию</option>
                                    @foreach($locations as $location)
                                        <option value="{{ $location->id }}" {{ $request->location_id == $location->id ? 'selected' : '' }}>
                                            {{ $location->name }}
                                        </option>
                                    @endforeach
                                </select>
                            </div>

                            <div>
                                <label for="game_id" class="block text-xs font-semibold text-slate-500 mb-1">Игра</label>
                                <select
                                    id="game_id"
                                    name="game_id"
                                    required
                                    class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    onchange="autoSubmit()"
                                >
                                    <option value="">Выберите игру</option>
                                    @foreach($games as $game)
                                        <option value="{{ $game->id }}" {{ $request->game_id == $game->id ? 'selected' : '' }}>
                                            {{ $game->name }}
                                        </option>
                                    @endforeach
                                </select>
                            </div>

                            <div>
                                <label for="game_version_id" class="block text-xs font-semibold text-slate-500 mb-1">Версия</label>
                                <select
                                    id="game_version_id"
                                    name="game_version_id"
                                    class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    onchange="autoSubmit()"
                                >
                                    <option value="">Авто</option>
                                    @foreach(($gameVersions ?? []) as $version)
                                        <option value="{{ $version->id }}" {{ $request->game_version_id == $version->id || (! $request->filled('game_version_id') && isset($selectedGameVersion) && $selectedGameVersion && $selectedGameVersion->id === $version->id) ? 'selected' : '' }}>
                                            {{ $version->name }}
                                        </option>
                                    @endforeach
                                </select>
                            </div>
                                </div>

                            <div class="grid gap-4 md:grid-cols-2">
                                <div>
                                    <label for="tariff_id" class="block text-xs font-semibold text-slate-500 mb-1">Тариф</label>
                                    <select
                                        id="tariff_id"
                                        name="tariff_id"
                                        class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                        onchange="autoSubmit()"
                                    >
                                        <option value="">Выберите тариф</option>
                                        @foreach($tariffs as $tariff)
                                            <option value="{{ $tariff->id }}" {{ $request->tariff_id == $tariff->id || (! $request->filled('tariff_id') && $selectedTariff && $selectedTariff->id === $tariff->id) ? 'selected' : '' }}>
                                                {{ $tariff->name }} ({{ ($tariff->billing_type ?? 'resources') === 'slots' ? 'слоты' : 'ресурсы' }})
                                            </option>
                                        @endforeach
                                    </select>
                                </div>

                                <div>
                                    <label for="period" class="block text-xs font-semibold text-slate-500 mb-1">Период аренды</label>
                                    <select
                                        id="period"
                                        name="period"
                                        class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                        onchange="autoSubmit()"
                                    >
                                        <option value="">Выберите период</option>
                                        <option value="15" {{ $request->period == 15 ? 'selected' : '' }}>15 дней</option>
                                        <option value="30" {{ $request->period == 30 ? 'selected' : '' }}>30 дней</option>
                                        <option value="60" {{ $request->period == 60 ? 'selected' : '' }}>60 дней</option>
                                        <option value="180" {{ $request->period == 180 ? 'selected' : '' }}>180 дней</option>
                                    </select>
                                </div>
                            </div>

                            @if($selectedTariff && ($order['billing_type'] ?? ($selectedTariff->billing_type ?? 'resources')) === 'slots')
                                <div>
                                    <div class="flex items-center justify-between mb-1">
                                        <label for="slots" class="block text-xs font-semibold text-slate-500">Слоты</label>
                                        <span class="text-xs font-semibold text-slate-100"><span id="slots_value">{{ (int) ($order['slots'] ?? ($request->slots ?? 10)) }}</span></span>
                                    </div>
                                    <input
                                        id="slots"
                                        name="slots"
                                        type="range"
                                        min="{{ $selectedTariff ? (int) ($selectedTariff->min_slots ?? 1) : 1 }}"
                                        max="{{ $selectedTariff ? (int) ($selectedTariff->max_slots ?? 1000) : 1000 }}"
                                        step="1"
                                        value="{{ (int) ($order['slots'] ?? ($request->slots ?? 10)) }}"
                                        class="w-full"
                                        oninput="document.getElementById('slots_value').textContent = this.value"
                                        onchange="autoSubmit()"
                                    >
                                </div>
                            @else
                                <input type="hidden" name="slots" value="1">
                            @endif

                            @if($selectedTariff && ($order['billing_type'] ?? ($selectedTariff->billing_type ?? 'resources')) === 'resources')
                                <div class="grid gap-4">
                                    <div>
                                        <div class="flex items-center justify-between mb-1">
                                            <label for="cpu_cores" class="block text-xs font-semibold text-slate-500">CPU</label>
                                            <span class="text-xs font-semibold text-slate-100"><span id="cpu_cores_value">{{ (int) ($order['cpu_cores'] ?? 0) }}</span> vCPU</span>
                                        </div>
                                        <input
                                            id="cpu_cores"
                                            name="cpu_cores"
                                            type="range"
                                            min="{{ (int) ($order['cpu_min'] ?? 0) }}"
                                            max="{{ (int) ($order['cpu_max'] ?? 0) }}"
                                            step="{{ (int) ($order['cpu_step'] ?? 1) }}"
                                            value="{{ (int) ($order['cpu_cores'] ?? 0) }}"
                                            class="w-full"
                                            onchange="autoSubmit()"
                                        >
                                    </div>
                                    <div>
                                        <div class="flex items-center justify-between mb-1">
                                            <label for="ram_gb" class="block text-xs font-semibold text-slate-500">RAM</label>
                                            <span class="text-xs font-semibold text-slate-100"><span id="ram_gb_value">{{ (int) ($order['ram_gb'] ?? 0) }}</span> ГБ</span>
                                        </div>
                                        <input
                                            id="ram_gb"
                                            name="ram_gb"
                                            type="range"
                                            min="{{ (int) ($order['ram_min'] ?? 0) }}"
                                            max="{{ (int) ($order['ram_max'] ?? 0) }}"
                                            step="{{ (int) ($order['ram_step'] ?? 1) }}"
                                            value="{{ (int) ($order['ram_gb'] ?? 0) }}"
                                            class="w-full"
                                            onchange="autoSubmit()"
                                        >
                                    </div>
                                    <div>
                                        <div class="flex items-center justify-between mb-1">
                                            <label for="disk_gb" class="block text-xs font-semibold text-slate-500">Диск</label>
                                            <span class="text-xs font-semibold text-slate-100"><span id="disk_gb_value">{{ (int) ($order['disk_gb'] ?? 0) }}</span> ГБ</span>
                                        </div>
                                        <input
                                            id="disk_gb"
                                            name="disk_gb"
                                            type="range"
                                            min="{{ (int) ($order['disk_min'] ?? 0) }}"
                                            max="{{ (int) ($order['disk_max'] ?? 0) }}"
                                            step="{{ (int) ($order['disk_step'] ?? 1) }}"
                                            value="{{ (int) ($order['disk_gb'] ?? 0) }}"
                                            class="w-full"
                                            onchange="autoSubmit()"
                                        >
                                    </div>
                                </div>
                            @endif

                            @if($selectedLocation && (is_array($selectedLocation->ip_pool) && count($selectedLocation->ip_pool) > 0))
                                <div>
                                    <label for="ip_choice" class="block text-xs font-semibold text-slate-500 mb-1">IP адрес</label>
                                    <select
                                        id="ip_choice"
                                        name="ip_choice"
                                        class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                        onchange="autoSubmit()"
                                    >
                                        <option value="">Автоматически</option>
                                        @foreach($selectedLocation->ip_pool as $ip)
                                            <option value="{{ $ip }}" {{ ($order['ip_choice'] ?? '') === $ip ? 'selected' : '' }}>{{ $ip }}</option>
                                        @endforeach
                                    </select>
                                </div>
                            @endif

                            @if($selectedTariff && ($selectedTariff->allow_antiddos ?? false))
                                <div class="flex items-center gap-3">
                                    <label class="inline-flex items-center gap-2 text-sm text-slate-200">
                                        <input
                                            type="checkbox"
                                            name="antiddos_enabled"
                                            value="1"
                                            class="h-4 w-4 rounded border-white/20 bg-black/10 text-slate-100 focus:ring-sky-500"
                                            {{ ($order['antiddos_enabled'] ?? false) ? 'checked' : '' }}
                                            onchange="autoSubmit()"
                                        >
                                        <span>Anti-DDoS</span>
                                    </label>
                                </div>
                            @endif

                            @php
                                $gameCode = strtolower((string) (($selectedTariff->game->code ?? '') ?: ($selectedTariff->game->slug ?? '')));
                                $isCs = $gameCode !== '' && (
                                    str_contains($gameCode, 'cs')
                                    || str_contains($gameCode, 'cstrike')
                                    || str_contains($gameCode, 'counter')
                                );
                                $hasTickrate = $isCs && (
                                    str_contains($gameCode, 'csgo')
                                    || str_contains($gameCode, 'css')
                                    || str_contains($gameCode, 'source')
                                    || str_contains($gameCode, 'v34')
                                );
                            @endphp

                            @if($selectedTariff && $isCs)
                                <div class="grid gap-4 md:grid-cols-2">
                                    <div>
                                        <label for="server_fps" class="block text-xs font-semibold text-slate-500 mb-1">FPS</label>
                                        <select
                                            id="server_fps"
                                            name="server_fps"
                                            class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                            onchange="autoSubmit()"
                                        >
                                            <option value="0" {{ (int) ($order['server_fps'] ?? 0) === 0 ? 'selected' : '' }}>По умолчанию</option>
                                            <option value="500" {{ (int) ($order['server_fps'] ?? 0) === 500 ? 'selected' : '' }}>500</option>
                                            <option value="1000" {{ (int) ($order['server_fps'] ?? 0) === 1000 ? 'selected' : '' }}>1000</option>
                                        </select>
                                    </div>

                                    @if($hasTickrate)
                                        <div>
                                            <label for="server_tickrate" class="block text-xs font-semibold text-slate-500 mb-1">Тикрейт</label>
                                            <select
                                                id="server_tickrate"
                                                name="server_tickrate"
                                                class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                                onchange="autoSubmit()"
                                            >
                                                <option value="0" {{ (int) ($order['server_tickrate'] ?? 0) === 0 ? 'selected' : '' }}>По умолчанию</option>
                                                <option value="66" {{ (int) ($order['server_tickrate'] ?? 0) === 66 ? 'selected' : '' }}>66</option>
                                                <option value="100" {{ (int) ($order['server_tickrate'] ?? 0) === 100 ? 'selected' : '' }}>100</option>
                                            </select>
                                        </div>
                                    @endif
                                </div>
                            @endif

                            <div>
                                <label for="promo_code" class="block text-xs font-semibold text-slate-500 mb-1">Промо-код (опционально)</label>
                                <input
                                    id="promo_code"
                                    name="promo_code"
                                    type="text"
                                    value="{{ $request->promo_code }}"
                                    class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                            </div>


                            <div class="rounded-2xl border border-white/10 bg-black/10 px-4 py-3">
                                <div class="text-[11px] font-semibold uppercase tracking-wide text-slate-300/70">Загруженность локации</div>
                                <div class="mt-1 text-sm text-slate-200">
                                    @if($request->location_id)
                                        Низкая загрузка (доступно {{ rand(10, 50) }} серверов)
                                    @else
                                        Выберите локацию для просмотра загрузки
                                    @endif
                                </div>
                            </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Правый блок -->
                <div class="flex flex-col gap-6">
                    @if($selectedTariff && $request->period && $calculatedCost !== null)
                        <div class="overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm transition-shadow duration-300 hover:shadow-md">
                            <div class="border-b border-white/10 bg-black/10 px-5 py-4">
                                <h2 class="text-sm font-semibold text-slate-100">Информация о заказе</h2>
                                <p class="mt-1 text-[12px] text-slate-300/70">Проверьте параметры перед оплатой.</p>
                            </div>

                            <div class="p-5">
                                <div class="divide-y divide-white/10 text-sm">
                                    <div class="flex items-center justify-between gap-3 py-2">
                                        <span class="text-slate-300/70">Игра</span>
                                        <span class="font-semibold text-slate-100">{{ $selectedTariff->game->name }}</span>
                                    </div>
                                    <div class="flex items-center justify-between gap-3 py-2">
                                        <span class="text-slate-300/70">Локация</span>
                                        <span class="font-semibold text-slate-100">{{ $selectedTariff->location->name }}</span>
                                    </div>
                                    <div class="flex items-center justify-between gap-3 py-2">
                                        <span class="text-slate-300/70">Тариф</span>
                                        <span class="font-semibold text-slate-100">{{ $selectedTariff->name }}</span>
                                    </div>
                                    <div class="grid gap-2 py-3 sm:grid-cols-2">
                                        <div class="rounded-2xl border border-white/10 bg-black/10 px-4 py-3">
                                            <div class="text-[11px] uppercase tracking-wide text-slate-300/70">CPU</div>
                                            <div class="mt-1 text-sm font-semibold text-slate-100">{{ (int) ($order['cpu_cores'] ?? ($selectedTariff->cpu_cores ?? 0)) }} vCPU</div>
                                        </div>
                                        <div class="rounded-2xl border border-white/10 bg-black/10 px-4 py-3">
                                            <div class="text-[11px] uppercase tracking-wide text-slate-300/70">RAM</div>
                                            <div class="mt-1 text-sm font-semibold text-slate-100">{{ (int) ($order['ram_gb'] ?? ($selectedTariff->ram_gb ?? 0)) }} GB</div>
                                        </div>
                                        <div class="rounded-2xl border border-white/10 bg-black/10 px-4 py-3">
                                            <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Диск</div>
                                            <div class="mt-1 text-sm font-semibold text-slate-100">{{ (int) ($order['disk_gb'] ?? ($selectedTariff->disk_gb ?? 0)) }} GB</div>
                                        </div>
                                        @if(($order['billing_type'] ?? ($selectedTariff->billing_type ?? 'resources')) === 'slots')
                                            <div class="rounded-2xl border border-white/10 bg-black/10 px-4 py-3">
                                                <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Слоты</div>
                                                <div class="mt-1 text-sm font-semibold text-slate-100">{{ (int) ($order['slots'] ?? ($request->slots ?? 10)) }}</div>
                                            </div>
                                        @endif
                                    </div>
                                    @if(($order['ip_choice'] ?? '') !== '')
                                        <div class="flex items-center justify-between gap-3 py-2">
                                            <span class="text-slate-300/70">IP</span>
                                            <span class="font-semibold text-slate-100">{{ $order['ip_choice'] }}</span>
                                        </div>
                                    @endif
                                    @if(($order['antiddos_enabled'] ?? false) === true)
                                        <div class="flex items-center justify-between gap-3 py-2">
                                            <span class="text-slate-300/70">Anti-DDoS</span>
                                            <span class="font-semibold text-slate-100">Включен</span>
                                        </div>
                                    @endif
                                    @if((int) ($order['server_fps'] ?? 0) > 0)
                                        <div class="flex items-center justify-between gap-3 py-2">
                                            <span class="text-slate-300/70">FPS</span>
                                            <span class="font-semibold text-slate-100">{{ (int) $order['server_fps'] }}</span>
                                        </div>
                                    @endif
                                    @if((int) ($order['server_tickrate'] ?? 0) > 0)
                                        <div class="flex items-center justify-between gap-3 py-2">
                                            <span class="text-slate-300/70">Tickrate</span>
                                            <span class="font-semibold text-slate-100">{{ (int) $order['server_tickrate'] }}</span>
                                        </div>
                                    @endif
                                    <div class="flex items-center justify-between gap-3 py-2">
                                        <span class="text-slate-300/70">Период</span>
                                        <span class="font-semibold text-slate-100">{{ $request->period }} дней</span>
                                    </div>
                                    @if($request->promo_code)
                                        <div class="flex items-center justify-between gap-3 py-2">
                                            <span class="text-slate-300/70">Промо-код</span>
                                            <span class="font-semibold text-slate-100">{{ $request->promo_code }}</span>
                                        </div>
                                    @endif
                                </div>

                                <div class="mt-4 rounded-2xl border border-white/10 bg-black/10 px-4 py-3">
                                    <div class="flex items-center justify-between">
                                        <span class="text-sm font-semibold text-slate-100">Итого</span>
                                        <span class="text-base font-semibold text-slate-100">
                                            {{ number_format((float) $calculatedCost, 2) }} ₽
                                        </span>
                                    </div>
                                </div>

                                <button
                                    type="submit"
                                    class="w-full mt-5 inline-flex items-center justify-center rounded-xl bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-emerald-700 active:scale-[0.99]"
                                >
                                    Заказать сервер
                                </button>
                            </div>
                        </div>
                    @else
                        <div class="flex flex-1 flex-col overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm transition-shadow duration-300 hover:shadow-md">
                            <div class="flex flex-1 items-center justify-center p-8">
                                <div class="mx-auto max-w-md text-center">
                                    <div class="mx-auto mb-4 inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-tr from-sky-50 to-indigo-50 ring-1 ring-sky-100">
                                        <svg class="h-6 w-6 text-sky-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                                        </svg>
                                    </div>
                                    <h3 class="text-base font-semibold text-slate-100">Выберите параметры</h3>
                                    <p class="mt-1 text-sm text-slate-300/70">Заполните форму слева — мы покажем итог и кнопку заказа.</p>
                                </div>
                            </div>
                        </div>
                    @endif
                </div>
            </form>
        </div>
    </section>
@endsection

<script>
let __rentUpdateTimer = null;
let __rentUpdateInFlight = false;

function __rentBuildQuery(form) {
    const params = new URLSearchParams();
    const fd = new FormData(form);

    for (const [key, value] of fd.entries()) {
        params.append(key, value);
    }

    return params;
}

async function __rentFetchAndReplaceForm() {
    const form = document.getElementById('rent-form');
    if (!form) {
        return;
    }

    if (__rentUpdateInFlight) {
        return;
    }
    __rentUpdateInFlight = true;

    try {
        const params = __rentBuildQuery(form);
        const url = '{{ route("rent-server") }}' + (params.toString() ? ('?' + params.toString()) : '');

        const resp = await fetch(url, {
            method: 'GET',
            headers: {
                'X-Requested-With': 'XMLHttpRequest',
            },
        });

        if (!resp.ok) {
            return;
        }

        const html = await resp.text();
        const doc = new DOMParser().parseFromString(html, 'text/html');
        const newForm = doc.getElementById('rent-form');
        const currentForm = document.getElementById('rent-form');

        if (newForm && currentForm) {
            currentForm.replaceWith(newForm);
            window.history.replaceState(null, '', url);
        }
    } catch (e) {
        // ignore
    } finally {
        __rentUpdateInFlight = false;
    }
}

function autoSubmit() {
    if (__rentUpdateTimer) {
        clearTimeout(__rentUpdateTimer);
    }
    __rentUpdateTimer = setTimeout(__rentFetchAndReplaceForm, 150);
}

</script>

<style>
    #cpu_cores,
    #ram_gb,
    #disk_gb {
        -webkit-appearance: none;
        appearance: none;
        width: 100%;
        height: 0.375rem;
        background: rgba(255, 255, 255, 0.12);
        border-radius: 9999px;
        outline: none;
    }

    #cpu_cores::-webkit-slider-runnable-track,
    #ram_gb::-webkit-slider-runnable-track,
    #disk_gb::-webkit-slider-runnable-track {
        height: 0.375rem;
        background: rgba(255, 255, 255, 0.12);
        border-radius: 9999px;
    }

    #cpu_cores::-webkit-slider-thumb,
    #ram_gb::-webkit-slider-thumb,
    #disk_gb::-webkit-slider-thumb {
        -webkit-appearance: none;
        appearance: none;
        width: 1rem;
        height: 1rem;
        margin-top: -0.3125rem;
        background: #38bdf8;
        border: 2px solid rgba(0, 0, 0, 0.25);
        border-radius: 9999px;
        cursor: pointer;
    }

    #cpu_cores::-moz-range-track,
    #ram_gb::-moz-range-track,
    #disk_gb::-moz-range-track {
        height: 0.375rem;
        background: rgba(255, 255, 255, 0.12);
        border-radius: 9999px;
    }

    #cpu_cores::-moz-range-thumb,
    #ram_gb::-moz-range-thumb,
    #disk_gb::-moz-range-thumb {
        width: 1rem;
        height: 1rem;
        background: #38bdf8;
        border: 2px solid rgba(0, 0, 0, 0.25);
        border-radius: 9999px;
        cursor: pointer;
    }

    #cpu_cores::-moz-range-progress,
    #ram_gb::-moz-range-progress,
    #disk_gb::-moz-range-progress {
        height: 0.375rem;
        background: rgba(56, 189, 248, 0.5);
        border-radius: 9999px;
    }

    #rent-form select option,
    #rent-form select optgroup {
        background-color: #242f3d;
        color: #e2e8f0;
    }
</style>
