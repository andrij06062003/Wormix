<?php

namespace App\Http\Middleware;

use App\Modules\UserSession;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class InternalAuth
{
    /**
     * Handle an incoming request.
     *
     * @param  \Closure(\Illuminate\Http\Request): (\Symfony\Component\HttpFoundation\Response)  $next
     */
    public function handle(Request $request, Closure $next): Response
    {
        //Token format {user_id}.{session_key}
        $auth_key = $request->header('X-SESSION-KEY');

        if(!$auth_key)
            return $this->error();

        $auth_key = explode('.', $auth_key);

        if(count($auth_key) !== 2)
            return $this->error();

        try{
            $session = new UserSession((int)$auth_key[0]);
            if($session->getSessionKey() !== $auth_key[1])
                return $this->error();
        }
        catch (\Exception){
            return $this->error();
        }

        $request->json()->add(['internal_user_id' => (int)$auth_key[0]]);
        return $next($request);
    }

    public function error()
    {
        return \response([
            'message' => 'Access Denied'
        ], 403);
    }
}
