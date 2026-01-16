<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Promotion;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\View\View;

class PromotionController extends Controller
{
    public function index(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $items = Promotion::query()
            ->orderByDesc('id')
            ->paginate(20);

        return view('admin.promotions.index', [
            'items' => $items,
        ]);
    }

    public function create(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return view('admin.promotions.create');
    }

    public function store(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $this->validateInput($request);

        $code = $validated['code'];
        $code = $code !== null ? trim((string) $code) : null;
        $code = $code !== '' ? mb_strtoupper($code) : null;

        Promotion::create([
            'title' => (string) $validated['title'],
            'code' => $code,
            'is_active' => (bool) $validated['is_active'],
            'starts_at' => $validated['starts_at'],
            'ends_at' => $validated['ends_at'],
            'applies_to' => $validated['applies_to'],
            'discount_type' => $validated['discount_type'],
            'discount_value' => (float) $validated['discount_value'],
            'bonus_percent' => (float) $validated['bonus_percent'],
            'bonus_fixed' => (float) $validated['bonus_fixed'],
            'max_uses' => $validated['max_uses'],
            'min_amount' => $validated['min_amount'],
            'only_new_users' => (bool) $validated['only_new_users'],
            'user_ids' => $this->parseIds($validated['user_ids']),
            'tariff_ids' => $this->parseIds($validated['tariff_ids']),
            'game_ids' => $this->parseIds($validated['game_ids']),
            'location_ids' => $this->parseIds($validated['location_ids']),
            'description' => $validated['description'],
        ]);

        return redirect()->route('admin.promotions.index')->with('success', 'Акция создана');
    }

    public function edit(Promotion $promotion): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return view('admin.promotions.edit', [
            'promotion' => $promotion,
        ]);
    }

    public function show(Promotion $promotion): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return redirect()->route('admin.promotions.edit', $promotion);
    }

    public function update(Request $request, Promotion $promotion): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $this->validateInput($request, (int) $promotion->id);

        $code = $validated['code'];
        $code = $code !== null ? trim((string) $code) : null;
        $code = $code !== '' ? mb_strtoupper($code) : null;

        $promotion->update([
            'title' => (string) $validated['title'],
            'code' => $code,
            'is_active' => (bool) $validated['is_active'],
            'starts_at' => $validated['starts_at'],
            'ends_at' => $validated['ends_at'],
            'applies_to' => $validated['applies_to'],
            'discount_type' => $validated['discount_type'],
            'discount_value' => (float) $validated['discount_value'],
            'bonus_percent' => (float) $validated['bonus_percent'],
            'bonus_fixed' => (float) $validated['bonus_fixed'],
            'max_uses' => $validated['max_uses'],
            'min_amount' => $validated['min_amount'],
            'only_new_users' => (bool) $validated['only_new_users'],
            'user_ids' => $this->parseIds($validated['user_ids']),
            'tariff_ids' => $this->parseIds($validated['tariff_ids']),
            'game_ids' => $this->parseIds($validated['game_ids']),
            'location_ids' => $this->parseIds($validated['location_ids']),
            'description' => $validated['description'],
        ]);

        return redirect()->route('admin.promotions.edit', $promotion)->with('success', 'Акция сохранена');
    }

    public function destroy(Promotion $promotion): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $promotion->delete();

        return redirect()->route('admin.promotions.index')->with('success', 'Акция удалена');
    }

    private function validateInput(Request $request, ?int $ignoreId = null): array
    {
        $codeRule = ['nullable', 'string', 'max:64'];
        $codeRule[] = 'unique:promotions,code' . ($ignoreId ? (',' . $ignoreId) : '');

        return $request->validate([
            'title' => ['required', 'string', 'min:2', 'max:191'],
            'code' => $codeRule,
            'is_active' => ['sometimes', 'boolean'],
            'starts_at' => ['nullable', 'date'],
            'ends_at' => ['nullable', 'date'],
            'applies_to' => ['nullable', 'array'],
            'applies_to.*' => ['string', 'in:rent,renew,topup'],
            'discount_type' => ['nullable', 'string', 'in:percent,fixed'],
            'discount_value' => ['nullable', 'numeric', 'min:0'],
            'bonus_percent' => ['nullable', 'numeric', 'min:0'],
            'bonus_fixed' => ['nullable', 'numeric', 'min:0'],
            'max_uses' => ['nullable', 'integer', 'min:1'],
            'min_amount' => ['nullable', 'numeric', 'min:0'],
            'only_new_users' => ['sometimes', 'boolean'],
            'user_ids' => ['nullable', 'string', 'max:2000'],
            'tariff_ids' => ['nullable', 'string', 'max:2000'],
            'game_ids' => ['nullable', 'string', 'max:2000'],
            'location_ids' => ['nullable', 'string', 'max:2000'],
            'description' => ['nullable', 'string', 'max:5000'],
        ]);
    }

    private function parseIds(mixed $value): ?array
    {
        $str = trim((string) $value);
        if ($str === '') {
            return null;
        }

        $str = str_replace([';', '\n', '\r', '\t'], [',', ',', ',', ','], $str);
        $parts = array_filter(array_map('trim', explode(',', $str)), fn ($p) => $p !== '');

        $out = [];
        foreach ($parts as $p) {
            $p = trim($p);
            if ($p === '') {
                continue;
            }
            if (! ctype_digit($p)) {
                continue;
            }
            $i = (int) $p;
            if ($i <= 0) {
                continue;
            }
            $out[$i] = $i;
        }

        $out = array_values($out);
        return count($out) > 0 ? $out : null;
    }
}
