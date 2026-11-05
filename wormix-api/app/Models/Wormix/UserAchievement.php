<?php

namespace App\Models\Wormix;

use Illuminate\Database\Eloquent\Model;

/**
 * @property int id
 * @property int user_id
 * @property int achievement_id
 * @property int value
 */
class UserAchievement extends Model
{
    protected $table = 'wormix_users_achievements';

    protected $fillable = ['user_id', 'achievement_id', 'value'];
}
