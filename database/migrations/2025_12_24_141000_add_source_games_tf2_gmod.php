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
                'name' => 'Team Fortress 2',
                'slug' => 'tf2',
                'code' => 'tf2',
                'query' => 'a2s',
                'description' => 'Team Fortress 2 dedicated server (Source)',
                'image' => 'games/tf2.png',
                'is_active' => true,
                'minport' => 27015,
                'maxport' => 27060,
                'status' => 1,
            ],
            [
                'name' => "Garry's Mod",
                'slug' => 'gmod',
                'code' => 'gmod',
                'query' => 'a2s',
                'description' => "Garry's Mod dedicated server (Source)",
                'image' => 'games/gmod.png',
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
        DB::table('games')->whereIn('slug', ['tf2', 'gmod'])->delete();
    }
};
