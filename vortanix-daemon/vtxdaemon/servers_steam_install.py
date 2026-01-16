import os
import shlex

from vtxdaemon.cmd import run_cmd
from vtxdaemon.logbuf import log
from vtxdaemon.servers_game import CS16_GAME_CODES, CS2_GAME_CODES


def _tail_file(path: str, max_bytes: int = 12000) -> str:
    try:
        if not os.path.exists(path):
            return ''
        with open(path, 'rb') as f:
            try:
                f.seek(0, os.SEEK_END)
                size = f.tell()
                start = max(0, size - max_bytes)
                f.seek(start, os.SEEK_SET)
            except Exception:
                pass
            data = f.read(max_bytes)
        return (data or b'').decode('utf-8', errors='ignore').strip()
    except Exception:
        return ''


async def _ensure_steamcmd() -> str:
    base = '/opt/vortanix/steamcmd'
    try:
        await run_cmd(f'mkdir -p {shlex.quote(base)}')
    except Exception:
        pass

    steamcmd_sh = os.path.join(base, 'steamcmd.sh')
    if os.path.exists(steamcmd_sh):
        return steamcmd_sh

    await log('Downloading steamcmd...')
    q_base = shlex.quote(base)
    await run_cmd(f'curl -fsSL https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz -o {q_base}/steamcmd.tar.gz')
    await run_cmd(f'tar -xzf {q_base}/steamcmd.tar.gz -C {q_base}/')
    await run_cmd(f'rm -f {q_base}/steamcmd.tar.gz')
    await run_cmd(f'chmod +x {shlex.quote(steamcmd_sh)}')
    return steamcmd_sh


async def steam_install(
    *,
    server_id: int,
    game: str,
    data_dir: str,
    app_id: int,
    branch: str,
) -> None:
    """Install server files via SteamCMD into data_dir.

    Keeps behavior compatible with original servers_runtime.py implementation.
    """
    steamcmd_sh = await _ensure_steamcmd()

    home = os.path.join(data_dir, '.steamcmd')
    q_home = shlex.quote(home)
    q_data_dir = shlex.quote(data_dir)
    await run_cmd(f'mkdir -p {q_home}')

    # We always use anonymous login.
    login = '+login anonymous'

    game_l = str(game or '').lower().strip()

    branch = str(branch or '').strip()
    beta = f'-beta {shlex.quote(branch)}' if branch != '' else ''

    await log(f'Installing via steamcmd: server_id={server_id} app_id={app_id} branch={branch}')
    cmd = (
        f'HOME={q_home} {shlex.quote(steamcmd_sh)} '
        f'+@ShutdownOnFailedCommand 1 +@NoPromptForPassword 1 '
        f'+force_install_dir {q_data_dir} {login} +app_info_update 1 +app_update {app_id} {beta} validate +quit'
    )

    stderr_log = os.path.join(home, 'Steam', 'logs', 'stderr.txt')
    content_log = os.path.join(home, 'Steam', 'logs', 'content_log.txt')

    async def _run_with_logs() -> None:
        try:
            await run_cmd(cmd)
            return
        except Exception as e:
            stderr_tail = _tail_file(stderr_log)
            content_tail = _tail_file(content_log)

            extra = ''
            if stderr_tail != '':
                extra += f"\n--- steamcmd stderr.txt (tail) ---\n{stderr_tail}"
            if content_tail != '':
                extra += f"\n--- steamcmd content_log.txt (tail) ---\n{content_tail}"
            msg = f'{e}{extra}' if extra != '' else str(e)
            raise Exception(msg)

    try:
        await _run_with_logs()
    except Exception as e:
        # SteamCMD sometimes fails with "Missing configuration" due to broken appcache/appinfo.
        # If detected, clear per-server Steam appcache and retry once.
        s = str(e)
        if 'Missing configuration' in s:
            try:
                await log(f'SteamCMD missing configuration; clearing appcache and retrying: server_id={server_id}')
                q_appcache = shlex.quote(os.path.join(home, 'Steam', 'appcache'))
                await run_cmd(f'rm -rf {q_appcache}')
            except Exception:
                pass
            await _run_with_logs()
            # success after retry
        else:
            raise

    # Legacy post-fixups
    if game_l in CS16_GAME_CODES:
        q_base = shlex.quote('/opt/vortanix/steamcmd')
        await run_cmd(
            'sh -c '
            + shlex.quote(
                f'if [ ! -f {q_data_dir}/libsteam_api.so ]; then '
                f'  p=$(find {q_data_dir} -maxdepth 8 -name libsteam_api.so -type f 2>/dev/null | head -n1); '
                f'  if [ "$p" = "" ]; then p=$(find {q_base} -maxdepth 10 -name libsteam_api.so -type f 2>/dev/null | head -n1); fi; '
                f'  if [ "$p" != "" ]; then cp -fL "$p" {q_data_dir}/libsteam_api.so; fi; '
                f'fi; '
                f'if [ -e {q_data_dir}/libsteam_api.so ]; then cp -fL {q_data_dir}/libsteam_api.so {q_data_dir}/libsteam_api.so 2>/dev/null || true; chmod 644 {q_data_dir}/libsteam_api.so 2>/dev/null || true; fi; '
                f'if [ -f {q_data_dir}/hlds_linux ]; then chmod +x {q_data_dir}/hlds_linux 2>/dev/null || true; fi; '
                f'if [ -f {q_data_dir}/hlds_run ]; then chmod +x {q_data_dir}/hlds_run 2>/dev/null || true; fi'
            )
        )

    if game_l in CS2_GAME_CODES:
        q_base = shlex.quote('/opt/vortanix/steamcmd')
        await run_cmd(
            'sh -c '
            + shlex.quote(
                f'mkdir -p {q_data_dir}/.steam/sdk64 2>/dev/null || true; '
                f'if [ ! -f {q_data_dir}/.steam/sdk64/steamclient.so ]; then '
                f'  p=$(find {q_data_dir}/game {q_data_dir} {q_data_dir}/.steamcmd {q_data_dir}/.steam {q_data_dir}/Steam -maxdepth 25 -name steamclient.so -type f 2>/dev/null | head -n1); '
                f'  if [ "$p" = "" ]; then p=$(find {q_base} -maxdepth 14 -name steamclient.so -type f 2>/dev/null | head -n1); fi; '
                f'  if [ "$p" != "" ]; then cp -fL "$p" {q_data_dir}/.steam/sdk64/steamclient.so; chmod 644 {q_data_dir}/.steam/sdk64/steamclient.so 2>/dev/null || true; fi; '
                f'fi'
            )
        )
