"use client";

import { type ReactNode } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronRight } from "lucide-react";
import { m } from "motion/react";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  type NavCollapsible,
  type NavItem,
  type NavLink,
  type NavGroup as NavGroupProps,
} from "./types";

function linkTarget(item: { url: string; newTab?: boolean }) {
  return item.newTab || /^(https?:)?\/\//i.test(item.url)
    ? { target: "_blank", rel: "noopener noreferrer" }
    : {};
}

export function NavGroup({ title, items }: NavGroupProps) {
  const pathname = usePathname();
  const { state, isMobile } = useSidebar();

  return (
    <SidebarGroup>
      {title ? <SidebarGroupLabel>{title}</SidebarGroupLabel> : null}
      <SidebarMenu>
        {items.map((item) => {
          const key = item.id ?? `${item.title}-${"url" in item ? item.url : "group"}`;

          if (!item.items)
            return (
              <SidebarMenuLink key={key} item={item} pathname={pathname} />
            );

          if (state === "collapsed" && !isMobile)
            return (
              <SidebarMenuCollapsedDropdown
                key={key}
                item={item}
                pathname={pathname}
              />
            );

          return (
            <SidebarMenuCollapsible
              key={key}
              item={item}
              pathname={pathname}
            />
          );
        })}
      </SidebarMenu>
    </SidebarGroup>
  );
}

function NavBadge({ children }: { children: ReactNode }) {
  return <Badge className="relative rounded-full px-1 py-0 text-xs">{children}</Badge>;
}

const ITEM_MOTION =
  "relative overflow-visible group-data-[collapsible=icon]:overflow-hidden data-[active=true]:bg-transparent [&>svg]:transition-transform [&>svg]:duration-200 hover:[&>svg]:scale-110";

function ActiveGlow({ active }: { active: boolean }) {
  if (!active) return null;
  return (
    <m.span
      layoutId="sidebar-active"
      className="absolute inset-0 rounded-md bg-sidebar-accent"
      transition={{ type: "spring", stiffness: 420, damping: 36 }}
    />
  );
}

function SidebarMenuLink({
  item,
  pathname,
}: {
  item: NavLink;
  pathname: string;
}) {
  const { setOpenMobile } = useSidebar();
  const active = checkIsActive(pathname, item);

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        asChild
        isActive={active}
        tooltip={item.title}
        className={ITEM_MOTION}
      >
        <Link href={item.url} {...linkTarget(item)} onClick={() => setOpenMobile(false)}>
          <ActiveGlow active={active} />
          {item.icon && <item.icon className="relative" />}
          <span className="relative">{item.title}</span>
          {item.badge && <NavBadge>{item.badge}</NavBadge>}
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}

function SidebarMenuCollapsible({
  item,
  pathname,
}: {
  item: NavCollapsible;
  pathname: string;
}) {
  const { setOpenMobile } = useSidebar();

  return (
    <Collapsible
      asChild
      defaultOpen={checkIsActive(pathname, item, true)}
      className="group/collapsible"
    >
      <SidebarMenuItem>
        <CollapsibleTrigger asChild>
          <SidebarMenuButton tooltip={item.title} className="[&>svg]:transition-transform [&>svg]:duration-200 hover:[&>svg:first-child]:scale-110">
            {item.icon && <item.icon />}
            <span>{item.title}</span>
            {item.badge && <NavBadge>{item.badge}</NavBadge>}
            <ChevronRight className="ms-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
          </SidebarMenuButton>
        </CollapsibleTrigger>
        <CollapsibleContent className="CollapsibleContent">
          <SidebarMenuSub>
            {item.items.map((subItem) => {
              const active = checkIsActive(pathname, subItem);
              return (
                <SidebarMenuSubItem key={subItem.id ?? subItem.title}>
                  <SidebarMenuSubButton
                    asChild
                    isActive={active}
                    className="relative data-[active=true]:bg-transparent"
                  >
                    <Link
                      href={subItem.url}
                      {...linkTarget(subItem)}
                      onClick={() => setOpenMobile(false)}
                    >
                      <ActiveGlow active={active} />
                      {subItem.icon && <subItem.icon className="relative" />}
                      <span className="relative">{subItem.title}</span>
                      {subItem.badge && <NavBadge>{subItem.badge}</NavBadge>}
                    </Link>
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              );
            })}
          </SidebarMenuSub>
        </CollapsibleContent>
      </SidebarMenuItem>
    </Collapsible>
  );
}

function SidebarMenuCollapsedDropdown({
  item,
  pathname,
}: {
  item: NavCollapsible;
  pathname: string;
}) {
  return (
    <SidebarMenuItem>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <SidebarMenuButton
            tooltip={item.title}
            isActive={checkIsActive(pathname, item)}
          >
            {item.icon && <item.icon />}
            <span>{item.title}</span>
            {item.badge && <NavBadge>{item.badge}</NavBadge>}
            <ChevronRight className="ms-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
          </SidebarMenuButton>
        </DropdownMenuTrigger>
        <DropdownMenuContent side="right" align="start" sideOffset={4}>
          <DropdownMenuLabel>
            {item.title} {item.badge ? `(${item.badge})` : ""}
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          {item.items.map((sub) => (
            <DropdownMenuItem key={sub.id ?? `${sub.title}-${sub.url}`} asChild>
              <Link
                href={sub.url}
                {...linkTarget(sub)}
                className={
                  checkIsActive(pathname, sub) ? "bg-secondary" : undefined
                }
              >
                {sub.icon && <sub.icon />}
                <span className="max-w-52 text-wrap">{sub.title}</span>
                {sub.badge && (
                  <span className="ms-auto text-xs">{sub.badge}</span>
                )}
              </Link>
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </SidebarMenuItem>
  );
}

function checkIsActive(pathname: string, item: NavItem, mainNav = false): boolean {
  const url = "url" in item ? item.url : undefined;
  return (
    pathname === url ||
    pathname.split("?")[0] === url ||
    !!item.items?.some((i) => i.url === pathname) ||
    (mainNav &&
      !!url &&
      pathname.split("/")[1] !== "" &&
      pathname.split("/")[1] === url.split("/")[1])
  );
}
