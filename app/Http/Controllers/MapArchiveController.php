<?php

namespace App\Http\Controllers;

use App\Models\Map;
use Illuminate\Support\Facades\Storage;
use Symfony\Component\HttpFoundation\BinaryFileResponse;

class MapArchiveController extends Controller
{
    public function download(Map $map): BinaryFileResponse
    {
        $path = (string) $map->archive_path;
        $disk = Storage::disk('local');
        if ($path === '' || ! $disk->exists($path)) {
            abort(404);
        }

        $filename = basename($path);

        return response()->download($disk->path($path), $filename);
    }
}
