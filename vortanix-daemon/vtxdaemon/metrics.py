from datetime import datetime, timezone

from vtxdaemon.cmd import run_cmd
from vtxdaemon.logbuf import log


async def collect_metrics_once():
    metrics = []
    measured_at = datetime.now(timezone.utc).isoformat()

    # CPU usage (from /proc/loadavg)
    try:
        with open('/proc/loadavg', 'r') as f:
            loadavg = f.read().strip().split()[0]
            load = float(loadavg)
            value = min(100, max(0, load * 25))
            metrics.append({'metric_type': 'cpu_usage', 'value': value, 'measured_at': measured_at})
    except Exception as e:
        await log(f'CPU load error: {e}')

    # RAM usage
    try:
        mem_info = {}
        with open('/proc/meminfo', 'r') as f:
            for line in f:
                if ':' in line:
                    key, val = line.split(':', 1)
                    key = key.strip()
                    val = val.strip().split()[0]
                    mem_info[key] = int(val)

        total = mem_info.get('MemTotal', 0) * 1024
        avail = mem_info.get('MemAvailable', 0) * 1024
        if total > 0:
            used = total - avail
            value = (used / total) * 100
            metrics.append({'metric_type': 'ram_usage', 'value': value, 'measured_at': measured_at})
    except Exception as e:
        await log(f'RAM error: {e}')

    # OS info
    try:
        uname = await run_cmd('uname -a')
        metrics.append({'metric_type': 'os_info', 'value': uname, 'measured_at': measured_at})
    except Exception as e:
        await log(f'OS info error: {e}')

    # CPU model
    try:
        cpu_model = await run_cmd("grep 'model name' /proc/cpuinfo | head -1 | cut -d: -f2")
        metrics.append({'metric_type': 'cpu_model', 'value': cpu_model.strip(), 'measured_at': measured_at})
    except Exception as e:
        await log(f'CPU model error: {e}')

    # RAM total (human readable)
    try:
        ram_total = await run_cmd("free -h | awk 'NR==2{print $2}'")
        metrics.append({'metric_type': 'ram_total', 'value': ram_total, 'measured_at': measured_at})
    except Exception as e:
        await log(f'RAM total error: {e}')

    # Disk usage
    try:
        disk_info = await run_cmd("df -h / | awk 'NR==2{print $2,$3,$4}'")
        parts = disk_info.split()
        if len(parts) >= 3:
            metrics.append({'metric_type': 'disk_total', 'value': parts[0], 'measured_at': measured_at})
            metrics.append({'metric_type': 'disk_used', 'value': parts[1], 'measured_at': measured_at})
            metrics.append({'metric_type': 'disk_available', 'value': parts[2], 'measured_at': measured_at})
    except Exception as e:
        await log(f'Disk error: {e}')

    # Uptime
    try:
        uptime = await run_cmd('uptime -p')
        metrics.append({'metric_type': 'uptime', 'value': uptime, 'measured_at': measured_at})
    except Exception as e:
        await log(f'Uptime error: {e}')

    # Daemon process metrics (CPU%, MEM%, RSS in MB)
    try:
        import os

        pid = os.getpid()
        proc_info = await run_cmd(f"ps -p {pid} -o %cpu=,%mem=,rss=")
        parts = proc_info.split()
        if len(parts) >= 3:
            cpu = float(parts[0])
            mem = float(parts[1])
            rss_kb = float(parts[2])
            rss_mb = rss_kb / 1024.0
            metrics.append({'metric_type': 'daemon_cpu_usage', 'value': cpu, 'measured_at': measured_at})
            metrics.append({'metric_type': 'daemon_ram_usage', 'value': mem, 'measured_at': measured_at})
            metrics.append({'metric_type': 'daemon_rss_mb', 'value': rss_mb, 'measured_at': measured_at})
    except Exception as e:
        await log(f'Daemon process metrics error: {e}')

    return metrics
