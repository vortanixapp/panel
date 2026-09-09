"use client";

import Link from "next/link";
import { Menu, X } from "lucide-react";
import { BrandLogo, BrandMark } from "@/components/brand-logo";
import { cn } from "@/lib/utils";
import { dashboardPath, type PanelVariant } from "@/lib/panel-paths";
import { Button } from "@/components/ui/button";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";

type AppTitleProps = {
  variant?: PanelVariant;
};

export function AppTitle({ variant = "user" }: AppTitleProps) {
  const { setOpenMobile } = useSidebar();
  const homeHref = dashboardPath(variant);

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton
          size="lg"
          className="gap-0 py-0 hover:bg-transparent active:bg-transparent"
          asChild
        >
          <div className="flex w-full items-center">
            <Link
              href={homeHref}
              onClick={() => setOpenMobile(false)}
              className="flex flex-1 items-center"
            >
              <BrandLogo
                size="sm"
                className="max-w-[160px] group-data-[collapsible=icon]:hidden"
                priority
              />
              <BrandMark
                className="hidden size-7 group-data-[collapsible=icon]:block"
                priority
              />
            </Link>
            <ToggleSidebar className="group-data-[collapsible=icon]:hidden" />
          </div>
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}

function ToggleSidebar({
  className,
  onClick,
  ...props
}: React.ComponentProps<typeof Button>) {
  const { toggleSidebar } = useSidebar();

  return (
    <Button
      data-sidebar="trigger"
      data-slot="sidebar-trigger"
      variant="ghost"
      size="icon"
      className={cn("aspect-square size-8 max-md:scale-125", className)}
      onClick={(event) => {
        onClick?.(event);
        toggleSidebar();
      }}
      {...props}
    >
      <X className="md:hidden" />
      <Menu className="max-md:hidden" />
      <span className="sr-only">Toggle Sidebar</span>
    </Button>
  );
}
