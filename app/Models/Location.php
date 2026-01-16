<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Location extends Model
{
    use HasFactory;

    protected $fillable = [
        'code',
        'name',
        'country',
        'city',
        'region',
        'description',
        'ip_address',
        'ip_pool',
        'mysql_host',
        'mysql_port',
        'mysql_root_username',
        'mysql_root_password',
        'phpmyadmin_port',
        'ssh_host',
        'ssh_user',
        'ssh_password',
        'ssh_port',
        'is_active',
        'sort_order',
    ];

    protected function casts(): array
    {
        return [
            'is_active' => 'boolean',
            'ssh_port' => 'integer',
            'mysql_port' => 'integer',
            'phpmyadmin_port' => 'integer',
            'ip_pool' => 'array',
        ];
    }

    public function metrics()
    {
        return $this->hasMany(LocationMetric::class);
    }

    public function setupStatuses()
    {
        return $this->hasMany(LocationSetupStatus::class);
    }

    public function daemon()
    {
        return $this->hasOne(LocationDaemon::class);
    }

    public function tariffs(): HasMany
    {
        return $this->hasMany(Tariff::class);
    }
}
