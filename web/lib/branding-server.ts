import { serverApiURL, serverRuntimeConfig } from "@/lib/runtime-config";

const CACHE_TTL_MS = 15_000;
let cache: { at: number; name: string } | null = null;

function fallbackName(): string {
  return serverRuntimeConfig().brand_name || "Vortanix";
}

export async function loadServerBrandName(): Promise<string> {
  if (cache && Date.now() - cache.at < CACHE_TTL_MS) return cache.name;
  let name = cache?.name ?? fallbackName();
  try {
    const res = await fetch(`${serverApiURL()}/v1/branding`, {
      cache: "no-store",
      signal: AbortSignal.timeout(1500),
    });
    if (res.ok) {
      const data = (await res.json()) as { panel_name?: unknown; brand_name?: unknown };
      const fresh =
        typeof data.panel_name === "string" && data.panel_name.trim()
          ? data.panel_name
          : typeof data.brand_name === "string"
            ? data.brand_name
            : "";
      if (fresh.trim()) name = fresh.trim();
    }
  } catch {
    name = cache?.name ?? name;
  }
  cache = { at: Date.now(), name };
  return name;
}
