<?php

namespace App\Http\Controllers;

use App\Models\SupportMessage;
use App\Models\SupportTicket;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;
use Illuminate\View\View;

class SupportController extends Controller
{
    public function index(): View
    {
        $user = Auth::user();

        $tickets = SupportTicket::query()
            ->where('user_id', (int) $user->id)
            ->orderByDesc('last_message_at')
            ->orderByDesc('created_at')
            ->paginate(15);

        return view('support.index', [
            'tickets' => $tickets,
        ]);
    }

    public function create(): View
    {
        return view('support.create');
    }

    public function store(Request $request): RedirectResponse
    {
        $user = Auth::user();

        $validated = $request->validate([
            'subject' => ['required', 'string', 'max:255'],
            'message' => ['required', 'string', 'max:10000'],
        ]);

        $ticket = DB::transaction(function () use ($user, $validated) {
            $ticket = SupportTicket::create([
                'user_id' => (int) $user->id,
                'assigned_admin_id' => null,
                'subject' => (string) $validated['subject'],
                'status' => 'open',
                'priority' => 'normal',
                'last_message_at' => now(),
                'closed_at' => null,
            ]);

            SupportMessage::create([
                'ticket_id' => (int) $ticket->id,
                'user_id' => (int) $user->id,
                'is_admin' => false,
                'message' => (string) $validated['message'],
            ]);

            return $ticket;
        });

        return redirect()->route('support.show', $ticket);
    }

    public function show(SupportTicket $ticket): View
    {
        $user = Auth::user();
        if ((int) $ticket->user_id !== (int) $user->id) {
            abort(403, 'Доступ запрещен');
        }

        $ticket->load(['messages.user']);

        return view('support.show', [
            'ticket' => $ticket,
        ]);
    }

    public function messages(Request $request, SupportTicket $ticket): JsonResponse
    {
        $user = Auth::user();
        if ((int) $ticket->user_id !== (int) $user->id) {
            abort(403, 'Доступ запрещен');
        }

        $afterId = (int) $request->query('after_id', 0);

        $messages = SupportMessage::query()
            ->where('ticket_id', (int) $ticket->id)
            ->when($afterId > 0, fn ($q) => $q->where('id', '>', $afterId))
            ->with('user')
            ->orderBy('id')
            ->limit(200)
            ->get()
            ->map(function (SupportMessage $m) {
                return [
                    'id' => (int) $m->id,
                    'is_admin' => (bool) $m->is_admin,
                    'message' => (string) $m->message,
                    'created_at' => (string) $m->created_at?->format('d.m.Y H:i'),
                    'user_email' => $m->user?->email,
                ];
            })
            ->values();

        return response()->json([
            'messages' => $messages,
        ]);
    }

    public function reply(Request $request, SupportTicket $ticket): RedirectResponse
    {
        $user = Auth::user();
        if ((int) $ticket->user_id !== (int) $user->id) {
            abort(403, 'Доступ запрещен');
        }

        if ((string) $ticket->status !== 'open') {
            return redirect()->route('support.show', $ticket)->with('error', 'Тикет закрыт');
        }

        $validated = $request->validate([
            'message' => ['required', 'string', 'max:10000'],
        ]);

        DB::transaction(function () use ($ticket, $user, $validated) {
            SupportMessage::create([
                'ticket_id' => (int) $ticket->id,
                'user_id' => (int) $user->id,
                'is_admin' => false,
                'message' => (string) $validated['message'],
            ]);

            $ticket->update([
                'last_message_at' => now(),
            ]);
        });

        return redirect()->route('support.show', $ticket);
    }

    public function close(Request $request, SupportTicket $ticket): RedirectResponse
    {
        $user = Auth::user();
        if ((int) $ticket->user_id !== (int) $user->id) {
            abort(403, 'Доступ запрещен');
        }

        if ((string) $ticket->status === 'closed') {
            return redirect()->route('support.show', $ticket);
        }

        $ticket->update([
            'status' => 'closed',
            'closed_at' => now(),
        ]);

        return redirect()->route('support.show', $ticket)->with('success', 'Тикет закрыт');
    }
}
