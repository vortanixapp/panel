import { localeTag } from "@/lib/i18n";

export function amountText(amount: number, digits = 0): string {
  const value = Number.isFinite(amount) ? amount : 0;
  return new Intl.NumberFormat(localeTag(), {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(value);
}

export function currencySymbol(currency: string): string {
  const code = (currency || "").toUpperCase();
  if (code === "RUB" || code === "") return "₽";
  if (code === "USD") return "$";
  if (code === "EUR") return "€";
  return code;
}

export function money(amount: number, currency: string, digits = 0): string {
  return `${amountText(amount, digits)} ${currencySymbol(currency)}`;
}

export function moneyPrecise(amount: number, currency: string): string {
  return money(amount, currency, Number.isInteger(Math.round(amount * 100) / 100) ? 0 : 2);
}

export function daysUntil(iso?: string | null): number | null {
  if (!iso) return null;
  const ts = new Date(iso).getTime();
  if (Number.isNaN(ts)) return null;
  return Math.ceil((ts - Date.now()) / 86_400_000);
}

export function shortDate(iso?: string | null): string {
  if (!iso) return "—";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleDateString(localeTag(), { day: "numeric", month: "long" });
}

export function relativeTime(iso: string, now = Date.now()): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "—";
  const rtf = new Intl.RelativeTimeFormat(localeTag(), { numeric: "auto" });
  const seconds = Math.round((date.getTime() - now) / 1000);
  const abs = Math.abs(seconds);
  if (abs < 45) return rtf.format(0, "second");
  if (abs < 3600) return rtf.format(Math.round(seconds / 60), "minute");
  if (abs < 86400) return rtf.format(Math.round(seconds / 3600), "hour");
  if (abs < 86400 * 7) return rtf.format(Math.round(seconds / 86400), "day");
  return date.toLocaleDateString(localeTag(), { day: "numeric", month: "short" });
}

export type DayPart = "night" | "morning" | "day" | "evening";

export function dayPart(date = new Date()): DayPart {
  const hour = date.getHours();
  if (hour < 5) return "night";
  if (hour < 12) return "morning";
  if (hour < 18) return "day";
  return "evening";
}

export function serverAddress(ip?: string | null, port?: number | null): string {
  if (!ip) return "";
  return port ? `${ip}:${port}` : ip;
}
