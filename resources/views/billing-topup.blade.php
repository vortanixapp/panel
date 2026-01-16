@extends('layouts.app-user')

@section('title', 'Пополнение баланса')
@section('page_title', 'Пополнение баланса')

@section('content')
    <section class="py-6 md:py-8">
        <div class="w-full px-4 sm:px-6 lg:px-8">
            <div class="mb-6 flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Пополнение баланса</h1>
                    <p class="mt-1 text-sm text-slate-300">Выберите платежную систему. Зачисление происходит после подтверждения оплаты.</p>
                </div>
                <div class="text-[12px] text-slate-300/70">
                    <a href="{{ route('billing') }}" class="underline underline-offset-2 hover:text-slate-200">Перейти к истории транзакций</a>
                </div>
            </div>

            @if (session('error'))
                <div class="mb-6 rounded-2xl border border-red-200 bg-red-50/70 px-4 py-3 text-sm text-red-800 shadow-sm">
                    {{ session('error') }}
                </div>
            @endif
            @if (session('success'))
                <div class="mb-6 rounded-2xl border border-green-200 bg-green-50/70 px-4 py-3 text-sm text-green-800 shadow-sm">
                    {{ session('success') }}
                </div>
            @endif

            @if(isset($payment) && $payment)
                <div class="mb-6 rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm">
                    <div class="text-sm text-slate-200">
                        Заявка на пополнение <span class="font-semibold">#{{ $payment->id }}</span>
                    </div>
                    <div class="mt-2 text-xs text-slate-300/80">
                        Статус: <span class="font-semibold">{{ $payment->status }}</span>
                    </div>
                    <div class="mt-3 text-sm text-slate-200">
                        Сумма: <span class="font-semibold">{{ number_format((float) $payment->amount, 2) }} ₽</span>
                    </div>
                    @if(!empty($payment->promo_code))
                        <div class="mt-2 text-sm text-slate-200">
                            Промокод: <span class="font-semibold">{{ $payment->promo_code }}</span>
                        </div>
                    @endif
                    @if((float) ($payment->bonus_amount ?? 0) > 0)
                        <div class="mt-2 text-sm text-slate-200">
                            Бонус: <span class="font-semibold">+{{ number_format((float) $payment->bonus_amount, 2) }} ₽</span>
                        </div>
                    @endif
                    @if((float) ($payment->credited_amount ?? 0) > 0)
                        <div class="mt-2 text-sm text-slate-200">
                            Итого к зачислению: <span class="font-semibold">{{ number_format((float) $payment->credited_amount, 2) }} ₽</span>
                        </div>
                    @endif
                    @if(!empty($payment->payment_method_id))
                        <div class="mt-2 text-xs text-slate-300/80">
                            Способ оплаты ID: <span class="font-semibold">{{ $payment->payment_method_id }}</span>
                        </div>
                    @endif
                    <div class="mt-2 text-[12px] text-slate-300/70">
                        Если оплата прошла, но баланс ещё не изменился — подожди 1-2 минуты и обнови страницу.
                    </div>
                </div>
            @endif

            <div class="mb-6 overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm">
                <div class="border-b border-white/10 bg-black/10 px-5 py-4">
                    <div class="flex items-center justify-between gap-2">
                        <h2 class="text-sm font-semibold text-slate-100">Мои пополнения</h2>
                        <span class="text-[11px] text-slate-300/70">Последние 20</span>
                    </div>
                </div>
                <div class="overflow-x-auto">
                    <table class="min-w-full divide-y divide-white/10 text-sm">
                        <thead class="bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                            <tr>
                                <th scope="col" class="px-4 py-2 text-left">Дата</th>
                                <th scope="col" class="px-4 py-2 text-left">ID</th>
                                <th scope="col" class="px-4 py-2 text-left">Статус</th>
                                <th scope="col" class="px-4 py-2 text-left">Сумма</th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-white/10 bg-transparent text-[13px]">
                            @forelse (($payments ?? []) as $p)
                                <tr class="hover:bg-black/10">
                                    <td class="px-4 py-2 align-top">
                                        <div class="font-medium text-slate-100">{{ $p->created_at ? $p->created_at->format('d.m.Y H:i') : '—' }}</div>
                                    </td>
                                    <td class="px-4 py-2 align-top">
                                        <a href="{{ route('billing.topup', ['payment' => $p->id]) }}" class="text-sky-300 hover:text-sky-200 underline underline-offset-2">
                                            #{{ $p->id }}
                                        </a>
                                    </td>
                                    <td class="px-4 py-2 align-top">
                                        <div class="text-slate-200">{{ $p->status }}</div>
                                    </td>
                                    <td class="px-4 py-2 align-top">
                                        <div class="font-semibold text-slate-100">{{ number_format((float) $p->amount, 2) }} ₽</div>
                                        @if((float) ($p->credited_amount ?? 0) > (float) ($p->amount ?? 0))
                                            <div class="text-[11px] text-slate-300/70">К зачислению: {{ number_format((float) $p->credited_amount, 2) }} ₽</div>
                                        @endif
                                    </td>
                                </tr>
                            @empty
                                <tr>
                                    <td colspan="4" class="px-4 py-8 text-center text-slate-300/70">
                                        Пока нет заявок на пополнение
                                    </td>
                                </tr>
                            @endforelse
                        </tbody>
                    </table>
                </div>
            </div>

            <div class="overflow-hidden rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm">
                <div class="border-b border-white/10 bg-black/10 px-5 py-4">
                    <h2 class="text-sm font-semibold text-slate-100">Новая оплата</h2>
                    <p class="mt-1 text-[12px] text-slate-300/70">Введите сумму и перейдите на страницу оплаты.</p>
                </div>

                <div class="p-5">
                    <form method="POST" action="{{ route('billing.topup.create') }}" class="grid gap-4" x-data="{ provider: '{{ old('provider', 'freekassa') }}' }">
                        @csrf
                        <div class="grid gap-4 sm:grid-cols-3">
                            <div class="sm:col-span-1">
                                <label for="provider" class="block text-xs font-semibold text-slate-500 mb-1">Платежная система</label>
                                <select
                                    id="provider"
                                    name="provider"
                                    class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    x-model="provider"
                                >
                                    <option value="freekassa">FreeKassa</option>
                                    <option value="nowpayments">NowPayments</option>
                                </select>
                                @error('provider')
                                    <div class="mt-2 text-xs text-rose-300">{{ $message }}</div>
                                @enderror
                            </div>

                            <div class="sm:col-span-1">
                                <label for="amount" class="block text-xs font-semibold text-slate-500 mb-1">Сумма пополнения (₽)</label>
                                <input
                                    id="amount"
                                    name="amount"
                                    type="number"
                                    min="1"
                                    step="0.01"
                                    required
                                    value="{{ old('amount', '100') }}"
                                    class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                />
                                @error('amount')
                                    <div class="mt-2 text-xs text-rose-300">{{ $message }}</div>
                                @enderror
                            </div>

                            <div class="sm:col-span-1">
                                <label for="promo_code" class="block text-xs font-semibold text-slate-500 mb-1">Промокод (бонус к зачислению)</label>
                                <input
                                    id="promo_code"
                                    name="promo_code"
                                    type="text"
                                    value="{{ old('promo_code') }}"
                                    class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                />
                                @error('promo_code')
                                    <div class="mt-2 text-xs text-rose-300">{{ $message }}</div>
                                @enderror
                            </div>

                            <div class="sm:col-span-1">
                                <label for="payment_method_id" class="block text-xs font-semibold text-slate-500 mb-1">Способ оплаты (ID)</label>
                                <select
                                    id="payment_method_id"
                                    name="payment_method_id"
                                    class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                    :required="provider === 'freekassa'"
                                    x-show="provider === 'freekassa'"
                                >
                                    <option value="">Выберите способ оплаты</option>
                                    @foreach(($methods ?? []) as $method)
                                        <option value="{{ (int) $method['id'] }}" {{ (string) old('payment_method_id') === (string) $method['id'] ? 'selected' : '' }}>
                                            {{ $method['name'] }}
                                        </option>
                                    @endforeach
                                </select>
                                @error('payment_method_id')
                                    <div class="mt-2 text-xs text-rose-300">{{ $message }}</div>
                                @enderror
                            </div>
                        </div>

                        <div class="text-[12px] text-slate-300/70">
                            Промокод добавляет бонус к зачислению на баланс (сумма оплаты в FreeKassa не меняется).
                        </div>

                        <div class="flex justify-end">
                            <button type="submit" class="inline-flex h-10 items-center justify-center rounded-xl bg-sky-600 px-5 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">
                                Перейти к оплате
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    </section>
@endsection
