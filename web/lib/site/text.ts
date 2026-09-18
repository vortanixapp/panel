import type { LText } from "@/lib/site/types";

export function pickText(value: LText | undefined, locale: string): string {
  const text = value?.[locale];
  return text && text.trim() ? text : "";
}

export function anyText(value: LText | undefined): string {
  for (const text of Object.values(value ?? {})) {
    if (text && text.trim()) return text;
  }
  return "";
}

export function localized(value: LText | undefined, locale: string, defaultLocale: string): string {
  return pickText(value, locale) || pickText(value, defaultLocale) || anyText(value);
}

export function withText(value: LText | undefined, locale: string, text: string): LText | undefined {
  const next: LText = { ...(value ?? {}) };
  if (text.trim()) {
    next[locale] = text;
  } else {
    delete next[locale];
  }
  return Object.keys(next).length > 0 ? next : undefined;
}
