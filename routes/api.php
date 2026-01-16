<?php

use App\Http\Controllers\Api\LocationController;
use App\Http\Controllers\Api\LocationServersController;
use App\Http\Controllers\Api\MetricController;
use App\Http\Controllers\Api\ServerRuntimeStatusController;
use App\Http\Controllers\Payments\FreekassaResultController;
use App\Http\Controllers\Payments\NowPaymentsIpnController;
use Illuminate\Support\Facades\Route;

Route::post('/metrics', [MetricController::class, 'store']);

Route::get('/locations/{code}/ssh-data', [LocationController::class, 'sshData']);

Route::get('/locations/{code}/servers', [LocationServersController::class, 'index']);

Route::post('/servers/runtime-status', [ServerRuntimeStatusController::class, 'store']);

Route::post('/payments/freekassa/result', FreekassaResultController::class);

Route::post('/payments/nowpayments/ipn', NowPaymentsIpnController::class);
