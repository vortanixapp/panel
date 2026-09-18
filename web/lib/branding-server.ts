import { cache } from "react";
import { headers } from "next/headers";
import type { BrandingPayload } from "@/lib/appearance";
import { serverApiURL, serverRuntimeConfig } from "@/lib/runtime-config";
import { cachedLoader } from "@/lib/server-cache";

const CACHE_TTL_MS = 3_000;
const brandings = cachedLoader<BrandingPayload | null>(CACHE_TTL_MS, 32);

export const loadServerBranding = cache(async (): Promise<BrandingPayload | null> => {
  const requestHeaders = await headers();
  const host = requestHeaders.get("x-forwarded-host") || requestHeaders.get("host") || "";
  const proto = requestHeaders.get("x-forwarded-proto") || "http";
  return brandings(`${proto}://${host}`, async (previous) => {
    try {
      const res = await fetch(`${serverApiURL()}/v1/branding`, {
        cache: "no-store",
        headers: host ? { "X-Forwarded-Host": host, "X-Forwarded-Proto": proto } : {},
        signal: AbortSignal.timeout(1500),
      });
      return res.ok ? ((await res.json()) as BrandingPayload) : previous ?? null;
    } catch {
      return previous ?? null;
    }
  });
});

export function brandTitle(branding: BrandingPayload | null): string {
  return (
    branding?.panel_name?.trim() ||
    branding?.brand_name?.trim() ||
    serverRuntimeConfig().brand_name ||
    "Vortanix"
  );
}
