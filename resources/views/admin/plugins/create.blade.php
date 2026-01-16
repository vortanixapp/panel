@extends('layouts.app-admin')

@section('page_title', 'Новый плагин')

@section('content')
    <section class="px-4 py-6 md:py-8">
        <div class="w-full max-w-none space-y-6">
            <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
                <div>
                    <h1 class="text-xl md:text-2xl font-semibold tracking-tight text-slate-100">Новый плагин</h1>
                    <p class="mt-1 text-sm text-slate-300/80">Добавьте плагин и загрузите архив (zip/tar/tar.gz).</p>
                </div>
                <a
                    href="{{ route('admin.plugins.index') }}"
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

                <form method="POST" action="{{ route('admin.plugins.store') }}" class="space-y-4" enctype="multipart/form-data">
                    @csrf

                    <div class="grid gap-4 md:grid-cols-2">
                        <div class="space-y-1">
                            <label for="name" class="text-xs font-medium text-slate-200">Название</label>
                            <input id="name" name="name" type="text" value="{{ old('name') }}" required class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                        </div>

                        <div class="space-y-1">
                            <label for="category" class="text-xs font-medium text-slate-200">Категория</label>
                            <input id="category" name="category" type="text" value="{{ old('category') }}" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500" placeholder="metamod / rehlds / admin / fun">
                        </div>

                        <div class="space-y-1">
                            <label for="slug" class="text-xs font-medium text-slate-200">Slug</label>
                            <input id="slug" name="slug" type="text" value="{{ old('slug') }}" required class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                        </div>

                        <div class="space-y-1">
                            <label for="version" class="text-xs font-medium text-slate-200">Версия</label>
                            <input id="version" name="version" type="text" value="{{ old('version') }}" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                        </div>

                        <div class="space-y-1">
                            <label for="archive_type" class="text-xs font-medium text-slate-200">Тип архива</label>
                            <select id="archive_type" name="archive_type" required class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                                @php $t = old('archive_type', 'zip'); @endphp
                                <option value="zip" @if($t==='zip') selected @endif>zip</option>
                                <option value="tar" @if($t==='tar') selected @endif>tar</option>
                                <option value="targz" @if($t==='targz') selected @endif>tar.gz</option>
                            </select>
                        </div>

                        <div class="space-y-1 md:col-span-2">
                            <label for="archive" class="text-xs font-medium text-slate-200">Архив</label>
                            <input id="archive" name="archive" type="file" class="block w-full text-xs text-slate-200">
                        </div>

                        <div class="space-y-1 md:col-span-2">
                            <label for="install_path" class="text-xs font-medium text-slate-200">Путь установки (относительно data_dir)</label>
                            <input id="install_path" name="install_path" type="text" value="{{ old('install_path', 'cstrike') }}" required class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500">
                        </div>

                        <div class="space-y-2 md:col-span-2">
                            <div class="text-xs font-medium text-slate-200">Поддерживаемые игры</div>
                            <label class="inline-flex items-center gap-2 text-xs text-slate-200">
                                <input type="checkbox" name="all_games" value="1" {{ old('all_games', true) ? 'checked' : '' }}>
                                <span>Все игры</span>
                            </label>

                            <div class="grid gap-2 md:grid-cols-3">
                                @foreach(($games ?? []) as $g)
                                    @php $code = strtolower((string) ($g->code ?? $g->slug ?? '')); @endphp
                                    @if($code !== '')
                                        <label class="inline-flex items-center gap-2 text-xs text-slate-200">
                                            <input type="checkbox" name="supported_games_codes[]" value="{{ $code }}" {{ in_array($code, old('supported_games_codes', []), true) ? 'checked' : '' }}>
                                            <span>{{ $g->name }} ({{ $code }})</span>
                                        </label>
                                    @endif
                                @endforeach
                            </div>
                            <div class="text-[11px] text-slate-300/70">Если включено «Все игры», список ниже игнорируется.</div>
                        </div>

                        <div class="space-y-2 md:col-span-2">
                            <div class="flex items-center justify-between">
                                <div class="text-xs font-medium text-slate-200">Файлы и строки</div>
                                <button type="button" id="addFileAction" class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-semibold text-slate-200 hover:bg-black/15">Добавить файл</button>
                            </div>
                            <div class="text-[11px] text-slate-300/70">Пути указывай относительно data_dir (например: cstrike/server.cfg). Каждая строка — с новой строки.</div>

                            <div id="fileActionsWrap" class="space-y-3">
                                <div class="rounded-xl border border-white/10 bg-black/10 p-3" data-file-action>
                                    <div class="mb-2 flex justify-end">
                                        <button type="button" class="removeFileAction inline-flex items-center rounded-md border border-rose-500/20 bg-rose-500/10 px-3 py-1.5 text-[11px] font-semibold text-rose-200 hover:bg-rose-500/15">Удалить</button>
                                    </div>
                                    <div class="grid gap-3 md:grid-cols-2">
                                        <div class="space-y-1">
                                            <label class="text-[11px] text-slate-300/80">Действие</label>
                                            @php $act0 = old('file_action_action.0', 'ensure_contains'); @endphp
                                            <select name="file_action_action[]" class="fileActionAction block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm">
                                                <option value="ensure_contains" @if($act0==='ensure_contains') selected @endif>Добавить строки, если нет</option>
                                                <option value="append_lines" @if($act0==='append_lines') selected @endif>Дополнить в конец</option>
                                                <option value="prepend_lines" @if($act0==='prepend_lines') selected @endif>Вставить в начало</option>
                                                <option value="remove_lines" @if($act0==='remove_lines') selected @endif>Удалить строки</option>
                                                <option value="write_file" @if($act0==='write_file') selected @endif>Перезаписать файл</option>
                                                <option value="replace_regex" @if($act0==='replace_regex') selected @endif>Заменить по regex</option>
                                            </select>
                                        </div>

                                        <div class="space-y-1">
                                            <label class="text-[11px] text-slate-300/80">Путь к файлу</label>
                                            <input name="file_action_path[]" type="text" value="{{ old('file_action_path.0', '') }}" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="cstrike/server.cfg">
                                        </div>

                                        <div class="space-y-1 fileActionRegexFields md:col-span-2">
                                            <label class="text-[11px] text-slate-300/80">Regex pattern</label>
                                            <input name="file_action_pattern[]" type="text" value="{{ old('file_action_pattern.0', '') }}" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="^sv_password.*$">
                                        </div>
                                        <div class="space-y-1 fileActionRegexFields md:col-span-2">
                                            <label class="text-[11px] text-slate-300/80">Regex replacement</label>
                                            <input name="file_action_replacement[]" type="text" value="{{ old('file_action_replacement.0', '') }}" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="sv_password \"123\"">
                                        </div>

                                        <div class="flex items-center gap-2 text-xs text-slate-200 fileActionCreateWrap md:col-span-2">
                                            <input type="hidden" name="file_action_create_if_missing[]" value="0">
                                            <input type="checkbox" name="file_action_create_if_missing[]" value="1" {{ old('file_action_create_if_missing.0', '1') ? 'checked' : '' }}>
                                            <span>Создать файл, если отсутствует</span>
                                        </div>

                                        <div class="space-y-1 md:col-span-2">
                                            <label class="text-[11px] text-slate-300/80">Строки</label>
                                            <textarea name="file_action_lines[]" rows="5" class="fileActionLines block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="sv_allowdownload \"1\"">{{ old('file_action_lines.0', '') }}</textarea>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="space-y-2 md:col-span-2">
                            <div class="flex items-center justify-between">
                                <div class="text-xs font-medium text-slate-200">Действия при удалении (uninstall)</div>
                                <button type="button" id="addUninstallAction" class="inline-flex items-center rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-[11px] font-semibold text-slate-200 hover:bg-black/15">Добавить файл</button>
                            </div>
                            <div class="text-[11px] text-slate-300/70">Эти действия выполняются при удалении плагина (до удаления файлов). Путь относительно data_dir.</div>

                            <div id="uninstallActionsWrap" class="space-y-3">
                                <div class="rounded-xl border border-white/10 bg-black/10 p-3" data-uninstall-action>
                                    <div class="mb-2 flex justify-end">
                                        <button type="button" class="removeUninstallAction inline-flex items-center rounded-md border border-rose-500/20 bg-rose-500/10 px-3 py-1.5 text-[11px] font-semibold text-rose-200 hover:bg-rose-500/15">Удалить</button>
                                    </div>
                                    <div class="grid gap-3 md:grid-cols-2">
                                        <div class="space-y-1">
                                            <label class="text-[11px] text-slate-300/80">Действие</label>
                                            @php $uact0 = old('uninstall_action_action.0', 'ensure_contains'); @endphp
                                            <select name="uninstall_action_action[]" class="uninstallActionAction block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm">
                                                <option value="ensure_contains" @if($uact0==='ensure_contains') selected @endif>Добавить строки, если нет</option>
                                                <option value="append_lines" @if($uact0==='append_lines') selected @endif>Дополнить в конец</option>
                                                <option value="prepend_lines" @if($uact0==='prepend_lines') selected @endif>Вставить в начало</option>
                                                <option value="remove_lines" @if($uact0==='remove_lines') selected @endif>Удалить строки</option>
                                                <option value="write_file" @if($uact0==='write_file') selected @endif>Перезаписать файл</option>
                                                <option value="replace_regex" @if($uact0==='replace_regex') selected @endif>Заменить по regex</option>
                                            </select>
                                        </div>

                                        <div class="space-y-1">
                                            <label class="text-[11px] text-slate-300/80">Путь к файлу</label>
                                            <input name="uninstall_action_path[]" type="text" value="{{ old('uninstall_action_path.0', '') }}" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="cstrike/liblist.gam">
                                        </div>

                                        <div class="space-y-1 uninstallActionRegexFields md:col-span-2">
                                            <label class="text-[11px] text-slate-300/80">Regex pattern</label>
                                            <input name="uninstall_action_pattern[]" type="text" value="{{ old('uninstall_action_pattern.0', '') }}" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="^gamedll_linux.*$">
                                        </div>
                                        <div class="space-y-1 uninstallActionRegexFields md:col-span-2">
                                            <label class="text-[11px] text-slate-300/80">Regex replacement</label>
                                            <input name="uninstall_action_replacement[]" type="text" value="{{ old('uninstall_action_replacement.0', '') }}" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="gamedll_linux \"dlls/cs_i386.so\"">
                                        </div>

                                        <div class="flex items-center gap-2 text-xs text-slate-200 uninstallActionCreateWrap md:col-span-2">
                                            <input type="hidden" name="uninstall_action_create_if_missing[]" value="0">
                                            <input type="checkbox" name="uninstall_action_create_if_missing[]" value="1" {{ old('uninstall_action_create_if_missing.0', '1') ? 'checked' : '' }}>
                                            <span>Создать файл, если отсутствует</span>
                                        </div>

                                        <div class="space-y-1 md:col-span-2">
                                            <label class="text-[11px] text-slate-300/80">Строки</label>
                                            <textarea name="uninstall_action_lines[]" rows="5" class="uninstallActionLines block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="gamedll_linux \"dlls/cs_i386.so\"">{{ old('uninstall_action_lines.0', '') }}</textarea>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="flex items-center gap-4 md:col-span-2">
                            <label class="inline-flex items-center gap-2 text-xs text-slate-200">
                                <input type="checkbox" name="restart_required" value="1" {{ old('restart_required') ? 'checked' : '' }}>
                                <span>Требуется рестарт</span>
                            </label>
                            <label class="inline-flex items-center gap-2 text-xs text-slate-200">
                                <input type="checkbox" name="active" value="1" {{ old('active', true) ? 'checked' : '' }}>
                                <span>Активен</span>
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

    <script>
        document.addEventListener('DOMContentLoaded', function () {
            const wrap = document.getElementById('fileActionsWrap');
            const btn = document.getElementById('addFileAction');
            if (!wrap || !btn) return;

            function updateRemoveButtons() {
                const blocks = wrap.querySelectorAll('[data-file-action]');
                const show = blocks.length > 1;
                blocks.forEach((b) => {
                    const rm = b.querySelector('.removeFileAction');
                    if (!rm) return;
                    rm.style.display = show ? '' : 'none';
                });
            }

            function updateBlockFields(block) {
                const sel = block.querySelector('.fileActionAction');
                if (!sel) return;
                const action = String(sel.value || '').toLowerCase();
                const regexFields = block.querySelectorAll('.fileActionRegexFields');
                regexFields.forEach((el) => {
                    el.style.display = action === 'replace_regex' ? '' : 'none';
                });
            }

            function updateAllBlocksFields() {
                wrap.querySelectorAll('[data-file-action]').forEach((b) => updateBlockFields(b));
            }

            function removeBlock(block) {
                const blocks = wrap.querySelectorAll('[data-file-action]');
                if (blocks.length <= 1) {
                    const input = block.querySelector('input[name="file_action_path[]"]');
                    const ta = block.querySelector('textarea[name="file_action_lines[]"]');
                    if (input) input.value = '';
                    if (ta) ta.value = '';
                    updateRemoveButtons();
                    return;
                }
                block.remove();
                updateRemoveButtons();
            }

            wrap.addEventListener('click', function (e) {
                const target = e.target;
                if (!(target instanceof HTMLElement)) return;
                if (!target.classList.contains('removeFileAction')) return;
                const block = target.closest('[data-file-action]');
                if (!block) return;
                removeBlock(block);
            });

            btn.addEventListener('click', function () {
                const block = document.createElement('div');
                block.className = 'rounded-xl border border-white/10 bg-black/10 p-3';
                block.setAttribute('data-file-action', '');
                block.innerHTML = `
                    <div class="mb-2 flex justify-end">
                        <button type="button" class="removeFileAction inline-flex items-center rounded-md border border-rose-500/20 bg-rose-500/10 px-3 py-1.5 text-[11px] font-semibold text-rose-200 hover:bg-rose-500/15">Удалить</button>
                    </div>
                    <div class="grid gap-3 md:grid-cols-2">
                        <div class="space-y-1">
                            <label class="text-[11px] text-slate-300/80">Действие</label>
                            <select name="file_action_action[]" class="fileActionAction block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm">
                                <option value="ensure_contains" selected>Добавить строки, если нет</option>
                                <option value="append_lines">Дополнить в конец</option>
                                <option value="prepend_lines">Вставить в начало</option>
                                <option value="remove_lines">Удалить строки</option>
                                <option value="write_file">Перезаписать файл</option>
                                <option value="replace_regex">Заменить по regex</option>
                            </select>
                        </div>
                        <div class="space-y-1">
                            <label class="text-[11px] text-slate-300/80">Путь к файлу</label>
                            <input name="file_action_path[]" type="text" value="" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="cstrike/server.cfg">
                        </div>
                        <div class="space-y-1 fileActionRegexFields md:col-span-2">
                            <label class="text-[11px] text-slate-300/80">Regex pattern</label>
                            <input name="file_action_pattern[]" type="text" value="" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="^sv_password.*$">
                        </div>
                        <div class="space-y-1 fileActionRegexFields md:col-span-2">
                            <label class="text-[11px] text-slate-300/80">Regex replacement</label>
                            <input name="file_action_replacement[]" type="text" value="" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="sv_password \\\"123\\\"">
                        </div>
                        <div class="flex items-center gap-2 text-xs text-slate-200 fileActionCreateWrap md:col-span-2">
                            <input type="hidden" name="file_action_create_if_missing[]" value="0">
                            <input type="checkbox" name="file_action_create_if_missing[]" value="1" checked>
                            <span>Создать файл, если отсутствует</span>
                        </div>
                        <div class="space-y-1 md:col-span-2">
                            <label class="text-[11px] text-slate-300/80">Строки</label>
                            <textarea name="file_action_lines[]" rows="5" class="fileActionLines block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="sv_allowdownload \"1\""></textarea>
                        </div>
                    </div>
                `;
                wrap.appendChild(block);
                updateRemoveButtons();
                updateAllBlocksFields();
            });

            updateRemoveButtons();
            updateAllBlocksFields();

            wrap.addEventListener('change', function (e) {
                const target = e.target;
                if (!(target instanceof HTMLElement)) return;
                if (!target.classList.contains('fileActionAction')) return;
                const block = target.closest('[data-file-action]');
                if (!block) return;
                updateBlockFields(block);
            });

            const uwrap = document.getElementById('uninstallActionsWrap');
            const ubtn = document.getElementById('addUninstallAction');
            if (!uwrap || !ubtn) return;

            function updateUninstallRemoveButtons() {
                const blocks = uwrap.querySelectorAll('[data-uninstall-action]');
                const show = blocks.length > 1;
                blocks.forEach((b) => {
                    const rm = b.querySelector('.removeUninstallAction');
                    if (!rm) return;
                    rm.style.display = show ? '' : 'none';
                });
            }

            function updateUninstallBlockFields(block) {
                const sel = block.querySelector('.uninstallActionAction');
                if (!sel) return;
                const action = String(sel.value || '').toLowerCase();
                const regexFields = block.querySelectorAll('.uninstallActionRegexFields');
                regexFields.forEach((el) => {
                    el.style.display = action === 'replace_regex' ? '' : 'none';
                });
            }

            function updateAllUninstallBlocksFields() {
                uwrap.querySelectorAll('[data-uninstall-action]').forEach((b) => updateUninstallBlockFields(b));
            }

            function removeUninstallBlock(block) {
                const blocks = uwrap.querySelectorAll('[data-uninstall-action]');
                if (blocks.length <= 1) {
                    const input = block.querySelector('input[name="uninstall_action_path[]"]');
                    const ta = block.querySelector('textarea[name="uninstall_action_lines[]"]');
                    if (input) input.value = '';
                    if (ta) ta.value = '';
                    updateUninstallRemoveButtons();
                    return;
                }
                block.remove();
                updateUninstallRemoveButtons();
            }

            uwrap.addEventListener('click', function (e) {
                const target = e.target;
                if (!(target instanceof HTMLElement)) return;
                if (!target.classList.contains('removeUninstallAction')) return;
                const block = target.closest('[data-uninstall-action]');
                if (!block) return;
                removeUninstallBlock(block);
            });

            ubtn.addEventListener('click', function () {
                const block = document.createElement('div');
                block.className = 'rounded-xl border border-white/10 bg-black/10 p-3';
                block.setAttribute('data-uninstall-action', '');
                block.innerHTML = `
                    <div class="mb-2 flex justify-end">
                        <button type="button" class="removeUninstallAction inline-flex items-center rounded-md border border-rose-500/20 bg-rose-500/10 px-3 py-1.5 text-[11px] font-semibold text-rose-200 hover:bg-rose-500/15">Удалить</button>
                    </div>
                    <div class="grid gap-3 md:grid-cols-2">
                        <div class="space-y-1">
                            <label class="text-[11px] text-slate-300/80">Действие</label>
                            <select name="uninstall_action_action[]" class="uninstallActionAction block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm">
                                <option value="ensure_contains" selected>Добавить строки, если нет</option>
                                <option value="append_lines">Дополнить в конец</option>
                                <option value="prepend_lines">Вставить в начало</option>
                                <option value="remove_lines">Удалить строки</option>
                                <option value="write_file">Перезаписать файл</option>
                                <option value="replace_regex">Заменить по regex</option>
                            </select>
                        </div>
                        <div class="space-y-1">
                            <label class="text-[11px] text-slate-300/80">Путь к файлу</label>
                            <input name="uninstall_action_path[]" type="text" value="" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="cstrike/liblist.gam">
                        </div>
                        <div class="space-y-1 uninstallActionRegexFields md:col-span-2">
                            <label class="text-[11px] text-slate-300/80">Regex pattern</label>
                            <input name="uninstall_action_pattern[]" type="text" value="" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="^gamedll_linux.*$">
                        </div>
                        <div class="space-y-1 uninstallActionRegexFields md:col-span-2">
                            <label class="text-[11px] text-slate-300/80">Regex replacement</label>
                            <input name="uninstall_action_replacement[]" type="text" value="" class="block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm" placeholder="gamedll_linux \\\"dlls/cs_i386.so\\\"">
                        </div>
                        <div class="flex items-center gap-2 text-xs text-slate-200 uninstallActionCreateWrap md:col-span-2">
                            <input type="hidden" name="uninstall_action_create_if_missing[]" value="0">
                            <input type="checkbox" name="uninstall_action_create_if_missing[]" value="1" checked>
                            <span>Создать файл, если отсутствует</span>
                        </div>
                        <div class="space-y-1 md:col-span-2">
                            <label class="text-[11px] text-slate-300/80">Строки</label>
                            <textarea name="uninstall_action_lines[]" rows="5" class="uninstallActionLines block w-full rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-100 shadow-sm"></textarea>
                        </div>
                    </div>
                `;
                uwrap.appendChild(block);
                updateUninstallRemoveButtons();
                updateAllUninstallBlocksFields();
            });

            uwrap.addEventListener('change', function (e) {
                const target = e.target;
                if (!(target instanceof HTMLElement)) return;
                if (!target.classList.contains('uninstallActionAction')) return;
                const block = target.closest('[data-uninstall-action]');
                if (!block) return;
                updateUninstallBlockFields(block);
            });

            updateUninstallRemoveButtons();
            updateAllUninstallBlocksFields();
        });
    </script>
@endsection
