import asyncio
from datetime import datetime
import os
import shlex
import shutil
from typing import Optional, Tuple

from aiohttp import web

from vtxdaemon.cmd import run_cmd
from vtxdaemon.config import (
    SAMP_DOCKER_IMAGE,
    CRMP_DOCKER_IMAGE,
    CS16_DOCKER_IMAGE,
    CSS_DOCKER_IMAGE,
    CS2_DOCKER_IMAGE,
    RUST_DOCKER_IMAGE,
    TF2_DOCKER_IMAGE,
    GMOD_DOCKER_IMAGE,
    MTA_DOCKER_IMAGE,
    MC_JAVA_DOCKER_IMAGE,
    MC_BEDROCK_DOCKER_IMAGE,
    UNTURNED_DOCKER_IMAGE,
)
from vtxdaemon.logbuf import log
from vtxdaemon.servers_common import (
    _assert_safe_data_dir,
    _disk_usage_mb,
    _docker_apply_limits,
    _docker_container_inspect_summary,
    _docker_first_udp_host_port,
    _docker_limit_flags,
    _ensure_server_data_dir,
    _ensure_server_disk_quota,
    _parse_disk_limit_mb,
    _parse_resource_limits,
    _server_data_dir,
)
from vtxdaemon.servers_fs import _wipe_data_dir_contents
from vtxdaemon.servers_rcon_samp import _samp_rcon
from vtxdaemon.servers_steam import _ensure_steamcmd_host_deps
from vtxdaemon.servers_docker import (
    _container_id_by_name,
    _docker_logs,
    _docker_restart,
    _docker_rm_force,
    _docker_run_detached,
    _docker_start,
    _docker_stats,
    _docker_stop,
    _ensure_docker_available,
)
from vtxdaemon.servers_archive import archive_bootstrap
from vtxdaemon.servers_steam_install import steam_install
from vtxdaemon.servers_game import (
    CRMP_GAME_CODES,
    CS16_GAME_CODES,
    CS2_GAME_CODES,
    CSS_GAME_CODES,
    GMOD_GAME_CODES,
    MC_BEDROCK_GAME_CODES,
    MC_JAVA_GAME_CODES,
    MTA_GAME_CODES,
    RUST_GAME_CODES,
    SAMP_GAME_CODES,
    TF2_GAME_CODES,
    UNTURNED_GAME_CODES,
    _ensure_cs2_steamclient,
    _ensure_rust_maxplayers_env,
    _get_container_name,
    _get_env_var,
    _get_ports_flag,
    _normalize_game,
    _require_game,
)

def _parse_cpu_shares(data: dict) -> Optional[int]:
    cpu_shares_raw = data.get('cpu_shares', None)
    if cpu_shares_raw is None:
        cpu_shares_raw = data.get('cpu_share', None)
    if cpu_shares_raw is None:
        cpu_shares_raw = data.get('cpu_weight', None)
    if cpu_shares_raw is None:
        return None

    try:
        cpu_shares = int(str(cpu_shares_raw).strip())
    except Exception:
        return None

    if cpu_shares <= 0:
        return None

    return max(2, min(cpu_shares, 262144))


async def create_server_handler(request):
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    try:
        game = _require_game(data)
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)
    server_id = data.get('server_id')

    if not isinstance(server_id, int):
        try:
            server_id = int(server_id)
        except Exception:
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    # Select appropriate Docker image and container based on game type
    if game in CS16_GAME_CODES:
        image = str(data.get('image') or CS16_DOCKER_IMAGE)
    elif game in CSS_GAME_CODES:
        image = str(data.get('image') or CSS_DOCKER_IMAGE)
    elif game in CS2_GAME_CODES:
        image = str(data.get('image') or CS2_DOCKER_IMAGE)
    elif game in RUST_GAME_CODES:
        image = str(data.get('image') or RUST_DOCKER_IMAGE)
    elif game in TF2_GAME_CODES:
        image = str(data.get('image') or TF2_DOCKER_IMAGE)
    elif game in GMOD_GAME_CODES:
        image = str(data.get('image') or GMOD_DOCKER_IMAGE)
    elif game in CRMP_GAME_CODES:
        image = str(data.get('image') or CRMP_DOCKER_IMAGE)
    elif game in MTA_GAME_CODES:
        image = str(data.get('image') or MTA_DOCKER_IMAGE)
    elif game in MC_JAVA_GAME_CODES:
        image = str(data.get('image') or MC_JAVA_DOCKER_IMAGE)
    elif game in MC_BEDROCK_GAME_CODES:
        image = str(data.get('image') or MC_BEDROCK_DOCKER_IMAGE)
    elif game in UNTURNED_GAME_CODES:
        image = str(data.get('image') or UNTURNED_DOCKER_IMAGE)
    else:
        image = str(data.get('image') or SAMP_DOCKER_IMAGE)
    container_name = _get_container_name(game, server_id)
    q_image = shlex.quote(image)
    port = data.get('port')

    if port is None:
        return web.json_response({'ok': False, 'error': 'port is required (int)'}, status=400)

    try:
        port = int(port)
    except Exception:
        return web.json_response({'ok': False, 'error': 'port must be int'}, status=400)

    if port < 1 or port > 65535:
        return web.json_response({'ok': False, 'error': 'port out of range (1..65535)'}, status=400)

    try:
        await _ensure_docker_available()
    except Exception as e:
        await log(f'Docker missing/error: {e}')
        return web.json_response({'ok': False, 'error': 'Docker is not available'}, status=500)

    data_dir = await _ensure_server_data_dir(server_id)

    archive_url = data.get('archive_url')
    version_name = data.get('version_name')
    steam_app_id = data.get('steam_app_id')
    steam_branch = data.get('steam_branch')

    disk_limit_mb = _parse_disk_limit_mb(data)
    allow_init_disk = False
    base_dir = os.path.dirname(data_dir)
    marker_path = os.path.join(base_dir, '.disk_quota')

    def _is_effectively_empty_dir(path: str) -> bool:
        try:
            if not os.path.isdir(path):
                return False
            entries = [e for e in os.listdir(path) if e not in ('lost+found',)]
            return len(entries) == 0
        except Exception:
            return False

    if disk_limit_mb is not None and (not os.path.exists(marker_path)):
        try:
            allow_init_disk = _is_effectively_empty_dir(data_dir)
        except Exception:
            allow_init_disk = False

    try:
        await _ensure_server_disk_quota(server_id, data_dir, disk_limit_mb, allow_init=allow_init_disk)
    except Exception as e:
        await log(f'Disk quota init/mount error: server_id={server_id} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    try:
        steam_app_id_int = int(steam_app_id) if steam_app_id is not None and str(steam_app_id).strip() != '' else 0
    except Exception:
        steam_app_id_int = 0

    wants_version = (isinstance(archive_url, str) and archive_url.strip() != '') or steam_app_id_int > 0
    try:
        await log(
            f'Create wants_version={wants_version} server_id={server_id} '
            f'archive_url={(str(archive_url)[:180] if archive_url is not None else "")} '
            f'steam_app_id={steam_app_id_int} data_dir={data_dir}'
        )
    except Exception:
        pass

    if wants_version:
        try:
            await log(f'Create wiping data dir: server_id={server_id} data_dir={data_dir} keep_lost_found=1')
            await _wipe_data_dir_contents(server_id, data_dir, keep_lost_found=True)
            await log(f'Create wipe complete: server_id={server_id}')
        except Exception:
            pass

    try:
        is_empty = _is_effectively_empty_dir(data_dir)
    except Exception:
        is_empty = False

    # Version-only mode: if /data is empty, archive_url OR steam_app_id must be provided.
    if is_empty and (not isinstance(archive_url, str) or archive_url.strip() == '') and steam_app_id_int <= 0:
        return web.json_response({'ok': False, 'error': 'archive_url or steam_app_id is required when server data directory is empty'}, status=400)

    # Steam source: install with steamcmd when version source is provided.
    if wants_version and steam_app_id_int > 0:
        try:
            await _ensure_steamcmd_host_deps()
            await steam_install(
                server_id=server_id,
                game=game,
                data_dir=data_dir,
                app_id=steam_app_id_int,
                branch=str(steam_branch or ''),
            )
        except Exception as e:
            await log(f'Steam install error: server_id={server_id} app_id={steam_app_id_int} error={e}')
            return web.json_response({'ok': False, 'error': str(e)}, status=500)

    if game in CS2_GAME_CODES:
        try:
            await _ensure_cs2_steamclient(data_dir)
        except Exception as e:
            await log(f'Ensure cs2 steamclient failed: server_id={server_id} error={e}')

    # Archive source: bootstrap /data from the archive when version source is provided.
    if wants_version and steam_app_id_int <= 0 and isinstance(archive_url, str) and archive_url.strip() != '':
        try:
            await archive_bootstrap(server_id=server_id, game=game, data_dir=data_dir, url=archive_url.strip(), version_name=version_name)
        except ValueError as e:
            return web.json_response({'ok': False, 'error': str(e)}, status=400)
        except Exception as e:
            await log(f'Archive bootstrap error: server_id={server_id} url={archive_url.strip()} error={e}')
            return web.json_response({'ok': False, 'error': str(e)}, status=500)

    if game in RUST_GAME_CODES:
        try:
            await _ensure_rust_maxplayers_env(data_dir, int(data.get('slots') or 0))
        except Exception:
            pass

    if game in CS2_GAME_CODES:
        try:
            await _ensure_cs2_steamclient(data_dir)
        except Exception as e:
            await log(f'Ensure cs2 steamclient failed (create): server_id={server_id} error={e}')

    await log(f'Server create requested: game={game} server_id={server_id} port={port} image={image}')

    cpu_cores, mem_mb = _parse_resource_limits(data)
    cpu_shares = _parse_cpu_shares(data)
    await log(f'Limits requested: server_id={server_id} cpu_cores={cpu_cores} cpu_shares={cpu_shares} mem_mb={mem_mb}')

    try:
        existing_id = await _container_id_by_name(container_name)
        if existing_id.strip() != '':
            try:
                await _docker_apply_limits(container_name, cpu_cores, mem_mb, cpu_shares)
            except Exception as e:
                await log(f'Docker update limits error: container={container_name} server_id={server_id} cpu_cores={cpu_cores} cpu_shares={cpu_shares} mem_mb={mem_mb} error={e}')
            mapped_port = None
            try:
                mapped_port = await _docker_first_udp_host_port(container_name)
            except Exception:
                mapped_port = None

            return web.json_response({
                'ok': True,
                'container_id': existing_id.strip(),
                'container_name': container_name,
                'port': mapped_port or port,
                'mapping': '',
            })
    except Exception:
        pass

    try:
        flags = _docker_limit_flags(cpu_cores, mem_mb, cpu_shares, True)
        flags2 = _docker_limit_flags(cpu_cores, mem_mb, cpu_shares, False)

        if game in RUST_GAME_CODES:
            try:
                await _ensure_rust_maxplayers_env(data_dir, int(data.get('slots') or 0))
            except Exception:
                pass

        env_var = _get_env_var(game, port, data)
        ports_flag = _get_ports_flag(game, port)

        container_id = await _docker_run_detached(
            container_name=container_name,
            image=image,
            data_dir=data_dir,
            ports_flag=ports_flag,
            env_var=env_var,
            flags=flags,
            fallback_flags=flags2,
            mem_mb=mem_mb,
        )

        await log(f'Server created: {container_name} id={container_id}')
        return web.json_response({
            'ok': True,
            'container_id': container_id.strip(),
            'container_name': container_name,
            'port': port,
        })
    except Exception as e:
        await log(f'Server create error: {e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)


async def server_status_handler(request):
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
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'})

    try:
        game = _require_game(data)
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)
    container_name = _get_container_name(game, server_id)
    await log(f'Server status requested: server_id={server_id} game={game or 'samp'} container={container_name}')

    try:
        await _ensure_docker_available()
    except Exception as e:
        await log(f'Docker missing/error: {e}')
        return web.json_response({'ok': False, 'error': 'Docker is not available'})

    status, _cpu_limit, _mem_limit_mb = await _docker_container_inspect_summary(container_name)
    return web.json_response({'ok': True, **status})


async def server_logs_handler(request):
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
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'})

    tail = data.get('tail', 200)
    try:
        tail = int(tail)
    except Exception:
        tail = 200
    tail = max(1, min(tail, 1000))

    try:
        game = _require_game(data)
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)
    container_name = _get_container_name(game, server_id)

    await log(f'Server logs requested: server_id={server_id} game={game or 'samp'} container={container_name} tail={tail}')

    try:
        await _ensure_docker_available()
    except Exception as e:
        await log(f'Docker missing/error: {e}')
        return web.json_response({'ok': False, 'error': 'Docker is not available'}, status=500)

    status, _cpu_limit, _mem_limit_mb = await _docker_container_inspect_summary(container_name)
    if not status.get('exists'):
        return web.json_response({'ok': True, **status, 'logs': []})

    try:
        output = await _docker_logs(container_name, tail)
        lines = output.splitlines() if output else []
        return web.json_response({'ok': True, **status, 'logs': lines})
    except Exception as e:
        await log(f'Server logs error: {e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)


async def server_rcon_handler(request):
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'})

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'})

    server_id = data.get('server_id')
    if not isinstance(server_id, int):
        try:
            server_id = int(server_id)
        except Exception:
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'})

    command = data.get('command')
    if not isinstance(command, str) or command.strip() == '':
        return web.json_response({'ok': False, 'error': 'command is required (string)'})

    password = data.get('password')
    if not isinstance(password, str) or password.strip() == '':
        return web.json_response({'ok': False, 'error': 'password is required (string)'})

    rcon_port = data.get('rcon_port')
    if rcon_port is not None:
        try:
            rcon_port = int(rcon_port)
        except Exception:
            return web.json_response({'ok': False, 'error': 'rcon_port must be int'})
        if rcon_port <= 0 or rcon_port > 65535:
            return web.json_response({'ok': False, 'error': 'rcon_port must be in range 1..65535'})

    header_host = data.get('header_host')
    if header_host is not None and not isinstance(header_host, str):
        return web.json_response({'ok': False, 'error': 'header_host must be string'})

    header_port = data.get('header_port')
    if header_port is not None:
        try:
            header_port = int(header_port)
        except Exception:
            return web.json_response({'ok': False, 'error': 'header_port must be int'})

    command = command.strip()
    password = password.strip()

    if len(command) > 512:
        return web.json_response({'ok': False, 'error': 'command too long'})
    if len(password) > 128:
        return web.json_response({'ok': False, 'error': 'password too long'})

    try:
        game = _require_game(data)
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)
    container_name = _get_container_name(game, server_id)
    await log(f'Server RCON requested: server_id={server_id} game={game or 'samp'} container={container_name} cmd={command}')

    if game not in SAMP_GAME_CODES and game not in MC_JAVA_GAME_CODES and game not in MC_BEDROCK_GAME_CODES:
        return web.json_response({'ok': False, 'error': 'RCON is supported only for SA-MP and Minecraft Java'}, status=400)

    if game in MC_BEDROCK_GAME_CODES:
        return web.json_response({'ok': False, 'error': 'RCON for Minecraft Bedrock is not supported yet'}, status=400)

    try:
        await _ensure_docker_available()
    except Exception as e:
        await log(f'Docker missing/error: {e}')
        return web.json_response({'ok': False, 'error': 'Docker is not available'})

    status, _cpu_limit, _mem_limit_mb = await _docker_container_inspect_summary(container_name)
    if not status.get('exists'):
        return web.json_response({'ok': False, 'error': 'Container not found', **status})

    if str(status.get('state') or '').lower() != 'running':
        return web.json_response({'ok': False, 'error': 'Server is not running', **status})

    try:
        if game in MC_JAVA_GAME_CODES:
            # Prefer executing rcon-cli inside container (itzg/minecraft-server provides it)
            port = int(rcon_port or 25575)
            cmd_q = shlex.quote(command)
            pw_q = shlex.quote(password)
            c_q = shlex.quote(container_name)
            out = await run_cmd(f'docker exec {c_q} rcon-cli --host 127.0.0.1 --port {port} --password {pw_q} {cmd_q}')
            return web.json_response({'ok': True, **status, 'output': out})

        # SA-MP
        host_port = status.get('port')
        if not isinstance(host_port, int) or host_port <= 0:
            return web.json_response({'ok': False, 'error': 'Server port mapping not found', **status})
        target_host = '127.0.0.1'
        target_port = int(host_port)
        header_h = target_host
        header_p = target_port
        output = await _samp_rcon(header_h, header_p, password, command, target_host, target_port)
        return web.json_response({'ok': True, **status, 'output': output})
    except Exception as e:
        await log(f'Server RCON error: {e}')
        return web.json_response({'ok': False, 'error': str(e)})


async def server_metrics_handler(request):
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

    game = str(data.get('game') or data.get('game_code') or '').lower().strip()
    container_name = _get_container_name(game, server_id)

    disk_total_mb, disk_used_mb, disk_avail_mb, disk_percent = None, None, None, None
    try:
        data_dir = _server_data_dir(server_id)
        if os.path.exists(data_dir):
            disk_total_mb, disk_used_mb, disk_avail_mb, disk_percent = await _disk_usage_mb(data_dir)
    except Exception:
        disk_total_mb, disk_used_mb, disk_avail_mb, disk_percent = None, None, None, None

    disk_payload = {
        'disk_total_mb': disk_total_mb,
        'disk_used_mb': disk_used_mb,
        'disk_avail_mb': disk_avail_mb,
        'disk_percent': disk_percent,
    }

    try:
        await _ensure_docker_available()
    except Exception as e:
        await log(f'Docker missing/error: {e}')
        return web.json_response({'ok': False, 'error': 'Docker is not available'}, status=500)

    status, cpu_limit, mem_limit_mb = await _docker_container_inspect_summary(container_name)
    if not status.get('exists'):
        return web.json_response({'ok': True, **status, **disk_payload, 'ts': datetime.utcnow().isoformat() + 'Z', 'cpu_percent': None, 'mem_percent': None, 'cpu_limit': None, 'mem_limit_mb': None})

    state = str(status.get('state') or '').lower()
    if state != 'running':
        return web.json_response({'ok': True, **status, **disk_payload, 'ts': datetime.utcnow().isoformat() + 'Z', 'cpu_percent': 0.0, 'mem_percent': 0.0, 'cpu_limit': cpu_limit, 'mem_limit_mb': mem_limit_mb})

    try:
        q_container = shlex.quote(container_name)
        out = await _docker_stats(container_name)
        out = (out or '').strip().splitlines()[0].strip() if (out or '').strip() else ''
        cpu_s, mem_s = (out.split('|', 1) + [''])[:2] if '|' in out else (out, '')
        cpu = float(cpu_s.strip().replace('%', '').strip() or 0.0)
        mem = float(mem_s.strip().replace('%', '').strip() or 0.0)
        if cpu_limit is not None and cpu_limit > 0:
            cpu = cpu / cpu_limit
        cpu = max(0.0, min(100.0, cpu))
        return web.json_response({'ok': True, **status, **disk_payload, 'ts': datetime.utcnow().isoformat() + 'Z', 'cpu_percent': cpu, 'mem_percent': mem, 'cpu_limit': cpu_limit, 'mem_limit_mb': mem_limit_mb})
    except Exception as e:
        await log(f'Server metrics error: {e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)


async def server_start_handler(request):
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

    try:
        game = _require_game(data)
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)
    container_name = _get_container_name(game, server_id)
    await log(f'Server start requested: server_id={server_id} game={game or 'samp'} container={container_name}')

    data_dir = await _ensure_server_data_dir(server_id)
    disk_limit_mb = _parse_disk_limit_mb(data)
    try:
        await _ensure_server_disk_quota(server_id, data_dir, disk_limit_mb, allow_init=False)
    except Exception as e:
        await log(f'Disk quota mount error: server_id={server_id} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    if game in CS2_GAME_CODES:
        try:
            await _ensure_cs2_steamclient(data_dir)
        except Exception as e:
            await log(f'Ensure cs2 steamclient failed (start): server_id={server_id} error={e}')

    if game in RUST_GAME_CODES:
        try:
            await _ensure_rust_maxplayers_env(data_dir, int(data.get('slots') or 0))
        except Exception:
            pass

    cpu_cores, mem_mb = _parse_resource_limits(data)
    cpu_shares = _parse_cpu_shares(data)
    await log(f'Limits requested: server_id={server_id} cpu_cores={cpu_cores} cpu_shares={cpu_shares} mem_mb={mem_mb}')
    try:
        await _docker_apply_limits(container_name, cpu_cores, mem_mb, cpu_shares)
    except Exception as e:
        await log(f'Docker update limits error: container={container_name} server_id={server_id} cpu_cores={cpu_cores} cpu_shares={cpu_shares} mem_mb={mem_mb} error={e}')

    try:
        await _docker_start(container_name)
    except Exception as e:
        await log(f'Server start error: {e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    status, _cpu_limit, _mem_limit_mb = await _docker_container_inspect_summary(container_name)
    return web.json_response({'ok': True, **status})


async def server_stop_handler(request):
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

    try:
        game = _require_game(data)
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)
    container_name = _get_container_name(game, server_id)

    # Idempotent stop: if container is missing or already not running, treat as success.
    try:
        status, _cpu_limit, _mem_limit_mb = await _docker_container_inspect_summary(container_name)
    except Exception as e:
        await log(f'Server stop inspect error: server_id={server_id} game={game} container={container_name} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    if (not status.get('exists')) or status.get('state') != 'running':
        await log(
            f"Server stop ignored (already stopped): server_id={server_id} game={game} container={container_name} state={status.get('state')}"
        )
        return web.json_response({'ok': True, **status})

    await log(f"Server stop requested: server_id={server_id} game={game} container={container_name}")

    try:
        await _docker_stop(container_name)
    except Exception as e:
        await log(f'Server stop error: {e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    status, _cpu_limit, _mem_limit_mb = await _docker_container_inspect_summary(container_name)
    return web.json_response({'ok': True, **status})


async def server_restart_handler(request):
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

    try:
        game = _require_game(data)
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)
    container_name = _get_container_name(game, server_id)
    await log(f"Server restart requested: server_id={server_id} game={game or 'samp'} container={container_name}")

    data_dir = await _ensure_server_data_dir(server_id)
    disk_limit_mb = _parse_disk_limit_mb(data)
    try:
        await _ensure_server_disk_quota(server_id, data_dir, disk_limit_mb, allow_init=False)
    except Exception as e:
        await log(f'Disk quota mount error: server_id={server_id} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    if game in CS2_GAME_CODES:
        try:
            await _ensure_cs2_steamclient(data_dir)
        except Exception as e:
            await log(f'Ensure cs2 steamclient failed (restart): server_id={server_id} error={e}')

    if game in RUST_GAME_CODES:
        try:
            await _ensure_rust_maxplayers_env(data_dir, int(data.get('slots') or 0))
        except Exception:
            pass

    cpu_cores, mem_mb = _parse_resource_limits(data)
    cpu_shares = _parse_cpu_shares(data)
    await log(f'Limits requested: server_id={server_id} cpu_cores={cpu_cores} cpu_shares={cpu_shares} mem_mb={mem_mb}')

    try:
        await _docker_apply_limits(container_name, cpu_cores, mem_mb, cpu_shares)
    except Exception as e:
        await log(f'Docker update limits error: container={container_name} server_id={server_id} cpu_cores={cpu_cores} cpu_shares={cpu_shares} mem_mb={mem_mb} error={e}')

    try:
        await _docker_restart(container_name)
    except Exception as e:
        await log(f'Server restart error: {e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    status, _cpu_limit, _mem_limit_mb = await _docker_container_inspect_summary(container_name)
    return web.json_response({'ok': True, **status})


async def server_reinstall_handler(request):
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    try:
        game = _require_game(data)
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)
    server_id = data.get('server_id')

    if not isinstance(server_id, int):
        try:
            server_id = int(server_id)
        except Exception:
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    # Select appropriate Docker image and container based on game type
    if game in CS16_GAME_CODES:
        image = str(data.get('image') or CS16_DOCKER_IMAGE)
    elif game in CSS_GAME_CODES:
        image = str(data.get('image') or CSS_DOCKER_IMAGE)
    elif game in CS2_GAME_CODES:
        image = str(data.get('image') or CS2_DOCKER_IMAGE)
    elif game in RUST_GAME_CODES:
        image = str(data.get('image') or RUST_DOCKER_IMAGE)
    elif game in TF2_GAME_CODES:
        image = str(data.get('image') or TF2_DOCKER_IMAGE)
    elif game in GMOD_GAME_CODES:
        image = str(data.get('image') or GMOD_DOCKER_IMAGE)
    elif game in CRMP_GAME_CODES:
        image = str(data.get('image') or CRMP_DOCKER_IMAGE)
    elif game in MTA_GAME_CODES:
        image = str(data.get('image') or MTA_DOCKER_IMAGE)
    elif game in MC_JAVA_GAME_CODES:
        image = str(data.get('image') or MC_JAVA_DOCKER_IMAGE)
    elif game in MC_BEDROCK_GAME_CODES:
        image = str(data.get('image') or MC_BEDROCK_DOCKER_IMAGE)
    elif game in UNTURNED_GAME_CODES:
        image = str(data.get('image') or UNTURNED_DOCKER_IMAGE)
    else:
        image = str(data.get('image') or SAMP_DOCKER_IMAGE)
    container_name = _get_container_name(game, server_id)
    q_image = shlex.quote(image)

    port = data.get('port')

    if port is None:
        return web.json_response({'ok': False, 'error': 'port is required (int)'}, status=400)

    try:
        port = int(port)
    except Exception:
        return web.json_response({'ok': False, 'error': 'port must be int'}, status=400)

    if port < 1 or port > 65535:
        return web.json_response({'ok': False, 'error': 'port out of range (1..65535)'}, status=400)

    try:
        await _ensure_docker_available()
    except Exception as e:
        await log(f'Docker missing/error: {e}')
        return web.json_response({'ok': False, 'error': 'Docker is not available'}, status=500)

    await log(f'Server reinstall requested: server_id={server_id} port={port} image={image}')

    try:
        await _docker_rm_force(container_name, with_volumes=True)
    except Exception:
        pass

    data_dir = await _ensure_server_data_dir(server_id)
    try:
        _assert_safe_data_dir(server_id, data_dir)
    except Exception as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    archive_url = data.get('archive_url')
    version_name = data.get('version_name')
    steam_app_id = data.get('steam_app_id')
    steam_branch = data.get('steam_branch')

    disk_limit_mb = _parse_disk_limit_mb(data)
    base_dir = os.path.dirname(data_dir)
    marker_path = os.path.join(base_dir, '.disk_quota')
    img_path = os.path.join(base_dir, 'disk.img')

    allow_init_disk = False
    if disk_limit_mb is not None and (not os.path.exists(marker_path)) and (not os.path.exists(img_path)):
        try:
            await _wipe_data_dir_contents(server_id, data_dir, keep_lost_found=False)
        except Exception:
            pass
        allow_init_disk = True

    try:
        await _ensure_server_disk_quota(server_id, data_dir, disk_limit_mb, allow_init=allow_init_disk)
    except Exception as e:
        await log(f'Disk quota init/mount error: server_id={server_id} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

    cpu_cores, mem_mb = _parse_resource_limits(data)
    cpu_shares = _parse_cpu_shares(data)

    def _is_effectively_empty_dir(path: str) -> bool:
        try:
            if not os.path.isdir(path):
                return False
            entries = [e for e in os.listdir(path) if e not in ('lost+found',)]
            return len(entries) == 0
        except Exception:
            return False

    try:
        steam_app_id_int = int(steam_app_id) if steam_app_id is not None and str(steam_app_id).strip() != '' else 0
    except Exception:
        steam_app_id_int = 0

    wants_version = (isinstance(archive_url, str) and str(archive_url).strip() != '') or steam_app_id_int > 0
    try:
        await log(
            f'Reinstall wants_version={wants_version} server_id={server_id} '
            f'archive_url={(str(archive_url)[:180] if archive_url is not None else "")} '
            f'steam_app_id={steam_app_id_int} data_dir={data_dir}'
        )
    except Exception:
        pass

    if wants_version:
        try:
            await log(f'Reinstall wiping data dir: server_id={server_id} data_dir={data_dir} keep_lost_found=1')
            await _wipe_data_dir_contents(server_id, data_dir, keep_lost_found=True)
            await log(f'Reinstall wipe complete: server_id={server_id}')
        except Exception:
            pass

    try:
        is_empty = _is_effectively_empty_dir(data_dir)
    except Exception:
        is_empty = False

    if wants_version and is_empty and (not isinstance(archive_url, str) or archive_url.strip() == '') and steam_app_id_int <= 0:
        return web.json_response({'ok': False, 'error': 'archive_url or steam_app_id is required for reinstall in version-only mode'}, status=400)

    # Steam source: install with steamcmd when version source is provided.
    if wants_version and steam_app_id_int > 0:
        try:
            await _ensure_steamcmd_host_deps()
            await steam_install(
                server_id=server_id,
                game=game,
                data_dir=data_dir,
                app_id=steam_app_id_int,
                branch=str(steam_branch or ''),
            )
        except Exception as e:
            await log(f'Steam install error: server_id={server_id} app_id={steam_app_id_int} error={e}')
            return web.json_response({'ok': False, 'error': str(e)}, status=500)

    # Archive source: bootstrap /data from the archive when version source is provided.
    if wants_version and steam_app_id_int <= 0 and isinstance(archive_url, str) and archive_url.strip() != '':
        try:
            await archive_bootstrap(server_id=server_id, game=game, data_dir=data_dir, url=archive_url.strip(), version_name=version_name)
        except ValueError as e:
            return web.json_response({'ok': False, 'error': str(e)}, status=400)
        except Exception as e:
            await log(f'Archive bootstrap error: server_id={server_id} url={archive_url.strip()} error={e}')
            return web.json_response({'ok': False, 'error': str(e)}, status=500)

    if game in RUST_GAME_CODES:
        try:
            await _ensure_rust_maxplayers_env(data_dir, int(data.get('slots') or 0))
        except Exception:
            pass

    try:
        flags = _docker_limit_flags(cpu_cores, mem_mb, cpu_shares, True)
        flags2 = _docker_limit_flags(cpu_cores, mem_mb, cpu_shares, False)

        env_var = _get_env_var(game, port, data)
        ports_flag = _get_ports_flag(game, port)

        container_id = await _docker_run_detached(
            container_name=container_name,
            image=image,
            data_dir=data_dir,
            ports_flag=ports_flag,
            env_var=env_var,
            flags=flags,
            fallback_flags=flags2,
            mem_mb=mem_mb,
        )

        await log(f'Server reinstalled: {container_name} id={container_id}')
        status, _cpu_limit, _mem_limit_mb = await _docker_container_inspect_summary(container_name)
        return web.json_response({
            'ok': True,
            'container_id': container_id.strip(),
            **status,
        })
    except Exception as e:
        await log(f'Server reinstall error: {e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)


async def delete_server_handler(request):
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

    try:
        game = _require_game(data)
    except ValueError as e:
        return web.json_response({'ok': False, 'error': str(e)}, status=400)
    container_name = _get_container_name(game, server_id)
    await log(f'Server delete requested: server_id={server_id} game={game or 'samp'} container={container_name}')

    try:
        output_parts = []

        try:
            output = await _docker_rm_force(container_name, with_volumes=False)
            if output:
                output_parts.append(output)
        except Exception as e:
            msg = str(e)
            if msg:
                output_parts.append(msg)

        data_dir = _server_data_dir(server_id)
        remove_error = None
        try:
            _assert_safe_data_dir(server_id, data_dir)
            base_dir = os.path.dirname(data_dir)
            # If disk quota is used, data_dir may be a mounted loopback filesystem.
            # Need to unmount and detach loop device, otherwise /dev/loop* will remain.
            async def _try_unmount(target_path: str) -> None:
                try:
                    src = await run_cmd(f'findmnt -n -o SOURCE --target {shlex.quote(target_path)}')
                    src = (src or '').strip()
                    if src == '':
                        return

                    try:
                        await run_cmd(f'umount {shlex.quote(target_path)}')
                        output_parts.append(f'umounted: {target_path}')
                    except Exception:
                        await run_cmd(f'umount -l {shlex.quote(target_path)}')
                        output_parts.append(f'umounted (lazy): {target_path}')

                    if src.startswith('/dev/loop'):
                        try:
                            await run_cmd(f'losetup -d {shlex.quote(src)}')
                            output_parts.append(f'loop detached: {src}')
                        except Exception as e:
                            output_parts.append(str(e))
                except Exception:
                    return

            # Unmount both the data dir and the server base dir (some setups mount at base dir)
            await _try_unmount(data_dir)
            await _try_unmount(base_dir)

            if os.path.exists(base_dir):
                try:
                    shutil.rmtree(base_dir)
                except Exception as e:
                    raise RuntimeError(f'Failed to remove server dir {base_dir}: {e}')

            if os.path.exists(base_dir):
                raise RuntimeError(f'Server dir still exists after delete: {base_dir}')

            output_parts.append(f'data removed: {base_dir}')
        except Exception as e:
            remove_error = str(e)
            output_parts.append(remove_error)

        if remove_error:
            return web.json_response({'ok': False, 'error': remove_error, 'output': '\n'.join([p for p in output_parts if p])}, status=500)

        return web.json_response({'ok': True, 'output': '\n'.join([p for p in output_parts if p])})
    except Exception as e:
        await log(f'Server delete error: {e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)
