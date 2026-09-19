import { localeTag } from "@/lib/i18n";
import type { BugReportSeverity } from "@/lib/api";

export const DRAFT_KEY = "vx.admin.bug-report.draft";

export const NODE_ALL = "__all__";

export const COMPONENTS = [
  "panel-ui",
  "core-api",
  "worker",
  "agent",
  "agent-relay",
  "console-gateway",
  "metrics-ingest",
  "status-page",
  "updater",
  "game-image",
  "unknown",
];

export const SEVERITIES: { key: BugReportSeverity; dot: string }[] = [
  { key: "low", dot: "var(--vx-ink-faint)" },
  { key: "medium", dot: "var(--vx-warn)" },
  { key: "high", dot: "color-mix(in srgb, var(--vx-danger) 65%, var(--vx-warn))" },
  { key: "critical", dot: "var(--vx-danger)" },
];

export type Draft = {
  title?: string;
  desc?: string;
  severity?: BugReportSeverity;
  component?: string;
  node?: string;
};

export function readDraft(): Draft | null {
  try {
    const raw = localStorage.getItem(DRAFT_KEY);
    if (!raw) return null;
    const draft = JSON.parse(raw) as Draft;
    return draft && typeof draft === "object" ? draft : null;
  } catch {
    return null;
  }
}

export function writeDraft(draft: Draft) {
  try {
    if (!draft.title?.trim() && !draft.desc?.trim()) localStorage.removeItem(DRAFT_KEY);
    else localStorage.setItem(DRAFT_KEY, JSON.stringify(draft));
  } catch {}
}

export function clearDraft() {
  try {
    localStorage.removeItem(DRAFT_KEY);
  } catch {}
}

const UA_RULES: { re: RegExp; name: string }[] = [
  { re: /Firefox\/([\d.]+)/, name: "Firefox" },
  { re: /Edg\/([\d.]+)/, name: "Edge" },
  { re: /OPR\/([\d.]+)/, name: "Opera" },
  { re: /YaBrowser\/([\d.]+)/, name: "Yandex Browser" },
  { re: /Chrome\/([\d.]+)/, name: "Chrome" },
  { re: /Version\/([\d.]+).+Safari/, name: "Safari" },
];

const OS_RULES: { re: RegExp; name: string }[] = [
  { re: /Windows NT 10\.0/, name: "Windows 10/11" },
  { re: /Windows NT ([\d.]+)/, name: "Windows" },
  { re: /Android ([\d.]+)/, name: "Android" },
  { re: /iPhone OS ([\d_]+)/, name: "iOS" },
  { re: /Mac OS X ([\d_]+)/, name: "macOS" },
  { re: /Linux/, name: "Linux" },
];

export function browserLabel(ua: string): string {
  if (!ua) return "";
  let browser = "";
  for (const rule of UA_RULES) {
    const m = rule.re.exec(ua);
    if (m) {
      browser = `${rule.name} ${(m[1] ?? "").split(".")[0]}`.trim();
      break;
    }
  }
  let os = "";
  for (const rule of OS_RULES) {
    if (rule.re.test(ua)) {
      os = rule.name;
      break;
    }
  }
  if (browser && os) return `${browser} / ${os}`;
  return browser || os || ua.slice(0, 60);
}

export function screenLabel(): string {
  if (typeof window === "undefined") return "";
  const ratio = Math.round((window.devicePixelRatio || 1) * 100) / 100;
  const size = `${window.innerWidth}×${window.innerHeight}`;
  return ratio === 1 ? size : `${size} @${ratio}x`;
}

export function nowLabel(): string {
  const now = new Date();
  const offset = -now.getTimezoneOffset() / 60;
  const sign = offset >= 0 ? "+" : "−";
  return `${now.toLocaleString(localeTag(), {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })} UTC${sign}${Math.abs(offset)}`;
}

export function versionTag(value: string): string {
  const v = value.trim();
  if (!v) return "";
  return /^\d+\.\d+/.test(v) ? `v${v}` : v;
}

export async function copyText(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return true;
    }
  } catch {}
  try {
    const area = document.createElement("textarea");
    area.value = text;
    area.setAttribute("readonly", "");
    area.style.position = "fixed";
    area.style.opacity = "0";
    document.body.appendChild(area);
    area.select();
    const ok = document.execCommand("copy");
    document.body.removeChild(area);
    return ok;
  } catch {
    return false;
  }
}
