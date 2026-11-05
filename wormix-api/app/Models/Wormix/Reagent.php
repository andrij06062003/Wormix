<?php

namespace App\Models\Wormix;

use Illuminate\Database\Eloquent\Model;

/**
 * @property int id
 * @property int reagent_id
 *
 * @property string name
 * @property string reagent_price
 */
class Reagent extends Model
{
    protected $table = 'wormix_reagents';

    protected $fillable = [
        'reagent_id',
        'name',
        'reagent_price'
    ];
}
