<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasTable('game_versions')) {
            return;
        }

        if (Schema::hasColumn('game_versions', 'archive_url')) {
            Schema::table('game_versions', function (Blueprint $table) {
                $table->dropColumn('archive_url');
            });
        }
    }

    public function down(): void
    {
        if (! Schema::hasTable('game_versions')) {
            return;
        }

        if (! Schema::hasColumn('game_versions', 'archive_url')) {
            Schema::table('game_versions', function (Blueprint $table) {
                $table->string('archive_url')->nullable()->after('source_type');
            });
        }
    }
};
