<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('server_maps', function (Blueprint $table) {
            $table->id();

            $table->foreignId('server_id')->constrained('servers')->cascadeOnDelete();
            $table->foreignId('map_id')->constrained('maps')->cascadeOnDelete();

            $table->boolean('installed')->default(false);
            $table->timestamp('installed_at')->nullable();
            $table->text('last_error')->nullable();

            $table->timestamps();

            $table->unique(['server_id', 'map_id']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('server_maps');
    }
};
