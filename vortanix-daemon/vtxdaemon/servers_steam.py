from vtxdaemon.cmd import run_cmd


async def _ensure_steamcmd_host_deps() -> None:
    # steamcmd is a 32-bit binary; ensure i386 loader exists on the host
    out = await run_cmd('sh -c "test -e /lib/ld-linux.so.2 || test -e /lib/i386-linux-gnu/ld-linux.so.2 && echo ok || echo missing"')
    if str(out).strip() != 'ok':
        raise Exception(
            'steamcmd host dependencies are missing (32-bit loader). '
            'Install i386 libs on location host: dpkg --add-architecture i386 && apt update && '
            'apt install libc6-i386 libc6:i386 lib32gcc-s1 libstdc++6:i386 libgcc-s1:i386 zlib1g:i386 libuuid1:i386 && '
            '(apt install libtinfo6:i386 libncurses6:i386 || apt install libtinfo5:i386 libncurses5:i386) && '
            '(apt install libcurl4t64:i386 || apt install libcurl4:i386)'
        )
