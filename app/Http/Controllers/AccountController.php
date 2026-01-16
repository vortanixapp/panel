<?php

namespace App\Http\Controllers;

use Carbon\Carbon;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Hash;
use Illuminate\View\View;

class AccountController extends Controller
{
    public function show(): View
    {
        $user = Auth::user();

        $sessions = collect();
        $currentSessionId = null;

        if (config('session.driver') === 'database' && $user) {
            $table = config('session.table', 'sessions');

            $rawSessions = DB::table($table)
                ->where('user_id', $user->id)
                ->orderByDesc('last_activity')
                ->limit(20)
                ->get();

            $sessions = $rawSessions->map(function ($row) {
                return [
                    'id' => $row->id,
                    'ip_address' => $row->ip_address,
                    'user_agent' => $row->user_agent,
                    'last_activity' => Carbon::createFromTimestamp($row->last_activity),
                ];
            });

            $currentSessionId = session()->getId();
        }

        return view('account.show', [
            'user' => $user,
            'sessions' => $sessions,
            'currentSessionId' => $currentSessionId,
        ]);
    }

    public function updateEmail(Request $request): RedirectResponse
    {
        $user = $request->user();

        $validated = $request->validate([
            'email' => ['required', 'email', 'max:255', 'unique:users,email,' . $user->id],
            'current_password' => ['required', 'current_password'],
        ]);

        $user->email = $validated['email'];
        $user->save();

        return back()->with('status', 'email-updated');
    }

    public function updatePassword(Request $request): RedirectResponse
    {
        $user = $request->user();

        $validated = $request->validate([
            'current_password' => ['required', 'current_password'],
            'password' => ['required', 'string', 'min:8', 'confirmed'],
        ]);

        $user->password = Hash::make($validated['password']);
        $user->save();

        return back()->with('status', 'password-updated');
    }

    public function updateProfile(Request $request): RedirectResponse
    {
        $user = $request->user();

        $validated = $request->validate([
            'name' => ['nullable', 'string', 'max:255'],
            'last_name' => ['nullable', 'string', 'max:255'],
            'phone' => ['nullable', 'string', 'max:32', 'regex:/^\+?[0-9\s\-\(\)]{6,32}$/'],
            'telegram_id' => ['nullable', 'string', 'max:64', 'regex:/^@?[A-Za-z0-9_]{4,64}$/'],
            'discord_id' => ['nullable', 'string', 'max:64', 'regex:/^[A-Za-z0-9_.#]{2,64}$/'],
            'vk_id' => ['nullable', 'string', 'max:64', 'regex:/^[A-Za-z0-9_.]{3,64}$/'],
        ]);

        $user->fill($validated);
        $user->save();

        return back()->with('status', 'profile-updated');
    }

    public function destroySession(Request $request): RedirectResponse
    {
        $user = $request->user();

        if (! $user || config('session.driver') !== 'database') {
            return back();
        }

        $validated = $request->validate([
            'session_id' => ['nullable', 'string'],
            'all_others' => ['nullable', 'boolean'],
        ]);

        $table = config('session.table', 'sessions');
        $currentSessionId = $request->session()->getId();

        if (! empty($validated['all_others'])) {
            DB::table($table)
                ->where('user_id', $user->id)
                ->where('id', '!=', $currentSessionId)
                ->delete();
        } elseif (! empty($validated['session_id'])) {
            DB::table($table)
                ->where('user_id', $user->id)
                ->where('id', $validated['session_id'])
                ->where('id', '!=', $currentSessionId)
                ->delete();
        }

        return back()->with('status', 'sessions-updated');
    }
}
