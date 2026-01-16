<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Map extends Model
{
    protected $fillable = [
        'name',
        'category',
        'slug',
        'version',
        'archive_path',
        'file_list',
        'restart_required',
        'active',
    ];

    protected $casts = [
        'file_list' => 'array',
        'restart_required' => 'boolean',
        'active' => 'boolean',
    ];

    public function serverMaps(): HasMany
    {
        return $this->hasMany(ServerMap::class);
    }
}
