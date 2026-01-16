import json
import os
import re
import shlex
import time
from typing import Any, Dict, List

from aiohttp import web

from vtxdaemon.logbuf import log
from vtxdaemon.servers_common import _ensure_server_data_dir, _server_data_dir
from vtxdaemon.servers_game import _get_container_name, _require_game


def _cron_state_path(server_id: int) -> str:
    return os.path.join(_server_data_dir(server_id), '.vtx_cron.json')


def _cron_d_path(server_id: int) -> str:
    return f'/etc/cron.d/vtx-server-{server_id}'


def _load_state(server_id: int) -> Dict[str, Any]:
    path = _cron_state_path(server_id)
    try:
        with open(path, 'r', encoding='utf-8') as f:
            data = json.load(f)
        if not isinstance(data, dict):
            return {'jobs': []}
        jobs = data.get('jobs')
        if not isinstance(jobs, list):
            return {'jobs': []}
        return {'jobs': jobs}
    except FileNotFoundError:
        return {'jobs': []}
    except Exception:
        return {'jobs': []}


def _save_state(server_id: int, state: Dict[str, Any]) -> None:
    path = _cron_state_path(server_id)
    d = os.path.dirname(path)
    os.makedirs(d, exist_ok=True)
    tmp = path + '.tmp'
    with open(tmp, 'w', encoding='utf-8') as f:
        json.dump(state, f, ensure_ascii=False)
    os.replace(tmp, path)


def _validate_schedule(expr: str) -> str:
    expr = str(expr or '').strip()
    parts = [p for p in expr.split() if p.strip()]
    if len(parts) != 5:
        raise ValueError('schedule must have 5 fields (min hour dom mon dow)')

    allowed = re.compile(r'^[0-9*/,-]+$')
    for p in parts:
        if p == '*':
            continue
        if not allowed.match(p):
            raise ValueError('schedule contains invalid characters')

    return ' '.join(parts)


def _validate_command(cmd: str) -> str:
    cmd = str(cmd or '').strip()
    if cmd == '':
        raise ValueError('command is required')
    if len(cmd) > 2000:
        raise ValueError('command is too long')
    return cmd


def _render_cron_d(server_id: int, container_name: str, jobs: List[Dict[str, Any]]) -> None:
    path = _cron_d_path(server_id)

    enabled_jobs: List[Dict[str, Any]] = []
    for j in jobs:
        if not isinstance(j, dict):
            continue
        if not j.get('enabled', True):
            continue
        sched = str(j.get('schedule') or '').strip()
        cmd = str(j.get('command') or '').strip()
        if sched == '' or cmd == '':
            continue
        enabled_jobs.append({'id': str(j.get('id') or ''), 'schedule': sched, 'command': cmd})

    if len(enabled_jobs) == 0:
        try:
            os.remove(path)
        except FileNotFoundError:
            pass
        return

    lines: List[str] = []
    for j in enabled_jobs:
        jid = j['id'] or 'job'
        q_container = shlex.quote(container_name)
        q_cmd = shlex.quote(j['command'])
        line = f"{j['schedule']} root docker exec {q_container} sh -lc {q_cmd} >/dev/null 2>&1 # {jid}"
        lines.append(line)

    content = '\n'.join(lines).rstrip('\n') + '\n'
    tmp = path + '.tmp'
    with open(tmp, 'w', encoding='utf-8') as f:
        f.write(content)
    os.replace(tmp, path)
    os.chmod(path, 0o644)


async def server_cron_list_handler(request: web.Request) -> web.Response:
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    try:
        server_id = int((data or {}).get('server_id'))
    except Exception:
        return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    try:
        await _ensure_server_data_dir(server_id)
    except Exception as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)

    state = _load_state(server_id)
    jobs = state.get('jobs') if isinstance(state, dict) else []
    if not isinstance(jobs, list):
        jobs = []

    return web.json_response({'ok': True, 'jobs': jobs})


async def server_cron_create_handler(request: web.Request) -> web.Response:
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    payload = data if isinstance(data, dict) else {}

    try:
        server_id = int(payload.get('server_id'))
    except Exception:
        return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    try:
        game = _require_game(payload)
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)

    try:
        await _ensure_server_data_dir(server_id)
    except Exception as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)

    try:
        schedule = _validate_schedule(payload.get('schedule'))
        command = _validate_command(payload.get('command'))
        enabled = bool(payload.get('enabled', True))
        name = str(payload.get('name') or '').strip()
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)

    state = _load_state(server_id)
    jobs = state.get('jobs') if isinstance(state, dict) else []
    if not isinstance(jobs, list):
        jobs = []

    job_id = str(int(time.time() * 1000))
    jobs.append({
        'id': job_id,
        'name': name,
        'schedule': schedule,
        'command': command,
        'enabled': enabled,
    })

    state = {'jobs': jobs}
    _save_state(server_id, state)

    container_name = _get_container_name(game, server_id)
    _render_cron_d(server_id, container_name, jobs)

    await log(f'Cron job created: server_id={server_id} id={job_id}')

    return web.json_response({'ok': True, 'job_id': job_id, 'jobs': jobs})


async def server_cron_delete_handler(request: web.Request) -> web.Response:
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    payload = data if isinstance(data, dict) else {}

    try:
        server_id = int(payload.get('server_id'))
    except Exception:
        return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    try:
        game = _require_game(payload)
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)

    job_id = str(payload.get('job_id') or '').strip()
    if job_id == '':
        return web.json_response({'ok': False, 'error': 'job_id is required'}, status=400)

    try:
        await _ensure_server_data_dir(server_id)
    except Exception as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)

    state = _load_state(server_id)
    jobs = state.get('jobs') if isinstance(state, dict) else []
    if not isinstance(jobs, list):
        jobs = []

    new_jobs = []
    removed = False
    for j in jobs:
        if isinstance(j, dict) and str(j.get('id') or '') == job_id:
            removed = True
            continue
        new_jobs.append(j)

    state = {'jobs': new_jobs}
    _save_state(server_id, state)

    container_name = _get_container_name(game, server_id)
    _render_cron_d(server_id, container_name, new_jobs)

    await log(f'Cron job delete: server_id={server_id} id={job_id} removed={removed}')

    return web.json_response({'ok': True, 'removed': removed, 'jobs': new_jobs})


async def server_cron_toggle_handler(request: web.Request) -> web.Response:
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    payload = data if isinstance(data, dict) else {}

    try:
        server_id = int(payload.get('server_id'))
    except Exception:
        return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    try:
        game = _require_game(payload)
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)

    job_id = str(payload.get('job_id') or '').strip()
    if job_id == '':
        return web.json_response({'ok': False, 'error': 'job_id is required'}, status=400)

    enabled = bool(payload.get('enabled', True))

    try:
        await _ensure_server_data_dir(server_id)
    except Exception as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)

    state = _load_state(server_id)
    jobs = state.get('jobs') if isinstance(state, dict) else []
    if not isinstance(jobs, list):
        jobs = []

    found = False
    for j in jobs:
        if isinstance(j, dict) and str(j.get('id') or '') == job_id:
            j['enabled'] = enabled
            found = True
            break

    state = {'jobs': jobs}
    _save_state(server_id, state)

    container_name = _get_container_name(game, server_id)
    _render_cron_d(server_id, container_name, jobs)

    await log(f'Cron job toggle: server_id={server_id} id={job_id} enabled={enabled} found={found}')

    return web.json_response({'ok': True, 'found': found, 'jobs': jobs})
