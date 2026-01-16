@extends('layouts.app-admin')

@section('page_title', 'Пользователи')

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
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Пользователи</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Список аккаунтов, зарегистрированных в GameCloud.</p>
                </div>
                <div class="flex items-center gap-3">
                    <a
                        href="{{ route('admin.users.create') }}"
                        class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-slate-800"
                    >
                        Добавить пользователя
                    </a>
                </div>
            </div>

            <div class="grid gap-4 md:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Всего пользователей</div>
                    <div class="mt-2 text-2xl font-semibold text-slate-100">{{ (int) ($counts['total'] ?? 0) }}</div>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Администраторы</div>
                    <div class="mt-2 text-2xl font-semibold text-slate-100">{{ (int) ($counts['admins'] ?? 0) }}</div>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Пользователи</div>
                    <div class="mt-2 text-2xl font-semibold text-slate-100">{{ (int) ($counts['users'] ?? 0) }}</div>
                </div>
            </div>

            <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                <form method="GET" action="{{ route('admin.users') }}" class="grid gap-3 md:grid-cols-12 md:items-end">
                    <div class="md:col-span-6">
                        <label for="q" class="block text-[11px] font-semibold uppercase tracking-wide text-slate-300/70">Поиск</label>
                        <input
                            id="q"
                            name="q"
                            value="{{ (string) ($q ?? '') }}"
                            placeholder="Имя, фамилия, логин, email"
                            class="mt-2 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                    </div>
                    <div class="md:col-span-3 md:max-w-xs">
                        <label for="role" class="block text-[11px] font-semibold uppercase tracking-wide text-slate-300/70">Группа</label>
                        <select
                            id="role"
                            name="role"
                            class="mt-2 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                            <option value="" {{ (string) ($role ?? '') === '' ? 'selected' : '' }}>Все</option>
                            <option value="admin" {{ (string) ($role ?? '') === 'admin' ? 'selected' : '' }}>Администраторы</option>
                            <option value="user" {{ (string) ($role ?? '') === 'user' ? 'selected' : '' }}>Пользователи</option>
                        </select>
                    </div>
                    <div class="md:col-span-3 flex gap-2 md:justify-end">
                        <button type="submit" class="inline-flex h-10 w-full items-center justify-center rounded-xl bg-sky-600 px-4 text-xs font-semibold text-white shadow-sm hover:bg-sky-500 md:w-32">
                            Найти
                        </button>
                        <a href="{{ route('admin.users') }}" class="inline-flex h-10 w-full items-center justify-center rounded-xl border border-white/10 bg-black/10 px-4 text-xs font-semibold text-slate-200 hover:bg-black/15 md:w-28">
                            Сброс
                        </a>
                    </div>
                </form>
            </div>

            <div class="overflow-hidden rounded-2xl border border-white/10 bg-[#242f3d] shadow-sm shadow-black/20 text-sm">
                <table class="hidden md:table min-w-full divide-y divide-white/10">
                    <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                        <tr>
                            <th scope="col" class="px-4 py-3 text-left">Пользователь</th>
                            <th scope="col" class="px-4 py-3 text-left">Логин</th>
                            <th scope="col" class="px-4 py-3 text-left">Email</th>
                            <th scope="col" class="px-4 py-3 text-left">Баланс</th>
                            <th scope="col" class="px-4 py-3 text-left">Группа</th>
                            <th scope="col" class="px-4 py-3 text-left">Регистрация</th>
                            <th scope="col" class="px-4 py-3 text-left">Действия</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-white/10 bg-[#242f3d] text-[13px]">
                        @forelse ($users as $user)
                            <tr>
                                <td class="px-4 py-3 align-top">
                                    @php
                                        $n = (string) ($user->name ?? '');
                                        $ln = (string) ($user->last_name ?? '');
                                        $initials = trim(mb_strtoupper(mb_substr($n, 0, 1) . mb_substr($ln, 0, 1)));
                                        $initials = $initials !== '' ? $initials : mb_strtoupper(mb_substr((string) $user->email, 0, 1));
                                    @endphp
                                    <div class="flex items-center gap-3">
                                        <div class="h-9 w-9 rounded-xl bg-black/20 ring-1 ring-white/10 flex items-center justify-center text-[11px] font-semibold text-slate-100">
                                            {{ $initials }}
                                        </div>
                                        <div>
                                            <div class="font-medium text-slate-100">
                                                {{ $user->name ?? 'Без имени' }}
                                                @if(!empty($user->last_name))
                                                    <span class="text-slate-300/80">{{ $user->last_name }}</span>
                                                @endif
                                            </div>
                                            <div class="text-[11px] text-slate-300/70">ID: {{ $user->id }}</div>
                                        </div>
                                    </div>
                                </td>
                                <td class="px-4 py-3 align-top text-slate-200">
                                    {{ $user->public_id ?? '—' }}
                                </td>
                                <td class="px-4 py-3 align-top text-slate-200">
                                    {{ $user->email }}
                                </td>
                                <td class="px-4 py-3 align-top text-slate-100 font-semibold">
                                    {{ number_format((float) ($user->balance ?? 0), 2, '.', ' ') }} ₽
                                </td>
                                <td class="px-4 py-3 align-top">
                                    @if($user->is_admin)
                                        <span class="inline-flex items-center rounded-full bg-slate-900 px-2 py-0.5 text-[11px] font-medium text-white">Администратор</span>
                                    @else
                                        <span class="inline-flex items-center rounded-full bg-black/10 px-2 py-0.5 text-[11px] font-medium text-slate-200 ring-1 ring-white/10">Пользователь</span>
                                    @endif
                                </td>
                                <td class="px-4 py-3 align-top text-slate-300/80">
                                    {{ $user->created_at?->format('d.m.Y H:i') ?? '—' }}
                                </td>
                                <td class="px-4 py-3 align-top text-slate-300/80">
                                    <div class="flex items-center gap-2">
                                        <a
                                            href="{{ route('admin.users.show', $user) }}"
                                            class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                            title="Просмотреть пользователя"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                                <path d="M1 12s4-8 9-4 9 4 9 4-4 8-9 4-9-4-9-4Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                <circle cx="10" cy="8" r="2" stroke-width="1.4" />
                                            </svg>
                                            <span class="sr-only">Просмотреть</span>
                                        </a>

                                        <a
                                            href="{{ route('admin.users.edit', $user) }}"
                                            class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                            title="Редактировать пользователя"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                                <path d="M5 13.5 4 16l2.5-1 7.5-7.5-1.5-1.5L5 13.5Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                <path d="M11.5 4 13 2.5 15.5 5 14 6.5" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                            </svg>
                                            <span class="sr-only">Редактировать</span>
                                        </a>

                                        <form
                                            method="POST"
                                            action="{{ route('admin.users.destroy', $user) }}"
                                            onsubmit="return confirm('Удалить пользователя {{ $user->email }}?');"
                                        >
                                            @csrf
                                            @method('DELETE')
                                            <button
                                                type="submit"
                                                class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-rose-500/30 bg-rose-500/10 text-rose-200 hover:bg-rose-500/15"
                                                title="Удалить пользователя"
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
                                <td colspan="4" class="px-4 py-8 text-center text-[13px] text-slate-300/80">
                                    Пользователи пока не найдены.
                                </td>
                            </tr>
                        @endforelse
                    </tbody>
                </table>

                @if ($users instanceof \Illuminate\Contracts\Pagination\Paginator || $users instanceof \Illuminate\Contracts\Pagination\LengthAwarePaginator)
                    <div class="border-t border-white/10 bg-black/10 px-4 py-3 text-xs text-slate-300/80">
                        {{ $users->links() }}
                    </div>
                @endif

                <!-- Mobile cards -->
                <div class="md:hidden divide-y divide-white/10">
                    @forelse ($users as $user)
                        <div class="px-4 py-4 space-y-3">
                            <div class="flex items-start justify-between gap-3">
                                @php
                                    $n = (string) ($user->name ?? '');
                                    $ln = (string) ($user->last_name ?? '');
                                    $initials = trim(mb_strtoupper(mb_substr($n, 0, 1) . mb_substr($ln, 0, 1)));
                                    $initials = $initials !== '' ? $initials : mb_strtoupper(mb_substr((string) $user->email, 0, 1));
                                @endphp
                                <div class="flex items-center gap-3">
                                    <div class="h-10 w-10 rounded-2xl bg-black/20 ring-1 ring-white/10 flex items-center justify-center text-[11px] font-semibold text-slate-100">
                                        {{ $initials }}
                                    </div>
                                    <div>
                                        <div class="font-medium text-slate-100">{{ $user->name ?? 'Без имени' }} {{ $user->last_name ?? '' }}</div>
                                        <div class="text-[11px] text-slate-300/70">{{ $user->public_id ?? ('ID: ' . $user->id) }}</div>
                                    </div>
                                </div>
                                @if($user->is_admin)
                                    <span class="inline-flex items-center rounded-full bg-slate-900 px-2 py-0.5 text-[11px] font-medium text-white">Администратор</span>
                                @else
                                    <span class="inline-flex items-center rounded-full bg-black/10 px-2 py-0.5 text-[11px] font-medium text-slate-200 ring-1 ring-white/10">Пользователь</span>
                                @endif
                            </div>
                            <div class="rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-200">
                                <div class="flex items-center justify-between">
                                    <span class="text-slate-300/70">Email</span>
                                    <span class="font-medium text-slate-100">{{ $user->email }}</span>
                                </div>
                                <div class="mt-1 flex items-center justify-between">
                                    <span class="text-slate-300/70">Баланс</span>
                                    <span class="font-semibold text-slate-100">{{ number_format((float) ($user->balance ?? 0), 2, '.', ' ') }} ₽</span>
                                </div>
                            </div>
                            <div class="text-xs text-slate-300/70">Регистрация: {{ $user->created_at?->format('d.m.Y H:i') ?? '—' }}</div>
                            <div class="flex items-center gap-2 pt-2 border-t border-white/10">
                                <a
                                    href="{{ route('admin.users.show', $user) }}"
                                    class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                    title="Просмотреть пользователя"
                                >
                                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                        <path d="M1 12s4-8 9-4 9 4 9 4-4 8-9 4-9-4-9-4Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                        <circle cx="10" cy="8" r="2" stroke-width="1.4" />
                                    </svg>
                                </a>
                                <a
                                    href="{{ route('admin.users.edit', $user) }}"
                                    class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                    title="Редактировать пользователя"
                                >
                                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-4 w-4">
                                        <path d="M5 13.5 4 16l2.5-1 7.5-7.5-1.5-1.5L5 13.5Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                        <path d="M11.5 4 13 2.5 15.5 5 14 6.5" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                    </svg>
                                </a>
                                <form
                                    method="POST"
                                    action="{{ route('admin.users.destroy', $user) }}"
                                    onsubmit="return confirm('Удалить пользователя {{ $user->email }}?');"
                                >
                                    @csrf
                                    @method('DELETE')
                                    <button
                                        type="submit"
                                        class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-rose-500/30 bg-rose-500/10 text-rose-200 hover:bg-rose-500/15"
                                        title="Удалить пользователя"
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
                        <div class="px-4 py-8 text-center text-[13px] text-slate-300/80">
                            Пользователи пока не найдены.
                        </div>
                    @endforelse
                </div>
        </div>
    </section>
@endsection
