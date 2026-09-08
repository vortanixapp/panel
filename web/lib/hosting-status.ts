import { getLocale, t } from "@/lib/i18n";

export type HostingStatusColor = "emerald" | "rose" | "amber" | "sky";

export function getHostingStatusMeta(status: string): {
  label: string;
  color: HostingStatusColor;
} {
  switch (status) {
    case "active":
      return { label: t("billing.hosting.status.active"), color: "emerald" };
    case "pending":
      return { label: t("billing.hosting.status.pending"), color: "amber" };
    case "suspended":
      return { label: t("billing.hosting.status.suspended"), color: "rose" };
    case "terminated":
      return { label: t("billing.hosting.status.terminated"), color: "rose" };
    case "error":
      return { label: t("billing.hosting.status.error"), color: "rose" };
    default:
      return { label: status, color: "sky" };
  }
}

export function hostingStatusColorClass(color: string): string {
  switch (color) {
    case "emerald":
      return "bg-emerald-500/10 text-emerald-500";
    case "rose":
      return "bg-rose-500/10 text-rose-500";
    case "amber":
      return "bg-amber-500/10 text-amber-500";
    case "sky":
      return "bg-sky-500/10 text-sky-500";
    default:
      return "bg-muted text-muted-foreground";
  }
}

export function getHostingStatusBadgeClass(status: string): string {
  const meta = getHostingStatusMeta(status);
  return hostingStatusColorClass(meta.color);
}

export function getHostingStatusLabel(
  status: string,
  statusLabel?: string | null
): string {
  return statusLabel || getHostingStatusMeta(status).label;
}

export function getHostingStatusColor(
  status: string,
  statusColor?: string | null
): string {
  return statusColor || getHostingStatusMeta(status).color;
}

export function getHostingDisplayDomain(account: {
  domain?: string | null;
  primary_domain?: string | null;
  username: string;
}): string {
  return account.domain || account.primary_domain || account.username;
}

export function formatHostingDate(value: string | null | undefined): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString(getLocale() === "en" ? "en-GB" : "ru");
}

export function formatHostingDateTime(value: string | null | undefined): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString(getLocale() === "en" ? "en-GB" : "ru");
}

export function isHostingExpired(expiresAt: string | null | undefined): boolean {
  if (!expiresAt) return false;
  const date = new Date(expiresAt);
  return !Number.isNaN(date.getTime()) && date < new Date();
}

export function planDiskLabel(plan?: {
  disk_gb?: number | null;
  disk_mb?: number | null;
} | null): string {
  if (!plan) return "0 GB";
  if (plan.disk_gb != null && plan.disk_gb > 0) return `${plan.disk_gb} GB`;
  if (plan.disk_mb != null && plan.disk_mb >= 1024) {
    return `${Math.round(plan.disk_mb / 1024)} GB`;
  }
  if (plan.disk_mb != null) return `${plan.disk_mb} MB`;
  return "0 GB";
}

export function planFeatureLimit(value: number | null | undefined): string {
  return value !== null && value !== undefined ? String(value) : "∞";
}
