"use client";

import { useLayout } from "@/context/layout-provider";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar";
import { useSidebarGroups } from "@/hooks/use-site-menu";
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
  const navGroups = useSidebarGroups(variant);

  return (
    <Sidebar collapsible={collapsible} variant={sidebarVariant}>
      <SidebarHeader>
        <AppTitle variant={variant} />
      </SidebarHeader>
      <SidebarContent>
        {navGroups.map((group) => (
          <NavGroup key={group.id ?? group.title} {...group} />
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
