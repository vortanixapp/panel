import type { AgentsList } from "@/lib/api";
import { localeTag } from "@/lib/i18n";

export function validDate(value?: string | null): Date | null {
  if (!value || value.startsWith("0001-")) return null;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}

export function formatDateTime(value?: string | null): string {
  const date = validDate(value);
  if (!date) return "—";
  return date.toLocaleString(localeTag(), {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatDay(value?: string | null): string {
  const date = validDate(value);
  if (!date) return "—";
  return date.toLocaleDateString(localeTag(), { day: "numeric", month: "long", year: "numeric" });
}

export function formatRelative(value?: string | null, now = Date.now()): string {
  const date = validDate(value);
  if (!date) return "—";
  const rtf = new Intl.RelativeTimeFormat(localeTag(), { numeric: "auto" });
  const seconds = Math.round((date.getTime() - now) / 1000);
  const abs = Math.abs(seconds);
  if (abs < 45) return rtf.format(0, "second");
  if (abs < 3600) return rtf.format(Math.round(seconds / 60), "minute");
  if (abs < 86400) return rtf.format(Math.round(seconds / 3600), "hour");
  if (abs < 86400 * 30) return rtf.format(Math.round(seconds / 86400), "day");
  return formatDay(value);
}

export function formatClock(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000));
  const minutes = Math.floor(total / 60);
  const seconds = total % 60;
  return `${minutes}:${String(seconds).padStart(2, "0")}`;
}

function parseVersion(value: string): number[] | null {
  const clean = value.trim().replace(/^v/i, "").split(/[-+]/)[0];
  const parts = clean.split(".");
  if (!clean || parts.length > 3 || parts.some((p) => !/^\d+$/.test(p))) return null;
  return [0, 1, 2].map((i) => Number(parts[i] ?? 0));
}

export function isNewerVersion(candidate?: string | null, current?: string | null): boolean {
  const a = parseVersion(candidate ?? "");
  const b = parseVersion(current ?? "");
  if (!a || !b) return false;
  for (let i = 0; i < 3; i++) {
    if (a[i] !== b[i]) return a[i] > b[i];
  }
  return false;
}

export const STEPS = ["backup", "pull", "restart", "health"] as const;
export type StepId = (typeof STEPS)[number];

const STEP_MARKERS: { step: StepId; pattern: RegExp }[] = [
  { step: "backup", pattern: /Копия базы|копия базы пропущена|Копия готова/ },
  { step: "pull", pattern: /Получение выпуска|Загрузка образов|закреплена версия/ },
  { step: "restart", pattern: /Перезапуск служб|пересоздаю caddy/ },
  { step: "health", pattern: /Ожидание готовности API/ },
];

export function reachedStep(log: string[]): number {
  let reached = -1;
  for (const line of log) {
    for (const marker of STEP_MARKERS) {
      if (marker.pattern.test(line)) {
        reached = Math.max(reached, STEPS.indexOf(marker.step));
      }
    }
  }
  return reached;
}

export function splitLogLine(line: string): { time: string; text: string } {
  const match = /^(\d{2}:\d{2}:\d{2})\s+(.*)$/.exec(line);
  return match ? { time: match[1], text: match[2] } : { time: "", text: line };
}

export type AgentsSummary = {
  total: number;
  current: number;
};

export function summarizeAgents(list: AgentsList): AgentsSummary {
  const s = list.summary;
  return { total: s.total, current: Math.max(0, s.total - s.outdated - s.never_connected) };
}

export function agentsBusy(list: AgentsList): boolean {
  return list.summary.updating > 0 || list.summary.restarting > 0;
}
