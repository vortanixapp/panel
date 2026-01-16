<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

class Promotion extends Model
{
    protected $fillable = [
        'title',
        'code',
        'is_active',
        'starts_at',
        'ends_at',
        'applies_to',
        'discount_type',
        'discount_value',
        'bonus_percent',
        'bonus_fixed',
        'max_uses',
        'used_count',
        'min_amount',
        'only_new_users',
        'user_ids',
        'tariff_ids',
        'game_ids',
        'location_ids',
        'description',
    ];

    protected $casts = [
        'is_active' => 'boolean',
        'starts_at' => 'datetime',
        'ends_at' => 'datetime',
        'applies_to' => 'array',
        'discount_value' => 'decimal:2',
        'bonus_percent' => 'decimal:2',
        'bonus_fixed' => 'decimal:2',
        'min_amount' => 'decimal:2',
        'only_new_users' => 'boolean',
        'user_ids' => 'array',
        'tariff_ids' => 'array',
        'game_ids' => 'array',
        'location_ids' => 'array',
    ];
}
