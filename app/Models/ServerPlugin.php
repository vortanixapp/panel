<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class ServerPlugin extends Model
{
    protected $fillable = [
        'server_id',
        'plugin_id',
        'installed',
        'enabled',
        'installed_at',
        'last_error',
    ];

    protected $casts = [
        'installed' => 'boolean',
        'enabled' => 'boolean',
        'installed_at' => 'datetime',
    ];

    public function server(): BelongsTo
    {
        return $this->belongsTo(Server::class);
    }

    public function plugin(): BelongsTo
    {
        return $this->belongsTo(Plugin::class);
    }
}
