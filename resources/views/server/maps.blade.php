@extends($layout ?? 'layouts.app-user')

@section('title', $server->name)

@section('breadcrumb')
    <a href="{{ route('dashboard') }}" class="text-xs text-slate-200 hover:text-white">Кабинет</a>
    <span class="h-1 w-1 rounded-full bg-white/25"></span>
    <a href="{{ route('my-servers') }}" class="text-xs text-slate-200 hover:text-white">Мои серверы</a>
    <span class="h-1 w-1 rounded-full bg-white/25"></span>
    <span class="text-xs text-slate-100">{{ $server->name }}</span>
@endsection

@section('content')
    <section class="py-6">
        <div class="w-full px-4 sm:px-6 lg:px-8">
            @php $tab = 'maps'; @endphp

            <div class="rounded-2xl bg-[#242f3d] text-slate-100 shadow-sm overflow-hidden">
                <div class="bg-black/10">
                    @include('server.partials.tabs')
                </div>
            </div>

            @if (session('success'))
                <div class="mt-6 rounded-md border border-emerald-500/20 bg-emerald-500/10 px-4 py-3 text-xs text-emerald-200">
                    {{ session('success') }}
                </div>
            @endif
            @if (session('error'))
                <div class="mt-6 rounded-md border border-rose-500/20 bg-rose-500/10 px-4 py-3 text-xs text-rose-200">
                    {{ session('error') }}
                </div>
            @endif

            <div class="mt-6 rounded-2xl border border-white/10 bg-[#242f3d] text-slate-100 shadow-sm overflow-hidden">
                <div class="border-b border-white/10 bg-black/10 px-4 py-3 text-[11px] uppercase tracking-wide text-slate-300/70">
                    Карты
                </div>

                <div class="p-4">
                    @if(empty($items))
                        <div class="text-sm text-slate-300/80">Карты пока не добавлены администратором.</div>
                    @else
                        @php
                            $categories = [];
                            foreach ($items as $row) {
                                $c = trim((string) (($row['map']->category ?? '') ?: ''));
                                if ($c === '') { $c = 'Другое'; }
                                $categories[$c] = true;
                            }
                            $categoriesList = array_keys($categories);
                            sort($categoriesList, SORT_NATURAL | SORT_FLAG_CASE);
                        @endphp

                        <div class="mb-4 flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
                            <div class="flex flex-col gap-2 md:flex-row md:items-end">
                                <div class="space-y-1">
                                    <label for="mapSearch" class="text-[11px] text-slate-300/70">Поиск</label>
                                    <input id="mapSearch" type="text" class="block w-full md:w-64 rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500" placeholder="de_dust2 / awp / ...">
                                </div>
                                <div class="space-y-1">
                                    <label for="mapCategory" class="text-[11px] text-slate-300/70">Категория</label>
                                    <select id="mapCategory" class="block w-full md:w-56 rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                                        <option value="">Все</option>
                                        @foreach($categoriesList as $c)
                                            <option value="{{ $c }}">{{ $c }}</option>
                                        @endforeach
                                    </select>
                                </div>
                            </div>
                            <div class="text-[11px] text-slate-300/70" id="mapsFilterHint"></div>
                        </div>

                        <div class="overflow-x-auto">
                            <table class="min-w-full divide-y divide-white/10 text-sm">
                                <thead class="text-[11px] uppercase tracking-wide text-slate-300/70">
                                    <tr>
                                        <th class="px-3 py-2 text-left">Карта</th>
                                        <th class="px-3 py-2 text-left">Версия</th>
                                        <th class="px-3 py-2 text-left">Статус</th>
                                        <th class="px-3 py-2 text-right">Действия</th>
                                    </tr>
                                </thead>
                                <tbody class="divide-y divide-white/10" id="mapsTableBody">
                                    @php $prevCat = null; @endphp
                                    @foreach($items as $row)
                                        @php
                                            $m = $row['map'];
                                            $sm = $row['serverMap'];
                                            $isInstalled = (bool) ($sm->installed ?? false);
                                            $cat = trim((string) ($m->category ?? ''));
                                            if ($cat === '') { $cat = 'Другое'; }
                                            $nameKey = strtolower(trim((string) ($m->name ?? '')));
                                            $slugKey = strtolower(trim((string) ($m->slug ?? '')));
                                        @endphp

                                        @if($prevCat !== $cat)
                                            @php $prevCat = $cat; @endphp
                                            <tr data-category-header="{{ $cat }}">
                                                <td colspan="4" class="px-3 py-2 bg-black/10 text-[11px] uppercase tracking-wide text-slate-300/70">
                                                    {{ $cat }}
                                                </td>
                                            </tr>
                                        @endif
                                        <tr data-map-row data-category="{{ $cat }}" data-name="{{ $nameKey }}" data-slug="{{ $slugKey }}">
                                            <td class="px-3 py-2 align-top">
                                                <div class="font-semibold text-slate-100">{{ $m->name }}</div>
                                                <div class="text-[11px] text-slate-300/70">{{ $m->slug }}</div>
                                                <div class="text-[11px] text-slate-300/70">Категория: {{ $cat }}</div>
                                                <div class="mt-1 text-[11px] text-slate-300/70">Путь: cstrike</div>
                                            </td>
                                            <td class="px-3 py-2 align-top text-slate-200">
                                                {{ $m->version ?: '—' }}
                                            </td>
                                            <td class="px-3 py-2 align-top">
                                                @if($isInstalled)
                                                    <div class="text-xs text-emerald-200">Установлен</div>
                                                @else
                                                    <div class="text-xs text-slate-300/80">Не установлен</div>
                                                @endif
                                                @if(!empty($sm->last_error))
                                                    <div class="mt-1 text-[11px] text-rose-200">{{ $sm->last_error }}</div>
                                                @endif
                                            </td>
                                            <td class="px-3 py-2 align-top">
                                                <div class="flex justify-end flex-wrap gap-2">
                                                    @if($isInstalled)
                                                        <form method="POST" action="{{ route('server.maps.install', ['server' => $server, 'map' => $m]) }}">
                                                            @csrf
                                                            <button type="submit" class="inline-flex items-center rounded-md border border-sky-500/30 bg-sky-500/10 px-3 py-2 text-[11px] font-semibold text-sky-200 hover:bg-sky-500/15">Обновить</button>
                                                        </form>

                                                        <form method="POST" action="{{ route('server.maps.uninstall', ['server' => $server, 'map' => $m]) }}" onsubmit="return confirm('Удалить карту {{ $m->name }} с сервера?');">
                                                            @csrf
                                                            <button type="submit" class="inline-flex items-center rounded-md border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-[11px] font-semibold text-rose-200 hover:bg-rose-500/15">Удалить</button>
                                                        </form>
                                                    @else
                                                        <form method="POST" action="{{ route('server.maps.install', ['server' => $server, 'map' => $m]) }}">
                                                            @csrf
                                                            <button type="submit" class="inline-flex items-center rounded-md bg-slate-900 px-3 py-2 text-[11px] font-semibold text-white hover:bg-slate-800">Установить</button>
                                                        </form>
                                                    @endif
                                                </div>
                                            </td>
                                        </tr>
                                    @endforeach
                                </tbody>
                            </table>
                        </div>
                    @endif
                </div>
            </div>
        </div>
    </section>

    <script>
        document.addEventListener('DOMContentLoaded', function () {
            const search = document.getElementById('mapSearch');
            const cat = document.getElementById('mapCategory');
            const tbody = document.getElementById('mapsTableBody');
            const hint = document.getElementById('mapsFilterHint');
            if (!search || !cat || !tbody) return;

            function applyFilters() {
                const q = String(search.value || '').toLowerCase().trim();
                const c = String(cat.value || '').trim();

                const rows = Array.from(tbody.querySelectorAll('tr[data-map-row]'));
                const headers = Array.from(tbody.querySelectorAll('tr[data-category-header]'));
                const visibleByCat = {};
                let visibleCount = 0;

                rows.forEach((r) => {
                    const rowCat = String(r.getAttribute('data-category') || '');
                    const name = String(r.getAttribute('data-name') || '');
                    const slug = String(r.getAttribute('data-slug') || '');
                    const matchCat = c === '' || rowCat === c;
                    const matchQ = q === '' || name.includes(q) || slug.includes(q);
                    const show = matchCat && matchQ;
                    r.style.display = show ? '' : 'none';
                    if (show) {
                        visibleByCat[rowCat] = true;
                        visibleCount += 1;
                    }
                });

                headers.forEach((h) => {
                    const headerCat = String(h.getAttribute('data-category-header') || '');
                    h.style.display = visibleByCat[headerCat] ? '' : 'none';
                });

                if (hint) {
                    if (q === '' && c === '') {
                        hint.textContent = '';
                    } else {
                        hint.textContent = `Найдено: ${visibleCount}`;
                    }
                }
            }

            search.addEventListener('input', applyFilters);
            cat.addEventListener('change', applyFilters);
            applyFilters();
        });
    </script>
@endsection
