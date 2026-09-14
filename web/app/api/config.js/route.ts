import {
  RUNTIME_CONFIG_GLOBAL,
  serverApiURL,
  serverRuntimeConfig,
} from "@/lib/runtime-config";
import {
  appearanceBootScript,
  BRANDING_GLOBAL,
  type BrandingPayload,
} from "@/lib/appearance";

export const dynamic = "force-dynamic";

const CACHE_TTL_MS = 15_000;
const cache = new Map<string, { at: number; data: BrandingPayload | null }>();

async function loadBranding(request: Request): Promise<BrandingPayload | null> {
  const host =
    request.headers.get("x-forwarded-host") || request.headers.get("host") || "";
  const proto =
    request.headers.get("x-forwarded-proto") ||
    new URL(request.url).protocol.replace(":", "");
  const key = `${proto}://${host}`;
  const hit = cache.get(key);
  if (hit && Date.now() - hit.at < CACHE_TTL_MS) return hit.data;

  let data = hit?.data ?? null;
  try {
    const res = await fetch(`${serverApiURL()}/v1/branding`, {
      cache: "no-store",
      headers: host ? { "X-Forwarded-Host": host, "X-Forwarded-Proto": proto } : {},
      signal: AbortSignal.timeout(1500),
    });
    if (res.ok) data = (await res.json()) as BrandingPayload;
  } catch {
    data = hit?.data ?? null;
  }
  cache.set(key, { at: Date.now(), data });
  return data;
}

export async function GET(request: Request) {
  const config = serverRuntimeConfig();
  const branding = await loadBranding(request);
  let body = `window.${RUNTIME_CONFIG_GLOBAL}=${JSON.stringify(config)};`;
  if (branding) {
    body += `window.${BRANDING_GLOBAL}=${JSON.stringify(branding)};`;
    body += appearanceBootScript(branding);
  }
  return new Response(body, {
    headers: {
      "Content-Type": "application/javascript; charset=utf-8",
      "Cache-Control": "no-store",
    },
  });
}
