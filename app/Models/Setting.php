<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Facades\Cache;

class Setting extends Model
{
    protected $fillable = [
        'key',
        'value',
    ];

    public static function allCached(): array
    {
        return Cache::rememberForever('settings.all', function () {
            return self::query()->pluck('value', 'key')->toArray();
        });
    }

    public static function getValue(string $key, mixed $default = null): mixed
    {
        $all = self::allCached();
        if (array_key_exists($key, $all)) {
            return $all[$key];
        }
        return $default;
    }

    public static function setValue(string $key, mixed $value): void
    {
        self::query()->updateOrCreate(
            ['key' => $key],
            ['value' => $value === null ? null : (string) $value]
        );

        Cache::forget('settings.all');
    }

    protected static function booted(): void
    {
        static::saved(function () {
            Cache::forget('settings.all');
        });

        static::deleted(function () {
            Cache::forget('settings.all');
        });
    }
}
