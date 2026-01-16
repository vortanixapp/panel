@extends('layouts.app-admin')

@section('page_title', 'Редактирование локации')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div>
                <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Редактирование локации</h1>
                <p class="mt-1 text-sm text-slate-300/80">Измените параметры дата‑центра или региона.</p>
            </div>

            <div class="rounded-2xl border border-white/10 bg-[#242f3d] p-6 shadow-sm shadow-black/20">
                @if ($errors->any())
                    <div class="mb-4 rounded-md border border-rose-500/20 bg-rose-500/10 px-3 py-2 text-xs text-rose-200">
                        <ul class="list-disc space-y-1 pl-4">
                            @foreach ($errors->all() as $error)
                                <li>{{ $error }}</li>
                            @endforeach
                        </ul>
                    </div>
                @endif

                <form method="POST" action="{{ route('admin.locations.update', $location) }}" class="space-y-4">
                    @csrf
                    @method('PUT')

                    <div class="grid gap-4 md:grid-cols-2">
                        <div class="space-y-1">
                            <label for="code" class="text-xs font-medium text-slate-200">Код локации</label>
                            <input
                                id="code"
                                name="code"
                                type="text"
                                value="{{ old('code', $location->code) }}"
                                required
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>

                        <div class="space-y-1">
                            <label for="name" class="text-xs font-medium text-slate-200">Название</label>
                            <input
                                id="name"
                                name="name"
                                type="text"
                                value="{{ old('name', $location->name) }}"
                                required
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                    </div>

                    <div class="grid gap-4 md:grid-cols-3">
                        <div class="space-y-1">
                            <label for="region" class="text-xs font-medium text-slate-200">Регион</label>
                            <input
                                id="region"
                                name="region"
                                type="text"
                                value="{{ old('region', $location->region) }}"
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                        <div class="space-y-1">
                            <label for="city" class="text-xs font-medium text-slate-200">Город</label>
                            <input
                                id="city"
                                name="city"
                                type="text"
                                value="{{ old('city', $location->city) }}"
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                        <div class="space-y-1">
                            <label for="country" class="text-xs font-medium text-slate-200">Страна</label>
                            <input
                                id="country"
                                name="country"
                                type="text"
                                value="{{ old('country', $location->country) }}"
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                    </div>

                    <div class="space-y-1">
                        <label for="description" class="text-xs font-medium text-slate-200">Описание</label>
                        <textarea
                            id="description"
                            name="description"
                            rows="3"
                            class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >{{ old('description', $location->description) }}</textarea>
                    </div>

                    <div class="space-y-1">
                        <label for="ip_address" class="text-xs font-medium text-slate-200">IP адрес (для игроков)</label>
                        <input
                            id="ip_address"
                            name="ip_address"
                            type="text"
                            placeholder="Публичный IP"
                            value="{{ old('ip_address', $location->ip_address) }}"
                            class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                        >
                    </div>

                    <div class="space-y-1">
                        <label for="ip_pool" class="text-xs font-medium text-slate-200">IP пул (JSON или список)</label>
                        <textarea
                            id="ip_pool"
                            name="ip_pool"
                            rows="2"
                            class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            placeholder='["1.2.3.4", "1.2.3.5"] или 1.2.3.4, 1.2.3.5'
                        >{{ old('ip_pool', $location->ip_pool ? json_encode($location->ip_pool) : '') }}</textarea>
                    </div>

                    <div class="space-y-1">
                        <label class="text-xs font-medium text-slate-200">SSH-доступ</label>
                        <div class="grid gap-3 md:grid-cols-4">
                            <div class="space-y-1">
                                <input
                                    name="ssh_host"
                                    type="text"
                                    placeholder="IP / hostname"
                                    value="{{ old('ssh_host', $location->ssh_host) }}"
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-[11px] text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                            </div>
                            <div class="space-y-1">
                                <input
                                    name="ssh_user"
                                    type="text"
                                    placeholder="user"
                                    value="{{ old('ssh_user', $location->ssh_user) }}"
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-[11px] text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                            </div>
                            <div class="space-y-1">
                                <input
                                    name="ssh_password"
                                    type="password"
                                    placeholder="новый пароль (необязательно)"
                                    value=""
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-[11px] text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                                <p class="mt-1 text-[11px] text-slate-300/70">Текущий пароль скрыт. Оставьте поле пустым, чтобы не менять его.</p>
                            </div>
                            <div class="space-y-1">
                                <input
                                    name="ssh_port"
                                    type="number"
                                    min="1"
                                    max="65535"
                                    placeholder="port"
                                    value="{{ old('ssh_port', $location->ssh_port ?? 22) }}"
                                    class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-[11px] text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                                >
                            </div>
                        </div>
                        <p class="mt-1 text-[11px] text-slate-300/70">Не храните здесь продакшн-пароли, используйте отдельные учётные данные для оркестрации.</p>
                    </div>

                    <div class="grid gap-4 md:grid-cols-2 items-center">
                        <div class="space-y-1">
                            <label for="sort_order" class="text-xs font-medium text-slate-200">Порядок сортировки</label>
                            <input
                                id="sort_order"
                                name="sort_order"
                                type="number"
                                min="0"
                                value="{{ old('sort_order', $location->sort_order) }}"
                                class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500"
                            >
                        </div>
                        <div class="flex items-center gap-3 pt-4 md:pt-6">
                            <label class="inline-flex items-center gap-2 text-[11px] text-slate-300/80">
                                <input
                                    type="checkbox"
                                    name="is_active"
                                    value="1"
                                    class="h-3 w-3 rounded border-white/10 bg-black/10 text-slate-100 focus:ring-slate-100"
                                    {{ old('is_active', $location->is_active) ? 'checked' : '' }}
                                >
                                <span>Локация активна</span>
                            </label>
                        </div>
                    </div>

                    <div class="flex items-center justify-end gap-3 pt-4">
                        <a href="{{ route('admin.locations.index') }}" class="text-xs text-slate-300/80 hover:text-white">Отмена</a>
                        <button
                            type="submit"
                            class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-slate-800"
                        >
                            Сохранить изменения
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </section>
@endsection
