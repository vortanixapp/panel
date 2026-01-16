<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('servers', function (Blueprint $table) {
            $table->unsignedInteger('slots')->nullable()->after('port');
            $table->decimal('cpu_cores', 6, 2)->nullable()->after('slots');
            $table->unsignedInteger('cpu_shares')->nullable()->after('cpu_cores');
            $table->unsignedInteger('ram_gb')->nullable()->after('cpu_shares');
            $table->unsignedInteger('disk_gb')->nullable()->after('ram_gb');

            $table->boolean('antiddos_enabled')->default(false)->after('disk_gb');

            $table->unsignedSmallInteger('server_fps')->nullable()->after('antiddos_enabled');
            $table->unsignedSmallInteger('server_tickrate')->nullable()->after('server_fps');
        });
    }

    public function down(): void
    {
        Schema::table('servers', function (Blueprint $table) {
            $table->dropColumn([
                'slots',
                'cpu_cores',
                'cpu_shares',
                'ram_gb',
                'disk_gb',
                'antiddos_enabled',
                'server_fps',
                'server_tickrate',
            ]);
        });
    }
};
