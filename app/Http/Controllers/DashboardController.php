<?php

namespace App\Http\Controllers;

use App\Models\News;
use App\Models\Server;
use App\Models\SupportTicket;
use App\Models\Transaction;
use Illuminate\Http\RedirectResponse;
use Illuminate\Support\Facades\Auth;
use Illuminate\View\View;

class DashboardController extends Controller
{
    public function index(): View|RedirectResponse
    {
        if (Auth::user() && Auth::user()->is_admin) {
            return redirect()->route('admin.dashboard');
        }

        $user = Auth::user();

        $totalServers = Server::where('user_id', $user->id)->count();
        $activeServers = Server::where('user_id', $user->id)->where('status', 'active')->count();

        $today = now()->startOfDay();
        $expiringSoonCount = Server::where('user_id', $user->id)
            ->whereNotNull('expires_at')
            ->get()
            ->filter(function ($s) use ($today) {
                if (! $s->expires_at) {
                    return false;
                }

                $expiresAt = $s->expires_at->copy()->startOfDay();
                $daysLeft = $today->diffInDays($expiresAt, false);

                return $daysLeft >= 0 && $daysLeft < 7;
            })
            ->count();

        $nextExpiryServer = Server::with(['game', 'location'])
            ->where('user_id', $user->id)
            ->whereNotNull('expires_at')
            ->orderBy('expires_at')
            ->first();

        $recentServers = Server::with(['game', 'location'])
            ->where('user_id', $user->id)
            ->orderBy('created_at', 'desc')
            ->limit(6)
            ->get();

        $recentTransactions = Transaction::where('user_id', $user->id)
            ->orderBy('created_at', 'desc')
            ->limit(8)
            ->get();

        $news = News::query()
            ->with('images')
            ->where('active', true)
            ->whereNotNull('published_at')
            ->where('published_at', '<=', now())
            ->orderByDesc('published_at')
            ->limit(3)
            ->get();

        $openSupportTicketsCount = (int) SupportTicket::query()
            ->where('user_id', (int) $user->id)
            ->where('status', 'open')
            ->count();

        return view('dashboard', [
            'balance' => $user->balance,
            'bonuses' => $user->bonuses,
            'totalServers' => $totalServers,
            'activeServers' => $activeServers,
            'expiringSoonCount' => $expiringSoonCount,
            'nextExpiryServer' => $nextExpiryServer,
            'recentServers' => $recentServers,
            'recentTransactions' => $recentTransactions,
            'news' => $news,
            'openSupportTicketsCount' => $openSupportTicketsCount,
        ]);
    }
}
