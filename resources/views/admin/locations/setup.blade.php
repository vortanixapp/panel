@extends('layouts.app-admin')

@section('page_title', 'Настройка локации: ' . $location->name)

@push('styles')
<style>
    .setup-item {
        transition: all 0.2s;
    }
    .setup-item:hover {
        transform: translateY(-1px);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    }
    .log-output {
        max-height: 600px;
        overflow-y: auto;
        font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
        font-size: 12px;
        background: rgba(0, 0, 0, 0.20);
        border: 1px solid rgba(255, 255, 255, 0.10);
        border-radius: 6px;
        padding: 8px;
        white-space: pre-wrap;
        color: rgba(226, 232, 240, 0.9);
    }
    .modal-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.5);
        z-index: 50;
    }
    .modal {
        position: fixed;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        background: #242f3d;
        border-radius: 8px;
        padding: 20px;
        max-width: 1200px;
        width: 90%;
        z-index: 51;
        box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
        border: 1px solid rgba(255, 255, 255, 0.10);
    }
</style>
@endpush

@section('content')
    <section class="px-4 py-6 md:py-8" x-data="setupComponent()">
        <div class="w-full max-w-none space-y-6">
            <div class="flex items-center justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Настройка локации</h1>
                    <p class="mt-1 text-sm text-slate-300/80">{{ $location->name }} ({{ $location->code }}) — {{ $location->ssh_host }}</p>
                </div>
                <a
                    href="{{ route('admin.locations.show', $location) }}"
                    class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                >
                    ← Назад к локации
                </a>
            </div>

            <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="flex items-center justify-between gap-3">
                        <h3 class="text-base font-semibold text-slate-100">Данные подключения</h3>
                        <span class="text-[11px] text-slate-300/70">Сохраняются после установки шагов</span>
                    </div>
                    <div class="mt-4 space-y-3 text-sm">
                        <div class="flex items-center justify-between gap-3 rounded-lg border border-white/10 bg-black/10 px-3 py-2">
                            <div class="text-slate-200">MySQL host:port</div>
                            <div class="font-mono text-slate-100">
                                {{ $location->mysql_host ? ($location->mysql_host . ':' . ($location->mysql_port ?? 3306)) : '—' }}
                            </div>
                        </div>
                        <div class="flex items-center justify-between gap-3 rounded-lg border border-white/10 bg-black/10 px-3 py-2">
                            <div class="text-slate-200">phpMyAdmin</div>
                            <div class="font-mono text-slate-100">
                                @if($location->phpmyadmin_port)
                                    <a
                                        class="text-sky-200 hover:text-sky-100 underline"
                                        href="http://{{ $location->ssh_host }}:{{ $location->phpmyadmin_port }}"
                                        target="_blank"
                                        rel="noreferrer"
                                    >
                                        http://{{ $location->ssh_host }}:{{ $location->phpmyadmin_port }}
                                    </a>
                                @else
                                    —
                                @endif
                            </div>
                        </div>
                    </div>
                </div>

                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="flex items-center justify-between gap-3">
                        <h3 class="text-base font-semibold text-slate-100">Порядок установки</h3>
                    </div>
                    <div class="mt-4 text-sm text-slate-300/80 space-y-2">
                        <div class="rounded-lg border border-white/10 bg-black/10 px-3 py-2">1) Пакеты → 2) Docker → 3) MySQL → 4) phpMyAdmin → 5) FTP → 6) Daemon → 7) Images</div>
                        <div class="rounded-lg border border-white/10 bg-black/10 px-3 py-2">Если шаг упал — открывай лог, исправляй причину и жми «Повторить».</div>
                    </div>
                </div>

                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="flex items-center justify-between mb-3">
                        <h3 class="text-base font-semibold text-slate-100">Проверка конфигурации</h3>
                        @php
                            $failedChecks = collect($checks ?? [])->filter(fn ($c) => !($c['ok'] ?? false));
                        @endphp
                        @if($failedChecks->count() === 0)
                            <span class="text-xs px-2 py-1 rounded-full bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20">Готово</span>
                        @else
                            <span class="text-xs px-2 py-1 rounded-full bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20">Не настроено: {{ $failedChecks->count() }}</span>
                        @endif
                    </div>

                    <div class="space-y-2">
                        @foreach(($checks ?? []) as $check)
                            <div class="rounded-lg border border-white/10 bg-black/10 p-3">
                                <div class="flex items-center justify-between gap-3">
                                    <div class="text-sm font-medium text-slate-100">{{ $check['label'] ?? $check['key'] ?? 'check' }}</div>
                                    @if(($check['ok'] ?? false))
                                        <span class="text-xs px-2 py-1 rounded-full bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20">OK</span>
                                    @else
                                        <span class="text-xs px-2 py-1 rounded-full bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20">Не настроено</span>
                                    @endif
                                </div>

                                @if(!($check['ok'] ?? false) && !empty($check['hint'] ?? null))
                                    <div class="mt-2 text-xs text-slate-300/80">{{ $check['hint'] }}</div>
                                @endif
                            </div>
                        @endforeach
                    </div>
                </div>
            </div>

            <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
                <!-- Основные пакеты -->
                <div class="setup-item flex flex-col rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20" data-component="packages">
                    <div class="flex items-start justify-between gap-3 mb-3">
                        <h3 class="text-base font-semibold text-slate-100">Основные пакеты</h3>
                        <span id="packages-status" class="text-[10px] px-2 py-1 rounded-full
                            @if(($statuses['packages'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20
                            @elseif(($statuses['packages'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING) bg-sky-500/10 text-sky-200 ring-1 ring-sky-500/20
                            @elseif(($statuses['packages'] ?? null) === \App\Models\LocationSetupStatus::STATUS_FAILED) bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20
                            @else bg-black/10 text-slate-200 ring-1 ring-white/10
                            @endif">
                            @switch($statuses['packages'] ?? null)
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLED)
                                    Установлено
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLING)
                                    Устанавливается
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_FAILED)
                                    Ошибка
                                    @break
                                @default
                                    Ожидает
                            @endswitch
                        </span>
                    </div>
                    <p class="text-xs text-slate-300/80 mb-3">Установка curl, wget, git, htop, vim, nano, ufw, fail2ban</p>
                    <button
                        id="install-packages"
                        class="w-full inline-flex items-center justify-center rounded-md
                            @if(($statuses['packages'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-600 text-white cursor-not-allowed
                            @else bg-slate-900 text-white hover:bg-slate-800
                            @endif px-4 py-1.5 text-xs font-semibold"
                        @if(($statuses['packages'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) disabled
                        @else @click="installComponent('packages')"
                        @endif>
                        @if(($statuses['packages'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED)
                            Готово
                        @elseif(($statuses['packages'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING)
                            Устанавливается...
                        @else
                            Установить пакеты
                        @endif
                    </button>
                    @if(($statuses['packages'] ?? null) === \App\Models\LocationSetupStatus::STATUS_FAILED)
                        <div class="mt-3 rounded-lg border border-rose-500/20 bg-rose-500/10 px-3 py-2 text-xs text-rose-100">Открой лог в модалке и проверь последнюю команду/ошибку.</div>
                    @endif
                </div>

                <!-- Docker -->
                <div class="setup-item flex flex-col rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20" data-component="docker">
                    <div class="flex items-start justify-between gap-3 mb-3">
                        <h3 class="text-base font-semibold text-slate-100">Docker</h3>
                        <span id="docker-status" class="text-[10px] px-2 py-1 rounded-full
                            @if(($statuses['docker'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20
                            @elseif(($statuses['docker'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING) bg-sky-500/10 text-sky-200 ring-1 ring-sky-500/20
                            @elseif(($statuses['docker'] ?? null) === \App\Models\LocationSetupStatus::STATUS_FAILED) bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20
                            @else bg-black/10 text-slate-200 ring-1 ring-white/10
                            @endif">
                            @switch($statuses['docker'] ?? null)
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLED)
                                    Установлено
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLING)
                                    Устанавливается
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_FAILED)
                                    Ошибка
                                    @break
                                @default
                                    Ожидает
                            @endswitch
                        </span>
                    </div>
                    <p class="text-xs text-slate-300/80 mb-3">Установка Docker Engine и запуск сервиса</p>
                    <button
                        id="install-docker"
                        class="w-full inline-flex items-center justify-center rounded-md
                            @if(($statuses['docker'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-600 text-white cursor-not-allowed
                            @else bg-slate-900 text-white hover:bg-slate-800
                            @endif px-4 py-1.5 text-xs font-semibold"
                        @if(($statuses['docker'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) disabled
                        @else @click="installComponent('docker')"
                        @endif>
                        @if(($statuses['docker'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED)
                            Готово
                        @elseif(($statuses['docker'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING)
                            Устанавливается...
                        @else
                            Установить Docker
                        @endif
                    </button>
                    @if(($statuses['docker'] ?? null) === \App\Models\LocationSetupStatus::STATUS_FAILED)
                        <div class="mt-3 rounded-lg border border-rose-500/20 bg-rose-500/10 px-3 py-2 text-xs text-rose-100">Если Docker уже установлен — проверь сервис: <span class="font-mono">systemctl status docker</span>.</div>
                    @endif
                </div>

                <!-- MySQL -->
                <div class="setup-item flex flex-col rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20" data-component="mysql">
                    <div class="flex items-start justify-between gap-3 mb-3">
                        <h3 class="text-base font-semibold text-slate-100">MySQL</h3>
                        <span id="mysql-status" class="text-[10px] px-2 py-1 rounded-full
                            @if(($statuses['mysql'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20
                            @elseif(($statuses['mysql'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING) bg-sky-500/10 text-sky-200 ring-1 ring-sky-500/20
                            @elseif(($statuses['mysql'] ?? null) === \App\Models\LocationSetupStatus::STATUS_FAILED) bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20
                            @else bg-black/10 text-slate-200 ring-1 ring-white/10
                            @endif">
                            @switch($statuses['mysql'] ?? null)
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLED)
                                    Установлено
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLING)
                                    Устанавливается
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_FAILED)
                                    Ошибка
                                    @break
                                @default
                                    Ожидает
                            @endswitch
                        </span>
                    </div>
                    <p class="text-xs text-slate-300/80 mb-3">Установка MySQL сервера с базовой настройкой безопасности</p>
                    <button
                        id="install-mysql"
                        class="w-full inline-flex items-center justify-center rounded-md
                            @if(($statuses['mysql'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-600 text-white cursor-not-allowed
                            @else bg-slate-900 text-white hover:bg-slate-800
                            @endif px-4 py-1.5 text-xs font-semibold"
                        @if(($statuses['mysql'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) disabled
                        @else @click="installComponent('mysql')"
                        @endif>
                        @if(($statuses['mysql'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED)
                            Готово
                        @elseif(($statuses['mysql'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING)
                            Устанавливается...
                        @else
                            Установить MySQL
                        @endif
                    </button>
                    @if(($statuses['mysql'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED && $location->mysql_host)
                        <div class="mt-3 rounded-lg border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-200">
                            Host: <span class="font-mono text-slate-100">{{ $location->mysql_host }}:{{ $location->mysql_port ?? 3306 }}</span>
                        </div>
                    @endif
                </div>

                <!-- phpMyAdmin -->
                <div class="setup-item flex flex-col rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20" data-component="phpmyadmin">
                    <div class="flex items-start justify-between gap-3 mb-3">
                        <h3 class="text-base font-semibold text-slate-100">phpMyAdmin</h3>
                        <span id="phpmyadmin-status" class="text-[10px] px-2 py-1 rounded-full
                            @if(($statuses['phpmyadmin'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20
                            @elseif(($statuses['phpmyadmin'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING) bg-sky-500/10 text-sky-200 ring-1 ring-sky-500/20
                            @elseif(($statuses['phpmyadmin'] ?? null) === \App\Models\LocationSetupStatus::STATUS_FAILED) bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20
                            @else bg-black/10 text-slate-200 ring-1 ring-white/10
                            @endif">
                            @switch($statuses['phpmyadmin'] ?? null)
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLED)
                                    Установлено
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLING)
                                    Устанавливается
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_FAILED)
                                    Ошибка
                                    @break
                                @default
                                    Ожидает
                            @endswitch
                        </span>
                    </div>
                    <p class="text-xs text-slate-300/80 mb-3">Установка phpMyAdmin в Docker и открытие порта 8081</p>
                    <button
                        id="install-phpmyadmin"
                        class="w-full inline-flex items-center justify-center rounded-md
                            @if(($statuses['phpmyadmin'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-600 text-white cursor-not-allowed
                            @else bg-slate-900 text-white hover:bg-slate-800
                            @endif px-4 py-1.5 text-xs font-semibold"
                        @if(($statuses['phpmyadmin'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) disabled
                        @else @click="installComponent('phpmyadmin')"
                        @endif>
                        @if(($statuses['phpmyadmin'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED)
                            Готово
                        @elseif(($statuses['phpmyadmin'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING)
                            Устанавливается...
                        @else
                            Установить phpMyAdmin
                        @endif
                    </button>
                    @if(($statuses['phpmyadmin'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED && $location->phpmyadmin_port)
                        <div class="mt-3 rounded-lg border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-200">
                            URL:
                            <a
                                class="font-mono text-sky-200 hover:text-sky-100 underline"
                                href="http://{{ $location->ssh_host }}:{{ $location->phpmyadmin_port }}"
                                target="_blank"
                                rel="noreferrer"
                            >
                                http://{{ $location->ssh_host }}:{{ $location->phpmyadmin_port }}
                            </a>
                        </div>
                    @endif
                </div>

                <!-- FTP -->
                <div class="setup-item flex flex-col rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20" data-component="ftp">
                    <div class="flex items-start justify-between gap-3 mb-3">
                        <h3 class="text-base font-semibold text-slate-100">FTP</h3>
                        <span id="ftp-status" class="text-[10px] px-2 py-1 rounded-full
                            @if(($statuses['ftp'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20
                            @elseif(($statuses['ftp'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING) bg-sky-500/10 text-sky-200 ring-1 ring-sky-500/20
                            @elseif(($statuses['ftp'] ?? null) === \App\Models\LocationSetupStatus::STATUS_FAILED) bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20
                            @else bg-black/10 text-slate-200 ring-1 ring-white/10
                            @endif">
                            @switch($statuses['ftp'] ?? null)
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLED)
                                    Установлено
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLING)
                                    Устанавливается
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_FAILED)
                                    Ошибка
                                    @break
                                @default
                                    Ожидает
                            @endswitch
                        </span>
                    </div>
                    <p class="text-xs text-slate-300/80 mb-3">Установка vsftpd и открытие порта 21 в firewall</p>
                    <button
                        id="install-ftp"
                        class="w-full inline-flex items-center justify-center rounded-md
                            @if(($statuses['ftp'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-600 text-white cursor-not-allowed
                            @else bg-slate-900 text-white hover:bg-slate-800
                            @endif px-4 py-1.5 text-xs font-semibold"
                        @if(($statuses['ftp'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) disabled
                        @else @click="installComponent('ftp')"
                        @endif>
                        @if(($statuses['ftp'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED)
                            Готово
                        @elseif(($statuses['ftp'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING)
                            Устанавливается...
                        @else
                            Установить FTP
                        @endif
                    </button>
                    <div class="mt-3 rounded-lg border border-white/10 bg-black/10 px-3 py-2 text-[11px] text-slate-300/80">Пользователи FTP создаются через daemon при создании сервера.</div>
                </div>

                <!-- Vortanix Daemon -->
                <div class="setup-item rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20" data-component="daemon">
                    <div class="flex items-start justify-between gap-3 mb-3">
                        <h3 class="text-base font-semibold text-slate-100">Vortanix Daemon</h3>
                        <span id="daemon-status" class="text-[10px] px-2 py-1 rounded-full
                            @if(($statuses['daemon'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20
                            @elseif(($statuses['daemon'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING) bg-sky-500/10 text-sky-200 ring-1 ring-sky-500/20
                            @elseif(($statuses['daemon'] ?? null) === \App\Models\LocationSetupStatus::STATUS_FAILED) bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20
                            @else bg-black/10 text-slate-200 ring-1 ring-white/10
                            @endif">
                            @switch($statuses['daemon'] ?? null)
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLED)
                                    Установлено
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLING)
                                    Устанавливается
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_FAILED)
                                    Ошибка
                                    @break
                                @default
                                    Ожидает
                            @endswitch
                        </span>
                    </div>
                    <p class="text-xs text-slate-300/80 mb-3">Установка Python, aiohttp, скачивание и настройка демона мониторинга, создание systemd сервиса</p>
                    <button
                        id="install-daemon"
                        class="w-full inline-flex items-center justify-center rounded-md
                            @if(($statuses['daemon'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-600 text-white cursor-not-allowed
                            @else bg-emerald-600 text-white hover:bg-emerald-500
                            @endif px-4 py-1.5 text-xs font-semibold"
                        @if(($statuses['daemon'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) disabled
                        @else @click="installComponent('daemon')"
                        @endif>
                        @if(($statuses['daemon'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED)
                            Готово
                        @elseif(($statuses['daemon'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING)
                            Устанавливается...
                        @else
                            Установить Vortanix Daemon
                        @endif
                    </button>
                </div>

                <!-- Сборка игровых образов -->
                <div class="setup-item rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20" data-component="images">
                    <div class="flex items-start justify-between gap-3 mb-3">
                        <h3 class="text-base font-semibold text-slate-100">Сборка игровых образов</h3>
                        <span id="images-status" class="text-[10px] px-2 py-1 rounded-full
                            @if(($statuses['images'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20
                            @elseif(($statuses['images'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING) bg-sky-500/10 text-sky-200 ring-1 ring-sky-500/20
                            @elseif(($statuses['images'] ?? null) === \App\Models\LocationSetupStatus::STATUS_FAILED) bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20
                            @else bg-black/10 text-slate-200 ring-1 ring-white/10
                            @endif">
                            @switch($statuses['images'] ?? null)
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLED)
                                    Собрано
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_INSTALLING)
                                    Собирается
                                    @break
                                @case(\App\Models\LocationSetupStatus::STATUS_FAILED)
                                    Ошибка
                                    @break
                                @default
                                    Ожидает
                            @endswitch
                        </span>
                    </div>
                    <p class="text-xs text-slate-300/80 mb-3">Запуск docker build для всех игровых runtime-образов на ноде</p>
                    <button
                        id="install-images"
                        class="w-full inline-flex items-center justify-center rounded-md
                            @if(($statuses['images'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) bg-emerald-600 text-white cursor-not-allowed
                            @else bg-slate-900 text-white hover:bg-slate-800
                            @endif px-4 py-1.5 text-xs font-semibold"
                        @if(($statuses['images'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED) disabled
                        @else @click="installComponent('images')"
                        @endif>
                        @if(($statuses['images'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLED)
                            Готово
                        @elseif(($statuses['images'] ?? null) === \App\Models\LocationSetupStatus::STATUS_INSTALLING)
                            Собирается...
                        @else
                            Собрать образы
                        @endif
                    </button>
                </div>
            </div>
        </div>

        <!-- Modal -->
        <div x-show="showModal" class="modal-backdrop" x-transition @click.self="showModal = false" @keydown.escape.window="showModal = false">
            <div class="modal" style="max-width: 1280px; width: min(1280px, calc(100% - 2rem)); padding: 0; overflow: hidden;">
                <div class="flex items-center justify-between gap-3 border-b border-white/10 bg-black/10 px-5 py-4">
                    <div>
                        <h2 class="text-base md:text-lg font-semibold text-slate-100" x-text="currentComponent ? 'Установка: ' + currentComponent : 'Консоль'"></h2>
                        <div class="mt-0.5 text-xs text-slate-300/70">Логи обновляются автоматически</div>
                    </div>
                    <div class="flex items-center gap-2">
                        <button
                            type="button"
                            class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                            @click="navigator.clipboard.writeText(logs[currentComponent] || '')"
                        >
                            Копировать лог
                        </button>
                        <button @click="showModal = false" class="text-slate-300/70 hover:text-white p-2">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                            </svg>
                        </button>
                    </div>
                </div>
                <div class="p-5">
                    <div class="log-output" style="max-height: 70vh;" x-text="logs[currentComponent] || 'Ожидание...'"></div>
                </div>
            </div>
        </div>
    </section>
@endsection

@push('scripts')
<script>
function setupComponent() {
    return {
        showModal: false,
        currentComponent: '',
        logs: {},

        installComponent(component) {
            const button = document.getElementById(`install-${component}`);
            const status = document.getElementById(`${component}-status`);

            button.disabled = true;
            button.textContent = 'Установка...';
            status.textContent = 'Выполняется';
            status.className = 'text-xs px-2 py-1 rounded-full bg-sky-500/10 text-sky-200 ring-1 ring-sky-500/20';

            // Открыть модальное окно
            this.showModal = true;
            this.currentComponent = component;
            this.logs = { [component]: 'Запуск установки...\n' };

            // Начать polling статуса
            const statusUrl = `/admin/locations/{{ $location->id }}/setup/status`;
            let lastLog = '';
            const statusInterval = setInterval(() => {
                fetch(statusUrl)
                .then(response => {
                    if (!response.ok) {
                        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                    }
                    return response.json();
                })
                .then(data => {
                    if (data.log && data.log !== lastLog) {
                        this.logs[component] = data.log;
                        lastLog = data.log;
                    }

                    if (data.completed) {
                        clearInterval(statusInterval);

                        const logText = data.log || '';
                        const isError = logText.includes('❌ Ошибка');

                        if (isError) {
                            status.textContent = 'Ошибка';
                            status.className = 'text-xs px-2 py-1 rounded-full bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20';
                            button.textContent = 'Повторить';
                            button.disabled = false;
                        } else {
                            status.textContent = 'Установлено';
                            status.className = 'text-xs px-2 py-1 rounded-full bg-emerald-500/10 text-emerald-200 ring-1 ring-emerald-500/20';
                            button.textContent = 'Готово';
                            button.className = 'w-full inline-flex items-center justify-center rounded-md bg-emerald-600 px-4 py-2 text-sm font-semibold text-white shadow-sm cursor-not-allowed';
                        }
                    }
                })
                .catch(error => {
                    console.error('Status polling error:', error);
                    this.logs[component] += `\n❌ Polling error: ${error.message}`;
                });
            }, 1500);

            const postUrl = `/admin/locations/{{ $location->id }}/setup/${component}`;
            fetch(postUrl, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-CSRF-TOKEN': document.querySelector('meta[name="csrf-token"]').getAttribute('content')
                },
                body: JSON.stringify({})
            })
            .then(response => response.json())
            .then(data => {
                // Если контроллер вернул ошибку до запуска job
                if (!data.success) {
                    clearInterval(statusInterval);
                    status.textContent = 'Ошибка';
                    status.className = 'text-xs px-2 py-1 rounded-full bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20';
                    button.textContent = 'Повторить';
                    button.disabled = false;
                    this.logs[component] += '\n❌ Ошибка: ' + (data.error || 'Неизвестная ошибка');
                } else if (data.message) {
                    // Просто дописываем служебное сообщение, не меняя статус
                    this.logs[component] += '\n' + data.message;
                }
            })
            .catch(error => {
                clearInterval(statusInterval);
                status.textContent = 'Ошибка';
                status.className = 'text-xs px-2 py-1 rounded-full bg-rose-500/10 text-rose-200 ring-1 ring-rose-500/20';
                button.textContent = 'Повторить';
                button.disabled = false;
                this.logs[component] += '\n❌ Ошибка сети: ' + error.message;
            });
        }
    };
}
</script>
@endpush
