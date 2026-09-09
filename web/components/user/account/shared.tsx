"use client";

import { MON } from "@/components/user/monitoring/shared";
import { localeTag, t } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export {
  Chip,
  MON,
  MON_BTN,
  MON_BTN_PRIMARY,
  MON_CARD,
  MON_INNER,
} from "@/components/user/monitoring/shared";

export type Tone = "info" | "ok" | "warn" | "bad";

export function toneColor(tone: Tone | string): string {
  switch (tone) {
    case "ok":
      return MON.ok;
    case "warn":
      return MON.warn;
    case "bad":
      return MON.bad;
    default:
      return MON.info;
  }
}

export function EventIcon({
  icon,
  tone = "info",
  size = 32,
  className,
}: {
  icon: string;
  tone?: Tone | string;
  size?: number;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "flex shrink-0 items-center justify-center rounded-[9px] bg-[var(--vx-veil)]",
        className
      )}
      style={{ width: size, height: size, color: toneColor(tone) }}
    >
      <i className={icon} style={{ fontSize: Math.round(size * 0.47) }} />
    </span>
  );
}

export function CategoryBadge({ children }: { children: React.ReactNode }) {
  return (
    <span className="inline-flex items-center gap-1.5 whitespace-nowrap rounded-full bg-[var(--vx-veil)] px-2 py-[2px] text-[11px] font-semibold text-[var(--vx-muted)]">
      {children}
    </span>
  );
}

export function StatCard({
  label,
  icon,
  value,
  sub,
  color = MON.fg,
}: {
  label: string;
  icon: string;
  value: string;
  sub: string;
  color?: string;
}) {
  return (
    <div className="rounded-[12px] border border-[var(--vx-border)] bg-[var(--vx-card)] px-4 py-3.5">
      <div className="flex items-center justify-between">
        <span className="text-[12px] text-[var(--vx-muted)]">{label}</span>
        <i className={cn(icon, "text-[15px] text-[var(--vx-faint)]")} />
      </div>
      <div className="mt-2 flex items-baseline gap-[7px]">
        <span className="text-[24px] font-bold tracking-[-0.02em]" style={{ color }}>
          {value}
        </span>
        <span className="text-[11px] text-[var(--vx-muted)]">{sub}</span>
      </div>
    </div>
  );
}

export function ListEmpty({
  icon,
  title,
  text,
  className,
}: {
  icon: string;
  title: string;
  text: string;
  className?: string;
}) {
  return (
    <div className={cn("px-4 py-14 text-center", className)}>
      <i className={cn(icon, "text-[30px] text-[var(--vx-faint)]")} />
      <div className="mt-3 text-[14px] font-semibold">{title}</div>
      <p className="mt-1.5 text-[12px] text-[var(--vx-muted)]">{text}</p>
    </div>
  );
}

const DAY_MS = 86_400_000;

export function formatTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleTimeString(localeTag(), {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function formatDateTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return `${d.toLocaleDateString(localeTag())} ${d.toLocaleTimeString(localeTag(), {
    hour: "2-digit",
    minute: "2-digit",
  })}`;
}

export function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleDateString(localeTag());
}

export function timeAgo(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const sec = Math.max(0, Math.round((Date.now() - d.getTime()) / 1000));
  if (sec < 60) return t("dashboard.time.just_now");
  const min = Math.round(sec / 60);
  if (min < 60) return t("dashboard.time.minutes_ago", { count: min });
  const hours = Math.round(min / 60);
  if (hours < 24) return t("dashboard.time.hours_ago", { count: hours });
  const days = Math.round(hours / 24);
  if (days === 1) return t("dashboard.time.yesterday");
  return t(
    plural(
      days,
      "dashboard.time.days_ago_one",
      "dashboard.time.days_ago_few",
      "dashboard.time.days_ago_many"
    ),
    { count: days }
  );
}

export function dayGroupTitle(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  const startOf = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const diff = Math.round((startOf(new Date()) - startOf(d)) / DAY_MS);
  const date = d.toLocaleDateString(localeTag());
  if (diff === 0) return t("dashboard.group.today", { date });
  if (diff === 1) return t("dashboard.group.yesterday", { date });
  return date;
}

export function dayKey(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "unknown";
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
}

export function plural(n: number, one: string, few: string, many: string): string {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return one;
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return few;
  return many;
}

export function countdown(targetIso: string | null, now: number): string {
  if (!targetIso) return "";
  const target = new Date(targetIso).getTime();
  if (Number.isNaN(target)) return "";
  const left = Math.max(0, Math.floor((target - now) / 1000));
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(Math.floor(left / 3600))}:${pad(Math.floor((left % 3600) / 60))}:${pad(left % 60)}`;
}

export function formatMoney(value: number): string {
  return new Intl.NumberFormat(localeTag(), { maximumFractionDigits: 2 }).format(value);
}
