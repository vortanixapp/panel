@extends('layouts.app-admin')

@section('page_title', 'Редактировать акцию')

@section('content')
    <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
        <div class="border-b border-white/10 bg-black/10 px-4 py-3 flex items-center justify-between gap-3">
            <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Акция #{{ (int) $promotion->id }}</div>
            <a href="{{ route('admin.promotions.index') }}" class="rounded-md border border-white/10 bg-black/10 px-2 py-1 text-xs text-slate-100 hover:bg-black/15">Назад</a>
        </div>
        <div class="p-4">
            @if(session('success'))
                <div class="mb-4 rounded-md bg-emerald-50 p-3 text-xs text-emerald-800">
                    {{ session('success') }}
                </div>
            @endif

            <form method="POST" action="{{ route('admin.promotions.update', $promotion) }}" class="space-y-4">
                @csrf
                @method('PUT')

                <div class="grid gap-4 md:grid-cols-2">
                    <div class="space-y-1 md:col-span-2">
                        <div class="text-slate-500 text-xs">Название</div>
                        <input name="title" value="{{ old('title', $promotion->title) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" required>
                        @error('title')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Код (опционально)</div>
                        <input name="code" value="{{ old('code', $promotion->code) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="PROMO10">
                        @error('code')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Активность</div>
                        <label class="inline-flex items-center gap-2 text-xs text-slate-200">
                            <input type="checkbox" name="is_active" value="1" {{ old('is_active', $promotion->is_active) ? 'checked' : '' }}>
                            <span>Активна</span>
                        </label>
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Начало (опц.)</div>
                        <input name="starts_at" value="{{ old('starts_at', $promotion->starts_at?->format('Y-m-d H:i')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="2026-01-09 12:00">
                        @error('starts_at')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Конец (опц.)</div>
                        <input name="ends_at" value="{{ old('ends_at', $promotion->ends_at?->format('Y-m-d H:i')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="2026-01-10 12:00">
                        @error('ends_at')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1 md:col-span-2">
                        <div class="text-slate-500 text-xs">Применять к</div>
                        @php
                            $applies = old('applies_to', is_array($promotion->applies_to) ? $promotion->applies_to : []);
                        @endphp
                        <div class="flex flex-wrap gap-3 text-xs text-slate-200">
                            <label class="inline-flex items-center gap-2"><input type="checkbox" name="applies_to[]" value="rent" {{ in_array('rent', $applies) ? 'checked' : '' }}> Аренда</label>
                            <label class="inline-flex items-center gap-2"><input type="checkbox" name="applies_to[]" value="renew" {{ in_array('renew', $applies) ? 'checked' : '' }}> Продление</label>
                            <label class="inline-flex items-center gap-2"><input type="checkbox" name="applies_to[]" value="topup" {{ in_array('topup', $applies) ? 'checked' : '' }}> Пополнение</label>
                        </div>
                        @error('applies_to')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                        @error('applies_to.*')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Тип скидки (для аренды/продления)</div>
                        <select name="discount_type" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                            <option value="" {{ old('discount_type', $promotion->discount_type) === null || old('discount_type', $promotion->discount_type) === '' ? 'selected' : '' }}>—</option>
                            <option value="percent" {{ old('discount_type', $promotion->discount_type) === 'percent' ? 'selected' : '' }}>Процент</option>
                            <option value="fixed" {{ old('discount_type', $promotion->discount_type) === 'fixed' ? 'selected' : '' }}>Фикс</option>
                        </select>
                        @error('discount_type')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Значение скидки</div>
                        <input name="discount_value" value="{{ old('discount_value', (float) $promotion->discount_value) }}" type="number" step="0.01" min="0" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                        @error('discount_value')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Бонус % (для пополнения)</div>
                        <input name="bonus_percent" value="{{ old('bonus_percent', (float) $promotion->bonus_percent) }}" type="number" step="0.01" min="0" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                        @error('bonus_percent')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Бонус фикс (для пополнения)</div>
                        <input name="bonus_fixed" value="{{ old('bonus_fixed', (float) $promotion->bonus_fixed) }}" type="number" step="0.01" min="0" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                        @error('bonus_fixed')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Макс. использований (опц.)</div>
                        <input name="max_uses" value="{{ old('max_uses', $promotion->max_uses) }}" type="number" min="1" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                        @error('max_uses')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                        <div class="text-[11px] text-slate-300/70">Использовано: {{ (int) ($promotion->used_count ?? 0) }}</div>
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Мин. сумма (опц.)</div>
                        <input name="min_amount" value="{{ old('min_amount', $promotion->min_amount) }}" type="number" step="0.01" min="0" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                        @error('min_amount')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1 md:col-span-2">
                        <label class="inline-flex items-center gap-2 text-xs text-slate-200">
                            <input type="checkbox" name="only_new_users" value="1" {{ old('only_new_users', $promotion->only_new_users) ? 'checked' : '' }}>
                            <span>Только для новых пользователей</span>
                        </label>
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">User IDs (через запятую)</div>
                        <input name="user_ids" value="{{ old('user_ids', is_array($promotion->user_ids) ? implode(',', $promotion->user_ids) : '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="1,2,3">
                        @error('user_ids')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Tariff IDs (через запятую)</div>
                        <input name="tariff_ids" value="{{ old('tariff_ids', is_array($promotion->tariff_ids) ? implode(',', $promotion->tariff_ids) : '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="10,11">
                        @error('tariff_ids')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Game IDs (через запятую)</div>
                        <input name="game_ids" value="{{ old('game_ids', is_array($promotion->game_ids) ? implode(',', $promotion->game_ids) : '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="5,6">
                        @error('game_ids')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Location IDs (через запятую)</div>
                        <input name="location_ids" value="{{ old('location_ids', is_array($promotion->location_ids) ? implode(',', $promotion->location_ids) : '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100" placeholder="1,2">
                        @error('location_ids')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1 md:col-span-2">
                        <div class="text-slate-500 text-xs">Описание (опц.)</div>
                        <textarea name="description" rows="4" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">{{ old('description', $promotion->description) }}</textarea>
                        @error('description')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>
                </div>

                <div class="pt-2 flex items-center justify-between gap-3">
                    <button type="submit" class="inline-flex items-center rounded-md bg-sky-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-sky-500">Сохранить</button>

                    <form method="POST" action="{{ route('admin.promotions.destroy', $promotion) }}" onsubmit="return confirm('Удалить акцию?')">
                        @csrf
                        @method('DELETE')
                        <button type="submit" class="inline-flex items-center rounded-md border border-white/10 bg-red-500/10 px-4 py-2 text-sm font-semibold text-red-200 hover:bg-red-500/15">Удалить</button>
                    </form>
                </div>
            </form>
        </div>
    </div>
@endsection
