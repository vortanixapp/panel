<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('game_versions', function (Blueprint $table) {
            $table->string('source_type')->default('archive')->after('name');
            $table->string('archive_url')->nullable()->after('source_type');
            $table->unsignedBigInteger('steam_app_id')->nullable()->after('archive_url');
            $table->string('steam_branch')->nullable()->after('steam_app_id');
        });

        if (Schema::hasColumn('game_versions', 'url')) {
            DB::table('game_versions')->whereNull('archive_url')->update([
                'archive_url' => DB::raw('`url`'),
            ]);
        }
    }

    public function down(): void
    {
        Schema::table('game_versions', function (Blueprint $table) {
            $table->dropColumn(['source_type', 'archive_url', 'steam_app_id', 'steam_branch']);
        });
    }
};
