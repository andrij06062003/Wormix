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
        Schema::create('wormix_craft', function (Blueprint $table) {
            $table->unsignedBigInteger('id')->primary();
            $table->bigInteger('upgrade_id')->unsigned()->unique('upgrade_idx');
            $table->bigInteger('prev_upgrade_id')->unsigned()->nullable();
            $table->string('description', 100)->nullable();
            $table->integer('required_level')->default(1);
            $table->integer('level');
            $table->json('reagents');
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('wormix_craft');
    }
};
