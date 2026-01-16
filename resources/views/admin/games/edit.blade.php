@extends('layouts.app-admin')

@section('page_title', 'Редактировать игру: ' . $game->name)

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            @php
                $tab = (string) request()->query('tab', 'settings');
            @endphp

            @if (session('error'))
                <div class="mb-3 rounded-md border border-rose-500/20 bg-rose-500/10 px-3 py-2 text-xs text-rose-200">
                    {{ session('error') }}
                </div>
            @endif

            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Редактировать игру: {{ $game->name }}</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Измените параметры игры в системе GameCloud.</p>
                </div>
                <div class="flex items-center gap-2">
                    <a
                        href="{{ route('admin.games.show', $game) }}"
                        class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                    >
                        ← К игре
                    </a>
                    <a
                        href="{{ route('admin.games') }}"
                        class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                    >
                        К списку
                    </a>
                </div>
            </div>

            <div class="flex flex-wrap items-center gap-2 text-xs">
                <a
                    href="{{ route('admin.games.edit', ['game' => $game, 'tab' => 'settings']) }}"
                    class="inline-flex items-center rounded-md border px-3 py-1.5 text-[11px] font-medium transition {{ $tab === 'settings' ? 'border-slate-900 bg-slate-900 text-white' : 'border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white' }}"
                >
                    Настройки
                </a>
                <a
                    href="{{ route('admin.games.edit', ['game' => $game, 'tab' => 'versions']) }}"
                    class="inline-flex items-center rounded-md border px-3 py-1.5 text-[11px] font-medium transition {{ $tab === 'versions' ? 'border-slate-900 bg-slate-900 text-white' : 'border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white' }}"
                >
                    Версии
                </a>
            </div>

            @if($tab === 'versions')
                <div class="grid gap-4 md:grid-cols-3 text-sm">
                    <div class="md:col-span-1 rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                        <h2 class="text-sm font-semibold text-slate-100">Добавить версию</h2>

                        @if ($errors->any())
                            <div class="mt-3 rounded-md border border-rose-500/20 bg-rose-500/10 px-3 py-2 text-xs text-rose-200">
                                <ul class="list-disc space-y-1 pl-4">
                                    @foreach ($errors->all() as $error)
                                        <li>{{ $error }}</li>
                                    @endforeach
                                </ul>
                            </div>
                        @endif

                        <form method="POST" action="{{ route('admin.games.versions.store', $game) }}" class="mt-3 space-y-3">
                            @csrf

                            <div class="space-y-1">
                                <label for="version_name" class="text-xs font-medium text-slate-200">Название версии</label>
                                <input
                                    id="version_name"
                                    name="name"
                                    type="text"
                                    value="{{ old('name') }}"
                                    required
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                            </div>

                            <div class="space-y-1">
                                <label for="version_source_type" class="text-xs font-medium text-slate-200">Источник</label>
                                <select
                                    id="version_source_type"
                                    name="source_type"
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                                    <option value="archive" {{ old('source_type', 'archive') === 'archive' ? 'selected' : '' }}>Archive (URL)</option>
                                    <option value="steam" {{ old('source_type') === 'steam' ? 'selected' : '' }}>Steam (App ID)</option>
                                </select>
                            </div>

                            <div class="space-y-1">
                                <label for="version_url" class="text-xs font-medium text-slate-200">URL</label>
                                <input
                                    id="version_url"
                                    name="url"
                                    type="text"
                                    value="{{ old('url') }}"
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                            </div>

                            <div class="grid grid-cols-2 gap-3">
                                <div class="space-y-1">
                                    <label for="version_steam_app_id" class="text-xs font-medium text-slate-200">Steam App ID</label>
                                    <input
                                        id="version_steam_app_id"
                                        name="steam_app_id"
                                        type="number"
                                        min="1"
                                        value="{{ old('steam_app_id') }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>

                                <div class="space-y-1">
                                    <label for="version_steam_branch" class="text-xs font-medium text-slate-200">Steam branch</label>
                                    <input
                                        id="version_steam_branch"
                                        name="steam_branch"
                                        type="text"
                                        value="{{ old('steam_branch') }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>
                            </div>

                            <div class="grid grid-cols-2 gap-3">
                                <div class="space-y-1">
                                    <label for="version_sort_order" class="text-xs font-medium text-slate-200">Позиция</label>
                                    <input
                                        id="version_sort_order"
                                        name="sort_order"
                                        type="number"
                                        min="0"
                                        value="{{ old('sort_order', 0) }}"
                                        class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                </div>

                                <div class="space-y-2">
                                    <div class="text-xs font-medium text-slate-200">Активна</div>
                                    <label class="inline-flex items-center gap-2 text-xs text-slate-200">
                                        <input type="checkbox" name="is_active" value="1" {{ old('is_active', 1) ? 'checked' : '' }} class="h-3 w-3 rounded border-white/10 bg-black/10 text-slate-100">
                                        <span>Да</span>
                                    </label>
                                </div>
                            </div>

                            <button
                                type="submit"
                                class="inline-flex items-center rounded-md bg-slate-900 px-3 py-2 text-[11px] font-semibold text-white shadow-sm hover:bg-slate-800"
                            >
                                Добавить
                            </button>
                        </form>
                    </div>

                    <div class="md:col-span-2 rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                        <h2 class="text-sm font-semibold text-slate-100">Список версий</h2>
                        <div class="mt-3 space-y-2">
                            @forelse(($game->versions ?? []) as $version)
                                <div class="flex flex-col gap-2 rounded-xl border border-white/10 bg-black/10 p-3 md:flex-row md:items-center md:justify-between">
                                    <div class="min-w-0">
                                        <div class="flex items-center gap-2">
                                            <div class="text-sm font-semibold text-slate-100">{{ $version->name }}</div>
                                            <span class="text-[11px] {{ $version->is_active ? 'text-emerald-300' : 'text-slate-300/70' }}">{{ $version->is_active ? 'Активна' : 'Неактивна' }}</span>
                                            <span class="text-[11px] text-slate-300/70">#{{ (int) ($version->sort_order ?? 0) }}</span>
                                        </div>
                                        @if(($version->source_type ?? 'archive') === 'steam')
                                            <div class="mt-1 text-[11px] text-slate-300/80 break-all">steam: app_id={{ (int) ($version->steam_app_id ?? 0) }}{{ ($version->steam_branch ?? '') !== '' ? ' branch=' . $version->steam_branch : '' }}</div>
                                        @else
                                            <div class="mt-1 text-[11px] text-slate-300/80 break-all">{{ (string) ($version->url ?? '') }}</div>
                                        @endif
                                    </div>

                                    <form method="POST" action="{{ route('admin.games.versions.destroy', ['game' => $game, 'version' => $version]) }}">
                                        @csrf
                                        @method('DELETE')
                                        <button type="submit" class="inline-flex items-center rounded-md border border-rose-500/20 bg-rose-500/10 px-3 py-1.5 text-[11px] font-medium text-rose-200 hover:bg-rose-500/15">
                                            Удалить
                                        </button>
                                    </form>
                                </div>
                            @empty
                                <div class="text-[13px] text-slate-300/70">Версий пока нет.</div>
                            @endforelse
                        </div>
                    </div>
                </div>
            @else
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

                    <form method="POST" action="{{ route('admin.games.update', $game) }}" class="space-y-4">
                        @csrf
                        @method('PUT')

                    <div class="space-y-1">
                        <label for="name" class="text-xs font-medium text-slate-200">Название игры</label>
                        <input
                            id="name"
                            name="name"
                            type="text"
                            value="{{ old('name', $game->name) }}"
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
                            value="{{ old('slug', $game->slug) }}"
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
                        >{{ old('description', $game->description) }}</textarea>
                    </div>

                    <div class="space-y-1">
                        <label for="code" class="text-xs font-medium text-slate-200">Код игры</label>
                        <input
                            id="code"
                            name="code"
                            type="text"
                            value="{{ old('code', $game->code) }}"
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
                            value="{{ old('query', $game->query) }}"
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
                                value="{{ old('minport', $game->minport) }}"
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
                                value="{{ old('maxport', $game->maxport) }}"
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
                                {{ old('status', $game->status) ? 'checked' : '' }}
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
                            Сохранить изменения
                        </button>
                    </div>
                </form>
            </div>
            @endif
        </div>
    </section>
@endsection
