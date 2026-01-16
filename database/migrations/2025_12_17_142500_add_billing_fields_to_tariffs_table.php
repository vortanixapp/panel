<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('tariffs', function (Blueprint $table) {
            $table->string('billing_type', 16)->default('resources')->after('game_id');
            $table->decimal('price_per_slot', 10, 2)->nullable()->after('billing_type');

            $table->decimal('price_per_cpu_core', 10, 2)->default(0)->after('price_per_slot');
            $table->decimal('price_per_ram_gb', 10, 2)->default(0)->after('price_per_cpu_core');
            $table->decimal('price_per_disk_gb', 10, 2)->default(0)->after('price_per_ram_gb');

            $table->unsignedInteger('cpu_min')->nullable()->after('price_per_disk_gb');
            $table->unsignedInteger('cpu_max')->nullable()->after('cpu_min');
            $table->unsignedInteger('cpu_step')->nullable()->after('cpu_max');

            $table->unsignedInteger('ram_min')->nullable()->after('cpu_step');
            $table->unsignedInteger('ram_max')->nullable()->after('ram_min');
            $table->unsignedInteger('ram_step')->nullable()->after('ram_max');

            $table->unsignedInteger('disk_min')->nullable()->after('ram_step');
            $table->unsignedInteger('disk_max')->nullable()->after('disk_min');
            $table->unsignedInteger('disk_step')->nullable()->after('disk_max');

            $table->boolean('allow_antiddos')->default(false)->after('disk_step');
            $table->decimal('antiddos_price', 10, 2)->default(0)->after('allow_antiddos');
        });
    }

    public function down(): void
    {
        Schema::table('tariffs', function (Blueprint $table) {
            $table->dropColumn([
                'billing_type',
                'price_per_slot',
                'price_per_cpu_core',
                'price_per_ram_gb',
                'price_per_disk_gb',
                'cpu_min',
                'cpu_max',
                'cpu_step',
                'ram_min',
                'ram_max',
                'ram_step',
                'disk_min',
                'disk_max',
                'disk_step',
                'allow_antiddos',
                'antiddos_price',
            ]);
        });
    }
};
