<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class LocationSetupStatus extends Model
{
    protected $fillable = [
        'location_id',
        'component',
        'status',
        'installed_at',
        'error_message',
    ];

    protected $casts = [
        'installed_at' => 'datetime',
    ];

    const STATUS_PENDING = 'pending';
    const STATUS_INSTALLING = 'installing';
    const STATUS_INSTALLED = 'installed';
    const STATUS_FAILED = 'failed';

    const COMPONENTS = ['packages', 'docker', 'mysql', 'phpmyadmin', 'ftp', 'daemon', 'images'];

    public function location(): BelongsTo
    {
        return $this->belongsTo(Location::class);
    }
}
