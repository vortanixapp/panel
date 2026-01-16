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
        DB::table('games')->insert([
            'name' => 'Counter-Strike 1.6',
            'slug' => 'cs16',
            'code' => 'cstrike',
            'query' => 'a2s',
            'description' => 'Classic Counter-Strike 1.6 multiplayer server',
            'image' => 'games/cs16.png',
            'is_active' => true,
            'minport' => 27015,
            'maxport' => 27030,
            'status' => 1,
            'created_at' => now(),
            'updated_at' => now(),
        ]);
    }


    public function down(): void
    {
        DB::table('games')->where('slug', 'cs16')->delete();
    }
};
