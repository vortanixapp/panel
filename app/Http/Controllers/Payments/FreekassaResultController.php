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

class FreekassaResultController extends Controller
{
    public function __invoke(Request $request)
    {
        $merchantId = (string) config('services.freekassa.merchant_id');
        $secret2 = (string) config('services.freekassa.secret2');

        if ($merchantId === '' || $secret2 === '') {
            return response('misconfigured', 500);
        }

        $reqMerchant = (string) $request->input('MERCHANT_ID', '');
        $amountRaw = (string) $request->input('AMOUNT', '');
        $orderId = (string) $request->input('MERCHANT_ORDER_ID', '');
        $sign = (string) $request->input('SIGN', '');

        if ($reqMerchant === '' || $amountRaw === '' || $orderId === '' || $sign === '') {
            return response('bad request', 400);
        }

        if ($reqMerchant !== $merchantId) {
            return response('wrong merchant', 400);
        }

        $expectedSign = md5($merchantId . ':' . $amountRaw . ':' . $secret2 . ':' . $orderId);
        if (strtolower($expectedSign) !== strtolower($sign)) {
            return response('wrong sign', 400);
        }

        $allowedIps = (array) config('services.freekassa.allowed_ips');
        $checkIp = (bool) config('services.freekassa.check_ip');
        if ($checkIp && ! empty($allowedIps)) {
            $ip = (string) ($request->header('X-Real-IP') ?: $request->ip());
            if (! in_array($ip, $allowedIps, true)) {
                return response('hacking attempt', 403);
            }
        }

        $paymentId = (int) $orderId;
        if ($paymentId <= 0) {
            return response('bad order id', 400);
        }

        $amount = (float) $amountRaw;
        if ($amount <= 0) {
            return response('bad amount', 400);
        }

        $intid = $request->input('intid');
        $intid = $intid !== null ? (string) $intid : null;

        try {
            DB::transaction(function () use ($paymentId, $amount, $amountRaw, $intid) {
                $payment = Payment::whereKey($paymentId)->lockForUpdate()->first();
                if (! $payment) {
                    throw new \RuntimeException('payment not found');
                }

                if ((string) $payment->provider !== 'freekassa') {
                    throw new \RuntimeException('wrong provider');
                }

                if ($payment->credited_at) {
                    $payment->update([
                        'status' => 'succeeded',
                    ]);
                    return;
                }

                $expectedAmount = (string) number_format((float) $payment->amount, 2, '.', '');
                $incomingAmount = (string) number_format((float) $amount, 2, '.', '');
                if ($expectedAmount !== $incomingAmount) {
                    throw new \RuntimeException('amount mismatch');
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

                $desc = 'Пополнение через FreeKassa #' . $payment->id;
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
                    'provider_payment_id' => $intid ?: null,
                    'provider_order_id' => (string) $paymentId,
                ]);
            });
        } catch (\Throwable $e) {
            Log::warning('FreeKassa result error: ' . $e->getMessage(), [
                'order_id' => $orderId,
                'amount' => $amountRaw,
            ]);

            return response('error', 500);
        }

        return response('YES', 200);
    }
}
