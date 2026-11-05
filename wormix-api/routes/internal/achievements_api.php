<?php

use App\Http\Controllers\Internal\AchievementsController;
use Illuminate\Support\Facades\Route;

// Called by the relay, which decodes the client's binary achievement commands.
// Auth is the achievement session from `login` (see AchievementsController).
Route::post('login', [AchievementsController::class, 'login']);
Route::post('get', [AchievementsController::class, 'get']);
Route::post('increase', [AchievementsController::class, 'increase']);
Route::post('choose_bonus', [AchievementsController::class, 'chooseBonus']);
