@extends('layouts.app-admin')

@section('page_title', 'Новый пользователь')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div>
                <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Новый пользователь</h1>
                <p class="mt-1 text-sm text-slate-300/80">Создайте аккаунт вручную и при необходимости задайте права администратора.</p>
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

                <form method="POST" action="{{ route('admin.users.store') }}" class="space-y-4">
                    @csrf

                    <div class="space-y-1">
                        <label for="name" class="text-xs font-medium text-slate-200">Имя</label>
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
                        <label for="email" class="text-xs font-medium text-slate-200">Email</label>
                        <input
                            id="email"
                            name="email"
                            type="email"
                            value="{{ old('email') }}"
                            required
                            class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                    </div>

                    <div class="grid gap-4 md:grid-cols-2">
                        <div class="space-y-1">
                            <label for="password" class="text-xs font-medium text-slate-200">Пароль</label>
                            <input
                                id="password"
                                name="password"
                                type="password"
                                required
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>

                        <div class="space-y-1">
                            <label for="password_confirmation" class="text-xs font-medium text-slate-200">Подтверждение пароля</label>
                            <input
                                id="password_confirmation"
                                name="password_confirmation"
                                type="password"
                                required
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                    </div>

                    <div class="flex items-center justify-between gap-3 pt-2">
                        <label class="inline-flex items-center gap-2 text-[11px] text-slate-300/80">
                            <input
                                type="checkbox"
                                name="is_admin"
                                value="1"
                                class="h-3 w-3 rounded border-white/10 bg-black/10 text-slate-100 focus:ring-slate-100"
                                {{ old('is_admin') ? 'checked' : '' }}
                            >
                            <span>Сделать пользователем администратором</span>
                        </label>
                    </div>

                    <div class="flex items-center justify-end gap-3 pt-4">
                        <a href="{{ route('admin.users') }}" class="text-xs text-slate-300/80 hover:text-white">Отмена</a>
                        <button
                            type="submit"
                            class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-slate-800"
                        >
                            Создать пользователя
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </section>
@endsection
