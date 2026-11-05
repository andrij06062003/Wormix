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
        Schema::create('wormix_reagents', function (Blueprint $table) {
            $table->id();
            $table->bigInteger('reagent_id')->unsigned()->unique('reagent_idx');
            $table->string('name', 100)->nullable();
            $table->bigInteger('reagent_price')->default(0);
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('wormix_reagents');
    }
};
