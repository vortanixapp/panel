<?php

namespace App\Livewire;

use Livewire\Component;

class PricingCalculator extends Component
{
    public string $game = 'cs2';

    public int $slots = 32;

    public int $memory = 4;

    public string $region = 'eu';

    public function updatedSlots(): void
    {
        $this->slots = max(4, min(256, $this->slots));
    }

    public function updatedMemory(): void
    {
        $this->memory = max(1, min(32, $this->memory));
    }

    public function getPriceProperty(): int
    {
        $base = match ($this->game) {
            'cs2' => 180,
            'minecraft' => 220,
            'rust' => 260,
            default => 200,
        };

        $slotFactor = match ($this->game) {
            'cs2' => 3,
            'minecraft' => 2,
            'rust' => 4,
            default => 3,
        };

        $regionMultiplier = match ($this->region) {
            'ru' => 0.9,
            'eu' => 1.0,
            'asia' => 1.1,
            default => 1.0,
        };

        $slotsPrice = max(0, $this->slots - 10) * $slotFactor;

        $memoryPrice = max(0, $this->memory - 2) * 40;

        $price = (int) round(($base + $slotsPrice + $memoryPrice) * $regionMultiplier);

        return max(100, $price);
    }

    public function render()
    {
        return view('livewire.pricing-calculator');
    }
}
