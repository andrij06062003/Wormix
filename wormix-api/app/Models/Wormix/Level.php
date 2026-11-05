<?php

namespace App\Models\Wormix;

use Illuminate\Database\Eloquent\Model;

/**
 * @property int id
 * @property int required_experience
 * @property int max_worms_count
 * @property array awards
 */
class Level extends Model
{
    protected $table = 'wormix_levels';

    protected $fillable = [
        'required_experience',
        'max_worms_count',
        'awards'
    ];

    protected $casts = [
        'awards' => 'json'
    ];
}
