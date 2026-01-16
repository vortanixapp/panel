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
            $table->string('code', 8)->nullable()->after('slug');
            $table->string('query', 8)->nullable()->after('code');
            $table->integer('minport')->default(1024)->after('query');
            $table->integer('maxport')->default(65535)->after('minport');
            $table->decimal('price', 10, 2)->default(0.00)->after('maxport');
            $table->tinyInteger('status')->default(1)->after('price');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('games', function (Blueprint $table) {
            $table->dropColumn(['code', 'query', 'minport', 'maxport', 'price', 'status']);
        });
    }
};
