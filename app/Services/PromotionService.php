<?php

namespace App\Services;

use App\Models\Promotion;
use App\Models\User;
use Illuminate\Support\Facades\DB;

class PromotionService
{
    public const APPLY_RENT = 'rent';
    public const APPLY_RENEW = 'renew';
    public const APPLY_TOPUP = 'topup';

    public function pickPromotion(
        string $applyTo,
        ?User $user,
        ?string $code,
        ?int $tariffId,
        ?int $gameId,
        ?int $locationId,
        float $amount
    ): array {
        $code = $code !== null ? trim($code) : null;
        $code = $code !== '' ? mb_strtoupper($code) : null;

        if ($code !== null) {
            $promo = Promotion::query()->where('code', $code)->first();
            if (! $promo) {
                return ['promotion' => null, 'error' => 'Промокод не найден'];
            }

            if (! $this->isEligible($promo, $applyTo, $user, $tariffId, $gameId, $locationId, $amount)) {
                return ['promotion' => null, 'error' => 'Промокод недоступен'];
            }

            return ['promotion' => $promo, 'error' => null];
        }

        $candidates = Promotion::query()
            ->whereNull('code')
            ->where('is_active', true)
            ->where(function ($q) {
                $q->whereNull('starts_at')->orWhere('starts_at', '<=', now());
            })
            ->where(function ($q) {
                $q->whereNull('ends_at')->orWhere('ends_at', '>=', now());
            })
            ->orderByDesc('id')
            ->limit(200)
            ->get();

        $best = null;
        $bestBenefit = 0.0;

        foreach ($candidates as $promo) {
            if (! $this->isEligible($promo, $applyTo, $user, $tariffId, $gameId, $locationId, $amount)) {
                continue;
            }

            $benefit = $this->calculateBenefit($promo, $applyTo, $amount);
            if ($benefit > $bestBenefit) {
                $bestBenefit = $benefit;
                $best = $promo;
            }
        }

        return ['promotion' => $best, 'error' => null];
    }

    public function applyToAmount(Promotion $promotion, string $applyTo, float $amount): array
    {
        $amount = max(0.0, round($amount, 2));

        $discount = 0.0;
        $bonus = 0.0;

        if ($applyTo === self::APPLY_TOPUP) {
            $bonusPercent = (float) $promotion->bonus_percent;
            $bonusFixed = (float) $promotion->bonus_fixed;
            $bonus = ($amount * ($bonusPercent / 100.0)) + $bonusFixed;
            $bonus = max(0.0, round($bonus, 2));

            return [
                'amount' => $amount,
                'discount' => 0.0,
                'bonus' => $bonus,
                'final_amount' => round($amount + $bonus, 2),
            ];
        }

        $type = strtolower((string) $promotion->discount_type);
        $value = (float) $promotion->discount_value;

        if ($type === 'percent') {
            $pct = max(0.0, min(100.0, $value));
            $discount = $amount * ($pct / 100.0);
        } elseif ($type === 'fixed') {
            $discount = max(0.0, $value);
        }

        $discount = max(0.0, round($discount, 2));
        $final = max(0.0, round($amount - $discount, 2));

        return [
            'amount' => $amount,
            'discount' => $discount,
            'bonus' => 0.0,
            'final_amount' => $final,
        ];
    }

    public function lockAndIncrementUsage(int $promotionId): void
    {
        DB::transaction(function () use ($promotionId) {
            $promo = Promotion::whereKey($promotionId)->lockForUpdate()->first();
            if (! $promo) {
                return;
            }

            if (! (bool) $promo->is_active) {
                return;
            }

            if ($promo->max_uses !== null && (int) $promo->used_count >= (int) $promo->max_uses) {
                return;
            }

            $promo->increment('used_count');
        });
    }

    private function isEligible(
        Promotion $promotion,
        string $applyTo,
        ?User $user,
        ?int $tariffId,
        ?int $gameId,
        ?int $locationId,
        float $amount
    ): bool {
        if (! (bool) $promotion->is_active) {
            return false;
        }

        if ($promotion->starts_at && $promotion->starts_at->isFuture()) {
            return false;
        }

        if ($promotion->ends_at && $promotion->ends_at->isPast()) {
            return false;
        }

        $appliesTo = $promotion->applies_to;
        if (is_array($appliesTo) && count($appliesTo) > 0) {
            $allowed = array_map('strval', $appliesTo);
            if (! in_array($applyTo, $allowed, true)) {
                return false;
            }
        }

        if ($promotion->max_uses !== null && (int) $promotion->used_count >= (int) $promotion->max_uses) {
            return false;
        }

        if ($promotion->min_amount !== null && (float) $promotion->min_amount > 0) {
            if ($amount < (float) $promotion->min_amount) {
                return false;
            }
        }

        if ((bool) $promotion->only_new_users) {
            if (! $user) {
                return false;
            }

            $hasServers = $user->servers()->limit(1)->exists();
            if ($hasServers) {
                return false;
            }
        }

        if ($user) {
            $userIds = $this->normalizeIntArray($promotion->user_ids);
            if (! empty($userIds) && ! in_array((int) $user->id, $userIds, true)) {
                return false;
            }
        } else {
            if (! empty($this->normalizeIntArray($promotion->user_ids))) {
                return false;
            }
        }

        $tariffIds = $this->normalizeIntArray($promotion->tariff_ids);
        if (! empty($tariffIds)) {
            if (! $tariffId || ! in_array((int) $tariffId, $tariffIds, true)) {
                return false;
            }
        }

        $gameIds = $this->normalizeIntArray($promotion->game_ids);
        if (! empty($gameIds)) {
            if (! $gameId || ! in_array((int) $gameId, $gameIds, true)) {
                return false;
            }
        }

        $locationIds = $this->normalizeIntArray($promotion->location_ids);
        if (! empty($locationIds)) {
            if (! $locationId || ! in_array((int) $locationId, $locationIds, true)) {
                return false;
            }
        }

        return true;
    }

    private function calculateBenefit(Promotion $promotion, string $applyTo, float $amount): float
    {
        $res = $this->applyToAmount($promotion, $applyTo, $amount);
        if ($applyTo === self::APPLY_TOPUP) {
            return (float) $res['bonus'];
        }

        return (float) $res['discount'];
    }

    private function normalizeIntArray(mixed $value): array
    {
        if (! is_array($value)) {
            return [];
        }

        $out = [];
        foreach ($value as $v) {
            if (is_int($v) || (is_string($v) && ctype_digit($v))) {
                $i = (int) $v;
                if ($i > 0) {
                    $out[$i] = $i;
                }
            }
        }

        return array_values($out);
    }
}
