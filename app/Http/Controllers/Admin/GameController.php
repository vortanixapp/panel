<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Game;
use App\Models\GameVersion;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\View\View;

class GameController extends Controller
{
    public function index(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $games = Game::orderByDesc('created_at')->paginate(10);

        return view('admin.games', [
            'games' => $games,
        ]);
    }

    public function create(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return view('admin.games.create');
    }

    public function store(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'name' => ['required', 'string', 'min:2', 'max:32'],
            'slug' => ['required', 'string', 'max:255', 'unique:games,slug'],
            'description' => ['nullable', 'string'],
            'image' => ['nullable', 'string', 'max:255'],
            'is_active' => ['sometimes', 'boolean'],
            'code' => ['required', 'string', 'min:2', 'max:8'],
            'query' => ['required', 'string', 'min:2', 'max:8'],
            'minport' => ['required', 'integer', 'min:1', 'max:65535'],
            'maxport' => ['required', 'integer', 'min:1', 'max:65535'],
            'status' => ['required', 'integer', 'in:0,1'],
        ]);

        Game::create([
            'name' => $validated['name'],
            'slug' => $validated['slug'],
            'description' => $validated['description'],
            'image' => $validated['image'],
            'is_active' => $validated['is_active'],
            'code' => $validated['code'],
            'query' => $validated['query'],
            'minport' => $validated['minport'],
            'maxport' => $validated['maxport'],
            'status' => $validated['status'],
        ]);

        return redirect()->route('admin.games');
    }

    public function show(Game $game): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('admin.games');
        }

        return view('admin.games.show', [
            'game' => $game,
        ]);
    }

    public function storeVersion(Request $request, Game $game): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('admin.games');
        }

        $validated = $request->validate([
            'name' => ['required', 'string', 'max:64'],
            'source_type' => ['required', 'string', 'in:archive,steam'],
            'url' => ['nullable', 'string', 'max:2048'],
            'steam_app_id' => ['nullable', 'integer', 'min:1'],
            'steam_branch' => ['nullable', 'string', 'max:64'],
            'is_active' => ['sometimes', 'boolean'],
            'sort_order' => ['nullable', 'integer', 'min:0'],
        ]);

        $sourceType = (string) $validated['source_type'];
        $archiveUrl = trim((string) $validated['url']);
        $steamAppId = (int) $validated['steam_app_id'];
        $steamBranch = trim((string) $validated['steam_branch']);

        if ($sourceType === 'archive' && $archiveUrl === '') {
            return redirect()->route('admin.games.edit', ['game' => $game, 'tab' => 'versions'])->with('error', 'Для archive версии нужна ссылка (archive_url)');
        }
        if ($sourceType === 'archive' && $archiveUrl !== '' && ! str_ends_with(strtolower($archiveUrl), '.zip')) {
            return redirect()->route('admin.games.edit', ['game' => $game, 'tab' => 'versions'])->with('error', 'Archive версия должна быть .zip');
        }
        if ($sourceType === 'steam' && (! $steamAppId || $steamAppId <= 0)) {
            return redirect()->route('admin.games.edit', ['game' => $game, 'tab' => 'versions'])->with('error', 'Для steam версии нужен Steam App ID');
        }

        GameVersion::create([
            'game_id' => $game->id,
            'name' => $validated['name'],
            'source_type' => $sourceType,
            'steam_app_id' => $sourceType === 'steam' ? $steamAppId : null,
            'steam_branch' => $sourceType === 'steam' && $steamBranch !== '' ? $steamBranch : null,
            'url' => $sourceType === 'archive' ? $archiveUrl : ('steam:' . (string) $steamAppId),
            'is_active' => $validated['is_active'],
            'sort_order' => $validated['sort_order'],
        ]);

        return redirect()->route('admin.games.edit', ['game' => $game, 'tab' => 'versions']);
    }

    public function destroyVersion(Game $game, GameVersion $version): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('admin.games');
        }

        if ((int) $version->game_id !== (int) $game->id) {
            return redirect()->route('admin.games.edit', ['game' => $game, 'tab' => 'versions']);
        }

        $version->delete();

        return redirect()->route('admin.games.edit', ['game' => $game, 'tab' => 'versions']);
    }

    public function edit(Game $game): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return view('admin.games.edit', [
            'game' => $game->load(['versions' => function ($q) {
                $q->orderBy('sort_order')->orderBy('id');
            }]),
        ]);
    }

    public function update(Request $request, Game $game): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'name' => ['required', 'string', 'min:2', 'max:32'],
            'slug' => ['required', 'string', 'max:255', 'unique:games,slug,' . $game->id],
            'description' => ['nullable', 'string'],
            'image' => ['nullable', 'string', 'max:255'],
            'is_active' => ['sometimes', 'boolean'],
            'code' => ['required', 'string', 'min:2', 'max:8'],
            'query' => ['required', 'string', 'min:2', 'max:8'],
            'minport' => ['required', 'integer', 'min:1', 'max:65535'],
            'maxport' => ['required', 'integer', 'min:1', 'max:65535'],
            'status' => ['required', 'integer', 'in:0,1'],
        ]);

        $game->update([
            'name' => $validated['name'],
            'slug' => $validated['slug'],
            'description' => $validated['description'],
            'image' => $validated['image'],
            'is_active' => $validated['is_active'],
            'code' => $validated['code'],
            'query' => $validated['query'],
            'minport' => $validated['minport'],
            'maxport' => $validated['maxport'],
            'status' => $validated['status'],
        ]);

        return redirect()->route('admin.games');
    }
    
    public function destroy(Game $game): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $game->delete();

        return redirect()->route('admin.games');
    }
}
