<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\SupportMessage;
use App\Models\SupportTicket;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;
use Illuminate\View\View;

class SupportTicketController extends Controller
{
    public function index(Request $request): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $status = (string) $request->query('status', 'open');
        if (! in_array($status, ['open', 'closed', 'all'], true)) {
            $status = 'open';
        }

        $q = trim((string) $request->query('q', ''));

        $query = SupportTicket::query()->with(['user', 'assignedAdmin']);

        if ($status !== 'all') {
            $query->where('status', $status);
        }

        if ($q !== '') {
            $query->where(function ($sub) use ($q) {
                if (ctype_digit($q)) {
                    $sub->orWhere('id', (int) $q);
                }

                $sub->orWhere('subject', 'like', '%' . $q . '%')
                    ->orWhereHas('user', function ($uq) use ($q) {
                        $uq->where('email', 'like', '%' . $q . '%')
                            ->orWhere('name', 'like', '%' . $q . '%')
                            ->orWhere('last_name', 'like', '%' . $q . '%');
                    });
            });
        }

        $tickets = $query
            ->orderByRaw("(status = 'open') desc")
            ->orderByDesc('last_message_at')
            ->orderByDesc('created_at')
            ->paginate(20)
            ->withQueryString();

        return view('admin.support.index', [
            'tickets' => $tickets,
            'status' => $status,
            'q' => $q,
        ]);
    }

    public function show(SupportTicket $ticket): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $ticket->load(['user', 'assignedAdmin', 'messages.user']);

        return view('admin.support.show', [
            'ticket' => $ticket,
        ]);
    }

    public function messages(Request $request, SupportTicket $ticket): JsonResponse|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
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
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        if ((string) $ticket->status !== 'open') {
            return redirect()->route('admin.support.show', $ticket)->with('error', 'Тикет закрыт');
        }

        $validated = $request->validate([
            'message' => ['required', 'string', 'max:10000'],
        ]);

        DB::transaction(function () use ($ticket, $validated) {
            $admin = Auth::user();

            if (! $ticket->assigned_admin_id) {
                $ticket->assigned_admin_id = (int) $admin->id;
            }

            $ticket->last_message_at = now();
            $ticket->save();

            SupportMessage::create([
                'ticket_id' => (int) $ticket->id,
                'user_id' => (int) $admin->id,
                'is_admin' => true,
                'message' => (string) $validated['message'],
            ]);
        });

        return redirect()->route('admin.support.show', $ticket);
    }

    public function close(Request $request, SupportTicket $ticket): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        if ((string) $ticket->status === 'closed') {
            return redirect()->route('admin.support.show', $ticket);
        }

        $ticket->update([
            'status' => 'closed',
            'closed_at' => now(),
        ]);

        return redirect()->route('admin.support.show', $ticket)->with('success', 'Тикет закрыт');
    }

    public function reopen(Request $request, SupportTicket $ticket): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        if ((string) $ticket->status === 'open') {
            return redirect()->route('admin.support.show', $ticket);
        }

        $ticket->update([
            'status' => 'open',
            'closed_at' => null,
            'last_message_at' => now(),
        ]);

        return redirect()->route('admin.support.show', $ticket)->with('success', 'Тикет открыт');
    }
}
