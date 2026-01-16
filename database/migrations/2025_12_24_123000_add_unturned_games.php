<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        $rows = [
            [
                'name' => 'Unturned (Vanilla)',
                'slug' => 'unturned',
                'code' => 'unturn',
                'query' => 'a2s',
                'description' => 'Unturned dedicated server (no mods)',
                'image' => 'games/unturned.png',
                'is_active' => true,
                'minport' => 27015,
                'maxport' => 27060,
                'status' => 1,
            ],
            [
                'name' => 'Unturned (RocketMod 4)',
                'slug' => 'untrm4',
                'code' => 'untrm4',
                'query' => 'a2s',
                'description' => 'Unturned dedicated server with RocketMod 4',
                'image' => 'games/unturned.png',
                'is_active' => true,
                'minport' => 27015,
                'maxport' => 27060,
                'status' => 1,
            ],
            [
                'name' => 'Unturned (RocketMod 5)',
                'slug' => 'untrm5',
                'code' => 'untrm5',
                'query' => 'a2s',
                'description' => 'Unturned dedicated server with RocketMod 5',
                'image' => 'games/unturned.png',
                'is_active' => true,
                'minport' => 27015,
                'maxport' => 27060,
                'status' => 1,
            ],
        ];

        foreach ($rows as $row) {
            DB::table('games')->updateOrInsert(
                ['slug' => $row['slug']],
                [
                    ...$row,
                    'created_at' => now(),
                    'updated_at' => now(),
                ]
            );
        }
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        DB::table('games')->whereIn('slug', ['unturned', 'untrm4', 'untrm5'])->delete();
    }
};
