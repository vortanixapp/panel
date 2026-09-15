import {
  fallbackI18n,
  isLocaleCode,
  normalizeLocaleCode,
  parseI18nPayload,
  type I18nPayload,
} from "@/lib/i18n";
import { serverApiURL } from "@/lib/runtime-config";

const CACHE_TTL_MS = 15_000;
const CACHE_LIMIT = 64;
const cache = new Map<string, { at: number; data: I18nPayload }>();

export async function loadServerI18n(
  requested: string | undefined
): Promise<I18nPayload> {
  const code = normalizeLocaleCode(requested);
  const key = isLocaleCode(code) ? code : "";
  const hit = cache.get(key);
  if (hit && Date.now() - hit.at < CACHE_TTL_MS) return hit.data;

  let data = hit?.data ?? fallbackI18n(key);
  try {
    const res = await fetch(
      `${serverApiURL()}/v1/i18n?locale=${encodeURIComponent(key)}`,
      { cache: "no-store", signal: AbortSignal.timeout(1500) }
    );
    if (res.ok) data = parseI18nPayload(await res.json(), key);
  } catch {
    data = hit?.data ?? data;
  }
  if (cache.size >= CACHE_LIMIT) cache.clear();
  cache.set(key, { at: Date.now(), data });
  return data;
}
