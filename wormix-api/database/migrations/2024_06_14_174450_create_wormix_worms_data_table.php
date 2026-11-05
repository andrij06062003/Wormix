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
        Schema::create('wormix_worms_data', function (Blueprint $table) {
            $table->foreignId('owner_id')->primary()->constrained('users')->cascadeOnDelete();

            $table->smallInteger('armor')->default(1);
            $table->smallInteger('attack')->default(1);

            $table->smallInteger('level')->default(1);
            $table->smallInteger('experience')->default(0);

            $table->smallInteger('hat')->default(0);

            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('wormix_worms_data');
    }
};
