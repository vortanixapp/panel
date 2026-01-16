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
            $table->string('ftp_host')->nullable()->after('runtime_status');
            $table->unsignedInteger('ftp_port')->nullable()->after('ftp_host');
            $table->string('ftp_username')->nullable()->after('ftp_port');
            $table->text('ftp_password')->nullable()->after('ftp_username');
            $table->string('ftp_root')->nullable()->after('ftp_password');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('servers', function (Blueprint $table) {
            $table->dropColumn([
                'ftp_host',
                'ftp_port',
                'ftp_username',
                'ftp_password',
                'ftp_root',
            ]);
        });
    }
};
