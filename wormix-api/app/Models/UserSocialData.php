<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

/**
 * @property int user_id
 * @property string first_name
 * @property string last_name
 * @property string nickname
 * @property string photo
 */
class UserSocialData extends Model
{
    protected $table = 'users_vk_data';

    protected $primaryKey = 'user_id';

    protected $fillable = [
        'first_name',
        'last_name',
        'nickname'
    ];

}
