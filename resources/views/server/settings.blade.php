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
                $tab = 'settings';
            @endphp

            <div class="rounded-2xl bg-[#242f3d] text-slate-100 shadow-sm overflow-hidden">
                <div class="bg-black/10">
                    @include('server.partials.tabs')
                </div>
            </div>

            <div class="mt-6 rounded-2xl border border-white/10 bg-[#242f3d] text-slate-100 shadow-sm overflow-hidden">
                <div class="border-b border-white/10 bg-black/10 px-4 py-3 text-[11px] uppercase tracking-wide text-slate-300/70">
                    Настройки сервера
                </div>
                <div class="p-4">
                    @php
                        $gameCode = strtolower((string) ($server->game->code ?? $server->game->slug ?? ''));
                        $isCs16 = in_array($gameCode, ['cs16', 'cstrike', 'counter-strike', 'counter_strike', 'cs_1_6'], true);
                        $isCs2 = in_array($gameCode, ['cs2', 'counter-strike2', 'counter_strike2', 'counter-strike_2', 'counter_strike_2'], true);
                        $isRust = in_array($gameCode, ['rust'], true);
                        $isTf2 = in_array($gameCode, ['tf2', 'teamfortress2', 'team_fortress_2', 'tf'], true);
                        $isCss = in_array($gameCode, ['css', 'cs:s', 'cs_source', 'counter-strike_source', 'counter_strike_source'], true);
                        $isGmod = in_array($gameCode, ['gmod', 'garrysmod', "garry's mod", 'garrys_mod'], true);
                        $isUnturned = in_array($gameCode, ['unturned', 'unturn', 'ut', 'untrm4', 'untrm5'], true);
                        $isMinecraft = in_array($gameCode, ['mcjava', 'mcpaper', 'mcspigot', 'mcforge', 'mcfabric', 'mcbedrock', 'mcbedrk', 'bedrock'], true);
                    @endphp

                    @if(in_array($gameCode, ['samp', 'crmp'], true))
                        @if(!empty($cfgError ?? null))
                            <div class="mb-4 rounded-md bg-amber-50 p-3 text-xs text-amber-800">
                                Не удалось прочитать server.cfg: {{ $cfgError }}
                            </div>
                        @endif

                        <form method="POST" action="{{ $gameCode === 'crmp' ? route('server.settings.crmp', $server) : route('server.settings.samp', $server) }}">
                            @csrf

                            <div class="grid gap-4 md:grid-cols-2 text-sm">
                                <div class="space-y-1">
                                    <div class="text-slate-500">Hostname</div>
                                    <input name="hostname" value="{{ old('hostname', $cfg['hostname'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="Vortanix SA-MP Server">
                                    <div class="text-xs text-slate-300/70">Название сервера, которое видят игроки в браузере SA-MP.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-500">WebURL</div>
                                    <input name="weburl" value="{{ old('weburl', $cfg['weburl'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="example.com">
                                    <div class="text-xs text-slate-300/70">Сайт/ссылка сервера, отображается в информации о сервере.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-500">RCON пароль</div>
                                    <input name="rcon_password" value="{{ old('rcon_password', $cfg['rcon_password'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="changeme">
                                    <div class="text-xs text-slate-300/70">Пароль для удалённого управления (RCON). Храни в секрете.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-500">Макс. игроков</div>
                                    <input name="maxplayers" type="number" min="1" max="1000" value="{{ old('maxplayers', $cfg['maxplayers'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="50">
                                    <div class="text-xs text-slate-300/70">Ограничение максимального онлайна.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-500">LanMode</div>
                                    <select name="lanmode" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $lanmode = (string) old('lanmode', $cfg['lanmode'] ?? '0'); @endphp
                                        <option value="0" @if($lanmode==='0') selected @endif>0</option>
                                        <option value="1" @if($lanmode==='1') selected @endif>1</option>
                                    </select>
                                    <div class="text-xs text-slate-300/70">1 — LAN режим (без публичной видимости), 0 — обычный режим.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-500">Announce</div>
                                    <select name="announce" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $announce = (string) old('announce', $cfg['announce'] ?? '0'); @endphp
                                        <option value="0" @if($announce==='0') selected @endif>0</option>
                                        <option value="1" @if($announce==='1') selected @endif>1</option>
                                    </select>
                                    <div class="text-xs text-slate-300/70">Анонсировать сервер в master-листе (если используется).</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-500">Query</div>
                                    <select name="query" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $query = (string) old('query', $cfg['query'] ?? '1'); @endphp
                                        <option value="0" @if($query==='0') selected @endif>0</option>
                                        <option value="1" @if($query==='1') selected @endif>1</option>
                                    </select>
                                    <div class="text-xs text-slate-500">Разрешить запросы статуса (онлайн/название) для браузера серверов.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-500">RCON</div>
                                    <select name="rcon" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $rcon = (string) old('rcon', $cfg['rcon'] ?? '0'); @endphp
                                        <option value="0" @if($rcon==='0') selected @endif>0</option>
                                        <option value="1" @if($rcon==='1') selected @endif>1</option>
                                    </select>
                                    <div class="text-xs text-slate-500">Включить/выключить удалённое управление RCON.</div>
                                </div>
                            </div>

                            <div class="mt-6 border-t border-white/10 pt-6">
                                <div class="mb-3 text-[11px] uppercase tracking-wide text-slate-300/70">Логи и файлы</div>
                                <div class="grid gap-4 md:grid-cols-2 text-sm">
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">logtimeformat</div>
                                        <input name="logtimeformat" value="{{ old('logtimeformat', $cfg['logtimeformat'] ?? '[%d/%m/%Y %H:%M:%S]') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="[%d/%m/%Y %H:%M:%S]">
                                        <div class="text-xs text-slate-300/70">Формат времени в логах сервера.</div>
                                    </div>

                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">logqueries</div>
                                        <select name="logqueries" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                            @php $logqueries = (string) old('logqueries', $cfg['logqueries'] ?? '0'); @endphp
                                            <option value="0" @if($logqueries==='0') selected @endif>0</option>
                                            <option value="1" @if($logqueries==='1') selected @endif>1</option>
                                        </select>
                                        <div class="text-xs text-slate-300/70">Логировать запросы (SQL) — полезно для отладки, может нагружать диск.</div>
                                    </div>

                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">logbans</div>
                                        <select name="logbans" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                            @php $logbans = (string) old('logbans', $cfg['logbans'] ?? '0'); @endphp
                                            <option value="0" @if($logbans==='0') selected @endif>0</option>
                                            <option value="1" @if($logbans==='1') selected @endif>1</option>
                                        </select>
                                        <div class="text-xs text-slate-300/70">Логировать баны/разбаны в файл логов.</div>
                                    </div>
                                </div>
                            </div>

                            <div class="mt-6 border-t border-white/10 pt-6">
                                <div class="mb-3 text-[11px] uppercase tracking-wide text-slate-300/70">Игровые настройки</div>
                                <div class="grid gap-4 md:grid-cols-2 text-sm">
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">onfoot_rate</div>
                                        <input name="onfoot_rate" type="number" min="1" max="1000" value="{{ old('onfoot_rate', $cfg['onfoot_rate'] ?? '40') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="40">
                                        <div class="text-xs text-slate-300/70">Частота синхронизации игроков пешком (больше — плавнее, но выше нагрузка).</div>
                                    </div>
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">incar_rate</div>
                                        <input name="incar_rate" type="number" min="1" max="1000" value="{{ old('incar_rate', $cfg['incar_rate'] ?? '40') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="40">
                                        <div class="text-xs text-slate-300/70">Частота синхронизации в транспорте.</div>
                                    </div>
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">weapon_rate</div>
                                        <input name="weapon_rate" type="number" min="1" max="1000" value="{{ old('weapon_rate', $cfg['weapon_rate'] ?? '40') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="40">
                                        <div class="text-xs text-slate-300/70">Частота синхронизации стрельбы/оружия.</div>
                                    </div>
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">stream_distance</div>
                                        <input name="stream_distance" type="number" step="0.1" min="0" max="10000" value="{{ old('stream_distance', $cfg['stream_distance'] ?? '300.0') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="300.0">
                                        <div class="text-xs text-slate-300/70">Дистанция, на которой объекты/игроки начинают стримиться.</div>
                                    </div>
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">stream_rate</div>
                                        <input name="stream_rate" type="number" min="0" max="1000000" value="{{ old('stream_rate', $cfg['stream_rate'] ?? '1000') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="1000">
                                        <div class="text-xs text-slate-300/70">Частота стриминга (меньше — чаще обновления, но выше нагрузка).</div>
                                    </div>
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">maxnpc</div>
                                        <input name="maxnpc" type="number" min="0" max="10000" value="{{ old('maxnpc', $cfg['maxnpc'] ?? '0') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="0">
                                        <div class="text-xs text-slate-300/70">Максимальное количество NPC на сервере.</div>
                                    </div>
                                </div>
                            </div>

                            <div class="mt-6 border-t border-white/10 pt-6">
                                <div class="mb-3 text-[11px] uppercase tracking-wide text-slate-300/70">Безопасность</div>
                                <div class="grid gap-4 md:grid-cols-2 text-sm">
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">anticheat</div>
                                        <select name="anticheat" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                            @php $anticheat = (string) old('anticheat', $cfg['anticheat'] ?? '1'); @endphp
                                            <option value="0" @if($anticheat==='0') selected @endif>0</option>
                                            <option value="1" @if($anticheat==='1') selected @endif>1</option>
                                        </select>
                                        <div class="text-xs text-slate-300/70">Включить встроенный античит SA-MP.</div>
                                    </div>

                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">lagcompmode</div>
                                        <select name="lagcompmode" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                            @php $lagcompmode = (string) old('lagcompmode', $cfg['lagcompmode'] ?? '1'); @endphp
                                            <option value="0" @if($lagcompmode==='0') selected @endif>0</option>
                                            <option value="1" @if($lagcompmode==='1') selected @endif>1</option>
                                        </select>
                                        <div class="text-xs text-slate-300/70">Режим лаг-компенсации для сетевой игры.</div>
                                    </div>

                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">connseedtime</div>
                                        <input name="connseedtime" type="number" min="0" max="10000000" value="{{ old('connseedtime', $cfg['connseedtime'] ?? '300000') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="300000">
                                        <div class="text-xs text-slate-300/70">Параметр защиты сетевых соединений (в миллисекундах).</div>
                                    </div>

                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">minconnectiontime</div>
                                        <input name="minconnectiontime" type="number" min="0" max="10000000" value="{{ old('minconnectiontime', $cfg['minconnectiontime'] ?? '0') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="0">
                                        <div class="text-xs text-slate-300/70">Минимальное время соединения (мс) для защиты от флуда подключений.</div>
                                    </div>

                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">messageholelimit</div>
                                        <input name="messageholelimit" type="number" min="0" max="10000000" value="{{ old('messageholelimit', $cfg['messageholelimit'] ?? '3000') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="3000">
                                        <div class="text-xs text-slate-300/70">Лимит «дыр» сообщений (анти-спам/анти-эксплойт).</div>
                                    </div>

                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">ackslimit</div>
                                        <input name="ackslimit" type="number" min="0" max="10000000" value="{{ old('ackslimit', $cfg['ackslimit'] ?? '3000') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="3000">
                                        <div class="text-xs text-slate-300/70">Лимит ACK-пакетов для защиты от сетевого спама.</div>
                                    </div>

                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">playertimeout</div>
                                        <input name="playertimeout" type="number" min="0" max="10000000" value="{{ old('playertimeout', $cfg['playertimeout'] ?? '10000') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="10000">
                                        <div class="text-xs text-slate-300/70">Таймаут игрока (мс) до отключения при отсутствии данных.</div>
                                    </div>
                                </div>
                            </div>

                            <div class="mt-6 border-t border-white/10 pt-6">
                                <div class="mb-3 text-[11px] uppercase tracking-wide text-slate-300/70">Прочее</div>
                                <div class="grid gap-4 md:grid-cols-2 text-sm">
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">worldtime</div>
                                        <input name="worldtime" type="number" min="0" max="23" value="{{ old('worldtime', $cfg['worldtime'] ?? '12') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="12">
                                        <div class="text-xs text-slate-300/70">Время в мире (час, 0–23).</div>
                                    </div>

                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">sleep</div>
                                        <input name="sleep" type="number" min="0" max="1000" value="{{ old('sleep', $cfg['sleep'] ?? '5') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="5">
                                        <div class="text-xs text-slate-300/70">Задержка цикла сервера (мс). Может снижать нагрузку.</div>
                                    </div>

                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">mtu</div>
                                        <input name="mtu" type="number" min="0" max="10000" value="{{ old('mtu', $cfg['mtu'] ?? '1400') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="1400">
                                        <div class="text-xs text-slate-300/70">MTU для сетевых пакетов. Меняй только если понимаешь, зачем.</div>
                                    </div>
                                </div>
                            </div>

                            <div class="mt-5 flex items-center gap-3">
                                <button type="submit" class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800">
                                    Сохранить
                                </button>
                                <div class="text-xs text-slate-500">Для применения может потребоваться перезапуск сервера.</div>
                            </div>
                        </form>

                        <div class="mt-6 rounded-md bg-black/10 border border-white/10 p-3 text-xs text-slate-200">
                            <div class="font-semibold text-slate-100">server.cfg (текущий)</div>
                            <pre class="mt-2 whitespace-pre-wrap break-words">{{ $cfgRaw ?? '' }}</pre>
                        </div>
                    @elseif($isRust)
                        @if(!empty($cfgError ?? null))
                            <div class="mb-4 rounded-md bg-amber-50 p-3 text-xs text-amber-800">
                                Не удалось прочитать rust.env: {{ $cfgError }}
                            </div>
                        @endif

                        <form method="POST" action="{{ route('server.settings.rust', $server) }}">
                            @csrf

                            <div class="grid gap-4 md:grid-cols-2 text-sm">
                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Hostname</div>
                                    <input name="hostname" value="{{ old('hostname', $cfg['RUST_HOSTNAME'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="Vortanix Rust Server">
                                    <div class="text-xs text-slate-300/70">Название сервера (server.hostname).</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Макс. игроков</div>
                                    <input name="maxplayers" type="number" min="1" max="500" value="{{ old('maxplayers', $cfg['RUST_MAXPLAYERS'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="50">
                                    <div class="text-xs text-slate-300/70">server.maxplayers (1–500).</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Размер мира</div>
                                    <input name="world_size" type="number" min="1000" max="6000" value="{{ old('world_size', $cfg['RUST_WORLD_SIZE'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="3500">
                                    <div class="text-xs text-slate-300/70">server.worldsize (1000–6000).</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Seed</div>
                                    <input name="seed" type="number" min="0" max="2147483647" value="{{ old('seed', $cfg['RUST_SEED'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                    <div class="text-xs text-slate-300/70">server.seed (если пусто — будет выбран автоматически).</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Identity</div>
                                    <input name="identity" value="{{ old('identity', $cfg['RUST_IDENTITY'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="server">
                                    <div class="text-xs text-slate-300/70">server.identity (название папки сохранений).</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Level</div>
                                    <input name="level" value="{{ old('level', $cfg['RUST_LEVEL'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="Procedural Map">
                                    <div class="text-xs text-slate-300/70">server.level (обычно Procedural Map).</div>
                                </div>

                                <div class="space-y-1 md:col-span-2">
                                    <div class="text-slate-300/70">RCON пароль</div>
                                    <input name="rcon_password" value="{{ old('rcon_password', $cfg['RUST_RCON_PASSWORD'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                    <div class="text-xs text-slate-300/70">Если пусто — будет сгенерирован при старте.</div>
                                </div>
                            </div>

                            <div class="mt-5 flex items-center gap-3">
                                <button type="submit" class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800">
                                    Сохранить
                                </button>
                                <div class="text-xs text-slate-500">Для применения требуется перезапуск сервера.</div>
                            </div>
                        </form>

                        <div class="mt-6 rounded-md bg-black/10 border border-white/10 p-3 text-xs text-slate-200">
                            <div class="font-semibold text-slate-100">rust.env (текущий: {{ $cfgPath ?? '' }})</div>
                            <pre class="mt-2 whitespace-pre-wrap break-words">{{ $cfgRaw ?? '' }}</pre>
                        </div>
                    @elseif($isCs2)
                        @if(!empty($cfgError ?? null))
                            <div class="mb-4 rounded-md bg-amber-50 p-3 text-xs text-amber-800">
                                Не удалось прочитать server.cfg: {{ $cfgError }}
                            </div>
                        @endif

                        <form method="POST" action="{{ route('server.settings.cs2', $server) }}">
                            @csrf

                            <div class="grid gap-4 md:grid-cols-2 text-sm">
                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Hostname</div>
                                    <input name="hostname" value="{{ old('hostname', $cfg['hostname'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="Vortanix CS2 Server">
                                    <div class="text-xs text-slate-300/70">Название сервера в браузере.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">RCON пароль</div>
                                    <input name="rcon_password" value="{{ old('rcon_password', $cfg['rcon_password'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="changeme">
                                    <div class="text-xs text-slate-300/70">Пароль RCON (rcon_password).</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">GSLT токен (sv_setsteamaccount)</div>
                                    <input name="sv_setsteamaccount" value="{{ old('sv_setsteamaccount', $cfg['sv_setsteamaccount'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                    <div class="text-xs text-slate-300/70">Нужен для Steam авторизации сервера. Токен можно получить в Steam (Game Server Login Token).</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Пароль сервера (sv_password)</div>
                                    <input name="sv_password" value="{{ old('sv_password', $cfg['sv_password'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                    <div class="text-xs text-slate-300/70">Если заполнено — вход на сервер будет по паролю.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Макс. игроков</div>
                                    <input value="{{ (int) ($server->slots ?? 0) }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 opacity-80" disabled>
                                    <div class="text-xs text-slate-300/70">Берётся из слотов тарифа и применяется как CS2_MAXPLAYERS при запуске.</div>
                                </div>
                            </div>

                            <div class="mt-5 flex items-center gap-3">
                                <button type="submit" class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800">
                                    Сохранить
                                </button>
                                <div class="text-xs text-slate-500">Для применения может потребоваться перезапуск сервера.</div>
                            </div>
                        </form>

                        <div class="mt-6 rounded-md bg-black/10 border border-white/10 p-3 text-xs text-slate-200">
                            <div class="font-semibold text-slate-100">server.cfg (текущий: {{ $cfgPath ?? '' }})</div>
                            <pre class="mt-2 whitespace-pre-wrap break-words">{{ $cfgRaw ?? '' }}</pre>
                        </div>
                    @elseif($isCs16)
                        @if(!empty($cfgError ?? null))
                            <div class="mb-4 rounded-md bg-amber-50 p-3 text-xs text-amber-800">
                                Не удалось прочитать server.cfg: {{ $cfgError }}
                            </div>
                        @endif

                        <form method="POST" action="{{ route('server.settings.cs16', $server) }}">
                            @csrf

                            <input type="hidden" name="cfg_path" value="{{ old('cfg_path', $cfgPath ?? 'cstrike/server.cfg') }}">

                            <div class="grid gap-4 md:grid-cols-2 text-sm">
                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Hostname</div>
                                    <input name="hostname" value="{{ old('hostname', $cfg['hostname'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="Vortanix CS 1.6 Server">
                                    <div class="text-xs text-slate-300/70">Название сервера в браузере.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">RCON пароль</div>
                                    <input name="rcon_password" value="{{ old('rcon_password', $cfg['rcon_password'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="changeme">
                                    <div class="text-xs text-slate-300/70">Пароль RCON (rcon_password).</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Пароль сервера (sv_password)</div>
                                    <input name="sv_password" value="{{ old('sv_password', $cfg['sv_password'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                    <div class="text-xs text-slate-300/70">Если заполнено — вход на сервер будет по паролю.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">sv_lan</div>
                                    <select name="sv_lan" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $svlan = (string) old('sv_lan', $cfg['sv_lan'] ?? '0'); @endphp
                                        <option value="0" @if($svlan==='0') selected @endif>0</option>
                                        <option value="1" @if($svlan==='1') selected @endif>1</option>
                                    </select>
                                    <div class="text-xs text-slate-300/70">1 — LAN режим, 0 — публичный.</div>
                                </div>
                            </div>

                            <div class="mt-6 border-t border-white/10 pt-6">
                                <div class="mb-3 text-[11px] uppercase tracking-wide text-slate-300/70">Режим и раунды</div>
                                <div class="grid gap-4 md:grid-cols-2 text-sm">
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">mp_timelimit</div>
                                        <input name="mp_timelimit" type="number" min="0" max="10000" value="{{ old('mp_timelimit', $cfg['mp_timelimit'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="0">
                                    </div>
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">mp_roundtime</div>
                                        <input name="mp_roundtime" type="number" min="0" max="10000" value="{{ old('mp_roundtime', $cfg['mp_roundtime'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="3">
                                    </div>
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">mp_friendlyfire</div>
                                        <select name="mp_friendlyfire" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                            @php $ff = (string) old('mp_friendlyfire', $cfg['mp_friendlyfire'] ?? '0'); @endphp
                                            <option value="0" @if($ff==='0') selected @endif>0</option>
                                            <option value="1" @if($ff==='1') selected @endif>1</option>
                                        </select>
                                    </div>
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">mp_autokick</div>
                                        <select name="mp_autokick" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                            @php $ak = (string) old('mp_autokick', $cfg['mp_autokick'] ?? '1'); @endphp
                                            <option value="0" @if($ak==='0') selected @endif>0</option>
                                            <option value="1" @if($ak==='1') selected @endif>1</option>
                                        </select>
                                    </div>
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">mp_autoteambalance</div>
                                        <select name="mp_autoteambalance" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                            @php $atb = (string) old('mp_autoteambalance', $cfg['mp_autoteambalance'] ?? '1'); @endphp
                                            <option value="0" @if($atb==='0') selected @endif>0</option>
                                            <option value="1" @if($atb==='1') selected @endif>1</option>
                                        </select>
                                    </div>
                                    <div class="space-y-1">
                                        <div class="text-slate-300/70">mp_limitteams</div>
                                        <input name="mp_limitteams" type="number" min="0" max="1000" value="{{ old('mp_limitteams', $cfg['mp_limitteams'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="2">
                                    </div>
                                </div>
                            </div>

                            <div class="mt-5 flex items-center gap-3">
                                <button type="submit" class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800">
                                    Сохранить
                                </button>
                                <div class="text-xs text-slate-500">Для применения может потребоваться перезапуск сервера.</div>
                            </div>
                        </form>

                        <div class="mt-6 rounded-md bg-black/10 border border-white/10 p-3 text-xs text-slate-200">
                            <div class="font-semibold text-slate-100">server.cfg (текущий: {{ $cfgPath ?? '' }})</div>
                            <pre class="mt-2 whitespace-pre-wrap break-words">{{ $cfgRaw ?? '' }}</pre>
                        </div>
                    @elseif($isTf2)
                        @if(!empty($cfgError ?? null))
                            <div class="mb-4 rounded-md bg-amber-50 p-3 text-xs text-amber-800">
                                Не удалось прочитать server.cfg: {{ $cfgError }}
                            </div>
                        @endif

                        <form method="POST" action="{{ route('server.settings.tf2', $server) }}">
                            @csrf

                            <div class="grid gap-4 md:grid-cols-2 text-sm">
                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Hostname</div>
                                    <input name="hostname" value="{{ old('hostname', $cfg['hostname'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="Vortanix TF2 Server">
                                    <div class="text-xs text-slate-300/70">Название сервера в браузере.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">RCON пароль</div>
                                    <input name="rcon_password" value="{{ old('rcon_password', $cfg['rcon_password'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="changeme">
                                    <div class="text-xs text-slate-300/70">Пароль RCON (rcon_password).</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Пароль сервера (sv_password)</div>
                                    <input name="sv_password" value="{{ old('sv_password', $cfg['sv_password'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                    <div class="text-xs text-slate-300/70">Если заполнено — вход на сервер будет по паролю.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">sv_lan</div>
                                    <select name="sv_lan" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $svlan = (string) old('sv_lan', $cfg['sv_lan'] ?? '0'); @endphp
                                        <option value="0" @if($svlan==='0') selected @endif>0</option>
                                        <option value="1" @if($svlan==='1') selected @endif>1</option>
                                    </select>
                                    <div class="text-xs text-slate-300/70">1 — LAN режим, 0 — публичный.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Контакт (sv_contact)</div>
                                    <input name="sv_contact" value="{{ old('sv_contact', $cfg['sv_contact'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="admin@example.com">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Теги (sv_tags)</div>
                                    <input name="sv_tags" value="{{ old('sv_tags', $cfg['sv_tags'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Боты (tf_bot_quota)</div>
                                    <input name="tf_bot_quota" type="number" min="0" max="1000" value="{{ old('tf_bot_quota', $cfg['tf_bot_quota'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="0">
                                </div>

                                <div class="space-y-1 md:col-span-2">
                                    <div class="text-slate-300/70">FastDL URL (sv_downloadurl)</div>
                                    <input name="sv_downloadurl" value="{{ old('sv_downloadurl', $cfg['sv_downloadurl'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="https://example.com/tf2">
                                </div>
                            </div>

                            <div class="mt-5 flex items-center gap-3">
                                <button type="submit" class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800">
                                    Сохранить
                                </button>
                                <div class="text-xs text-slate-500">Для применения может потребоваться перезапуск сервера.</div>
                            </div>
                        </form>

                        <div class="mt-6 rounded-md bg-black/10 border border-white/10 p-3 text-xs text-slate-200">
                            <div class="font-semibold text-slate-100">server.cfg (текущий: {{ $cfgPath ?? '' }})</div>
                            <pre class="mt-2 whitespace-pre-wrap break-words">{{ $cfgRaw ?? '' }}</pre>
                        </div>
                    @elseif($isCss)
                        @if(!empty($cfgError ?? null))
                            <div class="mb-4 rounded-md bg-amber-50 p-3 text-xs text-amber-800">
                                Не удалось прочитать server.cfg: {{ $cfgError }}
                            </div>
                        @endif

                        <form method="POST" action="{{ route('server.settings.css', $server) }}">
                            @csrf

                            <div class="grid gap-4 md:grid-cols-2 text-sm">
                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Hostname</div>
                                    <input name="hostname" value="{{ old('hostname', $cfg['hostname'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="Vortanix Counter-Strike: Source Server">
                                    <div class="text-xs text-slate-300/70">Название сервера в браузере.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">RCON пароль</div>
                                    <input name="rcon_password" value="{{ old('rcon_password', $cfg['rcon_password'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="changeme">
                                    <div class="text-xs text-slate-300/70">Пароль RCON (rcon_password).</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Пароль сервера (sv_password)</div>
                                    <input name="sv_password" value="{{ old('sv_password', $cfg['sv_password'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                    <div class="text-xs text-slate-300/70">Если заполнено — вход на сервер будет по паролю.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">sv_lan</div>
                                    <select name="sv_lan" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $svlan = (string) old('sv_lan', $cfg['sv_lan'] ?? '0'); @endphp
                                        <option value="0" @if($svlan==='0') selected @endif>0</option>
                                        <option value="1" @if($svlan==='1') selected @endif>1</option>
                                    </select>
                                    <div class="text-xs text-slate-300/70">1 — LAN режим, 0 — публичный.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Регион (sv_region)</div>
                                    <input name="sv_region" type="number" min="-1" max="255" value="{{ old('sv_region', $cfg['sv_region'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="3">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Контакт (sv_contact)</div>
                                    <input name="sv_contact" value="{{ old('sv_contact', $cfg['sv_contact'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="admin@example.com">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Теги (sv_tags)</div>
                                    <input name="sv_tags" value="{{ old('sv_tags', $cfg['sv_tags'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                </div>

                                <div class="space-y-1 md:col-span-2">
                                    <div class="text-slate-300/70">FastDL URL (sv_downloadurl)</div>
                                    <input name="sv_downloadurl" value="{{ old('sv_downloadurl', $cfg['sv_downloadurl'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="https://example.com/css">
                                </div>
                            </div>

                            <div class="mt-5 flex items-center gap-3">
                                <button type="submit" class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800">
                                    Сохранить
                                </button>
                                <div class="text-xs text-slate-500">Для применения может потребоваться перезапуск сервера.</div>
                            </div>
                        </form>

                        <div class="mt-6 rounded-md bg-black/10 border border-white/10 p-3 text-xs text-slate-200">
                            <div class="font-semibold text-slate-100">server.cfg (текущий: {{ $cfgPath ?? '' }})</div>
                            <pre class="mt-2 whitespace-pre-wrap break-words">{{ $cfgRaw ?? '' }}</pre>
                        </div>
                    @elseif($isGmod)
                        @if(!empty($cfgError ?? null))
                            <div class="mb-4 rounded-md bg-amber-50 p-3 text-xs text-amber-800">
                                Не удалось прочитать server.cfg: {{ $cfgError }}
                            </div>
                        @endif

                        <form method="POST" action="{{ route('server.settings.gmod', $server) }}">
                            @csrf

                            <div class="grid gap-4 md:grid-cols-2 text-sm">
                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Hostname</div>
                                    <input name="hostname" value="{{ old('hostname', $cfg['hostname'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="Vortanix Garrys Mod Server">
                                    <div class="text-xs text-slate-300/70">Название сервера в браузере.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">RCON пароль</div>
                                    <input name="rcon_password" value="{{ old('rcon_password', $cfg['rcon_password'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="changeme">
                                    <div class="text-xs text-slate-300/70">Пароль RCON (rcon_password).</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Пароль сервера (sv_password)</div>
                                    <input name="sv_password" value="{{ old('sv_password', $cfg['sv_password'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                    <div class="text-xs text-slate-300/70">Если заполнено — вход на сервер будет по паролю.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">sv_lan</div>
                                    <select name="sv_lan" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $svlan = (string) old('sv_lan', $cfg['sv_lan'] ?? '0'); @endphp
                                        <option value="0" @if($svlan==='0') selected @endif>0</option>
                                        <option value="1" @if($svlan==='1') selected @endif>1</option>
                                    </select>
                                    <div class="text-xs text-slate-300/70">1 — LAN режим, 0 — публичный.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Регион (sv_region)</div>
                                    <input name="sv_region" type="number" min="-1" max="255" value="{{ old('sv_region', $cfg['sv_region'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="3">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Контакт (sv_contact)</div>
                                    <input name="sv_contact" value="{{ old('sv_contact', $cfg['sv_contact'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="admin@example.com">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Теги (sv_tags)</div>
                                    <input name="sv_tags" value="{{ old('sv_tags', $cfg['sv_tags'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                </div>

                                <div class="space-y-1 md:col-span-2">
                                    <div class="text-slate-300/70">FastDL URL (sv_downloadurl)</div>
                                    <input name="sv_downloadurl" value="{{ old('sv_downloadurl', $cfg['sv_downloadurl'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="https://example.com/gmod">
                                </div>
                            </div>

                            <div class="mt-5 flex items-center gap-3">
                                <button type="submit" class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800">
                                    Сохранить
                                </button>
                                <div class="text-xs text-slate-500">Для применения может потребоваться перезапуск сервера.</div>
                            </div>
                        </form>

                        <div class="mt-6 rounded-md bg-black/10 border border-white/10 p-3 text-xs text-slate-200">
                            <div class="font-semibold text-slate-100">server.cfg (текущий: {{ $cfgPath ?? '' }})</div>
                            <pre class="mt-2 whitespace-pre-wrap break-words">{{ $cfgRaw ?? '' }}</pre>
                        </div>
                    @elseif($isUnturned)
                        @if(!empty($cfgError ?? null))
                            <div class="mb-4 rounded-md bg-amber-50 p-3 text-xs text-amber-800">
                                Не удалось прочитать Commands.dat: {{ $cfgError }}
                            </div>
                        @endif

                        <form method="POST" action="{{ route('server.settings.unturned', $server) }}">
                            @csrf

                            <div class="grid gap-4 md:grid-cols-2 text-sm">
                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Server Name (Name)</div>
                                    <input name="server_name" value="{{ old('server_name', $cfg['name'] ?? 'server') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="server">
                                    <div class="text-xs text-slate-300/70">Имя сервера / папки в /Servers. В текущей реализации используется путь Servers/server/Server/Commands.dat.</div>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Макс. игроков (MaxPlayers)</div>
                                    <input name="max_players" type="number" min="1" max="200" value="{{ old('max_players', $cfg['maxplayers'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="24">
                                    <div class="text-xs text-slate-300/70">Если тариф со слотами — будет принудительно равен слотам тарифа.</div>
                                </div>

                                <div class="space-y-1 md:col-span-2">
                                    <div class="text-slate-300/70">LoginToken</div>
                                    <input name="login_token" value="{{ old('login_token', $cfg['logintoken'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                    <div class="text-xs text-slate-300/70">Steam GSLT токен Unturned (если нужен). Оставь пустым чтобы удалить строку LoginToken.</div>
                                </div>

                                <div class="space-y-1 md:col-span-2">
                                    <div class="text-slate-300/70">Password</div>
                                    <input name="password" value="{{ old('password', $cfg['password'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                    <div class="text-xs text-slate-300/70">Пароль сервера. Оставь пустым чтобы удалить строку Password.</div>
                                </div>
                            </div>

                            <div class="mt-5 flex items-center gap-3">
                                <button type="submit" class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800">
                                    Сохранить
                                </button>
                                <div class="text-xs text-slate-500">Для применения может потребоваться перезапуск сервера.</div>
                            </div>
                        </form>

                        <div class="mt-6 rounded-md bg-black/10 border border-white/10 p-3 text-xs text-slate-200">
                            <div class="font-semibold text-slate-100">Commands.dat (текущий: {{ $cfgPath ?? '' }})</div>
                            <pre class="mt-2 whitespace-pre-wrap break-words">{{ $cfgRaw ?? '' }}</pre>
                        </div>
                    @elseif($isMinecraft)
                        @if(!empty($cfgError ?? null))
                            <div class="mb-4 rounded-md bg-amber-50 p-3 text-xs text-amber-800">
                                Не удалось прочитать server.properties: {{ $cfgError }}
                            </div>
                        @endif

                        <form method="POST" action="{{ route('server.settings.minecraft', $server) }}">
                            @csrf

                            <div class="grid gap-4 md:grid-cols-2 text-sm">
                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Описание сервера (MOTD)</div>
                                    <div class="text-xs text-slate-400">Текст, который виден в списке серверов Minecraft.</div>
                                    <input name="motd" value="{{ old('motd', $cfg['motd'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="A Minecraft Server">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Название мира (level-name)</div>
                                    <div class="text-xs text-slate-400">Имя папки мира в /data. Если изменить — будет создан/использован другой мир.</div>
                                    <input name="level_name" value="{{ old('level_name', $cfg['level-name'] ?? $cfg['level_name'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="world">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Макс. игроков (max-players)</div>
                                    <div class="text-xs text-slate-400">Лимит слотов на сервере. Не равен тарифным слотам, если они используются отдельно.</div>
                                    <input name="max_players" type="number" min="1" max="5000" value="{{ old('max_players', $cfg['max-players'] ?? $cfg['max_players'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="20">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Онлайн-режим (online-mode)</div>
                                    <div class="text-xs text-slate-400">true — только лицензия (проверка аккаунта Mojang/Microsoft). false — без проверки.</div>
                                    <select name="online_mode" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $om = (string) old('online_mode', $cfg['online-mode'] ?? 'true'); @endphp
                                        <option value="true" @if($om==='true') selected @endif>true</option>
                                        <option value="false" @if($om==='false') selected @endif>false</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">PvP (pvp)</div>
                                    <div class="text-xs text-slate-400">Разрешить урон между игроками.</div>
                                    <select name="pvp" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $pvp = (string) old('pvp', $cfg['pvp'] ?? 'true'); @endphp
                                        <option value="true" @if($pvp==='true') selected @endif>true</option>
                                        <option value="false" @if($pvp==='false') selected @endif>false</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Сложность (difficulty)</div>
                                    <div class="text-xs text-slate-400">Уровень сложности мобов и урона.</div>
                                    <select name="difficulty" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $diff = (string) old('difficulty', $cfg['difficulty'] ?? 'easy'); @endphp
                                        <option value="peaceful" @if($diff==='peaceful') selected @endif>peaceful</option>
                                        <option value="easy" @if($diff==='easy') selected @endif>easy</option>
                                        <option value="normal" @if($diff==='normal') selected @endif>normal</option>
                                        <option value="hard" @if($diff==='hard') selected @endif>hard</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Режим игры (gamemode)</div>
                                    <div class="text-xs text-slate-400">Режим по умолчанию для новых игроков.</div>
                                    <select name="gamemode" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $gm = (string) old('gamemode', $cfg['gamemode'] ?? 'survival'); @endphp
                                        <option value="survival" @if($gm==='survival') selected @endif>survival</option>
                                        <option value="creative" @if($gm==='creative') selected @endif>creative</option>
                                        <option value="adventure" @if($gm==='adventure') selected @endif>adventure</option>
                                        <option value="spectator" @if($gm==='spectator') selected @endif>spectator</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Хардкор (hardcore)</div>
                                    <div class="text-xs text-slate-400">true — режим хардкор (обычно бан/наблюдатель после смерти).</div>
                                    <select name="hardcore" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $hc = (string) old('hardcore', $cfg['hardcore'] ?? 'false'); @endphp
                                        <option value="true" @if($hc==='true') selected @endif>true</option>
                                        <option value="false" @if($hc==='false') selected @endif>false</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Принудительный режим (force-gamemode)</div>
                                    <div class="text-xs text-slate-400">Если true — игрокам принудительно ставится gamemode сервера.</div>
                                    <select name="force_gamemode" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $fgm = (string) old('force_gamemode', $cfg['force-gamemode'] ?? 'false'); @endphp
                                        <option value="true" @if($fgm==='true') selected @endif>true</option>
                                        <option value="false" @if($fgm==='false') selected @endif>false</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Разрешить полёт (allow-flight)</div>
                                    <div class="text-xs text-slate-400">Разрешает полёт (важно для некоторых модов/плагинов). Может снижать античит-эффект.</div>
                                    <select name="allow_flight" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $af = (string) old('allow_flight', $cfg['allow-flight'] ?? 'false'); @endphp
                                        <option value="true" @if($af==='true') selected @endif>true</option>
                                        <option value="false" @if($af==='false') selected @endif>false</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Дальность прорисовки (view-distance)</div>
                                    <div class="text-xs text-slate-400">Сколько чанков клиент загружает вокруг игрока. Больше — выше нагрузка.</div>
                                    <input name="view_distance" type="number" min="2" max="64" value="{{ old('view_distance', $cfg['view-distance'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="10">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Дистанция симуляции (simulation-distance)</div>
                                    <div class="text-xs text-slate-400">Радиус, в котором тикают сущности/редстоун. Больше — выше нагрузка.</div>
                                    <input name="simulation_distance" type="number" min="2" max="64" value="{{ old('simulation_distance', $cfg['simulation-distance'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="10">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Защита спавна (spawn-protection)</div>
                                    <div class="text-xs text-slate-400">Радиус защиты спавна от ломания/строительства для не-операторов.</div>
                                    <input name="spawn_protection" type="number" min="0" max="999999" value="{{ old('spawn_protection', $cfg['spawn-protection'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="16">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Разрешить Нижний мир (allow-nether)</div>
                                    <div class="text-xs text-slate-400">Включает/выключает Nether.</div>
                                    <select name="allow_nether" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $an = (string) old('allow_nether', $cfg['allow-nether'] ?? 'true'); @endphp
                                        <option value="true" @if($an==='true') selected @endif>true</option>
                                        <option value="false" @if($an==='false') selected @endif>false</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Командные блоки (enable-command-block)</div>
                                    <div class="text-xs text-slate-400">Разрешает использование командных блоков.</div>
                                    <select name="enable_command_block" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $cb = (string) old('enable_command_block', $cfg['enable-command-block'] ?? 'false'); @endphp
                                        <option value="true" @if($cb==='true') selected @endif>true</option>
                                        <option value="false" @if($cb==='false') selected @endif>false</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Белый список (white-list)</div>
                                    <div class="text-xs text-slate-400">Если true — вход только для игроков из whitelist.json (нужно управлять файлом).</div>
                                    <select name="white_list" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $wl = (string) old('white_list', $cfg['white-list'] ?? 'false'); @endphp
                                        <option value="true" @if($wl==='true') selected @endif>true</option>
                                        <option value="false" @if($wl==='false') selected @endif>false</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Принудительный whitelist (enforce-whitelist)</div>
                                    <div class="text-xs text-slate-400">Если true — не-whitelist игроки будут кикнуты при включении whitelist.</div>
                                    <select name="enforce_whitelist" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $ewl = (string) old('enforce_whitelist', $cfg['enforce-whitelist'] ?? 'false'); @endphp
                                        <option value="true" @if($ewl==='true') selected @endif>true</option>
                                        <option value="false" @if($ewl==='false') selected @endif>false</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Статус сервера (enable-status)</div>
                                    <div class="text-xs text-slate-400">Отвечать на пинг/статус (видимость в списке серверов).</div>
                                    <select name="enable_status" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $es = (string) old('enable_status', $cfg['enable-status'] ?? 'true'); @endphp
                                        <option value="true" @if($es==='true') selected @endif>true</option>
                                        <option value="false" @if($es==='false') selected @endif>false</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Query (enable-query)</div>
                                    <div class="text-xs text-slate-400">Включает GameSpy Query (используется некоторыми мониторингами).</div>
                                    <select name="enable_query" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $eq = (string) old('enable_query', $cfg['enable-query'] ?? 'false'); @endphp
                                        <option value="true" @if($eq==='true') selected @endif>true</option>
                                        <option value="false" @if($eq==='false') selected @endif>false</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Порт Query (query.port)</div>
                                    <div class="text-xs text-slate-400">Порт для Query. Важно: должен быть проброшен наружу, если нужен.</div>
                                    <input name="query_port" type="number" min="1" max="65535" value="{{ old('query_port', $cfg['query.port'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="25565">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">RCON (enable-rcon)</div>
                                    <div class="text-xs text-slate-400">Удалённая консоль. Важно: включайте только при необходимости.</div>
                                    <select name="enable_rcon" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100">
                                        @php $er = (string) old('enable_rcon', $cfg['enable-rcon'] ?? 'false'); @endphp
                                        <option value="true" @if($er==='true') selected @endif>true</option>
                                        <option value="false" @if($er==='false') selected @endif>false</option>
                                    </select>
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Пароль RCON (rcon.password)</div>
                                    <div class="text-xs text-slate-400">Пароль для доступа к RCON. Не используйте простой пароль.</div>
                                    <input name="rcon_password" value="{{ old('rcon_password', $cfg['rcon.password'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Порт RCON (rcon.port)</div>
                                    <div class="text-xs text-slate-400">Порт для RCON. Важно: должен быть проброшен наружу, если нужен.</div>
                                    <input name="rcon_port" type="number" min="1" max="65535" value="{{ old('rcon_port', $cfg['rcon.port'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="25575">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Кик за AFK (player-idle-timeout)</div>
                                    <div class="text-xs text-slate-400">В секундах. 0 — не кикать.</div>
                                    <input name="player_idle_timeout" type="number" min="0" max="999999" value="{{ old('player_idle_timeout', $cfg['player-idle-timeout'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="0">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Лимит тик-тайма (max-tick-time)</div>
                                    <div class="text-xs text-slate-400">Макс. время тика (мс) до остановки сервера. -1 — отключить.</div>
                                    <input name="max_tick_time" type="number" min="-1" max="2147483647" value="{{ old('max_tick_time', $cfg['max-tick-time'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="60000">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Макс. размер мира (max-world-size)</div>
                                    <div class="text-xs text-slate-400">Ограничение на границы мира (в блоках). Слишком большое — без смысла.</div>
                                    <input name="max_world_size" type="number" min="1" max="29999984" value="{{ old('max_world_size', $cfg['max-world-size'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="29999984">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Уровень прав OP (op-permission-level)</div>
                                    <div class="text-xs text-slate-400">1–4. Чем выше — тем больше команд доступно операторам.</div>
                                    <input name="op_permission_level" type="number" min="1" max="4" value="{{ old('op_permission_level', $cfg['op-permission-level'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="4">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Уровень прав функций (function-permission-level)</div>
                                    <div class="text-xs text-slate-400">1–4. Уровень доступа для /function и datapack-функций.</div>
                                    <input name="function_permission_level" type="number" min="1" max="4" value="{{ old('function_permission_level', $cfg['function-permission-level'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="2">
                                </div>

                                <div class="space-y-1">
                                    <div class="text-slate-300/70">Сжатие сети (network-compression-threshold)</div>
                                    <div class="text-xs text-slate-400">Порог (в байтах) для сжатия пакетов. -1 — отключить. Больше — меньше CPU, больше трафик.</div>
                                    <input name="network_compression_threshold" type="number" min="-1" max="1048576" value="{{ old('network_compression_threshold', $cfg['network-compression-threshold'] ?? '') }}" class="w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-sm text-slate-100 placeholder-slate-400" placeholder="256">
                                </div>
                            </div>

                            <div class="mt-5 flex items-center gap-3">
                                <button type="submit" class="inline-flex items-center rounded-md bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800">
                                    Сохранить
                                </button>
                                <div class="text-xs text-slate-500">Для применения может потребоваться перезапуск сервера.</div>
                            </div>
                        </form>

                        <div class="mt-6 rounded-md bg-black/10 border border-white/10 p-3 text-xs text-slate-200">
                            <div class="font-semibold text-slate-100">server.properties (текущий: {{ $cfgPath ?? '' }})</div>
                            <pre class="mt-2 whitespace-pre-wrap break-words">{{ $cfgRaw ?? '' }}</pre>
                        </div>
                    @else
                        <div class="text-sm text-slate-200">
                            Настройки для этой игры пока не поддерживаются.
                        </div>
                    @endif
                </div>
            </div>
        </div>
    </section>
@endsection
