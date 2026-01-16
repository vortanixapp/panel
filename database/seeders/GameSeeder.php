<?php

namespace Database\Seeders;

use App\Models\Game;
use Illuminate\Database\Console\Seeds\WithoutModelEvents;
use Illuminate\Database\Seeder;

class GameSeeder extends Seeder
{
    use WithoutModelEvents;

    /**
     * Run the database seeds.
     */
    public function run(): void
    {
        if (Game::where('slug', 'cs16')->exists()) {
            $this->command->info('CS 1.6 game already exists, skipping...');
            return;
        }

        Game::create([
            'name' => 'Counter-Strike 1.6',
            'slug' => 'cs16',
            'code' => 'cstrike',
            'query' => 'a2s',
            'description' => 'Classic Counter-Strike 1.6 multiplayer server with support for classic maps and game modes',
            'image' => 'games/cs16.png',
            'is_active' => true,
            'minport' => 27015,
            'maxport' => 27030,
            'price' => 0.00,
            'status' => 1,
        ]);

        $this->command->info('CS 1.6 game created successfully!');
    }
}
