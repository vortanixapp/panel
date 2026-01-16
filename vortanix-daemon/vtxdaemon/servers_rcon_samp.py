import asyncio
import socket


async def _samp_rcon(
    header_host: str,
    header_port: int,
    password: str,
    command: str,
    target_host: str,
    target_port: int,
    timeout_sec: float = 3.0,
) -> str:
    def _run() -> str:
        ip_parts = header_host.strip().split('.')
        if len(ip_parts) != 4:
            raise ValueError('Invalid host for SA-MP header')

        ip_bytes = bytes([int(p) & 0xFF for p in ip_parts])
        port_bytes = bytes([header_port & 0xFF, (header_port >> 8) & 0xFF])
        pass_b = password.encode('utf-8')
        cmd_b = command.encode('utf-8')

        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        try:
            s.settimeout(timeout_sec)

            ping_data = b'1337'
            ping_pkt = b'SAMP' + ip_bytes + port_bytes + b'p' + ping_data
            s.sendto(ping_pkt, (target_host, target_port))

            try:
                ping_resp, _ = s.recvfrom(2048)
                expected = b'p' + ping_data

                if (not ping_resp) or (len(ping_resp) < (10 + len(expected))):
                    raise ValueError('Invalid response from server')
                if ping_resp[10:10 + len(expected)] != expected:
                    raise ValueError('Invalid response from server')
            except socket.timeout:
                raise ValueError('Invalid response from server')

            pkt = b'SAMP' + ip_bytes + port_bytes + b'x'
            pkt += len(pass_b).to_bytes(2, 'little') + pass_b
            pkt += len(cmd_b).to_bytes(2, 'little') + cmd_b

            s.sendto(pkt, (target_host, target_port))
            lines = []
            while True:
                data, _ = s.recvfrom(2048)
                if not data:
                    break
                chunk = data[13:] if len(data) > 13 else b''
                text = chunk.decode('utf-8', errors='ignore').strip('\r\n\0 ')
                if text:
                    lines.append(text)
        except socket.timeout:
            pass
        finally:
            try:
                s.close()
            except Exception:
                pass

        return '\n'.join(lines)

    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, _run)
