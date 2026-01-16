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
            $table->string('container_id')->nullable()->after('status');
            $table->string('container_name')->nullable()->after('container_id');
            $table->string('provisioning_status')->nullable()->after('container_name');
            $table->text('provisioning_error')->nullable()->after('provisioning_status');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('servers', function (Blueprint $table) {
            $table->dropColumn([
                'container_id',
                'container_name',
                'provisioning_status',
                'provisioning_error',
            ]);
        });
    }
};
