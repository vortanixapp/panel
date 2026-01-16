<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class ServerUserPermission extends Model
{
    protected $table = 'server_user_permissions';

    protected $fillable = [
        'server_id',
        'user_id',

        'can_view_main',
        'can_view_console',
        'can_view_logs',
        'can_view_metrics',
        'can_view_ftp',
        'can_view_mysql',
        'can_view_cron',
        'can_view_firewall',
        'can_view_settings',
        'can_view_friends',

        'can_start',
        'can_stop',
        'can_restart',
        'can_reinstall',

        'can_console_command',
        'can_files',
        'can_cron_manage',
        'can_firewall_manage',
        'can_settings_edit',
    ];

    protected $casts = [
        'can_view_main' => 'boolean',
        'can_view_console' => 'boolean',
        'can_view_logs' => 'boolean',
        'can_view_metrics' => 'boolean',
        'can_view_ftp' => 'boolean',
        'can_view_mysql' => 'boolean',
        'can_view_cron' => 'boolean',
        'can_view_firewall' => 'boolean',
        'can_view_settings' => 'boolean',
        'can_view_friends' => 'boolean',

        'can_start' => 'boolean',
        'can_stop' => 'boolean',
        'can_restart' => 'boolean',
        'can_reinstall' => 'boolean',

        'can_console_command' => 'boolean',
        'can_files' => 'boolean',
        'can_cron_manage' => 'boolean',
        'can_firewall_manage' => 'boolean',
        'can_settings_edit' => 'boolean',
    ];

    public function server(): BelongsTo
    {
        return $this->belongsTo(Server::class);
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
