import { t } from "@/lib/i18n";

export function auditActionLabel(action: string): string {
  const key = `servers.admin.audit.action.${action.replace(/\./g, "_")}`;
  const label = t(key);
  return label === key ? action : label;
}

export function ownerStatusLabel(status: string): string {
  if (status === "active") return t("admin.users.status.active");
  if (status === "disabled") return t("admin.users.status.blocked");
  return status || "—";
}

export function cpuLimitLabel(limits: Record<string, unknown>): string | null {
  const cores = Number(limits.cpu_cores ?? limits.cpu);
  if (Number.isFinite(cores) && cores > 0) return String(cores);
  const shares = Number(limits.cpu_shares);
  if (Number.isFinite(shares) && shares > 0) {
    return t("servers.admin.card.cpu_shares", { value: shares });
  }
  return null;
}
