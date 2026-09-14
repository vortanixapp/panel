import { API_URL, fetchBranding, type Branding } from "@/lib/api";
import {
  applyAppearance,
  injectedBranding,
  rememberBranding,
} from "@/lib/appearance";
import { runtimeConfig } from "@/lib/runtime-config";

export const DEFAULT_BRAND = "VORTANIX";
export const DEFAULT_BRAND_LOGO_URL = "/branding/vortanix-logo-horizontal.png";
export const DEFAULT_BRAND_MARK_URL = "/branding/vortanix-mark-square.png";

export function normalizeBrandName(raw: string | undefined): string {
  return (raw?.trim() || DEFAULT_BRAND).toUpperCase();
}

export const BRAND_NAME = normalizeBrandName(runtimeConfig().brand_name);

export const BRAND_LOGO_URL = runtimeConfig().brand_logo_url;

export async function loadRuntimeBranding(): Promise<Branding | null> {
  if (typeof window === "undefined") return null;
  const injected = injectedBranding();
  if (injected) return injected;
  try {
    const fresh = await fetchBranding();
    rememberBranding(fresh);
    applyAppearance(fresh);
    return fresh;
  } catch {
    return null;
  }
}

export async function reloadRuntimeBranding(): Promise<Branding | null> {
  try {
    const fresh = await fetchBranding();
    rememberBranding(fresh);
    applyAppearance(fresh);
    return fresh;
  } catch {
    return null;
  }
}

export { API_URL };
