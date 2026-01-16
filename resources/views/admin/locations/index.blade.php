@extends('layouts.app-admin')

@section('page_title', 'Локации')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            @if (session('success'))
                <div class="rounded-md border border-emerald-500/20 bg-emerald-500/10 p-4">
                    <div class="flex">
                        <div class="flex-shrink-0">
                            <svg class="h-5 w-5 text-emerald-300" viewBox="0 0 20 20" fill="currentColor">
                                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                            </svg>
                        </div>
                        <div class="ml-3">
                            <p class="text-sm font-medium text-emerald-200">
                                {{ session('success') }}
                            </p>
                        </div>
                    </div>
                </div>
            @endif
            <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Локации</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Список доступных дата‑центров и регионов для размещения игровых серверов.</p>
                </div>
                <div class="flex flex-col gap-2 md:flex-row md:items-center">
                    <form method="GET" class="flex flex-wrap items-center gap-2 text-xs text-slate-300/80">
                        <select
                            name="status"
                            class="rounded-md border border-white/10 bg-black/10 px-2.5 py-1.5 text-[11px] text-slate-200 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                            <option value="active" {{ ($status ?? 'active') === 'active' ? 'selected' : '' }}>Только активные</option>
                            <option value="inactive" {{ ($status ?? 'active') === 'inactive' ? 'selected' : '' }}>Только неактивные</option>
                            <option value="all" {{ ($status ?? 'active') === 'all' ? 'selected' : '' }}>Все</option>
                        </select>

                        <select
                            name="region"
                            class="rounded-md border border-white/10 bg-black/10 px-2.5 py-1.5 text-[11px] text-slate-200 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                            <option value="">Все регионы</option>
                            @foreach ($regions ?? [] as $regionOption)
                                <option value="{{ $regionOption }}" {{ ($selectedRegion ?? '') === $regionOption ? 'selected' : '' }}>
                                    {{ $regionOption }}
                                </option>
                            @endforeach
                        </select>

                        <button
                            type="submit"
                            class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                        >
                            Применить
                        </button>
                    </form>

                    <a
                        href="{{ route('admin.locations.create') }}"
                        class="inline-flex items-center justify-center rounded-md bg-slate-900 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-slate-800"
                    >
                        Добавить локацию
                    </a>
                </div>
            </div>

            @if ($locations->isEmpty())
                <div class="rounded-2xl border border-dashed border-white/10 bg-black/10 p-4 text-[13px] text-slate-300/80">
                    Локации пока не заданы.
                </div>
            @else
                <div class="grid gap-4 md:grid-cols-3 text-sm">
                    @foreach ($locations as $location)
                        <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                            <div class="flex items-start justify-between gap-2">
                                <div class="space-y-1">
                                    @if($location->region)
                                        <p class="text-xs font-semibold text-slate-300/70">{{ $location->region }}</p>
                                    @endif
                                    <p class="text-sm font-semibold text-slate-100">{{ $location->name }}</p>
                                    @if($location->city || $location->country)
                                        <p class="text-[11px] text-slate-300/70">
                                            {{ $location->city }}@if($location->city && $location->country), @endif{{ $location->country }}
                                        </p>
                                    @endif
                                    @if($location->description)
                                        <p class="text-xs text-slate-300/80">{{ $location->description }}</p>
                                    @endif
                                </div>
                                <div class="flex items-center gap-1">
                                    <a
                                        href="{{ route('admin.locations.show', $location) }}"
                                        class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                        title="Просмотреть локацию"
                                    >
                                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                            <path d="M1 12s4-8 9-4 9 4 9 4-4 8-9 4-9-4-9-4Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                            <circle cx="10" cy="8" r="2" stroke-width="1.4" />
                                        </svg>
                                        <span class="sr-only">Просмотреть</span>
                                    </a>

                                    <a
                                        href="{{ route('admin.locations.edit', $location) }}"
                                        class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                        title="Редактировать локацию"
                                    >
                                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                            <path d="M5 13.5 4 16l2.5-1 7.5-7.5-1.5-1.5L5 13.5Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                            <path d="M11.5 4 13 2.5 15.5 5 14 6.5" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                        </svg>
                                        <span class="sr-only">Редактировать</span>
                                    </a>

                                    <form method="POST" action="{{ route('admin.locations.toggle', $location) }}">
                                        @csrf
                                        @method('PATCH')
                                        <button
                                            type="submit"
                                            class="inline-flex h-7 w-7 items-center justify-center rounded-md border {{ $location->is_active ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-200 hover:bg-emerald-500/15' : 'border-white/10 bg-black/10 text-slate-300/70 hover:bg-black/15 hover:text-slate-200' }}"
                                            title="{{ $location->is_active ? 'Деактивировать локацию' : 'Активировать локацию' }}"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                                <path d="M10 3v7" stroke-width="1.4" stroke-linecap="round" />
                                                <path d="M6 5.5A5.5 5.5 0 1 0 14 5.5" stroke-width="1.4" stroke-linecap="round" />
                                            </svg>
                                            <span class="sr-only">Переключить активность</span>
                                        </button>
                                    </form>

                                    <form method="POST" action="{{ route('admin.locations.destroy', $location) }}" onsubmit="return confirm('Удалить локацию {{ $location->name }}? Это действие нельзя отменить.')">
                                        @csrf
                                        @method('DELETE')
                                        <button
                                            type="submit"
                                            class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-rose-500/30 bg-rose-500/10 text-rose-200 hover:bg-rose-500/15"
                                            title="Удалить локацию"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                                <path d="M6.5 4.5l9 9M15.5 4.5l-9 9" stroke-width="1.4" stroke-linecap="round" />
                                            </svg>
                                            <span class="sr-only">Удалить</span>
                                        </button>
                                    </form>
                                </div>
                            </div>
                        </div>
                    @endforeach
                </div>
            @endif
        </div>
    </section>
@endsection
