@extends('layouts.app-admin')

@section('page_title', 'Игра: ' . $game->name)

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
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">
                        Игра: {{ $game->name }}
                    </h1>
                    <p class="mt-1 text-sm text-slate-300/80">
                        Детальная информация об игре.
                    </p>
                </div>
                <div class="flex items-center gap-3 text-xs">
                    <a
                        href="{{ route('admin.games') }}"
                        class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                    >
                        Назад к списку
                    </a>
                    <a
                        href="{{ route('admin.games.edit', $game) }}"
                        class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-[11px] font-semibold text-white shadow-sm hover:bg-slate-800"
                    >
                        Редактировать
                    </a>
                </div>
            </div>
            <div class="grid gap-4 md:grid-cols-3 text-sm">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Статус</p>
                    <p class="mt-2 text-sm font-semibold {{ $game->is_active ? 'text-emerald-300' : 'text-slate-300/70' }}">
                        {{ $game->is_active ? 'Активна' : 'Неактивна' }}
                    </p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Определяет доступность игры пользователям.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Slug</p>
                    <p class="mt-2 text-sm font-semibold text-slate-100">{{ $game->slug }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Уникальный идентификатор игры.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Дата создания</p>
                    <p class="mt-2 text-sm font-semibold text-slate-100">{{ $game->created_at->format('d.m.Y H:i') }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Когда игра была добавлена.</p>
                </div>
            </div>

            <div class="grid gap-4 md:grid-cols-3 text-sm">
                <div class="md:col-span-2 rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <h2 class="text-sm font-semibold text-slate-100">Информация об игре</h2>
                    <dl class="mt-3 grid gap-y-1 text-[13px] text-slate-200">
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Название</dt>
                            <dd class="flex-1">{{ $game->name }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Slug</dt>
                            <dd class="flex-1">{{ $game->slug }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Описание</dt>
                            <dd class="flex-1">{{ $game->description ?: '—' }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Код игры</dt>
                            <dd class="flex-1">{{ $game->code ?: '—' }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Query драйвер</dt>
                            <dd class="flex-1">{{ $game->query ?: '—' }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Мин. порт</dt>
                            <dd class="flex-1">{{ $game->minport }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Макс. порт</dt>
                            <dd class="flex-1">{{ $game->maxport }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Статус</dt>
                            <dd class="flex-1">{{ $game->status ? 'Включена' : 'Выключена' }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Дата создания</dt>
                            <dd class="flex-1">{{ $game->created_at->format('d.m.Y H:i:s') }}</dd>
                        </div>
                        <div class="flex gap-2">
                            <dt class="w-32 text-slate-300/70">Последнее обновление</dt>
                            <dd class="flex-1">{{ $game->updated_at->format('d.m.Y H:i:s') }}</dd>
                        </div>
                    </dl>
                </div>
                <div class="rounded-2xl border border-white/10 bg-black/10 p-4 shadow-sm shadow-black/20">
                    <h2 class="text-sm font-semibold text-slate-100 mb-2">Статистика</h2>
                    <div class="text-[11px] text-slate-300/80 space-y-1">
                        <p>Количество серверов: <span class="font-medium">0</span> (заглушка)</p>
                        <p>Общий баланс: <span class="font-medium">0 ₽</span> (заглушка)</p>
                        <p>Последнее обновление: <span class="font-medium">{{ $game->updated_at->format('d.m.Y H:i') }}</span></p>
                    </div>
                    <p class="mt-3 text-[10px] text-slate-300/70">Здесь можно добавить реальные данные: серверы, биллинг и т.д.</p>
                </div>
            </div>
        </div>
    </section>
@endsection
