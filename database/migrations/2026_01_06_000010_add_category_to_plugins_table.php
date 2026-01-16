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
            if (! Schema::hasColumn('plugins', 'category')) {
                $table->string('category', 64)->nullable()->after('slug');
            }
        });
    }

    public function down(): void
    {
        if (! Schema::hasTable('plugins')) {
            return;
        }

        Schema::table('plugins', function (Blueprint $table) {
            if (Schema::hasColumn('plugins', 'category')) {
                $table->dropColumn('category');
            }
        });
    }
};
