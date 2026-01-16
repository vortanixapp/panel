<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('tariffs', function (Blueprint $table) {
            $table->id();
            $table->string('name');
            $table->foreignId('location_id')->constrained()->onDelete('cascade');
            $table->foreignId('game_id')->constrained()->onDelete('cascade');
            $table->integer('min_slots')->default(1);
            $table->integer('max_slots')->default(100);
            $table->integer('min_ports')->default(1024);
            $table->integer('max_ports')->default(65535);
            $table->integer('cpu_cores')->default(1);
            $table->integer('ram_gb')->default(1);
            $table->integer('disk_gb')->default(10);
            $table->json('rental_periods')->nullable(); // array of days: [15,30,60,180]
            $table->json('renewal_periods')->nullable(); // array of days
            $table->json('discounts')->nullable(); // object or array
            $table->integer('position')->default(0);
            $table->boolean('is_available')->default(true);
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('tariffs');
    }
};
