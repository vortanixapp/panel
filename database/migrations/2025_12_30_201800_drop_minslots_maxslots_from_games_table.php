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
        Schema::table('games', function (Blueprint $table) {
            $columnsToDrop = [];

            if (Schema::hasColumn('games', 'minslots')) {
                $columnsToDrop[] = 'minslots';
            }

            if (Schema::hasColumn('games', 'maxslots')) {
                $columnsToDrop[] = 'maxslots';
            }

            if ($columnsToDrop !== []) {
                $table->dropColumn($columnsToDrop);
            }
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('games', function (Blueprint $table) {
            if (! Schema::hasColumn('games', 'minslots')) {
                $table->integer('minslots')->default(1)->after('query');
            }

            if (! Schema::hasColumn('games', 'maxslots')) {
                $table->integer('maxslots')->default(100)->after('minslots');
            }
        });
    }
};
