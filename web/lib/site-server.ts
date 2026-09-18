import { serverApiURL } from "@/lib/runtime-config";
import { parseSitePayload } from "@/lib/site/parse";
import type { SiteDocument } from "@/lib/site/types";

const CACHE_TTL_MS = 15_000;
let cache: { at: number; data: SiteDocument } | null = null;

export async function loadServerSite(fresh = false): Promise<SiteDocument> {
  if (!fresh && cache && Date.now() - cache.at < CACHE_TTL_MS) return cache.data;
  let data: SiteDocument = cache?.data ?? {};
  try {
    const res = await fetch(`${serverApiURL()}/v1/site`, {
      cache: "no-store",
      signal: AbortSignal.timeout(1500),
    });
    if (res.ok) data = parseSitePayload(await res.json());
  } catch {
    data = cache?.data ?? data;
  }
  cache = { at: Date.now(), data };
  return data;
}
