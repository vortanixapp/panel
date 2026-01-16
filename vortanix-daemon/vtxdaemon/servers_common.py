import json
import os
import re
import shlex
from typing import Optional, List, Tuple

from vtxdaemon.cmd import run_cmd


SERVERS_BASE_DIR = os.getenv('SERVERS_BASE_DIR', '/opt/vortanix/servers')
FASTDL_BASE_DIR = os.getenv('FASTDL_BASE_DIR', '/opt/vortanix/fastdl')
FTP_GROUP = os.getenv('VORTANIX_FTP_GROUP', 'vtxsrv')
FTP_PASV_MIN_PORT = int(os.getenv('VORTANIX_FTP_PASV_MIN_PORT', '21100'))
FTP_PASV_MAX_PORT = int(os.getenv('VORTANIX_FTP_PASV_MAX_PORT', '21110'))
MYSQL_PORT = int(os.getenv('VORTANIX_MYSQL_PORT', '3306'))
MYSQL_BIND_ADDRESS = os.getenv('VORTANIX_MYSQL_BIND_ADDRESS', '0.0.0.0')
MYSQL_CONFIG_FILE = os.getenv('VORTANIX_MYSQL_CONFIG_FILE', '/etc/mysql/mysql.conf.d/mysqld.cnf')


def _server_data_dir(server_id: int) -> str:
    return os.path.join(SERVERS_BASE_DIR, str(server_id), 'data')


def _fastdl_dir(server_id: int) -> str:
    return os.path.join(FASTDL_BASE_DIR, str(server_id))


def _assert_safe_data_dir(server_id: int, data_dir: str) -> None:
    base_real = os.path.realpath(SERVERS_BASE_DIR)
    expected_real = os.path.join(base_real, str(server_id), 'data')
    actual_real = os.path.realpath(data_dir)
    if actual_real != expected_real:
        raise RuntimeError('Unsafe data_dir')


async def _ensure_server_data_dir(server_id: int) -> str:
    data_dir = _server_data_dir(server_id)
    q_dir = shlex.quote(data_dir)

    await run_cmd(f'getent group {shlex.quote(FTP_GROUP)} >/dev/null 2>&1 || groupadd -r {shlex.quote(FTP_GROUP)}')

    await run_cmd(f'mkdir -p {q_dir}')
    _assert_safe_data_dir(server_id, data_dir)
    try:
        st = os.stat(data_dir)
        is_root_owner = (st.st_uid == 0)
    except Exception:
        is_root_owner = True

    if is_root_owner:
        await run_cmd(f'chown root:{shlex.quote(FTP_GROUP)} {q_dir} >/dev/null 2>&1 || true')
    else:
        await run_cmd(f'chgrp {shlex.quote(FTP_GROUP)} {q_dir} >/dev/null 2>&1 || true')
    await run_cmd(f'chmod 2775 {q_dir} >/dev/null 2>&1 || true')
    return data_dir


def _parse_disk_limit_mb(data: dict) -> Optional[int]:
    disk_mb_raw = data.get('disk_mb', None)
    if disk_mb_raw is None:
        disk_mb_raw = data.get('storage_mb', None)
    if disk_mb_raw is None:
        disk_mb_raw = data.get('hdd_mb', None)

    if disk_mb_raw is not None:
        try:
            mb = int(str(disk_mb_raw).strip())
            return max(512, min(mb, 2_097_152))
        except Exception:
            return None

    disk_gb_raw = data.get('disk_gb', None)
    if disk_gb_raw is None:
        disk_gb_raw = data.get('storage_gb', None)
    if disk_gb_raw is None:
        disk_gb_raw = data.get('hdd_gb', None)

    if disk_gb_raw is None:
        return None

    try:
        gb = float(str(disk_gb_raw).strip())
        if gb <= 0:
            return None
        mb = int(round(gb * 1024.0))
        return max(512, min(mb, 2_097_152))
    except Exception:
        return None


async def _docker_container_inspect_summary(container_name: str) -> Tuple[dict, Optional[float], Optional[int]]:
    q_name = shlex.quote(container_name)
    try:
        raw = await run_cmd(f'docker inspect {q_name}')
        data = json.loads(raw)
        if not isinstance(data, list) or len(data) == 0:
            raise RuntimeError('Invalid docker inspect output')

        info = data[0] or {}
        container_id = info.get('Id')

        state = ((info.get('State') or {}).get('Status') or 'unknown')
        state = str(state).strip().lower() or 'unknown'

        mapped_port = None
        ports = ((info.get('NetworkSettings') or {}).get('Ports') or {})
        if isinstance(ports, dict):
            for container_port, mappings in ports.items():
                if not isinstance(container_port, str) or not container_port.endswith('/udp'):
                    continue
                if not isinstance(mappings, list) or len(mappings) == 0:
                    continue
                host_port = (mappings[0] or {}).get('HostPort')
                if host_port is None:
                    continue
                try:
                    mapped_port = int(str(host_port).strip())
                    break
                except Exception:
                    mapped_port = None

        hostcfg = info.get('HostConfig') or {}
        try:
            nano = int(hostcfg.get('NanoCpus') or 0)
        except Exception:
            nano = 0
        try:
            quota = int(hostcfg.get('CpuQuota') or 0)
        except Exception:
            quota = 0
        try:
            period = int(hostcfg.get('CpuPeriod') or 0)
        except Exception:
            period = 0
        try:
            mem_b = int(hostcfg.get('Memory') or 0)
        except Exception:
            mem_b = 0

        cpu_limit: Optional[float] = None
        if nano > 0:
            cpu_limit = float(nano) / 1_000_000_000.0
        elif quota > 0 and period > 0:
            cpu_limit = float(quota) / float(period)

        mem_limit_mb: Optional[int] = None
        if mem_b > 0:
            mem_limit_mb = int(mem_b / 1024 / 1024)

        return {
            'exists': True,
            'container_id': container_id,
            'container_name': container_name,
            'state': state,
            'port': mapped_port,
        }, cpu_limit, mem_limit_mb
    except Exception as e:
        msg = str(e).lower()
        if 'no such object' in msg or 'not found' in msg:
            return {
                'exists': False,
                'container_id': None,
                'container_name': container_name,
                'state': 'missing',
                'port': None,
            }, None, None
        raise


async def _ensure_server_disk_quota(server_id: int, data_dir: str, disk_limit_mb: Optional[int], allow_init: bool = True) -> None:
    if disk_limit_mb is None:
        try:
            with open(os.path.join(os.path.join(SERVERS_BASE_DIR, str(server_id)), '.disk_quota'), 'r', encoding='utf-8') as f:
                disk_limit_mb = int((f.read() or '').strip() or '0')
        except Exception:
            disk_limit_mb = None

    base_dir = os.path.join(SERVERS_BASE_DIR, str(server_id))
    img_path = os.path.join(base_dir, 'disk.img')
    marker_path = os.path.join(base_dir, '.disk_quota')

    q_data_dir = shlex.quote(data_dir)
    q_img = shlex.quote(img_path)

    try:
        mounted = (await run_cmd(f"mountpoint -q {q_data_dir} && echo 1 || echo 0")).strip() == '1'
    except Exception:
        mounted = False

    if mounted:
        return

    if disk_limit_mb is None and os.path.exists(marker_path):
        try:
            with open(marker_path, 'r', encoding='utf-8') as f:
                disk_limit_mb = int((f.read() or '').strip() or '0')
        except Exception:
            disk_limit_mb = None

    if disk_limit_mb is None and os.path.exists(img_path):
        disk_limit_mb = 0

    if disk_limit_mb is None:
        return

    if not allow_init and (not os.path.exists(marker_path)) and (not os.path.exists(img_path)):
        return

    if allow_init and not os.path.exists(img_path) and not os.path.exists(marker_path):
        await run_cmd('command -v mkfs.ext4 >/dev/null 2>&1')
        try:
            if os.path.isdir(data_dir) and len(os.listdir(data_dir)) > 0:
                raise RuntimeError('data dir is not empty')
        except Exception as e:
            raise RuntimeError(f'Cannot enable disk limit: {e}')

        await run_cmd(f'mkdir -p {shlex.quote(base_dir)}')
        await run_cmd(f'truncate -s {disk_limit_mb}M {q_img}')
        await run_cmd(f'mkfs.ext4 -F {q_img} >/dev/null 2>&1')
        try:
            with open(marker_path, 'w', encoding='utf-8') as f:
                f.write(str(disk_limit_mb))
        except Exception:
            await run_cmd(f'printf %s {shlex.quote(str(disk_limit_mb))} > {shlex.quote(marker_path)}')

    if not os.path.exists(img_path):
        raise RuntimeError('disk image not found')

    await run_cmd(f'mount -o loop,noatime {q_img} {q_data_dir}')
    await run_cmd(f'chown root:{shlex.quote(FTP_GROUP)} {q_data_dir} >/dev/null 2>&1 || true')
    await run_cmd(f'chmod 2775 {q_data_dir} >/dev/null 2>&1 || true')


async def _disk_usage_mb(target_path: str) -> Tuple[Optional[int], Optional[int], Optional[int], Optional[float]]:
    try:
        q_path = shlex.quote(target_path)
        out = await run_cmd(f'df -mP {q_path} 2>/dev/null || true')
        lines = (out or '').strip().splitlines()
        if len(lines) < 2:
            return None, None, None, None

        line = lines[-1].strip()
        parts = re.split(r'\s+', line)
        if len(parts) < 6:
            return None, None, None, None

        total_mb = int(parts[1])
        used_mb = int(parts[2])
        avail_mb = int(parts[3])
        usep_s = str(parts[4]).strip()
        percent: Optional[float] = None
        if usep_s.endswith('%'):
            try:
                percent = float(usep_s[:-1])
            except Exception:
                percent = None

        return total_mb, used_mb, avail_mb, percent
    except Exception:
        return None, None, None, None


async def _docker_first_udp_host_port(container_name: str) -> Optional[int]:
    try:
        raw = await run_cmd(f'docker inspect {shlex.quote(container_name)}')
        data = json.loads(raw)
        if not isinstance(data, list) or len(data) == 0:
            return None

        ports = ((data[0] or {}).get('NetworkSettings') or {}).get('Ports') or {}
        if not isinstance(ports, dict):
            return None

        for container_port, mappings in ports.items():
            if not isinstance(container_port, str) or not container_port.endswith('/udp'):
                continue
            if not isinstance(mappings, list) or len(mappings) == 0:
                continue
            host_port = (mappings[0] or {}).get('HostPort')
            if host_port is None:
                continue
            try:
                return int(str(host_port).strip())
            except Exception:
                return None
        return None
    except Exception:
        return None


def _parse_resource_limits(data: dict) -> Tuple[Optional[float], Optional[int]]:
    cpu_raw = data.get('cpu_cores', None)
    if cpu_raw is None:
        cpu_raw = data.get('cpus', None)
    if cpu_raw is None:
        cpu_raw = data.get('cpu', None)

    cpu_cores: Optional[float] = None
    if cpu_raw is not None:
        try:
            cpu_cores = float(str(cpu_raw).strip())
        except Exception:
            cpu_cores = None

    mem_mb_raw = data.get('ram_mb', None)
    if mem_mb_raw is None:
        mem_mb_raw = data.get('mem_mb', None)
    if mem_mb_raw is None:
        mem_mb_raw = data.get('memory_mb', None)

    mem_mb: Optional[int] = None
    if mem_mb_raw is not None:
        try:
            mem_mb = int(str(mem_mb_raw).strip())
        except Exception:
            mem_mb = None
    else:
        mem_gb_raw = data.get('ram_gb', None)
        if mem_gb_raw is None:
            mem_gb_raw = data.get('memory_gb', None)
        if mem_gb_raw is not None:
            try:
                mem_mb = int(round(float(str(mem_gb_raw).strip()) * 1024.0))
            except Exception:
                mem_mb = None

    if cpu_cores is not None:
        if cpu_cores <= 0:
            cpu_cores = None
        else:
            cpu_cores = max(0.1, min(cpu_cores, 64.0))

    if mem_mb is not None:
        if mem_mb <= 0:
            mem_mb = None
        else:
            mem_mb = max(128, min(mem_mb, 262144))

    return cpu_cores, mem_mb


def _docker_limit_flags(
    cpu_cores: Optional[float],
    mem_mb: Optional[int],
    cpu_shares: Optional[int] = None,
    with_swap: bool = True,
) -> str:
    parts: List[str] = []
    if cpu_cores is not None:
        cpu_period = 100000
        cpu_quota = int(round(float(cpu_cores) * float(cpu_period)))
        cpu_quota = max(1000, cpu_quota)
        parts.append(f'--cpu-period {cpu_period}')
        parts.append(f'--cpu-quota {cpu_quota}')
    if cpu_shares is not None:
        parts.append(f'--cpu-shares {cpu_shares}')
    if mem_mb is not None:
        parts.append(f'--memory {mem_mb}m')
        if with_swap:
            parts.append(f'--memory-swap {mem_mb}m')
    return ' '.join(parts).strip()


async def _docker_apply_limits(container_name: str, cpu_cores: Optional[float], mem_mb: Optional[int], cpu_shares: Optional[int] = None) -> None:
    flags = _docker_limit_flags(cpu_cores, mem_mb, cpu_shares, True)
    if flags == '':
        return

    try:
        await run_cmd(f'docker update {flags} {shlex.quote(container_name)}')
        return
    except Exception as e:
        msg = str(e).lower()
        if mem_mb is not None and ('swap limit capabilities' in msg or 'swapaccount' in msg):
            flags2 = _docker_limit_flags(cpu_cores, mem_mb, cpu_shares, False)
            if flags2 != '':
                await run_cmd(f'docker update {flags2} {shlex.quote(container_name)}')
            return
        raise


async def _docker_container_limits(container_name: str) -> Tuple[Optional[float], Optional[int]]:
    try:
        raw = await run_cmd(
            "docker inspect -f '{{.HostConfig.NanoCpus}}|{{.HostConfig.CpuQuota}}|{{.HostConfig.CpuPeriod}}|{{.HostConfig.Memory}}' "
            + shlex.quote(container_name)
        )
        raw = (raw or '').strip().splitlines()[0].strip() if (raw or '').strip() else ''
        nano_s, quota_s, period_s, mem_s = (raw.split('|') + ['0', '0', '0', '0'])[:4]
        nano = int(nano_s.strip() or '0')
        quota = int(quota_s.strip() or '0')
        period = int(period_s.strip() or '0')
        mem_b = int(mem_s.strip() or '0')

        cpu_limit: Optional[float] = None
        if nano > 0:
            cpu_limit = float(nano) / 1_000_000_000.0
        elif quota > 0 and period > 0:
            cpu_limit = float(quota) / float(period)

        mem_limit_mb: Optional[int] = None
        if mem_b > 0:
            mem_limit_mb = int(mem_b / 1024 / 1024)

        return cpu_limit, mem_limit_mb
    except Exception:
        return None, None
