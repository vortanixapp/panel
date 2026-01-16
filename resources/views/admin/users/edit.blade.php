@extends('layouts.app-admin')

@section('page_title', 'Редактирование пользователя')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div>
                <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Редактирование пользователя</h1>
                <p class="mt-1 text-sm text-slate-300/80">Измените данные аккаунта и права доступа.</p>
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

                <form method="POST" action="{{ route('admin.users.update', $user) }}" class="space-y-4">
                    @csrf
                    @method('PUT')

                    <div class="grid gap-4 md:grid-cols-2">
                        <div class="space-y-1">
                            <label for="name" class="text-xs font-medium text-slate-200">Имя</label>
                            <input
                                id="name"
                                name="name"
                                type="text"
                                value="{{ old('name', $user->name) }}"
                                required
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>

                        <div class="space-y-1">
                            <label for="last_name" class="text-xs font-medium text-slate-200">Фамилия</label>
                            <input
                                id="last_name"
                                name="last_name"
                                type="text"
                                value="{{ old('last_name', $user->last_name) }}"
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                    </div>

                    <div class="grid gap-4 md:grid-cols-2">
                        <div class="space-y-1">
                            <label for="public_id" class="text-xs font-medium text-slate-200">Логин</label>
                            <input
                                id="public_id"
                                name="public_id"
                                type="text"
                                value="{{ old('public_id', $user->public_id) }}"
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>

                        <div class="space-y-1">
                            <label for="balance" class="text-xs font-medium text-slate-200">Баланс</label>
                            <input
                                id="balance"
                                name="balance"
                                type="number"
                                step="0.01"
                                min="0"
                                value="{{ old('balance', (string) $user->balance) }}"
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                    </div>

                    <div class="space-y-1">
                        <label for="email" class="text-xs font-medium text-slate-200">Email</label>
                        <input
                            id="email"
                            name="email"
                            type="email"
                            value="{{ old('email', $user->email) }}"
                            required
                            class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                    </div>

                    <div class="grid gap-4 md:grid-cols-2">
                        <div class="space-y-1">
                            <label for="phone" class="text-xs font-medium text-slate-200">Контакты (телефон)</label>
                            <input
                                id="phone"
                                name="phone"
                                type="text"
                                value="{{ old('phone', $user->phone) }}"
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                        <div class="space-y-1">
                            <label for="telegram_id" class="text-xs font-medium text-slate-200">Telegram</label>
                            <input
                                id="telegram_id"
                                name="telegram_id"
                                type="text"
                                value="{{ old('telegram_id', $user->telegram_id) }}"
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                    </div>

                    <div class="grid gap-4 md:grid-cols-2">
                        <div class="space-y-1">
                            <label for="discord_id" class="text-xs font-medium text-slate-200">Discord</label>
                            <input
                                id="discord_id"
                                name="discord_id"
                                type="text"
                                value="{{ old('discord_id', $user->discord_id) }}"
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                        <div class="space-y-1">
                            <label for="vk_id" class="text-xs font-medium text-slate-200">VK</label>
                            <input
                                id="vk_id"
                                name="vk_id"
                                type="text"
                                value="{{ old('vk_id', $user->vk_id) }}"
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                    </div>

                    <div class="grid gap-4 md:grid-cols-2">
                        <div class="space-y-1">
                            <label for="password" class="text-xs font-medium text-slate-200">Новый пароль</label>
                            <input
                                id="password"
                                name="password"
                                type="password"
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                            <p class="mt-1 text-[11px] text-slate-300/70">Оставьте поле пустым, чтобы не менять пароль.</p>
                        </div>

                        <div class="space-y-1">
                            <label for="password_confirmation" class="text-xs font-medium text-slate-200">Подтверждение пароля</label>
                            <input
                                id="password_confirmation"
                                name="password_confirmation"
                                type="password"
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                    </div>

                    <div class="space-y-1">
                        <label for="is_admin" class="text-xs font-medium text-slate-200">Группа</label>
                        <select
                            id="is_admin"
                            name="is_admin"
                            class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                            <option value="0" {{ (string) old('is_admin', (int) $user->is_admin) === '0' ? 'selected' : '' }}>Пользователь</option>
                            <option value="1" {{ (string) old('is_admin', (int) $user->is_admin) === '1' ? 'selected' : '' }}>Администратор</option>
                        </select>
                    </div>

                    <div class="flex items-center justify-end gap-3 pt-4">
                        <a href="{{ route('admin.users') }}" class="text-xs text-slate-300/80 hover:text-white">Отмена</a>
                        <button
                            type="submit"
                            class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-slate-800"
                        >
                            Сохранить изменения
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </section>
@endsection
