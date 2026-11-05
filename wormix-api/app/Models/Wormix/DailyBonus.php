<?php

namespace App\Models\Wormix;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;

/**
 * @property int id
 *
 * @property int login_sequence
 *
 * @property int bonus_type
 * @property int bonus_value
 *
 * @property boolean random_gift
 * @property int rand_min
 * @property int rand_max
 */
class DailyBonus extends Model
{
    protected $table = 'daily_bonuses';
    protected $fillable = [
        'login_sequence',

        'bonus_type',
        'bonus_value',

        'random_gift',
        'rand_min',
        'rand_max'
    ];

    protected $casts = [
        'random_gift' => 'boolean'
    ];
}
