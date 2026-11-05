<?php

namespace App\Http\Controllers\Internal;

use App\Http\Controllers\Controller;
use App\Http\Requests\Internal\Info\GetWhoPumpedReactionRequest;
use App\Http\Resources\Internal\Info\WhoPumpedReactionResult;
use App\Models\User;
use Illuminate\Http\Request;

class InfoController extends Controller
{
    public function getRating()
    {

    }

    public function getPumpedReaction(GetWhoPumpedReactionRequest $request)
    {
        return new WhoPumpedReactionResult(
            User::query()
                ->where('id', $request->json('internal_user_id'))
                ->get()
                ->first()
        );
    }
}
