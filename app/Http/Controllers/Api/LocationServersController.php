<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\Location;
use App\Models\Server;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class LocationServersController extends Controller
{
    public function index(Request $request, string $code): JsonResponse
    {
        $incomingToken = $request->header('X-Monitoring-Token');
        $expectedToken = config('services.monitoring.token');

        if ($expectedToken && $incomingToken !== $expectedToken) {
            return response()->json(['message' => 'Unauthorized'], 401);
        }

        $location = Location::where('code', $code)->first();
        if (! $location) {
            return response()->json(['message' => 'Location not found'], 404);
        }

        $servers = Server::query()
            ->where('location_id', $location->id)
            ->with(['game'])
            ->orderBy('id')
            ->get()
            ->map(function (Server $server) {
                return [
                    'server_id' => $server->id,
                    'game' => strtolower((string) $server->game->code),
                    'port' => (int) $server->port,
                    'status' => (string) $server->status,
                    'provisioning_status' => $server->provisioning_status,
                    'runtime_status' => $server->runtime_status,
                    'container_id' => $server->container_id,
                    'container_name' => $server->container_name,
                ];
            })
            ->values();

        return response()->json([
            'location_code' => $location->code,
            'servers' => $servers,
        ]);
    }
}
