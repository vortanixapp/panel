@extends('layouts.app-admin')

@section('page_title', 'Vortanix Daemon: ' . $location->name)

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex items-center justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Vortanix Daemon</h1>
                    <p class="mt-1 text-sm text-slate-300/80">{{ $location->name }} ({{ $location->code }}) — {{ $location->ssh_host }}</p>
                </div>
                <a
                    href="{{ route('admin.locations.show', $location) }}"
                    class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                >
                    ← Назад к локации
                </a>
            </div>

            @if($daemon)
                <div class="grid gap-4 md:grid-cols-2">
                    <!-- Статус -->
                    <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-6 shadow-sm shadow-black/20">
                        <h2 class="text-lg font-semibold text-slate-100 mb-4">Статус</h2>
                        <div class="space-y-3">
                            <div class="flex items-center justify-between">
                                <span class="text-sm text-slate-300/80">Состояние</span>
                                <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium
                                    @if($daemon->status === \App\Models\LocationDaemon::STATUS_ONLINE) bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20
                                    @elseif($daemon->status === \App\Models\LocationDaemon::STATUS_OFFLINE) bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20
                                    @else bg-black/10 text-slate-200 ring-1 ring-white/10
                                    @endif">
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
                            <div class="flex items-center justify-between">
                                <span class="text-sm text-slate-300/80">Последний контакт</span>
                                <span class="text-sm text-slate-100">{{ $daemon->last_seen ? $daemon->last_seen->diffForHumans() : '—' }}</span>
                            </div>
                        </div>
                    </div>

                    <!-- Информация о системе -->
                    <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-6 shadow-sm shadow-black/20">
                        <h2 class="text-lg font-semibold text-slate-100 mb-4">Система</h2>
                        <div class="space-y-3">
                            <div class="flex items-center justify-between">
                                <span class="text-sm text-slate-300/80">Платформа</span>
                                <span class="text-sm text-slate-100">{{ $daemon->platform ?: '—' }}</span>
                            </div>
                            <div class="flex items-center justify-between">
                                <span class="text-sm text-slate-300/80">Версия</span>
                                <span class="text-sm text-slate-100">{{ $daemon->version ?: '—' }}</span>
                            </div>
                            <div class="flex items-center justify-between">
                                <span class="text-sm text-slate-300/80">PID</span>
                                <span class="text-sm text-slate-100">{{ $daemon->pid ?: '—' }}</span>
                            </div>
                            <div class="flex items-center justify-between">
                                <span class="text-sm text-slate-300/80">Время работы</span>
                                <span class="text-sm text-slate-100">{{ $daemon->uptime_sec ? round($daemon->uptime_sec / 3600, 1) . ' ч' : '—' }}</span>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Управление -->
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-6 shadow-sm shadow-black/20">
                    <h2 class="text-lg font-semibold text-slate-100 mb-4">Управление</h2>
                    <div class="flex gap-3">
                        <button
                            onclick="refreshDaemonInfo()"
                            class="inline-flex items-center rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-blue-500"
                        >
                            Обновить данные
                        </button>
                        <button
                            onclick="restartDaemon()"
                            class="inline-flex items-center rounded-md bg-amber-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-amber-500"
                        >
                            Перезапустить
                        </button>
                    </div>
                </div>
            @else
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-6 shadow-sm shadow-black/20 text-center">
                    <div class="text-slate-300/80">
                        <svg class="mx-auto h-12 w-12 text-slate-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"></path>
                        </svg>
                        <h3 class="text-lg font-medium text-slate-100 mb-2">Демон не настроен</h3>
                        <p class="text-sm mb-4">Vortanix Daemon ещё не установлен на этой локации.</p>
                        <a
                            href="{{ route('admin.locations.setup', $location) }}"
                            class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-slate-800"
                        >
                            Перейти к настройке
                        </a>
                    </div>
                </div>
            @endif
        </div>
    </section>
@endsection

@push('scripts')
<script>
function refreshDaemonInfo() {
    fetch(`/admin/locations/{{ $location->id }}/daemon/refresh`, {
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

function restartDaemon() {
    if (!confirm('Перезапустить Vortanix Daemon?')) return;

    fetch(`/admin/locations/{{ $location->id }}/daemon/restart`, {
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
