<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

class PromoCode extends Model
{
    protected $fillable = [
        'code',
        'is_active',
        'bonus_percent',
        'bonus_fixed',
        'max_uses',
        'used_count',
        'expires_at',
    ];

    protected $casts = [
        'is_active' => 'boolean',
        'bonus_percent' => 'decimal:2',
        'bonus_fixed' => 'decimal:2',
        'expires_at' => 'datetime',
    ];
}
