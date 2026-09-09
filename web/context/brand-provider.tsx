"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import {
  BRAND_LOGO_URL,
  BRAND_NAME,
  getRuntimeBrandName,
  getRuntimeLogoUrl,
  getRuntimePrimaryColor,
  getRuntimeTemplateBlocks,
  getRuntimeTemplateColors,
  getRuntimeUserMenuVariant,
  loadRuntimeBranding,
} from "@/lib/brand";

type BrandState = {
  name: string;
  logoUrl: string;
  primaryColor: string;
  templateColors: Record<string, string>;
  templateBlocks: Record<string, boolean>;
  userMenuVariant: "default" | "screenshot";
  ready: boolean;
};

const BrandContext = createContext<BrandState>({
  name: BRAND_NAME,
  logoUrl: BRAND_LOGO_URL,
  primaryColor: "#6366f1",
  templateColors: {},
  templateBlocks: {},
  userMenuVariant: "default",
  ready: false,
});

function applyPrimaryColor(color: string) {
  const root = document.documentElement;
  root.style.setProperty("--brand-primary", color);
  root.style.setProperty("--primary", color);
  root.style.setProperty("--primary-foreground", readableTextOn(color));
  root.style.setProperty("--ring", color);
}

function readableTextOn(color: string): string {
  const rgb = hexToRgb(color);
  if (!rgb) return "#ffffff";
  const [r, g, b] = rgb.map((v) => {
    const c = v / 255;
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
  });
  const luminance = 0.2126 * r + 0.7152 * g + 0.0722 * b;
  return luminance > 0.45 ? "#0a0b0d" : "#ffffff";
}

function hexToRgb(color: string): [number, number, number] | null {
  const m = /^#?([\da-f]{3}|[\da-f]{6})$/i.exec(color.trim());
  if (!m) return null;
  let hex = m[1];
  if (hex.length === 3) {
    hex = hex
      .split("")
      .map((c) => c + c)
      .join("");
  }
  return [
    parseInt(hex.slice(0, 2), 16),
    parseInt(hex.slice(2, 4), 16),
    parseInt(hex.slice(4, 6), 16),
  ];
}

export function BrandProvider({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let cancelled = false;
    void loadRuntimeBranding().then(() => {
      if (cancelled) return;
      applyPrimaryColor(getRuntimePrimaryColor());
      setReady(true);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const value = useMemo<BrandState>(
    () => ({
      name: ready ? getRuntimeBrandName() : BRAND_NAME,
      logoUrl: ready ? getRuntimeLogoUrl() : BRAND_LOGO_URL,
      primaryColor: ready ? getRuntimePrimaryColor() : "#6366f1",
      templateColors: ready ? getRuntimeTemplateColors() : {},
      templateBlocks: ready ? getRuntimeTemplateBlocks() : {},
      userMenuVariant: ready ? getRuntimeUserMenuVariant() : "default",
      ready,
    }),
    [ready]
  );

  return <BrandContext.Provider value={value}>{children}</BrandContext.Provider>;
}

export function useBrand() {
  return useContext(BrandContext);
}
