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
        Schema::create('location_daemons', function (Blueprint $table) {
            $table->id();
            $table->foreignId('location_id')->constrained()->onDelete('cascade');
            $table->enum('status', ['online', 'offline', 'unknown'])->default('unknown');
            $table->string('version')->nullable();
            $table->integer('pid')->nullable();
            $table->float('uptime_sec')->nullable();
            $table->string('platform')->nullable();
            $table->timestamp('last_seen')->nullable();
            $table->timestamps();

            $table->unique('location_id');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('location_daemons');
    }
};
