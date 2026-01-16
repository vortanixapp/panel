<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;

class LocationMetric extends Model
{
    use HasFactory;

    protected $fillable = [
        'location_id',
        'metric_type',
        'value',
        'text_value',
        'measured_at',
    ];

    protected function casts(): array
    {
        return [
            'value' => 'float',
            'text_value' => 'string',
            'measured_at' => 'datetime',
        ];
    }
}
