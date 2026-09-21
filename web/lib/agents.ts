import { t, localeTag } from "@/lib/i18n";
import type { AgentLabel, AgentRow, AgentState } from "@/lib/api";

export type AgentTone = "ok" | "warn" | "danger" | "muted" | "info";

export const TONE_CLASS: Record<AgentTone, string> = {
  ok: "border-[color-mix(in_srgb,var(--vx-ok)_30%,transparent)] bg-[var(--vx-ok-tint)] text-[var(--vx-ok)]",
  warn: "border-[color-mix(in_srgb,var(--vx-warn)_30%,transparent)] bg-[var(--vx-warn-tint)] text-[var(--vx-warn)]",
  danger: "border-[color-mix(in_srgb,var(--vx-danger)_30%,transparent)] bg-[var(--vx-danger-tint)] text-[var(--vx-danger)]",
  info: "border-[color-mix(in_srgb,var(--vx-info)_30%,transparent)] bg-[var(--vx-info-tint)] text-[var(--vx-info)]",
  muted: "border-[var(--vx-border-2)] bg-transparent text-[var(--vx-muted)]",
};

export function stateTone(state: AgentState): AgentTone {
  if (state === "online") return "ok";
  if (state === "offline") return "danger";
  return "muted";
}

export function labelTone(label: AgentLabel): AgentTone {
  switch (label) {
    case "update_failed":
    case "disk_low":
      return "danger";
    case "updating":
    case "restarting":
      return "info";
    default:
      return "warn";
  }
}

export const AGENT_FILTERS = [
  "all",
  "problems",
  "online",
  "offline",
  "outdated",
  "updating",
  "update_failed",
  "disk_low",
  "never_connected",
] as const;

export type AgentFilter = (typeof AGENT_FILTERS)[number];

export function isProblem(row: AgentRow): boolean {
  return (
    row.state === "offline" ||
    row.labels.includes("update_failed") ||
    row.labels.includes("disk_low") ||
    row.labels.includes("outdated") ||
    row.labels.includes("legacy")
  );
}

export function matchesFilter(row: AgentRow, filter: AgentFilter): boolean {
  switch (filter) {
    case "all":
      return true;
    case "problems":
      return isProblem(row);
    case "online":
    case "offline":
    case "never_connected":
      return row.state === filter;
    default:
      return row.labels.includes(filter);
  }
}

export function matchesQuery(row: AgentRow, query: string): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return true;
  return [row.name, row.code, row.host, row.country, row.region, row.version, row.remote_addr ?? ""]
    .join(" ")
    .toLowerCase()
    .includes(q);
}

function trimNumber(v: number): string {
  return new Intl.NumberFormat(localeTag(), { maximumFractionDigits: v >= 100 ? 0 : 1 }).format(v);
}

export function formatMB(mb: number | null | undefined): string {
  if (mb == null || !Number.isFinite(mb)) return "—";
  if (mb >= 1024 * 1024) return `${trimNumber(mb / 1024 / 1024)} ${t("admin.agents.unit.tb")}`;
  if (mb >= 1024) return `${trimNumber(mb / 1024)} ${t("admin.agents.unit.gb")}`;
  return `${trimNumber(mb)} ${t("admin.agents.unit.mb")}`;
}

export function formatBytes(bytes: number | null | undefined): string {
  if (bytes == null || !Number.isFinite(bytes)) return "—";
  if (bytes < 1024 * 1024) return `${trimNumber(bytes / 1024)} ${t("admin.agents.unit.kb")}`;
  return formatMB(bytes / 1024 / 1024);
}

export function formatDuration(sec: number | null | undefined): string {
  if (sec == null || !Number.isFinite(sec) || sec < 0) return "—";
  const s = Math.floor(sec);
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return t("admin.agents.duration.dh", { d, h });
  if (h > 0) return t("admin.agents.duration.hm", { h, m });
  if (m > 0) return t("admin.agents.duration.m", { m });
  return t("admin.agents.duration.s", { s });
}

export function formatAgo(iso: string | null | undefined): string {
  if (!iso) return "—";
  const ms = Date.now() - new Date(iso).getTime();
  if (!Number.isFinite(ms)) return "—";
  if (ms < 60_000) return t("admin.agents.ago.now");
  return t("admin.agents.ago.value", { value: formatDuration(ms / 1000) });
}

export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString(localeTag(), {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function pct(v: number | null | undefined): number | null {
  return v == null || !Number.isFinite(v) ? null : Math.max(0, Math.min(100, v));
}

export function diskUsedPct(row: Pick<AgentRow, "resources">): number | null {
  const r = row.resources;
  if (r.disk_percent != null) return pct(r.disk_percent);
  if (r.disk_total_mb && r.disk_free_mb != null) {
    return pct(((r.disk_total_mb - r.disk_free_mb) / r.disk_total_mb) * 100);
  }
  return null;
}

export function agentErrorText(code: string, fallback: string): string {
  const key = `admin.agents.error.${code}`;
  const text = code ? t(key) : "";
  return text && text !== key ? text : fallback;
}
