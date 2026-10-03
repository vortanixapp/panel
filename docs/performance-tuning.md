# Tuning a game node

[English](performance-tuning.md) · [Русский](performance-tuning.ru.md)

Settings of the operating system of a game node (the machine with the agent). They
are not applied by the panel: check each one on a test node first, measure with
the monitoring page ([monitoring.md](monitoring.md)) and keep what helps.

- [CPU](#cpu)
- [Memory and swap](#memory-and-swap)
- [Network (UDP)](#network-udp)
- [Disk and file system](#disk-and-file-system)
- [What the panel can do itself](#what-the-panel-can-do-itself)
- [Checking the result](#checking-the-result)

## CPU

Game servers are sensitive to frequency drops. Use the `performance` governor (on Debian the package is `linux-cpupower`):

```bash
sudo apt-get install -y linux-tools-common linux-tools-generic
sudo cpupower frequency-set -g performance
cpupower frequency-info | grep "current policy"
```

To keep it after a reboot use `tuned` with the `latency-performance` profile, or
a systemd unit that runs the command at boot. On virtual machines the governor
is often not available and the host decides, then there is nothing to change.

Leave `irqbalance` running. Pinning containers to cores (`cpuset`) is not done by
the panel on purpose: without knowing the topology and the load it usually hurts.

## Memory and swap

A container never swaps: the agent starts it with `--memory-swap` equal to the
memory limit. Swap only matters for the host itself.

```ini
vm.swappiness = 10
```

A small swap (1–2 GB) is better than none: it lets the kernel push out rarely
used pages of system services instead of killing a game server.

Java games (Minecraft and others) gain from transparent huge pages in the
`madvise` mode together with the JVM flag `-XX:+UseTransparentHugePages` in the
startup parameters of the server:

```bash
cat /sys/kernel/mm/transparent_hugepage/enabled
echo madvise | sudo tee /sys/kernel/mm/transparent_hugepage/enabled
```

Do not set `always`: it slows down other programs and creates latency spikes.

## Network (UDP)

Most games send UDP. Under a load of players the default buffers overflow and
packets are dropped silently. File `/etc/sysctl.d/99-vortanix.conf`:

```ini
net.core.rmem_max = 26214400
net.core.wmem_max = 26214400
net.core.rmem_default = 1048576
net.core.wmem_default = 1048576
net.core.netdev_max_backlog = 16384
net.core.somaxconn = 4096
net.netfilter.nf_conntrack_max = 262144
net.netfilter.nf_conntrack_udp_timeout = 30
net.netfilter.nf_conntrack_udp_timeout_stream = 60
```

```bash
sudo sysctl --system
```

`nf_conntrack` keys exist only when the module is loaded (Docker loads it). If
`dmesg` shows `nf_conntrack: table full`, raise `nf_conntrack_max`.

Dropped packets are visible here, growing numbers mean the buffers are too small:

```bash
netstat -su | grep -i "receive errors\|buffer errors"
```

## Disk and file system

- **NVMe or SSD** for `/var/lib/vortanix` and `/var/lib/docker`. Worlds, logs and
  game files are random reads and writes, a spinning disk shows up as lag.
- **XFS** is the best choice: the agent limits the disk of a server with project
  quotas (`setquota`, `repquota`), which XFS supports with the `prjquota` mount
  option, and the install cache copies files with `reflink`, which XFS supports
  too. Create it with reflink on (the default since xfsprogs 5.1):

```bash
sudo mkfs.xfs -m reflink=1 /dev/nvme0n1p1
```

  `/etc/fstab`:

```text
/dev/nvme0n1p1  /var/lib/vortanix  xfs  defaults,noatime,prjquota  0 2
```

- Keep the install cache on the same file system as the servers
  (`/var/lib/vortanix/install-cache` next to `/var/lib/vortanix/servers`).
  Otherwise copying cannot use `reflink` and a full copy is made.
- `noatime` removes a write on every read.
- For NVMe the I/O scheduler `none` is right, for SSD `mq-deadline`:

```bash
cat /sys/block/nvme0n1/queue/scheduler
```

## What the panel can do itself

In **Administration → Settings → Nodes**, section "Server speed and isolation"
(all off by default, turn on one by one on a test node):

| Setting | What it gives |
|---|---|
| Game install cache on the node | A game is downloaded to the node once, new servers are copied from the cache |
| Share CPU by memory size | Under a busy processor a server with more memory gets more time |
| Soft memory limit | Under memory shortage the kernel takes memory first from servers above the share |

SteamCMD during an installation always runs with a low CPU weight, so installs
do not take time from running servers.

## Checking the result

1. Open **Administration → Performance** before and after a change, and compare
   the same period.
2. On the node: `top` or `htop` for CPU and memory, `iostat -x 1` for disk
   (`%util` and `await`), `netstat -su` for lost UDP packets.
3. Change one thing at a time. If a change does not show in the numbers, revert
   it.
