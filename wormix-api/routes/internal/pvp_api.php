<?php

use App\Http\Controllers\Internal\PvpController;
use Illuminate\Support\Facades\Route;

// Called by the game server (wormix-core) for PvP battles, acting as one player each time.
Route::post('profile', [PvpController::class, 'profile']);
Route::post('start_battle', [PvpController::class, 'startBattle']);
Route::post('end_battle', [PvpController::class, 'endBattle']);
