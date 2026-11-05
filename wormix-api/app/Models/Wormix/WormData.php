<?php

namespace App\Models\Wormix;

use App\Models\User;
use App\Observers\Wormix\WormDataObserver;
use Illuminate\Database\Eloquent\Attributes\ObservedBy;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasOne;

/**
 * @property int owner_id
 *
 * @property int armor
 * @property int attack
 *
 * @property int level
 * @property int experience
 *
 * @property int hat
 *
 * @property User owner
 * @property Level level_model
 */
#[ObservedBy(WormDataObserver::class)]
class WormData extends Model
{
    protected $table = 'wormix_worms_data';

    protected $primaryKey = 'owner_id';

    protected $fillable = [
        'level',
        'armor',
        'attack'
    ];

    public function owner() : BelongsTo
    {
        return $this->belongsTo(User::class, 'owner_id', 'id');
    }

    public function level_model() : HasOne
    {
        return $this->hasOne(Level::class, 'id', 'level');
    }
}
