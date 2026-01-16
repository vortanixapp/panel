@extends('layouts.app-user')

@section('title', 'Биллинг')
@section('page_title', 'Биллинг')

@section('content')
    <section class="py-6 md:py-8">
        <div class="w-full px-4 sm:px-6 lg:px-8">
            @php
                $balanceValue = $balance ?? (auth()->user()->balance ?? 0);
                $creditsTotal = $transactions->where('type', 'credit')->sum('amount');
                $debitsTotal = $transactions->where('type', 'debit')->sum('amount');
            @endphp

            <div class="mb-6 flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Биллинг</h1>
                    <p class="mt-1 text-sm text-slate-300">История транзакций и управление балансом</p>
                </div>
                <div class="flex flex-col gap-2 md:items-end">
                    <a href="{{ route('billing.topup') }}" class="inline-flex h-9 items-center justify-center rounded-xl bg-sky-600 px-4 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">
                        Пополнить баланс
                    </a>
                    <div class="text-[12px] text-slate-300/70">
                        Последние операции отображаются ниже.
                    </div>
                </div>
            </div>

            <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 mb-6 text-sm">
                <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm transition-shadow duration-300 hover:shadow-md">
                    <p class="text-xs font-semibold text-slate-300/80">Текущий баланс</p>
                    <p class="mt-2 text-2xl font-semibold text-slate-100">{{ number_format((float) $balanceValue, 2) }} ₽</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Доступно для аренды и продлений.</p>
                </div>

                <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm transition-shadow duration-300 hover:shadow-md">
                    <p class="text-xs font-semibold text-slate-300/80">Пополнения (на странице)</p>
                    <p class="mt-2 text-2xl font-semibold text-emerald-700">+{{ number_format((float) $creditsTotal, 2) }} ₽</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Сумма за текущую выборку.</p>
                </div>

                <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm transition-shadow duration-300 hover:shadow-md">
                    <p class="text-xs font-semibold text-slate-300/80">Списания (на странице)</p>
                    <p class="mt-2 text-2xl font-semibold text-rose-700">-{{ number_format((float) $debitsTotal, 2) }} ₽</p>
                    <p class="mt-1 text-[11px] text-slate-300/70">Сумма за текущую выборку.</p>
                </div>
            </div>

            <div class="overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm text-sm transition-shadow duration-300 hover:shadow-md">
                <div class="border-b border-white/10 bg-black/10 px-6 py-4">
                    <div class="flex items-center justify-between gap-2">
                        <h3 class="text-sm font-semibold text-slate-100">История транзакций</h3>
                        <span class="text-[11px] text-slate-300/70">Показано: {{ $transactions->count() }}</span>
                    </div>
                </div>

                <div class="overflow-x-auto">
                    <table class="hidden md:table min-w-full divide-y divide-white/10">
                        <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                            <tr>
                                <th scope="col" class="px-4 py-2 text-left">Дата</th>
                                <th scope="col" class="px-4 py-2 text-left">Тип</th>
                                <th scope="col" class="px-4 py-2 text-left">Сумма</th>
                                <th scope="col" class="px-4 py-2 text-left">Описание</th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-white/10 bg-transparent text-[13px]">
                            @forelse ($transactions as $transaction)
                                <tr class="hover:bg-black/10">
                                    <td class="px-4 py-2 align-top">
                                        <div class="font-medium text-slate-100">{{ $transaction->created_at->format('d.m.Y H:i') }}</div>
                                    </td>
                                    <td class="px-4 py-2 align-top">
                                        @if($transaction->type === 'credit')
                                            <span class="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 ring-1 ring-emerald-200">
                                                Пополнение
                                            </span>
                                        @else
                                            <span class="inline-flex items-center rounded-full bg-rose-50 px-2 py-0.5 text-[11px] font-semibold text-rose-700 ring-1 ring-rose-200">
                                                Списание
                                            </span>
                                        @endif
                                    </td>
                                    <td class="px-4 py-2 align-top">
                                        <div class="font-semibold {{ $transaction->type === 'credit' ? 'text-emerald-700' : 'text-rose-700' }}">
                                            {{ $transaction->type === 'credit' ? '+' : '-' }}{{ number_format($transaction->amount, 2) }} ₽
                                        </div>
                                    </td>
                                    <td class="px-4 py-2 align-top">
                                        <div class="text-slate-200">{{ $transaction->description ?: '—' }}</div>
                                    </td>
                                </tr>
                            @empty
                                <tr>
                                    <td colspan="4" class="px-4 py-8 text-center text-slate-300/70">
                                        Нет транзакций для отображения
                                    </td>
                                </tr>
                            @endforelse
                        </tbody>
                    </table>

                    <!-- Mobile cards -->
                    <div class="md:hidden">
                        @forelse ($transactions as $transaction)
                            <div class="border-b border-white/10 p-4 last:border-b-0">
                                <div class="flex items-center justify-between">
                                    <div>
                                        <div class="font-medium text-slate-100">{{ $transaction->created_at->format('d.m.Y H:i') }}</div>
                                        <div class="text-sm text-slate-300/80">{{ $transaction->description ?: '—' }}</div>
                                    </div>
                                    <div class="text-right">
                                        @if($transaction->type === 'credit')
                                            <div class="text-emerald-700 font-semibold">+{{ number_format($transaction->amount, 2) }} ₽</div>
                                            <span class="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 ring-1 ring-emerald-200">
                                                Пополнение
                                            </span>
                                        @else
                                            <div class="text-rose-700 font-semibold">-{{ number_format($transaction->amount, 2) }} ₽</div>
                                            <span class="inline-flex items-center rounded-full bg-rose-50 px-2 py-0.5 text-[11px] font-semibold text-rose-700 ring-1 ring-rose-200">
                                                Списание
                                            </span>
                                        @endif
                                    </div>
                                </div>
                            </div>
                        @empty
                            <div class="p-8 text-center text-slate-300/70">
                                Нет транзакций для отображения
                            </div>
                        @endforelse
                    </div>
                </div>

                @if ($transactions->hasPages())
                    <div class="border-t border-white/10 bg-black/10 px-6 py-4">
                        {{ $transactions->links() }}
                    </div>
                @endif
            </div>
        </div>
    </section>
@endsection
