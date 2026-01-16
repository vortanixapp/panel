@extends('layouts.app-admin')

@section('page_title', 'Рассылка')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex items-start justify-between gap-3">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Кампания #{{ (int) $mailing->id }}</h1>
                    <div class="mt-1 text-sm text-slate-300/80">{{ $mailing->title }}</div>
                </div>
                <a href="{{ route('admin.mailings.index') }}" class="rounded-xl border border-white/10 bg-black/10 px-4 py-2 text-xs text-slate-100 hover:bg-black/15">Назад</a>
            </div>

            @if(session('success'))
                <div class="rounded-xl bg-emerald-500/10 px-4 py-3 text-xs text-emerald-200 ring-1 ring-emerald-500/20">{{ session('success') }}</div>
            @endif

            <div class="grid gap-6 xl:grid-cols-3">
                <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden xl:col-span-2">
                    <div class="border-b border-white/10 bg-black/10 px-5 py-4 flex items-center justify-between gap-3">
                        <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Настройки</div>
                        <span class="inline-flex rounded-full bg-black/20 px-2 py-0.5 text-[11px] font-semibold text-slate-200 ring-1 ring-white/10">{{ $mailing->status }}</span>
                    </div>

                    <div class="p-5">
                        <form method="POST" action="{{ route('admin.mailings.update', $mailing) }}" class="space-y-5">
                            @csrf
                            @method('PUT')

                            <div class="space-y-1">
                                <div class="text-xs text-slate-300/70">Название</div>
                                <input name="title" value="{{ old('title', $mailing->title) }}" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100" required>
                            </div>

                            <div class="grid gap-4 md:grid-cols-2">
                                @php
                                    $channels = old('channels', is_array($mailing->channels) ? $mailing->channels : []);
                                    $aud = old('aud', is_array($mailing->audience) ? $mailing->audience : []);
                                @endphp
                                <div class="space-y-1">
                                    <div class="text-xs text-slate-300/70">Каналы</div>
                                    <div class="space-y-2 text-xs text-slate-200">
                                        <label class="flex items-center gap-2"><input type="checkbox" name="channels[]" value="email" {{ in_array('email', $channels) ? 'checked' : '' }}> Email</label>
                                        <label class="flex items-center gap-2"><input type="checkbox" name="channels[]" value="internal" {{ in_array('internal', $channels) ? 'checked' : '' }}> Внутренние уведомления</label>
                                        <label class="flex items-center gap-2"><input type="checkbox" name="channels[]" value="telegram" {{ in_array('telegram', $channels) ? 'checked' : '' }}> Telegram</label>
                                        <label class="flex items-center gap-2"><input type="checkbox" name="channels[]" value="sms" {{ in_array('sms', $channels) ? 'checked' : '' }}> SMS</label>
                                    </div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-xs text-slate-300/70">HTML</div>
                                    <label class="flex items-center gap-2 text-xs text-slate-200">
                                        <input type="checkbox" name="is_html" value="1" {{ old('is_html', $mailing->is_html) ? 'checked' : '' }}>
                                        <span>HTML</span>
                                    </label>

                                    <div class="mt-3 text-xs text-slate-300/70">План (опционально)</div>
                                    <input name="scheduled_at" value="{{ old('scheduled_at', $mailing->scheduled_at?->format('Y-m-d H:i')) }}" class="mt-1 w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100" placeholder="2026-01-09 20:00">
                                </div>
                            </div>

                            <div class="space-y-1">
                                <div class="text-xs text-slate-300/70">Тема</div>
                                <input name="subject" value="{{ old('subject', $mailing->subject) }}" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100">
                            </div>

                            <div class="space-y-1">
                                <div class="text-xs text-slate-300/70">Текст</div>
                                <textarea name="body" rows="10" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100">{{ old('body', $mailing->body) }}</textarea>
                            </div>

                            <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                                <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Аудитория</div>
                                <div class="mt-3 grid gap-4 md:grid-cols-2">
                                    <label class="flex items-center gap-2 text-xs text-slate-200"><input type="checkbox" name="only_admin" value="1" {{ old('only_admin', (bool) ($aud['only_admin'] ?? false)) ? 'checked' : '' }}> Только админы</label>
                                    <label class="flex items-center gap-2 text-xs text-slate-200"><input type="checkbox" name="only_non_admin" value="1" {{ old('only_non_admin', (bool) ($aud['only_non_admin'] ?? false)) ? 'checked' : '' }}> Только пользователи</label>

                                    <div class="space-y-1">
                                        <div class="text-xs text-slate-300/70">Наличие серверов</div>
                                        <select name="has_servers" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100">
                                            <option value="" {{ old('has_servers', $aud['has_servers'] ?? null) === null ? 'selected' : '' }}>—</option>
                                            <option value="yes" {{ old('has_servers') === 'yes' || (($aud['has_servers'] ?? null) === true && old('has_servers') === null) ? 'selected' : '' }}>Есть сервера</option>
                                            <option value="no" {{ old('has_servers') === 'no' || (($aud['has_servers'] ?? null) === false && old('has_servers') === null) ? 'selected' : '' }}>Нет серверов</option>
                                        </select>
                                    </div>

                                    <div class="space-y-1">
                                        <div class="text-xs text-slate-300/70">Мин. баланс</div>
                                        <input name="balance_min" value="{{ old('balance_min', $aud['balance_min'] ?? null) }}" type="number" step="0.01" min="0" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100">
                                    </div>

                                    <div class="space-y-1 md:col-span-2">
                                        <div class="text-xs text-slate-300/70">User IDs</div>
                                        <input name="user_ids" value="{{ old('user_ids', is_array($aud['user_ids'] ?? null) ? implode(',', $aud['user_ids']) : '') }}" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100" placeholder="1,2,3">
                                    </div>

                                    <div class="space-y-1 md:col-span-2">
                                        <div class="text-xs text-slate-300/70">Emails</div>
                                        <textarea name="emails" rows="3" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100">{{ old('emails', is_array($aud['emails'] ?? null) ? implode(',', $aud['emails']) : '') }}</textarea>
                                    </div>
                                </div>
                            </div>

                            <div class="flex items-center gap-3">
                                <button type="submit" class="rounded-xl bg-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">Сохранить</button>
                            </div>
                        </form>
                    </div>
                </div>

                <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
                    <div class="border-b border-white/10 bg-black/10 px-5 py-4">
                        <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Действия</div>
                    </div>
                    <div class="p-5 space-y-3">
                        <div class="text-xs text-slate-300/70">Всего: {{ (int) $mailing->total_recipients }}</div>
                        <div class="text-xs text-slate-300/70">OK: {{ (int) $mailing->sent_count }} | Fail: {{ (int) $mailing->failed_count }} | Skip: {{ (int) $mailing->skipped_count }}</div>
                        @if($mailing->last_error)
                            <div class="rounded-xl bg-red-500/10 px-3 py-2 text-xs text-red-200 ring-1 ring-red-500/20">{{ $mailing->last_error }}</div>
                        @endif

                        <form method="POST" action="{{ route('admin.mailings.start', $mailing) }}">
                            @csrf
                            <button type="submit" class="w-full rounded-xl bg-emerald-600 px-4 py-2 text-xs font-semibold text-white hover:bg-emerald-500">Запустить сейчас</button>
                        </form>

                        <form method="POST" action="{{ route('admin.mailings.schedule', $mailing) }}" class="space-y-2">
                            @csrf
                            <input name="scheduled_at" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-2 text-xs text-slate-100" placeholder="2026-01-09 20:00">
                            <button type="submit" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-2 text-xs text-slate-100 hover:bg-black/15">Запланировать</button>
                        </form>

                        <form method="POST" action="{{ route('admin.mailings.cancel', $mailing) }}">
                            @csrf
                            <button type="submit" class="w-full rounded-xl border border-white/10 bg-red-500/10 px-4 py-2 text-xs font-semibold text-red-200 hover:bg-red-500/15">Остановить</button>
                        </form>

                        <form method="POST" action="{{ route('admin.mailings.destroy', $mailing) }}" onsubmit="return confirm('Удалить кампанию?')">
                            @csrf
                            @method('DELETE')
                            <button type="submit" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-2 text-xs text-slate-100 hover:bg-black/15">Удалить</button>
                        </form>
                    </div>
                </div>
            </div>
        </div>
    </section>
@endsection
