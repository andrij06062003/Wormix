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
        Schema::create('wormix_login_sequence', function (Blueprint $table) {
            $table->foreignId('user_id')->primary()->constrained('users')->cascadeOnDelete();
            $table->date('last_login')->nullable();

            $table->smallInteger('bonus_type')->default(0);
            $table->smallInteger('bonus_count')->default(0);

            $table->smallInteger('login_sequence')->unsigned()->default(1);
            $table->boolean('gift_accepted')->default(false);

            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('wormix_login_sequence');
    }
};
