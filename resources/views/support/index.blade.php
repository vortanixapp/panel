@extends('layouts.app-user')

@section('page_title', 'Тех. поддержка')

@section('content')
    <section class="py-6 md:py-8">
        <div class="w-full px-4 sm:px-6 lg:px-8">
            <div class="mb-6 flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Тех. поддержка</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Создавайте тикеты и общайтесь с администраторами в чате.</p>
                </div>
                <a href="{{ route('support.create') }}" class="inline-flex h-10 items-center justify-center rounded-xl bg-sky-600 px-4 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">
                    Создать тикет
                </a>
            </div>

            @if (session('success'))
                <div class="mb-4 rounded-2xl border border-emerald-500/20 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
                    {{ session('success') }}
                </div>
            @endif
            @if (session('error'))
                <div class="mb-4 rounded-2xl border border-rose-500/20 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
                    {{ session('error') }}
                </div>
            @endif

            <div class="overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm">
                <div class="border-b border-white/10 bg-black/10 px-5 py-4">
                    <h2 class="text-sm font-semibold text-slate-100">Мои тикеты</h2>
                </div>
                <div class="overflow-x-auto">
                    <table class="min-w-full divide-y divide-white/10 text-sm">
                        <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                            <tr>
                                <th class="px-4 py-3 text-left">ID</th>
                                <th class="px-4 py-3 text-left">Тема</th>
                                <th class="px-4 py-3 text-left">Статус</th>
                                <th class="px-4 py-3 text-left">Обновлён</th>
                                <th class="px-4 py-3 text-right">Действие</th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-white/10 text-[13px]">
                            @forelse($tickets as $t)
                                <tr class="hover:bg-black/10">
                                    <td class="px-4 py-3 font-mono text-xs text-slate-300/80">#{{ $t->id }}</td>
                                    <td class="px-4 py-3 text-slate-100 font-medium">{{ $t->subject }}</td>
                                    <td class="px-4 py-3">
                                        @if((string) $t->status === 'closed')
                                            <span class="inline-flex items-center rounded-full bg-black/10 px-2 py-0.5 text-[11px] font-medium text-slate-200 ring-1 ring-white/10">Закрыт</span>
                                        @else
                                            <span class="inline-flex items-center rounded-full border border-sky-500/30 bg-sky-500/10 px-2 py-0.5 text-[11px] font-medium text-sky-200">Открыт</span>
                                        @endif
                                    </td>
                                    <td class="px-4 py-3 text-slate-300/80">{{ ($t->last_message_at ?? $t->updated_at)?->format('d.m.Y H:i') ?? '—' }}</td>
                                    <td class="px-4 py-3 text-right">
                                        <a href="{{ route('support.show', $t) }}" class="inline-flex h-9 items-center justify-center rounded-xl border border-white/10 bg-black/10 px-4 text-xs font-semibold text-slate-200 hover:bg-black/15 hover:text-white">
                                            Открыть
                                        </a>
                                    </td>
                                </tr>
                            @empty
                                <tr>
                                    <td colspan="5" class="px-4 py-8 text-center text-slate-300/70">Тикетов пока нет</td>
                                </tr>
                            @endforelse
                        </tbody>
                    </table>
                </div>
                @if(method_exists($tickets, 'links'))
                    <div class="border-t border-white/10 bg-black/10 px-4 py-3 text-xs text-slate-300/80">
                        {{ $tickets->links() }}
                    </div>
                @endif
            </div>
        </div>
    </section>
@endsection
