import os
import re
import shlex
import shutil

from aiohttp import web

from vtxdaemon.cmd import run_cmd
from vtxdaemon.logbuf import log
from vtxdaemon.servers_common import _ensure_server_data_dir, _ensure_server_disk_quota, _server_data_dir
from vtxdaemon.servers_fs import _post_archive_fix_permissions


def _sanitize_rel_path(rel_path: str) -> str:
    rel_path = str(rel_path or '').replace('\\', '/').strip()
    rel_path = rel_path.lstrip('/')
    if rel_path in ('', '.'):
        return ''

    parts: list[str] = []
    for p in rel_path.split('/'):
        if p in ('', '.'):
            continue
        if p == '..':
            raise ValueError('Invalid path')
        parts.append(p)

    return '/'.join(parts)


def _resolve_install_path(server_id: int, install_path: str) -> tuple[str, str, str]:
    root = os.path.realpath(_server_data_dir(server_id))
    rel = _sanitize_rel_path(install_path)
    if rel == '':
        raise ValueError('install_path is required')
    target = os.path.realpath(os.path.join(root, rel))
    if target != root and not target.startswith(root + os.sep):
        raise ValueError('Invalid install_path')
    return root, target, rel


def _resolve_data_path(server_id: int, rel_path: str) -> tuple[str, str, str]:
    root = os.path.realpath(_server_data_dir(server_id))
    rel = _sanitize_rel_path(rel_path)
    if rel == '':
        raise ValueError('path is required')
    target = os.path.realpath(os.path.join(root, rel))
    if target != root and not target.startswith(root + os.sep):
        raise ValueError('Invalid path')
    return root, target, rel


def _read_text(path: str) -> str:
    with open(path, 'r', encoding='utf-8', errors='ignore') as f:
        return f.read()


def _write_text(path: str, content: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content)


def _ensure_trailing_newline(s: str) -> str:
    return s if s.endswith('\n') else (s + '\n')


def _apply_file_actions(server_id: int, file_actions: list[dict]) -> None:
    for a in file_actions:
        if not isinstance(a, dict):
            raise ValueError('file_actions must be array of objects')

        path = str(a.get('path') or '')
        action = str(a.get('action') or '').strip().lower()
        create_if_missing = bool(a.get('create_if_missing', True))

        if action not in ('append_lines', 'prepend_lines', 'ensure_contains', 'replace_regex', 'write_file'):
            raise ValueError(f'Unsupported file action: {action}')

        _, target, rel = _resolve_data_path(server_id, path)

        exists = os.path.exists(target)
        if not exists and not create_if_missing and action != 'write_file':
            continue

        if action == 'write_file':
            content = a.get('content')
            if content is None:
                lines = a.get('lines')
                if isinstance(lines, list):
                    content = '\n'.join([str(x) for x in lines])
                else:
                    content = ''
            _write_text(target, str(content))
            continue

        if not exists:
            _write_text(target, '')
            exists = True

        raw = _read_text(target)

        if action in ('append_lines', 'prepend_lines'):
            lines = a.get('lines')
            if not isinstance(lines, list):
                raise ValueError(f'{action}.lines must be array')
            block = '\n'.join([str(x) for x in lines])
            block = _ensure_trailing_newline(block)
            if action == 'append_lines':
                out = _ensure_trailing_newline(raw) + block if raw.strip() != '' else block
            else:
                out = block + raw
            _write_text(target, out)
            continue

        if action == 'ensure_contains':
            line = a.get('line')
            lines = a.get('lines')
            want: list[str] = []
            if isinstance(lines, list):
                want.extend([str(x) for x in lines])
            if isinstance(line, str) and line.strip() != '':
                want.append(line)
            if len(want) == 0:
                raise ValueError('ensure_contains requires line or lines')

            current_lines = raw.splitlines()
            current_set = set([l.rstrip('\r\n') for l in current_lines])
            changed = False
            for w in want:
                if w not in current_set:
                    current_lines.append(w)
                    current_set.add(w)
                    changed = True
            if changed:
                _write_text(target, _ensure_trailing_newline('\n'.join(current_lines)))
            continue

        if action == 'replace_regex':
            pattern = str(a.get('pattern') or '')
            replacement = str(a.get('replacement') or '')
            if pattern.strip() == '':
                raise ValueError('replace_regex requires pattern')
            out = re.sub(pattern, replacement, raw, flags=re.MULTILINE)
            _write_text(target, out)
            continue


async def server_plugins_apply_handler(request):
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

    game = str(data.get('game') or '').strip()
    action = str(data.get('action') or '').strip().lower()
    install_path = str(data.get('install_path') or '').strip()

    if action not in ('install', 'uninstall'):
        return web.json_response({'ok': False, 'error': 'action must be install|uninstall'}, status=400)

    try:
        data_dir = await _ensure_server_data_dir(server_id)
        await _ensure_server_disk_quota(server_id, data_dir, None, allow_init=False)
    except Exception as e:
        await log(f'Plugins ensure data_dir error: server_id={server_id} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    try:
        root, target, rel = _resolve_install_path(server_id, install_path)
    except Exception as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)

    q_target = shlex.quote(target)

    if action == 'uninstall':
        try:
            await log(f'Plugin uninstall start: server_id={server_id} path={rel}')
            await run_cmd(f'rm -rf {q_target} >/dev/null 2>&1 || true')
            await log(f'Plugin uninstall done: server_id={server_id} path={rel}')
            return web.json_response({'ok': True})
        except Exception as e:
            await log(f'Plugin uninstall error: server_id={server_id} path={rel} error={e}')
            return web.json_response({'ok': False, 'error': str(e)}, status=500)

    archive_url = str(data.get('archive_url') or '').strip()
    archive_type = str(data.get('archive_type') or 'zip').strip().lower()

    file_actions = data.get('file_actions')
    if file_actions is None:
        file_actions = []
    if not isinstance(file_actions, list):
        return web.json_response({'ok': False, 'error': 'file_actions must be array'}, status=400)

    has_archive = archive_url != ''
    if has_archive and not (archive_url.startswith('http://') or archive_url.startswith('https://')):
        return web.json_response({'ok': False, 'error': 'archive_url must be http(s) URL'}, status=400)

    if (not has_archive) and len(file_actions) == 0:
        return web.json_response({'ok': False, 'error': 'Either archive_url or file_actions must be provided'}, status=400)

    if archive_type not in ('zip', 'tar', 'targz'):
        return web.json_response({'ok': False, 'error': 'archive_type must be zip|tar|targz'}, status=400)

    tmp_base = f'/tmp/vtx-plugin-{server_id}'
    q_tmp_base = shlex.quote(tmp_base)
    q_url = shlex.quote(archive_url)

    archive_file = 'archive.zip' if archive_type == 'zip' else ('archive.tar.gz' if archive_type == 'targz' else 'archive.tar')

    try:
        await log(f'Plugin install start: server_id={server_id} type={archive_type} path={rel} url={(archive_url[:180])}')

        if has_archive:
            await run_cmd(f'rm -rf {q_tmp_base} >/dev/null 2>&1 || true')
            await run_cmd(f'mkdir -p {q_tmp_base}/extract')

            await run_cmd(f'curl -fsSL {q_url} -o {q_tmp_base}/{archive_file}')

            if archive_type == 'zip':
                try:
                    await run_cmd('unzip -v >/dev/null 2>&1')
                except Exception:
                    raise Exception('unzip is required to extract .zip archives')
                await run_cmd(f'unzip -q {q_tmp_base}/{archive_file} -d {q_tmp_base}/extract')
            elif archive_type == 'targz':
                try:
                    await run_cmd('tar --version >/dev/null 2>&1')
                except Exception:
                    raise Exception('tar is required to extract tar.gz archives')
                await run_cmd(f'tar -xzf {q_tmp_base}/{archive_file} -C {q_tmp_base}/extract')
            else:
                try:
                    await run_cmd('tar --version >/dev/null 2>&1')
                except Exception:
                    raise Exception('tar is required to extract tar archives')
                await run_cmd(f'tar -xf {q_tmp_base}/{archive_file} -C {q_tmp_base}/extract')

            # Ensure target exists
            await run_cmd(f'mkdir -p {q_target}')

            # If archive has a single top-level directory, copy contents of that dir; else copy extract root.
            await run_cmd(
                f'if [ "$(find {q_tmp_base}/extract -mindepth 1 -maxdepth 1 | wc -l)" = "1" ] && '
                f'[ "$(find {q_tmp_base}/extract -mindepth 1 -maxdepth 1 -type d | wc -l)" = "1" ]; then '
                f'  d=$(find {q_tmp_base}/extract -mindepth 1 -maxdepth 1 -type d | head -n1); '
                f'  cp -a "$d"/. {q_target}/; '
                f'else '
                f'  cp -a {q_tmp_base}/extract/. {q_target}/; '
                f'fi'
            )

        if len(file_actions) > 0:
            await log(f'Plugin applying file_actions: server_id={server_id} count={len(file_actions)}')
            _apply_file_actions(server_id, file_actions)

        try:
            await _post_archive_fix_permissions(game, data_dir)
        except Exception:
            pass

        await log(f'Plugin install done: server_id={server_id} path={rel}')
        return web.json_response({'ok': True})
    except Exception as e:
        await log(f'Plugin install error: server_id={server_id} path={rel} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)
    finally:
        try:
            shutil.rmtree(tmp_base, ignore_errors=True)
        except Exception:
            pass
