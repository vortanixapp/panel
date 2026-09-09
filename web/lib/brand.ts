import { API_URL, fetchBranding, type Branding } from "@/lib/api";
import { runtimeConfig } from "@/lib/runtime-config";

export const DEFAULT_BRAND = "VORTANIX";
export const DEFAULT_BRAND_LOGO_URL = "/branding/vortanix-logo-horizontal.png";
export const DEFAULT_BRAND_MARK_URL = "/branding/vortanix-mark-square.png";

function normalizeBrandName(raw: string | undefined): string {
  return (raw?.trim() || DEFAULT_BRAND).toUpperCase();
}

export const BRAND_NAME = normalizeBrandName(runtimeConfig().brand_name);

export const BRAND_LOGO_URL = runtimeConfig().brand_logo_url;

let cachedBranding: Branding | null = null;

export async function loadRuntimeBranding() {
  if (typeof window === "undefined") return null;
  try {
    cachedBranding = await fetchBranding();
    return cachedBranding;
  } catch {
    return null;
  }
}

export function getRuntimeBrandName(): string {
  return normalizeBrandName(cachedBranding?.brand_name ?? BRAND_NAME);
}

export function getRuntimeLogoUrl(): string {
  return cachedBranding?.logo_url || BRAND_LOGO_URL;
}

export function getRuntimePrimaryColor(): string {
  return cachedBranding?.primary_color || "#6366f1";
}

export function getRuntimeTemplateColors(): Record<string, string> {
  return cachedBranding?.colors ?? {};
}

export function getRuntimeTemplateBlocks(): Record<string, boolean> {
  return cachedBranding?.blocks ?? {};
}

export function getRuntimeUserMenuVariant(): "default" | "screenshot" {
  return cachedBranding?.user_menu_variant === "screenshot" ? "screenshot" : "default";
}

export { API_URL };
