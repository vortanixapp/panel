export const panelTransfer = {
  "admin.panel_transfer.title": "Panel transfer",
  "admin.panel_transfer.subtitle":
    "Move the whole panel to another server: database, secrets and uploaded files",

  "admin.panel_transfer.tab.ssh": "Over SSH",
  "admin.panel_transfer.tab.archive": "By archive",

  "admin.panel_transfer.overview.title": "What moves",
  "admin.panel_transfer.overview.version": "Panel version",
  "admin.panel_transfer.overview.db": "Database",
  "admin.panel_transfer.overview.uploads": "Files",
  "admin.panel_transfer.overview.secrets": "Secrets in .env",
  "admin.panel_transfer.overview.address": "Current address",
  "admin.panel_transfer.overview.mode": "Install mode",
  "admin.panel_transfer.overview.nodes": "Locations",

  "admin.panel_transfer.skipped.title": "What stays behind",
  "admin.panel_transfer.skipped.redis": "Redis cache — rebuilt on the new server",
  "admin.panel_transfer.skipped.certs": "Let's Encrypt certificate — issued again",
  "admin.panel_transfer.skipped.servers": "Game server files — stay on the locations",

  "admin.panel_transfer.ssh.title": "New server",
  "admin.panel_transfer.ssh.hint":
    "A clean Debian or Ubuntu with root access and free ports 80 and 443. The panel installs itself, then the database, secrets and files move in",

  "admin.panel_transfer.target.host": "Server address",
  "admin.panel_transfer.target.port": "SSH port",
  "admin.panel_transfer.target.user": "User",
  "admin.panel_transfer.target.auth_password": "Password",
  "admin.panel_transfer.target.auth_key": "Private key",
  "admin.panel_transfer.target.password": "SSH password",
  "admin.panel_transfer.target.private_key": "SSH private key",

  "admin.panel_transfer.address.same": "Same domain",
  "admin.panel_transfer.address.same_hint":
    "The panel on the new server will answer on {address}. Switch DNS after the transfer — the nodes reconnect on their own",
  "admin.panel_transfer.address.new": "New panel address",

  "admin.panel_transfer.freeze.label": "Stop writes during the transfer",
  "admin.panel_transfer.freeze.hint":
    "This panel becomes read-only so the data does not drift from the copy. The mode stays on after the transfer",

  "admin.panel_transfer.start": "Start transfer",
  "admin.panel_transfer.starting": "Starting…",
  "admin.panel_transfer.started": "Transfer started",
  "admin.panel_transfer.cancel": "Cancel transfer",
  "admin.panel_transfer.cancelled": "Transfer cancelled",
  "admin.panel_transfer.unfreeze": "Turn off read-only",
  "admin.panel_transfer.frozen_banner": "The panel is in read-only mode",

  "admin.panel_transfer.confirm.start":
    "Start moving the panel to another server? It will install the same panel version there and move the database, secrets and files in. This panel keeps running",
  "admin.panel_transfer.confirm.start_ok": "Start",
  "admin.panel_transfer.confirm.unfreeze":
    "Turn off read-only mode? If the panel has already moved, changes made here will not reach the new server",
  "admin.panel_transfer.confirm.unfreeze_ok": "Turn off",

  "admin.panel_transfer.progress.title": "Transfer progress",
  "admin.panel_transfer.progress.bytes": "Sent {done} of {total}",
  "admin.panel_transfer.progress.agents": "Nodes switched",
  "admin.panel_transfer.progress.running_hint":
    "You can close this page — the transfer runs on the server and continues without it",

  "admin.panel_transfer.status.pending": "Queued",
  "admin.panel_transfer.status.running": "Running",
  "admin.panel_transfer.status.completed": "Completed",
  "admin.panel_transfer.status.failed": "Failed",
  "admin.panel_transfer.status.cancelled": "Cancelled",

  "admin.panel_transfer.stage.prepare": "Preparing",
  "admin.panel_transfer.stage.freeze": "Read-only",
  "admin.panel_transfer.stage.probe_target": "Checking server",
  "admin.panel_transfer.stage.install_docker": "Docker",
  "admin.panel_transfer.stage.clone": "Panel files",
  "admin.panel_transfer.stage.init_env": "Settings",
  "admin.panel_transfer.stage.compose_up": "Start",
  "admin.panel_transfer.stage.transfer": "Sending data",
  "admin.panel_transfer.stage.health": "Checking panel",
  "admin.panel_transfer.stage.relay_cert": "Relay certificate",
  "admin.panel_transfer.stage.agents": "Nodes",
  "admin.panel_transfer.stage.done": "Done",

  "admin.panel_transfer.export.title": "Download archive",
  "admin.panel_transfer.export.hint":
    "The archive holds the database, secrets and uploaded files. It can be loaded into a clean panel of the same or a newer version",
  "admin.panel_transfer.export.password": "Archive password (12 characters or more)",
  "admin.panel_transfer.export.warning":
    "The archive contains SECRETS_KEY, which encrypts payment gateway keys and location passwords. Keep the file like a password",
  "admin.panel_transfer.export.action": "Download archive",
  "admin.panel_transfer.export.busy": "Building archive…",
  "admin.panel_transfer.export.done": "Archive is ready",
  "admin.panel_transfer.export.failed": "Archive was not created",

  "admin.panel_transfer.import.title": "Upload archive",
  "admin.panel_transfer.import.hint":
    "The data of this panel will be replaced by the archive. Do this only on the new panel",
  "admin.panel_transfer.import.file": "Archive file",
  "admin.panel_transfer.import.password": "Archive password",
  "admin.panel_transfer.import.inspect": "Check archive",
  "admin.panel_transfer.import.inspect_failed": "Archive could not be read",
  "admin.panel_transfer.import.action": "Replace panel data",
  "admin.panel_transfer.import.busy": "Restoring…",
  "admin.panel_transfer.import.done": "Data restored",
  "admin.panel_transfer.import.failed": "Data was not restored",
  "admin.panel_transfer.import.warning":
    "The current database, secrets and files of this panel will be overwritten. The encryption key also comes from the archive",
  "admin.panel_transfer.import.archive_version": "Panel version",
  "admin.panel_transfer.import.archive_created": "Created",
  "admin.panel_transfer.import.archive_db": "Database",
  "admin.panel_transfer.import.archive_uploads": "Files",
  "admin.panel_transfer.import.archive_source": "Source address",

  "admin.panel_transfer.agents.title": "Nodes did not switch",
  "admin.panel_transfer.agents.hint":
    "Nodes left behind: {count}. Enter the access to the new server and retry — the panel will recreate the agents with the new address",
  "admin.panel_transfer.agents.retry": "Retry switching",
  "admin.panel_transfer.agents.started": "Node switching started",

  "admin.panel_transfer.nodes.title": "Locations",
  "admin.panel_transfer.nodes.empty": "No locations",
};
