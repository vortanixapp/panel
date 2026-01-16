<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Jobs\PrepareMailingRecipients;
use App\Models\Mailing;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;
use Illuminate\View\View;

class MailingController extends Controller
{
    public function index(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $items = Mailing::query()->orderByDesc('id')->paginate(20);

        return view('admin.mailings.index', [
            'items' => $items,
        ]);
    }

    public function create(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return view('admin.mailings.create');
    }

    public function store(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $this->validateInput($request);

        $validated = array_merge([
            'channels' => null,
            'subject' => null,
            'body' => null,
            'is_html' => true,
            'scheduled_at' => null,
            'only_admin' => false,
            'only_non_admin' => false,
            'has_servers' => null,
            'balance_min' => null,
            'user_ids' => '',
            'emails' => '',
        ], (array) $validated);

        $channels = $this->normalizeChannels($validated['channels']);
        $audience = $this->normalizeAudience($validated);

        $scheduledAt = $validated['scheduled_at'];
        $status = $scheduledAt ? 'scheduled' : 'draft';

        $mailing = Mailing::create([
            'title' => (string) $validated['title'],
            'status' => $status,
            'channels' => $channels,
            'audience' => $audience,
            'subject' => $validated['subject'],
            'body' => $validated['body'],
            'is_html' => (bool) $validated['is_html'],
            'scheduled_at' => $scheduledAt,
            'started_at' => null,
            'finished_at' => null,
            'total_recipients' => 0,
            'sent_count' => 0,
            'failed_count' => 0,
            'skipped_count' => 0,
            'last_error' => null,
        ]);

        return redirect()->route('admin.mailings.edit', $mailing)->with('success', 'Кампания создана');
    }

    public function edit(Mailing $mailing): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return view('admin.mailings.edit', [
            'mailing' => $mailing,
        ]);
    }

    public function show(Mailing $mailing): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return redirect()->route('admin.mailings.edit', $mailing);
    }

    public function update(Request $request, Mailing $mailing): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $this->validateInput($request);

        $validated = array_merge([
            'channels' => null,
            'subject' => null,
            'body' => null,
            'is_html' => true,
            'scheduled_at' => null,
            'only_admin' => false,
            'only_non_admin' => false,
            'has_servers' => null,
            'balance_min' => null,
            'user_ids' => '',
            'emails' => '',
        ], (array) $validated);

        $channels = $this->normalizeChannels($validated['channels']);
        $audience = $this->normalizeAudience($validated);

        $mailing->update([
            'title' => (string) $validated['title'],
            'channels' => $channels,
            'audience' => $audience,
            'subject' => $validated['subject'],
            'body' => $validated['body'],
            'is_html' => (bool) $validated['is_html'],
            'scheduled_at' => $validated['scheduled_at'],
        ]);

        return redirect()->route('admin.mailings.edit', $mailing)->with('success', 'Кампания сохранена');
    }

    public function destroy(Mailing $mailing): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $mailing->delete();

        return redirect()->route('admin.mailings.index')->with('success', 'Кампания удалена');
    }

    public function start(Mailing $mailing): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        DB::transaction(function () use ($mailing) {
            $m = Mailing::whereKey((int) $mailing->id)->lockForUpdate()->first();
            if (! $m) {
                return;
            }

            if (in_array((string) $m->status, ['sending', 'sent'], true)) {
                return;
            }

            $m->update([
                'status' => 'queued',
                'scheduled_at' => null,
                'started_at' => null,
                'finished_at' => null,
                'total_recipients' => 0,
                'sent_count' => 0,
                'failed_count' => 0,
                'skipped_count' => 0,
                'last_error' => null,
            ]);
        });

        PrepareMailingRecipients::dispatch((int) $mailing->id);

        return redirect()->route('admin.mailings.edit', $mailing)->with('success', 'Отправка запущена');
    }

    public function schedule(Request $request, Mailing $mailing): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'scheduled_at' => ['required', 'date'],
        ]);

        DB::transaction(function () use ($mailing, $validated) {
            $m = Mailing::whereKey((int) $mailing->id)->lockForUpdate()->first();
            if (! $m) {
                return;
            }

            if (in_array((string) $m->status, ['sending', 'sent'], true)) {
                return;
            }

            $m->update([
                'status' => 'scheduled',
                'scheduled_at' => $validated['scheduled_at'],
            ]);
        });

        return redirect()->route('admin.mailings.edit', $mailing)->with('success', 'Кампания запланирована');
    }

    public function cancel(Mailing $mailing): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        DB::transaction(function () use ($mailing) {
            $m = Mailing::whereKey((int) $mailing->id)->lockForUpdate()->first();
            if (! $m) {
                return;
            }

            if ((string) $m->status === 'sending') {
                $m->update([
                    'status' => 'draft',
                    'scheduled_at' => null,
                ]);
                return;
            }

            if ((string) $m->status === 'scheduled') {
                $m->update([
                    'status' => 'draft',
                    'scheduled_at' => null,
                ]);
            }
        });

        return redirect()->route('admin.mailings.edit', $mailing)->with('success', 'Кампания остановлена');
    }

    private function validateInput(Request $request): array
    {
        return $request->validate([
            'title' => ['required', 'string', 'min:2', 'max:191'],
            'channels' => ['nullable', 'array'],
            'channels.*' => ['string', 'in:email,internal,telegram,sms'],
            'subject' => ['nullable', 'string', 'max:255'],
            'body' => ['nullable', 'string', 'max:100000'],
            'is_html' => ['sometimes', 'boolean'],
            'scheduled_at' => ['nullable', 'date'],

            'only_admin' => ['sometimes', 'boolean'],
            'only_non_admin' => ['sometimes', 'boolean'],
            'has_servers' => ['nullable', 'in:yes,no'],
            'balance_min' => ['nullable', 'numeric', 'min:0'],
            'user_ids' => ['nullable', 'string', 'max:5000'],
            'emails' => ['nullable', 'string', 'max:20000'],
        ]);
    }

    private function normalizeChannels(mixed $channels): array
    {
        if (! is_array($channels)) {
            return ['email'];
        }
        $c = array_values(array_unique(array_map('strval', $channels)));
        $c = array_values(array_filter($c, fn ($v) => in_array($v, ['email', 'internal', 'telegram', 'sms'], true)));
        return count($c) > 0 ? $c : ['email'];
    }

    private function normalizeAudience(array $validated): array
    {
        $audience = [];

        $audience = array_merge([
            'only_admin' => false,
            'only_non_admin' => false,
            'has_servers' => null,
            'balance_min' => null,
            'user_ids' => '',
            'emails' => '',
        ], (array) $validated);

        $audience['only_admin'] = (bool) $audience['only_admin'];
        $audience['only_non_admin'] = (bool) $audience['only_non_admin'];

        $hasServers = $audience['has_servers'];
        if ($hasServers === 'yes') {
            $audience['has_servers'] = true;
        } elseif ($hasServers === 'no') {
            $audience['has_servers'] = false;
        } else {
            $audience['has_servers'] = null;
        }

        if (array_key_exists('balance_min', $validated) && $validated['balance_min'] !== null && $validated['balance_min'] !== '') {
            $audience['balance_min'] = (float) $validated['balance_min'];
        } else {
            $audience['balance_min'] = null;
        }

        $audience['user_ids'] = $this->parseCsvInts((string) $audience['user_ids']);
        $audience['emails'] = $this->parseCsvStrings((string) $audience['emails']);

        return $audience;
    }

    private function parseCsvInts(string $value): array
    {
        $value = trim($value);
        if ($value === '') {
            return [];
        }

        $value = str_replace(["\n", "\r", "\t", ';'], [',', ',', ',', ','], $value);
        $parts = array_filter(array_map('trim', explode(',', $value)), fn ($v) => $v !== '');
        $out = [];
        foreach ($parts as $p) {
            if (! ctype_digit($p)) {
                continue;
            }
            $i = (int) $p;
            if ($i > 0) {
                $out[$i] = $i;
            }
        }
        return array_values($out);
    }

    private function parseCsvStrings(string $value): array
    {
        $value = trim($value);
        if ($value === '') {
            return [];
        }

        $value = str_replace(["\n", "\r", "\t", ';'], [',', ',', ',', ','], $value);
        $parts = array_filter(array_map('trim', explode(',', $value)), fn ($v) => $v !== '');
        $out = [];
        foreach ($parts as $p) {
            $p = trim($p);
            if ($p !== '') {
                $out[$p] = $p;
            }
        }
        return array_values($out);
    }
}
