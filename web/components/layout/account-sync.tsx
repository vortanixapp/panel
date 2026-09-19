"use client";

import { useEffect, useRef } from "react";
import { fonts } from "@/config/fonts";
import { useFont } from "@/context/font-provider";
import { useLayout, type Collapsible } from "@/context/layout-provider";
import { useTheme } from "@/context/theme-provider";
import { useAccountQuery, useSetAccount } from "@/hooks/use-account";
import { updateAccount, type AccountPreferences } from "@/lib/api";
import { restoreDisplayTimeZone, setDisplayTimeZone } from "@/lib/timezone";

if (typeof window !== "undefined") {
  restoreDisplayTimeZone();
}

const THEMES = ["light", "dark", "system"] as const;
const VARIANTS = ["inset", "sidebar", "floating"] as const;
const COLLAPSIBLES = ["offcanvas", "icon", "none"] as const;

type Theme = (typeof THEMES)[number];
type Variant = (typeof VARIANTS)[number];
type Font = (typeof fonts)[number];

function pick<T extends string>(list: readonly T[], value: string | undefined, fallback: T): T {
  return value && (list as readonly string[]).includes(value) ? (value as T) : fallback;
}

export function AccountSync() {
  const { data } = useAccountQuery();
  const setAccount = useSetAccount();
  const { theme, setTheme, locked } = useTheme();
  const { font, setFont } = useFont();
  const { variant, setVariant, collapsible, setCollapsible } = useLayout();
  const zone = data?.user.timezone;
  const prefs = data?.user.preferences;
  const applied = useRef(false);
  const saved = useRef("");

  useEffect(() => {
    if (zone !== undefined) setDisplayTimeZone(zone);
  }, [zone]);

  useEffect(() => {
    if (!prefs || applied.current) return;
    applied.current = true;
    const next = {
      theme: locked ? theme : pick<Theme>(THEMES, prefs.theme, theme),
      font: pick<Font>(fonts, prefs.font, font),
      layout: pick<Variant>(VARIANTS, prefs.layout, variant),
      collapsible: pick<Collapsible>(COLLAPSIBLES, prefs.collapsible, collapsible),
    };
    saved.current = JSON.stringify(next);
    if (next.theme !== theme) setTheme(next.theme);
    if (next.font !== font) setFont(next.font);
    if (next.layout !== variant) setVariant(next.layout);
    if (next.collapsible !== collapsible) setCollapsible(next.collapsible);
  }, [prefs, locked, theme, font, variant, collapsible, setTheme, setFont, setVariant, setCollapsible]);

  useEffect(() => {
    if (!applied.current) return;
    const key = JSON.stringify({ theme, font, layout: variant, collapsible });
    if (key === saved.current) return;
    const timer = window.setTimeout(() => {
      saved.current = key;
      const patch: AccountPreferences = { font, layout: variant, collapsible };
      if (!locked) patch.theme = theme;
      updateAccount({ preferences: patch })
        .then((res) => setAccount(res.user))
        .catch(() => undefined);
    }, 700);
    return () => window.clearTimeout(timer);
  }, [theme, font, variant, collapsible, locked, setAccount]);

  return null;
}
