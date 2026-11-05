<?php

use Illuminate\Foundation\Application;
use Illuminate\Foundation\Configuration\Exceptions;
use Illuminate\Foundation\Configuration\Middleware;
use Illuminate\Validation\ValidationException;
use Laravel\Sanctum\Http\Middleware\CheckAbilities;
use Laravel\Sanctum\Http\Middleware\CheckForAnyAbility;
use Symfony\Component\HttpKernel\Exception\AccessDeniedHttpException;
use Symfony\Component\HttpKernel\Exception\MethodNotAllowedHttpException;
use Symfony\Component\HttpKernel\Exception\NotFoundHttpException;

return Application::configure(basePath: dirname(__DIR__))
    ->withRouting(
        api: __DIR__.'/../routes/api.php'
        //commands: __DIR__.'/../routes/console.php',
        //health: '/up',
    )
    ->withMiddleware(function (Middleware $middleware) {
        $middleware->alias([
            'abilities' => CheckAbilities::class,
            'ability' => CheckForAnyAbility::class,
            'auth' => \App\Http\Middleware\ApiAuth::class,
            'internal-auth' => \App\Http\Middleware\InternalAuth::class,
            'internal-request' => \App\Http\Middleware\InternalRequest::class
        ]);
    })
    ->withExceptions(function (Exceptions $exceptions) {
        $exceptions->render(function (ValidationException $exception){
            return response([
                'message' => $exception->getMessage(),
                'errors' => $exception->errors()
            ], 422);
        });
        $exceptions->render(function (NotFoundHttpException $exception){
           return response([
               'message' => 'Not Found'
           ], 404);
        });
        $exceptions->render(function (MethodNotAllowedHttpException $exception){
            return response([
                'message' => 'Method Not Allowed'
            ], 405);
        });
        $exceptions->render(function (AccessDeniedHttpException $exception){
            return response([
                'message' => 'Access Denied'
            ], 403);
        });
    })->create();
