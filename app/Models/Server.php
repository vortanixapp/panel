<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class Server extends Model
{
    protected $fillable = [
        'user_id',
        'tariff_id',
        'game_id',
        'game_version_id',
        'location_id',
        'name',
        'ip_address',
        'port',
        'slots',
        'cpu_cores',
        'cpu_shares',
        'ram_gb',
        'disk_gb',
        'antiddos_enabled',
        'server_fps',
        'server_tickrate',
        'expires_at',
        'status',
        'container_id',
        'container_name',
        'provisioning_status',
        'provisioning_error',
        'runtime_status',
        'auto_start_enabled',
        'ftp_host',
        'ftp_port',
        'ftp_username',
        'ftp_password',
        'ftp_root',
        'mysql_host',
        'mysql_port',
        'mysql_database',
        'mysql_username',
        'mysql_password',
    ];

    protected $casts = [
        'expires_at' => 'datetime',
        'port' => 'integer',
        'slots' => 'integer',
        'cpu_cores' => 'float',
        'cpu_shares' => 'integer',
        'ram_gb' => 'integer',
        'disk_gb' => 'integer',
        'antiddos_enabled' => 'boolean',
        'server_fps' => 'integer',
        'server_tickrate' => 'integer',
        'auto_start_enabled' => 'boolean',
        'ftp_port' => 'integer',
        'mysql_port' => 'integer',
    ];

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }

    public function tariff(): BelongsTo
    {
        return $this->belongsTo(Tariff::class);
    }

    public function game(): BelongsTo
    {
        return $this->belongsTo(Game::class);
    }

    public function gameVersion(): BelongsTo
    {
        return $this->belongsTo(GameVersion::class);
    }

    public function location(): BelongsTo
    {
        return $this->belongsTo(Location::class);
    }
}
