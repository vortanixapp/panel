import os
import shlex
from typing import Optional

from vtxdaemon.cmd import run_cmd


SAMP_GAME_CODES = ('samp', 'sa-mp', 'samp03', 'gta_samp')
CRMP_GAME_CODES = ('crmp', 'gta_crmp', 'cr-mp', 'cr-mp0.3')
MTA_GAME_CODES = ('mta', 'mtasa', 'mta-sa', 'mta_sa')
CS16_GAME_CODES = ('cs16', 'counter-strike', 'counter_strike', 'cs_1_6', 'cstrike')
CSS_GAME_CODES = ('css', 'cs:s', 'cs_source', 'counter-strike_source', 'counter_strike_source')
CS2_GAME_CODES = ('cs2', 'counter-strike2', 'counter_strike2', 'counter-strike_2', 'counter_strike_2')
RUST_GAME_CODES = ('rust',)
TF2_GAME_CODES = ('tf2', 'teamfortress2', 'team_fortress_2', 'tf')
GMOD_GAME_CODES = ('gmod', 'garrysmod', "garry's mod", 'garrys_mod')
MC_JAVA_GAME_CODES = ('mcjava', 'mcpaper', 'mcspigot', 'mcforge', 'mcfabric')
MC_BEDROCK_GAME_CODES = ('mcbedrock', 'mcbedrk', 'bedrock')
UNTURNED_GAME_CODES = ('unturned', 'unturn', 'ut', 'untrm4', 'untrm5')
SUPPORTED_GAME_CODES = SAMP_GAME_CODES + CRMP_GAME_CODES + MTA_GAME_CODES + CS16_GAME_CODES + CSS_GAME_CODES + CS2_GAME_CODES + RUST_GAME_CODES + TF2_GAME_CODES + GMOD_GAME_CODES + MC_JAVA_GAME_CODES + MC_BEDROCK_GAME_CODES + UNTURNED_GAME_CODES


def _normalize_game(game: str) -> str:
    return str(game or '').lower().strip()


def _require_game(data: dict) -> str:
    game = _normalize_game(data.get('game') or data.get('game_code') or '')
    if game == '':
        raise ValueError('game is required')
    if game not in SUPPORTED_GAME_CODES:
        raise ValueError(f'Unsupported game: {game}')
    return game


def _get_container_name(game: str, server_id: int) -> str:
    """Determine container name based on game type."""
    game = _normalize_game(game)
    if game in CRMP_GAME_CODES:
        return f'vtx-crmp-{server_id}'
    if game in MTA_GAME_CODES:
        return f'vtx-mta-{server_id}'
    if game in CS16_GAME_CODES:
        return f'vtx-cs16-{server_id}'
    if game in CSS_GAME_CODES:
        return f'vtx-css-{server_id}'
    if game in CS2_GAME_CODES:
        return f'vtx-cs2-{server_id}'
    if game in RUST_GAME_CODES:
        return f'vtx-rust-{server_id}'
    if game in TF2_GAME_CODES:
        return f'vtx-tf2-{server_id}'
    if game in GMOD_GAME_CODES:
        return f'vtx-gmod-{server_id}'
    if game in MC_JAVA_GAME_CODES or game in MC_BEDROCK_GAME_CODES:
        return f'vtx-mc-{server_id}'
    if game in UNTURNED_GAME_CODES:
        return f'vtx-unturned-{server_id}'
    else:
        return f'vtx-samp-{server_id}'


def _get_env_var(game: str, port: int, data: dict) -> str:
    """Determine environment variable based on game type."""
    game = _normalize_game(game)
    if game in CS16_GAME_CODES:
        return f'-e CS16_PORT={port}'
    if game in CSS_GAME_CODES:
        return f'-e CSS_PORT={port}'
    if game in CS2_GAME_CODES:
        maxplayers = None
        try:
            maxplayers = int(data.get('slots') or 0)
        except Exception:
            maxplayers = None
        extra = ''
        if maxplayers is not None and maxplayers > 0:
            extra = f' -e CS2_MAXPLAYERS={maxplayers}'
        return f'-e CS2_PORT={port}{extra}'
    if game in RUST_GAME_CODES:
        return f'-e RUST_PORT={port}'
    if game in TF2_GAME_CODES:
        return f'-e TF2_PORT={port}'
    if game in GMOD_GAME_CODES:
        return f'-e GMOD_PORT={port}'
    if game in CRMP_GAME_CODES:
        return f'-e CRMP_PORT={port}'
    if game in MTA_GAME_CODES:
        return f'-e MTA_PORT={port}'
    if game in MC_JAVA_GAME_CODES:
        type_map = {
            'mcjava': 'VANILLA',
            'mcpaper': 'PAPER',
            'mcspigot': 'SPIGOT',
            'mcforge': 'FORGE',
            'mcfabric': 'FABRIC',
        }
        mc_type = type_map.get(game, 'VANILLA')
        return f'-e EULA=TRUE -e TYPE={shlex.quote(mc_type)} -e SERVER_PORT={port}'
    if game in MC_BEDROCK_GAME_CODES:
        return f'-e EULA=TRUE -e SERVER_PORT={port}'
    if game in UNTURNED_GAME_CODES:
        server_type = ''
        if game == 'untrm4':
            server_type = 'rm4'
        elif game == 'untrm5':
            server_type = 'rm5'
        env = f'-e UNTURNED_PORT={port}'
        if server_type != '':
            env += f' -e SERVER_TYPE={shlex.quote(server_type)}'
        return env
    else:
        return f'-e SAMP_PORT={port}'


def _get_ports_flag(game: str, port: int) -> str:
    game = _normalize_game(game)
    if game in MC_JAVA_GAME_CODES:
        return f'-p {port}:{port}/tcp'
    if game in MC_BEDROCK_GAME_CODES:
        return f'-p {port}:{port}/udp'
    if game in UNTURNED_GAME_CODES:
        qport = port + 1
        return f'-p {port}:{port}/udp -p {qport}:{qport}/udp'

    if game in MTA_GAME_CODES:
        http_port = port + 2
        ase_port = port + 123
        return f'-p {port}:{port}/udp -p {ase_port}:{ase_port}/udp -p {http_port}:{http_port}/tcp'

    if game in RUST_GAME_CODES:
        qport = port + 1
        return f'-p {port}:{port}/udp -p {qport}:{qport}/udp -p {qport}:{qport}/tcp'
    ports_flag = f'-p {port}:{port}/udp'
    if game in CS16_GAME_CODES or game in CSS_GAME_CODES or game in CS2_GAME_CODES or game in TF2_GAME_CODES or game in GMOD_GAME_CODES:
        ports_flag += f' -p {port}:{port}/tcp'
    return ports_flag


async def _ensure_rust_maxplayers_env(data_dir: str, slots: int) -> None:
    """Ensure /data/rust.env contains RUST_MAXPLAYERS enforced by tariff slots.

    We preserve other lines (including comments) and only upsert RUST_MAXPLAYERS.
    """
    try:
        slots_int = int(slots)
    except Exception:
        return
    if slots_int <= 0:
        return

    path = os.path.join(str(data_dir), 'rust.env')
    desired_line = f'RUST_MAXPLAYERS="{slots_int}"\n'

    try:
        if not os.path.exists(path):
            with open(path, 'w', encoding='utf-8') as f:
                f.write(desired_line)
            return

        with open(path, 'r', encoding='utf-8', errors='ignore') as f:
            raw = f.read()

        lines = raw.splitlines(True)
        out = []
        replaced = False
        for line in lines:
            s = line.lstrip()
            if s.startswith('RUST_MAXPLAYERS=') or s.startswith('export RUST_MAXPLAYERS='):
                if not replaced:
                    out.append(desired_line)
                    replaced = True
                continue
            out.append(line)

        if not replaced:
            if len(out) > 0 and not out[-1].endswith('\n'):
                out[-1] = out[-1] + '\n'
            out.append(desired_line)

        new_raw = ''.join(out)
        if new_raw != raw:
            with open(path, 'w', encoding='utf-8') as f:
                f.write(new_raw)
    except Exception:
        # Never fail server lifecycle due to env file patching.
        return


async def _ensure_cs2_steamclient(data_dir: str) -> None:
    """Ensure /data/.steam/sdk64/steamclient.so exists for CS2 by copying from host SteamCMD."""
    q_data_dir = shlex.quote(data_dir)
    src = '/opt/vortanix/steamcmd/linux64/steamclient.so'
    q_src = shlex.quote(src)
    await run_cmd(
        'sh -c '
        + shlex.quote(
            f'mkdir -p {q_data_dir}/.steam/sdk64 2>/dev/null || true; '
            f'if [ ! -f {q_data_dir}/.steam/sdk64/steamclient.so ] && [ -f {q_src} ]; then '
            f'  cp -fL {q_src} {q_data_dir}/.steam/sdk64/steamclient.so; '
            f'  chmod 644 {q_data_dir}/.steam/sdk64/steamclient.so 2>/dev/null || true; '
            f'fi'
        )
    )
