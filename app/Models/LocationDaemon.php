<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class LocationDaemon extends Model
{
    protected $fillable = [
        'location_id',
        'status',
        'version',
        'pid',
        'uptime_sec',
        'platform',
        'last_seen',
    ];

    protected $casts = [
        'last_seen' => 'datetime',
        'uptime_sec' => 'float',
        'pid' => 'integer',
    ];

    const STATUS_ONLINE = 'online';
    const STATUS_OFFLINE = 'offline';
    const STATUS_UNKNOWN = 'unknown';

    public function location(): BelongsTo
    {
        return $this->belongsTo(Location::class);
    }
}
