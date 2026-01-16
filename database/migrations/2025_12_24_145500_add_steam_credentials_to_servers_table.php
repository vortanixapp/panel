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
        Schema::table('servers', function (Blueprint $table) {
            $table->string('steam_username')->nullable()->after('mysql_password');
            $table->text('steam_password')->nullable()->after('steam_username');
            $table->text('steam_guard_token')->nullable()->after('steam_password');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('servers', function (Blueprint $table) {
            $table->dropColumn([
                'steam_username',
                'steam_password',
                'steam_guard_token',
            ]);
        });
    }
};
