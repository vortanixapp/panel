import { serverApiURL, serverRuntimeConfig } from "@/lib/runtime-config";

export async function getTenantBootstrapped(): Promise<boolean> {
  const { tenant_slug: TENANT_SLUG } = serverRuntimeConfig();
  const API_URL = serverApiURL();
  try {
    const res = await fetch(
      `${API_URL}/v1/tenants/status?slug=${encodeURIComponent(TENANT_SLUG)}`,
      { cache: "no-store" }
    );
    if (!res.ok) return false;
    const data = (await res.json()) as { bootstrapped?: boolean };
    return !!data.bootstrapped;
  } catch {
    return false;
  }
}
