@extends('layouts.app-admin')

@section('title', 'Тарифы')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Тарифы</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Управление тарифами для серверов</p>
                </div>

                <a
                    href="{{ route('admin.tariffs.create') }}"
                    class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                >
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="mr-2 h-4 w-4">
                        <path d="M12 4v16m8-8H4" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                    </svg>
                    Создать тариф
                </a>
            </div>

            <div class="overflow-hidden rounded-2xl border border-white/10 bg-[#242f3d] shadow-sm shadow-black/20 text-sm">
                <table class="hidden md:table min-w-full divide-y divide-white/10">
                    <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/80">
                        <tr>
                            <th scope="col" class="px-4 py-2 text-left">Название</th>
                            <th scope="col" class="px-4 py-2 text-left">Локация</th>
                            <th scope="col" class="px-4 py-2 text-left">Игра</th>
                            <th scope="col" class="px-4 py-2 text-left">Тип</th>
                            <th scope="col" class="px-4 py-2 text-left">Параметры</th>
                            <th scope="col" class="px-4 py-2 text-left">Положение</th>
                            <th scope="col" class="px-4 py-2 text-left">Доступность</th>
                            <th scope="col" class="px-4 py-2 text-left">Действия</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-white/10 text-[13px]">
                        @forelse ($tariffs as $tariff)
                            <tr>
                                <td class="px-4 py-2 align-top">
                                    <div class="font-medium text-slate-100">{{ $tariff->name }}</div>
                                </td>
                                <td class="px-4 py-2 align-top">
                                    <div class="text-slate-200">{{ $tariff->location->name ?? '—' }}</div>
                                </td>
                                <td class="px-4 py-2 align-top">
                                    <div class="text-slate-200">{{ $tariff->game->name ?? '—' }}</div>
                                </td>
                                <td class="px-4 py-2 align-top">
                                    <div class="text-slate-200">{{ ($tariff->billing_type ?? 'resources') === 'slots' ? 'слоты' : 'ресурсы' }}</div>
                                </td>
                                <td class="px-4 py-2 align-top">
                                    @if (($tariff->billing_type ?? 'resources') === 'slots')
                                        <div class="text-slate-200">{{ $tariff->min_slots }} — {{ $tariff->max_slots }} слотов</div>
                                    @else
                                        <div class="text-slate-200">{{ $tariff->cpu_cores }} ядер / {{ $tariff->ram_gb }} ГБ / {{ $tariff->disk_gb }} ГБ</div>
                                    @endif
                                </td>
                                <td class="px-4 py-2 align-top">
                                    <div class="text-slate-200">{{ $tariff->position }}</div>
                                </td>
                                <td class="px-4 py-2 align-top">
                                    @if($tariff->is_available)
                                        <span class="inline-flex items-center rounded-full bg-emerald-500/15 px-2 py-0.5 text-[11px] font-medium text-emerald-200 ring-1 ring-emerald-500/20">
                                            Доступен
                                        </span>
                                    @else
                                        <span class="inline-flex items-center rounded-full bg-white/5 px-2 py-0.5 text-[11px] font-medium text-slate-200 ring-1 ring-white/10">
                                            Недоступен
                                        </span>
                                    @endif
                                </td>
                                <td class="px-4 py-2 align-top">
                                    <div class="flex items-center gap-2">
                                        <a
                                            href="{{ route('admin.tariffs.show', $tariff) }}"
                                            class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                            title="Просмотреть тариф"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                                <path d="M1 12s4-8 9-4 9 4 9 4-4 8-9 4-9-4-9-4Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                <circle cx="10" cy="8" r="2" stroke-width="1.4" />
                                            </svg>
                                        </a>
                                        <a
                                            href="{{ route('admin.tariffs.edit', $tariff) }}"
                                            class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                            title="Редактировать тариф"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                                <path d="m11.25 16.25 2.75-2.75L4.5 4.5l-2.75 2.75L11.25 16.25Z" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                                                <path d="M13 6l3-3" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                                            </svg>
                                        </a>

                                        <form method="POST" action="{{ route('admin.tariffs.duplicate', $tariff) }}" class="inline">
                                            @csrf
                                            <button
                                                type="submit"
                                                class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                                title="Копировать тариф"
                                            >
                                                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                                    <path d="M6 6h9v11H6z" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                                                    <path d="M5 14H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h9a1 1 0 0 1 1 1v1" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                                                </svg>
                                            </button>
                                        </form>

                                        <form method="POST" action="{{ route('admin.tariffs.destroy', $tariff) }}" class="inline">
                                            @csrf
                                            @method('DELETE')
                                            <button
                                                type="submit"
                                                class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-red-500/30 bg-red-500/10 text-red-200 hover:bg-red-500/15 hover:text-white"
                                                title="Удалить тариф"
                                                onclick="return confirm('Вы уверены, что хотите удалить этот тариф?')"
                                            >
                                                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                                    <path d="m14.5 7.5-7 7m0-7 7 7" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                                                </svg>
                                            </button>
                                        </form>
                                    </div>
                                </td>
                            </tr>
                        @empty
                            <tr>
                                <td colspan="8" class="px-4 py-8 text-center text-slate-300/80">
                                    Нет тарифов для отображения
                                </td>
                            </tr>
                        @endforelse
                    </tbody>
                </table>

                <!-- Mobile cards -->
                <div class="md:hidden">
                    @forelse ($tariffs as $tariff)
                        <div class="border-b border-white/10 p-4 last:border-b-0">
                            <div class="flex items-center justify-between">
                                <div>
                                    <div class="font-medium text-slate-100">{{ $tariff->name }}</div>
                                    <div class="text-sm text-slate-300/80">{{ $tariff->location->name ?? '—' }} / {{ $tariff->game->name ?? '—' }}</div>
                                    <div class="text-xs text-slate-300/70">Тип: {{ ($tariff->billing_type ?? 'resources') === 'slots' ? 'слоты' : 'ресурсы' }}</div>
                                    @if (($tariff->billing_type ?? 'resources') === 'slots')
                                        <div class="text-xs text-slate-300/70">{{ $tariff->min_slots }} — {{ $tariff->max_slots }} слотов</div>
                                    @else
                                        <div class="text-xs text-slate-300/70">{{ $tariff->cpu_cores }} ядер / {{ $tariff->ram_gb }} ГБ / {{ $tariff->disk_gb }} ГБ</div>
                                    @endif
                                    <div class="text-xs text-slate-300/70">Положение: {{ $tariff->position }}</div>
                                </div>
                                <div class="flex flex-col items-end gap-2">
                                    @if($tariff->is_available)
                                        <span class="inline-flex items-center rounded-full bg-emerald-500/15 px-2 py-0.5 text-[11px] font-medium text-emerald-200 ring-1 ring-emerald-500/20">
                                            Доступен
                                        </span>
                                    @else
                                        <span class="inline-flex items-center rounded-full bg-white/5 px-2 py-0.5 text-[11px] font-medium text-slate-200 ring-1 ring-white/10">
                                            Недоступен
                                        </span>
                                    @endif
                                    <div class="flex items-center gap-1">
                                        <a
                                            href="{{ route('admin.tariffs.show', $tariff) }}"
                                            class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                            title="Просмотреть тариф"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                                <path d="M1 12s4-8 9-4 9 4 9 4-4 8-9 4-9-4-9-4Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                <circle cx="10" cy="8" r="2" stroke-width="1.4" />
                                            </svg>
                                        </a>
                                        <a
                                            href="{{ route('admin.tariffs.edit', $tariff) }}"
                                            class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                            title="Редактировать тариф"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                                <path d="m11.25 16.25 2.75-2.75L4.5 4.5l-2.75 2.75L11.25 16.25Z" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                                                <path d="M13 6l3-3" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                                            </svg>
                                            <span class="sr-only">Редактировать</span>
                                        </a>

                                        <form method="POST" action="{{ route('admin.tariffs.duplicate', $tariff) }}" class="inline">
                                            @csrf
                                            <button
                                                type="submit"
                                                class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                                title="Копировать тариф"
                                            >
                                                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                                    <path d="M6 6h9v11H6z" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                                                    <path d="M5 14H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h9a1 1 0 0 1 1 1v1" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                                                </svg>
                                                <span class="sr-only">Копировать</span>
                                            </button>
                                        </form>

                                        <form method="POST" action="{{ route('admin.tariffs.destroy', $tariff) }}" class="inline">
                                            @csrf
                                            @method('DELETE')
                                            <button
                                                type="submit"
                                                class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-red-500/30 bg-red-500/10 text-red-200 hover:bg-red-500/15 hover:text-white"
                                                title="Удалить тариф"
                                                onclick="return confirm('Вы уверены, что хотите удалить этот тариф?')"
                                            >
                                                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                                    <path d="m14.5 7.5-7 7m0-7 7 7" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                                                </svg>
                                            </button>
                                        </form>
                                    </div>
                                </div>
                            </div>
                        </div>
                    @empty
                        <div class="p-8 text-center text-slate-300/80">
                            Нет тарифов для отображения
                        </div>
                    @endforelse
                </div>

                @if ($tariffs->hasPages())
                    <div class="border-t border-white/10 bg-black/10 px-4 py-3 text-xs text-slate-300/80">
                        {{ $tariffs->links() }}
                    </div>
                @endif
            </div>
        </div>
    </section>
@endsection
