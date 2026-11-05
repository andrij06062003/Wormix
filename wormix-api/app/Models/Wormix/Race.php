<?php

namespace App\Models\Wormix;

use Illuminate\Database\Eloquent\Model;

/**
 * @property int id
 * @property int race_id
 *
 * @property string race_name
 *
 * @property int price
 * @property int real_price
 *
 * @property int required_level
 */
class Race extends Model
{
    protected $table = 'wormix_races';

    protected $fillable = [
        'race_id',

        'race_name',

        'price',

        'real_price',
        'required_level'
    ];
}
