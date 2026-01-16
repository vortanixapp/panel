@php
    $ftpHost = (string) ($server->ftp_host ?? '');
    $ftpPort = (int) ($server->ftp_port ?? 21);
    $ftpUser = (string) ($server->ftp_username ?? '');
    $ftpRoot = (string) ($server->ftp_root ?? '');

    $ftpPass = '';
    try {
        if (! empty($server->ftp_password)) {
            $ftpPass = \Illuminate\Support\Facades\Crypt::decryptString($server->ftp_password);
        }
    } catch (\Throwable $e) {
        $ftpPass = '';
    }
@endphp

<div class="mt-6 rounded-2xl border border-white/10 bg-[#242f3d] text-slate-100 shadow-sm overflow-hidden">
    <div class="border-b border-white/10 bg-black/10 px-4 py-3 text-[11px] uppercase tracking-wide text-slate-300/70">
        FTP доступ
    </div>
    <div class="p-4">
        @if ($ftpHost === '' || $ftpUser === '' || $ftpPass === '')
            <div class="text-sm text-slate-200">
                FTP доступ ещё не настроен для этого сервера.
            </div>

            <div class="mt-4">
                <form method="POST" action="{{ route('server.ftp.reset-password', $server) }}">
                    @csrf
                    <button type="submit" class="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800">
                        Настроить FTP
                    </button>
                </form>
            </div>

        @else
            <div class="grid gap-3 text-sm">
                <div class="flex items-center justify-between border-b border-white/10 pb-2">
                    <span class="text-slate-300/70">Хост</span>
                    <span class="font-medium text-slate-100">{{ $ftpHost }}</span>
                </div>
                <div class="flex items-center justify-between border-b border-white/10 pb-2">
                    <span class="text-slate-300/70">Порт</span>
                    <span class="font-medium text-slate-100">{{ $ftpPort }}</span>
                </div>
                <div class="flex items-center justify-between border-b border-white/10 pb-2">
                    <span class="text-slate-300/70">Логин</span>
                    <span class="font-medium text-slate-100">{{ $ftpUser }}</span>
                </div>
                <div class="flex items-center justify-between border-b border-white/10 pb-2">
                    <span class="text-slate-300/70">Пароль</span>
                    <span class="font-medium text-slate-100">{{ $ftpPass }}</span>
                </div>
                <div class="flex items-center justify-between">
                    <span class="text-slate-300/70">Корневая папка</span>
                    <span class="font-medium text-slate-100">{{ $ftpRoot !== '' ? $ftpRoot : '-' }}</span>
                </div>
            </div>

            <div class="mt-4">
                <form method="POST" action="{{ route('server.ftp.reset-password', $server) }}" onsubmit="return confirm('Сменить FTP пароль? Текущий пароль перестанет работать.');">
                    @csrf
                    <button type="submit" class="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800">
                        Сменить пароль
                    </button>
                </form>
            </div>

            <div class="mt-6 rounded-2xl border border-white/10 bg-[#242f3d] text-slate-100 shadow-sm overflow-hidden">
                <div class="border-b border-white/10 bg-black/10 px-4 py-3 text-[11px] uppercase tracking-wide text-slate-300/70">
                    Настройка FASTDL
                </div>
                <div class="p-4">
                    <div class="text-sm text-slate-200 mb-4">
                        FastDL позволяет клиентам скачивать ресурсы сервера напрямую с HTTP, без FTP. Это ускоряет загрузку карт, моделей и звуков.
                    </div>
                    <div class="flex flex-wrap items-center gap-3">
                        <button type="button" id="vtxFastdlConfigure" class="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800">
                            Настроить
                        </button>
                        <button type="button" id="vtxFastdlUpdate" class="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800">
                            Обновить
                        </button>
                    </div>
                    <div class="mt-4 rounded-md bg-black/10 border border-white/10 p-3 text-xs text-slate-200">
                        <strong>Настроить:</strong> Автоматически установит <code>sv_downloadurl</code> в server.cfg.<br>
                        <strong>Обновить:</strong> Синхронизирует файлы сервера в FastDL директорию.
                    </div>
                </div>
            </div>

            <div class="mt-4 rounded-md bg-black/10 border border-white/10 p-3 text-xs text-slate-200">
                Для подключения используй FileZilla / WinSCP:
                <br>
                Протокол: FTP
                <br>
                Шифрование: Без шифрования
                <br>
                Режим передачи: Пассивный
            </div>

            <div class="mt-6 rounded-xl border border-white/10 overflow-hidden">
                <div class="flex flex-wrap items-center justify-between gap-2 border-b border-white/10 bg-black/10 px-4 py-3">
                    <div class="text-[11px] uppercase tracking-wide text-slate-300/70">Файлы сервера</div>
                    <div class="flex flex-wrap items-center gap-2">
                        <button type="button" id="vtxFmUp" class="rounded-md border border-white/10 bg-black/10 px-2.5 py-1.5 text-xs text-slate-200 hover:bg-black/15">Вверх</button>
                        <button type="button" id="vtxFmRefresh" class="rounded-md border border-white/10 bg-black/10 px-2.5 py-1.5 text-xs text-slate-200 hover:bg-black/15">Обновить</button>
                        <button type="button" id="vtxFmMkdir" class="rounded-md border border-white/10 bg-black/10 px-2.5 py-1.5 text-xs text-slate-200 hover:bg-black/15">Новая папка</button>
                        <input type="file" id="vtxFmUploadInput" class="hidden" />
                        <button type="button" id="vtxFmUpload" class="rounded-md bg-slate-900 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-800">Загрузить файл</button>
                    </div>
                </div>

                <div class="px-4 py-3 text-xs text-slate-300/70" id="vtxFmPath">/</div>

                <div class="px-4 pb-4">
                    <div id="vtxFmError" class="hidden mb-3 rounded-md bg-rose-50 px-3 py-2 text-xs text-rose-700"></div>
                    <div class="overflow-x-auto rounded-lg border border-white/10">
                        <table class="min-w-full text-sm">
                            <thead class="bg-black/10 text-xs text-slate-300/70">
                                <tr>
                                    <th class="px-3 py-2 text-left font-medium">Имя</th>
                                    <th class="px-3 py-2 text-left font-medium">Тип</th>
                                    <th class="px-3 py-2 text-right font-medium">Размер</th>
                                    <th class="px-3 py-2 text-right font-medium">Действия</th>
                                </tr>
                            </thead>
                            <tbody id="vtxFmBody" class="divide-y divide-white/10"></tbody>
                        </table>
                    </div>
                </div>
            </div>

            <div id="vtxFmModal" class="hidden fixed inset-0 z-50">
                <div class="absolute inset-0 bg-slate-900/40"></div>
                <div class="relative mx-auto mt-10 w-[95%] max-w-4xl rounded-2xl border border-white/10 bg-[#242f3d] shadow-xl overflow-hidden">
                    <div class="flex items-center justify-between gap-2 border-b border-white/10 bg-black/10 px-4 py-3">
                        <div class="text-sm font-medium text-slate-100" id="vtxFmModalTitle">Файл</div>
                        <div class="flex items-center gap-2">
                            <button type="button" id="vtxFmModalSave" class="rounded-md bg-slate-900 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-800">Сохранить</button>
                            <button type="button" id="vtxFmModalClose" class="rounded-md border border-white/10 bg-black/10 px-3 py-1.5 text-xs text-slate-200 hover:bg-black/15">Закрыть</button>
                        </div>
                    </div>
                    <div class="p-4">
                        <textarea id="vtxFmModalTextarea" class="h-[60vh] w-full rounded-lg border border-white/10 bg-black/10 p-3 text-xs font-mono text-slate-100 focus:border-sky-500 focus:outline-none placeholder-slate-400"></textarea>
                    </div>
                </div>
            </div>

            <div id="vtxFastdlModal" class="hidden fixed inset-0 z-50">
                <div class="absolute inset-0 bg-slate-900/40"></div>
                <div class="relative mx-auto mt-10 w-[95%] max-w-md rounded-2xl border border-white/10 bg-[#242f3d] shadow-xl overflow-hidden">
                    <div class="border-b border-white/10 bg-black/10 px-4 py-3">
                        <div class="text-sm font-medium text-slate-100" id="vtxFastdlModalTitle">Подтверждение</div>
                    </div>
                    <div class="p-4">
                        <div class="text-sm text-slate-200 mb-4" id="vtxFastdlModalMessage">Вы уверены?</div>
                        <div class="flex items-center gap-3">
                            <button type="button" id="vtxFastdlModalConfirm" class="rounded-md bg-slate-900 px-3 py-2 text-xs font-medium text-white hover:bg-slate-800">Да</button>
                            <button type="button" id="vtxFastdlModalCancel" class="rounded-md border border-white/10 bg-black/10 px-3 py-2 text-xs text-slate-200 hover:bg-black/15">Отмена</button>
                        </div>
                    </div>
                </div>
            </div>

            <div id="vtxToast" class="fixed top-4 right-4 z-50 hidden">
                <div class="rounded-lg border border-emerald-500/30 bg-emerald-600 shadow-lg p-4 text-sm text-white">
                    <div id="vtxToastMessage">Уведомление</div>
                </div>
            </div>

            @push('scripts')
            <script>
                document.addEventListener('DOMContentLoaded', function () {
                    const csrf = () => {
                        const el = document.querySelector('meta[name="csrf-token"]');
                        return el ? el.getAttribute('content') : '';
                    };

                    const urls = {
                        list: "{{ route('server.files.list', $server) }}",
                        read: "{{ route('server.files.read', $server) }}",
                        write: "{{ route('server.files.write', $server) }}",
                        mkdir: "{{ route('server.files.mkdir', $server) }}",
                        del: "{{ route('server.files.delete', $server) }}",
                        download: "{{ route('server.files.download', $server) }}",
                        upload: "{{ route('server.files.upload', $server) }}",
                    };

                    const elPath = document.getElementById('vtxFmPath');
                    const elBody = document.getElementById('vtxFmBody');
                    const elError = document.getElementById('vtxFmError');
                    const btnUp = document.getElementById('vtxFmUp');
                    const btnRefresh = document.getElementById('vtxFmRefresh');
                    const btnMkdir = document.getElementById('vtxFmMkdir');
                    const btnUpload = document.getElementById('vtxFmUpload');
                    const uploadInput = document.getElementById('vtxFmUploadInput');
                    const modal = document.getElementById('vtxFmModal');
                    const modalTitle = document.getElementById('vtxFmModalTitle');
                    const modalTextarea = document.getElementById('vtxFmModalTextarea');
                    const modalClose = document.getElementById('vtxFmModalClose');
                    const modalSave = document.getElementById('vtxFmModalSave');

                    const fastdlModal = document.getElementById('vtxFastdlModal');
                    const fastdlModalTitle = document.getElementById('vtxFastdlModalTitle');
                    const fastdlModalMessage = document.getElementById('vtxFastdlModalMessage');
                    const fastdlModalConfirm = document.getElementById('vtxFastdlModalConfirm');
                    const fastdlModalCancel = document.getElementById('vtxFastdlModalCancel');

                    const toast = document.getElementById('vtxToast');
                    const toastMessage = document.getElementById('vtxToastMessage');

                    const btnFastdlConfigure = document.getElementById('vtxFastdlConfigure');
                    const btnFastdlUpdate = document.getElementById('vtxFastdlUpdate');

                    let currentPath = '';
                    let editingPath = '';
                    let fastdlAction = null;

                    if (!elBody || !btnRefresh) {
                        return;
                    }

                    function showError(msg) {
                        elError.textContent = msg;
                        elError.classList.remove('hidden');
                    }

                    function clearError() {
                        elError.textContent = '';
                        elError.classList.add('hidden');
                    }

                    async function postJson(url, payload) {
                        const resp = await fetch(url, {
                            method: 'POST',
                            headers: {
                                'Content-Type': 'application/json',
                                'Accept': 'application/json',
                                'X-CSRF-TOKEN': csrf(),
                            },
                            body: JSON.stringify(payload || {}),
                        });
                        const data = await resp.json().catch(() => null);
                        if (!resp.ok || !data || data.ok !== true) {
                            const err = data && data.error ? data.error : ('HTTP ' + resp.status);
                            throw new Error(err);
                        }
                        return data;
                    }

                    function joinPath(base, name) {
                        if (!base) return name;
                        return base.replace(/\/+$/g, '') + '/' + name.replace(/^\/+/, '');
                    }

                    function parentPath(p) {
                        const s = (p || '').replace(/\/+$/g, '');
                        if (!s) return '';
                        const idx = s.lastIndexOf('/');
                        if (idx < 0) return '';
                        return s.slice(0, idx);
                    }

                    function render(entries) {
                        elBody.innerHTML = '';
                        const rows = Array.isArray(entries) ? entries : [];
                        if (rows.length === 0) {
                            const tr = document.createElement('tr');
                            tr.innerHTML = '<td class="px-3 py-3 text-sm text-slate-500" colspan="4">Папка пуста</td>';
                            elBody.appendChild(tr);
                            return;
                        }

                        for (const it of rows) {
                            const name = String(it.name || '');
                            const type = String(it.type || 'file');
                            const size = Number(it.size || 0);

                            const tr = document.createElement('tr');
                            const nameTd = document.createElement('td');
                            nameTd.className = 'px-3 py-2 text-slate-100';

                            const link = document.createElement('button');
                            link.type = 'button';
                            link.className = 'text-left hover:underline';
                            link.textContent = name;
                            link.addEventListener('click', function () {
                                if (type === 'dir') {
                                    currentPath = joinPath(currentPath, name);
                                    load();
                                } else {
                                    openEditor(joinPath(currentPath, name));
                                }
                            });
                            nameTd.appendChild(link);

                            const typeTd = document.createElement('td');
                            typeTd.className = 'px-3 py-2 text-slate-500';
                            typeTd.textContent = type === 'dir' ? 'Папка' : 'Файл';

                            const sizeTd = document.createElement('td');
                            sizeTd.className = 'px-3 py-2 text-right text-slate-500';
                            sizeTd.textContent = type === 'dir' ? '-' : String(size);

                            const actTd = document.createElement('td');
                            actTd.className = 'px-3 py-2 text-right';

                            const dl = document.createElement('button');
                            dl.type = 'button';
                            dl.className = 'mr-2 text-xs text-slate-200 hover:underline';
                            dl.textContent = 'Скачать';
                            dl.disabled = type === 'dir';
                            dl.addEventListener('click', function () {
                                downloadFile(joinPath(currentPath, name));
                            });

                            const del = document.createElement('button');
                            del.type = 'button';
                            del.className = 'text-xs text-rose-700 hover:underline';
                            del.textContent = 'Удалить';
                            del.addEventListener('click', function () {
                                const p = joinPath(currentPath, name);
                                const rec = type === 'dir';
                                const ok = rec
                                    ? confirm('Удалить папку и всё содержимое?')
                                    : confirm('Удалить файл?');
                                if (!ok) return;
                                removePath(p, rec);
                            });

                            actTd.appendChild(dl);
                            actTd.appendChild(del);

                            tr.appendChild(nameTd);
                            tr.appendChild(typeTd);
                            tr.appendChild(sizeTd);
                            tr.appendChild(actTd);
                            elBody.appendChild(tr);
                        }
                    }

                    async function load() {
                        clearError();
                        elPath.textContent = '/' + (currentPath || '');
                        try {
                            const data = await postJson(urls.list, { path: currentPath });
                            render(data.entries || []);
                        } catch (e) {
                            showError(e && e.message ? e.message : String(e));
                        }
                    }

                    async function removePath(path, recursive) {
                        clearError();
                        try {
                            await postJson(urls.del, { path: path, recursive: !!recursive });
                            await load();
                        } catch (e) {
                            showError(e && e.message ? e.message : String(e));
                        }
                    }

                    async function downloadFile(path) {
                        clearError();
                        try {
                            const data = await postJson(urls.download, { path: path });
                            const b64 = String(data.content_b64 || '');
                            const bin = atob(b64);
                            const bytes = new Uint8Array(bin.length);
                            for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
                            const blob = new Blob([bytes]);
                            const a = document.createElement('a');
                            a.href = URL.createObjectURL(blob);
                            a.download = data.filename || 'file';
                            document.body.appendChild(a);
                            a.click();
                            a.remove();
                            setTimeout(() => URL.revokeObjectURL(a.href), 3000);
                        } catch (e) {
                            showError(e && e.message ? e.message : String(e));
                        }
                    }

                    async function openEditor(path) {
                        clearError();
                        try {
                            const data = await postJson(urls.read, { path: path });
                            editingPath = String(data.path || path);
                            modalTitle.textContent = editingPath;
                            modalTextarea.value = String(data.content || '');
                            modal.classList.remove('hidden');
                        } catch (e) {
                            showError(e && e.message ? e.message : String(e));
                        }
                    }

                    async function saveEditor() {
                        clearError();
                        try {
                            await postJson(urls.write, { path: editingPath, content: modalTextarea.value });
                            modal.classList.add('hidden');
                            editingPath = '';
                            await load();
                        } catch (e) {
                            showError(e && e.message ? e.message : String(e));
                        }
                    }

                    btnRefresh.addEventListener('click', load);
                    btnUp.addEventListener('click', function () {
                        currentPath = parentPath(currentPath);
                        load();
                    });

                    btnMkdir.addEventListener('click', async function () {
                        const name = prompt('Имя папки');
                        if (!name) return;
                        const p = joinPath(currentPath, name);
                        clearError();
                        try {
                            await postJson(urls.mkdir, { path: p });
                            await load();
                        } catch (e) {
                            showError(e && e.message ? e.message : String(e));
                        }
                    });

                    btnUpload.addEventListener('click', function () {
                        uploadInput.value = '';
                        uploadInput.click();
                    });

                    uploadInput.addEventListener('change', async function () {
                        const f = uploadInput.files && uploadInput.files[0];
                        if (!f) return;
                        clearError();
                        try {
                            const fd = new FormData();
                            fd.append('path', currentPath);
                            fd.append('file', f);
                            const resp = await fetch(urls.upload, {
                                method: 'POST',
                                headers: {
                                    'X-CSRF-TOKEN': csrf(),
                                    'Accept': 'application/json',
                                },
                                body: fd,
                            });
                            const data = await resp.json();
                            if (!resp.ok || !data || data.ok !== true) {
                                const err = data && data.error ? data.error : ('HTTP ' + resp.status);
                                throw new Error(err);
                            }
                            await load();
                        } catch (e) {
                            showError(e && e.message ? e.message : String(e));
                        }
                    });

                    modalClose.addEventListener('click', function () {
                        modal.classList.add('hidden');
                        editingPath = '';
                    });
                    modalSave.addEventListener('click', saveEditor);

                    function showToast(msg) {
                        toastMessage.textContent = msg;
                        toast.classList.remove('hidden');
                        setTimeout(() => toast.classList.add('hidden'), 3000);
                    }

                    function showFastdlModal(title, message, action) {
                        fastdlModalTitle.textContent = title;
                        fastdlModalMessage.textContent = message;
                        fastdlAction = action;
                        fastdlModal.classList.remove('hidden');
                    }

                    fastdlModalCancel.addEventListener('click', function () {
                        fastdlModal.classList.add('hidden');
                        fastdlAction = null;
                    });

                    fastdlModalConfirm.addEventListener('click', async function () {
                        if (fastdlAction) {
                            await fastdlAction();
                            fastdlAction = null;
                        }
                        fastdlModal.classList.add('hidden');
                    });

                    if (btnFastdlConfigure) {
                        btnFastdlConfigure.addEventListener('click', function () {
                            showFastdlModal('Настройка FastDL', 'Автоматически настроить sv_downloadurl в server.cfg?', async () => {
                                clearError();
                                try {
                                    await postJson("{{ route('server.fastdl.configure', $server) }}");
                                    showToast('FastDL настроен. Перезапустите сервер для применения.');
                                } catch (e) {
                                    showError(e && e.message ? e.message : String(e));
                                }
                            });
                        });
                    }

                    if (btnFastdlUpdate) {
                        btnFastdlUpdate.addEventListener('click', function () {
                            showFastdlModal('Обновление FastDL', 'Синхронизировать файлы в FastDL? Это может занять время.', async () => {
                                clearError();
                                try {
                                    await postJson("{{ route('server.fastdl.update', $server) }}");
                                    showToast('FastDL обновлён.');
                                } catch (e) {
                                    showError(e && e.message ? e.message : String(e));
                                }
                            });
                        });
                    }

                    load();
                });
            </script>
            @endpush
        @endif
    </div>
</div>
