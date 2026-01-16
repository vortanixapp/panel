<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\File;
use Illuminate\View\View;
use Symfony\Component\Process\Process;

class LogsController extends Controller
{
    public function index(Request $request): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }
        $sources = $this->sources();
        $sourceKey = (string) $request->query('source', '');
        if ($sourceKey === '' && $request->query('file') !== null) {
            $sourceKey = 'laravel';
        }
        if ($sourceKey === '' || ! array_key_exists($sourceKey, $sources)) {
            $sourceKey = 'laravel';
        }

        $item = (string) $request->query('item', '');
        if ($item === '' && $request->query('file') !== null) {
            $item = (string) $request->query('file');
        }

        $items = $this->listItems($sourceKey);
        if ($item === '' && count($items) > 0) {
            $first = array_merge(['key' => ''], (array) $items[0]);
            $item = (string) $first['key'];
        }

        return view('admin.logs.index', [
            'sources' => $sources,
            'sourceKey' => $sourceKey,
            'items' => $items,
            'selected' => $item,
        ]);
    }

    public function tail(Request $request): JsonResponse|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $sourceKey = (string) $request->query('source', '');
        if ($sourceKey === '' && $request->query('file') !== null) {
            $sourceKey = 'laravel';
        }
        $sources = $this->sources();
        if ($sourceKey === '' || ! array_key_exists($sourceKey, $sources)) {
            return response()->json(['ok' => false, 'error' => 'Источник не найден'], 404);
        }

        $item = (string) $request->query('item', '');
        if ($item === '' && $request->query('file') !== null) {
            $item = (string) $request->query('file');
        }
        $lines = (int) $request->query('lines', 250);
        $lines = max(10, min(2000, $lines));

        $source = array_merge([
            'type' => '',
            'base_path' => '',
            'files' => [],
            'enabled' => false,
            'units' => [],
            'list_command' => [],
            'tail_command' => [],
        ], (array) $sources[$sourceKey]);
        $type = (string) $source['type'];

        if ($type === 'command') {
            if (! (bool) $source['enabled']) {
                return response()->json(['ok' => false, 'error' => 'Источник отключён'], 403);
            }

            $content = $this->tailCommand($sourceKey, $source, $item, $lines);
            if ($content === null) {
                return response()->json(['ok' => false, 'error' => 'Не удалось получить логи'], 404);
            }

            return response()->json([
                'ok' => true,
                'file' => $item,
                'content' => $content,
                'size' => null,
                'mtime' => null,
            ]);
        }

        $path = $this->resolvePathFromSource($sourceKey, $source, $item);
        if (! $path) {
            return response()->json(['ok' => false, 'error' => 'Файл не найден или недоступен'], 404);
        }

        $content = $this->readTail($path, $lines);

        return response()->json([
            'ok' => true,
            'file' => basename($path),
            'content' => $content,
            'size' => @filesize($path) ?: 0,
            'mtime' => @filemtime($path) ?: null,
        ]);
    }

    public function download(Request $request): \Symfony\Component\HttpFoundation\BinaryFileResponse|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $sourceKey = (string) $request->query('source', '');
        if ($sourceKey === '' && $request->query('file') !== null) {
            $sourceKey = 'laravel';
        }
        $sources = $this->sources();
        if ($sourceKey === '' || ! array_key_exists($sourceKey, $sources)) {
            abort(404);
        }

        $item = (string) $request->query('item', '');
        if ($item === '' && $request->query('file') !== null) {
            $item = (string) $request->query('file');
        }

        $source = array_merge([
            'type' => '',
            'base_path' => '',
            'files' => [],
            'enabled' => false,
            'units' => [],
            'list_command' => [],
            'tail_command' => [],
        ], (array) $sources[$sourceKey]);
        if ((string) $source['type'] === 'command') {
            abort(404);
        }

        $path = $this->resolvePathFromSource($sourceKey, $source, $item);
        if (! $path) {
            abort(404);
        }

        return response()->download($path, basename($path));
    }

    private function sources(): array
    {
        $src = (array) config('admin_logs.sources');
        return $src;
    }

    private function listItems(string $sourceKey): array
    {
        $sources = $this->sources();
        if (! array_key_exists($sourceKey, $sources)) {
            return [];
        }

        $source = array_merge([
            'type' => '',
            'base_path' => '',
            'files' => [],
            'enabled' => false,
            'units' => [],
            'list_command' => [],
            'tail_command' => [],
        ], (array) $sources[$sourceKey]);
        $type = (string) $source['type'];

        if ($type === 'directory') {
            $dir = (string) $source['base_path'];
            $items = [];
            if ($dir !== '' && is_dir($dir)) {
                foreach (File::files($dir) as $f) {
                    $name = $f->getFilename();
                    if (! str_ends_with(strtolower($name), '.log')) {
                        continue;
                    }
                    $items[] = [
                        'key' => $name,
                        'label' => $name,
                        'size' => (int) $f->getSize(),
                        'mtime' => (int) $f->getMTime(),
                    ];
                }
            }
            usort($items, fn ($a, $b) => ($b['mtime'] <=> $a['mtime']) ?: strcmp($a['label'], $b['label']));
            return $items;
        }

        if ($type === 'files') {
            $files = (array) $source['files'];
            $items = [];
            foreach ($files as $p) {
                if (! is_string($p) || $p === '') {
                    continue;
                }
                $items[] = [
                    'key' => $p,
                    'label' => basename($p),
                    'size' => is_file($p) ? ((int) @filesize($p) ?: 0) : 0,
                    'mtime' => is_file($p) ? ((int) @filemtime($p) ?: 0) : 0,
                ];
            }
            usort($items, fn ($a, $b) => ($b['mtime'] <=> $a['mtime']) ?: strcmp($a['label'], $b['label']));
            return $items;
        }

        if ($type === 'command') {
            if (! (bool) $source['enabled']) {
                return [];
            }

            if ($sourceKey === 'docker') {
                $names = $this->dockerContainers($source);
                return array_map(fn ($n) => ['key' => $n, 'label' => $n, 'size' => null, 'mtime' => null], $names);
            }

            if ($sourceKey === 'journal') {
                $units = (array) $source['units'];
                $out = [];
                foreach ($units as $u) {
                    if (! is_string($u) || $u === '') {
                        continue;
                    }
                    $out[] = ['key' => $u, 'label' => $u, 'size' => null, 'mtime' => null];
                }
                return $out;
            }
        }

        return [];
    }

    private function resolvePathFromSource(string $sourceKey, array $source, string $item): ?string
    {
        $source = array_merge([
            'type' => '',
            'base_path' => '',
            'files' => [],
            'enabled' => false,
            'units' => [],
            'list_command' => [],
            'tail_command' => [],
        ], (array) $source);
        $type = (string) $source['type'];

        if ($type === 'directory') {
            $dir = (string) $source['base_path'];
            $item = basename(trim($item));
            if ($dir === '' || $item === '' || ! str_ends_with(strtolower($item), '.log')) {
                return null;
            }
            $path = rtrim($dir, DIRECTORY_SEPARATOR) . DIRECTORY_SEPARATOR . $item;
            return $this->safeRealFile($dir, $path);
        }

        if ($type === 'files') {
            $files = (array) $source['files'];
            $item = trim($item);
            if ($item === '') {
                return null;
            }
            if (! in_array($item, $files, true)) {
                return null;
            }
            if (! is_file($item)) {
                return null;
            }
            return realpath($item) ?: null;
        }

        return null;
    }

    private function safeRealFile(string $baseDir, string $path): ?string
    {
        if (! is_file($path)) {
            return null;
        }
        $realDir = realpath($baseDir);
        $realPath = realpath($path);
        if (! $realDir || ! $realPath) {
            return null;
        }
        if (! str_starts_with($realPath, rtrim($realDir, DIRECTORY_SEPARATOR) . DIRECTORY_SEPARATOR)) {
            return null;
        }
        return $realPath;
    }

    private function dockerContainers(array $source): array
    {
        $source = array_merge([
            'list_command' => [],
        ], (array) $source);
        $cmd = (array) $source['list_command'];
        if (count($cmd) === 0) {
            return [];
        }
        $p = new Process($cmd);
        $p->setTimeout(3);
        $p->run();
        if (! $p->isSuccessful()) {
            return [];
        }
        $out = trim((string) $p->getOutput());
        if ($out === '') {
            return [];
        }
        $lines = array_values(array_filter(array_map('trim', explode("\n", $out)), fn ($v) => $v !== ''));
        $valid = [];
        foreach ($lines as $name) {
            if ($this->validContainerName($name)) {
                $valid[] = $name;
            }
        }
        sort($valid);
        return $valid;
    }

    private function tailCommand(string $sourceKey, array $source, string $item, int $lines): ?string
    {
        $item = trim($item);
        if ($item === '') {
            return null;
        }

        if ($sourceKey === 'docker') {
            if (! $this->validContainerName($item)) {
                return null;
            }
            $source = array_merge([
                'tail_command' => [],
            ], (array) $source);
            $base = (array) $source['tail_command'];
            if (count($base) < 3) {
                return null;
            }
            $cmd = array_merge(array_slice($base, 0, 3), [(string) $lines, $item]);
            $p = new Process($cmd);
            $p->setTimeout(5);
            $p->run();
            if (! $p->isSuccessful()) {
                return null;
            }
            return (string) $p->getOutput();
        }

        if ($sourceKey === 'journal') {
            $source = array_merge([
                'units' => [],
                'tail_command' => [],
            ], (array) $source);
            $units = (array) $source['units'];
            if (! in_array($item, $units, true)) {
                return null;
            }
            if (! $this->validSystemdUnit($item)) {
                return null;
            }
            $base = (array) $source['tail_command'];
            if (count($base) < 3) {
                return null;
            }
            $cmd = array_merge($base, [(string) $lines, '-u', $item]);
            $p = new Process($cmd);
            $p->setTimeout(5);
            $p->run();
            if (! $p->isSuccessful()) {
                return null;
            }
            return (string) $p->getOutput();
        }

        return null;
    }

    private function validContainerName(string $name): bool
    {
        return (bool) preg_match('/^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$/', $name);
    }

    private function validSystemdUnit(string $unit): bool
    {
        return (bool) preg_match('/^[a-zA-Z0-9@_.-]{1,255}(\\.service)?$/', $unit);
    }

    private function readTail(string $path, int $lines): string
    {
        $size = @filesize($path);
        if (! is_int($size) || $size <= 0) {
            return '';
        }

        $chunk = min($size, 512 * 1024);
        $fp = @fopen($path, 'rb');
        if (! $fp) {
            return '';
        }

        try {
            fseek($fp, -$chunk, SEEK_END);
            $data = (string) fread($fp, $chunk);
        } finally {
            fclose($fp);
        }

        $data = str_replace("\r\n", "\n", $data);
        $data = str_replace("\r", "\n", $data);
        $parts = explode("\n", $data);

        $tail = array_slice($parts, -$lines);
        return implode("\n", $tail);
    }
}
