<div class="space-y-4 text-xs md:text-sm" x-data>
    {{-- Stop trying to control. --}}

    <div class="grid gap-4">
        <div class="space-y-1">
            <label class="flex items-center justify-between gap-3 text-[11px] font-medium uppercase tracking-wide text-slate-300/70">
                <span>Игра</span>
                <span class="text-[10px] font-normal text-slate-300/70">Подбираем оптимальную конфигурацию</span>
            </label>
            <div class="relative">
                <select
                    wire:model.live="game"
                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                >
                    <option value="cs2">CS2 / CS:GO</option>
                    <option value="minecraft">Minecraft</option>
                    <option value="rust">Rust</option>
                </select>
            </div>
        </div>

        <div class="grid grid-cols-2 gap-3">
            <div class="space-y-1">
                <label class="flex items-center justify-between gap-3 text-[11px] font-medium uppercase tracking-wide text-slate-300/70">
                    <span>Слоты</span>
                    <span class="text-[10px] font-normal text-slate-300/70">4–256</span>
                </label>
                <input
                    type="number"
                    min="4"
                    max="256"
                    step="2"
                    wire:model.live="slots"
                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                >
            </div>

            <div class="space-y-1">
                <label class="flex items-center justify-between gap-3 text-[11px] font-medium uppercase tracking-wide text-slate-300/70">
                    <span>Память, ГБ</span>
                    <span class="text-[10px] font-normal text-slate-300/70">1–32</span>
                </label>
                <input
                    type="number"
                    min="1"
                    max="32"
                    step="1"
                    wire:model.live="memory"
                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                >
            </div>
        </div>

        <div class="space-y-1">
            <label class="flex items-center justify-between gap-3 text-[11px] font-medium uppercase tracking-wide text-slate-300/70">
                <span>Регион</span>
            </label>
            <div class="grid w-full grid-cols-3 gap-1 rounded-md bg-black/10 p-1 text-[11px] ring-1 ring-white/10">
                <button
                    type="button"
                    wire:click="$set('region','ru')"
                    class="rounded-[5px] px-3 py-1 font-medium transition"
                    :class="@js($region === 'ru') ? 'bg-[#242f3d] text-slate-100 ring-1 ring-white/10' : 'text-slate-300/70 hover:bg-black/10 hover:text-slate-100'"
                >
                    РФ
                </button>
                <button
                    type="button"
                    wire:click="$set('region','eu')"
                    class="rounded-[5px] px-3 py-1 font-medium transition"
                    :class="@js($region === 'eu') ? 'bg-[#242f3d] text-slate-100 ring-1 ring-white/10' : 'text-slate-300/70 hover:bg-black/10 hover:text-slate-100'"
                >
                    ЕС
                </button>
                <button
                    type="button"
                    wire:click="$set('region','asia')"
                    class="rounded-[5px] px-3 py-1 font-medium transition"
                    :class="@js($region === 'asia') ? 'bg-[#242f3d] text-slate-100 ring-1 ring-white/10' : 'text-slate-300/70 hover:bg-black/10 hover:text-slate-100'"
                >
                    Азия
                </button>
            </div>
        </div>
    </div>

    @php($price = $this->price)

    <div class="mt-4 rounded-xl border border-white/10 bg-black/10 p-4 space-y-3">
        <div class="flex items-center justify-between gap-3">
            <div>
                <p class="text-[11px] uppercase tracking-wide text-slate-300/70">Примерная стоимость</p>
                <p class="text-[11px] text-slate-300/70">Можно менять конфигурацию без пересоздания сервера</p>
            </div>
            <div class="text-right">
                <p class="text-2xl font-semibold text-slate-100">
                    {{ number_format($price, 0, ',', ' ') }} ₽/мес
                </p>
                <p class="text-[11px] text-emerald-300">Оплата по модели pay‑as‑you‑go</p>
            </div>
        </div>

        <div class="grid gap-2 text-[11px] text-slate-300/80 md:grid-cols-3">
            <p>
                <span class="text-slate-300/70">Игра:</span>
                <span class="ml-1 font-medium text-slate-100">
                    @switch($game)
                        @case('minecraft') Minecraft @break
                        @case('rust') Rust @break
                        @default CS2 / CS:GO
                    @endswitch
                </span>
            </p>
            <p>
                <span class="text-slate-300/70">Слоты:</span>
                <span class="ml-1 font-medium text-slate-100">{{ $slots }}</span>
            </p>
            <p>
                <span class="text-slate-300/70">Память:</span>
                <span class="ml-1 font-medium text-slate-100">{{ $memory }} ГБ</span>
            </p>
        </div>
    </div>
</div>
