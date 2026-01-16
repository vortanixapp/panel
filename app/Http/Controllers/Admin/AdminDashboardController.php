<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Location;
use App\Models\LocationMetric;
use App\Models\Payment;
use App\Models\Server;
use App\Models\SupportTicket;
use App\Models\User;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\View\View;

class AdminDashboardController extends Controller
{
    public function index(Request $request): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $totalUsers = User::count();
        $adminUsers = User::where('is_admin', true)->count();
        $regularUsers = User::where('is_admin', false)->count();
        $staffUsers = User::where('is_admin', true)->orderByDesc('created_at')->limit(6)->get();
        $totalLocations = Location::count();
        $activeLocations = Location::where('is_active', true)->count();

        $serversTotal = Server::count();
        $serversActive = Server::query()->where('status', 'active')->count();
        $serversExpired = Server::query()->where('status', 'expired')->count();
        $serversSuspended = Server::query()->where('status', 'suspended')->count();

        $provisioningInstalling = Server::query()->whereIn('provisioning_status', ['pending', 'installing', 'reinstalling'])->count();
        $provisioningFailed = Server::query()->where('provisioning_status', 'failed')->count();

        $failedServers = Server::query()
            ->where('provisioning_status', 'failed')
            ->orderByDesc('updated_at')
            ->with(['user', 'location', 'game', 'tariff'])
            ->limit(8)
            ->get();

        $serversExpiringSoon = Server::query()
            ->whereNotNull('expires_at')
            ->where('expires_at', '>', now())
            ->where('expires_at', '<=', now()->addDays(3))
            ->orderBy('expires_at')
            ->with(['user', 'location', 'game'])
            ->limit(8)
            ->get();

        $newUsers24h = User::query()->where('created_at', '>=', now()->subDay())->count();
        $newUsers7d = User::query()->where('created_at', '>=', now()->subDays(7))->count();

        $revenue24h = (float) Payment::query()
            ->where('status', 'succeeded')
            ->whereNotNull('credited_at')
            ->where('credited_at', '>=', now()->subDay())
            ->sum('credited_amount');

        if ($revenue24h <= 0) {
            $revenue24h = (float) Payment::query()
                ->where('status', 'succeeded')
                ->whereNotNull('credited_at')
                ->where('credited_at', '>=', now()->subDay())
                ->sum('amount');
        }

        $revenue7d = (float) Payment::query()
            ->where('status', 'succeeded')
            ->whereNotNull('credited_at')
            ->where('credited_at', '>=', now()->subDays(7))
            ->sum('credited_amount');
        if ($revenue7d <= 0) {
            $revenue7d = (float) Payment::query()
                ->where('status', 'succeeded')
                ->whereNotNull('credited_at')
                ->where('credited_at', '>=', now()->subDays(7))
                ->sum('amount');
        }

        $revenue30d = (float) Payment::query()
            ->where('status', 'succeeded')
            ->whereNotNull('credited_at')
            ->where('credited_at', '>=', now()->subDays(30))
            ->sum('credited_amount');
        if ($revenue30d <= 0) {
            $revenue30d = (float) Payment::query()
                ->where('status', 'succeeded')
                ->whereNotNull('credited_at')
                ->where('credited_at', '>=', now()->subDays(30))
                ->sum('amount');
        }

        $openSupportTicketsCount = (int) SupportTicket::query()->where('status', 'open')->count();
        $totalSupportTicketsCount = (int) SupportTicket::query()->count();

        $locationsForFilter = Location::query()
            ->orderBy('sort_order')
            ->orderBy('id')
            ->get(['id', 'code', 'name', 'is_active']);

        $locationId = (int) $request->query('location_id', 0);
        $selectedLocation = null;
        if ($locationId > 0) {
            $selectedLocation = $locationsForFilter->firstWhere('id', $locationId);
            if (! $selectedLocation) {
                $locationId = 0;
            }
        }

        $metricTypes = ['cpu_usage', 'ram_usage'];
        if ($locationId > 0) {
            $metrics = LocationMetric::query()
                ->where('location_id', $locationId)
                ->whereIn('metric_type', $metricTypes)
                ->orderByDesc('measured_at')
                ->limit(120)
                ->get()
                ->groupBy('metric_type');
        } else {
            $metrics = Location::where('is_active', true)
                ->with(['metrics' => function ($query) use ($metricTypes) {
                    $query->whereIn('metric_type', $metricTypes)
                        ->orderByDesc('measured_at')
                        ->limit(10);
                }])
                ->get()
                ->pluck('metrics')
                ->flatten()
                ->groupBy('metric_type');
        }

        $cpuSeries = $metrics->get('cpu_usage', collect())
            ->sortBy('measured_at')
            ->map(fn ($m) => ['x' => (string) $m->measured_at?->format('H:i'), 'y' => (float) $m->value])
            ->values();

        $ramSeries = $metrics->get('ram_usage', collect())
            ->sortBy('measured_at')
            ->map(fn ($m) => ['x' => (string) $m->measured_at?->format('H:i'), 'y' => (float) $m->value])
            ->values();

        return view('admin.dashboard', [
            'totalUsers' => $totalUsers,
            'adminUsers' => $adminUsers,
            'regularUsers' => $regularUsers,
            'staffUsers' => $staffUsers,
            'totalLocations' => $totalLocations,
            'activeLocations' => $activeLocations,
            'metrics' => $metrics,
            'cpuSeries' => $cpuSeries,
            'ramSeries' => $ramSeries,
            'locationsForFilter' => $locationsForFilter,
            'selectedLocationId' => $locationId,
            'selectedLocation' => $selectedLocation,
            'openSupportTicketsCount' => $openSupportTicketsCount,
            'totalSupportTicketsCount' => $totalSupportTicketsCount,

            'serversTotal' => $serversTotal,
            'serversActive' => $serversActive,
            'serversExpired' => $serversExpired,
            'serversSuspended' => $serversSuspended,
            'provisioningInstalling' => $provisioningInstalling,
            'provisioningFailed' => $provisioningFailed,
            'failedServers' => $failedServers,
            'serversExpiringSoon' => $serversExpiringSoon,

            'newUsers24h' => $newUsers24h,
            'newUsers7d' => $newUsers7d,
            'revenue24h' => $revenue24h,
            'revenue7d' => $revenue7d,
            'revenue30d' => $revenue30d,
        ]);
    }
}
