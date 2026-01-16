import asyncio
from typing import List


async def run_cmd(cmd: str) -> str:
    proc = await asyncio.create_subprocess_shell(
        cmd,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
    )
    stdout, stderr = await proc.communicate()
    if proc.returncode == 0:
        return stdout.decode('utf-8').strip()
    out_s = (stdout or b'').decode('utf-8', errors='ignore').strip()
    err_s = (stderr or b'').decode('utf-8', errors='ignore').strip()
    msg = err_s or out_s or 'Command failed'
    if out_s and err_s and out_s != err_s:
        msg = f'{err_s}\n{out_s}'
    raise Exception(msg)


async def run_argv(argv: List[str], timeout_sec: float = 10.0, max_output_bytes: int = 20000) -> str:
    if not isinstance(argv, list) or len(argv) == 0 or not all(isinstance(x, str) for x in argv):
        raise Exception('argv must be non-empty list[str]')

    proc = await asyncio.create_subprocess_exec(
        *argv,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
    )

    async def _read_limited(stream: asyncio.StreamReader) -> bytes:
        buf = bytearray()
        while True:
            chunk = await stream.read(4096)
            if not chunk:
                break
            remain = max_output_bytes - len(buf)
            if remain <= 0:
                break
            if len(chunk) > remain:
                buf.extend(chunk[:remain])
                break
            buf.extend(chunk)
        return bytes(buf)

    try:
        stdout_task = asyncio.create_task(_read_limited(proc.stdout))
        stderr_task = asyncio.create_task(_read_limited(proc.stderr))
        await asyncio.wait_for(proc.wait(), timeout=timeout_sec)
        stdout = await stdout_task
        stderr = await stderr_task
    except asyncio.TimeoutError:
        try:
            proc.kill()
        except Exception:
            pass
        raise Exception('Command timeout')

    out_s = (stdout or b'').decode('utf-8', errors='ignore').strip()
    err_s = (stderr or b'').decode('utf-8', errors='ignore').strip()

    if proc.returncode == 0:
        return out_s
    raise Exception(err_s or out_s or f'Command failed: exit {proc.returncode}')
