@extends('layouts.app-admin')

@section('page_title', 'Редактировать новость')

@section('content')
    <div class="rounded-3xl border border-white/10 bg-[#242f3d] shadow-sm overflow-hidden">
        <div class="border-b border-white/10 bg-black/10 px-4 py-3 flex items-center justify-between gap-3">
            <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Редактирование</div>
            <a href="{{ route('admin.news.index') }}" class="rounded-md border border-white/10 bg-black/10 px-2 py-1 text-xs text-slate-100 hover:bg-black/15">Назад</a>
        </div>
        <div class="p-4">
            @if(session('success'))
                <div class="mb-4 rounded-md bg-emerald-50 p-3 text-xs text-emerald-800">
                    {{ session('success') }}
                </div>
            @endif

            <form method="POST" action="{{ route('admin.news.update', $news) }}" enctype="multipart/form-data" class="space-y-4">
                @csrf
                @method('PUT')

                <div class="grid gap-4 md:grid-cols-2">
                    <div class="space-y-1 md:col-span-2">
                        <div class="text-slate-500 text-xs">Заголовок</div>
                        <input name="title" value="{{ old('title', $news->title) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" required>
                        @error('title')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Slug</div>
                        <input name="slug" value="{{ old('slug', $news->slug) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="news-item">
                        @error('slug')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1">
                        <div class="text-slate-500 text-xs">Дата публикации</div>
                        <input name="published_at" value="{{ old('published_at', optional($news->published_at)->format('Y-m-d H:i')) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="2026-01-09 12:00">
                        @error('published_at')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1 md:col-span-2">
                        <div class="text-slate-500 text-xs">Короткий текст</div>
                        <input name="excerpt" value="{{ old('excerpt', $news->excerpt) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400">
                        @error('excerpt')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1 md:col-span-2">
                        <div class="text-slate-500 text-xs">Текст</div>
                        <textarea name="body" rows="10" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400">{{ old('body', $news->body) }}</textarea>
                        @error('body')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    <div class="space-y-1 md:col-span-2">
                        <div class="text-slate-500 text-xs">Добавить изображения</div>
                        <input type="file" name="images[]" multiple class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                        @error('images')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                        @error('images.*')<div class="text-xs text-red-400">{{ $message }}</div>@enderror
                    </div>

                    @if(!empty($news->images) && count($news->images) > 0)
                        <div class="md:col-span-2">
                            <div class="text-slate-500 text-xs mb-2">Текущие изображения (отметь для удаления)</div>
                            <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                                @foreach($news->images as $img)
                                    @php $url = \Illuminate\Support\Facades\Storage::disk('public')->url((string) $img->path); @endphp
                                    <label class="rounded-2xl border border-white/10 bg-black/10 p-3 text-xs text-slate-200">
                                        <div class="aspect-video overflow-hidden rounded-xl border border-white/10 bg-black/20">
                                            <img src="{{ $url }}" alt="" class="h-full w-full object-cover">
                                        </div>
                                        <div class="mt-2 flex items-center justify-between gap-2">
                                            <a href="{{ $url }}" target="_blank" class="text-slate-200 hover:text-white underline">Открыть</a>
                                            <span class="inline-flex items-center gap-2">
                                                <input type="checkbox" name="delete_image_ids[]" value="{{ $img->id }}">
                                                <span>Удалить</span>
                                            </span>
                                        </div>
                                    </label>
                                @endforeach
                            </div>
                        </div>
                    @endif

                    <div class="space-y-1">
                        <label class="inline-flex items-center gap-2 text-xs text-slate-200">
                            <input type="checkbox" name="active" value="1" {{ old('active', $news->active ? '1' : '') ? 'checked' : '' }}>
                            <span>Активна</span>
                        </label>
                    </div>
                </div>

                <div class="pt-2 flex items-center gap-2">
                    <button type="submit" class="inline-flex items-center rounded-md bg-sky-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-sky-500">
                        Сохранить
                    </button>
                </div>
            </form>

            <form method="POST" action="{{ route('admin.news.destroy', $news) }}" onsubmit="return confirm('Удалить новость?')" class="mt-2">
                @csrf
                @method('DELETE')
                <button type="submit" class="inline-flex items-center rounded-md bg-red-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-red-500">
                    Удалить
                </button>
            </form>
        </div>
    </div>
@endsection
