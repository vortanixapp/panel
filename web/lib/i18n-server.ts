import { cache } from "react";
import { cookies, headers } from "next/headers";
import {
  acceptedLocales,
  fallbackI18n,
  isLocaleCode,
  LOCALE_COOKIE_NAME,
  normalizeLocaleCode,
  parseI18nPayload,
  pickLocale,
  type I18nPayload,
} from "@/lib/i18n";
import { serverApiURL } from "@/lib/runtime-config";
import { cachedLoader } from "@/lib/server-cache";

const CACHE_TTL_MS = 3_000;
const catalogs = cachedLoader<I18nPayload>(CACHE_TTL_MS, 64);

export function loadServerI18n(requested: string | undefined): Promise<I18nPayload> {
  const code = normalizeLocaleCode(requested);
  const key = isLocaleCode(code) ? code : "";
  return catalogs(key, async (previous) => {
    try {
      const res = await fetch(
        `${serverApiURL()}/v1/i18n?locale=${encodeURIComponent(key)}`,
        { cache: "no-store", signal: AbortSignal.timeout(1500) }
      );
      return res.ok ? parseI18nPayload(await res.json(), key) : previous ?? fallbackI18n(key);
    } catch {
      return previous ?? fallbackI18n(key);
    }
  });
}

export const loadRequestI18n = cache(
  async (): Promise<{ i18n: I18nPayload; hasCookie: boolean }> => {
    const [store, requestHeaders] = await Promise.all([cookies(), headers()]);
    const cookieLocale = store.get(LOCALE_COOKIE_NAME)?.value;
    const payload = await loadServerI18n(cookieLocale);
    if (cookieLocale) return { i18n: payload, hasCookie: true };
    const preferred = pickLocale(
      acceptedLocales(requestHeaders.get("accept-language")),
      payload.languages,
      payload.locale
    );
    return {
      i18n: preferred === payload.locale ? payload : await loadServerI18n(preferred),
      hasCookie: false,
    };
  }
);
