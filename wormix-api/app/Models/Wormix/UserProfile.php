<?php

namespace App\Models\Wormix;

use App\Models\User;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

/**
 * @property int user_id
 * @property int money
 * @property int real_money
 * @property int rating
 * @property int reaction_rate
 *
 * @property array reagents
 * @property array recipes
 *
 * @property HasMany weapons
 * @property User user
 * @property HasMany teammates
 */
class UserProfile extends Model
{
    protected $table = 'wormix_user_profiles';

    protected $primaryKey = 'user_id';

    protected $casts = [
        'reagents' => 'array',
        'recipes' => 'array'
    ];

    protected $fillable = [
        'money',
        'real_money',
        'rating',
        'reaction_rate'
    ];

    public function weapons() : HasMany
    {
        return $this->hasMany(UserWeapon::class, 'owner_id', 'user_id');
    }

    public function user() : BelongsTo
    {
        return $this->belongsTo(User::class, 'user_id', 'id');
    }

    public function teammates() : HasMany
    {
        return $this->hasMany(UserTeam::class, 'user_id', 'user_id');
    }
}
