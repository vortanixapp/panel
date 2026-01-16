@extends('layouts.app-admin')

@section('page_title', 'Игры')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            @if (session('error'))
                <div class="mb-3 rounded-md border border-rose-500/20 bg-rose-500/10 px-3 py-2 text-xs text-rose-200">
                    {{ session('error') }}
                </div>
            @endif

            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Игры</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Список игр, доступных в GameCloud.</p>
                </div>
                <div class="flex items-center gap-3">
                    <a
                        href="{{ route('admin.games.create') }}"
                        class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-slate-800"
                    >
                        Добавить игру
                    </a>
                </div>
            </div>

            <div class="overflow-hidden rounded-2xl border border-white/10 bg-[#242f3d] shadow-sm text-sm">
                <table class="hidden md:table min-w-full divide-y divide-white/10">
                    <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                        <tr>
                            <th scope="col" class="px-4 py-2 text-left">Название</th>
                            <th scope="col" class="px-4 py-2 text-left">Slug</th>
                            <th scope="col" class="px-4 py-2 text-left">Статус</th>
                            <th scope="col" class="px-4 py-2 text-left">Дата создания</th>
                            <th scope="col" class="px-4 py-2 text-left">Действия</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-white/10 text-[13px]">
                        @forelse ($games as $game)
                            <tr>
                                <td class="px-4 py-2 align-top">
                                    <div class="font-medium text-slate-100">{{ $game->name }}</div>
                                </td>
                                <td class="px-4 py-2 align-top">
                                    <div class="text-slate-200">{{ $game->slug }}</div>
                                </td>
                                <td class="px-4 py-2 align-top">
                                    @if($game->is_active)
                                        <span class="inline-flex items-center rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-medium text-emerald-300 ring-1 ring-emerald-500/20">
                                            Активна
                                        </span>
                                    @else
                                        <span class="inline-flex items-center rounded-full bg-black/10 px-2 py-0.5 text-[11px] font-medium text-slate-300 ring-1 ring-white/10">
                                            Неактивна
                                        </span>
                                    @endif
                                </td>
                                <td class="px-4 py-2 align-top text-slate-300/70">
                                    {{ $game->created_at?->format('d.m.Y H:i') ?? '—' }}
                                </td>
                                <td class="px-4 py-2 align-top text-slate-300/70">
                                    <div class="flex items-center gap-2">
                                        <a
                                            href="{{ route('admin.games.show', $game) }}"
                                            class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                            title="Просмотреть игру"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                                <path d="M1 12s4-8 9-4 9 4 9 4-4 8-9 4-9-4-9-4Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                <circle cx="10" cy="8" r="2" stroke-width="1.4" />
                                            </svg>
                                            <span class="sr-only">Просмотреть</span>
                                        </a>

                                        <a
                                            href="{{ route('admin.games.edit', $game) }}"
                                            class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                            title="Редактировать игру"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                                <path d="M5 13.5 4 16l2.5-1 7.5-7.5-1.5-1.5L5 13.5Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                <path d="M11.5 4 13 2.5 15.5 5 14 6.5" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                            </svg>
                                            <span class="sr-only">Редактировать</span>
                                        </a>

                                        <form
                                            method="POST"
                                            action="{{ route('admin.games.destroy', $game) }}"
                                            onsubmit="return confirm('Удалить игру {{ $game->name }}?');"
                                        >
                                            @csrf
                                            @method('DELETE')
                                            <button
                                                type="submit"
                                                class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-rose-500/20 bg-rose-500/10 text-rose-200 hover:bg-rose-500/15"
                                                title="Удалить игру"
                                            >
                                                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                                    <path d="M5 6h10" stroke-width="1.4" stroke-linecap="round" />
                                                    <path d="M8 6V4.5A1.5 1.5 0 0 1 9.5 3h1A1.5 1.5 0 0 1 12 4.5V6" stroke-width="1.4" stroke-linecap="round" />
                                                    <path d="M7 6h6l-.5 9a1.5 1.5 0 0 1-1.5 1.4h-2a1.5 1.5 0 0 1-1.5-1.4L7 6Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                </svg>
                                                <span class="sr-only">Удалить</span>
                                            </button>
                                        </form>
                                    </div>
                                </td>
                            </tr>
                        @empty
                            <tr>
                                <td colspan="5" class="px-4 py-8 text-center text-[13px] text-slate-300/70">
                                    Игры пока не найдены.
                                </td>
                            </tr>
                        @endforelse
                    </tbody>
                </table>

                @if ($games instanceof \Illuminate\Contracts\Pagination\Paginator || $games instanceof \Illuminate\Contracts\Pagination\LengthAwarePaginator)
                    <div class="border-t border-white/10 bg-black/10 px-4 py-3 text-xs text-slate-300/70">
                        {{ $games->links() }}
                    </div>
                @endif

                <!-- Mobile cards -->
                <div class="md:hidden divide-y divide-white/10">
                    @forelse ($games as $game)
                        <div class="px-4 py-4 space-y-3">
                            <div class="flex items-center justify-between">
                                <div class="font-medium text-slate-100">{{ $game->name }}</div>
                                @if($game->is_active)
                                    <span class="inline-flex items-center rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-medium text-emerald-300 ring-1 ring-emerald-500/20">
                                        Активна
                                    </span>
                                @else
                                    <span class="inline-flex items-center rounded-full bg-black/10 px-2 py-0.5 text-[11px] font-medium text-slate-300 ring-1 ring-white/10">
                                        Неактивна
                                    </span>
                                @endif
                            </div>
                            <div class="text-sm text-slate-200">{{ $game->slug }}</div>
                            <div class="text-xs text-slate-300/70">Создано: {{ $game->created_at?->format('d.m.Y H:i') ?? '—' }}</div>
                            <div class="flex items-center gap-2 pt-2 border-t border-white/10">
                                <a
                                    href="{{ route('admin.games.show', $game) }}"
                                    class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                    title="Просмотреть игру"
                                >
                                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                        <path d="M1 12s4-8 9-4 9 4 9 4-4 8-9 4-9-4-9-4Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                        <circle cx="10" cy="8" r="2" stroke-width="1.4" />
                                    </svg>
                                </a>
                                <a
                                    href="{{ route('admin.games.edit', $game) }}"
                                    class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                    title="Редактировать игру"
                                >
                                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                        <path d="M5 13.5 4 16l2.5-1 7.5-7.5-1.5-1.5L5 13.5Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                        <path d="M11.5 4 13 2.5 15.5 5 14 6.5" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                    </svg>
                                </a>
                                <form
                                    method="POST"
                                    action="{{ route('admin.games.destroy', $game) }}"
                                    onsubmit="return confirm('Удалить игру {{ $game->name }}?');"
                                >
                                    @csrf
                                    @method('DELETE')
                                    <button
                                        type="submit"
                                        class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-rose-500/20 bg-rose-500/10 text-rose-200 hover:bg-rose-500/15"
                                        title="Удалить игру"
                                    >
                                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                            <path d="M5 6h10" stroke-width="1.4" stroke-linecap="round" />
                                            <path d="M8 6V4.5A1.5 1.5 0 0 1 9.5 3h1A1.5 1.5 0 0 1 12 4.5V6" stroke-width="1.4" stroke-linecap="round" />
                                            <path d="M7 6h6l-.5 9a1.5 1.5 0 0 1-1.5 1.4h-2a1.5 1.5 0 0 1-1.5-1.4L7 6Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                        </svg>
                                    </button>
                                </form>
                            </div>
                        </div>
                    @empty
                        <div class="px-4 py-8 text-center text-[13px] text-slate-300/70">
                            Игры пока не найдены.
                        </div>
                    @endforelse
                </div>
        </div>
    </section>
@endsection
