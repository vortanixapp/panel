"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { usePathname } from "next/navigation";
import { useSite } from "@/context/site-provider";
import { useLocale } from "@/context/locale-provider";
import { hasSession } from "@/lib/api";
import { currentAccount } from "@/lib/accounts";
import { pageKeyOf } from "@/lib/site/pages";
import { localized } from "@/lib/site/text";
import type { LText, SitePage, Viewer } from "@/lib/site/types";

export type ViewerState = Viewer & { ready: boolean };

const GUEST: ViewerState = { loggedIn: false, role: "", ready: false };

export function useViewer(): ViewerState {
  const { viewer: forced } = useSite();
  const [viewer, setViewer] = useState<ViewerState>(GUEST);

  useEffect(() => {
    const account = currentAccount();
    setViewer({ loggedIn: hasSession(), role: account?.role ?? "", ready: true });
  }, []);

  return useMemo(() => (forced ? { ...forced, ready: true } : viewer), [forced, viewer]);
}

export function useSiteText(): (value: LText | undefined) => string {
  const { locale, defaultLocale } = useLocale();
  return useCallback(
    (value: LText | undefined) => localized(value, locale, defaultLocale),
    [locale, defaultLocale]
  );
}

export function usePageKey(): string {
  return pageKeyOf(usePathname() ?? "/");
}

export function useSitePage(key?: string): SitePage | undefined {
  const { document } = useSite();
  const current = usePageKey();
  return document.pages?.[key ?? current];
}
