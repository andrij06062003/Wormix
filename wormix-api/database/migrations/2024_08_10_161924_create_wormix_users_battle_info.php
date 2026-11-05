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
        Schema::create('wormix_users_battle_info', function (Blueprint $table) {
            $table->foreignId('user_id')->primary()->constrained('users')->cascadeOnDelete();


            $table->integer('battles_count')->unsigned()->default(10);

            $table->json('awards')->nullable();
            $table->bigInteger('current_battle_id')->default(0);
            $table->tinyInteger('battle_type')->default(0)->comment('0: default battle, 1: mission, 2: PvP');

            $table->integer('mission_id')->default(0);
            $table->integer('last_mission_id')->default(-1);

            $table->integer('last_boss_fight_time')->unsigned()->default(0);
            $table->integer('last_battle_time')->unsigned()->default(0);
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('wormix_users_battle_info');
    }
};
