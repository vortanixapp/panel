@extends('layouts.app-admin')

@section('page_title', 'Акции')

@section('content')
    <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
        <div class="border-b border-white/10 bg-black/10 px-4 py-3 flex items-center justify-between gap-3">
            <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Акции</div>
            <a href="{{ route('admin.promotions.create') }}" class="inline-flex items-center rounded-md bg-sky-600 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">Добавить</a>
        </div>
        <div class="p-4">
            @if(session('success'))
                <div class="mb-4 rounded-md bg-emerald-50 p-3 text-xs text-emerald-800">
                    {{ session('success') }}
                </div>
            @endif

            <div class="overflow-x-auto">
                <table class="min-w-full text-sm">
                    <thead class="text-left text-xs text-slate-300/70">
                        <tr>
                            <th class="py-2 pr-3">ID</th>
                            <th class="py-2 pr-3">Название</th>
                            <th class="py-2 pr-3">Код</th>
                            <th class="py-2 pr-3">Статус</th>
                            <th class="py-2 pr-3">Период</th>
                            <th class="py-2 pr-3">Применение</th>
                            <th class="py-2 pr-3">Использования</th>
                            <th class="py-2 pr-3"></th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-white/10">
                        @foreach($items as $p)
                            <tr>
                                <td class="py-3 pr-3 text-xs text-slate-300/70">#{{ (int) $p->id }}</td>
                                <td class="py-3 pr-3">
                                    <div class="font-semibold text-slate-100">{{ $p->title }}</div>
                                </td>
                                <td class="py-3 pr-3">
                                    <div class="text-xs text-slate-200">{{ $p->code ?? '—' }}</div>
                                </td>
                                <td class="py-3 pr-3">
                                    @if($p->is_active)
                                        <span class="inline-flex rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-semibold text-emerald-200 ring-1 ring-emerald-500/20">Активна</span>
                                    @else
                                        <span class="inline-flex rounded-full bg-slate-500/10 px-2 py-0.5 text-[11px] font-semibold text-slate-200 ring-1 ring-white/10">Выключена</span>
                                    @endif
                                </td>
                                <td class="py-3 pr-3 text-xs text-slate-300/70">
                                    <div>{{ $p->starts_at ? $p->starts_at->format('d.m.Y H:i') : '—' }}</div>
                                    <div>{{ $p->ends_at ? $p->ends_at->format('d.m.Y H:i') : '—' }}</div>
                                </td>
                                <td class="py-3 pr-3 text-xs text-slate-300/70">
                                    @php
                                        $applies = is_array($p->applies_to) ? $p->applies_to : [];
                                    @endphp
                                    {{ count($applies) > 0 ? implode(', ', $applies) : '—' }}
                                </td>
                                <td class="py-3 pr-3 text-xs text-slate-300/70">
                                    {{ (int) ($p->used_count ?? 0) }}{{ $p->max_uses !== null ? (' / ' . (int) $p->max_uses) : '' }}
                                </td>
                                <td class="py-3 text-right">
                                    <div class="flex items-center justify-end gap-2">
                                        <a href="{{ route('admin.promotions.edit', $p) }}" class="rounded-md border border-white/10 bg-black/10 px-2 py-1 text-xs text-slate-100 hover:bg-black/15">Ред.</a>
                                        <form method="POST" action="{{ route('admin.promotions.destroy', $p) }}" onsubmit="return confirm('Удалить акцию?')">
                                            @csrf
                                            @method('DELETE')
                                            <button type="submit" class="rounded-md border border-white/10 bg-red-500/10 px-2 py-1 text-xs text-red-200 hover:bg-red-500/15">Удалить</button>
                                        </form>
                                    </div>
                                </td>
                            </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>

            <div class="mt-4">
                {{ $items->links() }}
            </div>
        </div>
    </div>
@endsection
