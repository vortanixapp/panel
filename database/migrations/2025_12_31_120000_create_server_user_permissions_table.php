<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('server_user_permissions', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('server_id');
            $table->unsignedBigInteger('user_id');

            $table->boolean('can_view_main')->default(true);
            $table->boolean('can_view_console')->default(false);
            $table->boolean('can_view_logs')->default(false);
            $table->boolean('can_view_metrics')->default(false);
            $table->boolean('can_view_ftp')->default(false);
            $table->boolean('can_view_mysql')->default(false);
            $table->boolean('can_view_cron')->default(false);
            $table->boolean('can_view_firewall')->default(false);
            $table->boolean('can_view_settings')->default(false);
            $table->boolean('can_view_friends')->default(false);

            $table->boolean('can_start')->default(false);
            $table->boolean('can_stop')->default(false);
            $table->boolean('can_restart')->default(false);
            $table->boolean('can_reinstall')->default(false);

            $table->boolean('can_console_command')->default(false);
            $table->boolean('can_files')->default(false);
            $table->boolean('can_cron_manage')->default(false);
            $table->boolean('can_firewall_manage')->default(false);
            $table->boolean('can_settings_edit')->default(false);

            $table->timestamps();

            $table->unique(['server_id', 'user_id']);

            $table->foreign('server_id')->references('id')->on('servers')->onDelete('cascade');
            $table->foreign('user_id')->references('id')->on('users')->onDelete('cascade');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('server_user_permissions');
    }
};
