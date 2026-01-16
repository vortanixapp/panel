<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class ServerMap extends Model
{
    protected $fillable = [
        'server_id',
        'map_id',
        'installed',
        'installed_at',
        'last_error',
    ];

    protected $casts = [
        'installed' => 'boolean',
        'installed_at' => 'datetime',
    ];

    public function server(): BelongsTo
    {
        return $this->belongsTo(Server::class);
    }

    public function map(): BelongsTo
    {
        return $this->belongsTo(Map::class);
    }
}
