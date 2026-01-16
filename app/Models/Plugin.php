<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Plugin extends Model
{
    protected $fillable = [
        'name',
        'category',
        'slug',
        'version',
        'archive_type',
        'archive_path',
        'install_path',
        'supported_games',
        'file_actions',
        'uninstall_actions',
        'restart_required',
        'active',
    ];

    protected $casts = [
        'supported_games' => 'array',
        'file_actions' => 'array',
        'uninstall_actions' => 'array',
        'restart_required' => 'boolean',
        'active' => 'boolean',
    ];

    public function serverPlugins(): HasMany
    {
        return $this->hasMany(ServerPlugin::class);
    }
}
