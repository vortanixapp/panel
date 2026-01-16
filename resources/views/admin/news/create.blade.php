@extends('layouts.app-admin')

@section('page_title', 'Добавить новость')

@section('content')
    <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
        <div class="border-b border-white/10 bg-black/10 px-4 py-3 flex items-center justify-between gap-3">
            <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Новая новость</div>
            <a href="{{ route('admin.news.index') }}" class="rounded-md border border-white/10 bg-black/10 px-2 py-1 text-xs text-slate-100 hover:bg-black/15">Назад</a>
        </div>
        <div class="p-4">
            <form method="POST" action="{{ route('admin.news.store') }}" enctype="multipart/form-data" class="space-y-4">
                @csrf

                <div class="grid gap-4 md:grid-cols-2">
                    <div class="space-y-1 md:col-span-2">
                        <div class="text-slate-500 text-xs">Заголовок</div>
                        <input name="title" value="{{ old('title') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" required>
                        @error('title')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Slug</div>
                        <input name="slug" value="{{ old('slug') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="news-item">
                        @error('slug')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Дата публикации</div>
                        <input name="published_at" value="{{ old('published_at') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="2026-01-09 12:00">
                        @error('published_at')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1 md:col-span-2">
                        <div class="text-slate-500 text-xs">Короткий текст</div>
                        <input name="excerpt" value="{{ old('excerpt') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400">
                        @error('excerpt')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1 md:col-span-2">
                        <div class="text-slate-500 text-xs">Текст</div>
                        <textarea name="body" rows="8" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400">{{ old('body') }}</textarea>
                        @error('body')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1 md:col-span-2">
                        <div class="text-slate-500 text-xs">Изображения</div>
                        <input type="file" name="images[]" multiple class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                        @error('images')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                        @error('images.*')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <label class="inline-flex items-center gap-2 text-xs text-slate-200">
                            <input type="checkbox" name="active" value="1" {{ old('active') ? 'checked' : '' }}>
                            <span>Активна</span>
                        </label>
                    </div>
                </div>

                <div class="pt-2">
                    <button type="submit" class="inline-flex items-center rounded-md bg-sky-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-sky-500">
                        Сохранить
                    </button>
                </div>
            </form>
        </div>
    </div>
@endsection
