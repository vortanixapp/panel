@extends('layouts.app-admin')

@section('page_title', 'Карты')

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
                            <p class="text-sm font-medium text-emerald-200">{{ session('success') }}</p>
                        </div>
                    </div>
                </div>
            @endif

            @if ($errors->any())
                <div class="rounded-md border border-rose-500/20 bg-rose-500/10 p-4 text-xs text-rose-200">
                    <ul class="list-disc space-y-1 pl-4">
                        @foreach ($errors->all() as $error)
                            <li>{{ $error }}</li>
                        @endforeach
                    </ul>
                </div>
            @endif

            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Карты</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Каталог карт для CS 1.6 (архив zip распаковывается в cstrike/).</p>
                </div>
                <div class="flex items-center gap-3">
                    <a
                        href="{{ route('admin.maps.create') }}"
                        class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-slate-800"
                    >
                        Добавить карту
                    </a>
                </div>
            </div>

            <div class="overflow-hidden rounded-2xl border border-white/10 bg-[#242f3d] shadow-sm text-sm">
                <div class="overflow-x-auto">
                    <table class="min-w-full divide-y divide-white/10">
                        <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                            <tr>
                                <th scope="col" class="px-4 py-2 text-left">Название</th>
                                <th scope="col" class="px-4 py-2 text-left">Категория</th>
                                <th scope="col" class="px-4 py-2 text-left">Slug</th>
                                <th scope="col" class="px-4 py-2 text-left">Версия</th>
                                <th scope="col" class="px-4 py-2 text-left">Архив</th>
                                <th scope="col" class="px-4 py-2 text-left">Статус</th>
                                <th scope="col" class="px-4 py-2 text-left">Действия</th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-white/10 text-[13px]">
                            @forelse ($maps as $map)
                                <tr>
                                    <td class="px-4 py-2 align-top">
                                        <div class="font-medium text-slate-100">{{ $map->name }}</div>
                                    </td>
                                    <td class="px-4 py-2 align-top">
                                        <div class="text-slate-200">{{ $map->category ?: '—' }}</div>
                                    </td>
                                    <td class="px-4 py-2 align-top">
                                        <div class="text-slate-200">{{ $map->slug }}</div>
                                    </td>
                                    <td class="px-4 py-2 align-top">
                                        <div class="text-slate-200">{{ $map->version ?: '—' }}</div>
                                    </td>
                                    <td class="px-4 py-2 align-top">
                                        <div class="text-slate-200">zip</div>
                                    </td>
                                    <td class="px-4 py-2 align-top">
                                        @if($map->active)
                                            <span class="inline-flex items-center rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-medium text-emerald-300 ring-1 ring-emerald-500/20">
                                                Активна
                                            </span>
                                        @else
                                            <span class="inline-flex items-center rounded-full bg-black/10 px-2 py-0.5 text-[11px] font-medium text-slate-300 ring-1 ring-white/10">
                                                Неактивна
                                            </span>
                                        @endif
                                    </td>
                                    <td class="px-4 py-2 align-top">
                                        <div class="flex items-center gap-2">
                                            <a
                                                href="{{ route('admin.maps.edit', $map) }}"
                                                class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                                title="Редактировать"
                                            >
                                                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                                    <path d="M5 13.5 4 16l2.5-1 7.5-7.5-1.5-1.5L5 13.5Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                    <path d="M11.5 4 13 2.5 15.5 5 14 6.5" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                </svg>
                                            </a>
                                            <form
                                                method="POST"
                                                action="{{ route('admin.maps.destroy', $map) }}"
                                                onsubmit="return confirm('Удалить карту {{ $map->name }}?');"
                                            >
                                                @csrf
                                                @method('DELETE')
                                                <button
                                                    type="submit"
                                                    class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-rose-500/20 bg-rose-500/10 text-rose-200 hover:bg-rose-500/15"
                                                    title="Удалить"
                                                >
                                                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                                        <path d="M5 6h10" stroke-width="1.4" stroke-linecap="round" />
                                                        <path d="M8 6V4.5A1.5 1.5 0 0 1 9.5 3h1A1.5 1.5 0 0 1 12 4.5V6" stroke-width="1.4" stroke-linecap="round" />
                                                        <path d="M7 6h6l-.5 9a1.5 1.5 0 0 1-1.5 1.4h-2a1.5 1.5 0 0 1-1.5-1.4L7 6Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                    </svg>
                                                </button>
                                            </form>
                                        </div>
                                    </td>
                                </tr>
                            @empty
                                <tr>
                                    <td colspan="7" class="px-4 py-8 text-center text-[13px] text-slate-300/70">Карты пока не добавлены.</td>
                                </tr>
                            @endforelse
                        </tbody>
                    </table>
                </div>

                @if(method_exists($maps, 'links'))
                    <div class="border-t border-white/10 bg-black/10 px-4 py-3 text-xs text-slate-300/70">
                        {{ $maps->links() }}
                    </div>
                @endif
            </div>
        </div>
    </section>
@endsection
