<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Jobs\ReinstallServer;
use App\Models\Server;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Http;
use Illuminate\View\View;

class ServerController extends Controller
{
    public function index(Request $request): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $q = trim((string) $request->query('q', ''));

        $serversQuery = Server::query()->with(['user', 'game', 'location', 'tariff']);

        if ($q !== '') {
            $serversQuery->where(function ($sub) use ($q) {
                if (ctype_digit($q)) {
                    $sub->orWhere('id', (int) $q);
                }

                $sub->orWhere('name', 'like', '%' . $q . '%')
                    ->orWhere('ip_address', 'like', '%' . $q . '%')
                    ->orWhereHas('user', function ($uq) use ($q) {
                        $uq->where('email', 'like', '%' . $q . '%')
                            ->orWhere('name', 'like', '%' . $q . '%');
                    })
                    ->orWhereHas('game', function ($gq) use ($q) {
                        $gq->where('name', 'like', '%' . $q . '%')
                            ->orWhere('slug', 'like', '%' . $q . '%')
                            ->orWhere('code', 'like', '%' . $q . '%');
                    })
                    ->orWhereHas('location', function ($lq) use ($q) {
                        $lq->where('name', 'like', '%' . $q . '%')
                            ->orWhere('ssh_host', 'like', '%' . $q . '%');
                    });
            });
        }

        $servers = $serversQuery
            ->orderByDesc('created_at')
            ->paginate(20)
            ->withQueryString();

        $counts = [
            'total' => (int) Server::query()->count(),
            'reinstalling' => (int) Server::query()->where('provisioning_status', 'reinstalling')->count(),
            'with_error' => (int) Server::query()->whereNotNull('provisioning_error')->count(),
        ];

        return view('admin.servers.index', [
            'servers' => $servers,
            'q' => $q,
            'counts' => $counts,
        ]);
    }

    public function reinstall(Server $server, Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        if (in_array((string) $server->provisioning_status, ['reinstalling'], true)) {
            return redirect()->route('admin.servers.index')->with('error', 'Переустановка уже выполняется.');
        }

        $server->update([
            'provisioning_status' => 'reinstalling',
            'runtime_status' => 'reinstalling',
            'provisioning_error' => null,
        ]);

        dispatch(new ReinstallServer((int) $server->id));

        return redirect()->route('admin.servers.index')->with('success', 'Переустановка запущена');
    }

    public function destroy(Server $server, Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $server->load(['game', 'location']);

        $gameCode = strtolower((string) $server->game->code);
        if ($gameCode === '') {
            return redirect()->route('admin.servers.index')->with('error', 'Не удалось определить game code для сервера.');
        }

        $result = $this->callDaemon($server, '/servers/delete', [
            'server_id' => (int) $server->id,
            'game' => $gameCode,
        ], 30);

        if (! $result['ok']) {
            return redirect()->route('admin.servers.index')->with('error', 'Не удалось удалить контейнер на локации: ' . (string) $result['error']);
        }

        $server->delete();

        return redirect()->route('admin.servers.index')->with('success', 'Сервер удалён.');
    }

    private function callDaemon(Server $server, string $path, array $payload, int $timeoutSeconds = 30): array
    {
        $server->loadMissing('location');

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
                return ['ok' => false, 'error' => 'Daemon returned status ' . $response->status()];
            }

            $data = $response->json();
            if (! is_array($data) || ! $data['ok']) {
                return ['ok' => false, 'error' => (string) $data['error']];
            }

            return $data;
        } catch (\Throwable $e) {
            return ['ok' => false, 'error' => $e->getMessage()];
        }
    }
}
