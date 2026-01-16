<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (Schema::hasColumn('tariffs', 'prices')) {
            DB::table('tariffs')->update([
                'prices' => json_encode([]),
            ]);

            Schema::table('tariffs', function (Blueprint $table) {
                $table->dropColumn('prices');
            });
        }
    }

    public function down(): void
    {
        if (! Schema::hasColumn('tariffs', 'prices')) {
            Schema::table('tariffs', function (Blueprint $table) {
                $table->json('prices');
            });
        }
    }
};
