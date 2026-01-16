<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    public function up(): void
    {
        $now = now();

        $rust = DB::table('games')->where('slug', 'rust')->select('id')->first();
        if (! $rust) {
            return;
        }

        $gameId = (int) $rust->id;
        $locations = DB::table('locations')->select('id')->get();

        foreach ($locations as $loc) {
            $locationId = (int) $loc->id;

            $exists = DB::table('tariffs')
                ->where('location_id', $locationId)
                ->where('game_id', $gameId)
                ->exists();

            if ($exists) {
                continue;
            }

            DB::table('tariffs')->insert([
                'name' => 'Rust - Default',
                'location_id' => $locationId,
                'game_id' => $gameId,
                'billing_type' => 'resources',
                'price_per_slot' => null,
                'price_per_cpu_core' => 0,
                'price_per_ram_gb' => 0,
                'price_per_disk_gb' => 0,
                'cpu_min' => null,
                'cpu_max' => null,
                'cpu_step' => null,
                'ram_min' => null,
                'ram_max' => null,
                'ram_step' => null,
                'disk_min' => null,
                'disk_max' => null,
                'disk_step' => null,
                'allow_antiddos' => false,
                'antiddos_price' => 0,
                'min_slots' => 10,
                'max_slots' => 200,
                'cpu_cores' => 3,
                'cpu_shares' => null,
                'ram_gb' => 10,
                'disk_gb' => 25,
                'rental_periods' => json_encode([15, 30, 60, 180]),
                'renewal_periods' => json_encode([15, 30, 60, 180]),
                'discounts' => null,
                'position' => 0,
                'is_available' => true,
                'created_at' => $now,
                'updated_at' => $now,
            ]);
        }
    }

    public function down(): void
    {
        $rust = DB::table('games')->where('slug', 'rust')->select('id')->first();
        if (! $rust) {
            return;
        }

        DB::table('tariffs')->where('game_id', (int) $rust->id)->delete();
    }
};
