@extends('layouts.app-admin')

@section('page_title', 'Vortanix Daemons')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Vortanix Daemons</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Мониторинг и управление демонами Vortanix на всех локациях.</p>
                </div>
            </div>

            @if ($daemons->isEmpty())
                <div class="rounded-2xl border border-dashed border-white/10 bg-black/10 p-4 text-[13px] text-slate-300/80">
                    Демоны пока не обнаружены. Установите Vortanix Daemon на локациях через страницу настройки локации, чтобы они появились здесь.
                </div>
            @else
                <div class="grid gap-4 md:grid-cols-3 text-sm">
                    @foreach ($daemons as $daemon)
                        <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm">
                            <div class="flex items-start justify-between gap-2">
                                <div class="space-y-1">
                                    <p class="text-xs font-semibold text-slate-300/70">
                                        {{ $daemon->location->region ?? 'Локация' }}
                                    </p>
                                    <p class="text-sm font-semibold text-slate-100">
                                        {{ $daemon->location->name }} ({{ $daemon->location->code }})
                                    </p>
                                    <p class="text-[11px] text-slate-300/70">
                                        {{ $daemon->platform ?: 'Платформа неизвестна' }} • PID: {{ $daemon->pid ?: '—' }}
                                    </p>
                                    @if($daemon->uptime_sec)
                                        <p class="text-xs text-slate-300/80">Время работы: {{ round($daemon->uptime_sec / 3600, 1) }} ч</p>
                                    @endif
                                    <p class="text-xs text-slate-300/70">
                                        Последний контакт: {{ $daemon->last_seen ? $daemon->last_seen->diffForHumans() : '—' }}
                                    </p>
                                </div>
                                <div class="flex flex-col items-end gap-2">
                                    <div class="flex items-center gap-2 text-xs">
                                        <span class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px]
                                            @if($daemon->status === \App\Models\LocationDaemon::STATUS_ONLINE) bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20
                                            @elseif($daemon->status === \App\Models\LocationDaemon::STATUS_OFFLINE) bg-red-500/10 text-red-200 ring-1 ring-red-500/20
                                            @else bg-white/5 text-slate-200 ring-1 ring-white/10
                                            @endif">
                                            <span class="inline-block h-1.5 w-1.5 rounded-full
                                                @if($daemon->status === \App\Models\LocationDaemon::STATUS_ONLINE) bg-emerald-400
                                                @elseif($daemon->status === \App\Models\LocationDaemon::STATUS_OFFLINE) bg-red-400
                                                @else bg-slate-400
                                                @endif"></span>
                                            @switch($daemon->status)
                                                @case(\App\Models\LocationDaemon::STATUS_ONLINE)
                                                    Онлайн
                                                    @break
                                                @case(\App\Models\LocationDaemon::STATUS_OFFLINE)
                                                    Оффлайн
                                                    @break
                                                @default
                                                    Неизвестно
                                            @endswitch
                                        </span>
                                    </div>

                                    <div class="flex items-center gap-1">
                                        <a
                                            href="{{ route('admin.locations.daemon.show', $daemon->location) }}"
                                            class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-white/10 bg-black/10 text-slate-200 hover:bg-black/15 hover:text-white"
                                            title="Просмотреть демон"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                                <path d="M1 12s4-8 9-4 9 4 9 4-4 8-9 4-9-4-9-4Z" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                <circle cx="10" cy="8" r="2" stroke-width="1.4" />
                                            </svg>
                                            <span class="sr-only">Просмотреть</span>
                                        </a>

                                        <button
                                            onclick="refreshDaemon({{ $daemon->location->id }})"
                                            class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-sky-500/20 bg-sky-500/10 text-sky-200 hover:bg-sky-500/15 hover:text-white"
                                            title="Обновить состояние демона"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                                <path d="M4 4v4h4" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                                <path d="M4 8a6 6 0 1 1 1.76 4.24L4 14" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
                                            </svg>
                                            <span class="sr-only">Обновить</span>
                                        </button>

                                        <button
                                            onclick="restartDaemon({{ $daemon->location->id }})"
                                            class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-amber-500/20 bg-amber-500/10 text-amber-200 hover:bg-amber-500/15 hover:text-white"
                                            title="Перезапустить демон"
                                        >
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="none" stroke="currentColor" class="h-3.5 w-3.5">
                                                <path d="M10 3v7" stroke-width="1.4" stroke-linecap="round" />
                                                <path d="M6 5.5A5.5 5.5 0 1 0 14 5.5" stroke-width="1.4" stroke-linecap="round" />
                                            </svg>
                                            <span class="sr-only">Перезапустить</span>
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    @endforeach
                </div>
            @endif
        </div>
    </section>
@endsection

@push('scripts')
<script>
function refreshDaemon(locationId) {
    fetch(`/admin/locations/${locationId}/daemon/refresh`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-CSRF-TOKEN': document.querySelector('meta[name="csrf-token"]').getAttribute('content')
        }
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            location.reload();
        } else {
            alert('Ошибка обновления: ' + (data.error || 'Неизвестная ошибка'));
        }
    })
    .catch(error => {
        alert('Ошибка сети: ' + error.message);
    });
}

function restartDaemon(locationId) {
    if (!confirm('Перезапустить Vortanix Daemon на этой локации?')) return;

    fetch(`/admin/locations/${locationId}/daemon/restart`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-CSRF-TOKEN': document.querySelector('meta[name="csrf-token"]').getAttribute('content')
        }
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            alert('Демон перезапущен');
            location.reload();
        } else {
            alert('Ошибка перезапуска: ' + (data.error || 'Неизвестная ошибка'));
        }
    })
    .catch(error => {
        alert('Ошибка сети: ' + error.message);
    });
}
</script>
@endpush
