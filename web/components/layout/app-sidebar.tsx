"use client";

import { useEffect, useState } from "react";
import { useLayout } from "@/context/layout-provider";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar";
import { getNavGroupsForVariant } from "@/lib/nav";
import { useT, useTranslations } from "@/hooks/use-translations";
import type { PanelVariant } from "@/lib/panel-paths";
import { AppTitle } from "./app-title";
import { NavGroup } from "./nav-group";
import { NavUser } from "./nav-user";
import { SectionSwitcher } from "./section-switcher";

type AppSidebarProps = {
  email: string;
  role: string;
  variant: PanelVariant;
};

export function AppSidebar({ email, role, variant }: AppSidebarProps) {
  const { collapsible, variant: sidebarVariant } = useLayout();
  const t = useT();
  const [navGroups, setNavGroups] = useState(() =>
    getNavGroupsForVariant(variant, t)
  );

  const overrides = useTranslations();

  useEffect(() => {
    setNavGroups(getNavGroupsForVariant(variant, t));
    const onDisplayPrefs = () =>
      setNavGroups(getNavGroupsForVariant(variant, t));
    window.addEventListener("vortanix-display-prefs", onDisplayPrefs);
    return () =>
      window.removeEventListener("vortanix-display-prefs", onDisplayPrefs);
  }, [variant, overrides, t]);

  return (
    <Sidebar collapsible={collapsible} variant={sidebarVariant}>
      <SidebarHeader>
        <AppTitle variant={variant} />
      </SidebarHeader>
      <SidebarContent>
        {navGroups.map((group) => (
          <NavGroup key={group.title} {...group} />
        ))}
      </SidebarContent>
      <SidebarFooter className="gap-2">
        <SectionSwitcher role={role} />
        <NavUser email={email} role={role} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
