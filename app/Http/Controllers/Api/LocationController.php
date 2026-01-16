<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\Location;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class LocationController extends Controller
{
    public function sshData(Request $request, string $code): JsonResponse
    {
        $incomingToken = $request->header('X-Monitoring-Token');
        $expectedToken = config('services.monitoring.token');

        if ($expectedToken && $incomingToken !== $expectedToken) {
            return response()->json(['message' => 'Unauthorized'], 401);
        }

        $location = Location::where('code', $code)->first();

        if (!$location || !$location->is_active || !$location->ssh_host || !$location->ssh_user) {
            return response()->json(['message' => 'Location not configured for SSH'], 404);
        }

        return response()->json([
            'ssh_host' => $location->ssh_host,
            'ssh_user' => $location->ssh_user,
            'ssh_password' => $location->ssh_password,
            'ssh_port' => $location->ssh_port,
        ]);
    }
}
