<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class Tariff extends Model
{
    protected $fillable = [
        'name',
        'location_id',
        'game_id',
        'billing_type',
        'price_per_slot',
        'price_per_cpu_core',
        'price_per_ram_gb',
        'price_per_disk_gb',
        'base_price_monthly',
        'cpu_min',
        'cpu_max',
        'cpu_step',
        'ram_min',
        'ram_max',
        'ram_step',
        'disk_min',
        'disk_max',
        'disk_step',
        'allow_antiddos',
        'antiddos_price',
        'min_slots',
        'max_slots',
        'cpu_cores',
        'cpu_shares',
        'ram_gb',
        'disk_gb',
        'rental_periods',
        'renewal_periods',
        'discounts',
        'position',
        'is_available',
    ];

    protected $casts = [
        'rental_periods' => 'array',
        'renewal_periods' => 'array',
        'discounts' => 'array',
        'is_available' => 'boolean',
        'cpu_shares' => 'integer',
        'billing_type' => 'string',
        'price_per_slot' => 'float',
        'price_per_cpu_core' => 'float',
        'price_per_ram_gb' => 'float',
        'price_per_disk_gb' => 'float',
        'base_price_monthly' => 'float',
        'cpu_min' => 'integer',
        'cpu_max' => 'integer',
        'cpu_step' => 'integer',
        'ram_min' => 'integer',
        'ram_max' => 'integer',
        'ram_step' => 'integer',
        'disk_min' => 'integer',
        'disk_max' => 'integer',
        'disk_step' => 'integer',
        'allow_antiddos' => 'boolean',
        'antiddos_price' => 'float',
    ];

    public function location(): BelongsTo
    {
        return $this->belongsTo(Location::class);
    }

    public function game(): BelongsTo
    {
        return $this->belongsTo(Game::class);
    }
}
