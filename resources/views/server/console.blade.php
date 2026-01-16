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
            @php
                $tab = 'console';
            @endphp

            <div class="overflow-hidden rounded-3xl bg-[#242f3d] text-slate-100 shadow-sm transition-shadow duration-300 hover:shadow-md">
                <div>
                    @include('server.partials.tabs')
                </div>
            </div>

            @include('server.partials.console')
        </div>
    </section>

    @include('server.partials.scripts-console')
@endsection
