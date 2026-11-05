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
        Schema::create('wormix_levels', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('required_experience')->default(0);
            $table->unsignedInteger('max_worms_count')->default(1);
            $table->json('awards')->nullable();
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('wormix_levels');
    }
};
