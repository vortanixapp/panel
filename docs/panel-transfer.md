# Moving the panel to another server

[English](panel-transfer.md) · [Русский](panel-transfer.ru.md)

This document describes the supported way to move the control panel from one
machine to another: what is transferred, both transfer modes, the requirements
for the new machine, the DNS and game node switchover, and failure handling.

Panel section: **Administration → Panel transfer**. It is available to the panel
owner only and never through API keys: the operation hands out the `SECRETS_KEY`
encryption key and the whole database dump, and it gains root access to the new
machine. It cannot be delegated with a separate permission.

- [What is transferred](#what-is-transferred)
- [Requirements for the new machine](#requirements-for-the-new-machine)
- [SSH mode](#ssh-mode)
- [Archive mode](#archive-mode)
- [Read-only mode](#read-only-mode)
- [Panel address and DNS](#panel-address-and-dns)
- [Game nodes](#game-nodes)
- [Failure handling](#failure-handling)

## What is transferred

| Part | Source | Contents |
|---|---|---|
| Database `vortanix` | volume `vortanix_postgres` | users, servers, payments, settings |
| Uploaded files | volume `vortanix_uploads` | avatars, logo, ticket attachments, catalog archives |
| Secrets from `deploy/.env` | file on the panel machine | `SECRETS_KEY`, `JWT_SECRET`, SMTP, OAuth, registry credentials |

Not transferred:

- the Redis cache — rebuilt on the new machine;
- the Let's Encrypt certificate — issued again for the new address;
- game server files — they stay on the game nodes;
- `POSTGRES_PASSWORD` and `INTERNAL_SECRET` — these are machine-local
  passwords; the new machine keeps the ones generated during its install.

The address variables `SITE_ADDRESS`, `RELAY_PUBLIC_URL` and `FRONTEND_URL` are
not transferred: `scripts/init-env.sh` sets them on the new machine.

## Requirements for the new machine

- Debian or Ubuntu, x86_64 or aarch64;
- SSH access as `root` or as a user with passwordless `sudo`;
- free ports 80 and 443, plus 8443 when the panel runs by IP address;
- free space of at least three times the transferred data plus 5 GB;
- a domain whose A record will point to this machine, when the panel runs on a
  domain.

Docker does not have to be installed beforehand — the panel installs it.

## SSH mode

The panel installs itself on the new machine and streams the data over. Enter
the access to the new machine, choose the address, keep the write stop enabled
if you want it, and start the transfer.

The stages are shown in the section and written to the task log:

| Stage | Action |
|---|---|
| `prepare` | read the data set and its size |
| `freeze` | switch the current panel to read-only |
| `probe_target` | check the operating system, free space and ports |
| `install_docker` | install Docker |
| `clone` | fetch the panel files of the same version into `/opt/vortanix` |
| `init_env` | run `scripts/init-env.sh` with the chosen address |
| `compose_up` | start the stack and wait for it |
| `transfer` | stream the database, secrets and files |
| `health` | check the new panel |
| `relay_cert` | wait for the relay certificate (IP mode only) |
| `agents` | switch the game nodes to the new address |
| `done` | the transfer is finished |

The data is streamed: the dump is taken and sent at once, no intermediate file
is written and the updater port is never published. The database is restored in
a single transaction and the files are unpacked into a temporary directory and
moved into place by rename, so a dropped connection never leaves half the data.

The source panel keeps running in read-only mode. Shut it down by hand once the
new panel proves healthy:

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml down
```

## Archive mode

Use this when the panel cannot reach the new machine over SSH.

1. In **Panel transfer → By archive** set a password of at least twelve
   characters and download the `.vxt` archive. The archive is encrypted: the key
   is derived from the password with scrypt and the data is sealed with
   AES-256-GCM chunk by chunk, so a truncated or altered file is rejected.
2. Install the panel on the new machine as usual, see the
   [install guide](install.md). Its version must not be older than the version
   the archive was taken from.
3. On the new panel sign in as the owner, open the same section, press "Check
   archive" — the panel shows the version, the date and the data size — then
   "Replace panel data".

The archive contains `SECRETS_KEY`. Keep the file like a password: it unlocks
payment gateway keys, location SSH passwords and the relay TLS private key. Both
the download and the upload are recorded in the activity log.

## Read-only mode

Stopping writes is enabled by default when a transfer starts. In this mode the
panel answers `423` to every changing request except sign-in, sign-out and the
transfer section itself; the worker takes no new provisioning, backup, mailing
or billing jobs, while a job already running finishes.

After a successful transfer the mode stays on, so users do not write into a
stale copy while DNS still points at the old machine. Turn it off from the
transfer section — the panel warns that such changes will not reach the new
machine. On failure or cancellation the mode is lifted automatically.

## Panel address and DNS

**Same domain.** The new panel gets the same `SITE_ADDRESS`. Right after the
`done` stage point the domain's A record at the new machine. The Let's Encrypt
certificate is issued there on the first request and the game nodes reconnect on
their own, because their address has not changed.

**New address.** The new panel comes up on the given domain or IP address and is
available immediately. The panel switches the game nodes itself during the
`agents` stage.

## Game nodes

When the address changes, the panel walks the locations over SSH and recreates
the `vortanix-agent` container with the new `RELAY_URL`, and in IP mode with the
new relay certificate fingerprint as well. The node token is passed through
standard input and never appears in command arguments.

Nodes that could not be switched are counted during the transfer. Retry only
those with "Retry switching" in the same section, providing the access to the
new machine. A node can also be switched by hand with the agent install command
from its location page on the new panel.

## Failure handling

**Not enough space on the new machine.** The check runs before any change and
the transfer stops at `probe_target`. The source panel is untouched.

**Port 80 or 443 is busy.** Another web server is already running there. Stop it
or move to a clean machine.

**The connection dropped during the transfer.** The task fails and read-only
mode is lifted. A retry is safe: the database is restored in a single
transaction. Start the transfer again; the panel files on the new machine stay
and only their contents are refreshed.

**The new panel did not come up in time.** Check the stack log on the new
machine:

```bash
docker compose -f /opt/vortanix/deploy/docker-compose.yml logs --tail 100 api
```

**Avatars and the logo do not open after the transfer.** The `vortanix_uploads`
volume did not arrive. Make sure the `api` container was running on the source
machine during the transfer and repeat it.

**Payment gateway keys look empty after the transfer.** The `SECRETS_KEY` does
not match. This happens when the data was restored outside the transfer section.
The value travels with the secrets; compare it in `deploy/.env` on both
machines.

**A node stays offline.** From the node, check that the new relay address is
reachable and read the agent log:

```bash
docker logs vortanix-agent --tail 50
```
