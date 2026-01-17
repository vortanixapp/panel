@extends('layouts.app-admin')

@section('page_title', 'Лицензия')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div>
                <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Лицензия</h1>
                <p class="mt-1 text-sm text-slate-300/80">Управление лицензией панели и диагностика подключения к облаку.</p>
            </div>

            @if(session('success'))
                <div class="rounded-2xl border border-emerald-500/20 bg-emerald-500/10 p-4 text-sm text-emerald-200">
                    {{ session('success') }}
                </div>
            @endif
            @if(session('error'))
                <div class="rounded-2xl border border-rose-500/20 bg-rose-500/10 p-4 text-sm text-rose-200">
                    {{ session('error') }}
                </div>
            @endif
            @if($errors->any())
                <div class="rounded-2xl border border-rose-500/20 bg-rose-500/10 p-4 text-sm text-rose-200">
                    {{ $errors->first() }}
                </div>
            @endif

            <div class="grid gap-4 md:grid-cols-2">
                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="text-sm font-semibold text-slate-100">Текущая лицензия</div>
                    <div class="mt-3">
                        <div class="text-xs font-semibold text-slate-300/70">license.key</div>
                        <div class="mt-1 font-mono text-xs text-slate-100 break-all">{{ $licenseKey !== '' ? $licenseKey : '—' }}</div>
                    </div>

                    <form method="POST" action="{{ route('admin.license.store') }}" class="mt-4 space-y-3">
                        @csrf
                        <div>
                            <div class="text-xs font-semibold text-slate-300/70">Новый ключ</div>
                            <input
                                name="license_key"
                                placeholder="LICENSE-KEY"
                                class="mt-2 w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-white/20"
                                required
                            />
                        </div>
                        <div class="flex flex-wrap items-center gap-2">
                            <button type="submit" class="inline-flex items-center rounded-xl bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800 transition">
                                Сохранить
                            </button>
                        </div>
                    </form>

                    <form method="POST" action="{{ route('admin.license.clear') }}" class="mt-3">
                        @csrf
                        <button type="submit" class="inline-flex items-center rounded-xl border border-white/10 bg-black/10 px-4 py-2 text-sm font-semibold text-slate-200 hover:bg-black/15 hover:text-white transition">
                            Удалить ключ
                        </button>
                    </form>
                </div>

                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="text-sm font-semibold text-slate-100">Диагностика</div>

                    <div class="mt-4 rounded-xl border border-white/10 bg-black/10 p-3 text-sm">
                        <div class="grid gap-2">
                            <div class="flex items-center justify-between gap-4">
                                <span class="text-slate-300/80">Panel ID</span>
                                <span class="font-mono text-xs text-slate-100">{{ $panelId !== '' ? $panelId : '—' }}</span>
                            </div>
                            <div class="flex items-center justify-between gap-4">
                                <span class="text-slate-300/80">Server IP</span>
                                <span class="font-mono text-xs text-slate-100">{{ $serverIp !== '' ? $serverIp : '—' }}</span>
                            </div>
                            <div class="flex items-center justify-between gap-4">
                                <span class="text-slate-300/80">Последний heartbeat</span>
                                <span class="font-mono text-xs text-slate-100">{{ $hbCheckedAt > 0 ? \Carbon\Carbon::createFromTimestamp($hbCheckedAt)->toDateTimeString() : '—' }}</span>
                            </div>
                            <div class="flex items-center justify-between gap-4">
                                <span class="text-slate-300/80">Последний OK</span>
                                <span class="font-mono text-xs text-slate-100">{{ $hbLastOkAt > 0 ? \Carbon\Carbon::createFromTimestamp($hbLastOkAt)->toDateTimeString() : '—' }}</span>
                            </div>
                            <div class="flex items-center justify-between gap-4">
                                <span class="text-slate-300/80">Статус</span>
                                <span class="text-xs font-semibold {{ $hbValid ? 'text-emerald-300' : 'text-rose-200' }}">{{ $hbValid ? 'OK' : 'NO' }}</span>
                            </div>
                        </div>
                    </div>

                    @if(!empty($hbError))
                        <div class="mt-3 rounded-xl border border-rose-500/20 bg-rose-500/10 p-3 text-xs text-rose-200">
                            {{ $hbError }}
                        </div>
                    @endif

                    <div class="mt-3 text-[11px] text-slate-300/70">
                        Данные обновляются в реальном времени при каждом открытии страницы.
                    </div>
                </div>
            </div>
        </div>
    </section>
@endsection
