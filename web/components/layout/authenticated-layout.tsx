"use client";

import { getCookie } from "@/lib/cookies";
import { cn } from "@/lib/utils";
import { LayoutProvider } from "@/context/layout-provider";
import { SearchProvider } from "@/context/search-provider";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/layout/app-sidebar";
import { SkipToMain } from "@/components/skip-to-main";
import type { PanelVariant } from "@/lib/panel-paths";

type AuthenticatedLayoutProps = {
  children: React.ReactNode;
  email: string;
  role: string;
  variant?: PanelVariant;
};

export function AuthenticatedLayout({
  children,
  email,
  role,
  variant = "user",
}: AuthenticatedLayoutProps) {
  const defaultOpen = getCookie("sidebar_state") !== "false";

  return (
    <SearchProvider>
      <LayoutProvider>
        <SidebarProvider defaultOpen={defaultOpen}>
          <SkipToMain />
          <AppSidebar email={email} role={role} variant={variant} />
          <SidebarInset
            className={cn(
              "@container/content min-w-0",
              "has-data-[layout=fixed]:h-svh has-data-[layout=fixed]:overflow-hidden",
              "peer-data-[variant=inset]:has-data-[layout=fixed]:h-[calc(100svh-(var(--spacing)*4))]"
            )}
          >
            {children}
          </SidebarInset>
        </SidebarProvider>
      </LayoutProvider>
    </SearchProvider>
  );
}
