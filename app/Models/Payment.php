<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class Payment extends Model
{
    protected $fillable = [
        'user_id',
        'provider',
        'currency',
        'amount',
        'status',
        'provider_payment_id',
        'provider_order_id',
        'promo_code',
        'promotion_id',
        'bonus_amount',
        'credited_amount',
        'payment_method_id',
        'credited_at',
    ];

    protected $casts = [
        'amount' => 'decimal:2',
        'bonus_amount' => 'decimal:2',
        'credited_amount' => 'decimal:2',
        'payment_method_id' => 'integer',
        'promotion_id' => 'integer',
        'credited_at' => 'datetime',
    ];

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
