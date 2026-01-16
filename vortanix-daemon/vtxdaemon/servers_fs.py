import asyncio
import os
import shlex
import shutil

from vtxdaemon.cmd import run_cmd
from vtxdaemon.servers_common import _assert_safe_data_dir
from vtxdaemon.servers_game import _normalize_game


async def _wipe_data_dir_contents(server_id: int, data_dir: str, keep_lost_found: bool) -> None:
    _assert_safe_data_dir(server_id, data_dir)

    def _run() -> None:
        if not os.path.isdir(data_dir):
            return

        root_real = os.path.realpath(data_dir)
        for name in os.listdir(data_dir):
            if keep_lost_found and name == 'lost+found':
                continue

            target = os.path.join(data_dir, name)
            target_real = os.path.realpath(target)
            if target_real != root_real and not target_real.startswith(root_real + os.sep):
                raise RuntimeError('Unsafe path while wiping data_dir')

            if os.path.isdir(target) and not os.path.islink(target):
                shutil.rmtree(target)
            else:
                os.remove(target)

    loop = asyncio.get_event_loop()
    await loop.run_in_executor(None, _run)


async def _post_archive_fix_permissions(game: str, data_dir: str) -> None:
    q_data_dir = shlex.quote(data_dir)
    game_l = _normalize_game(game)

    binaries_by_game = {
        'samp': ['samp03svr', 'samp-npc'],
        'crmp': ['samp03svr-cr', 'crmp_server', 'cr-mp-server', 'samp03svr'],
        'cs16': ['hlds_linux', 'hlds_run'],
        'css': ['srcds_run', 'srcds_linux'],
        'tf2': ['srcds_run', 'srcds_linux'],
        'gmod': ['srcds_run', 'srcds_linux'],
        'cs2': ['cs2', 'game/bin/linuxsteamrt64/cs2'],
        'rust': ['RustDedicated'],
        'unturned': ['Unturned_Headless.x86_64', 'Unturned.x86_64', 'Unturned'],
        'mta': ['mta-server64', 'mta-server', 'x64/mta-server64', 'server/mta-server64'],
    }

    bins = binaries_by_game.get(game_l, [])

    parts = [
        f'cd {q_data_dir} 2>/dev/null || exit 0',
        'find . -maxdepth 3 -type f -name "*.sh" -exec chmod +x {} \\; 2>/dev/null || true',
        'find . -maxdepth 3 -type f -name "*.x86_64" -exec chmod +x {} \\; 2>/dev/null || true',
        'find . -maxdepth 3 -type f -name "*.bin" -exec chmod +x {} \\; 2>/dev/null || true',
        'find . -maxdepth 3 -type f -name "*.run" -exec chmod +x {} \\; 2>/dev/null || true',
    ]
    for b in bins:
        qb = shlex.quote(b)
        parts.append(f'if [ -f {qb} ]; then chmod +x {qb} 2>/dev/null || true; fi')

    for b in ('srcds_run', 'srcds_linux', 'hlds_run', 'hlds_linux', 'RustDedicated'):
        qb = shlex.quote(b)
        parts.append(f'if [ -f {qb} ]; then chmod +x {qb} 2>/dev/null || true; fi')

    try:
        await run_cmd(' ; '.join(parts))
    except Exception:
        pass
