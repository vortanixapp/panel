import ipaddress
import json
import os
import shlex
import time
from typing import Any, Dict, List, Tuple

from aiohttp import web

from vtxdaemon.cmd import run_cmd
from vtxdaemon.logbuf import log
from vtxdaemon.servers_common import _ensure_server_data_dir, _server_data_dir


def _state_path(server_id: int) -> str:
    return os.path.join(_server_data_dir(server_id), '.vtx_firewall.json')


def _load_state(server_id: int) -> Dict[str, Any]:
    path = _state_path(server_id)
    try:
        with open(path, 'r', encoding='utf-8') as f:
            data = json.load(f)
        if not isinstance(data, dict):
            data = {}
    except FileNotFoundError:
        data = {}
    except Exception:
        data = {}

    mode = str(data.get('mode') or 'allow').strip().lower()
    if mode not in ('allow', 'deny'):
        mode = 'allow'

    enabled = bool(data.get('enabled', False))
    rules = data.get('rules')
    if not isinstance(rules, list):
        rules = []

    out_rules: List[Dict[str, Any]] = []
    for r in rules:
        if not isinstance(r, dict):
            continue
        rid = str(r.get('id') or '').strip()
        cidr = str(r.get('cidr') or '').strip()
        proto = str(r.get('proto') or 'both').strip().lower()
        if proto not in ('udp', 'tcp', 'both'):
            proto = 'both'
        if rid == '' or cidr == '':
            continue
        out_rules.append({
            'id': rid,
            'cidr': cidr,
            'proto': proto,
            'enabled': bool(r.get('enabled', True)),
            'note': str(r.get('note') or '').strip(),
        })

    return {
        'enabled': enabled,
        'mode': mode,
        'rules': out_rules,
    }


def _save_state(server_id: int, state: Dict[str, Any]) -> None:
    path = _state_path(server_id)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    tmp = path + '.tmp'
    with open(tmp, 'w', encoding='utf-8') as f:
        json.dump(state, f, ensure_ascii=False)
    os.replace(tmp, path)


def _validate_cidr(cidr: str) -> str:
    cidr = str(cidr or '').strip()
    if cidr == '':
        raise ValueError('cidr is required')
    try:
        net = ipaddress.ip_network(cidr, strict=False)
    except Exception:
        raise ValueError('cidr is invalid')
    if net.version != 4:
        raise ValueError('Only IPv4 is supported')
    return str(net)


def _validate_proto(proto: str) -> str:
    proto = str(proto or '').strip().lower() or 'both'
    if proto not in ('udp', 'tcp', 'both'):
        raise ValueError('proto must be udp|tcp|both')
    return proto


def _chain_name(server_id: int) -> str:
    base = f'VTX-SRV-{server_id}'
    # iptables chain name limit is 28 chars
    return base[:28]


async def _ensure_iptables_available() -> None:
    await run_cmd('command -v iptables >/dev/null 2>&1')
    await run_cmd('iptables -L DOCKER-USER >/dev/null 2>&1 || iptables -N DOCKER-USER >/dev/null 2>&1 || true')


async def _iptables_chain_ensure(chain: str) -> None:
    q = shlex.quote(chain)
    await run_cmd(f'iptables -N {q} >/dev/null 2>&1 || true')


async def _iptables_chain_flush(chain: str) -> None:
    q = shlex.quote(chain)
    await run_cmd(f'iptables -F {q} >/dev/null 2>&1 || true')


async def _iptables_ensure_jump(chain: str, port: int, proto: str) -> None:
    q_chain = shlex.quote(chain)
    q_proto = shlex.quote(proto)
    # Insert at top if missing
    check = f'iptables -C DOCKER-USER -p {q_proto} --dport {port} -j {q_chain} >/dev/null 2>&1'
    insert = f'iptables -I DOCKER-USER 1 -p {q_proto} --dport {port} -j {q_chain}'
    await run_cmd(f'sh -c {shlex.quote(check + " || " + insert)}')


async def _iptables_delete_jump(chain: str, port: int, proto: str) -> None:
    q_chain = shlex.quote(chain)
    q_proto = shlex.quote(proto)
    # Delete all occurrences
    cmd = (
        f"sh -c {shlex.quote('while iptables -C DOCKER-USER -p ' + proto + ' --dport ' + str(port) + ' -j ' + chain + ' >/dev/null 2>&1; do iptables -D DOCKER-USER -p ' + proto + ' --dport ' + str(port) + ' -j ' + chain + '; done')}"
    )
    await run_cmd(cmd)


async def _iptables_delete_chain(chain: str) -> None:
    q_chain = shlex.quote(chain)
    await run_cmd(f'iptables -F {q_chain} >/dev/null 2>&1 || true')
    await run_cmd(f'iptables -X {q_chain} >/dev/null 2>&1 || true')


async def _apply_firewall(server_id: int, port: int, state: Dict[str, Any]) -> None:
    await _ensure_iptables_available()

    chain = _chain_name(server_id)
    await _iptables_chain_ensure(chain)
    await _iptables_chain_flush(chain)

    enabled = bool(state.get('enabled', False))
    mode = str(state.get('mode') or 'allow').strip().lower()
    rules = state.get('rules')
    if not isinstance(rules, list):
        rules = []

    # Ensure jump rules exist when enabled
    if enabled:
        await _iptables_ensure_jump(chain, port, 'udp')
        await _iptables_ensure_jump(chain, port, 'tcp')

    if not enabled:
        # Remove jump rules and chain (best-effort)
        try:
            await _iptables_delete_jump(chain, port, 'udp')
        except Exception:
            pass
        try:
            await _iptables_delete_jump(chain, port, 'tcp')
        except Exception:
            pass
        try:
            await _iptables_delete_chain(chain)
        except Exception:
            pass
        return

    q_chain = shlex.quote(chain)

    active_rules: List[Tuple[str, str, str]] = []
    for r in rules:
        if not isinstance(r, dict):
            continue
        if not r.get('enabled', True):
            continue
        cidr = str(r.get('cidr') or '').strip()
        proto = str(r.get('proto') or 'both').strip().lower()
        rid = str(r.get('id') or '').strip()
        if cidr == '' or rid == '':
            continue
        if proto not in ('udp', 'tcp', 'both'):
            proto = 'both'
        active_rules.append((rid, cidr, proto))

    if mode == 'allow':
        for rid, cidr, proto in active_rules:
            if proto in ('udp', 'both'):
                await run_cmd(f'iptables -A {q_chain} -p udp --dport {port} -s {shlex.quote(cidr)} -j ACCEPT')
            if proto in ('tcp', 'both'):
                await run_cmd(f'iptables -A {q_chain} -p tcp --dport {port} -s {shlex.quote(cidr)} -j ACCEPT')
        await run_cmd(f'iptables -A {q_chain} -p udp --dport {port} -j DROP')
        await run_cmd(f'iptables -A {q_chain} -p tcp --dport {port} -j DROP')
    else:
        for rid, cidr, proto in active_rules:
            if proto in ('udp', 'both'):
                await run_cmd(f'iptables -A {q_chain} -p udp --dport {port} -s {shlex.quote(cidr)} -j DROP')
            if proto in ('tcp', 'both'):
                await run_cmd(f'iptables -A {q_chain} -p tcp --dport {port} -s {shlex.quote(cidr)} -j DROP')
        await run_cmd(f'iptables -A {q_chain} -p udp --dport {port} -j ACCEPT')
        await run_cmd(f'iptables -A {q_chain} -p tcp --dport {port} -j ACCEPT')


async def server_firewall_list_handler(request: web.Request) -> web.Response:
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    try:
        server_id = int(data.get('server_id'))
    except Exception:
        return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    await _ensure_server_data_dir(server_id)
    state = _load_state(server_id)

    return web.json_response({'ok': True, **state})


async def server_firewall_set_handler(request: web.Request) -> web.Response:
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    try:
        server_id = int(data.get('server_id'))
    except Exception:
        return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    try:
        port = int(data.get('port'))
        port = max(1, min(65535, port))
    except Exception:
        return web.json_response({'ok': False, 'error': 'port is required (int)'}, status=400)

    enabled = bool(data.get('enabled', False))
    mode = str(data.get('mode') or 'allow').strip().lower()
    if mode not in ('allow', 'deny'):
        return web.json_response({'ok': False, 'error': 'mode must be allow|deny'}, status=400)

    rules_in = data.get('rules')
    if not isinstance(rules_in, list):
        rules_in = []

    # sanitize
    rules: List[Dict[str, Any]] = []
    for r in rules_in:
        if not isinstance(r, dict):
            continue
        rid = str(r.get('id') or '').strip()
        cidr = str(r.get('cidr') or '').strip()
        proto = str(r.get('proto') or 'both').strip().lower()
        note = str(r.get('note') or '').strip()
        ren = bool(r.get('enabled', True))
        if rid == '':
            continue
        try:
            cidr = _validate_cidr(cidr)
            proto = _validate_proto(proto)
        except ValueError:
            continue
        rules.append({'id': rid, 'cidr': cidr, 'proto': proto, 'enabled': ren, 'note': note})

    await _ensure_server_data_dir(server_id)

    state = {
        'enabled': enabled,
        'mode': mode,
        'rules': rules,
    }

    _save_state(server_id, state)

    try:
        await _apply_firewall(server_id, port, state)
    except Exception as e:
        await log(f'Firewall apply error: server_id={server_id} port={port} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    return web.json_response({'ok': True, **state})


async def server_firewall_add_rule_handler(request: web.Request) -> web.Response:
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    try:
        server_id = int(data.get('server_id'))
    except Exception:
        return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    try:
        port = int(data.get('port'))
        port = max(1, min(65535, port))
    except Exception:
        return web.json_response({'ok': False, 'error': 'port is required (int)'}, status=400)

    try:
        cidr = _validate_cidr(data.get('cidr'))
        proto = _validate_proto(data.get('proto'))
        note = str(data.get('note') or '').strip()
        enabled = bool(data.get('enabled', True))
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)

    await _ensure_server_data_dir(server_id)
    state = _load_state(server_id)

    rid = str(int(time.time() * 1000))
    state_rules = state.get('rules') if isinstance(state, dict) else []
    if not isinstance(state_rules, list):
        state_rules = []
    state_rules.append({'id': rid, 'cidr': cidr, 'proto': proto, 'enabled': enabled, 'note': note})

    state['rules'] = state_rules
    _save_state(server_id, state)

    try:
        await _apply_firewall(server_id, port, state)
    except Exception as e:
        await log(f'Firewall apply error: server_id={server_id} port={port} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    return web.json_response({'ok': True, **state})


async def server_firewall_delete_rule_handler(request: web.Request) -> web.Response:
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    try:
        server_id = int(data.get('server_id'))
    except Exception:
        return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    try:
        port = int(data.get('port'))
        port = max(1, min(65535, port))
    except Exception:
        return web.json_response({'ok': False, 'error': 'port is required (int)'}, status=400)

    rid = str(data.get('rule_id') or '').strip()
    if rid == '':
        return web.json_response({'ok': False, 'error': 'rule_id is required'}, status=400)

    await _ensure_server_data_dir(server_id)
    state = _load_state(server_id)

    rules = state.get('rules') if isinstance(state, dict) else []
    if not isinstance(rules, list):
        rules = []

    new_rules = []
    removed = False
    for r in rules:
        if isinstance(r, dict) and str(r.get('id') or '') == rid:
            removed = True
            continue
        new_rules.append(r)

    state['rules'] = new_rules
    _save_state(server_id, state)

    try:
        await _apply_firewall(server_id, port, state)
    except Exception as e:
        await log(f'Firewall apply error: server_id={server_id} port={port} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    return web.json_response({'ok': True, 'removed': removed, **state})


async def server_firewall_toggle_rule_handler(request: web.Request) -> web.Response:
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    try:
        server_id = int(data.get('server_id'))
    except Exception:
        return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    try:
        port = int(data.get('port'))
        port = max(1, min(65535, port))
    except Exception:
        return web.json_response({'ok': False, 'error': 'port is required (int)'}, status=400)

    rid = str(data.get('rule_id') or '').strip()
    if rid == '':
        return web.json_response({'ok': False, 'error': 'rule_id is required'}, status=400)

    enabled = bool(data.get('enabled', True))

    await _ensure_server_data_dir(server_id)
    state = _load_state(server_id)

    rules = state.get('rules') if isinstance(state, dict) else []
    if not isinstance(rules, list):
        rules = []

    found = False
    for r in rules:
        if isinstance(r, dict) and str(r.get('id') or '') == rid:
            r['enabled'] = enabled
            found = True
            break

    state['rules'] = rules
    _save_state(server_id, state)

    try:
        await _apply_firewall(server_id, port, state)
    except Exception as e:
        await log(f'Firewall apply error: server_id={server_id} port={port} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    return web.json_response({'ok': True, 'found': found, **state})


async def server_firewall_toggle_handler(request: web.Request) -> web.Response:
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    try:
        server_id = int(data.get('server_id'))
    except Exception:
        return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    try:
        port = int(data.get('port'))
        port = max(1, min(65535, port))
    except Exception:
        return web.json_response({'ok': False, 'error': 'port is required (int)'}, status=400)

    enabled = bool(data.get('enabled', False))

    await _ensure_server_data_dir(server_id)
    state = _load_state(server_id)
    state['enabled'] = enabled
    _save_state(server_id, state)

    try:
        await _apply_firewall(server_id, port, state)
    except Exception as e:
        await log(f'Firewall apply error: server_id={server_id} port={port} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    return web.json_response({'ok': True, **state})
