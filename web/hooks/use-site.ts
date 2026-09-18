"use client";

import { useCallback, useEffect, useState } from "react";
import { usePathname } from "next/navigation";
import { useSite } from "@/context/site-provider";
import { useLocale } from "@/context/locale-provider";
import { currentAccount } from "@/lib/accounts";
import { pageKeyOf } from "@/lib/site/pages";
import { localized } from "@/lib/site/text";
import type { LText, SitePage, Viewer } from "@/lib/site/types";

export function useViewer(): Viewer {
  const { viewer: forced, session } = useSite();
  const [viewer, setViewer] = useState<Viewer>(session);

  useEffect(() => {
    const account = currentAccount();
    const loggedIn = account !== null;
    const role = account?.role ?? "";
    setViewer((current) =>
      current.loggedIn === loggedIn && current.role === role ? current : { loggedIn, role }
    );
  }, []);

  return forced ?? viewer;
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
