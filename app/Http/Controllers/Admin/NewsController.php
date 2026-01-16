<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\News;
use App\Models\NewsImage;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Storage;
use Illuminate\Support\Str;
use Illuminate\View\View;

class NewsController extends Controller
{
    public function index(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $items = News::query()
            ->orderByDesc('published_at')
            ->orderByDesc('id')
            ->paginate(20);

        return view('admin.news.index', [
            'items' => $items,
        ]);
    }

    public function create(): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return view('admin.news.create');
    }

    public function store(Request $request): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'title' => ['required', 'string', 'min:2', 'max:191'],
            'slug' => ['nullable', 'string', 'max:191', 'unique:news,slug'],
            'excerpt' => ['nullable', 'string', 'max:255'],
            'body' => ['nullable', 'string'],
            'published_at' => ['nullable', 'date'],
            'active' => ['sometimes', 'boolean'],
            'images' => ['nullable', 'array'],
            'images.*' => ['nullable', 'file', 'max:5120'],
        ]);

        $slug = trim((string) $validated['slug']);
        if ($slug === '') {
            $slug = Str::slug((string) $validated['title']);
        }
        if ($slug === '') {
            $slug = 'news-' . date('YmdHis');
        }

        $publishedAt = $validated['published_at'];

        $news = News::create([
            'title' => $validated['title'],
            'slug' => $slug,
            'excerpt' => $validated['excerpt'],
            'body' => $validated['body'],
            'published_at' => $publishedAt,
            'active' => (bool) $validated['active'],
        ]);

        $images = $request->file('images');
        if (is_array($images) && count($images) > 0) {
            $sort = 0;
            foreach ($images as $img) {
                if (! $img || ! $img->isValid()) {
                    continue;
                }
                $ext = strtolower((string) $img->getClientOriginalExtension());
                if (! in_array($ext, ['png', 'jpg', 'jpeg', 'webp', 'gif'], true)) {
                    continue;
                }
                $filename = date('YmdHis') . '-' . Str::random(8) . '.' . $ext;
                $path = $img->storeAs('news/' . (string) $news->id, $filename, 'public');
                NewsImage::create([
                    'news_id' => (int) $news->id,
                    'path' => (string) $path,
                    'sort' => $sort,
                ]);
                $sort++;
            }
        }

        return redirect()->route('admin.news.index')->with('success', 'Новость добавлена');
    }

    public function edit(News $news): View|RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $news->load('images');

        return view('admin.news.edit', [
            'news' => $news,
        ]);
    }

    public function show(News $news): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        return redirect()->route('admin.news.edit', $news);
    }

    public function update(Request $request, News $news): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $validated = $request->validate([
            'title' => ['required', 'string', 'min:2', 'max:191'],
            'slug' => ['nullable', 'string', 'max:191', 'unique:news,slug,' . $news->id],
            'excerpt' => ['nullable', 'string', 'max:255'],
            'body' => ['nullable', 'string'],
            'published_at' => ['nullable', 'date'],
            'active' => ['sometimes', 'boolean'],
            'images' => ['nullable', 'array'],
            'images.*' => ['nullable', 'file', 'max:5120'],
            'delete_image_ids' => ['nullable', 'array'],
            'delete_image_ids.*' => ['nullable', 'integer'],
        ]);

        $slug = trim((string) $validated['slug']);
        if ($slug === '') {
            $slug = Str::slug((string) $validated['title']);
        }
        if ($slug === '') {
            $slug = (string) ($news->slug ?: ('news-' . date('YmdHis')));
        }

        $news->update([
            'title' => $validated['title'],
            'slug' => $slug,
            'excerpt' => $validated['excerpt'],
            'body' => $validated['body'],
            'published_at' => $validated['published_at'],
            'active' => (bool) $validated['active'],
        ]);

        $deleteIds = is_array($validated['delete_image_ids']) ? $validated['delete_image_ids'] : [];
        $deleteIds = array_values(array_unique(array_map(fn ($v) => (int) $v, $deleteIds)));
        if (count($deleteIds) > 0) {
            $toDelete = NewsImage::query()
                ->where('news_id', (int) $news->id)
                ->whereIn('id', $deleteIds)
                ->get();
            foreach ($toDelete as $img) {
                $p = (string) $img->path;
                if ($p !== '') {
                    Storage::disk('public')->delete($p);
                }
                $img->delete();
            }
        }

        $images = $request->file('images');
        if (is_array($images) && count($images) > 0) {
            $maxSort = (int) NewsImage::query()->where('news_id', (int) $news->id)->max('sort');
            $sort = $maxSort + 1;
            foreach ($images as $img) {
                if (! $img || ! $img->isValid()) {
                    continue;
                }
                $ext = strtolower((string) $img->getClientOriginalExtension());
                if (! in_array($ext, ['png', 'jpg', 'jpeg', 'webp', 'gif'], true)) {
                    continue;
                }
                $filename = date('YmdHis') . '-' . Str::random(8) . '.' . $ext;
                $path = $img->storeAs('news/' . (string) $news->id, $filename, 'public');
                NewsImage::create([
                    'news_id' => (int) $news->id,
                    'path' => (string) $path,
                    'sort' => $sort,
                ]);
                $sort++;
            }
        }

        return redirect()->route('admin.news.index')->with('success', 'Новость обновлена');
    }

    public function destroy(News $news): RedirectResponse
    {
        if (! Auth::user() || ! Auth::user()->is_admin) {
            return redirect()->route('dashboard');
        }

        $imgs = NewsImage::query()->where('news_id', (int) $news->id)->get();
        foreach ($imgs as $img) {
            $p = (string) $img->path;
            if ($p !== '') {
                Storage::disk('public')->delete($p);
            }
        }
        Storage::disk('public')->deleteDirectory('news/' . (string) $news->id);

        $news->delete();

        return redirect()->route('admin.news.index')->with('success', 'Новость удалена');
    }
}
