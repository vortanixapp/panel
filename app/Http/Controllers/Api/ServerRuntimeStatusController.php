<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\Location;
use App\Models\Server;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class ServerRuntimeStatusController extends Controller
{
    public function store(Request $request): JsonResponse
    {
        $incomingToken = $request->header('X-Monitoring-Token');
        $expectedToken = config('services.monitoring.token');

        if ($expectedToken && $incomingToken !== $expectedToken) {
            return response()->json(['message' => 'Unauthorized'], 401);
        }

        $validated = $request->validate([
            'location_code' => ['required', 'string', 'max:100'],
            'reported_at' => ['nullable', 'date'],
            'statuses' => ['required', 'array', 'min:1'],
            'statuses.*.server_id' => ['required', 'integer'],
            'statuses.*.runtime_status' => ['required', 'string', 'max:50'],
            'statuses.*.container_id' => ['nullable', 'string', 'max:255'],
            'statuses.*.container_name' => ['nullable', 'string', 'max:255'],
            'statuses.*.port' => ['nullable', 'integer', 'min:1', 'max:65535'],
            'statuses.*.state' => ['nullable', 'string', 'max:50'],
        ]);

        $location = Location::where('code', $validated['location_code'])->first();
        if (! $location) {
            return response()->json(['message' => 'Location not found'], 404);
        }

        $ids = collect($validated['statuses'])
            ->pluck('server_id')
            ->map(fn ($id) => (int) $id)
            ->unique()
            ->values()
            ->all();

        $servers = Server::query()
            ->where('location_id', $location->id)
            ->whereIn('id', $ids)
            ->get()
            ->keyBy('id');

        $updated = 0;
        $skipped = 0;

        foreach ($validated['statuses'] as $status) {
            $serverId = (int) $status['server_id'];
            $server = $servers->get($serverId);
            if (! $server) {
                $skipped++;
                continue;
            }

            $server->update([
                'runtime_status' => $status['runtime_status'],
                'container_id' => $status['container_id'],
                'container_name' => $status['container_name'],
            ]);

            $updated++;
        }

        return response()->json([
            'ok' => true,
            'location_code' => $location->code,
            'updated' => $updated,
            'skipped' => $skipped,
        ]);
    }
}
