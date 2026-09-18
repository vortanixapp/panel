"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import type { Branding } from "@/lib/api";
import { setFavicon } from "@/lib/appearance";
import {
  BRAND_LOGO_URL,
  BRAND_NAME,
  loadRuntimeBranding,
  normalizeBrandName,
  reloadRuntimeBranding,
} from "@/lib/brand";

type WhmcsLinks = {
  orderUrl: string;
  clientAreaUrl: string;
  ordersOnly: boolean;
};

type BrandState = {
  name: string;
  title: string;
  logoUrl: string;
  logoDarkUrl: string;
  iconUrl: string;
  templateBlocks: Record<string, boolean>;
  userMenuVariant: "default" | "screenshot";
  hero: Record<string, string>;
  links: Record<string, string>;
  whmcs: WhmcsLinks | null;
  legal: NonNullable<Branding["legal"]> | null;
  ready: boolean;
  refresh: () => Promise<void>;
};

const EMPTY: Record<string, never> = {};

const BrandContext = createContext<BrandState>({
  name: BRAND_NAME,
  title: BRAND_NAME,
  logoUrl: BRAND_LOGO_URL,
  logoDarkUrl: "",
  iconUrl: "",
  templateBlocks: EMPTY,
  userMenuVariant: "default",
  hero: EMPTY,
  links: EMPTY,
  whmcs: null,
  legal: null,
  ready: false,
  refresh: async () => undefined,
});

export function BrandProvider({ children }: { children: ReactNode }) {
  const [branding, setBranding] = useState<Branding | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let cancelled = false;
    void loadRuntimeBranding().then((loaded) => {
      if (cancelled) return;
      setBranding(loaded);
      setReady(true);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (branding?.icon_url) setFavicon(branding.icon_url);
  }, [branding?.icon_url]);

  const refresh = useCallback(async () => {
    const fresh = await reloadRuntimeBranding();
    if (fresh) setBranding(fresh);
  }, []);

  const value = useMemo<BrandState>(
    () => ({
      name: branding ? normalizeBrandName(branding.brand_name) : BRAND_NAME,
      title: branding?.panel_name?.trim() || (branding ? normalizeBrandName(branding.brand_name) : BRAND_NAME),
      logoUrl: branding?.logo_url || BRAND_LOGO_URL,
      logoDarkUrl: branding?.logo_dark_url || "",
      iconUrl: branding?.icon_url || "",
      templateBlocks: branding?.blocks ?? EMPTY,
      userMenuVariant:
        branding?.user_menu_variant === "screenshot" ? "screenshot" : "default",
      hero: branding?.appearance?.hero ?? EMPTY,
      links: branding?.appearance?.links ?? EMPTY,
      whmcs: branding?.whmcs?.order_url
        ? {
            orderUrl: branding.whmcs.order_url,
            clientAreaUrl: branding.whmcs.client_area_url || "",
            ordersOnly: Boolean(branding.whmcs.orders_only),
          }
        : null,
      legal: branding?.legal ?? null,
      ready,
      refresh,
    }),
    [branding, ready, refresh]
  );

  return <BrandContext.Provider value={value}>{children}</BrandContext.Provider>;
}

export function useBrand() {
  return useContext(BrandContext);
}
