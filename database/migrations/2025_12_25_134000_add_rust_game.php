<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    public function up(): void
    {
        $now = now();

        DB::table('games')->updateOrInsert(
            ['slug' => 'rust'],
            [
                'name' => 'Rust',
                'slug' => 'rust',
                'code' => 'rust',
                'query' => null,
                'description' => 'Rust dedicated server.',
                'image' => null,
                'is_active' => true,
                'minport' => 28015,
                'maxport' => 28100,
                'status' => 1,
                'created_at' => $now,
                'updated_at' => $now,
            ]
        );
    }

    public function down(): void
    {
        DB::table('games')->where('slug', 'rust')->delete();
    }
};
