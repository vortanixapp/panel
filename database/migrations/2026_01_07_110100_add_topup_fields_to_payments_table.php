<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('payments', function (Blueprint $table) {
            $table->string('promo_code')->nullable()->after('provider_order_id');
            $table->decimal('bonus_amount', 10, 2)->default(0)->after('promo_code');
            $table->decimal('credited_amount', 10, 2)->default(0)->after('bonus_amount');
            $table->unsignedInteger('payment_method_id')->nullable()->after('credited_amount');
        });
    }

    public function down(): void
    {
        Schema::table('payments', function (Blueprint $table) {
            $table->dropColumn(['promo_code', 'bonus_amount', 'credited_amount', 'payment_method_id']);
        });
    }
};
