<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('promotions', function (Blueprint $table) {
            $table->id();
            $table->string('title');
            $table->string('code')->nullable()->unique();
            $table->boolean('is_active')->default(true);

            $table->timestamp('starts_at')->nullable();
            $table->timestamp('ends_at')->nullable();

            $table->json('applies_to')->nullable();

            $table->string('discount_type', 16)->nullable();
            $table->decimal('discount_value', 10, 2)->default(0);

            $table->decimal('bonus_percent', 5, 2)->default(0);
            $table->decimal('bonus_fixed', 10, 2)->default(0);

            $table->unsignedInteger('max_uses')->nullable();
            $table->unsignedInteger('used_count')->default(0);

            $table->decimal('min_amount', 10, 2)->nullable();

            $table->boolean('only_new_users')->default(false);

            $table->json('user_ids')->nullable();
            $table->json('tariff_ids')->nullable();
            $table->json('game_ids')->nullable();
            $table->json('location_ids')->nullable();

            $table->text('description')->nullable();

            $table->timestamps();

            $table->index(['is_active', 'starts_at', 'ends_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('promotions');
    }
};
