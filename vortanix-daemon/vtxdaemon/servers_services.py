import os
import re
import shlex
import shutil

from aiohttp import web

from vtxdaemon.cmd import run_cmd
from vtxdaemon.logbuf import log
from vtxdaemon.servers_common import (
    FTP_GROUP,
    FTP_PASV_MIN_PORT,
    FTP_PASV_MAX_PORT,
    MYSQL_PORT,
    MYSQL_BIND_ADDRESS,
    MYSQL_CONFIG_FILE,
    _ensure_server_data_dir,
    _ensure_server_disk_quota,
    _fastdl_dir,
    _server_data_dir,
)


async def _ensure_vsftpd_config() -> None:
    await run_cmd('command -v vsftpd >/dev/null 2>&1')

    marker_path = '/opt/vortanix/.vsftpd_config_ok'

    conf = (
        'listen=NO\n'
        'listen_ipv6=YES\n'
        'anonymous_enable=NO\n'
        'local_enable=YES\n'
        'write_enable=YES\n'
        'local_umask=002\n'
        'check_shell=NO\n'
        'chroot_local_user=YES\n'
        'allow_writeable_chroot=YES\n'
        'pam_service_name=vsftpd\n'
        'pasv_enable=YES\n'
        f'pasv_min_port={FTP_PASV_MIN_PORT}\n'
        f'pasv_max_port={FTP_PASV_MAX_PORT}\n'
    )

    current = None
    try:
        if os.path.exists('/etc/vsftpd.conf'):
            with open('/etc/vsftpd.conf', 'r', encoding='utf-8') as f:
                current = f.read()
    except Exception:
        current = None

    changed = (current != conf)
    if changed:
        try:
            with open('/etc/vsftpd.conf', 'w', encoding='utf-8') as f:
                f.write(conf)
        except Exception:
            await run_cmd('printf %s ' + shlex.quote(conf) + ' > /etc/vsftpd.conf')

    await run_cmd('test -f /etc/shells || (touch /etc/shells && chmod 644 /etc/shells) || true')
    await run_cmd('grep -qxF /usr/sbin/nologin /etc/shells || echo /usr/sbin/nologin >> /etc/shells')

    if not os.path.exists(marker_path):
        await run_cmd('systemctl enable vsftpd >/dev/null 2>&1 || true')
        await run_cmd('systemctl start vsftpd >/dev/null 2>&1 || true')
        await run_cmd('ufw allow 21 >/dev/null 2>&1 || true')
        await run_cmd(f'ufw allow {FTP_PASV_MIN_PORT}:{FTP_PASV_MAX_PORT}/tcp >/dev/null 2>&1 || true')
        try:
            os.makedirs(os.path.dirname(marker_path), exist_ok=True)
            with open(marker_path, 'w', encoding='utf-8') as f:
                f.write('1')
        except Exception:
            await run_cmd(f'printf %s 1 > {shlex.quote(marker_path)}')

    if changed:
        await run_cmd('systemctl restart vsftpd >/dev/null 2>&1 || true')


async def server_ftp_create_user_handler(request):
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    server_id = data.get('server_id')
    if not isinstance(server_id, int):
        try:
            server_id = int(server_id)
        except Exception:
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    username = str(data.get('username') or '').strip()
    password = str(data.get('password') or '').strip()

    if not re.match(r'^[a-zA-Z0-9_-]{1,32}$', username):
        return web.json_response({'ok': False, 'error': 'username must match ^[a-zA-Z0-9_-]{1,32}$'}, status=400)
    if not re.match(r'^[a-zA-Z0-9]{8,64}$', password):
        return web.json_response({'ok': False, 'error': 'password must be 8..64 chars [a-zA-Z0-9]'}, status=400)

    data_dir = await _ensure_server_data_dir(server_id)
    await _ensure_server_disk_quota(server_id, data_dir, None, allow_init=False)
    await _ensure_vsftpd_config()

    q_user = shlex.quote(username)
    q_dir = shlex.quote(data_dir)
    marker_path = os.path.join(data_dir, '.ftp_perms_ok')

    try:
        await run_cmd(f'id -u {q_user} >/dev/null 2>&1 || useradd -d {q_dir} -s /usr/sbin/nologin -g {shlex.quote(FTP_GROUP)} -M {q_user}')
        await run_cmd(f'echo {shlex.quote(username + ":" + password)} | chpasswd')
        await run_cmd(f'usermod -d {q_dir} {q_user}')

        await run_cmd(f'chown {q_user}:{shlex.quote(FTP_GROUP)} {q_dir} >/dev/null 2>&1 || true')
        await run_cmd(f'chmod 2775 {q_dir} >/dev/null 2>&1 || true')

        if not os.path.exists(marker_path):
            await run_cmd(f'chown -R {q_user}:{shlex.quote(FTP_GROUP)} {q_dir} >/dev/null 2>&1 || true')
            await run_cmd(f'chmod -R u+rwX,g+rwX {q_dir} >/dev/null 2>&1 || true')
            try:
                with open(marker_path, 'w', encoding='utf-8') as f:
                    f.write('1')
            except Exception:
                await run_cmd(f'printf %s 1 > {shlex.quote(marker_path)}')

        return web.json_response({
            'ok': True,
            'server_id': server_id,
            'username': username,
            'root': data_dir,
            'port': 21,
            'pasv_min_port': FTP_PASV_MIN_PORT,
            'pasv_max_port': FTP_PASV_MAX_PORT,
        })
    except Exception as e:
        await log(f'FTP user create error: server_id={server_id} username={username} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)


async def server_fastdl_sync_handler(request):
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    server_id = data.get('server_id')
    if not isinstance(server_id, int):
        try:
            server_id = int(server_id)
        except Exception:
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    # Источник — корень сервера; FastDL складываем с сохранением структуры cstrike/
    src_root = _server_data_dir(server_id)
    fastdl_root = _fastdl_dir(server_id)
    src_cstrike = os.path.join(src_root, 'cstrike')
    dst_cstrike = os.path.join(fastdl_root, 'cstrike')

    allow_ext = {
        '.mdl', '.wav', '.mp3', '.spr', '.txt', '.res',
        '.bsp', '.wad', '.tga', '.bmp'
    }

    try:
        await _ensure_server_disk_quota(server_id, src_root, None, allow_init=False)
        os.makedirs(dst_cstrike, exist_ok=True)

        if not os.path.isdir(src_cstrike):
            return web.json_response({'ok': False, 'error': 'cstrike directory not found'}, status=400)

        copied = 0
        for root_dir, dirs, files in os.walk(src_cstrike):
            rel_root = os.path.relpath(root_dir, src_cstrike)
            if rel_root == '.':
                rel_root = ''
            target_root = os.path.join(dst_cstrike, rel_root)
            os.makedirs(target_root, exist_ok=True)

            for fname in files:
                ext = os.path.splitext(fname)[1].lower()
                if ext not in allow_ext:
                    continue
                src_path = os.path.join(root_dir, fname)
                dst_path = os.path.join(target_root, fname)
                try:
                    shutil.copy2(src_path, dst_path)
                    copied += 1
                except Exception:
                    continue

        q_fastdl = shlex.quote(fastdl_root)
        ftp_user = f'srv{server_id}'
        await run_cmd(f'chown -R {shlex.quote(ftp_user)}:{shlex.quote(FTP_GROUP)} {q_fastdl} >/dev/null 2>&1 || true')
        await run_cmd(f'find {q_fastdl} -type d -exec chmod 755 {{}} \\; >/dev/null 2>&1 || true')
        await run_cmd(f'find {q_fastdl} -type f -exec chmod 644 {{}} \\; >/dev/null 2>&1 || true')

        return web.json_response({
            'ok': True,
            'server_id': server_id,
            'src': src_root,
            'dst': fastdl_root,
            'copied': copied,
        })
    except Exception as e:
        await log(f'FastDL sync error: server_id={server_id} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)


async def _ensure_mysql_config() -> None:
    await run_cmd('command -v mysql >/dev/null 2>&1')

    conf_path = MYSQL_CONFIG_FILE
    if not os.path.exists(conf_path) and os.path.exists('/etc/mysql/mariadb.conf.d/50-server.cnf'):
        conf_path = '/etc/mysql/mariadb.conf.d/50-server.cnf'

    q_conf = shlex.quote(conf_path)
    bind_addr = str(MYSQL_BIND_ADDRESS).strip() or '0.0.0.0'

    await run_cmd('systemctl enable mysql >/dev/null 2>&1 || systemctl enable mariadb >/dev/null 2>&1 || true')
    await run_cmd('systemctl start mysql >/dev/null 2>&1 || systemctl start mariadb >/dev/null 2>&1 || true')

    if os.path.exists(conf_path):
        await run_cmd(
            f"(grep -q '^bind-address' {q_conf} && sed -i 's/^bind-address.*/bind-address = {bind_addr}/' {q_conf}) || echo 'bind-address = {bind_addr}' >> {q_conf}"
        )

    await run_cmd('systemctl restart mysql >/dev/null 2>&1 || systemctl restart mariadb >/dev/null 2>&1 || true')
    await run_cmd(f'ufw allow {MYSQL_PORT}/tcp >/dev/null 2>&1 || true')


def _mysql_escape_string(val: str) -> str:
    return str(val).replace('\\', '\\\\').replace("'", "\\'")


def _mysql_escape_ident(val: str) -> str:
    return str(val).replace('`', '``')


async def server_mysql_create_db_handler(request):
    try:
        data = await request.json()
    except Exception as e:
        return web.json_response({'ok': False, 'error': f'Invalid JSON: {e}'}, status=400)

    if not isinstance(data, dict):
        return web.json_response({'ok': False, 'error': 'Invalid payload'}, status=400)

    server_id = data.get('server_id')
    if not isinstance(server_id, int):
        try:
            server_id = int(server_id)
        except Exception:
            return web.json_response({'ok': False, 'error': 'server_id is required (int)'}, status=400)

    database = str(data.get('database') or f'srv{server_id}').strip()
    username = str(data.get('username') or f'srv{server_id}').strip()
    password = str(data.get('password') or '').strip()

    if not re.match(r'^[a-zA-Z0-9_]{1,64}$', database):
        return web.json_response({'ok': False, 'error': 'database must match ^[a-zA-Z0-9_]{1,64}$'}, status=400)
    if not re.match(r'^[a-zA-Z0-9_]{1,32}$', username):
        return web.json_response({'ok': False, 'error': 'username must match ^[a-zA-Z0-9_]{1,32}$'}, status=400)
    if not re.match(r'^[a-zA-Z0-9]{8,64}$', password):
        return web.json_response({'ok': False, 'error': 'password must be 8..64 chars [a-zA-Z0-9]'}, status=400)

    try:
        data_dir = await _ensure_server_data_dir(server_id)
        await _ensure_server_disk_quota(server_id, data_dir, None, allow_init=False)
        await _ensure_mysql_config()

        db_ident = _mysql_escape_ident(database)
        user_esc = _mysql_escape_string(username)
        pass_esc = _mysql_escape_string(password)

        sql = (
            f"CREATE DATABASE IF NOT EXISTS `{db_ident}` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
            f"CREATE USER IF NOT EXISTS '{user_esc}'@'%' IDENTIFIED BY '{pass_esc}';"
            f"ALTER USER '{user_esc}'@'%' IDENTIFIED BY '{pass_esc}';"
            f"GRANT ALL PRIVILEGES ON `{db_ident}`.* TO '{user_esc}'@'%';"
            "FLUSH PRIVILEGES;"
        )

        await run_cmd('mysql -e ' + shlex.quote(sql))

        return web.json_response({
            'ok': True,
            'server_id': server_id,
            'database': database,
            'username': username,
            'port': MYSQL_PORT,
        })
    except Exception as e:
        await log(f'MySQL create db error: server_id={server_id} database={database} username={username} error={e}')
        return web.json_response({'ok': False, 'error': str(e)}, status=500)
