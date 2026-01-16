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
            $table->string('mysql_host')->nullable()->after('ftp_root');
            $table->unsignedInteger('mysql_port')->nullable()->after('mysql_host');
            $table->string('mysql_database')->nullable()->after('mysql_port');
            $table->string('mysql_username')->nullable()->after('mysql_database');
            $table->text('mysql_password')->nullable()->after('mysql_username');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('servers', function (Blueprint $table) {
            $table->dropColumn([
                'mysql_host',
                'mysql_port',
                'mysql_database',
                'mysql_username',
                'mysql_password',
            ]);
        });
    }
};
