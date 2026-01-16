<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    public function up(): void
    {
        $now = now();

        DB::table('games')->updateOrInsert(
            ['slug' => 'samp'],
            [
                'name' => 'SA-MP',
                'slug' => 'samp',
                'code' => 'samp',
                'query' => 'samp',
                'description' => 'San Andreas Multiplayer dedicated server.',
                'image' => null,
                'is_active' => true,
                'minport' => 7777,
                'maxport' => 7877,
                'status' => 1,
                'created_at' => $now,
                'updated_at' => $now,
            ]
        );
    }

    public function down(): void
    {
        DB::table('games')->where('slug', 'samp')->delete();
    }
};
