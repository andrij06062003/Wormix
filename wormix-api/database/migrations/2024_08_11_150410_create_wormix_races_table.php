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
        Schema::create('wormix_races', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('race_id')->unique('r_game_idx');

            $table->string('race_name', 20);

            $table->integer('price')->default(0);
            $table->integer('real_price')->default(0);

            $table->integer('required_level')->default(0);

            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('wormix_races');
    }
};
