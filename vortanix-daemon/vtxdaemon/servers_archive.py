import shlex

from vtxdaemon.cmd import run_cmd
from vtxdaemon.logbuf import log
from vtxdaemon.servers_fs import _post_archive_fix_permissions


async def archive_bootstrap(*, server_id: int, game: str, data_dir: str, url: str, version_name: str | None = None) -> None:
    url = str(url or '').strip()
    if not (url.startswith('http://') or url.startswith('https://')):
        raise ValueError('archive_url must be http(s) URL')
    if not url.lower().endswith('.zip'):
        raise ValueError('archive_url must point to a .zip archive')

    await log(f'Bootstrapping server files from archive: server_id={server_id} game={game} version={version_name or ""} url={url}')

    tmp_base = f'/tmp/vtx-archive-{server_id}'
    q_tmp_base = shlex.quote(tmp_base)
    q_url = shlex.quote(url)
    q_data_dir = shlex.quote(data_dir)

    await run_cmd(f'rm -rf {q_tmp_base} >/dev/null 2>&1 || true')
    await run_cmd(f'mkdir -p {q_tmp_base}')

    await log(f'Archive download start: server_id={server_id}')
    await run_cmd(f'curl -fsSL {q_url} -o {q_tmp_base}/archive.zip')
    await log(f'Archive download done: server_id={server_id}')

    await run_cmd(f'mkdir -p {q_tmp_base}/extract')
    try:
        await run_cmd('unzip -v >/dev/null 2>&1')
    except Exception:
        raise Exception('unzip is required to extract .zip archives')

    await log(f'Archive unzip start: server_id={server_id}')
    await run_cmd(f'unzip -q {q_tmp_base}/archive.zip -d {q_tmp_base}/extract')
    await log(f'Archive unzip done: server_id={server_id}')

    await run_cmd(
        f'if [ "$(find {q_tmp_base}/extract -mindepth 1 -maxdepth 1 | wc -l)" = "1" ] && '
        f'[ "$(find {q_tmp_base}/extract -mindepth 1 -maxdepth 1 -type d | wc -l)" = "1" ]; then '
        f'  d=$(find {q_tmp_base}/extract -mindepth 1 -maxdepth 1 -type d | head -n1); '
        f'  cp -a "$d"/. {q_data_dir}/; '
        f'else '
        f'  cp -a {q_tmp_base}/extract/. {q_data_dir}/; '
        f'fi'
    )

    await _post_archive_fix_permissions(game, data_dir)
    await log(f'Archive copied into data_dir: server_id={server_id}')

    await run_cmd(f'rm -rf {q_tmp_base} >/dev/null 2>&1 || true')
