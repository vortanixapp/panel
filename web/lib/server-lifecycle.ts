import type { DashboardServer } from "@/lib/api";

export function isServerExpired(server: DashboardServer): boolean {
  if (!server.expires_at) return false;
  return new Date(server.expires_at) < new Date();
}

export function serverRuntime(server: DashboardServer): string {
  return (server.runtime_status || server.status || "").toLowerCase();
}

export function serverProvisioning(server: DashboardServer): string {
  return (server.provisioning_status || "").toLowerCase();
}

export function isServerProvisioning(server: DashboardServer): boolean {
  const prov = serverProvisioning(server);
  return ["pending", "installing", "provisioning", "reinstalling", "updating"].includes(prov);
}

export function canShowStart(server: DashboardServer): boolean {
  if (isServerExpired(server) || isServerProvisioning(server)) return false;
  if (serverProvisioning(server) === "failed") return false;
  return ["offline", "stopped", "missing"].includes(serverRuntime(server));
}

export function canShowStop(server: DashboardServer): boolean {
  if (isServerExpired(server) || isServerProvisioning(server)) return false;
  if (serverProvisioning(server) === "failed") return false;
  return serverRuntime(server) === "running";
}

export function canShowReinstall(server: DashboardServer): boolean {
  if (isServerExpired(server) || isServerProvisioning(server)) return false;
  const runtime = serverRuntime(server);
  return ["offline", "stopped", "missing"].includes(runtime) || serverProvisioning(server) === "failed";
}

export function canShowUpdate(server: DashboardServer): boolean {
  if (!server.steam_updatable || isServerExpired(server) || isServerProvisioning(server)) return false;
  if (serverProvisioning(server) === "failed") return false;
  return ["offline", "stopped", "missing"].includes(serverRuntime(server));
}

export function canSwitchVersion(server: DashboardServer): boolean {
  const versions = server.available_game_versions ?? [];
  if (versions.length === 0 || isServerExpired(server) || isServerProvisioning(server)) return false;
  return ["offline", "stopped", "missing"].includes(serverRuntime(server));
}
