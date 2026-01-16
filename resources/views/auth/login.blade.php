@extends('layouts.landing')

@section('content')
    <section class="flex-1 flex items-center justify-center bg-[#17212b]">
        <div class="mx-auto w-full max-w-md px-4 py-10">
            <div class="mb-6 text-center">
                <h1 class="text-xl font-semibold text-slate-100">Вход в аккаунт</h1>
                <p class="mt-1 text-xs text-slate-300/80">Управляйте игровыми серверами в личном кабинете.</p>
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

                <form method="POST" action="{{ route('login') }}" class="space-y-4">
                    @csrf

                    <div class="space-y-1">
                        <label for="email" class="text-xs font-medium text-slate-200">Email</label>
                        <input
                            id="email"
                            name="email"
                            type="email"
                            value="{{ old('email') }}"
                            required
                            autofocus
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

                    <div class="flex items-center justify-between gap-2">
                        <label class="inline-flex items-center gap-2 text-[11px] text-slate-300/80">
                            <input
                                type="checkbox"
                                name="remember"
                                class="h-3 w-3 rounded border-white/10 bg-black/10 text-sky-600 focus:ring-sky-500"
                            >
                            <span>Запомнить меня</span>
                        </label>
                        <a href="{{ route('register') }}" class="text-[11px] font-medium text-sky-300 hover:text-sky-200">
                            Нет аккаунта? Регистрация
                        </a>
                    </div>

                    <button
                        type="submit"
                        class="w-full rounded-md bg-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-sky-500 transition"
                    >
                        Войти
                    </button>
                </form>
            </div>

            <div class="mt-4 rounded-2xl border border-white/10 bg-black/10 p-4 text-xs text-slate-200" id="demo-credentials">
                <div class="text-[11px] font-semibold uppercase tracking-wide text-slate-300/70">Демо-доступ</div>
                <div class="mt-2 grid gap-2">
                    <button
                        type="button"
                        class="w-full rounded-xl border border-white/10 bg-[#242f3d]/40 px-3 py-2 text-left hover:bg-[#242f3d]/60 transition"
                        data-demo-email="user@vortanix.app"
                        data-demo-password="password"
                    >
                        <div class="flex items-center justify-between gap-3">
                            <span class="text-slate-300/80">Email</span>
                            <span class="font-mono text-slate-100">user@vortanix.app</span>
                        </div>
                        <div class="mt-1 flex items-center justify-between gap-3">
                            <span class="text-slate-300/80">Password</span>
                            <span class="font-mono text-slate-100">password</span>
                        </div>
                    </button>
                    <div class="h-px bg-white/10"></div>
                    <button
                        type="button"
                        class="w-full rounded-xl border border-white/10 bg-[#242f3d]/40 px-3 py-2 text-left hover:bg-[#242f3d]/60 transition"
                        data-demo-email="admin@vortanix.app"
                        data-demo-password="password"
                    >
                        <div class="flex items-center justify-between gap-3">
                            <span class="text-slate-300/80">Email</span>
                            <span class="font-mono text-slate-100">admin@vortanix.app</span>
                        </div>
                        <div class="mt-1 flex items-center justify-between gap-3">
                            <span class="text-slate-300/80">Password</span>
                            <span class="font-mono text-slate-100">password</span>
                        </div>
                    </button>
                </div>
            </div>
        </div>
    </section>

    <script>
        (() => {
            const root = document.getElementById('demo-credentials');
            if (!root) return;

            root.addEventListener('click', (e) => {
                const btn = e.target && e.target.closest ? e.target.closest('[data-demo-email]') : null;
                if (!btn) return;

                const email = btn.getAttribute('data-demo-email') || '';
                const password = btn.getAttribute('data-demo-password') || '';

                const emailInput = document.getElementById('email');
                const passwordInput = document.getElementById('password');
                if (!emailInput || !passwordInput) return;

                emailInput.value = email;
                passwordInput.value = password;

                emailInput.dispatchEvent(new Event('input', { bubbles: true }));
                passwordInput.dispatchEvent(new Event('input', { bubbles: true }));

                passwordInput.focus();
            });
        })();
    </script>
@endsection
