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
        Schema::create('daily_bonuses', function (Blueprint $table) {
            $table->id();
            $table->smallInteger('login_sequence')->default(1);
            $table->smallInteger('bonus_type')->default(0);
            $table->smallInteger('bonus_value')->default(0);
            $table->boolean('random_gift')->default(false);
            $table->smallInteger('rand_min')->default(1);
            $table->smallInteger('rand_max')->default(10);
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('daily_bonuses');
    }
};
