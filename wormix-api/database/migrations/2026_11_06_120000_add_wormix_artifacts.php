<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

/**
 * Artifacts (ported from the 1.25.0 client): an off-hand item, equipped in its own slot next to the hat.
 *  - wormix_worms_data.artifact: the equipped artifact's item id (2001+), 0 = none;
 *  - the artifacts themselves are items in wormix_weapons (ids 2001+), from resources/game/artifacts.json,
 *    so buying and owning them works like hats. game:init only runs on a new database, so they are added here.
 */
return new class extends Migration
{
    public function up(): void
    {
        if (!Schema::hasColumn('wormix_worms_data', 'artifact')) {
            Schema::table('wormix_worms_data', function (Blueprint $table) {
                $table->smallInteger('artifact')->default(0);
            });
        }

        $artifacts = json_decode(file_get_contents(resource_path('game/artifacts.json')), true);
        $existing = DB::table('wormix_weapons')->pluck('id')->all();
        $now = now();
        foreach ($artifacts as $artifact) {
            if (in_array($artifact['id'], $existing))
                continue;
            DB::table('wormix_weapons')->insert([
                'id' => $artifact['id'],
                'name' => $artifact['name'],
                'hide_in_shop' => !empty($artifact['hideInShop']),
                'is_starter' => false,
                'price' => $artifact['price'] ?? 0,
                'real_price' => $artifact['realprice'] ?? 0,
                'required_level' => $artifact['requiredLevel'] ?? 0,
                'required_rating' => $artifact['requiredRating'] ?? 0,
                // Bought for good, like hats. The boss-drop ones (with "duration") get an expiry when dropped.
                'infinity' => true,
                'one_day' => false,
                'created_at' => $now,
                'updated_at' => $now,
            ]);
        }
    }

    public function down(): void
    {
        $ids = array_column(json_decode(file_get_contents(resource_path('game/artifacts.json')), true), 'id');
        DB::table('wormix_users_weapons')->whereIn('weapon_id', $ids)->delete();
        DB::table('wormix_weapons')->whereIn('id', $ids)->delete();
        if (Schema::hasColumn('wormix_worms_data', 'artifact')) {
            Schema::table('wormix_worms_data', function (Blueprint $table) {
                $table->dropColumn('artifact');
            });
        }
    }
};
