<?php

namespace App\Models\Wormix;


use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @property int id
 * @property int user_id
 * @property int teammate_id
 * @property int order
 *
 * @property WormData owner
 * @property WormData teammate
 */
class UserTeam extends Model
{
    protected $table = 'wormix_users_teams';

    public function owner() : BelongsTo
    {
        return $this->belongsTo(WormData::class, 'user_id', 'owner_id');
    }

    public function teammate() : BelongsTo
    {
        return $this->belongsTo(WormData::class, 'teammate_id', 'owner_id');
    }

}
