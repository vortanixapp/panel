@extends('layouts.app-admin')

@section('page_title', 'Баг-репорт')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div>
                <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Баг-репорт</h1>
                <p class="mt-1 text-sm text-slate-300/80">Отправьте описание проблемы в биллинг. Администратор сможет проставить статус и комментарий.</p>
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
                    <div class="text-sm font-semibold text-slate-100">Отправка</div>

                    <form method="POST" action="{{ route('admin.bug-report.store') }}" class="mt-4 space-y-3">
                        @csrf
                        <div>
                            <div class="text-xs font-semibold text-slate-300/70">Заголовок</div>
                            <input
                                name="title"
                                value="{{ old('title') }}"
                                placeholder="Коротко: что не работает"
                                class="mt-2 w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-white/20"
                                required
                            />
                        </div>

                        <div>
                            <div class="text-xs font-semibold text-slate-300/70">Описание</div>
                            <textarea
                                name="description"
                                rows="6"
                                placeholder="Опишите проблему подробно"
                                class="mt-2 w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-white/20"
                                required
                            >{{ old('description') }}</textarea>
                        </div>

                        <div>
                            <div class="text-xs font-semibold text-slate-300/70">Шаги воспроизведения (опционально)</div>
                            <textarea
                                name="steps"
                                rows="4"
                                placeholder="1) ...\n2) ..."
                                class="mt-2 w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-white/20"
                            >{{ old('steps') }}</textarea>
                        </div>

                        <div class="grid gap-3 md:grid-cols-2">
                            <div>
                                <div class="text-xs font-semibold text-slate-300/70">Ожидаемое (опционально)</div>
                                <textarea
                                    name="expected"
                                    rows="3"
                                    class="mt-2 w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-white/20"
                                >{{ old('expected') }}</textarea>
                            </div>
                            <div>
                                <div class="text-xs font-semibold text-slate-300/70">Фактическое (опционально)</div>
                                <textarea
                                    name="actual"
                                    rows="3"
                                    class="mt-2 w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-white/20"
                                >{{ old('actual') }}</textarea>
                            </div>
                        </div>

                        <div>
                            <div class="text-xs font-semibold text-slate-300/70">Серьёзность</div>
                            <select
                                name="severity"
                                class="mt-2 w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-white focus:outline-none focus:ring-2 focus:ring-white/20"
                            >
                                @php($sev = old('severity', 'normal'))
                                <option value="low" {{ $sev === 'low' ? 'selected' : '' }}>Низкая</option>
                                <option value="normal" {{ $sev === 'normal' ? 'selected' : '' }}>Обычная</option>
                                <option value="high" {{ $sev === 'high' ? 'selected' : '' }}>Высокая</option>
                                <option value="critical" {{ $sev === 'critical' ? 'selected' : '' }}>Критическая</option>
                            </select>
                        </div>

                        <div class="flex flex-wrap items-center gap-2">
                            <button type="submit" class="inline-flex items-center rounded-xl bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800 transition">
                                Отправить
                            </button>
                        </div>
                    </form>
                </div>

                <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                    <div class="text-sm font-semibold text-slate-100">Диагностика (авто)</div>

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
                                <span class="text-slate-300/80">Версия панели</span>
                                <span class="font-mono text-xs text-slate-100">{{ $appVersion !== '' ? $appVersion : '—' }}</span>
                            </div>
                            <div class="flex items-center justify-between gap-4">
                                <span class="text-slate-300/80">license.key</span>
                                <span class="font-mono text-xs text-slate-100 break-all">{{ $licenseKey !== '' ? $licenseKey : '—' }}</span>
                            </div>
                        </div>
                    </div>

                    <div class="mt-3 text-[11px] text-slate-300/70">
                        Эти данные добавляются автоматически для ускорения диагностики.
                    </div>
                </div>
            </div>

            <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                    <div>
                        <div class="text-sm font-semibold text-slate-100">Отправленные баг-репорты</div>
                        <div class="mt-1 text-[12px] text-slate-300/70">Список синхронизируется с биллингом.</div>
                    </div>
                </div>

                <div class="mt-4 overflow-hidden rounded-2xl border border-white/10 bg-black/10 text-sm">
                    <table class="min-w-full divide-y divide-white/10">
                        <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                            <tr>
                                <th class="px-4 py-3 text-left">ID</th>
                                <th class="px-4 py-3 text-left">Заголовок</th>
                                <th class="px-4 py-3 text-left">Серьёзность</th>
                                <th class="px-4 py-3 text-left">Статус</th>
                                <th class="px-4 py-3 text-left">Комментарий</th>
                                <th class="px-4 py-3 text-left">Создано</th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-white/10 text-[13px]">
                            @forelse (($reports ?? []) as $r)
                                <tr class="hover:bg-white/5">
                                    <td class="px-4 py-3 font-mono text-xs text-slate-300/80">#{{ (int) ($r['id'] ?? 0) }}</td>
                                    <td class="px-4 py-3 text-slate-100 font-medium">{{ (string) ($r['title'] ?? '—') }}</td>
                                    <td class="px-4 py-3 text-slate-200">{{ (string) ($r['severity'] ?? '') !== '' ? (string) ($r['severity'] ?? '') : '—' }}</td>
                                    <td class="px-4 py-3">
                                        @if((string) ($r['status'] ?? '') === 'resolved')
                                            <span class="inline-flex items-center rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-medium text-emerald-200 ring-1 ring-emerald-500/20">Решено</span>
                                        @elseif((string) ($r['status'] ?? '') === 'in_progress')
                                            <span class="inline-flex items-center rounded-full bg-sky-500/10 px-2 py-0.5 text-[11px] font-medium text-sky-200 ring-1 ring-sky-500/20">В процессе</span>
                                        @else
                                            <span class="inline-flex items-center rounded-full bg-black/10 px-2 py-0.5 text-[11px] font-medium text-slate-200 ring-1 ring-white/10">Новый</span>
                                        @endif
                                    </td>
                                    <td class="px-4 py-3 text-slate-300/80">{{ (is_string($r['admin_comment'] ?? null) && (string) ($r['admin_comment'] ?? '') !== '') ? (string) ($r['admin_comment'] ?? '') : '—' }}</td>
                                    <td class="px-4 py-3 text-slate-300/80">{{ (string) ($r['created_at'] ?? '') !== '' ? (string) ($r['created_at'] ?? '') : '—' }}</td>
                                </tr>
                            @empty
                                <tr>
                                    <td colspan="6" class="px-4 py-8 text-center text-slate-300/70">Репортов пока нет</td>
                                </tr>
                            @endforelse
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </section>
@endsection
