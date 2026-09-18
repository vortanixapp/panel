import { parseAccountCookie } from "@/lib/accounts";
import { serverApiURL } from "@/lib/runtime-config";
import { cachedLoader } from "@/lib/server-cache";
import { parseSitePayload } from "@/lib/site/parse";
import type { SiteDocument, Viewer } from "@/lib/site/types";

const CACHE_TTL_MS = 3_000;
const site = cachedLoader<SiteDocument>(CACHE_TTL_MS, 1);

export function loadServerSite(fresh = false): Promise<SiteDocument> {
  return site(
    "",
    async (previous) => {
      try {
        const res = await fetch(`${serverApiURL()}/v1/site`, {
          cache: "no-store",
          signal: AbortSignal.timeout(1500),
        });
        return res.ok ? parseSitePayload(await res.json()) : previous ?? {};
      } catch {
        return previous ?? {};
      }
    },
    fresh
  );
}

export function viewerFromCookie(raw: string | undefined): Viewer {
  const account = parseAccountCookie(raw ?? "");
  return { loggedIn: account !== null, role: account?.role ?? "" };
}
