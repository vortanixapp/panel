<?php

namespace App\Http\Controllers;

use App\Models\Payment;
use App\Models\PromoCode;
use App\Models\Transaction;
use App\Services\PromotionService;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Http;
use Illuminate\View\View;

class UserBillingController extends Controller
{
    /**
     * Display the user's billing page with transactions.
     */
    public function index(): View
    {
        $user = auth()->user();
        $transactions = Transaction::where('user_id', $user->id)
            ->orderBy('created_at', 'desc')
            ->paginate(20);

        return view('billing', [
            'transactions' => $transactions,
            'balance' => $user->balance,
        ]);
    }

    public function topup(Request $request): View
    {
        $user = Auth::user();

        $methods = (array) config('services.freekassa.methods');
        $methods = array_values(array_filter($methods, fn ($m) => is_array($m) && array_key_exists('id', $m) && array_key_exists('name', $m)));

        $payment = null;
        $paymentId = $request->query('payment');
        if ($paymentId !== null) {
            $payment = Payment::where('user_id', $user->id)->whereKey((int) $paymentId)->first();
        }

        $payments = Payment::where('user_id', $user->id)
            ->orderBy('created_at', 'desc')
            ->limit(20)
            ->get();

        return view('billing-topup', [
            'payment' => $payment,
            'payments' => $payments,
            'methods' => $methods,
        ]);
    }

    public function createTopup(Request $request): RedirectResponse
    {
        $provider = (string) $request->input('provider', 'freekassa');
        if (! in_array($provider, ['freekassa', 'nowpayments'], true)) {
            $provider = 'freekassa';
        }

        $methods = (array) config('services.freekassa.methods');
        $methods = array_values(array_filter($methods, fn ($m) => is_array($m) && array_key_exists('id', $m) && array_key_exists('name', $m)));
        if ($provider === 'freekassa' && count($methods) === 0) {
            return redirect()->route('billing.topup')->with('error', 'Не настроены способы оплаты FreeKassa');
        }

        $validated = $request->validate([
            'provider' => ['nullable', 'string', 'max:32'],
            'amount' => ['required', 'numeric', 'min:1', 'max:1000000'],
            'promo_code' => ['nullable', 'string', 'max:64'],
            'payment_method_id' => [$provider === 'freekassa' ? 'required' : 'nullable', 'integer', 'min:1', 'max:9999'],
        ]);

        $validated = array_merge([
            'provider' => null,
            'amount' => 0,
            'promo_code' => null,
            'payment_method_id' => null,
        ], (array) $validated);

        if ($provider === 'nowpayments') {
            $apiKey = (string) config('services.nowpayments.api_key');
            $ipnSecret = (string) config('services.nowpayments.ipn_secret');
            $apiUrl = (string) config('services.nowpayments.api_url', 'https://api.nowpayments.io');
            $priceCurrency = (string) config('services.nowpayments.price_currency', 'RUB');
            $payCurrency = (string) config('services.nowpayments.pay_currency', 'usdt');

            if ($apiKey === '' || $ipnSecret === '') {
                return redirect()->route('billing.topup')->with('error', 'NowPayments не настроен');
            }

            $user = Auth::user();
            $amount = (float) $validated['amount'];
            $amountFormatted = number_format($amount, 2, '.', '');

            $promoCodeInput = trim((string) $validated['promo_code']);
            $promoCodeInput = $promoCodeInput !== '' ? mb_strtoupper($promoCodeInput) : null;

            try {
                $payment = DB::transaction(function () use ($user, $amount, $amountFormatted, $priceCurrency, $promoCodeInput) {
                    $promo = null;
                    $promotion = null;
                    if ($promoCodeInput !== null) {
                        $promoResult = app(PromotionService::class)->pickPromotion(
                            PromotionService::APPLY_TOPUP,
                            $user,
                            $promoCodeInput,
                            null,
                            null,
                            null,
                            (float) $amount
                        );
                        $promoResult = array_merge(['promotion' => null, 'error' => ''], (array) $promoResult);
                        $promotion = $promoResult['promotion'];
                        if (! $promotion) {
                            $promo = PromoCode::query()
                                ->where('code', $promoCodeInput)
                                ->first();

                            if (! $promo || ! $promo->is_active) {
                                throw new \RuntimeException((string) $promoResult['error'] !== '' ? (string) $promoResult['error'] : 'Промокод не найден или отключён');
                            }

                            if ($promo->expires_at && $promo->expires_at->isPast()) {
                                throw new \RuntimeException('Срок действия промокода истёк');
                            }

                            if ($promo->max_uses !== null && (int) $promo->used_count >= (int) $promo->max_uses) {
                                throw new \RuntimeException('Лимит использований промокода исчерпан');
                            }
                        }
                    }

                    if (! $promotion && $promoCodeInput === null) {
                        $promoResult = app(PromotionService::class)->pickPromotion(
                            PromotionService::APPLY_TOPUP,
                            $user,
                            null,
                            null,
                            null,
                            null,
                            (float) $amount
                        );
                        $promoResult = array_merge(['promotion' => null, 'error' => ''], (array) $promoResult);
                        $promotion = $promoResult['promotion'];
                    }

                    if ($promotion) {
                        $applied = app(PromotionService::class)->applyToAmount($promotion, PromotionService::APPLY_TOPUP, (float) $amount);
                        $applied = array_merge(['bonus' => 0, 'final_amount' => ($amount + 0)], (array) $applied);
                        $bonus = (float) $applied['bonus'];
                        $creditedAmount = (float) $applied['final_amount'];
                        $bonus = max(0.0, round($bonus, 2));
                        $creditedAmount = max(0.0, round($creditedAmount, 2));

                        return Payment::create([
                            'user_id' => $user->id,
                            'provider' => 'nowpayments',
                            'currency' => $priceCurrency,
                            'amount' => $amountFormatted,
                            'status' => 'pending',
                            'provider_order_id' => null,
                            'provider_payment_id' => null,
                            'promo_code' => $promotion->code,
                            'promotion_id' => (int) $promotion->id,
                            'bonus_amount' => number_format($bonus, 2, '.', ''),
                            'credited_amount' => number_format($creditedAmount, 2, '.', ''),
                            'payment_method_id' => null,
                            'credited_at' => null,
                        ]);
                    }

                    if ($promoCodeInput !== null) {
                        $promo = PromoCode::query()
                            ->where('code', $promoCodeInput)
                            ->first();

                        if (! $promo || ! $promo->is_active) {
                            throw new \RuntimeException('Промокод не найден или отключён');
                        }

                        if ($promo->expires_at && $promo->expires_at->isPast()) {
                            throw new \RuntimeException('Срок действия промокода истёк');
                        }

                        if ($promo->max_uses !== null && (int) $promo->used_count >= (int) $promo->max_uses) {
                            throw new \RuntimeException('Лимит использований промокода исчерпан');
                        }
                    }

                    $bonusPercent = $promo ? (float) $promo->bonus_percent : 0.0;
                    $bonusFixed = $promo ? (float) $promo->bonus_fixed : 0.0;
                    $bonus = ($amount * ($bonusPercent / 100.0)) + $bonusFixed;
                    $bonus = max(0.0, $bonus);
                    $bonus = round($bonus, 2);

                    $creditedAmount = $amount + $bonus;
                    $creditedAmount = round($creditedAmount, 2);

                    return Payment::create([
                        'user_id' => $user->id,
                        'provider' => 'nowpayments',
                        'currency' => $priceCurrency,
                        'amount' => $amountFormatted,
                        'status' => 'pending',
                        'provider_order_id' => null,
                        'provider_payment_id' => null,
                        'promo_code' => $promo ? $promo->code : null,
                        'promotion_id' => null,
                        'bonus_amount' => number_format($bonus, 2, '.', ''),
                        'credited_amount' => number_format($creditedAmount, 2, '.', ''),
                        'payment_method_id' => null,
                        'credited_at' => null,
                    ]);
                });
            } catch (\Throwable $e) {
                return redirect()->route('billing.topup')->with('error', $e->getMessage());
            }

            $ipnUrl = url('/api/payments/nowpayments/ipn');
            $successUrl = route('billing.topup.success', ['o' => $payment->id]);
            $cancelUrl = route('billing.topup.fail', ['o' => $payment->id]);

            $payload = [
                'price_amount' => (float) $amountFormatted,
                'price_currency' => $priceCurrency,
                'pay_currency' => $payCurrency,
                'order_id' => (string) $payment->id,
                'order_description' => 'Topup #' . $payment->id,
                'ipn_callback_url' => $ipnUrl,
                'success_url' => $successUrl,
                'cancel_url' => $cancelUrl,
            ];

            try {
                $resp = Http::withHeaders([
                    'x-api-key' => $apiKey,
                    'Content-Type' => 'application/json',
                ])->post(rtrim($apiUrl, '/') . '/v1/payment', $payload);

                if (! $resp->ok()) {
                    $payment->update(['status' => 'failed']);
                    return redirect()->route('billing.topup', ['payment' => $payment->id])->with('error', 'NowPayments: ошибка создания платежа');
                }

                $data = $resp->json();
                if (! is_array($data)) {
                    $payment->update(['status' => 'failed']);
                    return redirect()->route('billing.topup', ['payment' => $payment->id])->with('error', 'NowPayments: некорректный ответ');
                }

                $data = array_merge([
                    'payment_id' => '',
                    'invoice_url' => '',
                    'payment_url' => '',
                ], (array) $data);

                $providerPaymentId = (string) $data['payment_id'];
                $invoiceUrl = (string) $data['invoice_url'];
                if ($invoiceUrl === '') {
                    $invoiceUrl = (string) $data['payment_url'];
                }

                $payment->update([
                    'provider_payment_id' => $providerPaymentId !== '' ? $providerPaymentId : null,
                    'provider_order_id' => (string) $payment->id,
                ]);

                if ($invoiceUrl === '') {
                    return redirect()->route('billing.topup', ['payment' => $payment->id])->with('error', 'NowPayments: не получен URL оплаты');
                }

                return redirect()->away($invoiceUrl);
            } catch (\Throwable $e) {
                $payment->update(['status' => 'failed']);
                return redirect()->route('billing.topup', ['payment' => $payment->id])->with('error', 'NowPayments: ' . $e->getMessage());
            }
        }

        $merchantId = (string) config('services.freekassa.merchant_id');
        $secret1 = (string) config('services.freekassa.secret1');
        $currency = (string) config('services.freekassa.currency', 'RUB');
        $payUrl = (string) config('services.freekassa.pay_url', 'https://pay.fk.money/');

        if ($merchantId === '' || $secret1 === '') {
            return redirect()->route('billing.topup')->with('error', 'Платежный шлюз не настроен');
        }

        $user = Auth::user();
        $amount = (float) $validated['amount'];
        $amountFormatted = number_format($amount, 2, '.', '');

        $promoCodeInput = trim((string) $validated['promo_code']);
        $promoCodeInput = $promoCodeInput !== '' ? mb_strtoupper($promoCodeInput) : null;

        $paymentMethodId = $validated['payment_method_id'];
        $paymentMethodId = $paymentMethodId !== null ? (int) $paymentMethodId : null;
        $allowedMethodIds = array_map(fn ($m) => (int) $m['id'], $methods);
        if (! in_array((int) $paymentMethodId, $allowedMethodIds, true)) {
            return redirect()->route('billing.topup')->with('error', 'Выбран недоступный способ оплаты');
        }

        try {
            $payment = DB::transaction(function () use ($user, $amount, $amountFormatted, $currency, $promoCodeInput, $paymentMethodId) {
                $promo = null;
                $promotion = null;
                if ($promoCodeInput !== null) {
                    $promoResult = app(PromotionService::class)->pickPromotion(
                        PromotionService::APPLY_TOPUP,
                        $user,
                        $promoCodeInput,
                        null,
                        null,
                        null,
                        (float) $amount
                    );
                    $promoResult = array_merge(['promotion' => null, 'error' => ''], (array) $promoResult);
                    $promotion = $promoResult['promotion'];
                    if (! $promotion) {
                        $promo = PromoCode::query()
                            ->where('code', $promoCodeInput)
                            ->first();

                        if (! $promo || ! $promo->is_active) {
                            throw new \RuntimeException((string) $promoResult['error'] !== '' ? (string) $promoResult['error'] : 'Промокод не найден или отключён');
                        }

                        if ($promo->expires_at && $promo->expires_at->isPast()) {
                            throw new \RuntimeException('Срок действия промокода истёк');
                        }

                        if ($promo->max_uses !== null && (int) $promo->used_count >= (int) $promo->max_uses) {
                            throw new \RuntimeException('Лимит использований промокода исчерпан');
                        }
                    }
                }

                if (! $promotion && $promoCodeInput === null) {
                    $promoResult = app(PromotionService::class)->pickPromotion(
                        PromotionService::APPLY_TOPUP,
                        $user,
                        null,
                        null,
                        null,
                        null,
                        (float) $amount
                    );
                    $promoResult = array_merge(['promotion' => null, 'error' => ''], (array) $promoResult);
                    $promotion = $promoResult['promotion'];
                }

                if ($promotion) {
                    $applied = app(PromotionService::class)->applyToAmount($promotion, PromotionService::APPLY_TOPUP, (float) $amount);
                    $applied = array_merge(['bonus' => 0, 'final_amount' => ($amount + 0)], (array) $applied);
                    $bonus = (float) $applied['bonus'];
                    $creditedAmount = (float) $applied['final_amount'];
                    $bonus = max(0.0, round($bonus, 2));
                    $creditedAmount = max(0.0, round($creditedAmount, 2));

                    return Payment::create([
                        'user_id' => $user->id,
                        'provider' => 'freekassa',
                        'currency' => $currency,
                        'amount' => $amountFormatted,
                        'status' => 'pending',
                        'provider_order_id' => null,
                        'provider_payment_id' => null,
                        'promo_code' => $promotion->code,
                        'promotion_id' => (int) $promotion->id,
                        'bonus_amount' => number_format($bonus, 2, '.', ''),
                        'credited_amount' => number_format($creditedAmount, 2, '.', ''),
                        'payment_method_id' => $paymentMethodId,
                        'credited_at' => null,
                    ]);
                }

                if ($promoCodeInput !== null) {
                    $promo = PromoCode::query()
                        ->where('code', $promoCodeInput)
                        ->first();

                    if (! $promo || ! $promo->is_active) {
                        throw new \RuntimeException('Промокод не найден или отключён');
                    }

                    if ($promo->expires_at && $promo->expires_at->isPast()) {
                        throw new \RuntimeException('Срок действия промокода истёк');
                    }

                    if ($promo->max_uses !== null && (int) $promo->used_count >= (int) $promo->max_uses) {
                        throw new \RuntimeException('Лимит использований промокода исчерпан');
                    }
                }

                $bonusPercent = $promo ? (float) $promo->bonus_percent : 0.0;
                $bonusFixed = $promo ? (float) $promo->bonus_fixed : 0.0;
                $bonus = ($amount * ($bonusPercent / 100.0)) + $bonusFixed;
                $bonus = max(0.0, $bonus);
                $bonus = round($bonus, 2);

                $creditedAmount = $amount + $bonus;
                $creditedAmount = round($creditedAmount, 2);

                return Payment::create([
                    'user_id' => $user->id,
                    'provider' => 'freekassa',
                    'currency' => $currency,
                    'amount' => $amountFormatted,
                    'status' => 'pending',
                    'provider_order_id' => null,
                    'provider_payment_id' => null,
                    'promo_code' => $promo ? $promo->code : null,
                    'promotion_id' => null,
                    'bonus_amount' => number_format($bonus, 2, '.', ''),
                    'credited_amount' => number_format($creditedAmount, 2, '.', ''),
                    'payment_method_id' => $paymentMethodId,
                    'credited_at' => null,
                ]);
            });
        } catch (\Throwable $e) {
            return redirect()->route('billing.topup')->with('error', $e->getMessage());
        }

        $orderId = (string) $payment->id;
        $sign = md5($merchantId . ':' . $amountFormatted . ':' . $secret1 . ':' . $currency . ':' . $orderId);

        $query = http_build_query([
            'm' => $merchantId,
            'oa' => $amountFormatted,
            'o' => $orderId,
            's' => $sign,
            'currency' => $currency,
            'lang' => 'ru',
            'i' => $paymentMethodId,
        ]);

        $url = rtrim($payUrl, '/') . '/?' . $query;

        return redirect()->away($url);
    }

    public function topupSuccess(Request $request): RedirectResponse
    {
        $paymentId = (int) $request->query('o', 0);

        return redirect()
            ->route('billing.topup', ['payment' => $paymentId ?: null])
            ->with('success', 'Оплата принята. Ожидаем подтверждение от платежной системы.');
    }

    public function topupFail(Request $request): RedirectResponse
    {
        $paymentId = (int) $request->query('o', 0);

        return redirect()
            ->route('billing.topup', ['payment' => $paymentId ?: null])
            ->with('error', 'Оплата не завершена или была отменена');
    }
}
