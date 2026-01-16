#!/usr/bin/env python3
import asyncio
import os
import sys
import shlex

from aiohttp import web

from vtxdaemon.config import HOSTNAME, LOCATION_CODE, PORT
from vtxdaemon.logbuf import LOG_BUFFER, log
from vtxdaemon.metrics import collect_metrics_once
from vtxdaemon.middlewares import auth_middleware, cors_middleware
from vtxdaemon.servers_runtime import (
    create_server_handler,
    delete_server_handler,
    server_logs_handler,
    server_rcon_handler,
    server_metrics_handler,
    server_restart_handler,
    server_reinstall_handler,
    server_start_handler,
    server_status_handler,
    server_stop_handler,
)
from vtxdaemon.servers_query import server_online_batch_handler
from vtxdaemon.servers_files import (
    server_files_delete_handler,
    server_files_download_handler,
    server_files_list_handler,
    server_files_mkdir_handler,
    server_files_read_handler,
    server_files_upload_handler,
    server_files_write_handler,
)
from vtxdaemon.servers_services import (
    server_ftp_create_user_handler,
    server_mysql_create_db_handler,
    server_fastdl_sync_handler,
)
from vtxdaemon.servers_plugins import server_plugins_apply_handler
from vtxdaemon.servers_cron import (
    server_cron_list_handler,
    server_cron_create_handler,
    server_cron_delete_handler,
    server_cron_toggle_handler,
)
from vtxdaemon.servers_firewall import (
    server_firewall_list_handler,
    server_firewall_set_handler,
    server_firewall_add_rule_handler,
    server_firewall_delete_rule_handler,
    server_firewall_toggle_rule_handler,
    server_firewall_toggle_handler,
)

async def health_handler(request):
    return web.json_response({
        'status': 'ok',
        'location_code': LOCATION_CODE,
        'hostname': HOSTNAME
    })

async def info_handler(request):
    return web.json_response({
        'location_code': LOCATION_CODE,
        'hostname': HOSTNAME,
        'platform': sys.platform,
        'pid': os.getpid(),
        'uptime_sec': asyncio.get_event_loop().time()
    })

async def metrics_handler(request):
    try:
        metrics = await collect_metrics_once()
        await log(f'Collected {len(metrics)} metrics')
        return web.json_response({
            'location_code': LOCATION_CODE,
            'hostname': HOSTNAME,
            'metrics': metrics
        })
    except Exception as e:
        await log(f'Metrics collection error: {e}')
        return web.json_response({'error': str(e)}, status=500)

async def exec_handler(request):
    # Выполнение произвольной команды на сервере (ОЧЕНЬ мощный инструмент, предполагается, что доступ ограничен панелью админа)
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'error': f'Invalid JSON: {e}'}, status=400)

    cmd = data.get('cmd') if isinstance(data, dict) else None
    if not cmd or not isinstance(cmd, str):
        return web.json_response({'error': 'Field "cmd" (string) is required'}, status=400)

    cmd = cmd.strip()
    if cmd == '':
        return web.json_response({'error': 'Field "cmd" must be non-empty'}, status=400)

    allow_raw = os.getenv('VORTANIX_EXEC_ALLOWLIST', '').strip()
    if allow_raw == '':
        return web.json_response({'ok': False, 'error': 'exec is disabled'}, status=403)

    allow = [x.strip() for x in allow_raw.split(',') if x.strip()]
    argv = shlex.split(cmd)
    if len(argv) == 0:
        return web.json_response({'ok': False, 'error': 'Invalid cmd'}, status=400)

    exe = argv[0]
    if exe not in allow:
        return web.json_response({'ok': False, 'error': 'Command is not allowed'}, status=403)

    timeout_sec = float(os.getenv('VORTANIX_EXEC_TIMEOUT_SEC', '10') or '10')
    max_output_bytes = int(os.getenv('VORTANIX_EXEC_MAX_OUTPUT_BYTES', '20000') or '20000')
    timeout_sec = max(1.0, min(timeout_sec, 60.0))
    max_output_bytes = max(1000, min(max_output_bytes, 200000))

    await log(f'EXEC requested: {exe}')

    try:
        from vtxdaemon.cmd import run_argv

        output = await run_argv(argv, timeout_sec=timeout_sec, max_output_bytes=max_output_bytes)
        await log(f'EXEC success: {exe}')

        if output:
            for line in output.splitlines()[:200]:
                await log(f'OUT {line}')

        return web.json_response({'ok': True, 'output': output})
    except Exception as e:
        await log(f'EXEC error: {exe} -> {e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)

async def logs_handler(request):
    # Вернуть последние N строк логов
    try:
        tail = int(request.query.get('tail', '200'))
    except ValueError:
        tail = 200

    tail = max(1, min(tail, 500))
    lines = list(LOG_BUFFER)[-tail:]

    return web.json_response({
        'location_code': LOCATION_CODE,
        'hostname': HOSTNAME,
        'logs': lines,
    })

async def main():
    app = web.Application()

    app.middlewares.append(auth_middleware)
    app.middlewares.append(cors_middleware)

    app.router.add_get('/health', health_handler)
    app.router.add_get('/info', info_handler)
    app.router.add_get('/metrics', metrics_handler)
    app.router.add_get('/logs', logs_handler)
    app.router.add_post('/exec', exec_handler)
    app.router.add_post('/servers/create', create_server_handler)
    app.router.add_post('/servers/delete', delete_server_handler)
    app.router.add_post('/servers/ftp/create-user', server_ftp_create_user_handler)
    app.router.add_post('/servers/mysql/create-db', server_mysql_create_db_handler)
    app.router.add_post('/servers/fastdl/sync', server_fastdl_sync_handler)
    app.router.add_post('/servers/plugins/apply', server_plugins_apply_handler)
    app.router.add_post('/servers/files/list', server_files_list_handler)
    app.router.add_post('/servers/files/read', server_files_read_handler)
    app.router.add_post('/servers/files/write', server_files_write_handler)
    app.router.add_post('/servers/files/delete', server_files_delete_handler)
    app.router.add_post('/servers/files/mkdir', server_files_mkdir_handler)
    app.router.add_post('/servers/files/upload', server_files_upload_handler)
    app.router.add_post('/servers/files/download', server_files_download_handler)
    app.router.add_post('/servers/logs', server_logs_handler)
    app.router.add_post('/servers/rcon', server_rcon_handler)
    app.router.add_post('/servers/metrics', server_metrics_handler)
    app.router.add_post('/servers/status', server_status_handler)
    app.router.add_post('/servers/start', server_start_handler)
    app.router.add_post('/servers/stop', server_stop_handler)
    app.router.add_post('/servers/restart', server_restart_handler)
    app.router.add_post('/servers/reinstall', server_reinstall_handler)
    app.router.add_post('/servers/cron/list', server_cron_list_handler)
    app.router.add_post('/servers/cron/create', server_cron_create_handler)
    app.router.add_post('/servers/cron/delete', server_cron_delete_handler)
    app.router.add_post('/servers/cron/toggle', server_cron_toggle_handler)

    app.router.add_post('/servers/firewall/list', server_firewall_list_handler)
    app.router.add_post('/servers/firewall/set', server_firewall_set_handler)
    app.router.add_post('/servers/firewall/add-rule', server_firewall_add_rule_handler)
    app.router.add_post('/servers/firewall/delete-rule', server_firewall_delete_rule_handler)
    app.router.add_post('/servers/firewall/toggle-rule', server_firewall_toggle_rule_handler)
    app.router.add_post('/servers/firewall/toggle', server_firewall_toggle_handler)

    app.router.add_post('/servers/online-batch', server_online_batch_handler)

    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, '0.0.0.0', PORT)
    await site.start()

    await log(f'HTTP daemon listening on port {PORT} for location {LOCATION_CODE} (host={HOSTNAME})')

    # Graceful shutdown
    try:
        while True:
            await asyncio.sleep(3600)
    except KeyboardInterrupt:
        await log('Shutting down location-daemon')
        await runner.cleanup()

if __name__ == '__main__':
    asyncio.run(main())
