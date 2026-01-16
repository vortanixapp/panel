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
        Schema::table('locations', function (Blueprint $table) {
            $table->string('ssh_host')->nullable()->after('description');
            $table->string('ssh_user')->nullable()->after('ssh_host');
            $table->string('ssh_password')->nullable()->after('ssh_user');
            $table->unsignedInteger('ssh_port')->default(22)->after('ssh_password');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('locations', function (Blueprint $table) {
            $table->dropColumn(['ssh_host', 'ssh_user', 'ssh_password', 'ssh_port']);
        });
    }
};
