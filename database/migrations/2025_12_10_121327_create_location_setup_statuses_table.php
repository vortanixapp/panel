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
        Schema::create('location_setup_statuses', function (Blueprint $table) {
            $table->id();
            $table->foreignId('location_id')->constrained()->onDelete('cascade');
            $table->enum('component', ['packages', 'docker', 'mysql', 'ftp', 'daemon']);
            $table->enum('status', ['pending', 'installing', 'installed', 'failed'])->default('pending');
            $table->timestamp('installed_at')->nullable();
            $table->text('error_message')->nullable();
            $table->timestamps();

            $table->unique(['location_id', 'component']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('location_setup_statuses');
    }
};
