import { serverRuntimeConfig } from "@/lib/runtime-config";

let bootstrapped = false;

let checkedAt = 0;
const NEGATIVE_TTL_MS = 5_000;

let assumeUntil = 0;
const FAILURE_TTL_MS = 30_000;

export async function checkBootstrapped(): Promise<boolean> {
  if (bootstrapped) return true;
  if (Date.now() < assumeUntil) return true;
  if (Date.now() - checkedAt < NEGATIVE_TTL_MS) return false;

  const { api_url, tenant_slug } = serverRuntimeConfig();
  try {
    const res = await fetch(
      `${api_url}/v1/tenants/status?slug=${encodeURIComponent(tenant_slug)}`,
      { cache: "no-store", signal: AbortSignal.timeout(3_000) },
    );
    if (!res.ok) {
      assumeUntil = Date.now() + FAILURE_TTL_MS;
      return true;
    }
    const data = (await res.json()) as { bootstrapped?: boolean };
    checkedAt = Date.now();
    if (data.bootstrapped) bootstrapped = true;
    return Boolean(data.bootstrapped);
  } catch {
    assumeUntil = Date.now() + FAILURE_TTL_MS;
    return true;
  }
}
