<?php

namespace App\Models;

use App\Models\Wormix\HouseAction;
use App\Models\Wormix\LoginSequence;
use App\Models\Wormix\UserBattleInfo;
use App\Models\Wormix\UserProfile;
use App\Models\Wormix\WormData;
use App\Observers\UserObserver;
use Illuminate\Database\Eloquent\Attributes\ObservedBy;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\Relations\HasOne;
use Illuminate\Foundation\Auth\User as Authenticatable;
use Illuminate\Notifications\Notifiable;
use Laravel\Sanctum\HasApiTokens;

/**
 * @property int id
 * @property string login
 * @property string password
 *
 * @property UserProfile user_profile
 * @property WormData worm_data
 * @property UserSocialData social_data
 * @property UserBattleInfo battle_info
 * @property LoginSequence login_sequence
 */
#[ObservedBy(UserObserver::class)]
class User extends Authenticatable
{
    use Notifiable, HasApiTokens;

    protected $fillable = [
        'login',
        'password'
    ];

    protected $hidden = [
        'password'
    ];

    protected $casts = [
        'password' => 'hashed',
    ];

    public function user_profile() : HasOne
    {
        return $this->hasOne(UserProfile::class, 'user_id', 'id');
    }

    public function worm_data() : HasOne
    {
        return $this->hasOne(WormData::class, 'owner_id', 'id');
    }

    public function social_data(): HasOne
    {
        return $this->hasOne(UserSocialData::class, 'user_id', 'id');
    }

    public function battle_info() : HasOne
    {
        return $this->hasOne(UserBattleInfo::class, 'user_id', 'id');
    }

    public function login_sequence() : HasOne
    {
        return $this->hasOne(LoginSequence::class, 'user_id', 'id');
    }

    public function house_actions() : HasMany
    {
        return $this->hasMany(HouseAction::class, 'to_user_id', 'id');
    }
}
