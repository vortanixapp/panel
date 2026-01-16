@extends('layouts.app-admin')

@section('title', 'Тариф: ' . $tariff->name)

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Тариф: {{ $tariff->name }}</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Просмотр информации о тарифном плане и ресурсных лимитах.</p>
                </div>

                <div class="flex items-center gap-3">
                    <a
                        href="{{ route('admin.tariffs.edit', $tariff) }}"
                        class="inline-flex items-center rounded-md bg-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-sky-500"
                    >
                        Редактировать
                    </a>

                    <form method="POST" action="{{ route('admin.tariffs.duplicate', $tariff) }}" class="inline">
                        @csrf
                        <button
                            type="submit"
                            class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-4 py-2 text-xs font-semibold text-slate-200 shadow-sm hover:bg-black/15 hover:text-white"
                        >
                            Копировать
                        </button>
                    </form>

                    <form method="POST" action="{{ route('admin.tariffs.destroy', $tariff) }}" class="inline">
                        @csrf
                        @method('DELETE')
                        <button
                            type="submit"
                            class="inline-flex items-center rounded-md border border-red-500/30 bg-red-500/10 px-4 py-2 text-xs font-semibold text-red-200 shadow-sm hover:bg-red-500/15 hover:text-white"
                            onclick="return confirm('Вы уверены, что хотите удалить этот тариф?')"
                        >
                            Удалить тариф
                        </button>
                    </form>
                    <a
                        href="{{ route('admin.tariffs.index') }}"
                        class="text-xs text-slate-300/80 hover:text-white"
                    >
                        ← Назад к тарифам
                    </a>
                </div>
            </div>

            <div class="rounded-2xl border border-white/10 bg-[#242f3d] shadow-sm shadow-black/20">
                <div class="p-6">
                    <div class="grid gap-6 md:grid-cols-3">
                        <div class="space-y-4">
                            <div>
                                <div class="text-xs font-semibold text-slate-100">База</div>
                                <div class="mt-1 text-[11px] text-slate-300/70">Основная информация и статус.</div>
                            </div>

                            <div class="space-y-3 text-xs">
                                <div class="flex items-center justify-between gap-3">
                                    <div class="text-slate-300/70">Имя тарифа</div>
                                    <div class="text-right font-medium text-slate-100">{{ $tariff->name }}</div>
                                </div>
                                <div class="flex items-center justify-between gap-3">
                                    <div class="text-slate-300/70">Локация</div>
                                    <div class="text-right text-slate-100">{{ $tariff->location->name ?? '—' }}</div>
                                </div>
                                <div class="flex items-center justify-between gap-3">
                                    <div class="text-slate-300/70">Игра</div>
                                    <div class="text-right text-slate-100">{{ $tariff->game->name ?? '—' }}</div>
                                </div>
                                <div class="flex items-center justify-between gap-3">
                                    <div class="text-slate-300/70">Тип тарифа</div>
                                    <div class="text-right">
                                        <span class="inline-flex items-center rounded-full bg-white/5 px-2 py-0.5 text-[11px] font-medium text-slate-200 ring-1 ring-white/10">
                                            {{ ($tariff->billing_type ?? 'resources') === 'slots' ? 'Оплата за слоты' : 'Оплата за ресурсы' }}
                                        </span>
                                    </div>
                                </div>
                                <div class="flex items-center justify-between gap-3">
                                    <div class="text-slate-300/70">Доступность</div>
                                    <div class="text-right">
                                        @if($tariff->is_available)
                                            <span class="inline-flex items-center rounded-full bg-emerald-500/15 px-2 py-0.5 text-[11px] font-medium text-emerald-200 ring-1 ring-emerald-500/20">Доступен</span>
                                        @else
                                            <span class="inline-flex items-center rounded-full bg-white/5 px-2 py-0.5 text-[11px] font-medium text-slate-200 ring-1 ring-white/10">Недоступен</span>
                                        @endif
                                    </div>
                                </div>
                                <div class="flex items-center justify-between gap-3">
                                    <div class="text-slate-300/70">Положение</div>
                                    <div class="text-right text-slate-100">{{ $tariff->position }}</div>
                                </div>

                                <div class="h-px bg-white/10"></div>

                                <div class="space-y-2">
                                    <div class="text-xs font-medium text-slate-200">Периоды</div>
                                    <div class="flex flex-wrap gap-2">
                                        <div class="text-[11px] text-slate-300/70">Аренда:</div>
                                        <div class="text-xs text-slate-100">{{ $tariff->rental_periods ? implode(', ', $tariff->rental_periods) . ' дней' : '—' }}</div>
                                    </div>
                                    <div class="flex flex-wrap gap-2">
                                        <div class="text-[11px] text-slate-300/70">Продление:</div>
                                        <div class="text-xs text-slate-100">{{ $tariff->renewal_periods ? implode(', ', $tariff->renewal_periods) . ' дней' : '—' }}</div>
                                    </div>
                                </div>

                                <div class="h-px bg-white/10"></div>
                                <div class="flex items-center justify-between gap-3">
                                    <div class="text-slate-300/70">Дата создания</div>
                                    <div class="text-right text-slate-100">{{ $tariff->created_at->format('d.m.Y H:i:s') }}</div>
                                </div>
                                <div class="flex items-center justify-between gap-3">
                                    <div class="text-slate-300/70">Обновление</div>
                                    <div class="text-right text-slate-100">{{ $tariff->updated_at->format('d.m.Y H:i:s') }}</div>
                                </div>
                            </div>
                        </div>

                        <div class="space-y-4">
                            <div>
                                <div class="text-xs font-semibold text-slate-100">Ограничения и периоды</div>
                                <div class="mt-1 text-[11px] text-slate-300/70">Лимиты, диапазоны и доп. условия.</div>
                            </div>

                            <div class="space-y-4">
                                <div class="grid gap-3 md:grid-cols-2">
                                    <div class="rounded-lg border border-white/10 bg-black/10 p-3">
                                        <div class="text-[11px] text-slate-300/70">CPU</div>
                                        <div class="mt-1 text-xs font-medium text-slate-100">{{ (int) ($tariff->cpu_cores ?? 0) === 0 ? 'без лимита' : ($tariff->cpu_cores . ' ядер') }}</div>
                                        <div class="mt-1 text-[11px] text-slate-300/70">shares: {{ $tariff->cpu_shares ?? '—' }}</div>
                                    </div>
                                    <div class="rounded-lg border border-white/10 bg-black/10 p-3">
                                        <div class="text-[11px] text-slate-300/70">RAM / Диск</div>
                                        <div class="mt-1 text-xs font-medium text-slate-100">{{ $tariff->ram_gb }} ГБ / {{ $tariff->disk_gb }} ГБ</div>
                                    </div>
                                </div>

                            @if (($tariff->billing_type ?? 'resources') === 'slots')
                                <div class="rounded-lg border border-white/10 bg-black/10 p-3">
                                    <div class="text-[11px] text-slate-300/70">Слоты</div>
                                    <div class="mt-1 text-xs font-medium text-slate-100">{{ $tariff->min_slots }} — {{ $tariff->max_slots }}</div>
                                </div>
                            @else
                                <div class="space-y-2">
                                    <div class="text-xs font-medium text-slate-200">Диапазоны</div>
                                    <div class="grid gap-3 md:grid-cols-3">
                                        <div class="rounded-lg border border-white/10 bg-black/10 p-3">
                                            <div class="text-[11px] text-slate-300/70">CPU</div>
                                            <div class="mt-1 text-xs text-slate-100">{{ $tariff->cpu_min !== null ? $tariff->cpu_min : '—' }} — {{ $tariff->cpu_max !== null ? $tariff->cpu_max : '—' }}</div>
                                            <div class="mt-1 text-[11px] text-slate-300/70">step {{ $tariff->cpu_step !== null ? $tariff->cpu_step : '—' }}</div>
                                        </div>
                                        <div class="rounded-lg border border-white/10 bg-black/10 p-3">
                                            <div class="text-[11px] text-slate-300/70">RAM</div>
                                            <div class="mt-1 text-xs text-slate-100">{{ $tariff->ram_min !== null ? $tariff->ram_min : '—' }} — {{ $tariff->ram_max !== null ? $tariff->ram_max : '—' }}</div>
                                            <div class="mt-1 text-[11px] text-slate-300/70">step {{ $tariff->ram_step !== null ? $tariff->ram_step : '—' }}</div>
                                        </div>
                                        <div class="rounded-lg border border-white/10 bg-black/10 p-3">
                                            <div class="text-[11px] text-slate-300/70">Диск</div>
                                            <div class="mt-1 text-xs text-slate-100">{{ $tariff->disk_min !== null ? $tariff->disk_min : '—' }} — {{ $tariff->disk_max !== null ? $tariff->disk_max : '—' }}</div>
                                            <div class="mt-1 text-[11px] text-slate-300/70">step {{ $tariff->disk_step !== null ? $tariff->disk_step : '—' }}</div>
                                        </div>
                                    </div>
                                </div>
                            @endif

                            <div class="grid gap-3 md:grid-cols-2">
                                <div class="rounded-lg border border-white/10 bg-black/10 p-3">
                                    <div class="flex items-center justify-between gap-3">
                                        <div>
                                            <div class="text-xs font-medium text-slate-200">Anti-DDoS</div>
                                            <div class="mt-1 text-[11px] text-slate-300/70">{{ ($tariff->allow_antiddos ?? false) ? ('Да, ' . number_format((float) ($tariff->antiddos_price ?? 0), 2) . ' ₽/мес') : 'Нет' }}</div>
                                        </div>
                                        @if(($tariff->allow_antiddos ?? false))
                                            <span class="inline-flex h-2 w-2 rounded-full bg-emerald-400"></span>
                                        @else
                                            <span class="inline-flex h-2 w-2 rounded-full bg-white/25"></span>
                                        @endif
                                    </div>
                                </div>
                                <div class="rounded-lg border border-white/10 bg-black/10 p-3">
                                    <div class="text-xs font-medium text-slate-200">Скидки</div>
                                    <div class="mt-1 text-[11px] text-slate-300/70 break-all">{{ $tariff->discounts ? json_encode($tariff->discounts) : '—' }}</div>
                                </div>
                            </div>
                        </div>
                        </div>

                        <div class="space-y-4">
                            <div>
                                <div class="text-xs font-semibold text-slate-100">Биллинг</div>
                                <div class="mt-1 text-[11px] text-slate-300/70">Цены и формула расчёта.</div>
                            </div>

                            <div class="space-y-3 text-xs">
                                @if (($tariff->billing_type ?? 'resources') === 'slots')
                                <div class="flex items-center justify-between gap-3">
                                    <div class="text-slate-300/70">Цена за слот</div>
                                    <div class="text-right font-medium text-slate-100">{{ $tariff->price_per_slot !== null ? (number_format((float) $tariff->price_per_slot, 2) . ' ₽/мес') : '—' }}</div>
                                </div>
                            @else
                                <div class="flex items-center justify-between gap-3">
                                    <div class="text-slate-300/70">Базовая цена</div>
                                    <div class="text-right font-medium text-slate-100">{{ number_format((float) ($tariff->base_price_monthly ?? 0), 2) }} ₽/мес</div>
                                </div>
                                <div class="space-y-3">
                                    <div class="grid gap-3 md:grid-cols-3">
                                        <div class="rounded-lg border border-white/10 bg-black/10 p-3">
                                            <div class="text-[11px] text-slate-300/70">CPU</div>
                                            <div class="mt-1 text-xs font-medium text-slate-100">{{ number_format((float) ($tariff->price_per_cpu_core ?? 0), 2) }}</div>
                                            <div class="mt-1 text-[11px] text-slate-300/70">₽/ядро/мес</div>
                                        </div>
                                        <div class="rounded-lg border border-white/10 bg-black/10 p-3">
                                            <div class="text-[11px] text-slate-300/70">RAM</div>
                                            <div class="mt-1 text-xs font-medium text-slate-100">{{ number_format((float) ($tariff->price_per_ram_gb ?? 0), 2) }}</div>
                                            <div class="mt-1 text-[11px] text-slate-300/70">₽/ГБ/мес</div>
                                        </div>
                                        <div class="rounded-lg border border-white/10 bg-black/10 p-3">
                                            <div class="text-[11px] text-slate-300/70">Диск</div>
                                            <div class="mt-1 text-xs font-medium text-slate-100">{{ number_format((float) ($tariff->price_per_disk_gb ?? 0), 2) }}</div>
                                            <div class="mt-1 text-[11px] text-slate-300/70">₽/ГБ/мес</div>
                                        </div>
                                    </div>
                                </div>
                            @endif
                        </div>
                    </div>
                </div>

            </div>
        </div>
    </section>
@endsection
