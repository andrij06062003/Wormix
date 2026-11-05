<?php

namespace App\Models\Wormix;


use App\Observers\Wormix\LoginSequenceObserver;
use Illuminate\Database\Eloquent\Attributes\ObservedBy;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @property int user_id
 *
 * @property string last_login
 *
 * @property int bonus_type
 * @property int bonus_count
 *
 * @property int login_sequence
 * @property boolean gift_accepted
 */
#[ObservedBy(LoginSequenceObserver::class)]
class LoginSequence extends Model
{
    protected $table = 'wormix_login_sequence';

    protected $primaryKey = 'user_id';

    protected $fillable = [
        'last_login',
        'login_sequence'
    ];

    protected $casts = [
        'last_login' => 'datetime:Y-m-d',
        'gift_accepted' => 'boolean'
    ];
}
