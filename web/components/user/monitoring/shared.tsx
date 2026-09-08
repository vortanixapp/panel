"use client";

import { localeTag, t } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import type {
  MonitoringIncident,
  MonitoringPlayer,
  MonitoringServerRow,
  MonitoringSettings,
  MonitoringUptimeDay,
} from "@/lib/api";

export const MON = {
  fg: "var(--vx-fg)",
  mut: "var(--vx-muted)",
  dim: "var(--vx-dim)",
  faint: "var(--vx-faint)",
  ok: "var(--vx-ok)",
  warn: "var(--vx-warn)",
  bad: "var(--vx-danger)",
  info: "var(--vx-info)",
  primary: "var(--vx-fg-strong)",
} as const;

export const MON_CARD = "rounded-[14px] border border-[var(--vx-border)] bg-[var(--vx-card)]";
export const MON_INNER = "rounded-[12px] border border-[var(--vx-border)] bg-[var(--vx-bg)]";
export const MON_MUTED = "text-[var(--vx-muted)]";
export const MON_TH =
  "px-3 py-2.5 text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--vx-muted)] border-b border-[var(--vx-border)]";
export const MON_ROW = "border-b border-[var(--vx-divider)] transition-colors hover:bg-[var(--vx-card-2)]";
export const MON_INPUT =
  "w-full rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-inset)] px-3 py-2.5 text-[13px] text-[var(--vx-fg)] outline-none transition-colors focus:border-[var(--vx-border-hover)]";

const num = (v: unknown) => (Number.isFinite(Number(v)) ? Number(v) : 0);
const str = (v: unknown) => (typeof v === "string" ? v : "");
const list = <T,>(v: unknown): T[] => (Array.isArray(v) ? (v as T[]) : []);

export function normalizeServerRow(raw: Partial<MonitoringServerRow> | null | undefined): MonitoringServerRow {
  const r = (raw ?? {}) as Record<string, unknown>;
  return {
    id: str(r.id),
    name: str(r.name) || t("monitoring.server.untitled"),
    game_id: str(r.game_id),
    game_name: str(r.game_name),
    version: str(r.version),
    map: str(r.map),
    region: str(r.region),
    status: str(r.status) || "stopped",
    online: num(r.online),
    slots: num(r.slots),
    ping: num(r.ping),
    cpu: num(r.cpu),
    ram: num(r.ram),
    ram_limit_mb: num(r.ram_limit_mb),
    tps: num(r.tps),
    uptime: num(r.uptime),
    ip: str(r.ip),
    spark: list<number>(r.spark).map(num),
    public_enabled: Boolean(r.public_enabled),
    description: str(r.description),
    tags: list<string>(r.tags).map(str).filter(Boolean),
    discord: str(r.discord),
    website: str(r.website),
    votes: num(r.votes),
  };
}

export function normalizeSettings(raw: Partial<MonitoringSettings> | null | undefined): MonitoringSettings {
  const r = (raw ?? {}) as Record<string, unknown>;
  const flag = (v: unknown, fallback: boolean) => (typeof v === "boolean" ? v : fallback);
  return {
    public_enabled: flag(r.public_enabled, false),
    title: str(r.title),
    description: str(r.description),
    tags: list<string>(r.tags).map(str).filter(Boolean),
    discord: str(r.discord),
    website: str(r.website),
    show_players: flag(r.show_players, true),
    show_chart: flag(r.show_chart, true),
    show_incidents: flag(r.show_incidents, true),
    show_address: flag(r.show_address, true),
    show_version: flag(r.show_version, false),
    votes: num(r.votes),
  };
}

export function normalizePlayers(raw: unknown): MonitoringPlayer[] {
  return list<Record<string, unknown>>(raw)
    .map((p) => ({
      name: str(p.name),
      score: num(p.score),
      ping: num(p.ping),
      duration_sec: num(p.duration_sec),
    }))
    .filter((p) => p.name !== "");
}

export function normalizeIncidents(raw: unknown): MonitoringIncident[] {
  return list<Record<string, unknown>>(raw).map((i, idx) => ({
    id: num(i.id) || idx,
    title: str(i.title) || t("monitoring.incidents.fallback_title"),
    level: (["info", "warn", "bad"] as const).includes(i.level as "info")
      ? (i.level as MonitoringIncident["level"])
      : "info",
    body: str(i.body),
    started_at: str(i.started_at),
    duration_sec: num(i.duration_sec),
    resolved: Boolean(i.resolved),
  }));
}

export function normalizeUptimeDays(raw: unknown): MonitoringUptimeDay[] {
  return list<Record<string, unknown>>(raw).map((d) => ({
    day: str(d.day),
    uptime: num(d.uptime),
    has_data: Boolean(d.has_data),
  }));
}

export function gameIconSrc(gameId: string) {
  return `/games/${gameId || "mcjava"}.svg`;
}

export type MonStatusTone = { label: string; color: string; bg: string };

export function statusTone(status: string): MonStatusTone {
  switch (status) {
    case "running":
      return { label: t("monitoring.status.online"), color: MON.ok, bg: "var(--vx-ok-tint)" };
    case "starting":
    case "installing":
    case "reinstalling":
      return { label: t("monitoring.status.starting"), color: MON.warn, bg: "var(--vx-warn-tint)" };
    case "stopping":
      return { label: t("monitoring.status.stopping"), color: MON.warn, bg: "var(--vx-warn-tint)" };
    case "error":
      return { label: t("monitoring.status.error"), color: MON.bad, bg: "var(--vx-danger-tint)" };
    default:
      return { label: t("monitoring.status.offline"), color: MON.mut, bg: "var(--vx-tint)" };
  }
}

export function StatusPill({ status, className }: { status: string; className?: string }) {
  const tone = statusTone(status);
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-[3px] text-[11px] font-semibold",
        className
      )}
      style={{ background: tone.bg, color: tone.color }}
    >
      <span className="h-1.5 w-1.5 rounded-full" style={{ background: tone.color }} />
      {tone.label}
    </span>
  );
}

export function pingColor(ping: number): string {
  if (!ping) return MON.mut;
  if (ping < 30) return MON.ok;
  if (ping < 60) return MON.dim;
  return MON.warn;
}

export function pingText(ping: number): string {
  return ping ? t("monitoring.unit.ms", { value: ping }) : "—";
}

export function tpsText(tps: number): string {
  return tps ? Number(tps).toFixed(1) : "—";
}

export function uptimeText(uptime: number): string {
  return Number(uptime) > 0 ? `${Number(uptime).toFixed(2)}%` : "—";
}

export function loadPct(online: number, slots: number): number {
  if (!slots) return 0;
  return Math.min(100, Math.round(((online || 0) / slots) * 100));
}

export function initials(name: string): string {
  const letters = (name || "").replace(/[^A-Za-zА-Яа-я]/g, "").slice(0, 2);
  return letters ? letters.toUpperCase() : "??";
}

export function secondsToHms(sec: number): string {
  const s = Math.max(0, Math.floor(Number(sec) || 0));
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(Math.floor(s / 3600))}:${pad(Math.floor((s % 3600) / 60))}:${pad(s % 60)}`;
}

export function durationText(sec: number): string {
  const total = Math.max(0, Math.floor(Number(sec) || 0));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  if (hours > 0) {
    return minutes > 0
      ? t("monitoring.duration.hours_minutes", { hours, minutes })
      : t("monitoring.duration.hours", { hours });
  }
  if (minutes > 0) return t("monitoring.duration.minutes", { minutes });
  return t("monitoring.duration.seconds", { seconds: total });
}

export function shortDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleDateString(localeTag(), { day: "numeric", month: "short" });
}

export function relativeUpdate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  const sec = Math.max(0, Math.round((Date.now() - d.getTime()) / 1000));
  if (sec < 60) return t("monitoring.updated.seconds_ago", { value: sec });
  const min = Math.round(sec / 60);
  if (min < 60) return t("monitoring.updated.minutes_ago", { value: min });
  return t("monitoring.updated.hours_ago", { value: Math.round(min / 60) });
}

export function polyPoints(values: number[], w: number, h: number, cap?: number): string {
  if (!Array.isArray(values) || values.length === 0) return "";
  const max = Math.max(cap || 0, ...values, 1);
  if (values.length === 1) return `0,${(h - (values[0] / max) * (h - 2) - 1).toFixed(1)}`;
  return values
    .map((v, i) => {
      const x = (i / (values.length - 1)) * w;
      const y = h - (v / max) * (h - 2) - 1;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");
}

export function areaPath(values: number[], w: number, h: number, cap?: number): string {
  const pts = polyPoints(values, w, h, cap);
  if (!pts) return "";
  return `M0,${h} L${pts.split(" ").join(" L")} L${w},${h} Z`;
}

export function Sparkline({
  values,
  cap,
  className,
  stroke = MON.primary,
  filled = true,
}: {
  values: number[];
  cap?: number;
  className?: string;
  stroke?: string;
  filled?: boolean;
}) {
  const data = Array.isArray(values) && values.length > 0 ? values : [0, 0];
  return (
    <svg viewBox="0 0 100 30" preserveAspectRatio="none" className={cn("block", className)}>
      {filled && <path d={areaPath(data, 100, 30, cap)} fill="var(--vx-veil-strong)" />}
      <polyline
        points={polyPoints(data, 100, 30, cap)}
        fill="none"
        stroke={stroke}
        strokeWidth={1.2}
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}

export function LoadBar({ pct, className }: { pct: number; className?: string }) {
  const width = Math.max(0, Math.min(100, pct));
  return (
    <div className={cn("h-1 overflow-hidden rounded-full bg-[var(--vx-tint)]", className)}>
      <div
        className="h-full rounded-full transition-[width] duration-500"
        style={{ width: `${width}%`, background: width > 85 ? MON.warn : MON.primary }}
      />
    </div>
  );
}

export function Chip({
  active,
  className,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { active?: boolean }) {
  return (
    <button
      type="button"
      className={cn(
        "flex items-center gap-1.5 rounded-[6px] px-3 py-1.5 text-[12px] font-medium transition-colors",
        active ? "bg-[var(--vx-tint)] text-[var(--vx-fg)]" : "text-[var(--vx-muted)] hover:text-[var(--vx-fg)]",
        className
      )}
      {...props}
    />
  );
}

export const MON_BTN =
  "flex h-9 items-center gap-1.5 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-card)] px-3.5 text-[13px] font-medium text-[var(--vx-fg)] transition-colors hover:bg-[var(--vx-tint)] disabled:cursor-not-allowed disabled:opacity-55";
export const MON_BTN_PRIMARY =
  "flex h-9 items-center gap-1.5 rounded-[8px] bg-[var(--vx-fg-strong)] px-3.5 text-[13px] font-semibold text-[var(--vx-on-fill)] transition-colors hover:bg-white disabled:cursor-not-allowed disabled:opacity-55";

export function CopyAddress({
  address,
  className,
  onCopied,
}: {
  address: string;
  className?: string;
  onCopied?: () => void;
}) {
  if (!address) {
    return (
      <span className="font-mono text-[11px] text-[var(--vx-faint)]">
        {t("monitoring.address_hidden")}
      </span>
    );
  }
  return (
    <button
      type="button"
      onClick={() => {
        void navigator.clipboard?.writeText(address);
        onCopied?.();
      }}
      className={cn(
        "flex items-center gap-1.5 rounded-[6px] border border-[var(--vx-border)] bg-[var(--vx-bg)] px-2 py-[5px] font-mono text-[11px] text-[var(--vx-dim)] transition-colors hover:bg-[var(--vx-tint)] hover:text-white",
        className
      )}
    >
      {address}
      <i className="ri-file-copy-line text-[12px]" />
    </button>
  );
}
