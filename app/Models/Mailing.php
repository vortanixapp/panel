<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

class Mailing extends Model
{
    protected $fillable = [
        'title',
        'status',
        'channels',
        'audience',
        'subject',
        'body',
        'is_html',
        'scheduled_at',
        'started_at',
        'finished_at',
        'total_recipients',
        'sent_count',
        'failed_count',
        'skipped_count',
        'last_error',
    ];

    protected $casts = [
        'channels' => 'array',
        'audience' => 'array',
        'is_html' => 'boolean',
        'scheduled_at' => 'datetime',
        'started_at' => 'datetime',
        'finished_at' => 'datetime',
        'total_recipients' => 'integer',
        'sent_count' => 'integer',
        'failed_count' => 'integer',
        'skipped_count' => 'integer',
    ];
}
