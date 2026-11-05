<?php

namespace App\Models\Wormix;

use Illuminate\Database\Eloquent\Model;

/**
 * @property int user_id
 *
 * @property int battles_count
 *
 * @property int current_battle_id
 * @property array awards
 * @property int battle_type
 *
 * @property int mission_id
 * @property int last_mission_id
 *
 * @property int last_boss_fight_time
 * @property int last_battle_time
 *
 */
class UserBattleInfo extends Model
{
    protected $table = 'wormix_users_battle_info';
    protected $primaryKey = 'user_id';

    protected $casts = [
        'awards' => 'array'
    ];
}
