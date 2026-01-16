@extends('layouts.landing')

@section('content')
    <section class="flex-1 flex items-center justify-center bg-[#17212b]">
        <div class="mx-auto w-full max-w-md px-4 py-10">
            <div class="mb-6 text-center">
                <h1 class="text-xl font-semibold text-slate-100">Регистрация</h1>
                <p class="mt-1 text-xs text-slate-300/80">Создайте аккаунт, чтобы управлять игровыми серверами.</p>
            </div>

            <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-6 shadow-sm">
                @if ($errors->any())
                    <div class="mb-4 rounded-md border border-rose-500/20 bg-rose-500/10 px-3 py-2 text-xs text-rose-200">
                        <ul class="list-disc space-y-1 pl-4">
                            @foreach ($errors->all() as $error)
                                <li>{{ $error }}</li>
                            @endforeach
                        </ul>
                    </div>
                @endif

                <form method="POST" action="{{ route('register') }}" class="space-y-4">
                    @csrf

                    <div class="grid gap-4 md:grid-cols-2">
                        <div class="space-y-1">
                            <label for="name" class="text-xs font-medium text-slate-200">Имя</label>
                            <input
                                id="name"
                                name="name"
                                type="text"
                                value="{{ old('name') }}"
                                required
                                autofocus
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                        <div class="space-y-1">
                            <label for="last_name" class="text-xs font-medium text-slate-200">Фамилия</label>
                            <input
                                id="last_name"
                                name="last_name"
                                type="text"
                                value="{{ old('last_name') }}"
                                required
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
                            value="{{ old('email') }}"
                            required
                            class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                    </div>

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

                    <div class="flex items-center justify-between gap-2">
                        <p class="text-[11px] text-slate-300/70">
                            Регистрируясь, вы соглашаетесь с условиями сервиса.
                        </p>
                        <a href="{{ route('login') }}" class="text-[11px] font-medium text-sky-300 hover:text-sky-200">
                            Уже есть аккаунт? Войти
                        </a>
                    </div>

                    <button
                        type="submit"
                        class="w-full rounded-md bg-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-sky-500 transition"
                    >
                        Создать аккаунт
                    </button>
                </form>
            </div>
        </div>
    </section>
@endsection
