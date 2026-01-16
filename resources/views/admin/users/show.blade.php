@extends('layouts.app-admin')

@section('page_title', 'Пользователь: ' . $user->name)

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">
                        Пользователь: {{ $user->name }}
                    </h1>
                    <p class="mt-1 text-sm text-slate-300/80">
                        Детальная информация о пользователе и его активности.
                    </p>
                </div>
                <div class="flex items-center gap-3 text-xs">
                    <a
                        href="{{ route('admin.users') }}"
                        class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                    >
                        Назад к списку
                    </a>
                    <a
                        href="{{ route('admin.users.edit', $user) }}"
                        class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-[11px] font-semibold text-white shadow-sm hover:bg-slate-800"
                    >
                        Редактировать
                    </a>
                </div>
            </div>

            <div class="grid gap-4 md:grid-cols-3 text-sm">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Статус</p>
                    <p class="mt-2 text-sm font-semibold {{ $user->is_admin ? 'text-emerald-600' : 'text-slate-500' }}">
                        {{ $user->is_admin ? 'Администратор' : 'Пользователь' }}
                    </p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Управляет уровнем доступа к системе.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Email</p>
                    <p class="mt-2 text-sm font-semibold text-slate-100">{{ $user->email }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Адрес электронной почты для авторизации.</p>
                </div>
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <p class="text-xs font-semibold text-slate-300/70">Дата регистрации</p>
                    <p class="mt-2 text-sm font-semibold text-slate-100">{{ $user->created_at->format('d.m.Y H:i') }}</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Когда пользователь зарегистрировался.</p>
                </div>
            </div>

            <div class="grid gap-4 md:grid-cols-3 text-sm">
                <div class="md:col-span-3 rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <h2 class="text-sm font-semibold text-slate-100">Информация о пользователе</h2>

                    <div class="mt-3 grid gap-3 md:grid-cols-2 text-[13px]">
                        <div class="rounded-xl border border-white/10 bg-black/10 px-3 py-2">
                            <div class="text-[11px] text-slate-300/70">Имя</div>
                            <div class="mt-1 font-medium text-slate-100">{{ $user->name }}</div>
                        </div>
                        <div class="rounded-xl border border-white/10 bg-black/10 px-3 py-2">
                            <div class="text-[11px] text-slate-300/70">Фамилия</div>
                            <div class="mt-1 font-medium text-slate-100">{{ $user->last_name ?? '—' }}</div>
                        </div>
                        <div class="rounded-xl border border-white/10 bg-black/10 px-3 py-2">
                            <div class="text-[11px] text-slate-300/70">Логин</div>
                            <div class="mt-1 font-medium text-slate-100">{{ $user->public_id ?? '—' }}</div>
                        </div>
                        <div class="rounded-xl border border-white/10 bg-black/10 px-3 py-2">
                            <div class="text-[11px] text-slate-300/70">Почта</div>
                            <div class="mt-1 font-medium text-slate-100">{{ $user->email }}</div>
                        </div>
                        <div class="rounded-xl border border-white/10 bg-black/10 px-3 py-2">
                            <div class="text-[11px] text-slate-300/70">Пароль</div>
                            <div class="mt-1 font-medium text-slate-300/70">••••••••</div>
                        </div>
                        <div class="rounded-xl border border-white/10 bg-black/10 px-3 py-2">
                            <div class="text-[11px] text-slate-300/70">Группа</div>
                            <div class="mt-1 font-medium text-slate-100">{{ $user->is_admin ? 'Администратор' : 'Пользователь' }}</div>
                        </div>
                        <div class="rounded-xl border border-white/10 bg-black/10 px-3 py-2">
                            <div class="text-[11px] text-slate-300/70">Баланс</div>
                            <div class="mt-1 font-medium text-slate-100">{{ number_format((float) ($user->balance ?? 0), 2, '.', ' ') }} ₽</div>
                        </div>
                        <div class="rounded-xl border border-white/10 bg-black/10 px-3 py-2">
                            <div class="text-[11px] text-slate-300/70">Дата регистрации</div>
                            <div class="mt-1 font-medium text-slate-100">{{ $user->created_at->format('d.m.Y H:i') }}</div>
                        </div>
                        <div class="rounded-xl border border-white/10 bg-black/10 px-3 py-2 md:col-span-2">
                            <div class="text-[11px] text-slate-300/70">Контакты</div>
                            <div class="mt-2 grid gap-2 md:grid-cols-4">
                                <div class="rounded-lg border border-white/10 bg-black/10 px-2 py-2">
                                    <div class="text-[11px] text-slate-300/70">Телефон</div>
                                    <div class="mt-1 text-slate-100">{{ $user->phone ?? '—' }}</div>
                                </div>
                                <div class="rounded-lg border border-white/10 bg-black/10 px-2 py-2">
                                    <div class="text-[11px] text-slate-300/70">Telegram</div>
                                    <div class="mt-1 text-slate-100">{{ $user->telegram_id ?? '—' }}</div>
                                </div>
                                <div class="rounded-lg border border-white/10 bg-black/10 px-2 py-2">
                                    <div class="text-[11px] text-slate-300/70">Discord</div>
                                    <div class="mt-1 text-slate-100">{{ $user->discord_id ?? '—' }}</div>
                                </div>
                                <div class="rounded-lg border border-white/10 bg-black/10 px-2 py-2">
                                    <div class="text-[11px] text-slate-300/70">VK</div>
                                    <div class="mt-1 text-slate-100">{{ $user->vk_id ?? '—' }}</div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div class="grid gap-4 md:grid-cols-2 text-sm">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <h2 class="text-sm font-semibold text-slate-100">Сервера пользователя</h2>
                    <div class="mt-3 overflow-x-auto">
                        <table class="min-w-full divide-y divide-white/10 text-[13px]">
                            <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                                <tr>
                                    <th class="px-3 py-2 text-left">ID</th>
                                    <th class="px-3 py-2 text-left">Название</th>
                                    <th class="px-3 py-2 text-left">Игра</th>
                                    <th class="px-3 py-2 text-left">IP</th>
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-white/10">
                                @forelse(($servers ?? []) as $srv)
                                    <tr class="hover:bg-black/10">
                                        <td class="px-3 py-2 text-slate-200">#{{ $srv->id }}</td>
                                        <td class="px-3 py-2 text-slate-100">{{ $srv->name ?? '—' }}</td>
                                        <td class="px-3 py-2 text-slate-200">{{ $srv->game?->name ?? '—' }}</td>
                                        <td class="px-3 py-2 text-slate-300/80">{{ $srv->ip_address ? ($srv->ip_address . ':' . $srv->port) : '—' }}</td>
                                    </tr>
                                @empty
                                    <tr>
                                        <td colspan="4" class="px-3 py-4 text-slate-300/70">Серверов нет</td>
                                    </tr>
                                @endforelse
                            </tbody>
                        </table>
                    </div>
                </div>

                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <h2 class="text-sm font-semibold text-slate-100">Последние операции</h2>
                    <div class="mt-3 overflow-x-auto">
                        <table class="min-w-full divide-y divide-white/10 text-[13px]">
                            <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                                <tr>
                                    <th class="px-3 py-2 text-left">Дата</th>
                                    <th class="px-3 py-2 text-left">Тип</th>
                                    <th class="px-3 py-2 text-left">Сумма</th>
                                    <th class="px-3 py-2 text-left">Описание</th>
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-white/10">
                                @forelse(($transactions ?? []) as $t)
                                    <tr class="hover:bg-black/10">
                                        <td class="px-3 py-2 text-slate-300/80">{{ $t->created_at?->format('d.m.Y H:i') ?? '—' }}</td>
                                        <td class="px-3 py-2 text-slate-200">{{ $t->type }}</td>
                                        <td class="px-3 py-2 text-slate-100">{{ number_format((float) $t->amount, 2, '.', ' ') }} ₽</td>
                                        <td class="px-3 py-2 text-slate-200">{{ $t->description }}</td>
                                    </tr>
                                @empty
                                    <tr>
                                        <td colspan="4" class="px-3 py-4 text-slate-300/70">Операций нет</td>
                                    </tr>
                                @endforelse
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>

            <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20 text-sm">
                <h2 class="text-sm font-semibold text-slate-100">Последние авторизации</h2>
                @if(config('session.driver') !== 'database')
                    <div class="mt-3 text-[13px] text-slate-300/70">Для отображения авторизаций включите драйвер сессий database.</div>
                @else
                    <div class="mt-3 overflow-x-auto">
                        <table class="min-w-full divide-y divide-white/10 text-[13px]">
                            <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                                <tr>
                                    <th class="px-3 py-2 text-left">IP</th>
                                    <th class="px-3 py-2 text-left">User-Agent</th>
                                    <th class="px-3 py-2 text-left">Последняя активность</th>
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-white/10">
                                @forelse(($sessions ?? []) as $s)
                                    <tr class="hover:bg-black/10">
                                        <td class="px-3 py-2 text-slate-200">{{ $s['ip_address'] ?? '—' }}</td>
                                        <td class="px-3 py-2 text-slate-300/80">{{ $s['user_agent'] ?? '—' }}</td>
                                        <td class="px-3 py-2 text-slate-100">{{ isset($s['last_activity']) ? $s['last_activity']->format('d.m.Y H:i') : '—' }}</td>
                                    </tr>
                                @empty
                                    <tr>
                                        <td colspan="3" class="px-3 py-4 text-slate-300/70">Авторизаций нет</td>
                                    </tr>
                                @endforelse
                            </tbody>
                        </table>
                    </div>
                @endif
            </div>
        </div>
    </section>
@endsection
