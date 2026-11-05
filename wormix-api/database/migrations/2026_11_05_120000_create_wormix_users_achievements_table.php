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
        // One row per player and achievement: the counter the client increases (kills, visits...).
        Schema::create('wormix_users_achievements', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained('users')->cascadeOnDelete();
            $table->unsignedTinyInteger('achievement_id');
            $table->unsignedInteger('value')->default(0);
            $table->timestamps();

            $table->unique(['user_id', 'achievement_id']);
        });

        // How many award items (one per 1000 achievement points) the player has already taken.
        Schema::table('wormix_user_profiles', function (Blueprint $table) {
            $table->unsignedTinyInteger('invested_award_points')->default(0);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('wormix_user_profiles', function (Blueprint $table) {
            $table->dropColumn('invested_award_points');
        });
        Schema::dropIfExists('wormix_users_achievements');
    }
};
