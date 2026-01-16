@extends('layouts.app-admin')

@section('page_title', 'Новости')

@section('content')
    <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
        <div class="border-b border-white/10 bg-black/10 px-4 py-3 flex items-center justify-between gap-3">
            <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Новости</div>
            <a href="{{ route('admin.news.create') }}" class="inline-flex items-center rounded-md bg-sky-600 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">Добавить</a>
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
                            <th class="py-2 pr-3">Заголовок</th>
                            <th class="py-2 pr-3">Статус</th>
                            <th class="py-2 pr-3">Дата</th>
                            <th class="py-2 pr-3"></th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-white/10">
                        @foreach($items as $n)
                            <tr>
                                <td class="py-3 pr-3">
                                    <div class="font-semibold text-slate-100">{{ $n->title }}</div>
                                    <div class="text-xs text-slate-300/70">{{ $n->slug }}</div>
                                </td>
                                <td class="py-3 pr-3">
                                    @if($n->active)
                                        <span class="inline-flex rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-semibold text-emerald-200 ring-1 ring-emerald-500/20">Активна</span>
                                    @else
                                        <span class="inline-flex rounded-full bg-slate-500/10 px-2 py-0.5 text-[11px] font-semibold text-slate-200 ring-1 ring-white/10">Выключена</span>
                                    @endif
                                </td>
                                <td class="py-3 pr-3 text-xs text-slate-300/70">
                                    {{ optional($n->published_at)->format('d.m.Y H:i') ?? '—' }}
                                </td>
                                <td class="py-3 text-right">
                                    <div class="flex items-center justify-end gap-2">
                                        <a href="{{ route('admin.news.edit', $n) }}" class="rounded-md border border-white/10 bg-black/10 px-2 py-1 text-xs text-slate-100 hover:bg-black/15">Ред.</a>
                                        <form method="POST" action="{{ route('admin.news.destroy', $n) }}" onsubmit="return confirm('Удалить новость?')">
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
