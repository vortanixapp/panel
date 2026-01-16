import asyncio
import socket
import struct
from typing import Any, Dict, List, Optional, Tuple

from aiohttp import web


def _normalize_game(game: str) -> str:
    return str(game or '').lower().strip()


SOURCE_GAMES = {
    'cs16', 'counter-strike', 'counter_strike', 'cs_1_6', 'cstrike',
    'css', 'cs:s', 'cs_source', 'counter-strike_source', 'counter_strike_source',
    'cs2', 'counter-strike2', 'counter_strike2', 'counter-strike_2', 'counter_strike_2',
    'tf2', 'teamfortress2', 'team_fortress_2', 'tf',
    'gmod', 'garrysmod', "garry's mod", 'garrys_mod',
}

SAMP_GAMES = {'samp', 'sa-mp', 'samp03', 'gta_samp', 'crmp', 'gta_crmp', 'cr-mp', 'cr-mp0.3'}


def _read_cstr(buf: bytes, offset: int) -> Tuple[str, int]:
    end = buf.find(b'\x00', offset)
    if end < 0:
        return '', len(buf)
    try:
        return buf[offset:end].decode('utf-8', errors='ignore'), end + 1
    except Exception:
        return '', end + 1


def _query_source_udp(host: str, port: int, timeout_sec: float = 1.2) -> Tuple[Optional[int], Optional[int]]:
    pkt = b'\xFF\xFF\xFF\xFFTSource Engine Query\x00'

    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    s.settimeout(timeout_sec)
    try:
        s.sendto(pkt, (host, port))
        data, _ = s.recvfrom(4096)
    finally:
        try:
            s.close()
        except Exception:
            pass

    if not data or len(data) < 6:
        return None, None

    if not data.startswith(b'\xFF\xFF\xFF\xFF'):
        return None, None

    t = data[4]
    if t != 0x49:
        return None, None

    off = 5
    if off >= len(data):
        return None, None

    off += 1
    _, off = _read_cstr(data, off)
    _, off = _read_cstr(data, off)
    _, off = _read_cstr(data, off)
    _, off = _read_cstr(data, off)

    if off + 2 > len(data):
        return None, None
    off += 2

    if off + 3 > len(data):
        return None, None

    players = data[off]
    max_players = data[off + 1]

    return int(players), int(max_players)


def _query_samp_udp(host: str, port: int, timeout_sec: float = 1.2) -> Tuple[Optional[int], Optional[int]]:
    ip_parts = host.strip().split('.')
    if len(ip_parts) != 4:
        return None, None

    ip_bytes = bytes([int(p) & 0xFF for p in ip_parts])
    port_bytes = bytes([port & 0xFF, (port >> 8) & 0xFF])

    pkt = b'SAMP' + ip_bytes + port_bytes + b'i'

    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    s.settimeout(timeout_sec)
    try:
        s.sendto(pkt, (host, port))
        data, _ = s.recvfrom(4096)
    finally:
        try:
            s.close()
        except Exception:
            pass

    if not data or len(data) < 16:
        return None, None

    if not data.startswith(b'SAMP'):
        return None, None

    if len(data) < 11 + 1 + 2 + 2:
        return None, None

    off = 11
    try:
        _passworded = data[off]
        players, max_players = struct.unpack_from('<HH', data, off + 1)
        return int(players), int(max_players)
    except Exception:
        return None, None


def _query_one(game: str, host: str, port: int) -> Tuple[Optional[int], Optional[int], Optional[str]]:
    g = _normalize_game(game)
    if port <= 0:
        return None, None, 'Invalid port'

    if g in SOURCE_GAMES:
        try:
            o, m = _query_source_udp(host, port)
            return o, m, None
        except Exception as e:
            return None, None, str(e)

    if g in SAMP_GAMES:
        try:
            o, m = _query_samp_udp(host, port)
            return o, m, None
        except Exception as e:
            return None, None, str(e)

    return None, None, None


async def server_online_batch_handler(request: web.Request) -> web.Response:
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    items = data.get('items')
    if not isinstance(items, list):
        return web.json_response({'ok': False, 'error': 'items is required (array)'}, status=400)

    host = str(data.get('host') or '127.0.0.1').strip() or '127.0.0.1'

    loop = asyncio.get_event_loop()

    async def run_item(it: Any) -> Dict[str, Any]:
        if not isinstance(it, dict):
            return {'ok': False, 'error': 'Invalid item'}

        server_id = it.get('server_id')
        try:
            server_id = int(server_id)
        except Exception:
            server_id = None

        game = str(it.get('game') or it.get('game_code') or '').strip()
        try:
            port = int(it.get('port') or 0)
        except Exception:
            port = 0

        if server_id is None or port <= 0 or game.strip() == '':
            return {
                'server_id': server_id,
                'ok': False,
                'error': 'server_id/game/port required',
                'online': None,
                'max': None,
            }

        online, max_players, err = await loop.run_in_executor(None, _query_one, game, host, port)

        payload: Dict[str, Any] = {
            'server_id': int(server_id),
            'ok': True,
            'online': online,
            'max': max_players,
        }
        if err:
            payload['ok'] = False
            payload['error'] = err

        return payload

    tasks = [run_item(it) for it in items]
    results = await asyncio.gather(*tasks)

    return web.json_response({'ok': True, 'items': results})
