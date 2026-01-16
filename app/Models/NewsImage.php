<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class NewsImage extends Model
{
    protected $table = 'news_images';

    protected $fillable = [
        'news_id',
        'path',
        'sort',
    ];

    protected $casts = [
        'news_id' => 'integer',
        'sort' => 'integer',
    ];

    public function news(): BelongsTo
    {
        return $this->belongsTo(News::class, 'news_id');
    }
}
