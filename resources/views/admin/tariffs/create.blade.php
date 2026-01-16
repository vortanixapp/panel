@extends('layouts.app-admin')

@section('title', 'Создание тарифа')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Создание тарифа</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Заполните информацию о новом тарифе.</p>
                </div>
                <a
                    href="{{ route('admin.tariffs.index') }}"
                    class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                >
                    ← Назад к тарифам
                </a>
            </div>

            <form method="POST" action="{{ route('admin.tariffs.store') }}" class="space-y-6 rounded-2xl border border-white/10 bg-[#242f3d] p-6 shadow-sm shadow-black/20" x-data="{ billingType: '{{ old('billing_type', 'resources') }}', cpuCores: {{ (int) old('cpu_cores', 1) }} }">
                @csrf

                @if ($errors->any())
                    <div class="rounded-md border border-rose-500/20 bg-rose-500/10 px-4 py-3 text-xs text-rose-200">
                        <ul class="list-disc space-y-1 pl-4">
                            @foreach ($errors->all() as $error)
                                <li>{{ $error }}</li>
                            @endforeach
                        </ul>
                    </div>
                @endif

                <div class="grid gap-6 md:grid-cols-3">
                    <div class="space-y-4 md:order-1">
                        <div>
                            <div class="text-xs font-semibold text-slate-100">База</div>
                            <div class="mt-1 text-[11px] text-slate-300/70">Название, локация/игра и тип тарифа.</div>
                        </div>

                        <div class="space-y-1">
                            <label for="name" class="text-xs font-medium text-slate-200">Имя тарифа</label>
                            <input
                                id="name"
                                name="name"
                                type="text"
                                value="{{ old('name') }}"
                                required
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                            <p class="mt-1 text-[11px] text-slate-300/70">Название, которое увидит пользователь при выборе тарифа.</p>
                        </div>

                        <div class="grid gap-4">
                            <div class="space-y-1">
                                <label for="location_id" class="text-xs font-medium text-slate-200">Локация</label>
                                <select
                                    id="location_id"
                                    name="location_id"
                                    required
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                                    <option value="">— Выберите локацию —</option>
                                    @foreach($locations as $location)
                                        <option value="{{ $location->id }}" {{ old('location_id') == $location->id ? 'selected' : '' }}>
                                            {{ $location->name }}
                                        </option>
                                    @endforeach
                                </select>
                                <p class="mt-1 text-[11px] text-slate-300/70">Где будут размещаться серверы по этому тарифу.</p>
                            </div>

                            <div class="space-y-1">
                                <label for="game_id" class="text-xs font-medium text-slate-200">Игра</label>
                                <select
                                    id="game_id"
                                    name="game_id"
                                    required
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                                    <option value="">— Выберите игру —</option>
                                    @foreach($games as $game)
                                        <option value="{{ $game->id }}" {{ old('game_id') == $game->id ? 'selected' : '' }}>
                                            {{ $game->name }}
                                        </option>
                                    @endforeach
                                </select>
                                <p class="mt-1 text-[11px] text-slate-300/70">Определяет шаблон/инсталляцию и доступные параметры игры.</p>
                            </div>

                            <div class="space-y-1">
                                <label for="billing_type" class="text-xs font-medium text-slate-200">Тип тарифа</label>
                                <select
                                    id="billing_type"
                                    name="billing_type"
                                    required
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    x-model="billingType"
                                >
                                    <option value="resources" {{ old('billing_type', 'resources') === 'resources' ? 'selected' : '' }}>Оплата за ресурсы</option>
                                    <option value="slots" {{ old('billing_type') === 'slots' ? 'selected' : '' }}>Оплата за слоты</option>
                                </select>
                                <div class="mt-1 space-y-0.5 text-[11px] text-slate-300/70">
                                    <div x-show="billingType === 'resources'" x-cloak>Пользователь платит за CPU/RAM/диск (по заданным ценам и диапазонам).</div>
                                    <div x-show="billingType === 'slots'" x-cloak>Пользователь платит за количество слотов (мин/макс + цена за слот).</div>
                                </div>
                            </div>
                        </div>

                        <div class="grid gap-4">
                            <div class="space-y-1">
                                <label class="text-xs font-medium text-slate-200">Аренда</label>
                                <div class="flex flex-wrap gap-2">
                                    @foreach([15, 30, 60, 180] as $days)
                                        <label class="inline-flex">
                                            <input
                                                type="checkbox"
                                                name="rental_periods[]"
                                                value="{{ $days }}"
                                                {{ in_array($days, old('rental_periods', [])) ? 'checked' : '' }}
                                                class="peer sr-only"
                                            >
                                            <span class="inline-flex items-center justify-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-medium text-slate-200 shadow-sm hover:bg-black/15 peer-checked:border-sky-500 peer-checked:bg-sky-600 peer-checked:text-white">
                                                {{ $days }}
                                            </span>
                                        </label>
                                    @endforeach
                                </div>
                                <p class="mt-1 text-[11px] text-slate-300/70">Какие периоды доступны при первичной покупке.</p>
                            </div>

                            <div class="space-y-1">
                                <label class="text-xs font-medium text-slate-200">Продление</label>
                                <div class="flex flex-wrap gap-2">
                                    @foreach([15, 30, 60, 180] as $days)
                                        <label class="inline-flex">
                                            <input
                                                type="checkbox"
                                                name="renewal_periods[]"
                                                value="{{ $days }}"
                                                {{ in_array($days, old('renewal_periods', [])) ? 'checked' : '' }}
                                                class="peer sr-only"
                                            >
                                            <span class="inline-flex items-center justify-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs font-medium text-slate-200 shadow-sm hover:bg-black/15 peer-checked:border-sky-500 peer-checked:bg-sky-600 peer-checked:text-white">
                                                {{ $days }}
                                            </span>
                                        </label>
                                    @endforeach
                                </div>
                                <p class="mt-1 text-[11px] text-slate-300/70">Какие периоды доступны при продлении.</p>
                            </div>
                        </div>
                    </div>

                    <div class="space-y-4 md:order-3">
                        <div>
                            <div class="text-xs font-semibold text-slate-100">Биллинг</div>
                            <div class="mt-1 text-[11px] text-slate-300/70">Формула цены и доп. услуги.</div>
                        </div>

                        <div class="space-y-4" x-show="billingType === 'resources'" x-cloak>
                            <div>
                                <div class="text-xs font-semibold text-slate-100">Оплата за ресурсы</div>
                                <div class="mt-1 text-[11px] text-slate-300/70">Цены и диапазоны.</div>
                            </div>

                            <div class="grid gap-4 md:grid-cols-3">
                                <div class="space-y-1">
                                    <label for="price_per_cpu_core" class="text-xs font-medium text-slate-200">CPU ₽/ядро</label>
                                    <input
                                        id="price_per_cpu_core"
                                        name="price_per_cpu_core"
                                        type="number"
                                        step="0.01"
                                        min="0"
                                        value="{{ old('price_per_cpu_core', 0) }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>

                                <div class="space-y-1">
                                    <label for="price_per_ram_gb" class="text-xs font-medium text-slate-200">RAM ₽/ГБ</label>
                                    <input
                                        id="price_per_ram_gb"
                                        name="price_per_ram_gb"
                                        type="number"
                                        step="0.01"
                                        min="0"
                                        value="{{ old('price_per_ram_gb', 0) }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>

                                <div class="space-y-1">
                                    <label for="price_per_disk_gb" class="text-xs font-medium text-slate-200">Диск ₽/ГБ</label>
                                    <input
                                        id="price_per_disk_gb"
                                        name="price_per_disk_gb"
                                        type="number"
                                        step="0.01"
                                        min="0"
                                        value="{{ old('price_per_disk_gb', 0) }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>
                            </div>
                        </div>

                        <div class="space-y-4" x-show="billingType === 'slots'" x-cloak>
                            <div>
                                <div class="text-xs font-semibold text-slate-100">Оплата за слоты</div>
                                <div class="mt-1 text-[11px] text-slate-300/70">Цена слотов и базовая цена по периодам.</div>
                            </div>

                            <div class="grid gap-4 md:grid-cols-3">
                                <div class="space-y-1">
                                    <label for="price_per_slot" class="text-xs font-medium text-slate-200">Цена/слот (₽/мес)</label>
                                    <input
                                        id="price_per_slot"
                                        name="price_per_slot"
                                        type="number"
                                        step="0.01"
                                        min="0"
                                        value="{{ old('price_per_slot') }}"
                                        x-bind:required="billingType === 'slots'"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                    <p class="mt-1 text-[11px] text-slate-300/70">Итог = слоты × цена/слот (с учётом периода).</p>
                                </div>

                                <div class="space-y-1">
                                    <label for="min_slots" class="text-xs font-medium text-slate-200">Мин. слоты</label>
                                    <input
                                        id="min_slots"
                                        name="min_slots"
                                        type="number"
                                        min="1"
                                        value="{{ old('min_slots', 1) }}"
                                        x-bind:required="billingType === 'slots'"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                    <p class="mt-1 text-[11px] text-slate-300/70">Минимум слотов, доступный пользователю.</p>
                                </div>

                                <div class="space-y-1">
                                    <label for="max_slots" class="text-xs font-medium text-slate-200">Макс. слоты</label>
                                    <input
                                        id="max_slots"
                                        name="max_slots"
                                        type="number"
                                        min="1"
                                        value="{{ old('max_slots', 100) }}"
                                        x-bind:required="billingType === 'slots'"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                    <p class="mt-1 text-[11px] text-slate-300/70">Максимум слотов, доступный пользователю.</p>
                                </div>
                            </div>
                        </div>

                        <div class="grid gap-4">
                            <div class="space-y-1" x-show="billingType === 'resources'" x-cloak>
                                <label for="base_price_monthly" class="text-xs font-medium text-slate-200">Базовая цена (₽/мес)</label>
                                <input
                                    id="base_price_monthly"
                                    name="base_price_monthly"
                                    type="number"
                                    step="0.01"
                                    min="0"
                                    value="{{ old('base_price_monthly', 0) }}"
                                    x-bind:required="billingType === 'resources'"
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                                <p class="mt-1 text-[11px] text-slate-300/70">Базовая стоимость в месяц (к ней добавляется цена за CPU/RAM/Disk).</p>
                            </div>

                            <div class="grid gap-4 md:grid-cols-2">
                                <div class="space-y-1">
                                    <label for="position" class="text-xs font-medium text-slate-200">Положение</label>
                                    <input
                                        id="position"
                                        name="position"
                                        type="number"
                                        min="0"
                                        value="{{ old('position', 0) }}"
                                        required
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                    <p class="mt-1 text-[11px] text-slate-300/70">Сортировка в списках (меньше — выше).</p>
                                </div>
                                <div class="space-y-2">
                                    <div class="flex items-center justify-between gap-4">
                                        <div class="min-w-0">
                                            <div class="text-xs font-medium text-slate-200">Доступность</div>
                                            <div class="mt-1 text-[11px] text-slate-300/70">Показывать тариф пользователям.</div>
                                        </div>

                                        <label class="relative inline-flex shrink-0 cursor-pointer items-center">
                                            <input
                                                type="checkbox"
                                                name="is_available"
                                                value="1"
                                                class="peer sr-only"
                                                {{ old('is_available', 1) ? 'checked' : '' }}
                                            >
                                            <span class="h-5 w-9 rounded-full border border-white/10 bg-black/10 transition peer-checked:border-sky-500 peer-checked:bg-sky-600"></span>
                                            <span class="pointer-events-none absolute left-0.5 top-0.5 h-4 w-4 rounded-full bg-white/80 shadow transition peer-checked:translate-x-4"></span>
                                        </label>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="space-y-4 md:order-2">
                        <div>
                            <div class="text-xs font-semibold text-slate-100">Ограничения и цены периода</div>
                            <div class="mt-1 text-[11px] text-slate-300/70">Ресурсы сервера и базовая цена по дням.</div>
                        </div>

                        <div class="grid gap-4">
                            <div class="space-y-2">
                                <div class="flex items-center justify-between gap-4">
                                    <div class="min-w-0">
                                        <div class="text-xs font-medium text-slate-200">Anti-DDoS</div>
                                        <div class="mt-1 text-[11px] text-slate-300/70">Добавляется к стоимости при включении Anti-DDoS.</div>
                                    </div>

                                    <label class="relative inline-flex shrink-0 cursor-pointer items-center">
                                        <input
                                            type="checkbox"
                                            name="allow_antiddos"
                                            value="1"
                                            class="peer sr-only"
                                            {{ old('allow_antiddos') ? 'checked' : '' }}
                                        >
                                        <span class="h-5 w-9 rounded-full border border-white/10 bg-black/10 transition peer-checked:border-sky-500 peer-checked:bg-sky-600"></span>
                                        <span class="pointer-events-none absolute left-0.5 top-0.5 h-4 w-4 rounded-full bg-white/80 shadow transition peer-checked:translate-x-4"></span>
                                    </label>
                                </div>

                                <div class="space-y-1">
                                    <label for="antiddos_price" class="text-xs font-medium text-slate-200">Цена (₽/мес)</label>
                                    <input
                                        id="antiddos_price"
                                        name="antiddos_price"
                                        type="number"
                                        step="0.01"
                                        min="0"
                                        value="{{ old('antiddos_price', 0) }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>
                            </div>

                            <div class="space-y-1">
                                <label for="discounts" class="text-xs font-medium text-slate-200">Скидки (JSON)</label>
                                <textarea
                                    id="discounts"
                                    name="discounts"
                                    rows="3"
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    placeholder='{"15": 10, "30": 20, ...}'
                                >{{ old('discounts') }}</textarea>
                                <p class="mt-1 text-[11px] text-slate-300/70">Проценты скидок по периодам, если используется в логике биллинга.</p>
                            </div>
                        </div>

                        <div class="space-y-4" x-show="billingType === 'resources'" x-cloak>
                            <div class="grid gap-4 md:grid-cols-3">
                                <div class="space-y-1">
                                    <label for="cpu_min" class="text-xs font-medium text-slate-200">CPU min</label>
                                    <input
                                        id="cpu_min"
                                        name="cpu_min"
                                        type="number"
                                        min="0"
                                        value="{{ old('cpu_min') }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>
                                <div class="space-y-1">
                                    <label for="cpu_max" class="text-xs font-medium text-slate-200">CPU max</label>
                                    <input
                                        id="cpu_max"
                                        name="cpu_max"
                                        type="number"
                                        min="0"
                                        value="{{ old('cpu_max') }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>
                                <div class="space-y-1">
                                    <label for="cpu_step" class="text-xs font-medium text-slate-200">CPU step</label>
                                    <input
                                        id="cpu_step"
                                        name="cpu_step"
                                        type="number"
                                        min="1"
                                        value="{{ old('cpu_step') }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>
                            </div>

                            <div class="grid gap-4 md:grid-cols-3">
                                <div class="space-y-1">
                                    <label for="ram_min" class="text-xs font-medium text-slate-200">RAM min</label>
                                    <input
                                        id="ram_min"
                                        name="ram_min"
                                        type="number"
                                        min="0"
                                        value="{{ old('ram_min') }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>
                                <div class="space-y-1">
                                    <label for="ram_max" class="text-xs font-medium text-slate-200">RAM max</label>
                                    <input
                                        id="ram_max"
                                        name="ram_max"
                                        type="number"
                                        min="0"
                                        value="{{ old('ram_max') }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>
                                <div class="space-y-1">
                                    <label for="ram_step" class="text-xs font-medium text-slate-200">RAM step</label>
                                    <input
                                        id="ram_step"
                                        name="ram_step"
                                        type="number"
                                        min="1"
                                        value="{{ old('ram_step') }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>
                            </div>

                            <div class="grid gap-4 md:grid-cols-3">
                                <div class="space-y-1">
                                    <label for="disk_min" class="text-xs font-medium text-slate-200">Диск min</label>
                                    <input
                                        id="disk_min"
                                        name="disk_min"
                                        type="number"
                                        min="0"
                                        value="{{ old('disk_min') }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>
                                <div class="space-y-1">
                                    <label for="disk_max" class="text-xs font-medium text-slate-200">Диск max</label>
                                    <input
                                        id="disk_max"
                                        name="disk_max"
                                        type="number"
                                        min="0"
                                        value="{{ old('disk_max') }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>
                                <div class="space-y-1">
                                    <label for="disk_step" class="text-xs font-medium text-slate-200">Диск step</label>
                                    <input
                                        id="disk_step"
                                        name="disk_step"
                                        type="number"
                                        min="1"
                                        value="{{ old('disk_step') }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>
                            </div>
                        </div>

                        <div class="grid gap-4 md:grid-cols-2">
                            <div class="space-y-1">
                                <label for="cpu_cores" class="text-xs font-medium text-slate-200">CPU cores</label>
                                <input
                                    id="cpu_cores"
                                    name="cpu_cores"
                                    type="number"
                                    min="0"
                                    value="{{ old('cpu_cores', 1) }}"
                                    x-model.number="cpuCores"
                                    required
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                                <p class="mt-1 text-[11px] text-slate-300/70">0 — без лимита (тогда нужен CPU shares).</p>
                            </div>
                            <div class="space-y-1">
                                <label for="cpu_shares" class="text-xs font-medium text-slate-200">CPU shares</label>
                                <input
                                    id="cpu_shares"
                                    name="cpu_shares"
                                    type="number"
                                    min="2"
                                    max="262144"
                                    value="{{ old('cpu_shares') }}"
                                    x-bind:disabled="cpuCores !== 0"
                                    x-bind:required="cpuCores === 0"
                                    x-bind:class="cpuCores !== 0 ? 'opacity-50 cursor-not-allowed' : ''"
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                                <p class="mt-1 text-[11px] text-slate-300/70">Вес CPU для shared-режима (если cores = 0).</p>
                            </div>
                        </div>

                        <div class="grid gap-4 md:grid-cols-2">
                            <div class="space-y-1">
                                <label for="ram_gb" class="text-xs font-medium text-slate-200">RAM (ГБ)</label>
                                <input
                                    id="ram_gb"
                                    name="ram_gb"
                                    type="number"
                                    min="1"
                                    value="{{ old('ram_gb', 1) }}"
                                    required
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                                <p class="mt-1 text-[11px] text-slate-300/70">Лимит оперативной памяти контейнера.</p>
                            </div>
                            <div class="space-y-1">
                                <label for="disk_gb" class="text-xs font-medium text-slate-200">Диск (ГБ)</label>
                                <input
                                    id="disk_gb"
                                    name="disk_gb"
                                    type="number"
                                    min="1"
                                    value="{{ old('disk_gb', 10) }}"
                                    required
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                                <p class="mt-1 text-[11px] text-slate-300/70">Размер диска сервера (влияет на квоту).</p>
                            </div>
                        </div>

                    </div>
                </div>

                <div class="flex items-center justify-end gap-3 pt-4">
                    <a href="{{ route('admin.tariffs.index') }}" class="text-xs text-slate-300/80 hover:text-white">Отмена</a>
                    <button
                        type="submit"
                        class="inline-flex items-center rounded-md bg-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-sky-500"
                    >
                        Создать тариф
                    </button>
                </div>
            </form>
        </div>
    </section>
@endsection
