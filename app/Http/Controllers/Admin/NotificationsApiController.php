<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Services\LicenseCloudClient;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;

class NotificationsApiController extends Controller
{
    public function list(Request $request): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['message' => 'Forbidden'], 403);
        }

        $limit = (int) $request->query('limit', 10);
        if ($limit < 1) {
            $limit = 1;
        }
        if ($limit > 100) {
            $limit = 100;
        }

        $unread = (string) $request->query('unread', '') === '1';

        try {
            return response()->json(app(LicenseCloudClient::class)->listNotifications($limit, $unread));
        } catch (\Throwable $e) {
            return response()->json(['message' => 'Failed to load notifications'], 502);
        }
    }

    public function read(Request $request, int $id): JsonResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return response()->json(['message' => 'Forbidden'], 403);
        }

        try {
            return response()->json(app(LicenseCloudClient::class)->markNotificationRead($id));
        } catch (\Throwable $e) {
            return response()->json(['message' => 'Failed to mark as read'], 502);
        }
    }
}
