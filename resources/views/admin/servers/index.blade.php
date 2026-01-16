@extends('layouts.app-admin')

@section('page_title', 'Серверы')

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

            @if (session('error'))
                <div class="rounded-md border border-rose-500/20 bg-rose-500/10 p-4">
                    <div class="flex">
                        <div class="flex-shrink-0">
                            <svg class="h-5 w-5 text-rose-300" viewBox="0 0 20 20" fill="currentColor">
                                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-5a1 1 0 10-2 0 1 1 0 002 0zm-1-8a1 1 0 00-1 1v4a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
                            </svg>
                        </div>
                        <div class="ml-3">
                            <p class="text-sm font-medium text-rose-200">
                                {{ session('error') }}
                            </p>
                        </div>
                    </div>
                </div>
            @endif

            <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Серверы</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Список всех игровых серверов в системе.</p>
                </div>

                <form method="GET" class="flex flex-col gap-2 md:flex-row md:items-center">
                    <input
                        name="q"
                        value="{{ $q ?? '' }}"
                        class="w-full md:w-80 rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-200 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        placeholder="Поиск: #id, имя, IP, пользователь, игра, локация"
                    />
                    <button
                        type="submit"
                        class="inline-flex items-center justify-center rounded-md bg-slate-900 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-slate-800"
                    >
                        Найти
                    </button>
                </form>
            </div>

            <div class="grid gap-4 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Всего серверов</div>
                    <div class="mt-2 text-2xl font-semibold text-slate-100">{{ (int) ($counts['total'] ?? 0) }}</div>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Переустановка</div>
                    <div class="mt-2 text-2xl font-semibold text-slate-100">{{ (int) ($counts['reinstalling'] ?? 0) }}</div>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="text-[11px] uppercase tracking-wide text-slate-300/70">С ошибкой провижининга</div>
                    <div class="mt-2 text-2xl font-semibold text-slate-100">{{ (int) ($counts['with_error'] ?? 0) }}</div>
                </div>
            </div>

            <div class="overflow-hidden rounded-2xl border border-white/10 bg-[#242f3d] shadow-sm shadow-black/20">
                <div class="hidden md:block overflow-x-auto">
                    <table class="min-w-full divide-y divide-white/10 text-sm">
                        <thead class="bg-black/10">
                            <tr class="text-left text-[11px] uppercase tracking-wide text-slate-300/70">
                                <th class="px-4 py-3">ID</th>
                                <th class="px-4 py-3">Сервер</th>
                                <th class="px-4 py-3">Пользователь</th>
                                <th class="px-4 py-3">Игра</th>
                                <th class="px-4 py-3">Локация</th>
                                <th class="px-4 py-3">IP:Port</th>
                                <th class="px-4 py-3">Статус</th>
                                <th class="px-4 py-3 text-right">Действия</th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-white/10">
                            @forelse ($servers as $server)
                                <tr class="text-slate-200 hover:bg-black/10">
                                    <td class="px-4 py-3 font-mono text-xs text-slate-300/80">#{{ $server->id }}</td>
                                    <td class="px-4 py-3">
                                        <div class="font-semibold text-slate-100">{{ $server->name }}</div>
                                        <div class="text-[11px] text-slate-300/70">{{ $server->created_at?->format('d.m.Y H:i') }}</div>
                                    </td>
                                    <td class="px-4 py-3">
                                        <div class="text-slate-100">{{ $server->user?->email ?? '—' }}</div>
                                        <div class="text-[11px] text-slate-300/70">ID: {{ $server->user_id }}</div>
                                    </td>
                                    <td class="px-4 py-3">
                                        <div class="text-slate-100">{{ $server->game?->name ?? '—' }}</div>
                                        <div class="text-[11px] text-slate-300/70">{{ $server->game?->code ?? $server->game?->slug ?? '' }}</div>
                                    </td>
                                    <td class="px-4 py-3">
                                        <div class="text-slate-100">{{ $server->location?->name ?? '—' }}</div>
                                        <div class="text-[11px] text-slate-300/70">{{ $server->location?->ssh_host ?? '' }}</div>
                                    </td>
                                    <td class="px-4 py-3 font-mono text-xs text-slate-200">{{ $server->ip_address }}:{{ $server->port }}</td>
                                    <td class="px-4 py-3">
                                        <div class="inline-flex items-center gap-2">
                                            <span class="inline-flex items-center rounded-full bg-black/10 px-2 py-0.5 text-[11px] font-medium text-slate-200 ring-1 ring-white/10">
                                                {{ $server->status ?? '—' }}
                                            </span>
                                            @if(!empty($server->provisioning_status))
                                                <span class="inline-flex items-center rounded-full border border-sky-500/30 bg-sky-500/10 px-2 py-0.5 text-[11px] font-medium text-sky-200">
                                                    {{ $server->provisioning_status }}
                                                </span>
                                            @endif
                                            @if(!empty($server->provisioning_error))
                                                <span class="inline-flex items-center rounded-full border border-rose-500/30 bg-rose-500/10 px-2 py-0.5 text-[11px] font-medium text-rose-200">
                                                    error
                                                </span>
                                            @endif
                                        </div>
                                    </td>
                                    <td class="px-4 py-3">
                                        <div class="flex justify-end gap-2">
                                            <a
                                                href="{{ route('admin.servers.manage', $server) }}"
                                                class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-2 text-[11px] font-semibold text-slate-200 hover:bg-black/15 hover:text-white"
                                            >
                                                Управление
                                            </a>
                                            <form
                                                method="POST"
                                                action="{{ route('admin.servers.reinstall', $server) }}"
                                                onsubmit="return confirm('Переустановить сервер #{{ $server->id }}? Данные в /data будут пересозданы в зависимости от версии игры.');"
                                            >
                                                @csrf
                                                <button
                                                    type="submit"
                                                    class="inline-flex items-center rounded-md border border-sky-500/30 bg-sky-500/10 px-3 py-2 text-[11px] font-semibold text-sky-200 hover:bg-sky-500/15"
                                                >
                                                    Переустановить
                                                </button>
                                            </form>
                                            <form
                                                method="POST"
                                                action="{{ route('admin.servers.destroy', $server) }}"
                                                onsubmit="return confirm('Полностью удалить сервер #{{ $server->id }}? Будет удалён контейнер на ноде и запись в базе.');"
                                            >
                                                @csrf
                                                @method('DELETE')
                                                <button
                                                    type="submit"
                                                    class="inline-flex items-center rounded-md border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-[11px] font-semibold text-rose-200 hover:bg-rose-500/15"
                                                >
                                                    Полное удаление
                                                </button>
                                            </form>
                                        </div>
                                    </td>
                                </tr>
                            @empty
                                <tr>
                                    <td class="px-4 py-6 text-center text-sm text-slate-300/80" colspan="8">Серверы не найдены.</td>
                                </tr>
                            @endforelse
                        </tbody>
                    </table>
                </div>

                <div class="md:hidden divide-y divide-white/10">
                    @forelse ($servers as $server)
                        <div class="px-4 py-4 space-y-3">
                            <div class="flex items-start justify-between gap-3">
                                <div>
                                    <div class="text-[11px] text-slate-300/70">#{{ $server->id }}</div>
                                    <div class="mt-1 font-semibold text-slate-100">{{ $server->name }}</div>
                                    <div class="mt-1 text-[12px] text-slate-300/80 font-mono">{{ $server->ip_address }}:{{ $server->port }}</div>
                                </div>
                                <div class="text-right">
                                    <div class="inline-flex items-center rounded-full bg-black/10 px-2 py-0.5 text-[11px] font-medium text-slate-200 ring-1 ring-white/10">
                                        {{ $server->status ?? '—' }}
                                    </div>
                                    @if(!empty($server->provisioning_status))
                                        <div class="mt-1 inline-flex items-center rounded-full border border-sky-500/30 bg-sky-500/10 px-2 py-0.5 text-[11px] font-medium text-sky-200">
                                            {{ $server->provisioning_status }}
                                        </div>
                                    @endif
                                </div>
                            </div>

                            <div class="rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-200 space-y-1">
                                <div class="flex items-center justify-between gap-3">
                                    <span class="text-slate-300/70">Пользователь</span>
                                    <span class="font-medium text-slate-100">{{ $server->user?->email ?? '—' }}</span>
                                </div>
                                <div class="flex items-center justify-between gap-3">
                                    <span class="text-slate-300/70">Игра</span>
                                    <span class="font-medium text-slate-100">{{ $server->game?->name ?? '—' }}</span>
                                </div>
                                <div class="flex items-center justify-between gap-3">
                                    <span class="text-slate-300/70">Локация</span>
                                    <span class="font-medium text-slate-100">{{ $server->location?->name ?? '—' }}</span>
                                </div>
                            </div>

                            <div class="flex items-center gap-2 pt-2 border-t border-white/10">
                                <a
                                    href="{{ route('admin.servers.manage', $server) }}"
                                    class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-2 text-[11px] font-semibold text-slate-200 hover:bg-black/15 hover:text-white"
                                >
                                    Управление
                                </a>
                                <form
                                    method="POST"
                                    action="{{ route('admin.servers.reinstall', $server) }}"
                                    onsubmit="return confirm('Переустановить сервер #{{ $server->id }}? Данные в /data будут пересозданы в зависимости от версии игры.');"
                                >
                                    @csrf
                                    <button
                                        type="submit"
                                        class="inline-flex items-center rounded-md border border-sky-500/30 bg-sky-500/10 px-3 py-2 text-[11px] font-semibold text-sky-200 hover:bg-sky-500/15"
                                    >
                                        Переустановить
                                    </button>
                                </form>
                                <form
                                    method="POST"
                                    action="{{ route('admin.servers.destroy', $server) }}"
                                    onsubmit="return confirm('Полностью удалить сервер #{{ $server->id }}? Будет удалён контейнер на ноде и запись в базе.');"
                                >
                                    @csrf
                                    @method('DELETE')
                                    <button
                                        type="submit"
                                        class="inline-flex items-center rounded-md border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-[11px] font-semibold text-rose-200 hover:bg-rose-500/15"
                                    >
                                        Полное удаление
                                    </button>
                                </form>
                            </div>
                        </div>
                    @empty
                        <div class="px-4 py-6 text-center text-sm text-slate-300/80">Серверы не найдены.</div>
                    @endforelse
                </div>

                @if(method_exists($servers, 'links'))
                    <div class="border-t border-white/10 bg-black/10 px-4 py-3">
                        {{ $servers->links() }}
                    </div>
                @endif
            </div>
        </div>
    </section>
@endsection
