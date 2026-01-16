<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Server;
use App\Models\Transaction;
use App\Models\User;
use Carbon\Carbon;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;
use Illuminate\View\View;

class UserController extends Controller
{
    public function index(Request $request): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $q = trim((string) $request->query('q', ''));
        $role = (string) $request->query('role', '');

        $query = User::query();
        if ($q !== '') {
            $query->where(function ($sub) use ($q) {
                $sub
                    ->where('name', 'like', '%' . $q . '%')
                    ->orWhere('last_name', 'like', '%' . $q . '%')
                    ->orWhere('email', 'like', '%' . $q . '%')
                    ->orWhere('public_id', 'like', '%' . $q . '%');
            });
        }

        if ($role === 'admin') {
            $query->where('is_admin', true);
        } elseif ($role === 'user') {
            $query->where('is_admin', false);
        }

        $users = $query->orderByDesc('created_at')->paginate(15)->withQueryString();

        $counts = [
            'total' => (int) User::query()->count(),
            'admins' => (int) User::query()->where('is_admin', true)->count(),
            'users' => (int) User::query()->where('is_admin', false)->count(),
        ];

        return view('admin.users', [
            'users' => $users,
            'q' => $q,
            'role' => $role,
            'counts' => $counts,
        ]);
    }

    public function create(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return view('admin.users.create');
    }

    public function store(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'email' => ['required', 'email', 'max:255', 'unique:users,email'],
            'password' => ['required', 'string', 'min:8', 'confirmed'],
            'is_admin' => ['sometimes', 'boolean'],
        ]);

        User::create([
            'name' => $validated['name'],
            'email' => $validated['email'],
            'password' => bcrypt($validated['password']),
            'is_admin' => $validated['is_admin'],
        ]);

        return redirect()->route('admin.users');
    }

    public function show(User $user): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('admin.users');
        }

        $servers = Server::query()
            ->where('user_id', (int) $user->id)
            ->with(['game', 'tariff', 'location'])
            ->orderByDesc('created_at')
            ->limit(20)
            ->get();

        $transactions = Transaction::query()
            ->where('user_id', (int) $user->id)
            ->orderByDesc('created_at')
            ->limit(20)
            ->get();

        $transactionsCreditSum = (float) Transaction::query()
            ->where('user_id', (int) $user->id)
            ->where('type', 'credit')
            ->sum('amount');

        $transactionsDebitSum = (float) Transaction::query()
            ->where('user_id', (int) $user->id)
            ->where('type', 'debit')
            ->sum('amount');

        $sessions = collect();
        if (config('session.driver') === 'database') {
            $table = config('session.table', 'sessions');
            $rawSessions = DB::table($table)
                ->orderByDesc('last_activity')
                ->limit(200)
                ->get();

            $extractUserId = static function ($payload): ?int {
                if (! is_string($payload) || $payload === '') {
                    return null;
                }

                $data = null;

                $decoded = base64_decode($payload, true);
                if (is_string($decoded) && $decoded !== '') {
                    $data = @unserialize($decoded);
                }

                if (! is_array($data)) {
                    $data = @unserialize($payload);
                }

                if (! is_array($data)) {
                    $data = json_decode($payload, true);
                }

                if (! is_array($data)) {
                    return null;
                }

                if (array_key_exists('user_id', $data)) {
                    $v = $data['user_id'];
                    if (is_int($v)) {
                        return $v;
                    }
                    if (is_string($v) && ctype_digit($v)) {
                        return (int) $v;
                    }
                }

                foreach ($data as $key => $value) {
                    if (! is_string($key)) {
                        continue;
                    }

                    if (! str_starts_with($key, 'login_')) {
                        continue;
                    }

                    if (is_int($value)) {
                        return $value;
                    }
                    if (is_string($value) && ctype_digit($value)) {
                        return (int) $value;
                    }
                }

                return null;
            };

            $sessions = $rawSessions
                ->filter(function ($row) use ($user, $extractUserId) {
                    if (! empty($row->user_id) && (int) $row->user_id === (int) $user->id) {
                        return true;
                    }

                    $payloadUserId = $extractUserId($row->payload);
                    return $payloadUserId !== null && (int) $payloadUserId === (int) $user->id;
                })
                ->take(20)
                ->map(function ($row) {
                    return [
                        'id' => $row->id,
                        'ip_address' => $row->ip_address,
                        'user_agent' => $row->user_agent,
                        'last_activity' => Carbon::createFromTimestamp($row->last_activity),
                    ];
                })
                ->values();
        }

        return view('admin.users.show', [
            'user' => $user,
            'servers' => $servers,
            'transactions' => $transactions,
            'transactionsCreditSum' => $transactionsCreditSum,
            'transactionsDebitSum' => $transactionsDebitSum,
            'sessions' => $sessions,
        ]);
    }

    public function edit(User $user): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return view('admin.users.edit', [
            'user' => $user,
        ]);
    }

    public function update(Request $request, User $user): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'last_name' => ['nullable', 'string', 'max:255'],
            'public_id' => ['nullable', 'string', 'max:64', 'unique:users,public_id,' . $user->id],
            'email' => ['required', 'email', 'max:255', 'unique:users,email,' . $user->id],
            'password' => ['nullable', 'string', 'min:8', 'confirmed'],
            'balance' => ['nullable', 'numeric', 'min:0', 'max:999999999'],
            'phone' => ['nullable', 'string', 'max:32'],
            'telegram_id' => ['nullable', 'string', 'max:64'],
            'discord_id' => ['nullable', 'string', 'max:64'],
            'vk_id' => ['nullable', 'string', 'max:64'],
            'is_admin' => ['sometimes', 'boolean'],
        ]);

        $data = [
            'name' => $validated['name'],
            'last_name' => $validated['last_name'],
            'public_id' => $validated['public_id'],
            'email' => $validated['email'],
            'is_admin' => $validated['is_admin'],
        ];

        if (array_key_exists('balance', $validated) && $validated['balance'] !== null) {
            $data['balance'] = number_format((float) $validated['balance'], 2, '.', '');
        }
        $data['phone'] = $validated['phone'];
        $data['telegram_id'] = $validated['telegram_id'];
        $data['discord_id'] = $validated['discord_id'];
        $data['vk_id'] = $validated['vk_id'];

        if (! empty($validated['password'])) {
            $data['password'] = bcrypt($validated['password']);
        }

        $user->update($data);

        return redirect()->route('admin.users');
    }

    public function destroy(User $user): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }
        if (Auth::id() === $user->id) {
            return redirect()->route('admin.users')->with('error', 'Нельзя удалить собственный аккаунт.');
        }

        $user->delete();

        return redirect()->route('admin.users');
    }
}
