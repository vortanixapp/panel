import shlex

from vtxdaemon.cmd import run_cmd


async def _ensure_docker_available() -> None:
    await run_cmd('docker --version')


async def _container_id_by_name(container_name: str) -> str:
    return await run_cmd(f"docker ps -a --filter name={shlex.quote(container_name)} --format '{{{{.ID}}}}'")


async def _docker_rm_force(container_name: str, with_volumes: bool = False) -> str:
    q = shlex.quote(container_name)
    if with_volumes:
        return await run_cmd(f'docker rm -f -v {q}')
    return await run_cmd(f'docker rm -f {q}')


async def _docker_start(container_name: str) -> str:
    return await run_cmd(f'docker start {shlex.quote(container_name)}')


async def _docker_stop(container_name: str) -> str:
    return await run_cmd(f'docker stop {shlex.quote(container_name)}')


async def _docker_restart(container_name: str) -> str:
    return await run_cmd(f'docker restart {shlex.quote(container_name)}')


async def _docker_logs(container_name: str, tail: int) -> str:
    return await run_cmd(f'docker logs --tail {int(tail)} {shlex.quote(container_name)}')


async def _docker_stats(container_name: str) -> str:
    q_container = shlex.quote(container_name)
    return await run_cmd(f"docker stats --no-stream --format '{{{{.CPUPerc}}}}|{{{{.MemPerc}}}}' {q_container}")


async def _docker_run_detached(
    *,
    container_name: str,
    image: str,
    data_dir: str,
    ports_flag: str,
    env_var: str,
    flags: str,
    fallback_flags: str,
    mem_mb,
) -> str:
    q_image = shlex.quote(image)
    cmd = f'docker run -d --name {shlex.quote(container_name)} --restart unless-stopped'
    if flags:
        cmd += f' {flags}'
    cmd += f' -v {shlex.quote(data_dir)}:/data {ports_flag} {env_var} {q_image}'

    try:
        return await run_cmd(cmd)
    except Exception as e:
        msg = str(e).lower()
        if ('pull access denied' in msg) or ('repository does not exist' in msg) or ('manifest' in msg and 'not found' in msg) or ('not found' in msg and 'manifest' in msg):
            raise Exception(
                f'Docker image not available: {image}. '
                'The repository/tag may be missing or private. '
                'Build/pull images on the location (Admin -> Location -> Images) or run docker login if registry requires auth.'
            )

        if mem_mb is not None and ('swap limit capabilities' in msg or 'swapaccount' in msg) and fallback_flags:
            cmd2 = (
                f'docker run -d --name {shlex.quote(container_name)} --restart unless-stopped '
                f'{fallback_flags} -v {shlex.quote(data_dir)}:/data {ports_flag} {env_var} {q_image}'
            )
            return await run_cmd(cmd2)

        raise
