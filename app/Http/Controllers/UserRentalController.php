<?php

namespace App\Http\Controllers;

use App\Models\Game;
use App\Models\GameVersion;
use App\Models\Location;
use App\Models\Server;
use App\Models\Tariff;
use App\Models\Transaction;
use App\Jobs\ProvisionServer;
use App\Services\PromotionService;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\View\View;

class UserRentalController extends Controller
{
    private const RENT_RETURN_KEYS = [
        'location_id',
        'game_id',
        'game_version_id',
        'tariff_id',
        'slots',
        'cpu_cores',
        'ram_gb',
        'disk_gb',
        'antiddos_enabled',
        'ip_choice',
        'server_fps',
        'server_tickrate',
        'period',
        'promo_code',
    ];

    private const SUPPORTED_BY_DAEMON = [
        'samp', 'sa-mp', 'samp03', 'gta_samp',
        'crmp', 'gta_crmp', 'cr-mp', 'cr-mp0.3',
        'mta', 'mtasa', 'mta-sa', 'mta_sa',
        'css', 'cs:s', 'cs_source', 'counter-strike_source', 'counter_strike_source',
        'cs2', 'counter-strike2', 'counter_strike2', 'counter-strike_2', 'counter_strike_2',
        'rust',
        'cs16', 'counter-strike', 'counter_strike', 'cs_1_6', 'cstrike',
        'tf2', 'teamfortress2', 'team_fortress_2', 'tf',
        'gmod', 'garrysmod', "garry's mod", 'garrys_mod',
        'mcjava', 'mcpaper', 'mcspigot', 'mcforge', 'mcfabric',
        'mcbedrock', 'mcbedrk', 'bedrock',
        'unturned', 'unturn', 'ut', 'untrm4', 'untrm5',
    ];

    private function redirectRentWithError(array $input, string $error): RedirectResponse
    {
        return redirect()->route('rent-server', $input)->with('error', $error);
    }

    public function rent(Request $request): View
    {
        $locationsQuery = Location::where('is_active', true)
            ->whereHas('tariffs', function ($q) {
                $q->where('is_available', true);
            });

        if ($request->filled('game_id')) {
            $locationsQuery->whereHas('tariffs', function ($q) use ($request) {
                $q->where('is_available', true)
                    ->where('game_id', $request->game_id);
            });
        }

        $locations = $locationsQuery->orderBy('sort_order')->get();

        $gamesQuery = Game::where('is_active', true);

        $gamesQuery->whereHas('tariffs', function ($q) use ($request) {
            $q->where('is_available', true);

            if ($request->filled('location_id')) {
                $q->where('location_id', $request->location_id);
            }
        });

        $games = $gamesQuery->orderBy('name')->get();

        $tariffs = collect();
        $selectedTariff = null;
        $selectedLocation = null;
        $gameVersions = collect();
        $selectedGameVersion = null;
        $calculatedCost = null;
        $order = [];

        if ($request->filled('location_id') && $request->filled('game_id')) {
            $gameVersions = GameVersion::where('game_id', (int) $request->game_id)
                ->where('is_active', true)
                ->orderBy('sort_order')
                ->orderBy('id')
                ->get();

            if ($request->filled('game_version_id')) {
                $selectedGameVersion = $gameVersions->firstWhere('id', (int) $request->game_version_id);
            }
            if (! $selectedGameVersion) {
                $selectedGameVersion = $gameVersions->first();
            }

            $tariffs = Tariff::with(['location', 'game'])
                ->where('location_id', $request->location_id)
                ->where('game_id', $request->game_id)
                ->where('is_available', true)
                ->orderBy('position')
                ->get();

            if ($request->filled('tariff_id')) {
                $selectedTariff = $tariffs->firstWhere('id', (int) $request->tariff_id);
            }

            if (! $selectedTariff) {
                $selectedTariff = $tariffs->first();
            }

            $selectedLocation = $locations->firstWhere('id', (int) $request->location_id);

            if ($selectedTariff) {
                $slots = (int) $request->input('slots');
                $slots = max(1, $slots);
                $slots = max((int) $selectedTariff->min_slots, min((int) $selectedTariff->max_slots, $slots));

                $billingType = (string) $selectedTariff->billing_type;

                $cpuMin = (int) $selectedTariff->cpu_min;
                $cpuMax = (int) $selectedTariff->cpu_max;
                $cpuStep = (int) $selectedTariff->cpu_step;
                $ramMin = (int) $selectedTariff->ram_min;
                $ramMax = (int) $selectedTariff->ram_max;
                $ramStep = (int) $selectedTariff->ram_step;
                $diskMin = (int) $selectedTariff->disk_min;
                $diskMax = (int) $selectedTariff->disk_max;
                $diskStep = (int) $selectedTariff->disk_step;

                $cpuMin = max(0, $cpuMin);
                $cpuMax = max($cpuMin, $cpuMax);
                $cpuStep = max(1, $cpuStep);
                $ramMin = max(0, $ramMin);
                $ramMax = max($ramMin, $ramMax);
                $ramStep = max(1, $ramStep);
                $diskMin = max(0, $diskMin);
                $diskMax = max($diskMin, $diskMax);
                $diskStep = max(1, $diskStep);

                $cpu = (float) $selectedTariff->cpu_cores;
                $ram = (int) $selectedTariff->ram_gb;
                $disk = (int) $selectedTariff->disk_gb;

                if ($billingType === 'resources') {
                    $cpu = (float) $request->input('cpu_cores');
                    $ram = (int) $request->input('ram_gb');
                    $disk = (int) $request->input('disk_gb');

                    $cpu = max($cpuMin, min($cpuMax, (int) $cpu));
                    $ram = max($ramMin, min($ramMax, (int) $ram));
                    $disk = max($diskMin, min($diskMax, (int) $disk));
                }

                $antiddosEnabled = (bool) $request->boolean('antiddos_enabled');
                if (! (bool) $selectedTariff->allow_antiddos) {
                    $antiddosEnabled = false;
                }

                $ipChoice = (string) $request->input('ip_choice');

                $gameCode = strtolower((string) ($selectedTariff->game->code ?: $selectedTariff->game->slug));
                $serverFps = (int) $request->input('server_fps');
                $serverTickrate = (int) $request->input('server_tickrate');
                if ($serverFps !== 500 && $serverFps !== 1000) {
                    $serverFps = 0;
                }
                if ($serverTickrate !== 66 && $serverTickrate !== 100) {
                    $serverTickrate = 0;
                }

                $order = [
                    'billing_type' => $billingType,
                    'slots' => $slots,
                    'cpu_cores' => $cpu,
                    'ram_gb' => $ram,
                    'disk_gb' => $disk,
                    'antiddos_enabled' => $antiddosEnabled,
                    'ip_choice' => $ipChoice,
                    'game_code' => $gameCode,
                    'server_fps' => $serverFps,
                    'server_tickrate' => $serverTickrate,
                    'cpu_min' => $cpuMin,
                    'cpu_max' => $cpuMax,
                    'cpu_step' => $cpuStep,
                    'ram_min' => $ramMin,
                    'ram_max' => $ramMax,
                    'ram_step' => $ramStep,
                    'disk_min' => $diskMin,
                    'disk_max' => $diskMax,
                    'disk_step' => $diskStep,
                ];

                if ($request->filled('period')) {
                    $periodDays = (int) $request->input('period');
                    $factor = $periodDays / 30;
                    $base = 0.0;
                    if ($billingType !== 'slots') {
                        $base = (float) (((float) $selectedTariff->base_price_monthly) * $factor);
                    }

                    $extra = 0.0;
                    if ($billingType === 'slots') {
                        $extra = (float) ($slots * (float) $selectedTariff->price_per_slot * $factor);
                    } else {
                        $extra = (float) ((
                            ((float) $cpu * (float) $selectedTariff->price_per_cpu_core)
                            + ((float) $ram * (float) $selectedTariff->price_per_ram_gb)
                            + ((float) $disk * (float) $selectedTariff->price_per_disk_gb)
                        ) * $factor);
                    }

                    $addons = 0.0;
                    if ($antiddosEnabled) {
                        $addons += (float) ((float) $selectedTariff->antiddos_price * $factor);
                    }

                    $cost = $base + $extra + $addons;

                    $promoCodeInput = trim((string) $request->input('promo_code'));
                    $promoCodeInput = $promoCodeInput !== '' ? mb_strtoupper($promoCodeInput) : null;

                    $promoResult = app(PromotionService::class)->pickPromotion(
                        PromotionService::APPLY_RENT,
                        Auth::user(),
                        $promoCodeInput,
                        (int) $selectedTariff->id ?: null,
                        (int) $selectedTariff->game_id ?: null,
                        (int) $selectedTariff->location_id ?: null,
                        (float) $cost
                    );

                    $promoResult = array_merge(['promotion' => null, 'error' => null], (array) $promoResult);
                    $promo = $promoResult['promotion'];
                    if ($promo) {
                        $applied = app(PromotionService::class)->applyToAmount($promo, PromotionService::APPLY_RENT, (float) $cost);
                        $applied = array_merge(['final_amount' => $cost], (array) $applied);
                        $cost = (float) $applied['final_amount'];
                    }

                    $calculatedCost = $cost;
                }
            }
        }

        return view('rent-server', [
            'locations' => $locations,
            'games' => $games,
            'tariffs' => $tariffs,
            'selectedTariff' => $selectedTariff,
            'selectedLocation' => $selectedLocation,
            'gameVersions' => $gameVersions,
            'selectedGameVersion' => $selectedGameVersion,
            'request' => $request,
            'calculatedCost' => $calculatedCost,
            'order' => $order,
        ]);
    }

    public function submitRent(Request $request): \Illuminate\Http\RedirectResponse
    {
        $validated = $request->validate([
            'location_id' => ['required', 'exists:locations,id'],
            'game_id' => ['required', 'exists:games,id'],
            'tariff_id' => ['nullable', 'exists:tariffs,id'],
            'game_version_id' => ['nullable', 'exists:game_versions,id'],
            'slots' => ['nullable', 'integer', 'min:1'],
            'cpu_cores' => ['nullable', 'numeric', 'min:0'],
            'ram_gb' => ['nullable', 'integer', 'min:0'],
            'disk_gb' => ['nullable', 'integer', 'min:0'],
            'antiddos_enabled' => ['sometimes', 'boolean'],
            'ip_choice' => ['nullable', 'string', 'max:255'],
            'server_fps' => ['nullable', 'integer', 'in:0,500,1000'],
            'server_tickrate' => ['nullable', 'integer', 'in:0,66,100'],
            'period' => ['required', 'integer', 'in:15,30,60,180'],
            'promo_code' => ['nullable', 'string', 'max:255'],
        ]);

        $validated = array_merge([
            'tariff_id' => null,
            'game_version_id' => null,
            'slots' => null,
            'cpu_cores' => null,
            'ram_gb' => null,
            'disk_gb' => null,
            'antiddos_enabled' => false,
            'ip_choice' => null,
            'server_fps' => null,
            'server_tickrate' => null,
            'promo_code' => null,
        ], (array) $validated);

        $returnInput = $request->only(self::RENT_RETURN_KEYS);

        $user = Auth::user();

        $tariffQuery = Tariff::where('is_available', true)
            ->where('location_id', $validated['location_id'])
            ->where('game_id', $validated['game_id']);

        if (! empty($validated['tariff_id'])) {
            $tariffQuery->where('id', $validated['tariff_id']);
        }

        $tariff = $tariffQuery->first();

        if (! $tariff) {
            return $this->redirectRentWithError($returnInput, 'Тариф не найден');
        }

        $tariff->loadMissing('game');
        $game = $tariff->game;
        if (! $game) {
            return $this->redirectRentWithError($returnInput, 'Игра не найдена');
        }

        $gameVersion = null;
        if (! empty($validated['game_version_id'])) {
            $gameVersion = GameVersion::where('id', (int) $validated['game_version_id'])
                ->where('game_id', (int) $validated['game_id'])
                ->where('is_active', true)
                ->first();

            if (! $gameVersion) {
                return $this->redirectRentWithError($returnInput, 'Версия игры не найдена');
            }
        } else {
            $gameVersion = GameVersion::where('game_id', (int) $validated['game_id'])
                ->where('is_active', true)
                ->orderBy('sort_order')
                ->orderBy('id')
                ->first();
        }

        if ($gameVersion) {
            $src = (string) $gameVersion->source_type;
            if ($src === 'steam') {
                $appId = (int) $gameVersion->steam_app_id;
                if ($appId <= 0) {
                    return $this->redirectRentWithError($returnInput, 'Выбранная версия Steam некорректна (нет steam_app_id)');
                }
            } else {
                $aurl = trim((string) $gameVersion->url);
                if ($aurl === '') {
                    return $this->redirectRentWithError($returnInput, 'Выбранная версия Archive некорректна (нет archive_url)');
                }
            }
        }

        $periodDays = (int) $validated['period'];
        $factor = $periodDays / 30;

        $slots = (int) $validated['slots'];
        $slots = max(1, $slots);
        $slots = max((int) $tariff->min_slots, min((int) $tariff->max_slots, $slots));

        $billingType = (string) $tariff->billing_type;

        $cpu = (float) $tariff->cpu_cores;
        $ram = (int) $tariff->ram_gb;
        $disk = (int) $tariff->disk_gb;

        if ($billingType === 'resources') {
            $cpuMin = (int) $tariff->cpu_min;
            $cpuMax = (int) $tariff->cpu_max;
            $ramMin = (int) $tariff->ram_min;
            $ramMax = (int) $tariff->ram_max;
            $diskMin = (int) $tariff->disk_min;
            $diskMax = (int) $tariff->disk_max;

            $cpuMin = max(0, $cpuMin);
            $cpuMax = max($cpuMin, $cpuMax);
            $ramMin = max(0, $ramMin);
            $ramMax = max($ramMin, $ramMax);
            $diskMin = max(0, $diskMin);
            $diskMax = max($diskMin, $diskMax);

            $cpu = (float) $validated['cpu_cores'];
            $ram = (int) $validated['ram_gb'];
            $disk = (int) $validated['disk_gb'];

            $cpu = max($cpuMin, min($cpuMax, (int) $cpu));
            $ram = max($ramMin, min($ramMax, (int) $ram));
            $disk = max($diskMin, min($diskMax, (int) $disk));
        }

        $base = 0.0;
        if ($billingType !== 'slots') {
            $base = (float) (((float) $tariff->base_price_monthly) * $factor);
        }
        $extra = 0.0;
        if ($billingType === 'slots') {
            $extra = (float) ($slots * (float) $tariff->price_per_slot * $factor);
        } else {
            $extra = (float) ((
                ((float) $cpu * (float) $tariff->price_per_cpu_core)
                + ((float) $ram * (float) $tariff->price_per_ram_gb)
                + ((float) $disk * (float) $tariff->price_per_disk_gb)
            ) * $factor);
        }

        $antiddosEnabled = (bool) $request->boolean('antiddos_enabled');
        if (! (bool) $tariff->allow_antiddos) {
            $antiddosEnabled = false;
        }
        $addons = 0.0;
        if ($antiddosEnabled) {
            $addons += (float) ((float) $tariff->antiddos_price * $factor);
        }

        $cost = $base + $extra + $addons;

        $promoCodeInput = trim((string) $validated['promo_code']);
        $promoCodeInput = $promoCodeInput !== '' ? mb_strtoupper($promoCodeInput) : null;

        $promoResult = app(PromotionService::class)->pickPromotion(
            PromotionService::APPLY_RENT,
            $user,
            $promoCodeInput,
            (int) $tariff->id ?: null,
            (int) $tariff->game_id ?: null,
            (int) $tariff->location_id ?: null,
            (float) $cost
        );

        $promoResult = array_merge(['promotion' => null, 'error' => null], (array) $promoResult);

        if ($promoResult['error'] && $promoCodeInput !== null) {
            return $this->redirectRentWithError($returnInput, (string) $promoResult['error']);
        }

        $promotion = $promoResult['promotion'];
        if ($promotion) {
            $applied = app(PromotionService::class)->applyToAmount($promotion, PromotionService::APPLY_RENT, (float) $cost);
            $applied = array_merge(['final_amount' => $cost], (array) $applied);
            $cost = (float) $applied['final_amount'];
        }

        $cost = round((float) $cost, 2);

        if ($cost <= 0) {
            return $this->redirectRentWithError($returnInput, 'Не удалось рассчитать стоимость');
        }

        if ($user->balance < $cost) {
            return $this->redirectRentWithError($returnInput, 'Недостаточно средств на балансе');
        }

        $location = Location::find($validated['location_id']);
        if (! $location || ! $location->ssh_host) {
            return $this->redirectRentWithError($returnInput, 'Для локации не настроен хост демона');
        }

        $gameCode = strtolower((string) ($tariff->game->code ?: $tariff->game->slug));
        if ($gameCode === '' || ! in_array($gameCode, self::SUPPORTED_BY_DAEMON, true)) {
            return $this->redirectRentWithError($returnInput, 'Эта игра пока не поддерживается на локации (нужна поддержка в location-daemon).');
        }

        $publicIp = (string) ($location->ip_address ?: $location->ssh_host);
        $ipChoice = (string) $validated['ip_choice'];
        if ($ipChoice !== '') {
            $allowed = [];
            if (is_array($location->ip_pool)) {
                $allowed = array_map('strval', $location->ip_pool);
            }
            if ($location->ip_address) {
                $allowed[] = (string) $location->ip_address;
            }
            $allowed = array_values(array_unique(array_filter($allowed)));
            if (in_array($ipChoice, $allowed, true)) {
                $publicIp = $ipChoice;
            }
        }

        $serverFps = $validated['server_fps'];
        $serverFps = $serverFps !== null ? (int) $serverFps : null;
        $serverTickrate = $validated['server_tickrate'];
        $serverTickrate = $serverTickrate !== null ? (int) $serverTickrate : null;

        $server = DB::transaction(function () use ($user, $tariff, $game, $gameVersion, $validated, $cost, $periodDays, $location, $publicIp, $slots, $cpu, $ram, $disk, $antiddosEnabled, $serverFps, $serverTickrate, $promotion) {
            Location::whereKey($location->id)->lockForUpdate()->first();

            $portMin = (int) $game->minport;
            $portMax = (int) $game->maxport;
            $portMin = max(1, min(65535, $portMin));
            $portMax = max(1, min(65535, $portMax));
            if ($portMin > $portMax) {
                [$portMin, $portMax] = [$portMax, $portMin];
            }

            $gameCode = strtolower((string) ($tariff->game->code ?: $tariff->game->slug));
            $isMta = in_array($gameCode, ['mta', 'mtasa', 'mta-sa', 'mta_sa'], true);
            $isUnturned = in_array($gameCode, ['unturned', 'unturn', 'ut', 'untrm4', 'untrm5'], true);
            $isRust = in_array($gameCode, ['rust'], true);

            $serversInLoc = Server::where('location_id', $location->id)
                ->with(['game:id,code,slug'])
                ->get(['id', 'port', 'game_id']);

            $used = [];
            foreach ($serversInLoc as $s) {
                $p0 = (int) $s->port;
                if ($p0 <= 0) {
                    continue;
                }
                $used[$p0] = true;

                $sGameCode = strtolower((string) ($s->game->code ?: $s->game->slug));

                if (in_array($sGameCode, ['mta', 'mtasa', 'mta-sa', 'mta_sa'], true)) {
                    $used[$p0 + 2] = true;
                    $used[$p0 + 123] = true;
                }
                if (in_array($sGameCode, ['unturned', 'unturn', 'ut', 'untrm4', 'untrm5'], true)) {
                    $used[$p0 + 1] = true;
                }
                if (in_array($sGameCode, ['rust'], true)) {
                    $used[$p0 + 1] = true;
                }
            }

            $port = null;
            for ($p = $portMin; $p <= $portMax; $p++) {
                if (array_key_exists($p, $used)) {
                    continue;
                }

                if ($isMta) {
                    $http = $p + 2;
                    $ase = $p + 123;
                    if ($http > 65535 || $ase > 65535) {
                        continue;
                    }
                    if (array_key_exists($http, $used) || array_key_exists($ase, $used)) {
                        continue;
                    }
                }

                if ($isUnturned) {
                    $qport = $p + 1;
                    if ($qport > 65535) {
                        continue;
                    }
                    if (array_key_exists($qport, $used)) {
                        continue;
                    }
                }

                if ($isRust) {
                    $qport = $p + 1;
                    if ($qport > 65535) {
                        continue;
                    }
                    if (array_key_exists($qport, $used)) {
                        continue;
                    }
                }

                $port = $p;
                break;
            }
            if ($port === null) {
                throw new \RuntimeException('Нет свободного порта в диапазоне ' . $portMin . '-' . $portMax);
            }

            $user->decrement('balance', $cost);

            if ($promotion) {
                app(PromotionService::class)->lockAndIncrementUsage((int) $promotion->id);
            }

            $server = Server::create([
                'user_id' => $user->id,
                'tariff_id' => $tariff->id,
                'game_id' => $validated['game_id'],
                'game_version_id' => $gameVersion ? $gameVersion->id : null,
                'location_id' => $validated['location_id'],
                'name' => 'Server',
                'ip_address' => $publicIp,
                'port' => $port,
                'slots' => $slots,
                'cpu_cores' => $cpu,
                'cpu_shares' => $tariff->cpu_shares,
                'ram_gb' => $ram,
                'disk_gb' => $disk,
                'antiddos_enabled' => $antiddosEnabled,
                'server_fps' => $serverFps ?: null,
                'server_tickrate' => $serverTickrate ?: null,
                'expires_at' => now()->addDays($periodDays),
                'status' => 'suspended',
                'provisioning_status' => 'pending',
            ]);

            $server->update([
                'name' => 'Server #' . $server->id,
            ]);

            Transaction::create([
                'user_id' => $user->id,
                'type' => 'debit',
                'amount' => $cost,
                'description' => 'Аренда сервера #' . $server->id,
            ]);

            return $server;
        });

        ProvisionServer::dispatch((int) $server->id)->afterResponse();

        return redirect()->route('server.show', $server)->with('success', 'Сервер создан и ставится на установку.');
    }

    public function myServers(): View
    {
        $uid = (int) (auth()->id());

        $sharedIds = DB::table('server_user_permissions')
            ->where('user_id', $uid)
            ->pluck('server_id')
            ->map(fn ($v) => (int) $v)
            ->filter(fn ($v) => $v > 0)
            ->values()
            ->all();

        $servers = Server::query()
            ->with(['game', 'location', 'tariff'])
            ->where(function ($q) use ($uid, $sharedIds) {
                $q->where('user_id', $uid);
                if (! empty($sharedIds)) {
                    $q->orWhereIn('id', $sharedIds);
                }
            })
            ->orderBy('created_at', 'desc')
            ->get();

        return view('my-servers', [
            'servers' => $servers,
        ]);
    }
}