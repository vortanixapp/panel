<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class MailingDelivery extends Model
{
    protected $fillable = [
        'mailing_id',
        'user_id',
        'channel',
        'address',
        'status',
        'attempts',
        'sent_at',
        'error',
    ];

    protected $casts = [
        'mailing_id' => 'integer',
        'user_id' => 'integer',
        'attempts' => 'integer',
        'sent_at' => 'datetime',
    ];

    public function mailing(): BelongsTo
    {
        return $this->belongsTo(Mailing::class);
    }

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
