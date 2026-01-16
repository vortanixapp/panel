<?php

namespace App\Http\Controllers;

use App\Jobs\ReinstallServer;
use App\Models\Server;
use App\Models\ServerUserPermission;
use App\Models\Map;
use App\Models\ServerMap;
use App\Models\Plugin;
use App\Models\ServerPlugin;
use App\Models\Tariff;
use App\Models\Transaction;
use App\Models\User;
use App\Services\PromotionService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Crypt;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\URL;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Str;
use Illuminate\View\View;

class ServerController extends Controller
{
    public function status(Server $server): JsonResponse
    {
        $this->authorizeServer($server);
        $this->abortIfExpired($server);

        $server->loadMissing(['location', 'game']);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));

        $probe = $this->callDaemon($server, '/servers/status', [
            'server_id' => (int) $server->id,
            'game' => $gameCode,
        ], 10);

        if ((bool) $probe['ok'] === true) {
            $daemonState = strtolower((string) $probe['state']);
            $runtimeStatus = $this->mapRuntimeStatus($daemonState);

            $updates = [
                'runtime_status' => $runtimeStatus,
            ];

            if (array_key_exists('container_id', $probe)) {
                $updates['container_id'] = $probe['container_id'] ?: null;
            }
            if (array_key_exists('container_name', $probe)) {
                $updates['container_name'] = $probe['container_name'] ?: null;
            }
            if (array_key_exists('port', $probe) && is_int($probe['port'])) {
                $updates['port'] = (int) $probe['port'];
            }

            $prov = strtolower((string) $server->provisioning_status);
            if ($daemonState === 'running') {
                if (in_array($prov, ['pending', 'installing', 'reinstalling', 'failed'], true)) {
                    $updates['provisioning_status'] = 'running';
                    $updates['provisioning_error'] = null;
                }

                $currStatus = strtolower((string) $server->status);
                if ($currStatus === 'suspended') {
                    $updates['status'] = 'active';
                }
            }

            if (! empty($updates)) {
                $server->update($updates);
            }
        } elseif ((string) $probe['error'] !== '') {

        }

        $server->refresh();

        return response()->json([
            'ok' => true,
            'server_id' => (int) $server->id,
            'provisioning_status' => (string) $server->provisioning_status,
            'runtime_status' => (string) $server->runtime_status,
            'provisioning_error' => (string) $server->provisioning_error,
        ]);
    }

    private function enforceServerSlotsWithinTariff(Server $server): void
    {
        $server->loadMissing(['tariff']);

        $tariff = $server->tariff;
        if (! $tariff) {
            return;
        }

        if ((string) $tariff->billing_type !== 'slots') {
            return;
        }

        $min = (int) $tariff->min_slots;
        $max = (int) $tariff->max_slots;
        $min = max(1, $min);
        $max = max($min, $max);

        $current = (int) $server->slots;
        $clamped = max($min, min($max, $current));
        if ($clamped !== $current) {
            $server->update(['slots' => $clamped]);
            $server->refresh();
        }
    }

    private function getMaxPlayersFromContent(string $path, string $content): ?int
    {
        $lowerPath = strtolower($path);

        if (str_ends_with($lowerPath, 'server.cfg')) {
            if (preg_match('/^\s*maxplayers\s+(\d+)\b/im', $content, $m)) {
                return (int) $m[1];
            }
        }

        if (str_ends_with($lowerPath, 'server.properties')) {
            if (preg_match('/^\s*max-players\s*=\s*(\d+)\b/im', $content, $m)) {
                return (int) $m[1];
            }
        }

        return null;
    }

    private function enforceMaxPlayersInContent(string $path, string $content, int $slots): string
    {
        $slots = max(1, $slots);
        $lowerPath = strtolower($path);

        if (str_ends_with($lowerPath, 'server.cfg')) {
            if (preg_match('/^\s*maxplayers\s+\d+\b/im', $content)) {
                $replaced = preg_replace('/^\s*maxplayers\s+\d+\b/im', 'maxplayers ' . $slots, $content, 1);
                if ($replaced !== null) {
                    $content = $replaced;
                }
                return $content;
            }

            $trimmed = rtrim($content, "\r\n");
            return $trimmed . "\nmaxplayers " . $slots . "\n";
        }

        if (str_ends_with($lowerPath, 'server.properties')) {
            if (preg_match('/^\s*max-players\s*=\s*\d+\b/im', $content)) {
                $replaced = preg_replace('/^\s*max-players\s*=\s*\d+\b/im', 'max-players=' . $slots, $content, 1);
                if ($replaced !== null) {
                    $content = $replaced;
                }
                return $content;
            }

            $trimmed = rtrim($content, "\r\n");
            return $trimmed . "\nmax-players=" . $slots . "\n";
        }

        return $content;
    }

    public function show(Server $server): View
    {
        $this->authorizeServer($server);

        $layout = request()->routeIs('admin.servers.manage') ? 'layouts.app-admin' : 'layouts.app-user';
        $serverBaseUrl = request()->routeIs('admin.servers.manage')
            ? route('admin.servers.manage', $server)
            : route('server.show', $server);

        $isExpired = $server->expires_at && now()->greaterThan($server->expires_at);

        $tab = (string) request()->query('tab', 'main');
        if (! in_array($tab, ['main', 'console', 'logs', 'metrics', 'ftp', 'mysql', 'cron', 'firewall', 'friends', 'settings', 'plugins', 'maps'], true)) {
            $tab = 'main';
        }
        $prov = strtolower((string) $server->provisioning_status);
        $isProvisioning = in_array($prov, ['pending', 'installing', 'reinstalling'], true);
        if ($isProvisioning && ! in_array($tab, ['main', 'logs'], true)) {
            $tab = 'main';
        }
        if ($prov === 'failed') {
            $tab = 'main';
        }
        if ($tab === 'console') {
            return view('server.console', [
                'server' => $server->load(['game', 'location', 'tariff']),
                'layout' => $layout,
                'serverBaseUrl' => $serverBaseUrl,
            ]);
        }

        if ($tab === 'logs') {
            return view('server.logs', [
                'server' => $server->load(['game', 'location', 'tariff']),
                'layout' => $layout,
                'serverBaseUrl' => $serverBaseUrl,
            ]);
        }

        if ($tab === 'metrics') {
            return view('server.metrics', [
                'server' => $server->load(['game', 'location', 'tariff']),
                'layout' => $layout,
                'serverBaseUrl' => $serverBaseUrl,
            ]);
        }

        if ($tab === 'ftp') {
            return view('server.ftp', [
                'server' => $server->load(['game', 'location', 'tariff']),
                'layout' => $layout,
                'serverBaseUrl' => $serverBaseUrl,
            ]);
        }

        if ($tab === 'mysql') {
            return view('server.mysql', [
                'server' => $server->load(['game', 'location', 'tariff']),
                'layout' => $layout,
                'serverBaseUrl' => $serverBaseUrl,
            ]);
        }

        if ($tab === 'cron') {
            $this->requirePermission($server, 'can_view_cron');
            return view('server.cron', [
                'server' => $server->load(['game', 'location', 'tariff']),
                'layout' => $layout,
                'serverBaseUrl' => $serverBaseUrl,
            ]);
        }

        if ($tab === 'firewall') {
            $this->requirePermission($server, 'can_view_firewall');
            return view('server.firewall', [
                'server' => $server->load(['game', 'location', 'tariff']),
                'layout' => $layout,
                'serverBaseUrl' => $serverBaseUrl,
            ]);
        }

        if ($tab === 'friends') {
            $this->requirePermission($server, 'can_view_friends');
            return view('server.friends', [
                'server' => $server->load(['game', 'location', 'tariff']),
                'layout' => $layout,
                'serverBaseUrl' => $serverBaseUrl,
            ]);
        }

        if ($tab === 'maps') {
            $this->requirePermission($server, 'can_settings_edit');
            $this->abortIfExpired($server);
            $this->abortIfReinstalling($server);

            $server->load(['game', 'location', 'tariff']);

            $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
            if (! in_array($gameCode, ['cs16', 'cstrike', 'counter-strike', 'counter_strike', 'cs_1_6'], true)) {
                return view('server.main', [
                    'server' => $server,
                    'layout' => $layout,
                    'serverBaseUrl' => $serverBaseUrl,
                ]);
            }

            $maps = Map::query()
                ->where('active', true)
                ->orderBy('category')
                ->orderBy('name')
                ->get();

            $serverMaps = ServerMap::query()
                ->where('server_id', (int) $server->id)
                ->get()
                ->keyBy('map_id');

            $items = [];
            foreach ($maps as $m) {
                $sm = $serverMaps->get((int) $m->id);
                if (! $sm) {
                    $sm = new ServerMap([
                        'server_id' => (int) $server->id,
                        'map_id' => (int) $m->id,
                        'installed' => false,
                    ]);
                }

                $items[] = [
                    'map' => $m,
                    'serverMap' => $sm,
                ];
            }

            return view('server.maps', [
                'server' => $server,
                'items' => $items,
                'layout' => $layout,
                'serverBaseUrl' => $serverBaseUrl,
            ]);
        }

        if ($tab === 'plugins') {
            $this->requirePermission($server, 'can_settings_edit');
            $this->abortIfExpired($server);
            $this->abortIfReinstalling($server);

            $server->load(['game', 'location', 'tariff']);

            $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));

            $plugins = Plugin::query()
                ->where('active', true)
                ->orderBy('category')
                ->orderBy('name')
                ->get();

            $serverPlugins = ServerPlugin::query()
                ->where('server_id', (int) $server->id)
                ->get()
                ->keyBy('plugin_id');

            $items = [];
            foreach ($plugins as $p) {
                $supported = $p->supported_games;
                $isSupported = true;
                if (is_array($supported) && count($supported) > 0) {
                    $supportedNorm = array_map(fn ($v) => strtolower((string) $v), $supported);
                    $isSupported = in_array('*', $supportedNorm, true) || in_array($gameCode, $supportedNorm, true);
                }
                if (! $isSupported) {
                    continue;
                }

                $sp = $serverPlugins->get((int) $p->id);
                if (! $sp) {
                    $sp = new ServerPlugin([
                        'server_id' => (int) $server->id,
                        'plugin_id' => (int) $p->id,
                        'installed' => false,
                        'enabled' => true,
                    ]);
                }

                $items[] = [
                    'plugin' => $p,
                    'serverPlugin' => $sp,
                ];
            }

            return view('server.plugins', [
                'server' => $server,
                'items' => $items,
                'layout' => $layout,
                'serverBaseUrl' => $serverBaseUrl,
            ]);
        }

        if ($tab === 'settings') {
            $server->load(['game', 'location', 'tariff']);

            $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
            $isMinecraft = in_array($gameCode, ['mcjava', 'mcpaper', 'mcspigot', 'mcforge', 'mcfabric', 'mcbedrock', 'mcbedrk', 'bedrock'], true);
            $isRust = in_array($gameCode, ['rust'], true);
            $cfgRaw = '';
            $cfg = [];
            $cfgError = null;
            $cfgPath = '';

            $load = null;
            if (in_array($gameCode, ['samp', 'crmp'], true)) {
                $load = $this->loadCfgFromDaemon($server, 'server.cfg', fn (string $raw) => $this->parseSampServerCfg($raw));
            } elseif (in_array($gameCode, ['tf2', 'teamfortress2', 'team_fortress_2', 'tf'], true)) {
                $load = $this->loadCfgFromDaemon($server, 'tf/cfg/server.cfg', fn (string $raw) => $this->parseCs16ServerCfg($raw));
            } elseif (in_array($gameCode, ['css', 'cs:s', 'cs_source', 'counter-strike_source', 'counter_strike_source'], true)) {
                $load = $this->loadCfgFromDaemon($server, 'cstrike/cfg/server.cfg', fn (string $raw) => $this->parseCs16ServerCfg($raw));
            } elseif (in_array($gameCode, ['gmod', 'garrysmod', "garry's mod", 'garrys_mod'], true)) {
                $load = $this->loadCfgFromDaemon($server, 'garrysmod/cfg/server.cfg', fn (string $raw) => $this->parseCs16ServerCfg($raw));
            } elseif (in_array($gameCode, ['unturned', 'unturn', 'ut', 'untrm4', 'untrm5'], true)) {
                $load = $this->loadCfgFromDaemon($server, 'Servers/server/Server/Commands.dat', fn (string $raw) => $this->parseUnturnedCommandsDat($raw));
            }

            if (is_array($load)) {
                $cfgRaw = (string) $load['cfgRaw'];
                $cfg = is_array($load['cfg']) ? (array) $load['cfg'] : [];
                $cfgPath = (string) $load['cfgPath'];
                $cfgError = $load['cfgError'];
            }

            if ($isRust) {
                $p = 'rust.env';
                $load = $this->loadCfgFromDaemon($server, $p, fn (string $raw) => $this->parseEnvFile($raw));
                $cfgRaw = (string) $load['cfgRaw'];
                $cfg = is_array($load['cfg']) ? (array) $load['cfg'] : [];
                $cfgPath = (string) $load['cfgPath'];
                $cfgError = $load['cfgError'];
            }

            if (in_array($gameCode, ['cs2', 'counter-strike2', 'counter_strike2', 'counter-strike_2', 'counter_strike_2'], true)) {
                $p = 'game/csgo/cfg/server.cfg';
                $load = $this->loadCfgFromDaemon($server, $p, fn (string $raw) => $this->parseCs16ServerCfg($raw));
                $cfgRaw = (string) $load['cfgRaw'];
                $cfg = is_array($load['cfg']) ? (array) $load['cfg'] : [];
                $cfgPath = (string) $load['cfgPath'];
                $cfgError = $load['cfgError'];
            }

            if ($isMinecraft) {
                $p = 'server.properties';
                $load = $this->loadCfgFromDaemon($server, $p, fn (string $raw) => $this->parseMinecraftServerProperties($raw));
                $cfgRaw = (string) $load['cfgRaw'];
                $cfg = is_array($load['cfg']) ? (array) $load['cfg'] : [];
                $cfgPath = (string) $load['cfgPath'];
                $cfgError = $load['cfgError'];
            }

            if (in_array($gameCode, ['cs16', 'cstrike', 'counter-strike', 'counter_strike', 'cs_1_6'], true)) {
                $pathsToTry = ['cstrike/server.cfg', 'server.cfg'];
                foreach ($pathsToTry as $p) {
                    $load = $this->loadCfgFromDaemon($server, $p, fn (string $raw) => $this->parseCs16ServerCfg($raw));
                    if ($load['cfgError'] === null) {
                        $cfgRaw = (string) $load['cfgRaw'];
                        $cfg = is_array($load['cfg']) ? (array) $load['cfg'] : [];
                        $cfgPath = (string) $load['cfgPath'];
                        $cfgError = null;
                        break;
                    }
                    $cfgError = (string) $load['cfgError'];
                }
            }

            return view('server.settings', [
                'server' => $server,
                'cfgRaw' => $cfgRaw,
                'cfg' => $cfg,
                'cfgError' => $cfgError,
                'cfgPath' => $cfgPath,
                'layout' => $layout,
                'serverBaseUrl' => $serverBaseUrl,
            ]);
        }

        return view('server.main', [
            'server' => $server->load(['game', 'location', 'tariff']),
            'layout' => $layout,
            'serverBaseUrl' => $serverBaseUrl,
        ]);
    }

    public function pluginInstall(Server $server, Plugin $plugin): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);

        $server->load(['game', 'location']);

        if (! $plugin->active) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'plugins'])->with('error', 'Плагин отключен администратором');
        }

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        $supported = $plugin->supported_games;
        if (is_array($supported) && count($supported) > 0) {
            $supportedNorm = array_map(fn ($v) => strtolower((string) $v), $supported);
            $isSupported = in_array('*', $supportedNorm, true) || in_array($gameCode, $supportedNorm, true);
            if (! $isSupported) {
                return redirect()->route('server.show', ['server' => $server, 'tab' => 'plugins'])->with('error', 'Плагин не поддерживается для этой игры');
            }
        }

        $installPathRaw = trim((string) $plugin->install_path);
        $installPath = trim($installPathRaw, " \t\n\r\0\x0B/");
        if ($installPath !== '' && str_contains($installPath, '..')) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'plugins'])->with('error', 'Некорректный install_path у плагина');
        }

        $archiveUrl = null;
        if ((string) $plugin->archive_path !== '') {
            $archiveUrl = URL::temporarySignedRoute(
                'plugins.download',
                now()->addMinutes(10),
                ['plugin' => (int) $plugin->id]
            );
        }

        $fileActions = $plugin->file_actions;
        if (! is_array($fileActions)) {
            $fileActions = null;
        }

        if ($archiveUrl === null && (! is_array($fileActions) || count($fileActions) === 0)) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'plugins'])->with('error', 'У плагина нет архива и нет file_actions');
        }

        $payload = [
            'server_id' => (int) $server->id,
            'game' => $gameCode,
            'action' => 'install',
            'archive_url' => $archiveUrl,
            'archive_type' => (string) $plugin->archive_type,
            'install_path' => $installPath,
            'file_actions' => $fileActions,
        ];

        $result = $this->callDaemon($server, '/servers/plugins/apply', $payload, 180);

        $sp = ServerPlugin::query()->firstOrNew([
            'server_id' => (int) $server->id,
            'plugin_id' => (int) $plugin->id,
        ]);

        if (! (bool) $result['ok']) {
            $sp->fill([
                'installed' => (bool) $sp->installed,
                'enabled' => (bool) $sp->enabled,
                'last_error' => (string) $result['error'],
            ])->save();

            return redirect()->route('server.show', ['server' => $server, 'tab' => 'plugins'])->with('error', 'Не удалось установить плагин: ' . (string) $result['error']);
        }

        $sp->fill([
            'installed' => true,
            'enabled' => (bool) $sp->enabled,
            'installed_at' => now(),
            'last_error' => null,
        ])->save();

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'plugins'])->with('success', 'Плагин установлен');
    }

    public function pluginUninstall(Server $server, Plugin $plugin): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);

        $server->load(['game', 'location']);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        $installPathRaw = trim((string) $plugin->install_path);
        $installPath = trim($installPathRaw, " \t\n\r\0\x0B/");
        if ($installPath !== '' && str_contains($installPath, '..')) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'plugins'])->with('error', 'Некорректный install_path у плагина');
        }

        if ($installPath === '') {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'plugins'])->with('error', 'Нельзя удалить плагин из корня сервера');
        }

        $uninstallActions = $plugin->uninstall_actions;
        if (! is_array($uninstallActions)) {
            $uninstallActions = null;
        }

        $payload = [
            'server_id' => (int) $server->id,
            'game' => $gameCode,
            'action' => 'uninstall',
            'install_path' => $installPath,
        ];

        if (is_array($uninstallActions) && count($uninstallActions) > 0) {
            $payload['uninstall_actions'] = $uninstallActions;
        }

        $result = $this->callDaemon($server, '/servers/plugins/apply', $payload, 180);

        $sp = ServerPlugin::query()->firstOrNew([
            'server_id' => (int) $server->id,
            'plugin_id' => (int) $plugin->id,
        ]);

        if (! (bool) $result['ok']) {
            $sp->fill([
                'last_error' => (string) $result['error'],
            ])->save();

            return redirect()->route('server.show', ['server' => $server, 'tab' => 'plugins'])->with('error', 'Не удалось удалить плагин: ' . (string) $result['error']);
        }

        $sp->fill([
            'installed' => false,
            'enabled' => (bool) $sp->enabled,
            'last_error' => null,
        ])->save();

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'plugins'])->with('success', 'Плагин удалён');
    }

    public function pluginToggle(Server $server, Plugin $plugin, Request $request): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);

        $enabled = (string) $request->input('enabled', '1');
        $enabledBool = $enabled === '1' || strtolower($enabled) === 'true' || $enabled === 'on';

        $sp = ServerPlugin::query()->firstOrNew([
            'server_id' => (int) $server->id,
            'plugin_id' => (int) $plugin->id,
        ]);

        $sp->fill([
            'enabled' => $enabledBool,
        ])->save();

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'plugins'])->with('success', 'Настройка плагина сохранена');
    }

    public function mapInstall(Server $server, Map $map): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);

        $server->load(['game', 'location']);

        if (! $map->active) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'maps'])->with('error', 'Карта отключена администратором');
        }

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if (! in_array($gameCode, ['cs16', 'cstrike', 'counter-strike', 'counter_strike', 'cs_1_6'], true)) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'maps'])->with('error', 'Карты доступны только для CS 1.6');
        }

        $archiveUrl = null;
        if ((string) $map->archive_path !== '') {
            $archiveUrl = URL::temporarySignedRoute(
                'maps.download',
                now()->addMinutes(10),
                ['map' => (int) $map->id]
            );
        }

        if ($archiveUrl === null) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'maps'])->with('error', 'У карты нет архива');
        }

        $payload = [
            'server_id' => (int) $server->id,
            'game' => $gameCode,
            'action' => 'install',
            'archive_url' => $archiveUrl,
            'install_path' => '',
        ];

        $result = $this->callDaemon($server, '/servers/maps/apply', $payload, 180);

        $sm = ServerMap::query()->firstOrNew([
            'server_id' => (int) $server->id,
            'map_id' => (int) $map->id,
        ]);

        if (! (bool) $result['ok']) {
            $sm->fill([
                'installed' => (bool) $sm->installed,
                'last_error' => (string) $result['error'],
            ])->save();

            return redirect()->route('server.show', ['server' => $server, 'tab' => 'maps'])->with('error', 'Не удалось установить карту: ' . (string) $result['error']);
        }

        $sm->fill([
            'installed' => true,
            'installed_at' => now(),
            'last_error' => null,
        ])->save();

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'maps'])->with('success', 'Карта установлена');
    }

    public function mapUninstall(Server $server, Map $map): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);

        $server->load(['game', 'location']);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if (! in_array($gameCode, ['cs16', 'cstrike', 'counter-strike', 'counter_strike', 'cs_1_6'], true)) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'maps'])->with('error', 'Карты доступны только для CS 1.6');
        }

        $list = $map->file_list;
        if (! is_array($list) || count($list) === 0) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'maps'])->with('error', 'У карты нет списка файлов для удаления');
        }

        $paths = [];
        foreach ($list as $p) {
            $p = ltrim(str_replace('\\', '/', (string) $p), '/');
            if ($p === '' || str_contains($p, '..')) {
                continue;
            }

            if (str_starts_with(strtolower($p), 'cstrike/')) {
                $paths[] = $p;
            } else {
                $paths[] = 'cstrike/' . $p;
            }
        }
        $paths = array_values(array_unique($paths));
        if (count($paths) === 0) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'maps'])->with('error', 'Некорректный список файлов');
        }

        $payload = [
            'server_id' => (int) $server->id,
            'game' => $gameCode,
            'action' => 'uninstall',
            'paths' => $paths,
        ];

        $result = $this->callDaemon($server, '/servers/maps/apply', $payload, 180);

        $sm = ServerMap::query()->firstOrNew([
            'server_id' => (int) $server->id,
            'map_id' => (int) $map->id,
        ]);

        if (! (bool) $result['ok']) {
            $sm->fill([
                'last_error' => (string) $result['error'],
            ])->save();

            return redirect()->route('server.show', ['server' => $server, 'tab' => 'maps'])->with('error', 'Не удалось удалить карту: ' . (string) $result['error']);
        }

        $sm->fill([
            'installed' => false,
            'last_error' => null,
        ])->save();

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'maps'])->with('success', 'Карта удалена');
    }

    private function abortIfReinstalling(Server $server): void
    {
        if (strtolower((string) $server->provisioning_status) === 'reinstalling') {
            abort(403, 'Доступ запрещен');
        }
    }

    private function abortIfExpired(Server $server): void
    {
        if (! $server->expires_at) {
            return;
        }

        if (now()->greaterThan($server->expires_at)) {
            abort(403, 'Срок аренды сервера истёк');
        }
    }

    public function consoleLogs(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_view_logs');
        $this->abortIfExpired($server);
        $server->load(['location']);

        $tail = (int) $request->query('tail', 200);
        $tail = max(1, min($tail, 1000));

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));

        $result = $this->callDaemon($server, '/servers/logs', [
            'server_id' => $server->id,
            'tail' => $tail,
            'game' => $gameCode,
        ]);

        if (! $result['ok']) {
            return response()->json(['ok' => false, 'error' => $result['error']], 502);
        }

        return response()->json($result);
    }

    public function firewallList(Server $server): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_view_firewall');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location']);

        $result = $this->callDaemon($server, '/servers/firewall/list', [
            'server_id' => (int) $server->id,
        ], 20);

        if (! (bool) $result['ok']) {
            return response()->json(['ok' => false, 'error' => (string) $result['error']], 502);
        }

        return response()->json($result);
    }

    public function firewallToggle(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_firewall_manage');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location']);

        $validated = $request->validate([
            'enabled' => ['required', 'boolean'],
        ]);

        $result = $this->callDaemon($server, '/servers/firewall/toggle', [
            'server_id' => (int) $server->id,
            'port' => (int) $server->port,
            'enabled' => (bool) $validated['enabled'],
        ], 30);

        if (! (bool) $result['ok']) {
            return response()->json(['ok' => false, 'error' => (string) $result['error']], 502);
        }

        return response()->json($result);
    }

    public function firewallSet(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_firewall_manage');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location']);

        $validated = $request->validate([
            'enabled' => ['required', 'boolean'],
            'mode' => ['required', 'string', 'in:allow,deny'],
        ]);

        $result = $this->callDaemon($server, '/servers/firewall/set', [
            'server_id' => (int) $server->id,
            'port' => (int) $server->port,
            'enabled' => (bool) $validated['enabled'],
            'mode' => (string) $validated['mode'],
        ], 30);

        if (! (bool) $result['ok']) {
            return response()->json(['ok' => false, 'error' => (string) $result['error']], 502);
        }

        return response()->json($result);
    }

    public function firewallAddRule(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_firewall_manage');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location']);

        $validated = $request->validate([
            'cidr' => ['required', 'string', 'max:64'],
            'proto' => ['required', 'string', 'in:udp,tcp,both'],
            'note' => ['nullable', 'string', 'max:128'],
            'enabled' => ['sometimes', 'boolean'],
        ]);

        $result = $this->callDaemon($server, '/servers/firewall/add-rule', [
            'server_id' => (int) $server->id,
            'port' => (int) $server->port,
            'cidr' => (string) $validated['cidr'],
            'proto' => (string) $validated['proto'],
            'note' => (string) $validated['note'],
            'enabled' => (bool) $validated['enabled'],
        ], 30);

        if (! (bool) $result['ok']) {
            return response()->json(['ok' => false, 'error' => (string) $result['error']], 502);
        }

        return response()->json($result);
    }

    public function firewallDeleteRule(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_firewall_manage');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location']);

        $validated = $request->validate([
            'rule_id' => ['required', 'string', 'max:64'],
        ]);

        $result = $this->callDaemon($server, '/servers/firewall/delete-rule', [
            'server_id' => (int) $server->id,
            'port' => (int) $server->port,
            'rule_id' => (string) $validated['rule_id'],
        ], 30);

        if (! (bool) $result['ok']) {
            return response()->json(['ok' => false, 'error' => (string) $result['error']], 502);
        }

        return response()->json($result);
    }

    public function firewallToggleRule(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_firewall_manage');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location']);

        $validated = $request->validate([
            'rule_id' => ['required', 'string', 'max:64'],
            'enabled' => ['required', 'boolean'],
        ]);

        $result = $this->callDaemon($server, '/servers/firewall/toggle-rule', [
            'server_id' => (int) $server->id,
            'port' => (int) $server->port,
            'rule_id' => (string) $validated['rule_id'],
            'enabled' => (bool) $validated['enabled'],
        ], 30);

        if (! (bool) $result['ok']) {
            return response()->json(['ok' => false, 'error' => (string) $result['error']], 502);
        }

        return response()->json($result);
    }

    public function consoleCommand(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_console_command');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location']);

        $validated = $request->validate([
            'command' => ['required', 'string', 'max:512'],
        ]);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        $isMinecraft = in_array($gameCode, ['mcjava', 'mcpaper', 'mcspigot', 'mcforge', 'mcfabric'], true);
        $isCs16 = in_array($gameCode, ['cs16', 'cstrike', 'counter-strike', 'counter_strike', 'cs_1_6'], true);
        if ($gameCode !== 'samp' && ! $isMinecraft && ! $isCs16) {
            return response()->json(['ok' => false, 'error' => 'Отправка команд пока поддерживается только для SA-MP, CS 1.6 и Minecraft Java'], 400);
        }

        if ($isMinecraft) {
            $read = $this->callDaemon($server, '/servers/files/read', [
                'server_id' => $server->id,
                'path' => 'server.properties',
            ]);

            if (! $read['ok'] || ! is_string($read['content'])) {
                return response()->json(['ok' => false, 'error' => 'Не удалось прочитать server.properties: ' . (string) $read['error']], 502);
            }

            $props = $this->parseMinecraftServerProperties((string) $read['content']);
            $enableRcon = strtolower((string) $props['enable-rcon']) === 'true';
            $rconPassword = (string) $props['rcon.password'];
            $rconPort = (int) $props['rcon.port'];
            if (! $enableRcon) {
                return response()->json(['ok' => false, 'error' => 'В server.properties выключен enable-rcon'], 400);
            }
            if ($rconPassword === '') {
                return response()->json(['ok' => false, 'error' => 'В server.properties не задан rcon.password'], 400);
            }
            if ($rconPort <= 0) {
                $rconPort = 25575;
            }

            $payload = [
                'server_id' => $server->id,
                'command' => (string) $validated['command'],
                'password' => $rconPassword,
                'rcon_port' => $rconPort,
                'game' => $gameCode,
            ];

            $result = $this->callDaemon($server, '/servers/rcon', $payload, 10);
            if (! (bool) $result['ok']) {
                return response()->json(['ok' => false, 'error' => (string) $result['error']], 502);
            }

            return response()->json($result);
        }

        if ($isCs16) {
            $read = $this->callDaemon($server, '/servers/files/read', [
                'server_id' => $server->id,
                'path' => 'cstrike/server.cfg',
            ]);

            if (! (bool) $read['ok'] || ! is_string($read['content'])) {
                return response()->json(['ok' => false, 'error' => 'Не удалось прочитать cstrike/server.cfg: ' . (string) $read['error']], 502);
            }

            $cfg = $this->parseCs16ServerCfg((string) $read['content']);
            $rconPassword = (string) $cfg['rcon_password'];
            if ($rconPassword === '') {
                return response()->json(['ok' => false, 'error' => 'В cstrike/server.cfg не задан rcon_password'], 400);
            }

            $payload = [
                'server_id' => $server->id,
                'command' => (string) $validated['command'],
                'password' => $rconPassword,
                'game' => $gameCode,
            ];

            $result = $this->callDaemon($server, '/servers/rcon', $payload, 10);
            if (! (bool) $result['ok']) {
                return response()->json(['ok' => false, 'error' => (string) $result['error']], 502);
            }

            return response()->json($result);
        }

        $read = $this->callDaemon($server, '/servers/files/read', [
            'server_id' => $server->id,
            'path' => 'server.cfg',
        ]);

        if (! (bool) $read['ok'] || ! is_string($read['content'])) {
            return response()->json(['ok' => false, 'error' => 'Не удалось прочитать server.cfg: ' . (string) $read['error']], 502);
        }

        $cfg = $this->parseSampServerCfg((string) $read['content']);
        $rconPassword = (string) $cfg['rcon_password'];
        if ($rconPassword === '') {
            return response()->json(['ok' => false, 'error' => 'В server.cfg не задан rcon_password'], 400);
        }

        $payload = [
            'server_id' => $server->id,
            'command' => (string) $validated['command'],
            'password' => $rconPassword,
        ];

        $payload['game'] = $gameCode;

        $headerHost = (string) $server->ip_address;
        if (preg_match('/^\d{1,3}(?:\.\d{1,3}){3}$/', $headerHost)) {
            $payload['header_host'] = $headerHost;
        }

        $headerPort = (int) $server->port;
        if ($headerPort > 0) {
            $payload['header_port'] = $headerPort;
        }

        $result = $this->callDaemon($server, '/servers/rcon', $payload, 10);
        if (! (bool) $result['ok']) {
            return response()->json(['ok' => false, 'error' => (string) $result['error']], 502);
        }

        return response()->json($result);
    }

    public function metrics(Server $server): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_view_metrics');
        $this->abortIfExpired($server);
        $server->load(['location']);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));

        $result = $this->callDaemon($server, '/servers/metrics', [
            'server_id' => $server->id,
            'game' => $gameCode,
        ]);

        if (! $result['ok']) {
            return response()->json(['ok' => false, 'error' => $result['error']], 502);
        }

        return response()->json($result);
    }

    public function start(Server $server): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_start');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location', 'game', 'tariff']);

        $this->enforceServerSlotsWithinTariff($server);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));

        $result = $this->callDaemon($server, '/servers/start', [
            'server_id' => $server->id,
            'game' => $gameCode,
            'cpu_cores' => (float) $server->cpu_cores,
            'cpu_shares' => $server->cpu_shares,
            'ram_gb' => (float) $server->ram_gb,
            'disk_gb' => (float) $server->disk_gb,
            'slots' => (int) $server->slots,
            'server_fps' => $server->server_fps,
            'server_tickrate' => $server->server_tickrate,
            'antiddos_enabled' => (bool) $server->antiddos_enabled,
        ]);

        if (! $result['ok']) {
            return redirect()->route('server.show', $server)->with('error', $result['error']);
        }

        $server->update([
            'runtime_status' => $this->mapRuntimeStatus($result['state']),
            'port' => (int) $result['port'],
        ]);

        return redirect()->route('server.show', $server)->with('success', 'Сервер запущен');
    }

    public function stop(Server $server): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_stop');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location', 'game', 'tariff']);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));

        $result = $this->callDaemon($server, '/servers/stop', [
            'server_id' => $server->id,
            'game' => $gameCode,
        ]);

        if (! $result['ok']) {
            return redirect()->route('server.show', $server)->with('error', $result['error']);
        }

        $server->update([
            'runtime_status' => $this->mapRuntimeStatus($result['state']),
            'port' => (int) $result['port'],
        ]);

        return redirect()->route('server.show', $server)->with('success', 'Сервер остановлен');
    }

    public function restart(Server $server): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_restart');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location', 'game', 'tariff']);

        $this->enforceServerSlotsWithinTariff($server);

        $server->update([
            'runtime_status' => 'restarting',
        ]);

        dispatch(new \App\Jobs\RestartServer((int) $server->id));

        return redirect()->route('server.show', $server)->with('success', 'Перезапуск запущен, обновите статус через пару секунд.');
    }

    public function updateAutoStart(Server $server, Request $request): RedirectResponse
    {
        $this->authorizeServer($server);
        if (! $this->isServerOwner($server)) {
            abort(403, 'Доступ запрещен');
        }

        $validated = $request->validate([
            'enabled' => ['required', 'boolean'],
        ]);

        $server->update([
            'auto_start_enabled' => (bool) $validated['enabled'],
        ]);

        return redirect()->route('server.show', $server)->with('success', 'Настройка автоподнятия сохранена');
    }

    public function renew(Server $server, Request $request): RedirectResponse
    {
        $this->authorizeServer($server);
        if (! $this->isServerOwner($server)) {
            abort(403, 'Доступ запрещен');
        }

        $validated = $request->validate([
            'period' => ['required', 'integer', 'in:15,30,60,180'],
            'promo_code' => ['nullable', 'string', 'max:255'],
        ]);

        $periodDays = (int) $validated['period'];
        $server->load(['tariff', 'game', 'location']);

        $tariff = $server->tariff;
        if (! $tariff || ! ($tariff instanceof Tariff)) {
            return redirect()->route('server.show', $server)->with('error', 'Тариф не найден');
        }

        $renewalPeriods = $tariff->renewal_periods;
        if (is_array($renewalPeriods) && count($renewalPeriods) > 0) {
            $allowed = array_map('intval', $renewalPeriods);
            if (! in_array($periodDays, $allowed, true)) {
                return redirect()->route('server.show', $server)->with('error', 'Этот период продления недоступен для тарифа');
            }
        }

        $billingType = (string) $tariff->billing_type;
        $factor = $periodDays / 30;

        $slots = (int) $server->slots;
        $slots = max((int) $tariff->min_slots, $slots);
        $slots = min((int) $tariff->max_slots, $slots);

        $cpu = (float) $server->cpu_cores;
        $ram = (float) $server->ram_gb;
        $disk = (float) $server->disk_gb;

        $base = 0.0;
        if ($billingType !== 'slots') {
            $base = (float) (((float) $tariff->base_price_monthly) * $factor);
        }

        $extra = 0.0;
        if ($billingType === 'slots') {
            $extra = (float) ($slots * (float) $tariff->price_per_slot * $factor);
        } else {
            $extra = (float) (((
                ((float) $cpu * (float) $tariff->price_per_cpu_core)
                + ((float) $ram * (float) $tariff->price_per_ram_gb)
                + ((float) $disk * (float) $tariff->price_per_disk_gb)
            )) * $factor);
        }

        $addons = 0.0;
        if ((bool) $server->antiddos_enabled) {
            $addons += (float) ((float) $tariff->antiddos_price * $factor);
        }

        $cost = $base + $extra + $addons;
        $cost = round($cost, 2);
        if ($cost <= 0) {
            return redirect()->route('server.show', $server)->with('error', 'Не удалось рассчитать стоимость продления');
        }

        $promoCodeInput = trim((string) $validated['promo_code']);
        $promoCodeInput = $promoCodeInput !== '' ? mb_strtoupper($promoCodeInput) : null;

        $promoResult = app(PromotionService::class)->pickPromotion(
            PromotionService::APPLY_RENEW,
            Auth::user(),
            $promoCodeInput,
            (int) $tariff->id ?: null,
            (int) $tariff->game_id ?: null,
            (int) $tariff->location_id ?: null,
            (float) $cost
        );

        if ($promoResult['error'] && $promoCodeInput !== null) {
            return redirect()->route('server.show', $server)->with('error', (string) $promoResult['error']);
        }

        $promotion = $promoResult['promotion'];
        if ($promotion) {
            $applied = app(PromotionService::class)->applyToAmount($promotion, PromotionService::APPLY_RENEW, (float) $cost);
            $cost = (float) $applied['final_amount'];
        }

        $cost = round((float) $cost, 2);
        if ($cost <= 0) {
            return redirect()->route('server.show', $server)->with('error', 'Не удалось рассчитать стоимость продления');
        }

        $userId = (int) $server->user_id;
        $message = null;

        try {
            DB::transaction(function () use ($server, $userId, $cost, $periodDays, $promotion) {
                $u = User::whereKey($userId)->lockForUpdate()->first();
                $s = Server::whereKey((int) $server->id)->lockForUpdate()->first();

                if (! $u || ! $s) {
                    throw new \RuntimeException('not found');
                }

                if ((float) $u->balance < $cost) {
                    throw new \RuntimeException('Недостаточно средств на балансе');
                }

                $baseDate = $s->expires_at && now()->lessThan($s->expires_at) ? $s->expires_at : now();
                $newExpires = $baseDate->copy()->addDays($periodDays);

                $u->decrement('balance', $cost);

                if ($promotion) {
                    app(PromotionService::class)->lockAndIncrementUsage((int) $promotion->id);
                }
                $s->update([
                    'expires_at' => $newExpires,
                    'status' => 'active',
                ]);

                Transaction::create([
                    'user_id' => $u->id,
                    'type' => 'debit',
                    'amount' => $cost,
                    'description' => 'Продление сервера #' . $s->id . ' на ' . $periodDays . ' дней',
                ]);
            });
        } catch (\Throwable $e) {
            $msg = $e->getMessage();
            if ($msg === 'not found') {
                $msg = 'Сервер или пользователь не найден';
            }
            return redirect()->route('server.show', $server)->with('error', $msg);
        }

        $server->refresh();

        $prov = strtolower((string) $server->provisioning_status);
        $isProvisioning = in_array($prov, ['pending', 'installing', 'reinstalling'], true);
        $runtime = strtolower((string) $server->runtime_status);

        if ((bool) $server->auto_start_enabled && ! $isProvisioning && in_array($runtime, ['offline', 'stopped'], true)) {
            try {
                $server->load(['location', 'game', 'tariff']);
                $this->enforceServerSlotsWithinTariff($server);

                $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
                $result = $this->callDaemon($server, '/servers/start', [
                    'server_id' => $server->id,
                    'game' => $gameCode,
                    'cpu_cores' => (float) $server->cpu_cores,
                    'cpu_shares' => $server->cpu_shares,
                    'ram_gb' => (float) $server->ram_gb,
                    'disk_gb' => (float) $server->disk_gb,
                    'slots' => (int) $server->slots,
                    'server_fps' => $server->server_fps,
                    'server_tickrate' => $server->server_tickrate,
                    'antiddos_enabled' => (bool) $server->antiddos_enabled,
                ], 30);

                if ((bool) $result['ok'] === true) {
                    $server->update([
                        'runtime_status' => $this->mapRuntimeStatus($result['state']),
                        'port' => (int) $result['port'],
                    ]);
                    $message = 'Продление выполнено. Сервер автоматически запущен.';
                } else {
                    $message = 'Продление выполнено. Автозапуск не удался: ' . (string) $result['error'];
                }
            } catch (\Throwable $e) {
                $message = 'Продление выполнено. Автозапуск не удался.';
            }
        }

        return redirect()->route('server.show', $server)->with('success', $message ?: 'Продление выполнено');
    }

    public function reinstall(Request $request, Server $server): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_reinstall');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location', 'game', 'tariff', 'gameVersion']);

        $this->enforceServerSlotsWithinTariff($server);

        $server->update([
            'provisioning_status' => 'reinstalling',
            'runtime_status' => 'reinstalling',
            'provisioning_error' => null,
        ]);

        dispatch(new ReinstallServer((int) $server->id));

        return redirect()->route('server.show', $server)->with('success', 'Переустановка запущена');
    }

    public function resetFtpPassword(Server $server): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_view_ftp');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location']);

        $ftpUser = (string) ($server->ftp_username ?: ('srv' . $server->id));
        $ftpPass = Str::password(16, true, true, false, false);

        $result = $this->callDaemon($server, '/servers/ftp/create-user', [
            'server_id' => $server->id,
            'username' => $ftpUser,
            'password' => $ftpPass,
        ]);

        if (! $result['ok']) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'ftp'])->with('error', $result['error']);
        }

        $publicIp = (string) ($server->location->ip_address ?: $server->location->ssh_host);

        $server->update([
            'ftp_host' => $publicIp,
            'ftp_port' => (int) $result['port'],
            'ftp_username' => $ftpUser,
            'ftp_password' => Crypt::encryptString($ftpPass),
            'ftp_root' => (string) $result['root'],
        ]);

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'ftp'])->with('success', 'Пароль FTP обновлён');
    }

    public function resetMySqlPassword(Server $server): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_view_mysql');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location']);

        $dbName = (string) ($server->mysql_database ?: ('srv' . $server->id));
        $dbUser = (string) ($server->mysql_username ?: ('srv' . $server->id));
        $dbPass = Str::password(16, true, true, false, false);

        $result = $this->callDaemon($server, '/servers/mysql/create-db', [
            'server_id' => $server->id,
            'database' => $dbName,
            'username' => $dbUser,
            'password' => $dbPass,
        ], 60);

        if (! $result['ok']) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'mysql'])->with('error', $result['error']);
        }

        $publicIp = (string) (($server->location->mysql_host ?: $server->location->ip_address) ?: $server->location->ssh_host);
        $server->update([
            'mysql_host' => $publicIp,
            'mysql_port' => (int) ($server->location->mysql_port ?: $result['port']),
            'mysql_database' => $dbName,
            'mysql_username' => $dbUser,
            'mysql_password' => Crypt::encryptString($dbPass),
        ]);

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'mysql'])->with('success', 'Пароль MySQL обновлён');
    }

    public function configureFastdl(Server $server): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location']);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if (! in_array($gameCode, ['cs16', 'cstrike', 'counter-strike', 'counter_strike', 'cs_1_6', 'cs:s', 'cs_source', 'counter-strike_source', 'counter_strike_source', 'tf2', 'teamfortress2', 'team_fortress_2', 'tf', 'gmod', 'garrysmod', "garry's mod", 'garrys_mod'], true)) {
            return response()->json(['ok' => false, 'error' => 'FastDL поддерживается только для CS 1.6, CS:S, TF2, Garry\'s Mod'], 400);
        }

        $fastdlBaseUrl = config('app.fastdl_base_url', 'http://' . ($server->location->ip_address ?: $server->location->ssh_host) . '/fastdl');
        $svDownloadurl = rtrim((string) $fastdlBaseUrl, '/') . '/' . $server->id . '/';

        $cfgPath = match ($gameCode) {
            'cs16', 'cstrike', 'counter-strike', 'counter_strike', 'cs_1_6' => 'cstrike/server.cfg',
            'cs:s', 'cs_source', 'counter-strike_source', 'counter_strike_source' => 'cstrike/cfg/server.cfg',
            'tf2', 'teamfortress2', 'team_fortress_2', 'tf' => 'tf/cfg/server.cfg',
            'gmod', 'garrysmod', "garry's mod", 'garrys_mod' => 'garrysmod/cfg/server.cfg',
            default => 'server.cfg',
        };

        $raw = $this->readDaemonFileContent($server, $cfgPath);
        $updated = $this->applyCs16ServerCfgChanges($raw, [
            'sv_downloadurl' => $svDownloadurl,
            'sv_consistency' => '1',
            'sv_allowupload' => '1',
            'sv_allowdownload' => '1',
        ]);

        $write = $this->writeDaemonFile($server, $cfgPath, $updated, 60);
        if (! (bool) $write['ok']) {
            return response()->json(['ok' => false, 'error' => 'Не удалось сохранить server.cfg: ' . (string) $write['error']], 500);
        }

        return response()->json(['ok' => true]);
    }

    public function updateFastdl(Server $server): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);

        $result = $this->callDaemon($server, '/servers/fastdl/sync', [
            'server_id' => $server->id,
        ], 120);

        if (! $result['ok']) {
            return response()->json(['ok' => false, 'error' => $result['error']], 502);
        }

        return response()->json($result);
    }

    public function updateSampSettings(Server $server, Request $request): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location', 'tariff']);

        $this->enforceServerSlotsWithinTariff($server);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if ($gameCode !== 'samp') {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Настройки для этой игры пока не поддерживаются');
        }

        $validated = $request->validate([
            'hostname' => ['nullable', 'string', 'max:128'],
            'weburl' => ['nullable', 'string', 'max:255'],
            'rcon_password' => ['nullable', 'string', 'max:128'],
            'maxplayers' => ['nullable', 'integer', 'min:1', 'max:1000'],
            'lanmode' => ['nullable', 'in:0,1'],
            'announce' => ['nullable', 'in:0,1'],
            'query' => ['nullable', 'in:0,1'],
            'rcon' => ['nullable', 'in:0,1'],

            'logtimeformat' => ['nullable', 'string', 'max:128'],
            'logqueries' => ['nullable', 'in:0,1'],
            'logbans' => ['nullable', 'in:0,1'],

            'onfoot_rate' => ['nullable', 'integer', 'min:1', 'max:1000'],
            'incar_rate' => ['nullable', 'integer', 'min:1', 'max:1000'],
            'weapon_rate' => ['nullable', 'integer', 'min:1', 'max:1000'],
            'stream_distance' => ['nullable', 'numeric', 'min:0', 'max:10000'],
            'stream_rate' => ['nullable', 'integer', 'min:0', 'max:1000000'],
            'maxnpc' => ['nullable', 'integer', 'min:0', 'max:10000'],

            'anticheat' => ['nullable', 'in:0,1'],
            'lagcompmode' => ['nullable', 'in:0,1'],
            'connseedtime' => ['nullable', 'integer', 'min:0', 'max:10000000'],
            'minconnectiontime' => ['nullable', 'integer', 'min:0', 'max:10000000'],
            'messageholelimit' => ['nullable', 'integer', 'min:0', 'max:10000000'],
            'ackslimit' => ['nullable', 'integer', 'min:0', 'max:10000000'],
            'playertimeout' => ['nullable', 'integer', 'min:0', 'max:10000000'],

            'worldtime' => ['nullable', 'integer', 'min:0', 'max:23'],
            'sleep' => ['nullable', 'integer', 'min:0', 'max:1000'],
            'mtu' => ['nullable', 'integer', 'min:0', 'max:10000'],
        ]);

        if ((string) $server->tariff->billing_type === 'slots') {
            $validated['maxplayers'] = (int) $server->slots;
        }

        $read = $this->readDaemonFile($server, 'server.cfg');

        if (! (bool) $read['ok'] || ! is_string($read['content'])) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Не удалось прочитать server.cfg: ' . (string) $read['error']);
        }

        $raw = (string) $read['content'];
        $updated = $this->applySampServerCfgChanges($raw, $validated);

        if ((string) $server->tariff->billing_type === 'slots') {
            $updated = $this->enforceMaxPlayersInContent('server.cfg', $updated, (int) $server->slots);
        }

        $write = $this->writeDaemonFile($server, 'server.cfg', $updated, 60);

        if (! (bool) $write['ok']) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Не удалось сохранить server.cfg: ' . (string) $write['error']);
        }

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('success', 'Настройки сохранены');
    }

    public function updateUnturnedSettings(Server $server, Request $request): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location', 'tariff']);

        $this->enforceServerSlotsWithinTariff($server);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if (! in_array($gameCode, ['unturned', 'unturn', 'ut', 'untrm4', 'untrm5'], true)) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Настройки для этой игры пока не поддерживаются');
        }

        $validated = $request->validate([
            'server_name' => ['nullable', 'string', 'max:64'],
            'login_token' => ['nullable', 'string', 'max:128'],
            'max_players' => ['nullable', 'integer', 'min:1', 'max:200'],
            'password' => ['nullable', 'string', 'max:64'],
        ]);

        if ((string) $server->tariff->billing_type === 'slots') {
            $validated['max_players'] = (int) $server->slots;
        }

        $cfgPath = 'Servers/server/Server/Commands.dat';

        $read = $this->callDaemon($server, '/servers/files/read', [
            'server_id' => $server->id,
            'path' => $cfgPath,
        ]);

        $raw = '';
        if ((bool) $read['ok'] && is_string($read['content'])) {
            $raw = (string) $read['content'];
        }

        $updated = $this->applyUnturnedCommandsDatChanges($raw, $validated);

        $write = $this->callDaemon($server, '/servers/files/write', [
            'server_id' => $server->id,
            'path' => $cfgPath,
            'content' => $updated,
        ], 60);

        if (! (bool) $write['ok']) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Не удалось сохранить Commands.dat: ' . (string) $write['error']);
        }

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('success', 'Настройки сохранены');
    }

    public function updateRustSettings(Server $server, Request $request): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location', 'tariff']);

        $this->enforceServerSlotsWithinTariff($server);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if ($gameCode !== 'rust') {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Настройки для этой игры пока не поддерживаются');
        }

        $validated = $request->validate([
            'hostname' => ['nullable', 'string', 'max:128'],
            'maxplayers' => ['nullable', 'integer', 'min:1', 'max:500'],
            'world_size' => ['nullable', 'integer', 'min:1000', 'max:6000'],
            'seed' => ['nullable', 'integer', 'min:0', 'max:2147483647'],
            'identity' => ['nullable', 'string', 'max:64'],
            'level' => ['nullable', 'string', 'max:128'],
            'rcon_password' => ['nullable', 'string', 'max:128'],
        ]);

        if ((string) $server->tariff->billing_type === 'slots') {
            $validated['maxplayers'] = (int) $server->slots;
        }

        $env = [];
        if ((string) $validated['hostname'] !== '') {
            $env['RUST_HOSTNAME'] = (string) $validated['hostname'];
        }
        $env['RUST_MAXPLAYERS'] = (string) ((int) $validated['maxplayers']);
        $env['RUST_WORLD_SIZE'] = (string) ((int) $validated['world_size']);
        $env['RUST_SEED'] = (string) ((int) $validated['seed']);
        if ((string) $validated['identity'] !== '') {
            $env['RUST_IDENTITY'] = (string) $validated['identity'];
        }
        if ((string) $validated['level'] !== '') {
            $env['RUST_LEVEL'] = (string) $validated['level'];
        }
        if ((string) $validated['rcon_password'] !== '') {
            $env['RUST_RCON_PASSWORD'] = (string) $validated['rcon_password'];
        }

        $content = $this->renderEnvFile($env);

        $write = $this->callDaemon($server, '/servers/files/write', [
            'server_id' => $server->id,
            'path' => 'rust.env',
            'content' => $content,
        ], 60);

        if (! (bool) $write['ok']) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Не удалось сохранить rust.env: ' . (string) $write['error']);
        }

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('success', 'Настройки сохранены');
    }

    public function updateCs2Settings(Server $server, Request $request): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location', 'tariff']);

        $this->enforceServerSlotsWithinTariff($server);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if (! in_array($gameCode, ['cs2', 'counter-strike2', 'counter_strike2', 'counter-strike_2', 'counter_strike_2'], true)) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Настройки для этой игры пока не поддерживаются');
        }

        $validated = $request->validate([
            'hostname' => ['nullable', 'string', 'max:128'],
            'rcon_password' => ['nullable', 'string', 'max:128'],
            'sv_setsteamaccount' => ['nullable', 'string', 'max:128'],
            'sv_password' => ['nullable', 'string', 'max:128'],
        ]);

        $cfgPath = 'game/csgo/cfg/server.cfg';

        $raw = $this->readDaemonFileContent($server, $cfgPath);
        $updated = $this->applyCs16ServerCfgChanges($raw, $validated);
        $write = $this->writeDaemonFile($server, $cfgPath, $updated, 60);
        if (! (bool) $write['ok']) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Не удалось сохранить server.cfg: ' . (string) $write['error']);
        }

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('success', 'Настройки сохранены');
    }

    public function updateTf2Settings(Server $server, Request $request): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location', 'tariff']);

        $this->enforceServerSlotsWithinTariff($server);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if (! in_array($gameCode, ['tf2', 'teamfortress2', 'team_fortress_2', 'tf'], true)) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Настройки для этой игры пока не поддерживаются');
        }

        $validated = $request->validate([
            'hostname' => ['nullable', 'string', 'max:128'],
            'rcon_password' => ['nullable', 'string', 'max:128'],
            'sv_password' => ['nullable', 'string', 'max:128'],
            'sv_lan' => ['nullable', 'in:0,1'],
            'sv_contact' => ['nullable', 'string', 'max:128'],
            'sv_tags' => ['nullable', 'string', 'max:128'],
            'tf_bot_quota' => ['nullable', 'integer', 'min:0', 'max:1000'],
            'sv_downloadurl' => ['nullable', 'string', 'max:255'],
        ]);

        $cfgPath = 'tf/cfg/server.cfg';

        $raw = $this->readDaemonFileContent($server, $cfgPath);
        $updated = $this->applyCs16ServerCfgChanges($raw, $validated);
        $write = $this->writeDaemonFile($server, $cfgPath, $updated, 60);
        if (! (bool) $write['ok']) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Не удалось сохранить server.cfg: ' . (string) $write['error']);
        }

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('success', 'Настройки сохранены');
    }

    public function updateCssSettings(Server $server, Request $request): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location', 'tariff']);

        $this->enforceServerSlotsWithinTariff($server);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if (! in_array($gameCode, ['css', 'cs:s', 'cs_source', 'counter-strike_source', 'counter_strike_source'], true)) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Настройки для этой игры пока не поддерживаются');
        }

        $validated = $request->validate([
            'hostname' => ['nullable', 'string', 'max:128'],
            'rcon_password' => ['nullable', 'string', 'max:128'],
            'sv_password' => ['nullable', 'string', 'max:128'],
            'sv_lan' => ['nullable', 'in:0,1'],
            'sv_region' => ['nullable', 'integer', 'min:-1', 'max:255'],
            'sv_contact' => ['nullable', 'string', 'max:128'],
            'sv_tags' => ['nullable', 'string', 'max:128'],
            'sv_downloadurl' => ['nullable', 'string', 'max:255'],
        ]);

        $cfgPath = 'cstrike/cfg/server.cfg';

        $raw = $this->readDaemonFileContent($server, $cfgPath);
        $updated = $this->applyCs16ServerCfgChanges($raw, $validated);
        $write = $this->writeDaemonFile($server, $cfgPath, $updated, 60);
        if (! (bool) $write['ok']) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Не удалось сохранить server.cfg: ' . (string) $write['error']);
        }

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('success', 'Настройки сохранены');
    }

    public function updateGmodSettings(Server $server, Request $request): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location', 'tariff']);

        $this->enforceServerSlotsWithinTariff($server);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if (! in_array($gameCode, ['gmod', 'garrysmod', "garry's mod", 'garrys_mod'], true)) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Настройки для этой игры пока не поддерживаются');
        }

        $validated = $request->validate([
            'hostname' => ['nullable', 'string', 'max:128'],
            'rcon_password' => ['nullable', 'string', 'max:128'],
            'sv_password' => ['nullable', 'string', 'max:128'],
            'sv_lan' => ['nullable', 'in:0,1'],
            'sv_region' => ['nullable', 'integer', 'min:-1', 'max:255'],
            'sv_contact' => ['nullable', 'string', 'max:128'],
            'sv_tags' => ['nullable', 'string', 'max:128'],
            'sv_downloadurl' => ['nullable', 'string', 'max:255'],
        ]);

        $cfgPath = 'garrysmod/cfg/server.cfg';

        $raw = $this->readDaemonFileContent($server, $cfgPath);
        $updated = $this->applyCs16ServerCfgChanges($raw, $validated);
        $write = $this->writeDaemonFile($server, $cfgPath, $updated, 60);
        if (! (bool) $write['ok']) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Не удалось сохранить server.cfg: ' . (string) $write['error']);
        }

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('success', 'Настройки сохранены');
    }

    public function updateCrmpSettings(Server $server, Request $request): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location', 'tariff']);

        $this->enforceServerSlotsWithinTariff($server);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if ($gameCode !== 'crmp') {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Настройки для этой игры пока не поддерживаются');
        }

        $validated = $request->validate([
            'hostname' => ['nullable', 'string', 'max:128'],
            'weburl' => ['nullable', 'string', 'max:255'],
            'rcon_password' => ['nullable', 'string', 'max:128'],
            'maxplayers' => ['nullable', 'integer', 'min:1', 'max:1000'],
            'lanmode' => ['nullable', 'in:0,1'],
            'announce' => ['nullable', 'in:0,1'],
            'query' => ['nullable', 'in:0,1'],
            'rcon' => ['nullable', 'in:0,1'],

            'logtimeformat' => ['nullable', 'string', 'max:128'],
            'logqueries' => ['nullable', 'in:0,1'],
            'logbans' => ['nullable', 'in:0,1'],

            'onfoot_rate' => ['nullable', 'integer', 'min:1', 'max:1000'],
            'incar_rate' => ['nullable', 'integer', 'min:1', 'max:1000'],
            'weapon_rate' => ['nullable', 'integer', 'min:1', 'max:1000'],
            'stream_distance' => ['nullable', 'numeric', 'min:0', 'max:10000'],
            'stream_rate' => ['nullable', 'integer', 'min:0', 'max:1000000'],
            'maxnpc' => ['nullable', 'integer', 'min:0', 'max:10000'],

            'anticheat' => ['nullable', 'in:0,1'],
            'lagcompmode' => ['nullable', 'in:0,1'],
            'connseedtime' => ['nullable', 'integer', 'min:0', 'max:10000000'],
            'minconnectiontime' => ['nullable', 'integer', 'min:0', 'max:10000000'],
            'messageholelimit' => ['nullable', 'integer', 'min:0', 'max:10000000'],
            'ackslimit' => ['nullable', 'integer', 'min:0', 'max:10000000'],
            'playertimeout' => ['nullable', 'integer', 'min:0', 'max:10000000'],

            'worldtime' => ['nullable', 'integer', 'min:0', 'max:23'],
            'sleep' => ['nullable', 'integer', 'min:0', 'max:1000'],
            'mtu' => ['nullable', 'integer', 'min:0', 'max:10000'],
        ]);

        if ((string) $server->tariff->billing_type === 'slots') {
            $validated['maxplayers'] = (int) $server->slots;
        }

        $read = $this->readDaemonFile($server, 'server.cfg');

        if (! (bool) $read['ok'] || ! is_string($read['content'])) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Не удалось прочитать server.cfg: ' . (string) $read['error']);
        }

        $raw = (string) $read['content'];
        $updated = $this->applySampServerCfgChanges($raw, $validated);

        if ((string) $server->tariff->billing_type === 'slots') {
            $updated = $this->enforceMaxPlayersInContent('server.cfg', $updated, (int) $server->slots);
        }

        $write = $this->writeDaemonFile($server, 'server.cfg', $updated, 60);

        if (! (bool) $write['ok']) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Не удалось сохранить server.cfg: ' . (string) $write['error']);
        }

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('success', 'Настройки сохранены');
    }

    public function updateMinecraftSettings(Server $server, Request $request): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location', 'tariff']);

        $this->enforceServerSlotsWithinTariff($server);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if (! in_array($gameCode, ['mcjava', 'mcpaper', 'mcspigot', 'mcforge', 'mcfabric', 'mcbedrock', 'mcbedrk', 'bedrock'], true)) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Настройки для этой игры пока не поддерживаются');
        }

        $validated = $request->validate([
            'motd' => ['nullable', 'string', 'max:255'],
            'level_name' => ['nullable', 'string', 'max:64'],
            'max_players' => ['nullable', 'integer', 'min:1', 'max:5000'],
            'online_mode' => ['nullable', 'in:true,false'],
            'pvp' => ['nullable', 'in:true,false'],
            'difficulty' => ['nullable', 'in:peaceful,easy,normal,hard'],
            'gamemode' => ['nullable', 'in:survival,creative,adventure,spectator'],
            'hardcore' => ['nullable', 'in:true,false'],
            'force_gamemode' => ['nullable', 'in:true,false'],
            'allow_flight' => ['nullable', 'in:true,false'],
            'view_distance' => ['nullable', 'integer', 'min:2', 'max:64'],
            'simulation_distance' => ['nullable', 'integer', 'min:2', 'max:64'],
            'spawn_protection' => ['nullable', 'integer', 'min:0', 'max:999999'],
            'allow_nether' => ['nullable', 'in:true,false'],
            'enable_command_block' => ['nullable', 'in:true,false'],
            'white_list' => ['nullable', 'in:true,false'],
            'enforce_whitelist' => ['nullable', 'in:true,false'],
            'enable_status' => ['nullable', 'in:true,false'],
            'enable_query' => ['nullable', 'in:true,false'],
            'query_port' => ['nullable', 'integer', 'min:1', 'max:65535'],
            'enable_rcon' => ['nullable', 'in:true,false'],
            'rcon_password' => ['nullable', 'string', 'max:128'],
            'rcon_port' => ['nullable', 'integer', 'min:1', 'max:65535'],
            'player_idle_timeout' => ['nullable', 'integer', 'min:0', 'max:999999'],
            'max_tick_time' => ['nullable', 'integer', 'min:-1', 'max:2147483647'],
            'max_world_size' => ['nullable', 'integer', 'min:1', 'max:29999984'],
            'op_permission_level' => ['nullable', 'integer', 'min:1', 'max:4'],
            'function_permission_level' => ['nullable', 'integer', 'min:1', 'max:4'],
            'network_compression_threshold' => ['nullable', 'integer', 'min:-1', 'max:1048576'],
        ]);

        if ((string) $server->tariff->billing_type === 'slots') {
            $validated['max_players'] = (int) $server->slots;
        }

        $read = $this->callDaemon($server, '/servers/files/read', [
            'server_id' => $server->id,
            'path' => 'server.properties',
        ]);

        $raw = '';
        if ((bool) $read['ok'] && is_string($read['content'])) {
            $raw = (string) $read['content'];
        }

        $updated = $this->applyMinecraftServerPropertiesChanges($raw, $validated);

        if ((string) $server->tariff->billing_type === 'slots') {
            $updated = $this->enforceMaxPlayersInContent('server.properties', $updated, (int) $server->slots);
        }

        $write = $this->callDaemon($server, '/servers/files/write', [
            'server_id' => $server->id,
            'path' => 'server.properties',
            'content' => $updated,
        ], 60);

        if (! (bool) $write['ok']) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Не удалось сохранить server.properties: ' . (string) $write['error']);
        }

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('success', 'Настройки сохранены');
    }

    public function updateCs16Settings(Server $server, Request $request): RedirectResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_settings_edit');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location']);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if (! in_array($gameCode, ['cs16', 'cstrike', 'counter-strike', 'counter_strike', 'cs_1_6'], true)) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Настройки для этой игры пока не поддерживаются');
        }

        $validated = $request->validate([
            'cfg_path' => ['nullable', 'string', 'in:cstrike/server.cfg,server.cfg'],
            'hostname' => ['nullable', 'string', 'max:128'],
            'rcon_password' => ['nullable', 'string', 'max:128'],
            'sv_password' => ['nullable', 'string', 'max:128'],
            'sv_lan' => ['nullable', 'in:0,1'],
            'mp_timelimit' => ['nullable', 'integer', 'min:0', 'max:10000'],
            'mp_roundtime' => ['nullable', 'integer', 'min:0', 'max:10000'],
            'mp_friendlyfire' => ['nullable', 'in:0,1'],
            'mp_autokick' => ['nullable', 'in:0,1'],
            'mp_autoteambalance' => ['nullable', 'in:0,1'],
            'mp_limitteams' => ['nullable', 'integer', 'min:0', 'max:1000'],
        ]);

        $cfgPath = (string) $validated['cfg_path'];
        if ($cfgPath === '') {
            $cfgPath = 'cstrike/server.cfg';
        }

        $raw = $this->readDaemonFileContent($server, $cfgPath);

        $updated = $this->applyCs16ServerCfgChanges($raw, $validated);

        $write = $this->writeDaemonFile($server, $cfgPath, $updated, 60);

        if (! (bool) $write['ok']) {
            return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('error', 'Не удалось сохранить server.cfg: ' . (string) $write['error']);
        }

        return redirect()->route('server.show', ['server' => $server, 'tab' => 'settings'])->with('success', 'Настройки сохранены');
    }

    public function filesList(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_files');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location']);
        $this->enforceServerSlotsWithinTariff($server);

        $path = (string) $request->input('path', '');
        $content = (string) $request->input('content', '');

        $result = $this->callDaemon($server, '/servers/files/list', [
            'server_id' => $server->id,
            'path' => $path,
        ]);

        if (! $result['ok']) {
            return response()->json(['ok' => false, 'error' => $result['error']], 502);
        }

        return response()->json($result);
    }

    public function filesRead(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_files');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);

        $path = (string) $request->input('path', '');
        $result = $this->callDaemon($server, '/servers/files/read', [
            'server_id' => $server->id,
            'path' => $path,
        ]);

        if (! $result['ok']) {
            return response()->json(['ok' => false, 'error' => $result['error']], 502);
        }

        return response()->json($result);
    }

    public function filesWrite(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_files');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['tariff']);
        $this->enforceServerSlotsWithinTariff($server);

        $path = (string) $request->input('path', '');
        $content = (string) $request->input('content', '');

        if ($path !== '' && (string) $server->tariff->billing_type === 'slots') {
            $content = $this->enforceMaxPlayersInContent($path, $content, (int) $server->slots);
        }

        $result = $this->callDaemon($server, '/servers/files/write', [
            'server_id' => $server->id,
            'path' => $path,
            'content' => $content,
        ], 60);

        if (! $result['ok']) {
            return response()->json(['ok' => false, 'error' => $result['error']], 502);
        }

        return response()->json($result);
    }

    public function filesMkdir(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_files');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);

        $path = (string) $request->input('path', '');
        $result = $this->callDaemon($server, '/servers/files/mkdir', [
            'server_id' => $server->id,
            'path' => $path,
        ]);

        if (! $result['ok']) {
            return response()->json(['ok' => false, 'error' => $result['error']], 502);
        }

        return response()->json($result);
    }

    public function filesDelete(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_files');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);

        $path = (string) $request->input('path', '');
        $recursive = (bool) $request->input('recursive', false);

        $result = $this->callDaemon($server, '/servers/files/delete', [
            'server_id' => $server->id,
            'path' => $path,
            'recursive' => $recursive,
        ], 120);

        if (! $result['ok']) {
            return response()->json(['ok' => false, 'error' => $result['error']], 502);
        }

        return response()->json($result);
    }

    public function filesDownload(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_files');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);

        $path = (string) $request->input('path', '');
        $result = $this->callDaemon($server, '/servers/files/download', [
            'server_id' => $server->id,
            'path' => $path,
        ], 120);

        if (! $result['ok']) {
            return response()->json(['ok' => false, 'error' => $result['error']], 502);
        }

        return response()->json($result);
    }

    public function filesUpload(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_files');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['location']);

        $path = (string) $request->input('path', '');
        $file = $request->file('file');
        if (! $file || ! $file->isValid()) {
            return response()->json(['ok' => false, 'error' => 'Invalid file'], 400);
        }

        $maxSize = 50 * 1024 * 1024;
        if ($file->getSize() !== null && $file->getSize() > $maxSize) {
            return response()->json(['ok' => false, 'error' => 'File too large'], 400);
        }

        $host = (string) $server->location->ssh_host;
        if ($host === '') {
            return response()->json(['ok' => false, 'error' => 'Для локации не настроен хост демона'], 400);
        }

        $daemonPort = (int) config('services.location_daemon.port', 9201);
        $daemonToken = (string) config('services.location_daemon.token', '');
        $url = sprintf('http://%s:%d%s', $host, $daemonPort, '/servers/files/upload');

        try {
            $http = Http::timeout(120);
            if ($daemonToken !== '') {
                $http = $http->withHeaders([
                    'X-Location-Daemon-Token' => $daemonToken,
                ]);
            }

            $resp = $http
                ->attach('file', file_get_contents($file->getRealPath()), $file->getClientOriginalName())
                ->post($url, [
                    'server_id' => (string) $server->id,
                    'path' => $path,
                ]);

            if (! $resp->successful()) {
                return response()->json(['ok' => false, 'error' => 'Daemon returned status ' . $resp->status()], 502);
            }

            $data = $resp->json();
            if (! is_array($data) || ! (bool) $data['ok']) {
                return response()->json(['ok' => false, 'error' => (string) $data['error']], 502);
            }

            return response()->json($data);
        } catch (\Throwable $e) {
            return response()->json(['ok' => false, 'error' => $e->getMessage()], 502);
        }
    }

    public function cronList(Server $server): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_view_cron');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location']);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if ($gameCode === '') {
            return response()->json(['ok' => false, 'error' => 'Не удалось определить game code'], 400);
        }

        $result = $this->callDaemon($server, '/servers/cron/list', [
            'server_id' => (int) $server->id,
            'game' => $gameCode,
        ], 20);

        if (! (bool) $result['ok']) {
            return response()->json(['ok' => false, 'error' => (string) $result['error']], 502);
        }

        return response()->json($result);
    }

    public function cronCreate(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_cron_manage');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location']);

        $validated = $request->validate([
            'name' => ['nullable', 'string', 'max:64'],
            'schedule' => ['required', 'string', 'max:128'],
            'command' => ['required', 'string', 'max:512'],
            'enabled' => ['sometimes', 'boolean'],
        ]);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if ($gameCode === '') {
            return response()->json(['ok' => false, 'error' => 'Не удалось определить game code'], 400);
        }

        $result = $this->callDaemon($server, '/servers/cron/create', [
            'server_id' => (int) $server->id,
            'game' => $gameCode,
            'name' => (string) $validated['name'],
            'schedule' => (string) $validated['schedule'],
            'command' => (string) $validated['command'],
            'enabled' => (bool) $validated['enabled'],
        ], 30);

        if (! (bool) $result['ok']) {
            return response()->json(['ok' => false, 'error' => (string) $result['error']], 502);
        }

        return response()->json($result);
    }

    public function cronDelete(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_cron_manage');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location']);

        $validated = $request->validate([
            'job_id' => ['required', 'string', 'max:64'],
        ]);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if ($gameCode === '') {
            return response()->json(['ok' => false, 'error' => 'Не удалось определить game code'], 400);
        }

        $result = $this->callDaemon($server, '/servers/cron/delete', [
            'server_id' => (int) $server->id,
            'game' => $gameCode,
            'job_id' => (string) $validated['job_id'],
        ], 30);

        if (! (bool) $result['ok']) {
            return response()->json(['ok' => false, 'error' => (string) $result['error']], 502);
        }

        return response()->json($result);
    }

    public function cronToggle(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        $this->requirePermission($server, 'can_cron_manage');
        $this->abortIfExpired($server);
        $this->abortIfReinstalling($server);
        $server->load(['game', 'location']);

        $validated = $request->validate([
            'job_id' => ['required', 'string', 'max:64'],
            'enabled' => ['required', 'boolean'],
        ]);

        $gameCode = strtolower((string) ($server->game->code ?: $server->game->slug));
        if ($gameCode === '') {
            return response()->json(['ok' => false, 'error' => 'Не удалось определить game code'], 400);
        }

        $result = $this->callDaemon($server, '/servers/cron/toggle', [
            'server_id' => (int) $server->id,
            'game' => $gameCode,
            'job_id' => (string) $validated['job_id'],
            'enabled' => (bool) $validated['enabled'],
        ], 30);

        if (! (bool) $result['ok']) {
            return response()->json(['ok' => false, 'error' => (string) $result['error']], 502);
        }

        return response()->json($result);
    }

    private function authorizeServer(Server $server): void
    {
        $authUser = auth()->user();
        if ($authUser && (bool) $authUser->is_admin) {
            return;
        }

        $uid = (int) auth()->id();
        if ($uid <= 0) {
            abort(403, 'Доступ запрещен');
        }

        if ((int) $server->user_id === $uid) {
            return;
        }

        $perm = ServerUserPermission::query()
            ->where('server_id', (int) $server->id)
            ->where('user_id', $uid)
            ->first();

        if (! $perm) {
            abort(403, 'Доступ запрещен');
        }
    }

    private function isServerOwner(Server $server): bool
    {
        $authUser = auth()->user();
        if ($authUser && (bool) $authUser->is_admin) {
            return true;
        }
        return (int) $server->user_id === (int) auth()->id();
    }

    private function getServerPermission(Server $server): ?ServerUserPermission
    {
        $authUser = auth()->user();
        if ($authUser && (bool) $authUser->is_admin) {
            return null;
        }
        $uid = (int) auth()->id();
        if ($uid <= 0) {
            return null;
        }
        if ((int) $server->user_id === $uid) {
            return null;
        }
        return ServerUserPermission::query()
            ->where('server_id', (int) $server->id)
            ->where('user_id', $uid)
            ->first();
    }

    private function requirePermission(Server $server, string $permissionColumn): void
    {
        if ($this->isServerOwner($server)) {
            return;
        }

        $perm = $this->getServerPermission($server);
        if (! $perm) {
            abort(403, 'Доступ запрещен');
        }

        if (! array_key_exists($permissionColumn, $perm->getAttributes())) {
            abort(403, 'Доступ запрещен');
        }

        if (! (bool) $perm->{$permissionColumn}) {
            abort(403, 'Доступ запрещен');
        }
    }

    public function friendsList(Server $server): JsonResponse
    {
        $this->authorizeServer($server);
        if (! $this->isServerOwner($server)) {
            abort(403, 'Доступ запрещен');
        }

        $rows = DB::table('server_user_permissions as sup')
            ->join('users as u', 'u.id', '=', 'sup.user_id')
            ->where('sup.server_id', (int) $server->id)
            ->orderBy('sup.id', 'desc')
            ->get([
                'sup.*',
                'u.email as user_email',
                'u.name as user_name',
                'u.last_name as user_last_name',
            ]);

        return response()->json([
            'ok' => true,
            'items' => $rows,
        ]);
    }

    public function friendsAdd(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        if (! $this->isServerOwner($server)) {
            abort(403, 'Доступ запрещен');
        }

        $validated = $request->validate([
            'email' => ['required', 'email', 'max:255'],
        ]);

        $email = strtolower(trim((string) $validated['email']));
        $user = User::query()->whereRaw('LOWER(email) = ?', [$email])->first();
        if (! $user) {
            return response()->json(['ok' => false, 'error' => 'Пользователь с таким email не найден'], 404);
        }

        if ((int) $user->id === (int) $server->user_id) {
            return response()->json(['ok' => false, 'error' => 'Нельзя добавить владельца'], 400);
        }

        ServerUserPermission::query()->updateOrCreate(
            ['server_id' => (int) $server->id, 'user_id' => (int) $user->id],
            [
                'can_view_main' => true,
                'can_view_console' => false,
                'can_view_logs' => false,
                'can_view_metrics' => false,
                'can_view_ftp' => false,
                'can_view_mysql' => false,
                'can_view_cron' => false,
                'can_view_firewall' => false,
                'can_view_settings' => false,
                'can_view_friends' => false,

                'can_start' => false,
                'can_stop' => false,
                'can_restart' => false,
                'can_reinstall' => false,

                'can_console_command' => false,
                'can_files' => false,
                'can_cron_manage' => false,
                'can_firewall_manage' => false,
                'can_settings_edit' => false,
            ]
        );

        return $this->friendsList($server);
    }

    public function friendsUpdate(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        if (! $this->isServerOwner($server)) {
            abort(403, 'Доступ запрещен');
        }

        $validated = $request->validate([
            'user_id' => ['required', 'integer'],

            'can_view_main' => ['sometimes', 'boolean'],
            'can_view_console' => ['sometimes', 'boolean'],
            'can_view_logs' => ['sometimes', 'boolean'],
            'can_view_metrics' => ['sometimes', 'boolean'],
            'can_view_ftp' => ['sometimes', 'boolean'],
            'can_view_mysql' => ['sometimes', 'boolean'],
            'can_view_cron' => ['sometimes', 'boolean'],
            'can_view_firewall' => ['sometimes', 'boolean'],
            'can_view_settings' => ['sometimes', 'boolean'],
            'can_view_friends' => ['sometimes', 'boolean'],

            'can_start' => ['sometimes', 'boolean'],
            'can_stop' => ['sometimes', 'boolean'],
            'can_restart' => ['sometimes', 'boolean'],
            'can_reinstall' => ['sometimes', 'boolean'],

            'can_console_command' => ['sometimes', 'boolean'],
            'can_files' => ['sometimes', 'boolean'],
            'can_cron_manage' => ['sometimes', 'boolean'],
            'can_firewall_manage' => ['sometimes', 'boolean'],
            'can_settings_edit' => ['sometimes', 'boolean'],
        ]);

        $userId = (int) $validated['user_id'];
        $perm = ServerUserPermission::query()
            ->where('server_id', (int) $server->id)
            ->where('user_id', $userId)
            ->first();

        if (! $perm) {
            return response()->json(['ok' => false, 'error' => 'Друг не найден'], 404);
        }

        $updates = $validated;
        unset($updates['user_id']);

        $perm->fill($updates);
        $perm->save();

        return $this->friendsList($server);
    }

    public function friendsDelete(Server $server, Request $request): JsonResponse
    {
        $this->authorizeServer($server);
        if (! $this->isServerOwner($server)) {
            abort(403, 'Доступ запрещен');
        }

        $validated = $request->validate([
            'user_id' => ['required', 'integer'],
        ]);

        $userId = (int) $validated['user_id'];
        ServerUserPermission::query()
            ->where('server_id', (int) $server->id)
            ->where('user_id', $userId)
            ->delete();

        return $this->friendsList($server);
    }

    private function parseSampServerCfg(string $content): array
    {
        $cfg = [];
        $lines = preg_split('/\r\n|\r|\n/', $content);
        if (! is_array($lines)) {
            return $cfg;
        }

        foreach ($lines as $line) {
            $line = trim((string) $line);
            if ($line === '' || str_starts_with($line, '#') || str_starts_with($line, '//')) {
                continue;
            }
            if (! preg_match('/^([A-Za-z0-9_]+)\s+(.*)$/', $line, $m)) {
                continue;
            }
            $key = strtolower((string) $m[1]);
            $value = (string) $m[2];
            if (! array_key_exists($key, $cfg)) {
                $cfg[$key] = $value;
            }
        }

        return $cfg;
    }

    private function applySampServerCfgChanges(string $content, array $changes): string
    {
        $map = [];
        foreach ([
            'hostname',
            'weburl',
            'rcon_password',
            'maxplayers',
            'lanmode',
            'announce',
            'query',
            'rcon',

            'logtimeformat',
            'logqueries',
            'logbans',

            'onfoot_rate',
            'incar_rate',
            'weapon_rate',
            'stream_distance',
            'stream_rate',
            'maxnpc',

            'anticheat',
            'lagcompmode',
            'connseedtime',
            'minconnectiontime',
            'messageholelimit',
            'ackslimit',
            'playertimeout',

            'worldtime',
            'sleep',
            'mtu',
        ] as $key) {
            if (array_key_exists($key, $changes) && $changes[$key] !== null && $changes[$key] !== '') {
                $map[$key] = (string) $changes[$key];
            }
        }

        $lines = preg_split('/\r\n|\r|\n/', $content);
        if (! is_array($lines)) {
            $lines = [$content];
        }

        $seen = [];
        $removeKeys = [
            'classes',
            'spawn_protection',
            'spawn_protection_time',
            'weather',
        ];
        foreach ($lines as $i => $line) {
            $orig = (string) $line;
            $trim = ltrim($orig);
            if ($trim === '' || str_starts_with($trim, '#') || str_starts_with($trim, '//')) {
                continue;
            }
            if (! preg_match('/^([A-Za-z0-9_]+)\s+/', $trim, $m)) {
                continue;
            }
            $key = strtolower((string) $m[1]);

            if (in_array($key, $removeKeys, true)) {
                $lines[$i] = '';
                continue;
            }
            if (array_key_exists($key, $map) && ! (bool) $seen[$key]) {
                $lines[$i] = $m[1] . ' ' . $map[$key];
                $seen[$key] = true;
            }
        }

        foreach ($map as $key => $value) {
            if (! (bool) $seen[$key]) {
                $lines[] = $key . ' ' . $value;
            }
        }

        $out = implode("\n", array_map(fn ($l) => rtrim((string) $l, "\r\n"), $lines));
        return rtrim($out, "\n") . "\n";
    }

    private function parseEnvFile(string $content): array
    {
        $out = [];
        $lines = preg_split('/\r\n|\r|\n/', $content);
        if (! is_array($lines)) {
            $lines = [$content];
        }
        foreach ($lines as $line) {
            $line = trim((string) $line);
            if ($line === '' || str_starts_with($line, '#')) {
                continue;
            }
            if (! str_contains($line, '=')) {
                continue;
            }
            [$k, $v] = explode('=', $line, 2);
            $k = trim((string) $k);
            $v = trim((string) $v);
            if ($k === '') {
                continue;
            }
            if ((str_starts_with($v, '"') && str_ends_with($v, '"')) || (str_starts_with($v, "'") && str_ends_with($v, "'"))) {
                $v = substr($v, 1, -1);
            }
            $out[$k] = $v;
        }
        return $out;
    }

    private function renderEnvFile(array $vars): string
    {
        $lines = [];
        foreach ($vars as $k => $v) {
            $k = trim((string) $k);
            if ($k === '') {
                continue;
            }
            $v = (string) $v;
            $v = str_replace('"', '\\"', $v);
            $lines[] = $k . '="' . $v . '"';
        }
        $out = implode("\n", $lines);
        return rtrim($out, "\n") . "\n";
    }

    private function parseUnturnedCommandsDat(string $content): array
    {
        $out = [];
        $lines = preg_split('/\r\n|\r|\n/', $content);
        if (! is_array($lines)) {
            return $out;
        }

        foreach ($lines as $line) {
            $line = trim((string) $line);
            if ($line === '') {
                continue;
            }
            $parts = preg_split('/\s+/', $line, 2);
            if (! is_array($parts) || count($parts) < 1) {
                continue;
            }
            $key = strtolower(trim((string) $parts[0]));
            if ($key === '') {
                continue;
            }
            $val = trim((string) $parts[1]);
            $out[$key] = $val;
        }

        return $out;
    }

    private function applyUnturnedCommandsDatChanges(string $content, array $changes): string
    {
        $lines = preg_split('/\r\n|\r|\n/', $content);
        if (! is_array($lines)) {
            $lines = [$content];
        }

        $map = [];
        if (array_key_exists('server_name', $changes) && $changes['server_name'] !== null && (string) $changes['server_name'] !== '') {
            $map['name'] = (string) $changes['server_name'];
        }
        if (array_key_exists('login_token', $changes) && $changes['login_token'] !== null) {
            $map['logintoken'] = (string) $changes['login_token'];
        }
        if (array_key_exists('max_players', $changes) && $changes['max_players'] !== null && (string) $changes['max_players'] !== '') {
            $map['maxplayers'] = (string) ((int) $changes['max_players']);
        }
        if (array_key_exists('password', $changes) && $changes['password'] !== null) {
            $map['password'] = (string) $changes['password'];
        }

        $seen = [];
        foreach ($lines as $i => $line) {
            $orig = (string) $line;
            $trim = trim($orig);
            if ($trim === '') {
                continue;
            }
            $parts = preg_split('/\s+/', $trim, 2);
            if (! is_array($parts) || count($parts) < 1) {
                continue;
            }
            $key = strtolower(trim((string) $parts[0]));
            if ($key === '') {
                continue;
            }

            if (array_key_exists($key, $map) && ! (bool) $seen[$key]) {
                $val = $map[$key];
                if ($val === '' && in_array($key, ['password', 'logintoken'], true)) {
                    $lines[$i] = '';
                    $seen[$key] = true;
                    continue;
                }
                $lines[$i] = trim((string) $parts[0]) . ' ' . $val;
                $seen[$key] = true;
            }
        }

        foreach ($map as $key => $val) {
            if ((bool) $seen[$key]) {
                continue;
            }
            if ($val === '' && in_array($key, ['password', 'logintoken'], true)) {
                continue;
            }
            $lines[] = $key . ' ' . $val;
        }

        $lines = array_values(array_filter(array_map(fn ($l) => rtrim((string) $l, "\r\n"), $lines), fn ($l) => $l !== ''));
        $lower = array_map(fn ($l) => strtolower(trim((string) $l)), $lines);
        if (! in_array('internetserver', $lower, true)) {
            $lines[] = 'InternetServer';
        }

        $out = implode("\n", $lines);
        return rtrim($out, "\n") . "\n";
    }

    private function parseMinecraftServerProperties(string $content): array
    {
        $cfg = [];
        $lines = preg_split('/\r\n|\r|\n/', $content);
        if (! is_array($lines)) {
            return $cfg;
        }

        foreach ($lines as $line) {
            $line = (string) $line;
            $trim = trim($line);
            if ($trim === '' || str_starts_with($trim, '#')) {
                continue;
            }
            $pos = strpos($trim, '=');
            if ($pos === false) {
                continue;
            }
            $key = strtolower(trim(substr($trim, 0, $pos)));
            $value = trim(substr($trim, $pos + 1));
            if ($key !== '' && ! array_key_exists($key, $cfg)) {
                $cfg[$key] = $value;
            }
        }

        return $cfg;
    }

    private function applyMinecraftServerPropertiesChanges(string $content, array $changes): string
    {
        $map = [];

        $pairs = [
            'motd' => 'motd',
            'level_name' => 'level-name',
            'max_players' => 'max-players',
            'online_mode' => 'online-mode',
            'pvp' => 'pvp',
            'difficulty' => 'difficulty',
            'gamemode' => 'gamemode',
            'hardcore' => 'hardcore',
            'force_gamemode' => 'force-gamemode',
            'allow_flight' => 'allow-flight',
            'view_distance' => 'view-distance',
            'simulation_distance' => 'simulation-distance',
            'spawn_protection' => 'spawn-protection',
            'allow_nether' => 'allow-nether',
            'enable_command_block' => 'enable-command-block',
            'white_list' => 'white-list',
            'enforce_whitelist' => 'enforce-whitelist',
            'enable_status' => 'enable-status',
            'enable_query' => 'enable-query',
            'query_port' => 'query.port',
            'enable_rcon' => 'enable-rcon',
            'rcon_password' => 'rcon.password',
            'rcon_port' => 'rcon.port',
            'player_idle_timeout' => 'player-idle-timeout',
            'max_tick_time' => 'max-tick-time',
            'max_world_size' => 'max-world-size',
            'op_permission_level' => 'op-permission-level',
            'function_permission_level' => 'function-permission-level',
            'network_compression_threshold' => 'network-compression-threshold',
        ];

        foreach ($pairs as $inputKey => $propKey) {
            if (array_key_exists($inputKey, $changes) && $changes[$inputKey] !== null && $changes[$inputKey] !== '') {
                $map[strtolower($propKey)] = (string) $changes[$inputKey];
            }
        }

        $lines = preg_split('/\r\n|\r|\n/', $content);
        if (! is_array($lines)) {
            $lines = [$content];
        }

        $seen = [];
        foreach ($lines as $i => $line) {
            $orig = (string) $line;
            $trim = ltrim($orig);
            if ($trim === '' || str_starts_with($trim, '#')) {
                continue;
            }
            $pos = strpos($trim, '=');
            if ($pos === false) {
                continue;
            }

            $keyRaw = trim(substr($trim, 0, $pos));
            $keyLower = strtolower($keyRaw);
            if (array_key_exists($keyLower, $map) && ! (bool) $seen[$keyLower]) {
                $lines[$i] = $keyRaw . '=' . $map[$keyLower];
                $seen[$keyLower] = true;
            }
        }

        foreach ($map as $key => $value) {
            if (! (bool) $seen[$key]) {
                $lines[] = $key . '=' . $value;
            }
        }

        $out = implode("\n", array_map(fn ($l) => rtrim((string) $l, "\r\n"), $lines));
        return rtrim($out, "\n") . "\n";
    }

    private function parseCs16ServerCfg(string $content): array
    {
        $cfg = [];
        $lines = preg_split('/\r\n|\r|\n/', $content);
        if (! is_array($lines)) {
            return $cfg;
        }

        foreach ($lines as $line) {
            $line = trim((string) $line);
            if ($line === '' || str_starts_with($line, '//')) {
                continue;
            }

            if (! preg_match('/^([A-Za-z0-9_]+)\s+(.*)$/', $line, $m)) {
                continue;
            }

            $key = strtolower((string) $m[1]);
            $value = trim((string) $m[2]);
            $value = trim($value, "\"'");
            if (! array_key_exists($key, $cfg)) {
                $cfg[$key] = $value;
            }
        }

        return $cfg;
    }

    private function quoteCs16Value(string $value): string
    {
        $v = (string) $value;
        if ($v === '') {
            return '""';
        }
        if (preg_match('/\s|;|"/', $v)) {
            $v = str_replace('"', '\\"', $v);
            return '"' . $v . '"';
        }
        return $v;
    }

    private function quoteCs16ValueForce(string $value): string
    {
        $v = (string) $value;
        $v = str_replace('"', '\\"', $v);
        return '"' . $v . '"';
    }

    private function applyCs16ServerCfgChanges(string $content, array $changes): string
    {
        $map = [];
        foreach ([
            'hostname',
            'rcon_password',
            'sv_setsteamaccount',
            'sv_password',
            'sv_lan',
            'sv_contact',
            'sv_tags',
            'sv_downloadurl',
            'sv_allowdownload',
            'sv_allowupload',
            'sv_consistency',
            'tf_bot_quota',
            'sv_region',
            'mp_timelimit',
            'mp_roundtime',
            'mp_friendlyfire',
            'mp_autokick',
            'mp_autoteambalance',
            'mp_limitteams',
        ] as $key) {
            if (array_key_exists($key, $changes) && $changes[$key] !== null && $changes[$key] !== '') {
                $map[$key] = (string) $changes[$key];
            }
        }

        $lines = preg_split('/\r\n|\r|\n/', $content);
        if (! is_array($lines)) {
            $lines = [$content];
        }

        $seen = [];
        foreach ($lines as $i => $line) {
            $orig = (string) $line;
            $trim = ltrim($orig);
            if ($trim === '' || str_starts_with($trim, '//')) {
                continue;
            }
            if (! preg_match('/^([A-Za-z0-9_]+)\s+/', $trim, $m)) {
                continue;
            }
            $keyLower = strtolower((string) $m[1]);

            if (array_key_exists($keyLower, $map) && ! (bool) $seen[$keyLower]) {
                $val = $map[$keyLower];
                if ($keyLower === 'sv_downloadurl') {
                    $val = $this->quoteCs16ValueForce($val);
                } elseif (in_array($keyLower, ['sv_allowdownload', 'sv_allowupload', 'sv_consistency'], true)) {
                    $val = $this->quoteCs16ValueForce($val);
                } elseif (in_array($keyLower, ['hostname', 'rcon_password', 'sv_setsteamaccount', 'sv_password', 'sv_contact', 'sv_tags'], true)) {
                    $val = $this->quoteCs16Value($val);
                }
                $lines[$i] = $m[1] . ' ' . $val;
                $seen[$keyLower] = true;
            }
        }

        foreach ($map as $key => $value) {
            if (! (bool) $seen[$key]) {
                $val = $value;
                if ($key === 'sv_downloadurl') {
                    $val = $this->quoteCs16ValueForce($val);
                } elseif (in_array($key, ['sv_allowdownload', 'sv_allowupload', 'sv_consistency'], true)) {
                    $val = $this->quoteCs16ValueForce($val);
                } elseif (in_array($key, ['hostname', 'rcon_password', 'sv_password', 'sv_contact', 'sv_tags'], true)) {
                    $val = $this->quoteCs16Value($val);
                }
                $lines[] = $key . ' ' . $val;
            }
        }

        $out = implode("\n", array_map(fn ($l) => rtrim((string) $l, "\r\n"), $lines));
        return rtrim($out, "\n") . "\n";
    }

    private function readDaemonFile(Server $server, string $path): array
    {
        return $this->callDaemon($server, '/servers/files/read', [
            'server_id' => $server->id,
            'path' => $path,
        ]);
    }

    private function readDaemonFileContent(Server $server, string $path): string
    {
        $read = $this->readDaemonFile($server, $path);
        if ((bool) $read['ok'] && is_string($read['content'])) {
            return (string) $read['content'];
        }
        return '';
    }

    private function writeDaemonFile(Server $server, string $path, string $content, int $timeoutSeconds = 60): array
    {
        return $this->callDaemon($server, '/servers/files/write', [
            'server_id' => $server->id,
            'path' => $path,
            'content' => $content,
        ], $timeoutSeconds);
    }

    private function loadCfgFromDaemon(Server $server, string $path, callable $parser): array
    {
        $read = $this->readDaemonFile($server, $path);
        if ((bool) $read['ok'] && is_string($read['content'])) {
            $raw = (string) $read['content'];
            return [
                'cfgRaw' => $raw,
                'cfg' => $parser($raw),
                'cfgPath' => $path,
                'cfgError' => null,
            ];
        }

        return [
            'cfgRaw' => '',
            'cfg' => [],
            'cfgPath' => $path,
            'cfgError' => (string) $read['error'],
        ];
    }

    private function callDaemon(Server $server, string $path, array $payload, int $timeoutSeconds = 30): array
    {
        $host = (string) $server->location->ssh_host;
        if ($host === '') {
            return ['ok' => false, 'error' => 'Для локации не настроен хост демона'];
        }

        $daemonPort = (int) config('services.location_daemon.port', 9201);
        $daemonToken = (string) config('services.location_daemon.token', '');

        $url = sprintf('http://%s:%d%s', $host, $daemonPort, $path);

        try {
            $http = Http::timeout($timeoutSeconds);
            if ($daemonToken !== '') {
                $http = $http->withHeaders([
                    'X-Location-Daemon-Token' => $daemonToken,
                ]);
            }

            $response = $http->post($url, $payload);

            if (! $response->successful()) {
                $err = '';
                try {
                    $j = $response->json();
                    if (is_array($j)) {
                        $err = (string) $j['error'];
                    }
                } catch (\Throwable $e) {
                    $err = '';
                }
                $err = trim($err);
                if ($err !== '') {
                    return ['ok' => false, 'error' => $err];
                }
                return ['ok' => false, 'error' => 'Daemon returned status ' . $response->status()];
            }

            $data = $response->json();
            if (! is_array($data) || ! (bool) $data['ok']) {
                return ['ok' => false, 'error' => (string) $data['error']];
            }

            return $data;
        } catch (\Throwable $e) {
            return ['ok' => false, 'error' => $e->getMessage()];
        }
    }

    private function mapRuntimeStatus($daemonState): string
    {
        $state = strtolower((string) $daemonState);
        return match ($state) {
            'running' => 'running',
            'restarting' => 'restarting',
            'stopped', 'exited', 'dead', 'paused', 'created', 'removing' => 'offline',
            'missing' => 'missing',
            default => 'unknown',
        };
    }
}
