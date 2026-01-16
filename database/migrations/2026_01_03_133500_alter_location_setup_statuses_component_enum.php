<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        DB::statement("ALTER TABLE `location_setup_statuses` MODIFY `component` ENUM('packages','docker','mysql','phpmyadmin','ftp','daemon','images') NOT NULL");
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        DB::statement("ALTER TABLE `location_setup_statuses` MODIFY `component` ENUM('packages','docker','mysql','ftp','daemon') NOT NULL");
    }
};
