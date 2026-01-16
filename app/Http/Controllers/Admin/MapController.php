<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Map;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Storage;
use Illuminate\Support\Str;
use Illuminate\View\View;

class MapController extends Controller
{
    public function index(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $maps = Map::query()
            ->orderBy('category')
            ->orderBy('name')
            ->paginate(20);

        return view('admin.maps.index', [
            'maps' => $maps,
        ]);
    }

    public function create(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return view('admin.maps.create');
    }

    private function buildFileListFromZip(string $absPath): array
    {
        if (! file_exists($absPath)) {
            throw new \RuntimeException('Архив не найден: ' . $absPath);
        }

        $zip = new \ZipArchive();
        $ok = $zip->open($absPath);
        if ($ok !== true) {
            throw new \RuntimeException('Не удалось открыть zip архив (code=' . (string) $ok . ')');
        }

        $files = [];
        for ($i = 0; $i < $zip->numFiles; $i++) {
            $stat = $zip->statIndex($i);
            if (! is_array($stat)) {
                continue;
            }
            $name = (string) $stat['name'];
            $name = str_replace('\\', '/', $name);
            $name = ltrim($name, '/');
            if ($name === '' || str_ends_with($name, '/')) {
                continue;
            }
            if (str_contains($name, '..')) {
                continue;
            }
            $parts = array_values(array_filter(explode('/', $name), fn ($p) => $p !== '' && $p !== '.' && $p !== '..'));
            if (count($parts) === 0) {
                continue;
            }
            $safe = implode('/', $parts);
            $files[$safe] = true;
        }

        $zip->close();

        $list = array_keys($files);
        sort($list, SORT_NATURAL | SORT_FLAG_CASE);
        return $list;
    }

    public function store(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'name' => ['required', 'string', 'min:2', 'max:128'],
            'category' => ['nullable', 'string', 'max:64'],
            'slug' => ['required', 'string', 'max:128', 'unique:maps,slug'],
            'version' => ['nullable', 'string', 'max:64'],
            'archive' => ['required', 'file', 'max:512000'],
            'restart_required' => ['sometimes', 'boolean'],
            'active' => ['sometimes', 'boolean'],
        ]);

        $slug = (string) $validated['slug'];
        $archive = $request->file('archive');
        if (! $archive || ! $archive->isValid()) {
            return back()->withErrors(['archive' => 'Некорректный файл'])->withInput();
        }

        $ext = strtolower((string) $archive->getClientOriginalExtension());
        if ($ext !== 'zip') {
            return back()->withErrors(['archive' => 'Поддерживается только zip'])->withInput();
        }

        $safeName = Str::slug(pathinfo((string) $archive->getClientOriginalName(), PATHINFO_FILENAME));
        $safeName = $safeName !== '' ? $safeName : 'archive';

        $storedPath = $archive->storeAs(
            'maps/' . $slug,
            date('YmdHis') . '-' . $safeName . '.zip'
        );

        if (! is_string($storedPath) || $storedPath === '') {
            return back()->withErrors(['archive' => 'Не удалось сохранить архив'])->withInput();
        }

        $abs = Storage::disk('local')->path($storedPath);
        $fileList = $this->buildFileListFromZip($abs);

        Map::create([
            'name' => $validated['name'],
            'category' => (string) $validated['category'],
            'slug' => $slug,
            'version' => $validated['version'],
            'archive_path' => $storedPath,
            'file_list' => $fileList,
            'restart_required' => (bool) $validated['restart_required'],
            'active' => (bool) $validated['active'],
        ]);

        return redirect()->route('admin.maps.index')->with('success', 'Карта добавлена');
    }

    public function edit(Map $map): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return view('admin.maps.edit', [
            'map' => $map,
        ]);
    }

    public function show(Map $map): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return redirect()->route('admin.maps.edit', $map);
    }

    public function update(Request $request, Map $map): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'name' => ['required', 'string', 'min:2', 'max:128'],
            'category' => ['nullable', 'string', 'max:64'],
            'slug' => ['required', 'string', 'max:128', 'unique:maps,slug,' . $map->id],
            'version' => ['nullable', 'string', 'max:64'],
            'archive' => ['nullable', 'file', 'max:512000'],
            'restart_required' => ['sometimes', 'boolean'],
            'active' => ['sometimes', 'boolean'],
        ]);

        $slug = (string) $validated['slug'];

        $storedPath = (string) $map->archive_path;
        $fileList = (array) $map->file_list;

        $archive = $request->file('archive');
        if ($archive && $archive->isValid()) {
            $ext = strtolower((string) $archive->getClientOriginalExtension());
            if ($ext !== 'zip') {
                return back()->withErrors(['archive' => 'Поддерживается только zip'])->withInput();
            }

            $safeName = Str::slug(pathinfo((string) $archive->getClientOriginalName(), PATHINFO_FILENAME));
            $safeName = $safeName !== '' ? $safeName : 'archive';

            $storedPath = $archive->storeAs(
                'maps/' . $slug,
                date('YmdHis') . '-' . $safeName . '.zip'
            );

            if (! is_string($storedPath) || $storedPath === '') {
                return back()->withErrors(['archive' => 'Не удалось сохранить архив'])->withInput();
            }

            $abs = Storage::disk('local')->path($storedPath);
            $fileList = $this->buildFileListFromZip($abs);
        }

        $map->update([
            'name' => $validated['name'],
            'category' => (string) $validated['category'],
            'slug' => $slug,
            'version' => $validated['version'],
            'archive_path' => $storedPath !== '' ? $storedPath : null,
            'file_list' => $fileList,
            'restart_required' => (bool) $validated['restart_required'],
            'active' => (bool) $validated['active'],
        ]);

        return redirect()->route('admin.maps.index')->with('success', 'Карта обновлена');
    }

    public function destroy(Map $map): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $map->delete();

        return redirect()->route('admin.maps.index')->with('success', 'Карта удалена');
    }
}
