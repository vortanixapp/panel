@php
    $isReinstalling = strtolower((string) ($server->provisioning_status ?? '')) === 'reinstalling';
    $isInstallFailed = strtolower((string) ($server->provisioning_status ?? '')) === 'failed';
    $isProvisioning = in_array(strtolower((string) ($server->provisioning_status ?? '')), ['pending', 'installing', 'reinstalling'], true);
    $isExpired = $server->expires_at && now()->greaterThan($server->expires_at);
    $gameCode = strtolower((string) ($server->game->code ?? $server->game->slug ?? ''));
    $isCs16 = in_array($gameCode, ['cs16', 'cstrike', 'counter-strike', 'counter_strike', 'cs_1_6'], true);
    $authUser = auth()->user();
    $isOwner = ($authUser && (bool) ($authUser->is_admin ?? false)) || ((int) ($server->user_id ?? 0) === (int) (auth()->id() ?? 0));
    $perm = null;
    if (! $isOwner) {
        $perm = \App\Models\ServerUserPermission::query()
            ->where('server_id', (int) $server->id)
            ->where('user_id', (int) (auth()->id() ?? 0))
            ->first();
    }
    $linkCls = 'flex flex-1 items-center justify-center gap-2';
    $activeCls = 'text-white font-semibold hover:text-white';
    $idleCls = 'text-slate-200 hover:text-white';
    $disabledCls = 'text-slate-400 opacity-60 cursor-not-allowed pointer-events-none';
    $baseUrl = (string) ($serverBaseUrl ?? route('server.show', $server));
@endphp

<nav class="w-full gap-6 px-4 py-3 text-[12px] flex flex-wrap bg-[#242f3d]">
    <a href="{{ $baseUrl }}" class="flex items-center gap-2 {{ ($tab === 'main') ? $activeCls : $idleCls }}">
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0h6"/></svg>
        <span>Основное</span>
    </a>
    <a href="{{ $baseUrl }}?tab=console" class="{{ $linkCls }} {{ ($tab === 'console') ? $activeCls : $idleCls }} {{ ($isProvisioning || $isInstallFailed || $isExpired || (! $isOwner && ! ($perm && $perm->can_view_console))) ? $disabledCls : '' }}">
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 4h14a2 2 0 012 2v12a2 2 0 01-2 2H5a2 2 0 01-2-2V6a2 2 0 012-2z"/></svg>
        <span>Консоль</span>
    </a>
    <a href="{{ $baseUrl }}?tab=logs" class="{{ $linkCls }} {{ ($tab === 'logs') ? $activeCls : $idleCls }} {{ ($isInstallFailed || $isExpired || (! $isOwner && ! ($perm && $perm->can_view_logs))) ? $disabledCls : '' }}">
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17v-2a4 4 0 118 0v2"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h10M7 11h10M7 15h6"/></svg>
        <span>Логи</span>
    </a>
    <a href="{{ $baseUrl }}?tab=metrics" class="{{ $linkCls }} {{ ($tab === 'metrics') ? $activeCls : $idleCls }} {{ ($isProvisioning || $isInstallFailed || $isExpired || (! $isOwner && ! ($perm && $perm->can_view_metrics))) ? $disabledCls : '' }}">
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 19V5"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 17l3-3 3 2 6-6"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 11v6H4"/></svg>
        <span>Метрики</span>
    </a>
    <a href="{{ $baseUrl }}?tab=ftp" class="{{ $linkCls }} {{ ($tab === 'ftp') ? $activeCls : $idleCls }} {{ ($isProvisioning || $isInstallFailed || $isExpired || (! $isOwner && ! ($perm && $perm->can_view_ftp))) ? $disabledCls : '' }}">
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16h8M8 12h8m-7 8h6a2 2 0 002-2V6a2 2 0 00-2-2H9a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
        <span>FTP</span>
    </a>
    <a href="{{ $baseUrl }}?tab=mysql" class="{{ $linkCls }} {{ ($tab === 'mysql') ? $activeCls : $idleCls }} {{ ($isProvisioning || $isInstallFailed || $isExpired || (! $isOwner && ! ($perm && $perm->can_view_mysql))) ? $disabledCls : '' }}">
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7a4 4 0 014-4h8a4 4 0 014 4v1a3 3 0 01-3 3H7a3 3 0 01-3-3V7z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 14a4 4 0 004 4h8a4 4 0 004-4v-1a3 3 0 00-3-3H7a3 3 0 00-3 3v1z"/></svg>
        <span>MySQL</span>
    </a>
    <a href="{{ $baseUrl }}?tab=cron" class="{{ $linkCls }} {{ ($tab === 'cron') ? $activeCls : $idleCls }} {{ ($isProvisioning || $isInstallFailed || $isExpired || (! $isOwner && ! ($perm && $perm->can_view_cron))) ? $disabledCls : '' }}">
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 22a10 10 0 110-20 10 10 0 010 20z"/></svg>
        <span>Планировщик</span>
    </a>
    <a href="{{ $baseUrl }}?tab=firewall" class="{{ $linkCls }} {{ ($tab === 'firewall') ? $activeCls : $idleCls }} {{ ($isProvisioning || $isInstallFailed || $isExpired || (! $isOwner && ! ($perm && $perm->can_view_firewall))) ? $disabledCls : '' }}">
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 2l7 4v6c0 5-3 9-7 10-4-1-7-5-7-10V6l7-4z"/></svg>
        <span>Firewall</span>
    </a>
    <a href="{{ $baseUrl }}?tab=settings" class="{{ $linkCls }} {{ ($tab === 'settings') ? $activeCls : $idleCls }} {{ ($isProvisioning || $isInstallFailed || $isExpired || (! $isOwner && ! ($perm && $perm->can_view_settings))) ? $disabledCls : '' }}">
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>
        <span>Настройки</span>
    </a>
    @if ($isCs16)
        <a href="{{ $baseUrl }}?tab=maps" class="{{ $linkCls }} {{ ($tab === 'maps') ? $activeCls : $idleCls }} {{ ($isProvisioning || $isInstallFailed || $isExpired || (! $isOwner && ! ($perm && $perm->can_settings_edit))) ? $disabledCls : '' }}">
            <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 01.553-.894L9 2m0 18l6-3m-6 3V2m6 15l5.447-2.724A1 1 0 0021 13.382V2.618a1 1 0 00-.553-.894L15 0m0 17V0m0 0L9 2"/></svg>
            <span>Карты</span>
        </a>
    @endif
    <a href="{{ $baseUrl }}?tab=plugins" class="{{ $linkCls }} {{ ($tab === 'plugins') ? $activeCls : $idleCls }} {{ ($isProvisioning || $isInstallFailed || $isExpired) ? $disabledCls : '' }}">
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 20l9-5-9-5-9 5 9 5z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 12l9-5-9-5-9 5 9 5z"/></svg>
        <span>Плагины</span>
    </a>
    <a href="{{ $baseUrl }}?tab=friends" class="{{ $linkCls }} {{ ($tab === 'friends') ? $activeCls : $idleCls }} {{ ($isProvisioning || $isInstallFailed || $isExpired || (! $isOwner && ! ($perm && $perm->can_view_friends))) ? $disabledCls : '' }}">
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zM21 10a3 3 0 11-6 0 3 3 0 016 0zM3 10a3 3 0 116 0 3 3 0 01-6 0z"/></svg>
        <span>Друзья</span>
    </a>
</nav>

@if ($isProvisioning)
    <script>
        document.addEventListener('DOMContentLoaded', function () {
            const url = "{{ route('server.status', $server) }}";
            async function tick() {
                try {
                    const resp = await fetch(url, { headers: { 'Accept': 'application/json' } });
                    const data = await resp.json().catch(() => null);
                    if (!resp.ok || !data || data.ok !== true) {
                        return;
                    }
                    const prov = String(data.provisioning_status || '').toLowerCase();
                    if (prov !== 'pending' && prov !== 'installing' && prov !== 'reinstalling') {
                        window.location.reload();
                    }
                } catch (e) {
                }
            }
            tick();
            setInterval(tick, 3000);
        });
    </script>
@endif
