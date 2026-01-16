<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

class News extends Model
{
    protected $table = 'news';

    protected $fillable = [
        'title',
        'slug',
        'excerpt',
        'body',
        'published_at',
        'active',
    ];

    protected $casts = [
        'published_at' => 'datetime',
        'active' => 'boolean',
    ];

    public function images(): HasMany
    {
        return $this->hasMany(NewsImage::class, 'news_id')->orderBy('sort')->orderBy('id');
    }
}
