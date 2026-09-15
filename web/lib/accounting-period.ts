import { localeTag } from "@/lib/i18n";

export const PERIOD_PRESETS = [
  "month",
  "prev_month",
  "quarter",
  "prev_quarter",
  "half_year",
  "nine_months",
  "year",
  "prev_year",
] as const;

export type PeriodPreset = (typeof PERIOD_PRESETS)[number];

export function isoDate(d: Date): string {
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${month}-${day}`;
}

export function presetRange(preset: PeriodPreset, now = new Date()): { from: string; to: string } {
  const y = now.getFullYear();
  const m = now.getMonth();
  const q = Math.floor(m / 3) * 3;
  const range = (start: Date, end: Date) => ({ from: isoDate(start), to: isoDate(end) });
  switch (preset) {
    case "month":
      return range(new Date(y, m, 1), new Date(y, m + 1, 0));
    case "prev_month":
      return range(new Date(y, m - 1, 1), new Date(y, m, 0));
    case "quarter":
      return range(new Date(y, q, 1), new Date(y, q + 3, 0));
    case "prev_quarter":
      return range(new Date(y, q - 3, 1), new Date(y, q, 0));
    case "half_year":
      return range(new Date(y, 0, 1), new Date(y, 6, 0));
    case "nine_months":
      return range(new Date(y, 0, 1), new Date(y, 9, 0));
    case "year":
      return range(new Date(y, 0, 1), new Date(y, 12, 0));
    case "prev_year":
      return range(new Date(y - 1, 0, 1), new Date(y - 1, 12, 0));
  }
}

export function yearToDate(now = new Date()): { from: string; to: string } {
  return { from: isoDate(new Date(now.getFullYear(), 0, 1)), to: isoDate(now) };
}

export function money(value: number): string {
  return value.toLocaleString(localeTag(), {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

export function monthTitle(period: string): string {
  const [year, month] = period.split("-").map(Number);
  if (!year || !month) return period;
  const label = new Date(year, month - 1, 1).toLocaleDateString(localeTag(), {
    month: "long",
    year: "numeric",
  });
  return label.charAt(0).toUpperCase() + label.slice(1);
}
