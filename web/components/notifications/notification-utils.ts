import type { NotificationTone, PanelNotification } from "@/lib/api";
import { localeTag, t } from "@/lib/i18n";

export const TONE_ICON: Record<NotificationTone, string> = {
  ok: "bg-[var(--vx-ok-tint)] text-[var(--vx-ok)]",
  warn: "bg-[var(--vx-warn-tint)] text-[var(--vx-warn)]",
  bad: "bg-[var(--vx-danger-tint)] text-[var(--vx-danger)]",
  info: "bg-[var(--vx-info-tint)] text-[var(--vx-info)]",
};

export const TONE_ACCENT: Record<NotificationTone, string> = {
  ok: "bg-[var(--vx-ok)]",
  warn: "bg-[var(--vx-warn)]",
  bad: "bg-[var(--vx-danger)]",
  info: "bg-[var(--vx-info)]",
};

export type NotificationSection = {
  key: string;
  title: string;
  items: PanelNotification[];
};

function startOfDay(d: Date) {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

export function dayLabel(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  const diff = Math.round((startOfDay(new Date()) - startOfDay(d)) / 86_400_000);
  if (diff === 0) return t("notifications.today");
  if (diff === 1) return t("notifications.yesterday");
  return d.toLocaleDateString(localeTag(), {
    day: "numeric",
    month: "long",
    ...(d.getFullYear() !== new Date().getFullYear() ? { year: "numeric" } : {}),
  });
}

export function groupByDay(items: PanelNotification[]): NotificationSection[] {
  const sections: NotificationSection[] = [];
  for (const item of items) {
    const d = new Date(item.created_at);
    const key = Number.isNaN(d.getTime())
      ? "unknown"
      : `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
    const last = sections[sections.length - 1];
    if (last && last.key === key) {
      last.items.push(item);
    } else {
      sections.push({ key, title: dayLabel(item.created_at), items: [item] });
    }
  }
  return sections;
}

export function relativeTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const sec = Math.max(0, Math.round((Date.now() - d.getTime()) / 1000));
  if (sec < 60) return t("notifications.time.now");
  const min = Math.round(sec / 60);
  if (min < 60) return t("notifications.time.minutes", { count: min });
  const hours = Math.round(min / 60);
  if (hours < 24) return t("notifications.time.hours", { count: hours });
  return d.toLocaleTimeString(localeTag(), { hour: "2-digit", minute: "2-digit" });
}

export function fullTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString(localeTag(), {
    day: "numeric",
    month: "long",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}
