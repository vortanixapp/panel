<!DOCTYPE html>
<html lang="{{ str_replace('_', '-', app()->getLocale()) }}">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="csrf-token" content="{{ csrf_token() }}">
    <title>{{ config('app.name', 'Vortanix') }} — Установка</title>
    @vite(['resources/css/app.css', 'resources/js/app.js'])
</head>
<body class="min-h-screen bg-[#17212b] text-slate-100">
<div class="mx-auto max-w-3xl px-4 py-10">
    <div class="rounded-3xl border border-white/10 bg-white/5 p-6">
        <div class="text-sm font-semibold text-slate-300">Установка панели</div>
        <div class="mt-2 text-2xl font-semibold">{{ config('app.name', 'Vortanix') }}</div>

        @if ($errors->any())
            <div class="mt-4 rounded-2xl border border-rose-500/20 bg-rose-500/10 p-4 text-sm text-rose-200">
                <div class="font-semibold">Ошибка</div>
                <div class="mt-2 grid gap-1">
                    @foreach ($errors->all() as $error)
                        <div>{{ $error }}</div>
                    @endforeach
                </div>
            </div>
        @endif

        <div class="mt-6 grid gap-4">
            <div class="rounded-2xl border border-white/10 bg-zinc-950/20 p-4">
                <div class="text-sm font-semibold">Проверка окружения</div>
                <div class="mt-3 grid gap-2 text-sm text-slate-200">
                    @foreach ($checks as $c)
                        <div class="flex items-center justify-between gap-3">
                            <div class="text-slate-300">{{ $c['label'] }}</div>
                            <div class="font-semibold {{ $c['ok'] ? 'text-emerald-200' : 'text-rose-200' }}">{{ $c['ok'] ? 'OK' : 'FAIL' }}</div>
                        </div>
                    @endforeach
                    <div class="flex items-center justify-between gap-3">
                        <div class="text-slate-300">Подключение к БД</div>
                        <div class="font-semibold {{ $dbOk ? 'text-emerald-200' : 'text-rose-200' }}">{{ $dbOk ? 'OK' : 'FAIL' }}</div>
                    </div>
                    @if (!$dbOk && $dbError)
                        <div class="text-xs text-rose-200/90 break-all">{{ $dbError }}</div>
                    @endif
                    <div class="flex items-center justify-between gap-3">
                        <div class="text-slate-300">APP_KEY</div>
                        <div class="font-semibold {{ $appKeyPresent ? 'text-emerald-200' : 'text-amber-200' }}">{{ $appKeyPresent ? 'установлен' : 'не установлен (будет сгенерирован)' }}</div>
                    </div>
                    <div class="flex items-center justify-between gap-3">
                        <div class="text-slate-300">.env доступен на запись</div>
                        <div class="font-semibold {{ $envWritable ? 'text-emerald-200' : 'text-amber-200' }}">{{ $envWritable ? 'да' : 'нет' }}</div>
                    </div>
                    <div class="flex items-center justify-between gap-3">
                        <div class="text-slate-300">Миграции</div>
                        <div class="font-semibold {{ $migrationsApplied ? 'text-emerald-200' : 'text-amber-200' }}">{{ $migrationsApplied ? 'применены' : 'не применены (будут применены)' }}</div>
                    </div>
                </div>
            </div>

            <form method="POST" action="{{ route('install.submit') }}" class="grid gap-4">
                @csrf

                <div class="rounded-2xl border border-white/10 bg-zinc-950/20 p-4">
                    <div class="text-sm font-semibold">Подключение к MySQL</div>
                    <div class="mt-4 grid gap-3 md:grid-cols-2">
                        <div>
                            <label class="text-xs font-semibold text-slate-400">Host</label>
                            <input name="db_host" value="{{ old('db_host', $db['host'] ?? '127.0.0.1') }}" class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white" required>
                        </div>
                        <div>
                            <label class="text-xs font-semibold text-slate-400">Port</label>
                            <input name="db_port" value="{{ old('db_port', $db['port'] ?? '3306') }}" class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white" required>
                        </div>
                        <div>
                            <label class="text-xs font-semibold text-slate-400">Database</label>
                            <input name="db_database" value="{{ old('db_database', $db['database'] ?? '') }}" class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white" required>
                        </div>
                        <div>
                            <label class="text-xs font-semibold text-slate-400">Username</label>
                            <input name="db_username" value="{{ old('db_username', $db['username'] ?? '') }}" class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white" required>
                        </div>
                        <div class="md:col-span-2">
                            <label class="text-xs font-semibold text-slate-400">Password</label>
                            <input name="db_password" type="password" value="{{ old('db_password', $db['password'] ?? '') }}" class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white">
                        </div>
                    </div>
                    @if (!$envWritable)
                        <div class="mt-3 text-xs text-amber-200/90">.env недоступен на запись — установщик сможет подключиться, но параметры БД не будут сохранены в .env автоматически.</div>
                    @endif
                </div>

                <div class="rounded-2xl border border-white/10 bg-zinc-950/20 p-4">
                    <div class="text-sm font-semibold">License Cloud</div>
                    <div class="mt-4 grid gap-3 md:grid-cols-2">
                        <div class="md:col-span-2">
                            <label class="text-xs font-semibold text-slate-400">LICENSE_CLOUD_URL</label>
                            <input name="license_cloud_url" value="{{ old('license_cloud_url', $license['url'] ?? '') }}" class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white" required>
                        </div>
                        <div>
                            <label class="text-xs font-semibold text-slate-400">LICENSE_CLOUD_PANEL_ID</label>
                            <input name="license_cloud_panel_id" value="{{ old('license_cloud_panel_id', $license['panel_id'] ?? '') }}" class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white" required>
                        </div>
                        <div>
                            <label class="text-xs font-semibold text-slate-400">LICENSE_CLOUD_SERVER_IP</label>
                            <input name="license_cloud_server_ip" value="{{ old('license_cloud_server_ip', $license['server_ip'] ?? '') }}" class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white">
                        </div>
                        <div class="md:col-span-2">
                            <label class="text-xs font-semibold text-slate-400">LICENSE_CLOUD_HMAC_SECRET</label>
                            <input name="license_cloud_hmac_secret" type="password" value="{{ old('license_cloud_hmac_secret', $license['hmac_secret'] ?? '') }}" class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white" required>
                        </div>
                    </div>
                    @if (!$envWritable)
                        <div class="mt-3 text-xs text-amber-200/90">.env недоступен на запись — значения License Cloud не будут сохранены в .env автоматически.</div>
                    @endif
                </div>

                <div class="rounded-2xl border border-white/10 bg-zinc-950/20 p-4">
                    <div class="text-sm font-semibold">Создать администратора</div>
                    <div class="mt-4 grid gap-3">
                        <div>
                            <label class="text-xs font-semibold text-slate-400">Имя</label>
                            <input name="admin_name" value="{{ old('admin_name') }}" class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white" required>
                        </div>
                        <div>
                            <label class="text-xs font-semibold text-slate-400">Email</label>
                            <input name="admin_email" type="email" value="{{ old('admin_email') }}" class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white" required>
                        </div>
                        <div class="grid gap-3 md:grid-cols-2">
                            <div>
                                <label class="text-xs font-semibold text-slate-400">Пароль</label>
                                <input name="admin_password" type="password" class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white" required>
                            </div>
                            <div>
                                <label class="text-xs font-semibold text-slate-400">Повтор пароля</label>
                                <input name="admin_password_confirmation" type="password" class="mt-2 w-full rounded-xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-white" required>
                            </div>
                        </div>
                    </div>
                </div>

                <button
                    type="submit"
                    class="inline-flex items-center justify-center rounded-xl bg-emerald-600 px-4 py-3 text-sm font-semibold text-white hover:bg-emerald-500"
                >
                    Установить
                </button>
            </form>

            <div class="text-xs text-slate-400">
                После установки тебя перенаправит в админ‑панель.
            </div>
        </div>
    </div>
</div>
</body>
</html>
