<?php

namespace App\Models\Wormix;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @property int id
 * @property int owner_id
 * @property int weapon_id
 * @property int count
 * @property int expire_at
 *
 * @property Weapon weapon
 */
class UserWeapon extends Model
{
    protected $table = 'wormix_users_weapons';

    public function weapon() : BelongsTo
    {
        return $this->belongsTo(Weapon::class, 'weapon_id', 'id');
    }
}
