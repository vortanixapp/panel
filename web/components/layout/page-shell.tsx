"use client";

import { ConfigDrawer } from "@/components/config-drawer";
import { Header } from "@/components/layout/header";
import { Main } from "@/components/layout/main";
import { ProfileDropdown } from "@/components/profile-dropdown";
import { Search } from "@/components/search";
import { NotificationsBell } from "@/components/notifications-bell";
import { LanguageSwitch } from "@/components/language-switch";
import { ThemeSwitch } from "@/components/theme-switch";
import type { PanelVariant } from "@/lib/panel-paths";

export function PageShell({
  children,
  fixed = true,
  fluid = true,
  variant = "user",
  title,
}: {
  children: React.ReactNode;
  fixed?: boolean;
  fluid?: boolean;
  variant?: PanelVariant;
  title?: string;
}) {
  void variant;

  return (
    <>
      <Header fixed={fixed}>
        <Search className="me-auto" />
        <NotificationsBell />
        <LanguageSwitch />
        <ThemeSwitch />
        <ConfigDrawer />
        <ProfileDropdown />
      </Header>
      <Main fixed={fixed} fluid={fluid}>
        {title ? <h1 className="mb-6 text-2xl font-semibold tracking-tight">{title}</h1> : null}
        {children}
      </Main>
    </>
  );
}
