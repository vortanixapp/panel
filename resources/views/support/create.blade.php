@extends('layouts.app-user')

@section('page_title', 'Новый тикет')

@section('content')
    <section class="py-6 md:py-8">
        <div class="w-full px-4 sm:px-6 lg:px-8">
            <div class="mb-6 flex items-end justify-between gap-3">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Новый тикет</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Опиши проблему и отправь сообщение в поддержку.</p>
                </div>
                <a href="{{ route('support.index') }}" class="text-xs text-slate-300/80 hover:text-white">Назад</a>
            </div>

            <div class="rounded-3xl border border-white/10 bg-[#242f3d] p-5 shadow-sm">
                @if ($errors->any())
                    <div class="mb-4 rounded-2xl border border-rose-500/20 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
                        <ul class="list-disc space-y-1 pl-4">
                            @foreach ($errors->all() as $error)
                                <li>{{ $error }}</li>
                            @endforeach
                        </ul>
                    </div>
                @endif

                <form method="POST" action="{{ route('support.store') }}" class="grid gap-4">
                    @csrf

                    <div class="space-y-1">
                        <label for="subject" class="block text-xs font-semibold text-slate-300/70">Тема</label>
                        <input id="subject" name="subject" value="{{ old('subject') }}" required class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500" />
                    </div>

                    <div class="space-y-1">
                        <label for="message" class="block text-xs font-semibold text-slate-300/70">Сообщение</label>
                        <textarea id="message" name="message" rows="7" required class="block w-full rounded-xl border border-white/10 bg-black/10 px-3 py-2.5 text-sm text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">{{ old('message') }}</textarea>
                    </div>

                    <div class="flex justify-end">
                        <button type="submit" class="inline-flex h-10 items-center justify-center rounded-xl bg-sky-600 px-5 text-xs font-semibold text-white shadow-sm hover:bg-sky-500">
                            Отправить
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </section>
@endsection
