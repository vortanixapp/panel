<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    public function up(): void
    {
        $now = now();

        DB::table('games')->updateOrInsert(
            ['slug' => 'css'],
            [
                'name' => 'Counter-Strike: Source',
                'slug' => 'css',
                'code' => 'css',
                'query' => 'a2s',
                'description' => 'Counter-Strike: Source dedicated server.',
                'image' => null,
                'is_active' => true,
                'minport' => 27015,
                'maxport' => 27030,
                'status' => 1,
                'created_at' => $now,
                'updated_at' => $now,
            ]
        );

        DB::table('games')->updateOrInsert(
            ['slug' => 'cs2'],
            [
                'name' => 'Counter-Strike 2',
                'slug' => 'cs2',
                'code' => 'cs2',
                'query' => 'a2s',
                'description' => 'Counter-Strike 2 dedicated server.',
                'image' => null,
                'is_active' => true,
                'minport' => 27015,
                'maxport' => 27030,
                'status' => 1,
                'created_at' => $now,
                'updated_at' => $now,
            ]
        );
    }

    public function down(): void
    {
        DB::table('games')->whereIn('slug', ['css', 'cs2'])->delete();
    }
};
