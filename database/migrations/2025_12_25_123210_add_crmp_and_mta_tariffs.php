<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    public function up(): void
    {
        $now = now();

        $locations = DB::table('locations')->select('id')->get();
        $games = DB::table('games')->whereIn('slug', ['crmp', 'mta'])->select('id', 'slug')->get();

        $gameIdsBySlug = [];
        foreach ($games as $g) {
            $gameIdsBySlug[(string) $g->slug] = (int) $g->id;
        }

        foreach ($locations as $loc) {
            $locationId = (int) $loc->id;

            foreach (['crmp', 'mta'] as $slug) {
                if (! isset($gameIdsBySlug[$slug])) {
                    continue;
                }

                $gameId = $gameIdsBySlug[$slug];
                $name = $slug === 'mta' ? 'MTA:SA - Default' : 'CRMP - Default';

                $exists = DB::table('tariffs')
                    ->where('location_id', $locationId)
                    ->where('game_id', $gameId)
                    ->exists();

                if ($exists) {
                    continue;
                }

                DB::table('tariffs')->insert([
                    'name' => $name,
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
                    'min_slots' => 1,
                    'max_slots' => 1000,
                    'cpu_cores' => 1,
                    'cpu_shares' => null,
                    'ram_gb' => 2,
                    'disk_gb' => 10,
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
    }

    public function down(): void
    {
        $games = DB::table('games')->whereIn('slug', ['crmp', 'mta'])->select('id')->get();
        $gameIds = $games->pluck('id')->map(fn ($v) => (int) $v)->all();

        if (! empty($gameIds)) {
            DB::table('tariffs')->whereIn('game_id', $gameIds)->delete();
        }
    }
};
