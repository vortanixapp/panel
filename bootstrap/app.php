<?php

use Illuminate\Foundation\Application;
use Illuminate\Foundation\Configuration\Exceptions;
use Illuminate\Foundation\Configuration\Middleware;

return Application::configure(basePath: dirname(__DIR__))
    ->withRouting(
        web: __DIR__.'/../routes/web.php',
        api: __DIR__.'/../routes/api.php',
        commands: __DIR__.'/../routes/console.php',
        health: '/up',
    )
    ->withMiddleware(function (Middleware $middleware): void {
        $middleware->appendToGroup('web', \App\Http\Middleware\SecurityHeaders::class);
        $middleware->appendToGroup('web', \App\Http\Middleware\SetLocale::class);
        $middleware->appendToGroup('web', \App\Http\Middleware\EnsureValidLicense::class);
    })
    ->withSchedule(function ($schedule): void {
        $schedule->command('metrics:pull-daemon-all')->everyMinute();
        $schedule->command('servers:pull-daemon-status-all')->everyMinute();
        $schedule->command('servers:stop-expired')->everyMinute();
        $schedule->command('mailings:dispatch-scheduled')->everyMinute();
        $schedule->command('integrity:check')->everyMinute();
    })
    ->withExceptions(function (Exceptions $exceptions): void {
        //
    })->create();
