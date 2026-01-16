<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Game;
use App\Models\Location;
use App\Models\Tariff;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\View\View;

class TariffController extends Controller
{
    public function index(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $tariffs = Tariff::with(['location', 'game'])->orderBy('position')->paginate(10);

        return view('admin.tariffs', [
            'tariffs' => $tariffs,
        ]);
    }

    public function create(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $locations = Location::all();
        $games = Game::all();

        return view('admin.tariffs.create', [
            'locations' => $locations,
            'games' => $games,
        ]);
    }

    public function store(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'location_id' => ['required', 'exists:locations,id'],
            'game_id' => ['required', 'exists:games,id'],
            'billing_type' => ['required', 'string', 'in:resources,slots'],
            'price_per_slot' => ['nullable', 'numeric', 'min:0', 'required_if:billing_type,slots'],
            'price_per_cpu_core' => ['nullable', 'numeric', 'min:0'],
            'price_per_ram_gb' => ['nullable', 'numeric', 'min:0'],
            'price_per_disk_gb' => ['nullable', 'numeric', 'min:0'],
            'base_price_monthly' => ['nullable', 'numeric', 'min:0', 'required_if:billing_type,resources'],
            'cpu_min' => ['nullable', 'integer', 'min:0'],
            'cpu_max' => ['nullable', 'integer', 'min:0'],
            'cpu_step' => ['nullable', 'integer', 'min:1'],
            'ram_min' => ['nullable', 'integer', 'min:0'],
            'ram_max' => ['nullable', 'integer', 'min:0'],
            'ram_step' => ['nullable', 'integer', 'min:1'],
            'disk_min' => ['nullable', 'integer', 'min:0'],
            'disk_max' => ['nullable', 'integer', 'min:0'],
            'disk_step' => ['nullable', 'integer', 'min:1'],
            'allow_antiddos' => ['sometimes', 'boolean'],
            'antiddos_price' => ['nullable', 'numeric', 'min:0'],
            'min_slots' => ['nullable', 'integer', 'min:1', 'required_if:billing_type,slots'],
            'max_slots' => ['nullable', 'integer', 'min:1', 'required_if:billing_type,slots'],
            'cpu_cores' => ['required', 'integer', 'min:0'],
            'cpu_shares' => ['nullable', 'integer', 'min:2', 'max:262144', 'required_if:cpu_cores,0'],
            'ram_gb' => ['required', 'integer', 'min:1'],
            'disk_gb' => ['required', 'integer', 'min:1'],
            'rental_periods' => ['required', 'array', 'min:1'],
            'renewal_periods' => ['required', 'array', 'min:1'],
            'discounts' => ['nullable', 'string'],
            'position' => ['required', 'integer', 'min:0'],
            'is_available' => ['sometimes', 'boolean'],
        ], [
            'rental_periods.required' => 'Выберите хотя бы один период для аренды.',
            'rental_periods.min' => 'Выберите хотя бы один период для аренды.',
            'renewal_periods.required' => 'Выберите хотя бы один период для продления.',
            'renewal_periods.min' => 'Выберите хотя бы один период для продления.',
        ]);

        $billingType = (string) $validated['billing_type'];

        $discounts = null;
        if (array_key_exists('discounts', $validated) && $validated['discounts'] !== null && $validated['discounts'] !== '') {
            $decodedDiscounts = json_decode($validated['discounts'], true);
            if (! is_array($decodedDiscounts)) {
                return back()->withErrors(['discounts' => 'Некорректный JSON.'])->withInput();
            }
            $discounts = $decodedDiscounts;
        }

        Tariff::create([
            'name' => $validated['name'],
            'location_id' => $validated['location_id'],
            'game_id' => $validated['game_id'],
            'billing_type' => $billingType,
            'price_per_slot' => $billingType === 'slots' ? $validated['price_per_slot'] : null,
            'price_per_cpu_core' => $billingType === 'resources' ? (float) $validated['price_per_cpu_core'] : (float) null,
            'price_per_ram_gb' => $billingType === 'resources' ? (float) $validated['price_per_ram_gb'] : (float) null,
            'price_per_disk_gb' => $billingType === 'resources' ? (float) $validated['price_per_disk_gb'] : (float) null,
            'base_price_monthly' => $billingType === 'resources' ? (float) $validated['base_price_monthly'] : null,
            'cpu_min' => $validated['cpu_min'],
            'cpu_max' => $validated['cpu_max'],
            'cpu_step' => $validated['cpu_step'],
            'ram_min' => $validated['ram_min'],
            'ram_max' => $validated['ram_max'],
            'ram_step' => $validated['ram_step'],
            'disk_min' => $validated['disk_min'],
            'disk_max' => $validated['disk_max'],
            'disk_step' => $validated['disk_step'],
            'allow_antiddos' => (bool) $validated['allow_antiddos'],
            'antiddos_price' => (float) $validated['antiddos_price'],
            'min_slots' => $billingType === 'slots' ? (int) $validated['min_slots'] : null,
            'max_slots' => $billingType === 'slots' ? (int) $validated['max_slots'] : null,
            'cpu_cores' => $validated['cpu_cores'],
            'cpu_shares' => $validated['cpu_shares'],
            'ram_gb' => $validated['ram_gb'],
            'disk_gb' => $validated['disk_gb'],
            'rental_periods' => $validated['rental_periods'],
            'renewal_periods' => $validated['renewal_periods'],
            'discounts' => $discounts,
            'position' => $validated['position'],
            'is_available' => (bool) $validated['is_available'],
        ]);

        return redirect()->route('admin.tariffs.index');
    }

    public function show(Tariff $tariff): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('admin.tariffs.index');
        }

        return view('admin.tariffs.show', [
            'tariff' => $tariff->load(['location', 'game']),
        ]);
    }

    public function edit(Tariff $tariff): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $locations = Location::all();
        $games = Game::all();

        return view('admin.tariffs.edit', [
            'tariff' => $tariff,
            'locations' => $locations,
            'games' => $games,
        ]);
    }

    public function update(Request $request, Tariff $tariff): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'location_id' => ['required', 'exists:locations,id'],
            'game_id' => ['required', 'exists:games,id'],
            'billing_type' => ['required', 'string', 'in:resources,slots'],
            'price_per_slot' => ['nullable', 'numeric', 'min:0', 'required_if:billing_type,slots'],
            'price_per_cpu_core' => ['nullable', 'numeric', 'min:0'],
            'price_per_ram_gb' => ['nullable', 'numeric', 'min:0'],
            'price_per_disk_gb' => ['nullable', 'numeric', 'min:0'],
            'base_price_monthly' => ['nullable', 'numeric', 'min:0', 'required_if:billing_type,resources'],
            'cpu_min' => ['nullable', 'integer', 'min:0'],
            'cpu_max' => ['nullable', 'integer', 'min:0'],
            'cpu_step' => ['nullable', 'integer', 'min:1'],
            'ram_min' => ['nullable', 'integer', 'min:0'],
            'ram_max' => ['nullable', 'integer', 'min:0'],
            'ram_step' => ['nullable', 'integer', 'min:1'],
            'disk_min' => ['nullable', 'integer', 'min:0'],
            'disk_max' => ['nullable', 'integer', 'min:0'],
            'disk_step' => ['nullable', 'integer', 'min:1'],
            'allow_antiddos' => ['sometimes', 'boolean'],
            'antiddos_price' => ['nullable', 'numeric', 'min:0'],
            'min_slots' => ['nullable', 'integer', 'min:1', 'required_if:billing_type,slots'],
            'max_slots' => ['nullable', 'integer', 'min:1', 'required_if:billing_type,slots'],
            'cpu_cores' => ['required', 'integer', 'min:0'],
            'cpu_shares' => ['nullable', 'integer', 'min:2', 'max:262144', 'required_if:cpu_cores,0'],
            'ram_gb' => ['required', 'integer', 'min:1'],
            'disk_gb' => ['required', 'integer', 'min:1'],
            'rental_periods' => ['required', 'array', 'min:1'],
            'renewal_periods' => ['required', 'array', 'min:1'],
            'discounts' => ['nullable', 'string'],
            'position' => ['required', 'integer', 'min:0'],
            'is_available' => ['sometimes', 'boolean'],
        ], [
            'rental_periods.required' => 'Выберите хотя бы один период для аренды.',
            'rental_periods.min' => 'Выберите хотя бы один период для аренды.',
            'renewal_periods.required' => 'Выберите хотя бы один период для продления.',
            'renewal_periods.min' => 'Выберите хотя бы один период для продления.',
        ]);

        $billingType = (string) $validated['billing_type'];

        $discounts = null;
        if (array_key_exists('discounts', $validated) && $validated['discounts'] !== null && $validated['discounts'] !== '') {
            $decodedDiscounts = json_decode($validated['discounts'], true);
            if (! is_array($decodedDiscounts)) {
                return back()->withErrors(['discounts' => 'Некорректный JSON.'])->withInput();
            }
            $discounts = $decodedDiscounts;
        }

        $tariff->update([
            'name' => $validated['name'],
            'location_id' => $validated['location_id'],
            'game_id' => $validated['game_id'],
            'billing_type' => $billingType,
            'price_per_slot' => $billingType === 'slots' ? $validated['price_per_slot'] : null,
            'price_per_cpu_core' => $billingType === 'resources' ? (float) $validated['price_per_cpu_core'] : (float) null,
            'price_per_ram_gb' => $billingType === 'resources' ? (float) $validated['price_per_ram_gb'] : (float) null,
            'price_per_disk_gb' => $billingType === 'resources' ? (float) $validated['price_per_disk_gb'] : (float) null,
            'base_price_monthly' => $billingType === 'resources' ? (float) $validated['base_price_monthly'] : null,
            'cpu_min' => $validated['cpu_min'],
            'cpu_max' => $validated['cpu_max'],
            'cpu_step' => $validated['cpu_step'],
            'ram_min' => $validated['ram_min'],
            'ram_max' => $validated['ram_max'],
            'ram_step' => $validated['ram_step'],
            'disk_min' => $validated['disk_min'],
            'disk_max' => $validated['disk_max'],
            'disk_step' => $validated['disk_step'],
            'allow_antiddos' => (bool) $validated['allow_antiddos'],
            'antiddos_price' => (float) $validated['antiddos_price'],
            'min_slots' => $billingType === 'slots' ? (int) $validated['min_slots'] : null,
            'max_slots' => $billingType === 'slots' ? (int) $validated['max_slots'] : null,
            'cpu_cores' => $validated['cpu_cores'],
            'cpu_shares' => $validated['cpu_shares'],
            'ram_gb' => $validated['ram_gb'],
            'disk_gb' => $validated['disk_gb'],
            'rental_periods' => $validated['rental_periods'],
            'renewal_periods' => $validated['renewal_periods'],
            'discounts' => $discounts,
            'position' => $validated['position'],
            'is_available' => (bool) $validated['is_available'],
        ]);

        return redirect()->route('admin.tariffs.index');
    }

    public function destroy(Tariff $tariff): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $tariff->delete();

        return redirect()->route('admin.tariffs.index');
    }

    public function duplicate(Tariff $tariff): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $copy = Tariff::create([
            'name' => $tariff->name . ' (копия)',
            'location_id' => $tariff->location_id,
            'game_id' => $tariff->game_id,
            'billing_type' => $tariff->billing_type,
            'price_per_slot' => $tariff->price_per_slot,
            'price_per_cpu_core' => $tariff->price_per_cpu_core,
            'price_per_ram_gb' => $tariff->price_per_ram_gb,
            'price_per_disk_gb' => $tariff->price_per_disk_gb,
            'base_price_monthly' => $tariff->base_price_monthly,
            'cpu_min' => $tariff->cpu_min,
            'cpu_max' => $tariff->cpu_max,
            'cpu_step' => $tariff->cpu_step,
            'ram_min' => $tariff->ram_min,
            'ram_max' => $tariff->ram_max,
            'ram_step' => $tariff->ram_step,
            'disk_min' => $tariff->disk_min,
            'disk_max' => $tariff->disk_max,
            'disk_step' => $tariff->disk_step,
            'allow_antiddos' => $tariff->allow_antiddos,
            'antiddos_price' => $tariff->antiddos_price,
            'min_slots' => $tariff->min_slots,
            'max_slots' => $tariff->max_slots,
            'cpu_cores' => $tariff->cpu_cores,
            'cpu_shares' => $tariff->cpu_shares,
            'ram_gb' => $tariff->ram_gb,
            'disk_gb' => $tariff->disk_gb,
            'rental_periods' => $tariff->rental_periods,
            'renewal_periods' => $tariff->renewal_periods,
            'discounts' => $tariff->discounts,
            'position' => $tariff->position,
            'is_available' => $tariff->is_available,
        ]);

        return redirect()->route('admin.tariffs.edit', $copy);
    }
}
