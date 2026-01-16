<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\Location;
use App\Models\LocationMetric;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class MetricController extends Controller
{
    public function store(Request $request): JsonResponse
    {
        $incomingToken = $request->header('X-Monitoring-Token');
        $expectedToken = config('services.monitoring.token');

        if ($expectedToken && $incomingToken !== $expectedToken) {
            return response()->json(['message' => 'Unauthorized'], 401);
        }

        $validated = $request->validate([
            'location_code' => ['required_without:location_id', 'string', 'max:100'],
            'location_id'   => ['nullable', 'integer', 'exists:locations,id'],
            'metric_type'   => ['required', 'string', 'max:100'],
            'value'         => ['required'],
            'measured_at'   => ['nullable', 'date'],
        ]);

        $location = null;

        if (! empty($validated['location_id'])) {
            $location = Location::findOrFail($validated['location_id']);
        } else {
            $location = Location::where('code', $validated['location_code'])->firstOrFail();
        }

        LocationMetric::create([
            'location_id' => $location->id,
            'metric_type' => $validated['metric_type'],
            'value'       => in_array($validated['metric_type'], ['os_info', 'cpu_model', 'ram_total', 'disk_total', 'disk_used', 'disk_available', 'uptime'])
                ? 0
                : $validated['value'],
            'text_value'  => in_array($validated['metric_type'], ['os_info', 'cpu_model', 'ram_total', 'disk_total', 'disk_used', 'disk_available', 'uptime'])
                ? $validated['value']
                : null,
            'measured_at' => $validated['measured_at'],
        ]);

        return response()->json(['message' => 'ok'], 201);
    }
}
