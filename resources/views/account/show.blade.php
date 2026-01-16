@extends('layouts.app-user')

@section('page_title', 'Профиль')

@section('content')
    <section class="py-6 md:py-8">
        <div class="w-full px-4 sm:px-6 lg:px-8">
            <div class="w-full space-y-6">
            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Профиль</h1>
                    <p class="mt-1 text-sm text-slate-300">Информация об аккаунте, контакты и настройки безопасности.</p>
                </div>
            </div>

            <div x-data="{ tab: 'profile' }" class="space-y-4">
                <div class="grid gap-4 md:grid-cols-[240px_1fr] md:items-start">
                    <div class="hidden md:block">
                        <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-2 text-[12px] text-slate-200 shadow-sm">
                            <button
                                type="button"
                                class="flex w-full items-center rounded-xl px-3 py-2 text-left transition-all duration-200"
                                :class="tab === 'profile' ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white'"
                                @click="tab = 'profile'"
                            >
                                Профиль
                            </button>
                            <button
                                type="button"
                                class="mt-1 flex w-full items-center rounded-xl px-3 py-2 text-left transition-all duration-200"
                                :class="tab === 'contacts' ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white'"
                                @click="tab = 'contacts'"
                            >
                                Контакты
                            </button>
                            <button
                                type="button"
                                class="mt-1 flex w-full items-center rounded-xl px-3 py-2 text-left transition-all duration-200"
                                :class="tab === 'security' ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white'"
                                @click="tab = 'security'"
                            >
                                Безопасность
                            </button>
                            <button
                                type="button"
                                class="mt-1 flex w-full items-center rounded-xl px-3 py-2 text-left transition-all duration-200"
                                :class="tab === '2fa' ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white'"
                                @click="tab = '2fa'"
                            >
                                2FA
                            </button>
                            <button
                                type="button"
                                class="mt-1 flex w-full items-center rounded-xl px-3 py-2 text-left transition-all duration-200"
                                :class="tab === 'sessions' ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white'"
                                @click="tab = 'sessions'"
                            >
                                Сессии
                            </button>
                        </div>
                    </div>

                    <div class="space-y-4">
                        <div class="md:hidden flex flex-wrap gap-1 rounded-2xl border border-white/10 bg-[#242f3d] p-1 text-[12px] text-slate-200 shadow-sm">
                            <button
                                type="button"
                                class="inline-flex items-center rounded-xl px-3 py-2 transition-all duration-200"
                                :class="tab === 'profile' ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white'"
                                @click="tab = 'profile'"
                            >
                                Профиль
                            </button>
                            <button
                                type="button"
                                class="inline-flex items-center rounded-xl px-3 py-2 transition-all duration-200"
                                :class="tab === 'contacts' ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white'"
                                @click="tab = 'contacts'"
                            >
                                Контакты
                            </button>
                            <button
                                type="button"
                                class="inline-flex items-center rounded-xl px-3 py-2 transition-all duration-200"
                                :class="tab === 'security' ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white'"
                                @click="tab = 'security'"
                            >
                                Безопасность
                            </button>
                            <button
                                type="button"
                                class="inline-flex items-center rounded-xl px-3 py-2 transition-all duration-200"
                                :class="tab === '2fa' ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white'"
                                @click="tab = '2fa'"
                            >
                                2FA
                            </button>
                            <button
                                type="button"
                                class="inline-flex items-center rounded-xl px-3 py-2 transition-all duration-200"
                                :class="tab === 'sessions' ? 'bg-black/15 text-white font-semibold ring-1 ring-white/10' : 'text-slate-200 hover:bg-black/10 hover:text-white'"
                                @click="tab = 'sessions'"
                            >
                                Сессии
                            </button>
                        </div>

                <div x-show="tab === 'profile'" x-transition:enter="transition ease-out duration-200" x-transition:enter-start="opacity-0 translate-y-1" x-transition:enter-end="opacity-100 translate-y-0" class="text-sm">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm transition-shadow duration-300 hover:shadow-md">
                        <p class="text-xs font-semibold text-slate-300/70">Основная информация</p>
                        <div class="mt-3 grid gap-4 md:grid-cols-2">
                            <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                                <div class="text-[11px] font-semibold uppercase tracking-wide text-slate-300/70">Аккаунт</div>
                                <dl class="mt-2 space-y-2 text-[13px] text-slate-200">
                                    <div class="flex items-center justify-between gap-3">
                                        <dt class="text-slate-300/70">Ваш ID</dt>
                                        <dd class="font-medium text-slate-100">{{ $user->public_id ?? $user->id }}</dd>
                                    </div>
                                    <div class="flex items-center justify-between gap-3">
                                        <dt class="text-slate-300/70">E-mail</dt>
                                        <dd class="font-medium text-slate-100 truncate max-w-[14rem] sm:max-w-[18rem]">{{ $user->email }}</dd>
                                    </div>
                                    <div class="flex items-center justify-between gap-3">
                                        <dt class="text-slate-300/70">С нами уже</dt>
                                        <dd class="font-medium text-slate-100">{{ optional($user->created_at)->diffForHumans() ?? '—' }}</dd>
                                    </div>
                                </dl>
                            </div>

                            <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                                <div class="text-[11px] font-semibold uppercase tracking-wide text-slate-300/70">Персональные данные</div>
                                <dl class="mt-2 space-y-2 text-[13px] text-slate-200">
                                    <div class="flex items-center justify-between gap-3">
                                        <dt class="text-slate-300/70">Имя</dt>
                                        <dd class="font-medium text-slate-100">{{ $user->name ?? '—' }}</dd>
                                    </div>
                                    <div class="flex items-center justify-between gap-3">
                                        <dt class="text-slate-300/70">Фамилия</dt>
                                        <dd class="font-medium text-slate-100">{{ $user->last_name ?? '—' }}</dd>
                                    </div>
                                    <div class="flex items-center justify-between gap-3">
                                        <dt class="text-slate-300/70">Номер телефона</dt>
                                        <dd class="font-medium text-slate-100">{{ $user->phone ?? '—' }}</dd>
                                    </div>
                                </dl>
                            </div>

                            <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                                <div class="text-[11px] font-semibold uppercase tracking-wide text-slate-300/70">Соцсети</div>
                                <dl class="mt-2 space-y-2 text-[13px] text-slate-200">
                                    <div class="flex items-center justify-between gap-3">
                                        <dt class="text-slate-300/70">Telegram ID</dt>
                                        <dd class="font-medium text-slate-100">{{ $user->telegram_id ?? '—' }}</dd>
                                    </div>
                                    <div class="flex items-center justify-between gap-3">
                                        <dt class="text-slate-300/70">Discord ID</dt>
                                        <dd class="font-medium text-slate-100">{{ $user->discord_id ?? '—' }}</dd>
                                    </div>
                                    <div class="flex items-center justify-between gap-3">
                                        <dt class="text-slate-300/70">VK ID</dt>
                                        <dd class="font-medium text-slate-100">{{ $user->vk_id ?? '—' }}</dd>
                                    </div>
                                </dl>
                            </div>

                            <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                                <div class="text-[11px] font-semibold uppercase tracking-wide text-slate-300/70">Профиль</div>
                                <div class="mt-2 flex items-center justify-between gap-3">
                                    <div>
                                        <div class="text-[12px] text-slate-300/70">Бонусы</div>
                                        <div class="mt-1 text-sm font-semibold text-slate-100">{{ $user->bonuses ?? 0 }}</div>
                                    </div>
                                    <div class="inline-flex items-center rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-[11px] font-semibold text-emerald-700">
                                        Аккаунт активен
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <div x-show="tab === 'contacts'" x-cloak x-transition:enter="transition ease-out duration-200" x-transition:enter-start="opacity-0 translate-y-1" x-transition:enter-end="opacity-100 translate-y-0" class="grid gap-4 lg:grid-cols-2 text-sm">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm transition-shadow duration-300 hover:shadow-md">
                        <h2 class="text-sm font-semibold text-slate-100">Контакты</h2>
                        <p class="mt-1 text-[11px] text-slate-300/70">Основные способы связи и уведомлений.</p>

                        <form method="POST" action="{{ route('account.email.update') }}" class="mt-3 space-y-3 text-[13px] text-slate-200">
                            @csrf

                            <div>
                                <label for="email" class="text-slate-300/70 text-xs">E-mail для уведомлений</label>
                                <input
                                    id="email"
                                    type="email"
                                    name="email"
                                    value="{{ old('email', $user->email) }}"
                                    class="mt-1 w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 placeholder-slate-400 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    required
                                    autocomplete="email"
                                >
                                @error('email')
                                    <p class="mt-1 text-[11px] text-red-600">{{ $message }}</p>
                                @enderror
                            </div>

                            <div>
                                <label for="current_password_email" class="text-slate-300/70 text-xs">Текущий пароль</label>
                                <input
                                    id="current_password_email"
                                    type="password"
                                    name="current_password"
                                    class="mt-1 w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 placeholder-slate-400 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    required
                                    autocomplete="current-password"
                                >
                                @error('current_password')
                                    <p class="mt-1 text-[11px] text-red-600">{{ $message }}</p>
                                @enderror
                            </div>

                            <div class="flex items-center justify-between gap-2 pt-1">
                                <p class="text-[11px] text-slate-300/70">Мы отправим уведомление на старый и новый адрес.</p>
                                <button type="submit" class="inline-flex items-center rounded-xl bg-slate-900 px-4 py-2 text-[11px] font-semibold text-white shadow-sm hover:bg-slate-800">
                                    Сохранить e-mail
                                </button>
                            </div>

                            @if(session('status') === 'email-updated')
                                <p class="text-[11px] text-emerald-600">E-mail успешно обновлён.</p>
                            @endif
                        </form>
                    </div>
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm transition-shadow duration-300 hover:shadow-md">
                        <h2 class="text-sm font-semibold text-slate-100">Дополнительные данные</h2>
                        <p class="mt-1 text-[11px] text-slate-300/70">Имя, фамилия и дополнительные контакты.</p>

                        <form method="POST" action="{{ route('account.profile.update') }}" class="mt-3 space-y-3 text-[13px] text-slate-200">
                            @csrf

                            <div class="grid gap-3 md:grid-cols-2">
                                <div>
                                    <label for="name" class="text-slate-300/70 text-xs">Имя</label>
                                    <input
                                        id="name"
                                        type="text"
                                        name="name"
                                        value="{{ old('name', $user->name) }}"
                                        class="mt-1 w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 placeholder-slate-400 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                        autocomplete="name"
                                    >
                                    @error('name')
                                        <p class="mt-1 text-[11px] text-red-600">{{ $message }}</p>
                                    @enderror
                                </div>
                                <div>
                                    <label for="last_name" class="text-slate-300/70 text-xs">Фамилия</label>
                                    <input
                                        id="last_name"
                                        type="text"
                                        name="last_name"
                                        value="{{ old('last_name', $user->last_name) }}"
                                        class="mt-1 w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 placeholder-slate-400 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                        autocomplete="family-name"
                                    >
                                    @error('last_name')
                                        <p class="mt-1 text-[11px] text-red-600">{{ $message }}</p>
                                    @enderror
                                </div>
                            </div>

                            <div>
                                <label for="phone" class="text-slate-300/70 text-xs">Номер телефона</label>
                                <input
                                    id="phone"
                                    type="text"
                                    name="phone"
                                    value="{{ old('phone', $user->phone) }}"
                                    class="mt-1 w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 placeholder-slate-400 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    autocomplete="tel"
                                >
                                @error('phone')
                                    <p class="mt-1 text-[11px] text-red-600">{{ $message }}</p>
                                @enderror
                            </div>

                            <div class="grid gap-3 md:grid-cols-3">
                                <div>
                                    <label for="telegram_id" class="text-slate-300/70 text-xs">Telegram ID</label>
                                    <input
                                        id="telegram_id"
                                        type="text"
                                        name="telegram_id"
                                        value="{{ old('telegram_id', $user->telegram_id) }}"
                                        class="mt-1 w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 placeholder-slate-400 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                    @error('telegram_id')
                                        <p class="mt-1 text-[11px] text-red-600">{{ $message }}</p>
                                    @enderror
                                </div>
                                <div>
                                    <label for="discord_id" class="text-slate-300/70 text-xs">Discord ID</label>
                                    <input
                                        id="discord_id"
                                        type="text"
                                        name="discord_id"
                                        value="{{ old('discord_id', $user->discord_id) }}"
                                        class="mt-1 w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 placeholder-slate-400 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                    @error('discord_id')
                                        <p class="mt-1 text-[11px] text-red-600">{{ $message }}</p>
                                    @enderror
                                </div>
                                <div>
                                    <label for="vk_id" class="text-slate-300/70 text-xs">VK ID</label>
                                    <input
                                        id="vk_id"
                                        type="text"
                                        name="vk_id"
                                        value="{{ old('vk_id', $user->vk_id) }}"
                                        class="mt-1 w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 placeholder-slate-400 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    >
                                    @error('vk_id')
                                        <p class="mt-1 text-[11px] text-red-600">{{ $message }}</p>
                                    @enderror
                                </div>
                            </div>

                            <div class="flex items-center justify-between gap-2 pt-1">
                                <p class="text-[11px] text-slate-300/70">Эти данные помогают нам связаться с вами при необходимости.</p>
                                <button type="submit" class="inline-flex items-center rounded-xl border border-white/10 bg-black/10 px-4 py-2 text-[11px] font-semibold text-slate-200 shadow-sm transition hover:bg-black/15 hover:text-white">
                                    Сохранить профиль
                                </button>
                            </div>

                            @if(session('status') === 'profile-updated')
                                <p class="text-[11px] text-emerald-600">Профиль успешно обновлён.</p>
                            @endif
                        </form>
                    </div>
                </div>

                <div x-show="tab === 'security'" x-cloak x-transition:enter="transition ease-out duration-200" x-transition:enter-start="opacity-0 translate-y-1" x-transition:enter-end="opacity-100 translate-y-0" class="text-sm">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm transition-shadow duration-300 hover:shadow-md">
                        <h2 class="text-sm font-semibold text-slate-100">Безопасность</h2>
                        <p class="mt-1 text-[11px] text-slate-300/70">Пароль и общие настройки безопасности аккаунта.</p>

                        <form method="POST" action="{{ route('account.password.update') }}" class="mt-3 space-y-3 text-[13px] text-slate-200">
                            @csrf

                            <div>
                                <label for="current_password" class="text-xs text-slate-300/70">Текущий пароль</label>
                                <input
                                    id="current_password"
                                    type="password"
                                    name="current_password"
                                    class="mt-1 w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 placeholder-slate-400 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    required
                                    autocomplete="current-password"
                                >
                                @error('current_password')
                                    <p class="mt-1 text-[11px] text-red-600">{{ $message }}</p>
                                @enderror
                            </div>

                            <div>
                                <label for="password" class="text-xs text-slate-300/70">Новый пароль</label>
                                <input
                                    id="password"
                                    type="password"
                                    name="password"
                                    class="mt-1 w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 placeholder-slate-400 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    required
                                    autocomplete="new-password"
                                >
                                @error('password')
                                    <p class="mt-1 text-[11px] text-red-600">{{ $message }}</p>
                                @enderror
                            </div>

                            <div>
                                <label for="password_confirmation" class="text-xs text-slate-300/70">Подтверждение пароля</label>
                                <input
                                    id="password_confirmation"
                                    type="password"
                                    name="password_confirmation"
                                    class="mt-1 w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 placeholder-slate-400 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    required
                                    autocomplete="new-password"
                                >
                            </div>

                            <div class="flex items-center justify-between gap-2 pt-1">
                                <p class="text-[11px] text-slate-300/70">Используйте сложный пароль длиной не менее 8 символов.</p>
                                <button type="submit" class="inline-flex items-center rounded-xl border border-white/10 bg-black/10 px-4 py-2 text-[11px] font-semibold text-slate-200 shadow-sm transition hover:bg-black/15 hover:text-white">
                                    Обновить пароль
                                </button>
                            </div>

                            @if(session('status') === 'password-updated')
                                <p class="text-[11px] text-emerald-600">Пароль успешно обновлён.</p>
                            @endif
                        </form>
                    </div>
                </div>

                <div x-show="tab === '2fa'" x-cloak x-transition:enter="transition ease-out duration-200" x-transition:enter-start="opacity-0 translate-y-1" x-transition:enter-end="opacity-100 translate-y-0" class="text-sm">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm transition-shadow duration-300 hover:shadow-md">
                        <h2 class="text-sm font-semibold text-slate-100">Двухфакторная аутентификация</h2>
                        <p class="mt-1 text-[11px] text-slate-300/70">Дополнительный уровень защиты вашего аккаунта.</p>
                        <div class="mt-3 flex items-center justify-between">
                            <div>
                                <p class="text-xs text-slate-300/70">Статус</p>
                                <p class="mt-1 inline-flex items-center gap-1 rounded-full bg-black/10 px-2 py-0.5 text-[11px] text-slate-200 ring-1 ring-white/10">
                                    <span class="h-1.5 w-1.5 rounded-full bg-slate-400"></span>
                                    Отключена
                                </p>
                            </div>
                            <button type="button" class="inline-flex items-center rounded-xl bg-slate-900 px-4 py-2 text-[11px] font-semibold text-white shadow-sm hover:bg-slate-800">
                                Настроить 2FA
                            </button>
                        </div>
                    </div>
                </div>

                <div x-show="tab === 'sessions'" x-cloak x-transition:enter="transition ease-out duration-200" x-transition:enter-start="opacity-0 translate-y-1" x-transition:enter-end="opacity-100 translate-y-0" class="text-sm">
                    <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm transition-shadow duration-300 hover:shadow-md">
                        <h2 class="text-sm font-semibold text-slate-100">Активные сессии</h2>
                        <p class="mt-1 text-[11px] text-slate-300/70">Подключённые устройства и недавние входы.</p>

                        @if(isset($sessions) && $sessions->isNotEmpty())
                            <div class="mt-3 mb-2 flex items-center justify-between text-[11px] text-slate-300/70">
                                <span>Всего сессий: {{ $sessions->count() }}</span>
                                <form method="POST" action="{{ route('account.sessions.destroy') }}" class="inline-flex items-center gap-2">
                                    @csrf
                                    <input type="hidden" name="all_others" value="1">
                                    <button type="submit" class="inline-flex items-center rounded-xl border border-white/10 bg-black/10 px-4 py-2 text-[11px] font-semibold text-slate-200 shadow-sm transition hover:bg-black/15 hover:text-white">
                                        Завершить все другие сессии
                                    </button>
                                </form>
                            </div>

                            <div class="mt-1 space-y-2 text-[12px] text-slate-200 max-h-56 overflow-y-auto">
                                @foreach($sessions as $session)
                                    <div class="rounded-2xl border border-white/10 bg-black/10 px-4 py-3 transition hover:bg-black/15">
                                        <div class="flex items-center justify-between gap-2">
                                            <div class="font-medium text-slate-100 truncate">
                                                {{ Str::limit($session['user_agent'] ?? 'Неизвестное устройство', 40) }}
                                            </div>
                                            <div class="flex items-center gap-2">
                                                @if(isset($currentSessionId) && $session['id'] === $currentSessionId)
                                                    <span class="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-[10px] font-medium text-emerald-700">Текущая</span>
                                                @else
                                                    <form method="POST" action="{{ route('account.sessions.destroy') }}" class="inline-flex">
                                                        @csrf
                                                        <input type="hidden" name="session_id" value="{{ $session['id'] }}">
                                                        <button type="submit" class="inline-flex items-center rounded-xl border border-rose-200 bg-rose-50 px-3 py-1 text-[10px] font-semibold text-rose-700 shadow-sm transition hover:bg-rose-100">
                                                            Завершить
                                                        </button>
                                                    </form>
                                                @endif
                                            </div>
                                        </div>
                                        <div class="mt-1 flex items-center justify-between text-[11px] text-slate-300/70">
                                            <span>IP: {{ $session['ip_address'] ?? '—' }}</span>
                                            <span>Активность: {{ $session['last_activity']->diffForHumans() }}</span>
                                        </div>
                                    </div>
                                @endforeach
                            </div>
                        @else
                            <div class="mt-3 rounded-2xl border border-dashed border-white/10 bg-black/10 p-4 text-[12px] text-slate-300/70">
                                Активных сессий не найдено или хранение сессий в базе данных отключено.
                            </div>
                        @endif
                    </div>
                </div>
            </div>
        </div>
    </section>
@endsection
