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
        Schema::create('server_plugins', function (Blueprint $table) {
            $table->id();

            $table->foreignId('server_id')->constrained('servers')->cascadeOnDelete();
            $table->foreignId('plugin_id')->constrained('plugins')->cascadeOnDelete();

            $table->boolean('installed')->default(false);
            $table->boolean('enabled')->default(true);
            $table->timestamp('installed_at')->nullable();
            $table->text('last_error')->nullable();

            $table->timestamps();

            $table->unique(['server_id', 'plugin_id']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('server_plugins');
    }
};
