@extends('layouts.app-admin')

@section('page_title', 'Рассылка')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex items-center justify-between gap-3">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Рассылка</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Кампании рассылок</p>
                </div>
                <a href="{{ route('admin.mailings.create') }}" class="rounded-xl bg-sky-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">Создать</a>
            </div>

            @if(session('success'))
                <div class="rounded-xl bg-emerald-500/10 px-4 py-3 text-xs text-emerald-200 ring-1 ring-emerald-500/20">{{ session('success') }}</div>
            @endif

            <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
                <div class="border-b border-white/10 bg-black/10 px-5 py-4">
                    <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Список кампаний</div>
                </div>

                <div class="overflow-x-auto">
                    <table class="min-w-full text-sm">
                        <thead class="text-left text-xs text-slate-300/70">
                        <tr>
                            <th class="py-3 px-5">ID</th>
                            <th class="py-3 px-5">Название</th>
                            <th class="py-3 px-5">Статус</th>
                            <th class="py-3 px-5">Каналы</th>
                            <th class="py-3 px-5">План</th>
                            <th class="py-3 px-5">Счётчики</th>
                            <th class="py-3 px-5"></th>
                        </tr>
                        </thead>
                        <tbody class="divide-y divide-white/10">
                        @foreach($items as $m)
                            @php
                                $channels = is_array($m->channels) ? $m->channels : [];
                            @endphp
                            <tr>
                                <td class="py-3 px-5 text-xs text-slate-300/70">#{{ (int) $m->id }}</td>
                                <td class="py-3 px-5">
                                    <div class="font-semibold text-slate-100">{{ $m->title }}</div>
                                </td>
                                <td class="py-3 px-5">
                                    <span class="inline-flex rounded-full bg-black/20 px-2 py-0.5 text-[11px] font-semibold text-slate-200 ring-1 ring-white/10">{{ $m->status }}</span>
                                </td>
                                <td class="py-3 px-5 text-xs text-slate-300/70">
                                    {{ count($channels) ? implode(', ', $channels) : '—' }}
                                </td>
                                <td class="py-3 px-5 text-xs text-slate-300/70">
                                    {{ $m->scheduled_at ? $m->scheduled_at->format('d.m.Y H:i') : '—' }}
                                </td>
                                <td class="py-3 px-5 text-xs text-slate-300/70">
                                    <div>Всего: {{ (int) $m->total_recipients }}</div>
                                    <div>OK: {{ (int) $m->sent_count }} | Fail: {{ (int) $m->failed_count }} | Skip: {{ (int) $m->skipped_count }}</div>
                                </td>
                                <td class="py-3 px-5 text-right">
                                    <a href="{{ route('admin.mailings.edit', $m) }}" class="rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs text-slate-100 hover:bg-black/15">Открыть</a>
                                </td>
                            </tr>
                        @endforeach
                        </tbody>
                    </table>
                </div>

                <div class="px-5 py-4">
                    {{ $items->links() }}
                </div>
            </div>
        </div>
    </section>
@endsection
