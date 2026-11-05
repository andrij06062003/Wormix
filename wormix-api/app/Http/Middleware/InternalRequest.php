<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Log;
use Symfony\Component\HttpFoundation\Response;

class InternalRequest
{
    /**
     * Handle an incoming request.
     *
     * @param  \Closure(\Illuminate\Http\Request): (\Symfony\Component\HttpFoundation\Response)  $next
     */
    public function handle(Request $request, Closure $next): Response
    {
        $tcp_session = $request->header('X-TCP-SESSION');
        if($tcp_session == null){
            return \response([
               'message' => 'Not found'
            ], 404);
        }
        $request->json()->add(['tcp_session' => $request->header('X-TCP-SESSION')]);
        return $next($request);
    }
}
