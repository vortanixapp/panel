<?php

namespace App\Http\Controllers;

use App\Models\Plugin;
use Illuminate\Support\Facades\Storage;
use Symfony\Component\HttpFoundation\BinaryFileResponse;

class PluginArchiveController extends Controller
{
    public function download(Plugin $plugin): BinaryFileResponse
    {
        $path = (string) $plugin->archive_path;
        $disk = Storage::disk('local');
        if ($path === '' || ! $disk->exists($path)) {
            abort(404);
        }

        $filename = basename($path);

        return response()->download($disk->path($path), $filename);
    }
}
