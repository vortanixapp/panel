<?php

namespace App\Http\Controllers;

use Illuminate\View\View;

class FeaturesController extends Controller
{
    public function __invoke(): View
    {
        return view('features');
    }
}
