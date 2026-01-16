<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('users', function (Blueprint $table) {
            $table->string('last_name')->nullable()->after('name');
            $table->string('public_id')->nullable()->unique()->after('id');
            $table->unsignedInteger('bonuses')->default(0)->after('password');
            $table->string('phone')->nullable()->after('email');
            $table->string('telegram_id')->nullable()->after('phone');
            $table->string('discord_id')->nullable()->after('telegram_id');
            $table->string('vk_id')->nullable()->after('discord_id');
        });
    }

    public function down(): void
    {
        Schema::table('users', function (Blueprint $table) {
            $table->dropColumn([
                'last_name',
                'public_id',
                'bonuses',
                'phone',
                'telegram_id',
                'discord_id',
                'vk_id',
            ]);
        });
    }
};
