@extends('layouts.app-admin')

@section('page_title', 'Новая карта')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Новая карта</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Добавьте карту и загрузите zip архив (распаковывается в cstrike/).</p>
                </div>
                <a
                    href="{{ route('admin.maps.index') }}"
                    class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-medium text-slate-200 hover:bg-black/15 hover:text-white"
                >
                    ← Назад к списку
                </a>
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

                <form method="POST" action="{{ route('admin.maps.store') }}" class="space-y-4" enctype="multipart/form-data">
                    @csrf

                    <div class="grid gap-4 md:grid-cols-2">
                        <div class="space-y-1">
                            <label for="name" class="text-xs font-medium text-slate-200">Название</label>
                            <input id="name" name="name" type="text" value="{{ old('name') }}" required class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                        </div>

                        <div class="space-y-1">
                            <label for="category" class="text-xs font-medium text-slate-200">Категория</label>
                            <input id="category" name="category" type="text" value="{{ old('category') }}" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500" placeholder="de / aim / awp">
                        </div>

                        <div class="space-y-1">
                            <label for="slug" class="text-xs font-medium text-slate-200">Slug</label>
                            <input id="slug" name="slug" type="text" value="{{ old('slug') }}" required class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                        </div>

                        <div class="space-y-1">
                            <label for="version" class="text-xs font-medium text-slate-200">Версия</label>
                            <input id="version" name="version" type="text" value="{{ old('version') }}" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                        </div>

                        <div class="space-y-1 md:col-span-2">
                            <label for="archive" class="text-xs font-medium text-slate-200">Zip архив</label>
                            <input id="archive" name="archive" type="file" required class="block w-full text-xs text-slate-200" accept=".zip">
                        </div>

                        <div class="flex items-center gap-4 md:col-span-2">
                            <label class="inline-flex items-center gap-2 text-xs text-slate-200">
                                <input type="checkbox" name="restart_required" value="1" {{ old('restart_required') ? 'checked' : '' }}>
                                <span>Требуется рестарт</span>
                            </label>
                            <label class="inline-flex items-center gap-2 text-xs text-slate-200">
                                <input type="checkbox" name="active" value="1" {{ old('active', true) ? 'checked' : '' }}>
                                <span>Активна</span>
                            </label>
                        </div>
                    </div>

                    <div class="pt-2">
                        <button type="submit" class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-slate-800">
                            Сохранить
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </section>
@endsection
