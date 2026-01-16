<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasTable('plugins')) {
            return;
        }

        Schema::table('plugins', function (Blueprint $table) {
            if (! Schema::hasColumn('plugins', 'file_actions')) {
                $table->json('file_actions')->nullable()->after('supported_games');
            }
        });
    }

    public function down(): void
    {
        if (! Schema::hasTable('plugins')) {
            return;
        }

        Schema::table('plugins', function (Blueprint $table) {
            if (Schema::hasColumn('plugins', 'file_actions')) {
                $table->dropColumn('file_actions');
            }
        });
    }
};
