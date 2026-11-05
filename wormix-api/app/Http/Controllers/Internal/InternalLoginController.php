<?php

namespace App\Http\Controllers\Internal;

use App\Events\InternalLoginEvent;
use App\Http\Controllers\Controller;
use App\Http\Requests\Internal\Account\LoginRequest;
use App\Http\Resources\Internal\Account\EnterAccount;
use App\Http\Resources\Internal\Account\LoginError;
use App\Models\User;
use App\Modules\UserSession;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Event;
use Illuminate\Support\Facades\Log;

class InternalLoginController extends Controller
{

    public function login(LoginRequest $request)
    {
        try{
            $session = new UserSession($request->json(['Id']));

            if($session->getAuthKey() != $request->json('AuthKey') || $session->isLoggedIn()){
                return [
                    'type' => 'LoginError',
                    'data' => new LoginError($request, 1)
                ];
            }
            else{

                $old_session =  $session->getTcpSession();
                $session->setTcpSession($request->json('tcp_session'));
                Event::dispatch(new InternalLoginEvent($session->getSessionUser()));
                $session->loggedIn();
                return [
                    'type' => 'EnterAccount',
                    'old_session' => $old_session,
                    'data' => new  EnterAccount($session->getSessionUser(), $request->json('Id').'.'.$session->setSessionKey())
                ];
            }
        }catch (\Exception $ex){
            Log::error("Inernal login error", [
                'request' => $request->json()->all(),
                'message' => $ex->getMessage()
            ]);
            return [
                'type' => 'LoginError',
                'data' => new LoginError($request, 0)
            ];
        }
    }

}
