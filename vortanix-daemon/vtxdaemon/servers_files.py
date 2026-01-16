import base64
from datetime import datetime
import os
import re
import shlex
import shutil
from typing import Tuple

from aiohttp import web

from vtxdaemon.cmd import run_cmd
from vtxdaemon.logbuf import log
from vtxdaemon.servers_common import FTP_GROUP, _ensure_server_data_dir, _ensure_server_disk_quota, _server_data_dir


async def _ensure_server_storage_dir(server_id: int) -> str:
    data_dir = await _ensure_server_data_dir(server_id)
    await _ensure_server_disk_quota(server_id, data_dir, None, allow_init=False)
    return data_dir


def _sanitize_rel_path(rel_path: str) -> str:
    rel_path = str(rel_path or '').replace('\\', '/').strip()
    rel_path = rel_path.lstrip('/')
    if rel_path in ('', '.'):
        return ''

    parts = []
    for p in rel_path.split('/'):
        if p in ('', '.'):
            continue
        if p == '..':
            raise ValueError('Invalid path')
        parts.append(p)

    return '/'.join(parts)


def _resolve_server_path(server_id: int, rel_path: str) -> Tuple[str, str, str]:
    root = os.path.realpath(_server_data_dir(server_id))
    rel = _sanitize_rel_path(rel_path)
    target = os.path.realpath(os.path.join(root, rel))
    if target != root and not target.startswith(root + os.sep):
        raise ValueError('Invalid path')
    return root, target, rel


async def server_files_list_handler(request):
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    server_id = data.get('server_id')
    if not isinstance(server_id, int):
        try:
            server_id = int(server_id)
        except Exception:
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    rel_path = str(data.get('path') or '')

    try:
        await _ensure_server_storage_dir(server_id)
        root, target, rel = _resolve_server_path(server_id, rel_path)
        if not os.path.isdir(target):
            return web.json_response({'ok': False, 'error': 'Not a directory'}, status=400)

        entries = []
        with os.scandir(target) as it:
            for entry in it:
                try:
                    st = entry.stat(follow_symlinks=False)
                    is_dir = entry.is_dir(follow_symlinks=False)
                    entries.append({
                        'name': entry.name,
                        'type': 'dir' if is_dir else 'file',
                        'size': int(st.st_size) if not is_dir else 0,
                        'mtime': datetime.fromtimestamp(st.st_mtime).isoformat(),
                    })
                except Exception:
                    continue

        entries.sort(key=lambda x: (0 if x.get('type') == 'dir' else 1, str(x.get('name') or '').lower()))
        return web.json_response({'ok': True, 'server_id': server_id, 'path': rel, 'entries': entries})
    except Exception as e:
        await log(f'Files list error: server_id={server_id} path={rel_path} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)


async def server_files_read_handler(request):
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    server_id = data.get('server_id')
    if not isinstance(server_id, int):
        try:
            server_id = int(server_id)
        except Exception:
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    rel_path = str(data.get('path') or '')

    try:
        await _ensure_server_storage_dir(server_id)
        root, target, rel = _resolve_server_path(server_id, rel_path)
        if not os.path.isfile(target):
            return web.json_response({'ok': False, 'error': 'Not a file'}, status=400)

        size = int(os.path.getsize(target))
        if size > 1024 * 1024:
            return web.json_response({'ok': False, 'error': 'File too large'}, status=400)

        with open(target, 'rb') as f:
            raw = f.read()

        try:
            content = raw.decode('utf-8')
        except Exception:
            return web.json_response({'ok': False, 'error': 'File is not UTF-8 text'}, status=400)

        return web.json_response({'ok': True, 'server_id': server_id, 'path': rel, 'content': content, 'size': size})
    except Exception as e:
        await log(f'Files read error: server_id={server_id} path={rel_path} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)


async def server_files_download_handler(request):
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    server_id = data.get('server_id')
    if not isinstance(server_id, int):
        try:
            server_id = int(server_id)
        except Exception:
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    rel_path = str(data.get('path') or '')

    try:
        await _ensure_server_storage_dir(server_id)
        root, target, rel = _resolve_server_path(server_id, rel_path)
        if not os.path.isfile(target):
            return web.json_response({'ok': False, 'error': 'Not a file'}, status=400)

        size = int(os.path.getsize(target))
        if size > 20 * 1024 * 1024:
            return web.json_response({'ok': False, 'error': 'File too large'}, status=400)

        with open(target, 'rb') as f:
            raw = f.read()

        return web.json_response({
            'ok': True,
            'server_id': server_id,
            'path': rel,
            'filename': os.path.basename(target),
            'content_b64': base64.b64encode(raw).decode('ascii'),
            'size': size,
        })
    except Exception as e:
        await log(f'Files download error: server_id={server_id} path={rel_path} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)


async def server_files_write_handler(request):
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    server_id = data.get('server_id')
    if not isinstance(server_id, int):
        try:
            server_id = int(server_id)
        except Exception:
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    rel_path = str(data.get('path') or '')
    content = data.get('content')
    if not isinstance(content, str):
        return web.json_response({'ok': False, 'error': 'content must be string'}, status=400)

    try:
        await _ensure_server_storage_dir(server_id)
        root, target, rel = _resolve_server_path(server_id, rel_path)
        if rel == '':
            return web.json_response({'ok': False, 'error': 'Invalid file path'}, status=400)

        raw = content.encode('utf-8')
        if len(raw) > 1024 * 1024:
            return web.json_response({'ok': False, 'error': 'File too large'}, status=400)

        os.makedirs(os.path.dirname(target), exist_ok=True)
        with open(target, 'wb') as f:
            f.write(raw)

        q_target = shlex.quote(target)
        ftp_user = f'srv{server_id}'
        await run_cmd(f'chown {shlex.quote(ftp_user)}:{shlex.quote(FTP_GROUP)} {q_target} >/dev/null 2>&1 || true')
        await run_cmd(f'chmod 664 {q_target} >/dev/null 2>&1 || true')

        return web.json_response({'ok': True, 'server_id': server_id, 'path': rel})
    except Exception as e:
        await log(f'Files write error: server_id={server_id} path={rel_path} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)


async def server_files_delete_handler(request):
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    server_id = data.get('server_id')
    if not isinstance(server_id, int):
        try:
            server_id = int(server_id)
        except Exception:
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    rel_path = str(data.get('path') or '')
    recursive = bool(data.get('recursive') or False)

    try:
        await _ensure_server_storage_dir(server_id)
        root, target, rel = _resolve_server_path(server_id, rel_path)
        if rel == '':
            return web.json_response({'ok': False, 'error': 'Cannot delete root'}, status=400)

        if os.path.isdir(target) and not os.path.islink(target):
            if recursive:
                shutil.rmtree(target)
            else:
                os.rmdir(target)
        else:
            os.remove(target)

        return web.json_response({'ok': True, 'server_id': server_id, 'path': rel})
    except Exception as e:
        await log(f'Files delete error: server_id={server_id} path={rel_path} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)


async def server_files_mkdir_handler(request):
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    server_id = data.get('server_id')
    if not isinstance(server_id, int):
        try:
            server_id = int(server_id)
        except Exception:
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    rel_path = str(data.get('path') or '')

    try:
        await _ensure_server_storage_dir(server_id)
        root, target, rel = _resolve_server_path(server_id, rel_path)
        if rel == '':
            return web.json_response({'ok': False, 'error': 'Invalid directory path'}, status=400)

        os.makedirs(target, exist_ok=True)
        q_target = shlex.quote(target)
        ftp_user = f'srv{server_id}'
        await run_cmd(f'chown {shlex.quote(ftp_user)}:{shlex.quote(FTP_GROUP)} {q_target} >/dev/null 2>&1 || true')
        await run_cmd(f'chmod 2775 {q_target} >/dev/null 2>&1 || true')
        return web.json_response({'ok': True, 'server_id': server_id, 'path': rel})
    except Exception as e:
        await log(f'Files mkdir error: server_id={server_id} path={rel_path} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)


async def server_files_upload_handler(request):
    try:
        reader = await request.multipart()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid multipart: {e}'}, status=400)

    server_id = None
    rel_dir = ''
    file_part = None
    filename = ''

    try:
        while True:
            part = await reader.next()
            if part is None:
                break
            if part.name == 'server_id':
                server_id_raw = (await part.text()).strip()
                server_id = int(server_id_raw)
            elif part.name == 'path':
                rel_dir = (await part.text()).strip()
            elif part.name == 'file':
                file_part = part
                filename = str(part.filename or '').strip()

        if server_id is None:
            return web.json_response({'ok': False, 'error': 'server_id is required'}, status=400)
        if file_part is None:
            return web.json_response({'ok': False, 'error': 'file is required'}, status=400)
        if filename == '' or '/' in filename or '\\' in filename:
            return web.json_response({'ok': False, 'error': 'Invalid filename'}, status=400)

        await _ensure_server_storage_dir(server_id)
        root, dir_target, rel = _resolve_server_path(server_id, rel_dir)
        if not os.path.isdir(dir_target):
            os.makedirs(dir_target, exist_ok=True)

        file_target = os.path.realpath(os.path.join(dir_target, filename))
        root_real = os.path.realpath(root)
        if file_target != root_real and not file_target.startswith(root_real + os.sep):
            return web.json_response({'ok': False, 'error': 'Invalid path'}, status=400)

        total = 0
        with open(file_target, 'wb') as f:
            while True:
                chunk = await file_part.read_chunk(size=1024 * 256)
                if not chunk:
                    break
                total += len(chunk)
                if total > 50 * 1024 * 1024:
                    return web.json_response({'ok': False, 'error': 'File too large'}, status=400)
                f.write(chunk)

        q_target = shlex.quote(file_target)
        ftp_user = f'srv{server_id}'
        await run_cmd(f'chown {shlex.quote(ftp_user)}:{shlex.quote(FTP_GROUP)} {q_target} >/dev/null 2>&1 || true')
        await run_cmd(f'chmod 664 {q_target} >/dev/null 2>&1 || true')

        rel_file = (rel + '/' if rel != '' else '') + filename
        return web.json_response({'ok': True, 'server_id': server_id, 'path': rel_file, 'size': total})
    except Exception as e:
        await log(f'Files upload error: server_id={server_id} path={rel_dir} filename={filename} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)
