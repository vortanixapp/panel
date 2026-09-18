"use client";

import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type { NavGroup } from "@/components/layout/types";
import { useLocale } from "@/context/locale-provider";
import { useSite } from "@/context/site-provider";
import { useMe } from "@/hooks/use-queries";
import { useViewer } from "@/hooks/use-site";
import { useT } from "@/hooks/use-translations";
import { hasSession } from "@/lib/api";
import { sidebarGroups } from "@/lib/nav";
import type { PanelVariant } from "@/lib/panel-paths";
import { queryKeys } from "@/lib/query-keys";
import { fetchAdminSiteMenu } from "@/lib/site/api";
import { landingAnchors, resolveMenu, type ResolvedItem } from "@/lib/site/menu";
import type { MenuName, SiteDocument, SiteMenu, Viewer } from "@/lib/site/types";

function withMenu(doc: SiteDocument, name: MenuName, menu: SiteMenu | null | undefined): SiteDocument {
  if (!menu) return doc;
  return { ...doc, menus: { ...(doc.menus ?? {}), [name]: menu } };
}

export function useAdminSiteMenu(variant: PanelVariant) {
  const { inEditor } = useSite();
  const enabled = variant === "admin" && !inEditor && hasSession();
  const query = useQuery({
    queryKey: queryKeys.adminSiteMenu,
    queryFn: fetchAdminSiteMenu,
    enabled,
    staleTime: 60_000,
  });
  return { enabled, menu: query.data?.menu, pending: enabled && query.isPending };
}

export function useSidebarGroups(variant: PanelVariant): NavGroup[] {
  const t = useT();
  const { locale, defaultLocale } = useLocale();
  const { document, viewer: forced } = useSite();
  const { data: me } = useMe();
  const [prefsVersion, setPrefsVersion] = useState(0);
  const adminMenu = useAdminSiteMenu(variant);

  useEffect(() => {
    const onPrefs = () => setPrefsVersion((v) => v + 1);
    window.addEventListener("vortanix-display-prefs", onPrefs);
    return () => window.removeEventListener("vortanix-display-prefs", onPrefs);
  }, []);

  return useMemo(() => {
    void prefsVersion;
    const doc = adminMenu.enabled ? withMenu(document, "admin_sidebar", adminMenu.menu) : document;
    const viewer: Viewer = forced ?? { loggedIn: true, role: me?.role ?? "" };
    return sidebarGroups(variant, doc, { t, locale, defaultLocale, viewer });
  }, [variant, adminMenu.enabled, adminMenu.menu, document, forced, me?.role, t, locale, defaultLocale, prefsVersion]);
}

export function useSiteMenu(name: "site_header" | "site_footer"): ResolvedItem[] {
  const t = useT();
  const { locale, defaultLocale } = useLocale();
  const { document } = useSite();
  const viewer = useViewer();
  return useMemo(
    () =>
      resolveMenu(name, document, {
        t,
        locale,
        defaultLocale,
        viewer,
        anchors: landingAnchors(document),
      }),
    [name, document, t, locale, defaultLocale, viewer]
  );
}
