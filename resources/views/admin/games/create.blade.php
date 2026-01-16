@extends('layouts.app-admin')

@section('page_title', 'Новая игра')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Новая игра</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Добавьте новую игру в систему GameCloud.</p>
                </div>
                <a
                    href="{{ route('admin.games') }}"
                    class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                >
                    ← Назад к списку
                </a>
            </div>

            <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-6 shadow-sm shadow-black/20">
                @if ($errors->any())
                    <div class="mb-4 rounded-md border border-rose-500/20 bg-rose-500/10 px-3 py-2 text-xs text-rose-200">
                        <ul class="list-disc space-y-1 pl-4">
                            @foreach ($errors->all() as $error)
                                <li>{{ $error }}</li>
                            @endforeach
                        </ul>
                    </div>
                @endif

                <form method="POST" action="{{ route('admin.games.store') }}" class="space-y-4">
                    @csrf

                    <div class="space-y-1">
                        <label for="name" class="text-xs font-medium text-slate-200">Название игры</label>
                        <input
                            id="name"
                            name="name"
                            type="text"
                            value="{{ old('name') }}"
                            required
                            class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                    </div>

                    <div class="space-y-1">
                        <label for="slug" class="text-xs font-medium text-slate-200">Slug (идентификатор)</label>
                        <input
                            id="slug"
                            name="slug"
                            type="text"
                            value="{{ old('slug') }}"
                            required
                            class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                    </div>

                    <div class="space-y-1">
                        <label for="description" class="text-xs font-medium text-slate-200">Описание</label>
                        <textarea
                            id="description"
                            name="description"
                            rows="4"
                            class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >{{ old('description') }}</textarea>
                    </div>

                    <div class="space-y-1">
                        <label for="code" class="text-xs font-medium text-slate-200">Код игры</label>
                        <input
                            id="code"
                            name="code"
                            type="text"
                            value="{{ old('code') }}"
                            required
                            class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                    </div>

                    <div class="space-y-1">
                        <label for="query" class="text-xs font-medium text-slate-200">Query драйвер</label>
                        <input
                            id="query"
                            name="query"
                            type="text"
                            value="{{ old('query') }}"
                            required
                            class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                    </div>

                    <div class="grid gap-4 md:grid-cols-2">
                        <div class="space-y-1">
                            <label for="minport" class="text-xs font-medium text-slate-200">Мин. порт</label>
                            <input
                                id="minport"
                                name="minport"
                                type="number"
                                min="1"
                                max="65535"
                                value="{{ old('minport', 1024) }}"
                                required
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>

                        <div class="space-y-1">
                            <label for="maxport" class="text-xs font-medium text-slate-200">Макс. порт</label>
                            <input
                                id="maxport"
                                name="maxport"
                                type="number"
                                min="1"
                                max="65535"
                                value="{{ old('maxport', 65535) }}"
                                required
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                    </div>

                    <div class="flex items-center justify-between gap-3 pt-2">
                        <label class="inline-flex items-center gap-2 text-[11px] text-slate-300/80">
                            <input
                                type="checkbox"
                                name="status"
                                value="1"
                                class="h-3 w-3 rounded border-white/10 bg-black/10 text-slate-100 focus:ring-slate-900"
                                {{ old('status', 1) ? 'checked' : '' }}
                            >
                            <span>Игра включена</span>
                        </label>
                    </div>

                    <div class="flex items-center justify-end gap-3 pt-4">
                        <a href="{{ route('admin.games') }}" class="text-xs text-slate-300/70 hover:text-slate-200">Отмена</a>
                        <button
                            type="submit"
                            class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-slate-800"
                        >
                            Создать игру
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </section>
@endsection
