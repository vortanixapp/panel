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
            if (! Schema::hasColumn('plugins', 'uninstall_actions')) {
                $table->json('uninstall_actions')->nullable()->after('file_actions');
            }
        });
    }

    public function down(): void
    {
        if (! Schema::hasTable('plugins')) {
            return;
        }

        Schema::table('plugins', function (Blueprint $table) {
            if (Schema::hasColumn('plugins', 'uninstall_actions')) {
                $table->dropColumn('uninstall_actions');
            }
        });
    }
};
