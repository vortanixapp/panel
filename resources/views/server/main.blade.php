@extends($layout ?? 'layouts.app-user')

@section('title', $server->name)

@section('breadcrumb')
    <a href="{{ route('dashboard') }}" class="text-xs text-slate-200 hover:text-white">Кабинет</a>
    <span class="h-1 w-1 rounded-full bg-white/25"></span>
    <a href="{{ route('my-servers') }}" class="text-xs text-slate-200 hover:text-white">Мои серверы</a>
    <span class="h-1 w-1 rounded-full bg-white/25"></span>
    <span class="text-xs text-slate-100">{{ $server->name }}</span>
@endsection

@section('content')
    <section class="py-6">
        <div class="w-full px-4 sm:px-6 lg:px-8">
            @php
                $tab = 'main';
                $daysLeft = max(0, (int) ceil(now()->diffInSeconds($server->expires_at, false) / 86400));

                $runtime = strtolower((string) ($server->runtime_status ?? ''));
                $prov = strtolower((string) ($server->provisioning_status ?? ''));
                $isExpired = $server->expires_at && now()->greaterThan($server->expires_at);

                $label = 'Неизвестно';
                $badge = 'bg-slate-100 text-slate-700';

                if (in_array($prov, ['pending', 'installing'], true)) {
                    $label = 'Установка';
                    $badge = 'bg-amber-100 text-amber-700';
                } elseif ($prov === 'failed') {
                    $label = 'Ошибка установки';
                    $badge = 'bg-rose-100 text-rose-700';
                } elseif ($prov === 'reinstalling') {
                    $label = 'Переустановка';
                    $badge = 'bg-amber-100 text-amber-700';
                } elseif ($runtime === 'running') {
                    $label = 'Работает';
                    $badge = 'bg-green-100 text-green-700';
                } elseif ($runtime === 'restarting') {
                    $label = 'Перезапуск';
                    $badge = 'bg-amber-100 text-amber-700';
                } elseif ($runtime === 'offline' || $runtime === 'stopped') {
                    $label = 'Выключен';
                    $badge = 'bg-gray-100 text-gray-700';
                } elseif ($runtime === 'missing') {
                    $label = 'Не установлен';
                    $badge = 'bg-rose-100 text-rose-700';
                }

                $isReinstalling = $prov === 'reinstalling';
                $isInstallFailed = $prov === 'failed';

                $isProvisioning = in_array($prov, ['pending', 'installing', 'reinstalling'], true);
                $showRestart = ! $isProvisioning && $runtime === 'running';
                $showStop = ! $isProvisioning && $runtime === 'running';
                $showStart = ! $isProvisioning && in_array($runtime, ['offline', 'stopped'], true);
                $showReinstall = ! $isProvisioning && (in_array($runtime, ['offline', 'stopped', 'missing'], true) || $isInstallFailed);

                if ($isInstallFailed) {
                    $showRestart = false;
                    $showStop = false;
                    $showStart = false;
                }

                if ($isExpired) {
                    $showRestart = false;
                    $showStop = false;
                    $showStart = false;
                    $showReinstall = false;
                }
            @endphp

            @if (session('success'))
                <div class="mb-4 rounded-md border border-emerald-500/20 bg-emerald-500/10 px-4 py-3 text-xs text-emerald-200">
                    {{ session('success') }}
                </div>
            @endif
            @if (session('error'))
                <div class="mb-4 rounded-md border border-rose-500/20 bg-rose-500/10 px-4 py-3 text-xs text-rose-200">
                    {{ session('error') }}
                </div>
            @endif

            <div class="rounded-2xl bg-white text-slate-900 shadow-sm overflow-hidden">
                <div class="bg-slate-50">
                    @include('server.partials.tabs')
                </div>
            </div>

            @include('server.partials.main')
        </div>
    </section>

    @if (! $isExpired)
        @include('server.partials.scripts-main')
    @endif
@endsection
