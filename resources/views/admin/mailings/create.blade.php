@extends('layouts.app-admin')

@section('page_title', 'Создать рассылку')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex items-center justify-between gap-3">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Создать рассылку</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Новая кампания</p>
                </div>
                <a href="{{ route('admin.mailings.index') }}" class="rounded-xl border border-white/10 bg-black/10 px-4 py-2 text-xs text-slate-100 hover:bg-black/15">Назад</a>
            </div>

            <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
                <div class="border-b border-white/10 bg-black/10 px-5 py-4">
                    <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Параметры</div>
                </div>
                <div class="p-5">
                    <form method="POST" action="{{ route('admin.mailings.store') }}" class="space-y-5">
                        @csrf

                        <div class="space-y-1">
                            <div class="text-xs text-slate-300/70">Название</div>
                            <input name="title" value="{{ old('title') }}" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100" required>
                            @error('title')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                        </div>

                        <div class="grid gap-4 md:grid-cols-2">
                            <div class="space-y-1">
                                <div class="text-xs text-slate-300/70">Каналы</div>
                                <div class="space-y-2 text-xs text-slate-200">
                                    <label class="flex items-center gap-2"><input type="checkbox" name="channels[]" value="email" {{ in_array('email', old('channels', ['email'])) ? 'checked' : '' }}> Email</label>
                                    <label class="flex items-center gap-2"><input type="checkbox" name="channels[]" value="internal" {{ in_array('internal', old('channels', [])) ? 'checked' : '' }}> Внутренние уведомления</label>
                                    <label class="flex items-center gap-2"><input type="checkbox" name="channels[]" value="telegram" {{ in_array('telegram', old('channels', [])) ? 'checked' : '' }}> Telegram</label>
                                    <label class="flex items-center gap-2"><input type="checkbox" name="channels[]" value="sms" {{ in_array('sms', old('channels', [])) ? 'checked' : '' }}> SMS</label>
                                </div>
                                @error('channels')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                                @error('channels.*')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                            </div>

                            <div class="space-y-1">
                                <div class="text-xs text-slate-300/70">План (опционально)</div>
                                <input name="scheduled_at" value="{{ old('scheduled_at') }}" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100" placeholder="2026-01-09 20:00">
                                @error('scheduled_at')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                                <label class="mt-2 flex items-center gap-2 text-xs text-slate-200">
                                    <input type="checkbox" name="is_html" value="1" {{ old('is_html', 1) ? 'checked' : '' }}>
                                    <span>HTML</span>
                                </label>
                            </div>
                        </div>

                        <div class="space-y-1">
                            <div class="text-xs text-slate-300/70">Тема (для Email/уведомлений)</div>
                            <input name="subject" value="{{ old('subject') }}" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100">
                            @error('subject')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                        </div>

                        <div class="space-y-1">
                            <div class="text-xs text-slate-300/70">Текст</div>
                            <textarea name="body" rows="10" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100">{{ old('body') }}</textarea>
                            @error('body')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                        </div>

                        <div class="rounded-2xl border border-white/10 bg-black/10 p-4">
                            <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Аудитория</div>
                            <div class="mt-3 grid gap-4 md:grid-cols-2">
                                <label class="flex items-center gap-2 text-xs text-slate-200"><input type="checkbox" name="only_admin" value="1" {{ old('only_admin') ? 'checked' : '' }}> Только админы</label>
                                <label class="flex items-center gap-2 text-xs text-slate-200"><input type="checkbox" name="only_non_admin" value="1" {{ old('only_non_admin') ? 'checked' : '' }}> Только пользователи (не админы)</label>

                                <div class="space-y-1">
                                    <div class="text-xs text-slate-300/70">Наличие серверов</div>
                                    <select name="has_servers" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100">
                                        <option value="" {{ old('has_servers') === null || old('has_servers') === '' ? 'selected' : '' }}>—</option>
                                        <option value="yes" {{ old('has_servers') === 'yes' ? 'selected' : '' }}>Есть сервера</option>
                                        <option value="no" {{ old('has_servers') === 'no' ? 'selected' : '' }}>Нет серверов</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-xs text-slate-300/70">Мин. баланс</div>
                                    <input name="balance_min" value="{{ old('balance_min') }}" type="number" step="0.01" min="0" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100">
                                </div>

                                <div class="space-y-1 md:col-span-2">
                                    <div class="text-xs text-slate-300/70">User IDs (через запятую)</div>
                                    <input name="user_ids" value="{{ old('user_ids') }}" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100" placeholder="1,2,3">
                                </div>

                                <div class="space-y-1 md:col-span-2">
                                    <div class="text-xs text-slate-300/70">Emails (через запятую)</div>
                                    <textarea name="emails" rows="3" class="w-full rounded-xl border border-white/10 bg-black/10 px-4 py-3 text-sm text-slate-100">{{ old('emails') }}</textarea>
                                </div>
                            </div>
                        </div>

                        <div class="flex items-center gap-3">
                            <button type="submit" class="rounded-xl bg-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">Сохранить</button>
                            <span class="text-xs text-slate-300/70">После создания можно запустить или запланировать.</span>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    </section>
@endsection
