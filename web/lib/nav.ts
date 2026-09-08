import { adminSidebarNavGroups } from "@/components/layout/data/sidebar-data-admin";
import { userSidebarNavGroups } from "@/components/layout/data/sidebar-data-user";
import type { NavGroup, NavItem } from "@/components/layout/types";
import type { PanelVariant } from "@/lib/panel-paths";
import { isNavUrlVisible } from "@/lib/user-preferences";

function filterNavItem(item: NavItem): NavItem | null {
  if ("url" in item && item.url) {
    if (!isNavUrlVisible(item.url)) return null;
    return item;
  }

  if (item.items) {
    const subItems = item.items.filter((sub) => isNavUrlVisible(sub.url));
    if (subItems.length === 0) return null;
    return { ...item, items: subItems };
  }

  return item;
}

function filterNavGroups(groups: NavGroup[]): NavGroup[] {
  return groups
    .map((group) => {
      const items = group.items
        .map((item) => filterNavItem(item))
        .filter((item): item is NavItem => item !== null);
      if (items.length === 0) return null;
      return { ...group, items };
    })
    .filter((group): group is NavGroup => group !== null);
}

export function getUserNavGroups(): NavGroup[] {
  return filterNavGroups(userSidebarNavGroups());
}

export function getAdminNavGroups(): NavGroup[] {
  return filterNavGroups(adminSidebarNavGroups());
}

export function getNavGroupsForVariant(variant: PanelVariant): NavGroup[] {
  return variant === "admin" ? getAdminNavGroups() : getUserNavGroups();
}

export function getVisibleNavGroups(role: string): NavGroup[] {
  void role;
  return getUserNavGroups();
}
