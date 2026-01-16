<?php

namespace App\Http\Controllers;

use Illuminate\View\View;

class GamesController extends Controller
{
    public function index(): View
    {
        return view('games');
    }

    public function show(string $slug): View
    {
        $view = match ($slug) {
            'minecraft'   => 'games.minecraft',
            'cs2-csgo'    => 'games.cs2-csgo',
            'rust'        => 'games.rust',
            'valheim'     => 'games.valheim',
            'automation'  => 'games.automation',
            'custom'      => 'games.custom',
            default       => null,
        };

        if (! $view) {
            abort(404);
        }

        return view($view);
    }
}
