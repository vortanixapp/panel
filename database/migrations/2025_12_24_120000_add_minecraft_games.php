<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;
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
                'name' => 'Minecraft Java (Vanilla)',
                'slug' => 'mcjava',
                'code' => 'mcjava',
                'query' => 'mc',
                'description' => 'Minecraft Java Edition (Vanilla) server',
                'image' => 'games/mcjava.png',
                'is_active' => true,
                'minport' => 25565,
                'maxport' => 25600,
                'status' => 1,
            ],
            [
                'name' => 'Minecraft Java (Paper)',
                'slug' => 'mcpaper',
                'code' => 'mcpaper',
                'query' => 'mc',
                'description' => 'Minecraft Java Edition (Paper) server',
                'image' => 'games/mcjava.png',
                'is_active' => true,
                'minport' => 25565,
                'maxport' => 25600,
                'status' => 1,
            ],
            [
                'name' => 'Minecraft Java (Spigot)',
                'slug' => 'mcspigot',
                'code' => 'mcspigot',
                'query' => 'mc',
                'description' => 'Minecraft Java Edition (Spigot) server',
                'image' => 'games/mcjava.png',
                'is_active' => true,
                'minport' => 25565,
                'maxport' => 25600,
                'status' => 1,
            ],
            [
                'name' => 'Minecraft Java (Forge)',
                'slug' => 'mcforge',
                'code' => 'mcforge',
                'query' => 'mc',
                'description' => 'Minecraft Java Edition (Forge) server',
                'image' => 'games/mcjava.png',
                'is_active' => true,
                'minport' => 25565,
                'maxport' => 25600,
                'status' => 1,
            ],
            [
                'name' => 'Minecraft Java (Fabric)',
                'slug' => 'mcfabric',
                'code' => 'mcfabric',
                'query' => 'mc',
                'description' => 'Minecraft Java Edition (Fabric) server',
                'image' => 'games/mcjava.png',
                'is_active' => true,
                'minport' => 25565,
                'maxport' => 25600,
                'status' => 1,
            ],
            [
                'name' => 'Minecraft Bedrock',
                'slug' => 'mcbedrock',
                'code' => 'bedrock',
                'query' => 'mc',
                'description' => 'Minecraft Bedrock Edition server',
                'image' => 'games/mcbedrock.png',
                'is_active' => true,
                'minport' => 19132,
                'maxport' => 19160,
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
        DB::table('games')->whereIn('slug', ['mcjava', 'mcpaper', 'mcspigot', 'mcforge', 'mcfabric', 'mcbedrock'])->delete();
    }
};
