<?php

namespace App\Models\Wormix;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @property int id
 * @property int user_id
 * @property int to_user_id
 * @property int action_type
 *
 * @property UserProfile user_profile
 * @property UserProfile to_user_profile
 */
class HouseAction extends Model
{
    protected $table = 'wormix_house_actions';

    public function to_user_profile() : BelongsTo
    {
        return $this->belongsTo(UserProfile::class, 'to_user_id', 'user_id');
    }

    public function user_profile() : BelongsTo
    {
        return $this->belongsTo(UserProfile::class, 'user_id', 'user_id');
    }

    protected $casts = [
        'action_type' => 'integer'
    ];
}
