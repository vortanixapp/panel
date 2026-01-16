<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    public function up(): void
    {
        $now = now();

        DB::table('games')->updateOrInsert(
            ['slug' => 'crmp'],
            [
                'name' => 'CRMP',
                'slug' => 'crmp',
                'code' => 'crmp',
                'query' => null,
                'description' => 'CRMP server (based on SA-MP).',
                'image' => null,
                'is_active' => true,
                'minport' => 7777,
                'maxport' => 7877,
                'status' => 1,
                'created_at' => $now,
                'updated_at' => $now,
            ]
        );

        DB::table('games')->updateOrInsert(
            ['slug' => 'mta'],
            [
                'name' => 'MTA:SA',
                'slug' => 'mta',
                'code' => 'mta',
                'query' => null,
                'description' => 'Multi Theft Auto: San Andreas dedicated server.',
                'image' => null,
                'is_active' => true,
                'minport' => 22003,
                'maxport' => 22100,
                'status' => 1,
                'created_at' => $now,
                'updated_at' => $now,
            ]
        );
    }

    public function down(): void
    {
        DB::table('games')->whereIn('slug', ['crmp', 'mta'])->delete();
    }
};
