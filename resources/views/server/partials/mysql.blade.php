@php
    $dbHost = (string) ($server->mysql_host ?? '');
    $dbPort = (int) ($server->mysql_port ?? 3306);
    $dbName = (string) ($server->mysql_database ?? '');
    $dbUser = (string) ($server->mysql_username ?? '');

    $dbPass = '';
    try {
        if (! empty($server->mysql_password)) {
            $dbPass = \Illuminate\Support\Facades\Crypt::decryptString($server->mysql_password);
        }
    } catch (\Throwable $e) {
        $dbPass = '';
    }
@endphp

@php
    $pmaHost = (string) ((($server->location->ip_address ?? '') ?: ($server->location->ssh_host ?? '')) ?: '');
    $pmaPort = (int) ($server->location->phpmyadmin_port ?? 0);
    $pmaUrl = ($pmaHost !== '' && $pmaPort > 0) ? ('http://' . $pmaHost . ':' . $pmaPort) : '';
@endphp

<div class="mt-6 rounded-2xl border border-white/10 bg-[#242f3d] text-slate-100 shadow-sm overflow-hidden">
    <div class="border-b border-white/10 bg-black/10 px-4 py-3 text-[11px] uppercase tracking-wide text-slate-300/70">
        MySQL доступ
    </div>
    <div class="p-4">
        @if ($dbHost === '' || $dbName === '' || $dbUser === '' || $dbPass === '')
            <div class="text-sm text-slate-200">
                MySQL доступ ещё не настроен для этого сервера.
            </div>

            <div class="mt-4">
                <form method="POST" action="{{ route('server.mysql.reset-password', $server) }}" onsubmit="return confirm('Создать MySQL для сервера и сгенерировать пароль?');">
                    @csrf
                    <button type="submit" class="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800">
                        Создать MySQL
                    </button>
                </form>
            </div>
        @else
            <div class="grid gap-3 text-sm">
                <div class="flex items-center justify-between border-b border-white/10 pb-2">
                    <span class="text-slate-300/70">Хост</span>
                    <span class="font-medium text-slate-100">{{ $dbHost }}</span>
                </div>
                <div class="flex items-center justify-between border-b border-white/10 pb-2">
                    <span class="text-slate-300/70">Порт</span>
                    <span class="font-medium text-slate-100">{{ $dbPort }}</span>
                </div>
                <div class="flex items-center justify-between border-b border-white/10 pb-2">
                    <span class="text-slate-300/70">База</span>
                    <span class="font-medium text-slate-100">{{ $dbName }}</span>
                </div>
                <div class="flex items-center justify-between border-b border-white/10 pb-2">
                    <span class="text-slate-300/70">Пользователь</span>
                    <span class="font-medium text-slate-100">{{ $dbUser }}</span>
                </div>
                <div class="flex items-center justify-between">
                    <span class="text-slate-300/70">Пароль</span>
                    <span class="font-medium text-slate-100">{{ $dbPass }}</span>
                </div>
            </div>

            <div class="mt-4">
                <form method="POST" action="{{ route('server.mysql.reset-password', $server) }}" onsubmit="return confirm('Сменить пароль MySQL? Текущий пароль перестанет работать.');">
                    @csrf
                    <button type="submit" class="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800">
                        Сменить пароль
                    </button>
                </form>
            </div>

            @if($pmaUrl !== '')
                <div class="mt-3">
                    <a href="{{ $pmaUrl }}" target="_blank" class="inline-flex items-center rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-500">
                        Открыть phpMyAdmin
                    </a>
                    <div class="mt-2 text-xs text-slate-300/70">Откроется {{ $pmaUrl }} (логин: {{ $dbUser }})</div>
                </div>
            @endif

            <div class="mt-4 rounded-md bg-black/10 border border-white/10 p-3 text-xs text-slate-200">
                Для подключения используй любой MySQL клиент:
                <br>
                Host: {{ $dbHost }}
                <br>
                Port: {{ $dbPort }}
                <br>
                Database: {{ $dbName }}
                <br>
                User: {{ $dbUser }}
                <br>
                Password: {{ $dbPass }}
            </div>
        @endif
    </div>
</div>
