<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Game;
use App\Models\Plugin;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Str;
use Illuminate\View\View;

class PluginController extends Controller
{
    public function index(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $plugins = Plugin::query()
            ->orderBy('category')
            ->orderBy('name')
            ->paginate(20);

        return view('admin.plugins.index', [
            'plugins' => $plugins,
        ]);
    }

    public function create(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $games = Game::query()->orderBy('name')->get();

        return view('admin.plugins.create', [
            'games' => $games,
        ]);
    }

    public function store(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'name' => ['required', 'string', 'min:2', 'max:128'],
            'category' => ['nullable', 'string', 'max:64'],
            'slug' => ['required', 'string', 'max:128', 'unique:plugins,slug'],
            'version' => ['nullable', 'string', 'max:64'],
            'archive_type' => ['required', 'string', 'in:zip,tar,targz'],
            'archive' => ['nullable', 'file', 'max:512000'],
            'install_path' => ['required', 'string', 'max:255'],
            'all_games' => ['sometimes', 'boolean'],
            'supported_games_codes' => ['nullable', 'array'],
            'supported_games_codes.*' => ['nullable', 'string', 'max:32'],
            'file_action_path' => ['nullable', 'array'],
            'file_action_path.*' => ['nullable', 'string', 'max:255'],
            'file_action_action' => ['nullable', 'array'],
            'file_action_action.*' => ['nullable', 'string', 'in:ensure_contains,append_lines,prepend_lines,remove_lines,write_file,replace_regex'],
            'file_action_create_if_missing' => ['nullable', 'array'],
            'file_action_create_if_missing.*' => ['nullable'],
            'file_action_pattern' => ['nullable', 'array'],
            'file_action_pattern.*' => ['nullable', 'string', 'max:500'],
            'file_action_replacement' => ['nullable', 'array'],
            'file_action_replacement.*' => ['nullable', 'string', 'max:500'],
            'file_action_lines' => ['nullable', 'array'],
            'file_action_lines.*' => ['nullable', 'string'],
            'uninstall_action_path' => ['nullable', 'array'],
            'uninstall_action_path.*' => ['nullable', 'string', 'max:255'],
            'uninstall_action_action' => ['nullable', 'array'],
            'uninstall_action_action.*' => ['nullable', 'string', 'in:ensure_contains,append_lines,prepend_lines,remove_lines,write_file,replace_regex'],
            'uninstall_action_create_if_missing' => ['nullable', 'array'],
            'uninstall_action_create_if_missing.*' => ['nullable'],
            'uninstall_action_pattern' => ['nullable', 'array'],
            'uninstall_action_pattern.*' => ['nullable', 'string', 'max:500'],
            'uninstall_action_replacement' => ['nullable', 'array'],
            'uninstall_action_replacement.*' => ['nullable', 'string', 'max:500'],
            'uninstall_action_lines' => ['nullable', 'array'],
            'uninstall_action_lines.*' => ['nullable', 'string'],
            'restart_required' => ['sometimes', 'boolean'],
            'active' => ['sometimes', 'boolean'],
        ]);

        $installPathRaw = trim((string) $validated['install_path']);
        $installPath = trim($installPathRaw, " \t\n\r\0\x0B/");
        if ($installPath !== '' && str_contains($installPath, '..')) {
            return back()->withErrors(['install_path' => 'Некорректный путь установки'])->withInput();
        }

        $supportedGames = null;
        $allGames = (bool) $validated['all_games'];
        $codes = is_array($validated['supported_games_codes']) ? $validated['supported_games_codes'] : [];
        $codes = array_values(array_filter(array_map(fn ($v) => strtolower(trim((string) $v)), $codes), fn ($v) => $v !== '' && $v !== '*'));
        $codes = array_values(array_unique($codes));
        if ($allGames) {
            $supportedGames = ['*'];
        } elseif (count($codes) > 0) {
            $supportedGames = $codes;
        }

        $fileActions = null;
        $paths = is_array($validated['file_action_path']) ? $validated['file_action_path'] : [];
        $texts = is_array($validated['file_action_lines']) ? $validated['file_action_lines'] : [];
        $actionsRaw = is_array($validated['file_action_action']) ? $validated['file_action_action'] : [];
        $createRaw = is_array($validated['file_action_create_if_missing']) ? $validated['file_action_create_if_missing'] : [];
        $patterns = is_array($validated['file_action_pattern']) ? $validated['file_action_pattern'] : [];
        $replacements = is_array($validated['file_action_replacement']) ? $validated['file_action_replacement'] : [];
        $actions = [];
        foreach ($paths as $i => $pRaw) {
            $p = trim((string) $pRaw);
            if ($p === '' || str_contains($p, '..')) {
                continue;
            }

            $action = strtolower(trim((string) $actionsRaw[$i]));
            if (! in_array($action, ['ensure_contains', 'append_lines', 'prepend_lines', 'remove_lines', 'write_file', 'replace_regex'], true)) {
                $action = 'ensure_contains';
            }

            $createIfMissing = (string) $createRaw[$i];
            $createIfMissingBool = $createIfMissing === '1' || strtolower($createIfMissing) === 'true' || $createIfMissing === 'on';

            $t = (string) $texts[$i];
            $lines = preg_split('/\r\n|\r|\n/', $t);
            if (! is_array($lines)) {
                $lines = [];
            }
            $lines = array_values(array_filter(array_map(fn ($l) => rtrim((string) $l, "\r\n"), $lines), fn ($l) => trim($l) !== ''));

            if ($action === 'replace_regex') {
                $pattern = (string) $patterns[$i];
                $replacement = (string) $replacements[$i];
                if (trim($pattern) === '') {
                    continue;
                }
                $actions[] = [
                    'path' => ltrim(str_replace('\\\\', '/', $p), '/'),
                    'action' => 'replace_regex',
                    'create_if_missing' => $createIfMissingBool,
                    'pattern' => $pattern,
                    'replacement' => $replacement,
                ];
                continue;
            }

            if (count($lines) === 0) {
                continue;
            }

            $actions[] = [
                'path' => ltrim(str_replace('\\', '/', $p), '/'),
                'action' => $action,
                'create_if_missing' => $createIfMissingBool,
                'lines' => $lines,
            ];
        }

        if (count($actions) > 0) {
            $fileActions = $actions;
        }

        $uninstallActions = null;
        $uPaths = is_array($validated['uninstall_action_path']) ? $validated['uninstall_action_path'] : [];
        $uTexts = is_array($validated['uninstall_action_lines']) ? $validated['uninstall_action_lines'] : [];
        $uActionsRaw = is_array($validated['uninstall_action_action']) ? $validated['uninstall_action_action'] : [];
        $uCreateRaw = is_array($validated['uninstall_action_create_if_missing']) ? $validated['uninstall_action_create_if_missing'] : [];
        $uPatterns = is_array($validated['uninstall_action_pattern']) ? $validated['uninstall_action_pattern'] : [];
        $uReplacements = is_array($validated['uninstall_action_replacement']) ? $validated['uninstall_action_replacement'] : [];
        $uList = [];
        foreach ($uPaths as $i => $pRaw) {
            $p = trim((string) $pRaw);
            if ($p === '' || str_contains($p, '..')) {
                continue;
            }

            $action = strtolower(trim((string) $uActionsRaw[$i]));
            if (! in_array($action, ['ensure_contains', 'append_lines', 'prepend_lines', 'remove_lines', 'write_file', 'replace_regex'], true)) {
                $action = 'ensure_contains';
            }

            $createIfMissing = (string) $uCreateRaw[$i];
            $createIfMissingBool = $createIfMissing === '1' || strtolower($createIfMissing) === 'true' || $createIfMissing === 'on';

            $t = (string) $uTexts[$i];
            $lines = preg_split('/\r\n|\r|\n/', $t);
            if (! is_array($lines)) {
                $lines = [];
            }
            $lines = array_values(array_filter(array_map(fn ($l) => rtrim((string) $l, "\r\n"), $lines), fn ($l) => trim($l) !== ''));

            if ($action === 'replace_regex') {
                $pattern = (string) $uPatterns[$i];
                $replacement = (string) $uReplacements[$i];
                if (trim($pattern) === '') {
                    continue;
                }
                $uList[] = [
                    'path' => ltrim(str_replace('\\\\', '/', $p), '/'),
                    'action' => 'replace_regex',
                    'create_if_missing' => $createIfMissingBool,
                    'pattern' => $pattern,
                    'replacement' => $replacement,
                ];
                continue;
            }

            if (count($lines) === 0) {
                continue;
            }

            $uList[] = [
                'path' => ltrim(str_replace('\\', '/', $p), '/'),
                'action' => $action,
                'create_if_missing' => $createIfMissingBool,
                'lines' => $lines,
            ];
        }
        if (count($uList) > 0) {
            $uninstallActions = $uList;
        }

        $slug = (string) $validated['slug'];
        $archive = $request->file('archive');
        $storedPath = null;
        if ($archive) {
            $ext = $archive->getClientOriginalExtension();
            $ext = $ext !== '' ? ('.' . $ext) : '';
            $safeName = Str::slug(pathinfo((string) $archive->getClientOriginalName(), PATHINFO_FILENAME));
            $safeName = $safeName !== '' ? $safeName : 'archive';

            $storedPath = $archive->storeAs(
                'plugins/' . $slug,
                date('YmdHis') . '-' . $safeName . $ext
            );

            if (! is_string($storedPath) || $storedPath === '') {
                return back()->withErrors(['archive' => 'Не удалось сохранить архив'])->withInput();
            }
        }

        Plugin::create([
            'name' => $validated['name'],
            'category' => (string) $validated['category'],
            'slug' => $slug,
            'version' => $validated['version'],
            'archive_type' => $validated['archive_type'],
            'archive_path' => $storedPath,
            'install_path' => $installPath,
            'supported_games' => $supportedGames,
            'file_actions' => $fileActions,
            'uninstall_actions' => $uninstallActions,
            'restart_required' => (bool) $validated['restart_required'],
            'active' => (bool) $validated['active'],
        ]);

        return redirect()->route('admin.plugins.index')->with('success', 'Плагин добавлен');
    }

    public function edit(Plugin $plugin): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }
        $games = Game::query()->orderBy('name')->get();

        return view('admin.plugins.edit', [
            'plugin' => $plugin,
            'games' => $games,
        ]);
    }

    public function show(Plugin $plugin): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return redirect()->route('admin.plugins.edit', $plugin);
    }

    public function update(Request $request, Plugin $plugin): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'name' => ['required', 'string', 'min:2', 'max:128'],
            'category' => ['nullable', 'string', 'max:64'],
            'slug' => ['required', 'string', 'max:128', 'unique:plugins,slug,' . $plugin->id],
            'version' => ['nullable', 'string', 'max:64'],
            'archive_type' => ['required', 'string', 'in:zip,tar,targz'],
            'archive' => ['nullable', 'file', 'max:512000'],
            'install_path' => ['required', 'string', 'max:255'],
            'all_games' => ['sometimes', 'boolean'],
            'supported_games_codes' => ['nullable', 'array'],
            'supported_games_codes.*' => ['nullable', 'string', 'max:32'],
            'file_action_path' => ['nullable', 'array'],
            'file_action_path.*' => ['nullable', 'string', 'max:255'],
            'file_action_action' => ['nullable', 'array'],
            'file_action_action.*' => ['nullable', 'string', 'in:ensure_contains,append_lines,prepend_lines,remove_lines,write_file,replace_regex'],
            'file_action_create_if_missing' => ['nullable', 'array'],
            'file_action_create_if_missing.*' => ['nullable'],
            'file_action_pattern' => ['nullable', 'array'],
            'file_action_pattern.*' => ['nullable', 'string', 'max:500'],
            'file_action_replacement' => ['nullable', 'array'],
            'file_action_replacement.*' => ['nullable', 'string', 'max:500'],
            'file_action_lines' => ['nullable', 'array'],
            'file_action_lines.*' => ['nullable', 'string'],
            'uninstall_action_path' => ['nullable', 'array'],
            'uninstall_action_path.*' => ['nullable', 'string', 'max:255'],
            'uninstall_action_action' => ['nullable', 'array'],
            'uninstall_action_action.*' => ['nullable', 'string', 'in:ensure_contains,append_lines,prepend_lines,remove_lines,write_file,replace_regex'],
            'uninstall_action_create_if_missing' => ['nullable', 'array'],
            'uninstall_action_create_if_missing.*' => ['nullable'],
            'uninstall_action_pattern' => ['nullable', 'array'],
            'uninstall_action_pattern.*' => ['nullable', 'string', 'max:500'],
            'uninstall_action_replacement' => ['nullable', 'array'],
            'uninstall_action_replacement.*' => ['nullable', 'string', 'max:500'],
            'uninstall_action_lines' => ['nullable', 'array'],
            'uninstall_action_lines.*' => ['nullable', 'string'],
            'restart_required' => ['sometimes', 'boolean'],
            'active' => ['sometimes', 'boolean'],
        ]);

        $installPathRaw = trim((string) $validated['install_path']);
        $installPath = trim($installPathRaw, " \t\n\r\0\x0B/");
        if ($installPath !== '' && str_contains($installPath, '..')) {
            return back()->withErrors(['install_path' => 'Некорректный путь установки'])->withInput();
        }

        $supportedGames = null;
        $allGames = (bool) $validated['all_games'];
        $codes = is_array($validated['supported_games_codes']) ? $validated['supported_games_codes'] : [];
        $codes = array_values(array_filter(array_map(fn ($v) => strtolower(trim((string) $v)), $codes), fn ($v) => $v !== '' && $v !== '*'));
        $codes = array_values(array_unique($codes));
        if ($allGames) {
            $supportedGames = ['*'];
        } elseif (count($codes) > 0) {
            $supportedGames = $codes;
        }

        $fileActions = null;
        $paths = is_array($validated['file_action_path']) ? $validated['file_action_path'] : [];
        $texts = is_array($validated['file_action_lines']) ? $validated['file_action_lines'] : [];
        $actionsRaw = is_array($validated['file_action_action']) ? $validated['file_action_action'] : [];
        $createRaw = is_array($validated['file_action_create_if_missing']) ? $validated['file_action_create_if_missing'] : [];
        $patterns = is_array($validated['file_action_pattern']) ? $validated['file_action_pattern'] : [];
        $replacements = is_array($validated['file_action_replacement']) ? $validated['file_action_replacement'] : [];
        $actions = [];
        foreach ($paths as $i => $pRaw) {
            $p = trim((string) $pRaw);
            if ($p === '' || str_contains($p, '..')) {
                continue;
            }

            $action = strtolower(trim((string) $actionsRaw[$i]));
            if (! in_array($action, ['ensure_contains', 'append_lines', 'prepend_lines', 'remove_lines', 'write_file', 'replace_regex'], true)) {
                $action = 'ensure_contains';
            }

            $createIfMissing = (string) $createRaw[$i];
            $createIfMissingBool = $createIfMissing === '1' || strtolower($createIfMissing) === 'true' || $createIfMissing === 'on';

            $t = (string) $texts[$i];
            $lines = preg_split('/\r\n|\r|\n/', $t);
            if (! is_array($lines)) {
                $lines = [];
            }
            $lines = array_values(array_filter(array_map(fn ($l) => rtrim((string) $l, "\r\n"), $lines), fn ($l) => trim($l) !== ''));

            if ($action === 'replace_regex') {
                $pattern = (string) $patterns[$i];
                $replacement = (string) $replacements[$i];
                if (trim($pattern) === '') {
                    continue;
                }
                $actions[] = [
                    'path' => ltrim(str_replace('\\', '/', $p), '/'),
                    'action' => 'replace_regex',
                    'create_if_missing' => $createIfMissingBool,
                    'pattern' => $pattern,
                    'replacement' => $replacement,
                ];
                continue;
            }

            if (count($lines) === 0) {
                continue;
            }

            $actions[] = [
                'path' => ltrim(str_replace('\\', '/', $p), '/'),
                'action' => $action,
                'create_if_missing' => $createIfMissingBool,
                'lines' => $lines,
            ];
        }

        if (count($actions) > 0) {
            $fileActions = $actions;
        }

        $uninstallActions = null;
        $uPaths = is_array($validated['uninstall_action_path']) ? $validated['uninstall_action_path'] : [];
        $uTexts = is_array($validated['uninstall_action_lines']) ? $validated['uninstall_action_lines'] : [];
        $uActionsRaw = is_array($validated['uninstall_action_action']) ? $validated['uninstall_action_action'] : [];
        $uCreateRaw = is_array($validated['uninstall_action_create_if_missing']) ? $validated['uninstall_action_create_if_missing'] : [];
        $uPatterns = is_array($validated['uninstall_action_pattern']) ? $validated['uninstall_action_pattern'] : [];
        $uReplacements = is_array($validated['uninstall_action_replacement']) ? $validated['uninstall_action_replacement'] : [];
        $uList = [];
        foreach ($uPaths as $i => $pRaw) {
            $p = trim((string) $pRaw);
            if ($p === '' || str_contains($p, '..')) {
                continue;
            }

            $action = strtolower(trim((string) $uActionsRaw[$i]));
            if (! in_array($action, ['ensure_contains', 'append_lines', 'prepend_lines', 'remove_lines', 'write_file', 'replace_regex'], true)) {
                $action = 'ensure_contains';
            }

            $createIfMissing = (string) $uCreateRaw[$i];
            $createIfMissingBool = $createIfMissing === '1' || strtolower($createIfMissing) === 'true' || $createIfMissing === 'on';

            $t = (string) $uTexts[$i];
            $lines = preg_split('/\r\n|\r|\n/', $t);
            if (! is_array($lines)) {
                $lines = [];
            }
            $lines = array_values(array_filter(array_map(fn ($l) => rtrim((string) $l, "\r\n"), $lines), fn ($l) => trim($l) !== ''));

            if ($action === 'replace_regex') {
                $pattern = (string) $uPatterns[$i];
                $replacement = (string) $uReplacements[$i];
                if (trim($pattern) === '') {
                    continue;
                }
                $uList[] = [
                    'path' => ltrim(str_replace('\\', '/', $p), '/'),
                    'action' => 'replace_regex',
                    'create_if_missing' => $createIfMissingBool,
                    'pattern' => $pattern,
                    'replacement' => $replacement,
                ];
                continue;
            }

            if (count($lines) === 0) {
                continue;
            }

            $uList[] = [
                'path' => ltrim(str_replace('\\', '/', $p), '/'),
                'action' => $action,
                'create_if_missing' => $createIfMissingBool,
                'lines' => $lines,
            ];
        }
        if (count($uList) > 0) {
            $uninstallActions = $uList;
        }

        $slug = (string) $validated['slug'];
        $archivePath = $plugin->archive_path;

        $archive = $request->file('archive');
        if ($archive) {
            $ext = $archive->getClientOriginalExtension();
            $ext = $ext !== '' ? ('.' . $ext) : '';
            $safeName = Str::slug(pathinfo((string) $archive->getClientOriginalName(), PATHINFO_FILENAME));
            $safeName = $safeName !== '' ? $safeName : 'archive';

            $storedPath = $archive->storeAs(
                'plugins/' . $slug,
                date('YmdHis') . '-' . $safeName . $ext
            );

            if (! is_string($storedPath) || $storedPath === '') {
                return back()->withErrors(['archive' => 'Не удалось сохранить архив'])->withInput();
            }
            $archivePath = $storedPath;
        }

        $plugin->update([
            'name' => $validated['name'],
            'category' => (string) $validated['category'],
            'slug' => $slug,
            'version' => $validated['version'],
            'archive_type' => $validated['archive_type'],
            'archive_path' => $archivePath,
            'install_path' => $installPath,
            'supported_games' => $supportedGames,
            'file_actions' => $fileActions,
            'uninstall_actions' => $uninstallActions,
            'restart_required' => (bool) $validated['restart_required'],
            'active' => (bool) $validated['active'],
        ]);

        return redirect()->route('admin.plugins.index')->with('success', 'Плагин обновлён');
    }

    public function destroy(Plugin $plugin): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $plugin->delete();

        return redirect()->route('admin.plugins.index')->with('success', 'Плагин удалён');
    }
}
