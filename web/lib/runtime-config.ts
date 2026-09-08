export const RUNTIME_CONFIG_GLOBAL = "__VORTANIX_CONFIG__";

export type RuntimeConfig = {
  api_url: string;
  console_url: string;
  tenant_slug: string;
  brand_name: string;
  brand_logo_url: string;
  version: string;
};

const DEFAULT_API_URL = "http://localhost:8080";
const DEFAULT_CONSOLE_URL = "ws://localhost:8083";
const DEFAULT_TENANT_SLUG = "dev";

declare global {
  interface Window {
    __VORTANIX_CONFIG__?: Partial<RuntimeConfig>;
  }
}

function envConfig(): RuntimeConfig {
  return {
    api_url: process.env.NEXT_PUBLIC_API_URL?.trim() || DEFAULT_API_URL,
    console_url: process.env.NEXT_PUBLIC_CONSOLE_URL?.trim() || DEFAULT_CONSOLE_URL,
    tenant_slug: process.env.NEXT_PUBLIC_TENANT_SLUG?.trim() || DEFAULT_TENANT_SLUG,
    brand_name: process.env.NEXT_PUBLIC_BRAND_NAME?.trim() || "",
    brand_logo_url: process.env.NEXT_PUBLIC_BRAND_LOGO_URL?.trim() || "",
    version: process.env.VORTANIX_VERSION?.trim() || "dev",
  };
}

export function serverRuntimeConfig(): RuntimeConfig {
  return envConfig();
}

export function runtimeConfig(): RuntimeConfig {
  const env = envConfig();
  if (typeof window === "undefined") return env;

  const injected = window[RUNTIME_CONFIG_GLOBAL];
  if (!injected) return env;

  return {
    api_url: injected.api_url?.trim() || env.api_url,
    console_url: injected.console_url?.trim() || env.console_url,
    tenant_slug: injected.tenant_slug?.trim() || env.tenant_slug,
    brand_name: injected.brand_name?.trim() || env.brand_name,
    brand_logo_url: injected.brand_logo_url?.trim() || env.brand_logo_url,
    version: injected.version?.trim() || env.version,
  };
}
