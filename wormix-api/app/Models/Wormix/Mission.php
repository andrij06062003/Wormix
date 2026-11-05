<?php

namespace App\Models\Wormix;

use Illuminate\Database\Eloquent\Model;

/**
 * @property int id
 * @property int mission_id
 * @property int required_level
 * @property array awards
 */
class Mission extends Model
{
    protected $table = 'wormix_missions';

    protected $casts =[
        'awards' => 'array'
    ];
}
