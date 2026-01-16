@extends('layouts.app-admin')

@section('page_title', 'Тех. поддержка')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Тех. поддержка</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Управление тикетами пользователей.</p>
                </div>
            </div>

            <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-4 shadow-sm shadow-black/20">
                <form method="GET" action="{{ route('admin.support.index') }}" class="grid gap-3 md:grid-cols-12 md:items-end">
                    <div class="md:col-span-7">
                        <label for="q" class="block text-[11px] font-semibold uppercase tracking-wide text-slate-300/70">Поиск</label>
                        <input
                            id="q"
                            name="q"
                            value="{{ (string) ($q ?? '') }}"
                            placeholder="#id, тема, email пользователя"
                            class="mt-2 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                    </div>
                    <div class="md:col-span-3">
                        <label for="status" class="block text-[11px] font-semibold uppercase tracking-wide text-slate-300/70">Статус</label>
                        <select
                            id="status"
                            name="status"
                            class="mt-2 block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2 text-[13px] text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                            <option value="open" {{ (string) ($status ?? 'open') === 'open' ? 'selected' : '' }}>Открытые</option>
                            <option value="closed" {{ (string) ($status ?? '') === 'closed' ? 'selected' : '' }}>Закрытые</option>
                            <option value="all" {{ (string) ($status ?? '') === 'all' ? 'selected' : '' }}>Все</option>
                        </select>
                    </div>
                    <div class="md:col-span-2 flex gap-2">
                        <button type="submit" class="inline-flex h-10 flex-1 items-center justify-center rounded-xl bg-sky-600 px-4 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">
                            Найти
                        </button>
                        <a href="{{ route('admin.support.index') }}" class="inline-flex h-10 items-center justify-center rounded-xl border border-white/10 bg-black/10 px-4 text-xs font-semibold text-slate-200 hover:bg-black/15">
                            Сброс
                        </a>
                    </div>
                </form>
            </div>

            <div class="overflow-hidden rounded-2xl border border-white/10 bg-[#242f3d] shadow-sm shadow-black/20 text-sm">
                <table class="min-w-full divide-y divide-white/10">
                    <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                        <tr>
                            <th class="px-4 py-3 text-left">ID</th>
                            <th class="px-4 py-3 text-left">Тема</th>
                            <th class="px-4 py-3 text-left">Пользователь</th>
                            <th class="px-4 py-3 text-left">Статус</th>
                            <th class="px-4 py-3 text-left">Обновлён</th>
                            <th class="px-4 py-3 text-right">Действия</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-white/10 bg-[#242f3d] text-[13px]">
                        @forelse ($tickets as $t)
                            <tr class="hover:bg-black/10">
                                <td class="px-4 py-3 font-mono text-xs text-slate-300/80">#{{ $t->id }}</td>
                                <td class="px-4 py-3 text-slate-100 font-medium">{{ $t->subject }}</td>
                                <td class="px-4 py-3 text-slate-200">{{ $t->user?->email ?? '—' }}</td>
                                <td class="px-4 py-3">
                                    @if((string) $t->status === 'closed')
                                        <span class="inline-flex items-center rounded-full bg-black/10 px-2 py-0.5 text-[11px] font-medium text-slate-200 ring-1 ring-white/10">Закрыт</span>
                                    @else
                                        <span class="inline-flex items-center rounded-full border border-sky-500/30 bg-sky-500/10 px-2 py-0.5 text-[11px] font-medium text-sky-200">Открыт</span>
                                    @endif
                                </td>
                                <td class="px-4 py-3 text-slate-300/80">{{ ($t->last_message_at ?? $t->updated_at)?->format('d.m.Y H:i') ?? '—' }}</td>
                                <td class="px-4 py-3 text-right">
                                    <a href="{{ route('admin.support.show', $t) }}" class="inline-flex h-9 items-center justify-center rounded-xl border border-white/10 bg-black/10 px-4 text-xs font-semibold text-slate-200 hover:bg-black/15 hover:text-white">
                                        Открыть
                                    </a>
                                </td>
                            </tr>
                        @empty
                            <tr>
                                <td colspan="6" class="px-4 py-8 text-center text-slate-300/70">Тикеты не найдены</td>
                            </tr>
                        @endforelse
                    </tbody>
                </table>

                @if(method_exists($tickets, 'links'))
                    <div class="border-t border-white/10 bg-black/10 px-4 py-3">
                        {{ $tickets->links() }}
                    </div>
                @endif
            </div>
        </div>
    </section>
@endsection
