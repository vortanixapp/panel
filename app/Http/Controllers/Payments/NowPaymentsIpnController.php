<?php

namespace App\Http\Controllers\Payments;

use App\Http\Controllers\Controller;
use App\Models\Payment;
use App\Models\Promotion;
use App\Models\PromoCode;
use App\Models\Transaction;
use App\Models\User;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;

class NowPaymentsIpnController extends Controller
{
    public function __invoke(Request $request)
    {
        $ipnSecret = (string) config('services.nowpayments.ipn_secret');
        if ($ipnSecret === '') {
            return response('misconfigured', 500);
        }

        $raw = (string) $request->getContent();
        $signature = (string) $request->header('x-nowpayments-sig');
        if ($signature === '') {
            return response('bad request', 400);
        }

        $expected = hash_hmac('sha512', $raw, $ipnSecret);
        if (! hash_equals(strtolower($expected), strtolower($signature))) {
            return response('wrong sign', 400);
        }

        $payload = $request->json()->all();
        if (! is_array($payload)) {
            return response('bad request', 400);
        }

        $orderId = (string) $payload['order_id'];
        $paymentId = (string) $payload['payment_id'];
        $status = (string) $payload['payment_status'];
        $priceAmountRaw = $payload['price_amount'];
        $priceCurrency = (string) $payload['price_currency'];

        if ($orderId === '' || $status === '') {
            return response('bad request', 400);
        }

        $localPaymentId = (int) $orderId;
        if ($localPaymentId <= 0) {
            return response('bad order id', 400);
        }

        $paidStatuses = ['finished', 'confirmed'];
        if (! in_array($status, $paidStatuses, true)) {
            return response('OK', 200);
        }

        $incomingAmount = null;
        if ($priceAmountRaw !== null && is_numeric($priceAmountRaw)) {
            $incomingAmount = (string) number_format((float) $priceAmountRaw, 2, '.', '');
        }

        try {
            DB::transaction(function () use ($localPaymentId, $incomingAmount, $priceCurrency, $paymentId) {
                $payment = Payment::whereKey($localPaymentId)->lockForUpdate()->first();
                if (! $payment) {
                    throw new \RuntimeException('payment not found');
                }

                if ((string) $payment->provider !== 'nowpayments') {
                    throw new \RuntimeException('wrong provider');
                }

                if ($payment->credited_at) {
                    $payment->update([
                        'status' => 'succeeded',
                    ]);
                    return;
                }

                if ($incomingAmount !== null) {
                    $expectedAmount = (string) number_format((float) $payment->amount, 2, '.', '');
                    if ($expectedAmount !== $incomingAmount) {
                        throw new \RuntimeException('amount mismatch');
                    }
                }

                $expectedCurrency = (string) $payment->currency;
                if ($priceCurrency !== '' && $expectedCurrency !== '' && strtoupper($expectedCurrency) !== strtoupper($priceCurrency)) {
                    throw new \RuntimeException('currency mismatch');
                }

                $credited = (float) ($payment->credited_amount ?: $payment->amount);
                $credited = max(0.0, $credited);
                $credited = round($credited, 2);
                if ($credited <= 0) {
                    throw new \RuntimeException('credited amount invalid');
                }

                if (! empty($payment->promo_code)) {
                    $promo = PromoCode::query()
                        ->where('code', (string) $payment->promo_code)
                        ->lockForUpdate()
                        ->first();

                    if (! $promo || ! $promo->is_active) {
                        throw new \RuntimeException('promo invalid');
                    }

                    if ($promo->expires_at && $promo->expires_at->isPast()) {
                        throw new \RuntimeException('promo expired');
                    }

                    if ($promo->max_uses !== null && (int) $promo->used_count >= (int) $promo->max_uses) {
                        throw new \RuntimeException('promo max uses');
                    }

                    $promo->increment('used_count');
                }

                if (! empty($payment->promotion_id)) {
                    $promotion = Promotion::query()
                        ->whereKey((int) $payment->promotion_id)
                        ->lockForUpdate()
                        ->first();

                    if (! $promotion || ! $promotion->is_active) {
                        throw new \RuntimeException('promotion invalid');
                    }

                    if ($promotion->starts_at && $promotion->starts_at->isFuture()) {
                        throw new \RuntimeException('promotion not started');
                    }

                    if ($promotion->ends_at && $promotion->ends_at->isPast()) {
                        throw new \RuntimeException('promotion expired');
                    }

                    if ($promotion->max_uses !== null && (int) $promotion->used_count >= (int) $promotion->max_uses) {
                        throw new \RuntimeException('promotion max uses');
                    }

                    $promotion->increment('used_count');
                }

                $user = User::whereKey($payment->user_id)->lockForUpdate()->first();
                if (! $user) {
                    throw new \RuntimeException('user not found');
                }

                $user->increment('balance', $credited);

                $desc = 'Пополнение через NowPayments #' . $payment->id;
                if (! empty($payment->promo_code)) {
                    $desc .= ' (промокод ' . $payment->promo_code . ')';
                }

                Transaction::create([
                    'user_id' => $user->id,
                    'type' => 'credit',
                    'amount' => $credited,
                    'description' => $desc,
                ]);

                $payment->update([
                    'status' => 'succeeded',
                    'credited_at' => now(),
                    'provider_payment_id' => $paymentId !== '' ? $paymentId : null,
                    'provider_order_id' => (string) $payment->id,
                ]);
            });
        } catch (\Throwable $e) {
            Log::warning('NowPayments IPN error: ' . $e->getMessage(), [
                'order_id' => $orderId,
                'payment_id' => $paymentId,
                'status' => $status,
            ]);

            return response('error', 500);
        }

        return response('OK', 200);
    }
}
