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
        Schema::create('wormix_user_profiles', function (Blueprint $table) {
            $table->foreignId('user_id')->primary()->constrained('users')->cascadeOnDelete();

            $table->integer('money')->unsigned()->default(450)->comment('fuses');
            $table->integer('real_money')->unsigned()->default(3)->comment('rubies');

            $table->integer('rating')->unsigned()->default(0)->comment('user rating');
            $table->integer('reaction_rate')->unsigned()->default(0)->comment('user reaction rate');

            $table->json('reagents')->default("[]")->comment('user reagents');
            $table->json('recipes')->default("[]")->comment('user craft weapons');

            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('wormix_user_profiles');
    }
};
