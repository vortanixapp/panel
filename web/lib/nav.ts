import type { NavGroup, NavItem } from "@/components/layout/types";
import type { PanelVariant } from "@/lib/panel-paths";
import { siteIcon } from "@/lib/site/icons";
import { resolveMenu, type ResolveContext, type ResolvedItem } from "@/lib/site/menu";
import type { SiteDocument } from "@/lib/site/types";
import { isNavUrlVisible } from "@/lib/user-preferences";

function toNavItem(item: ResolvedItem): NavItem | null {
  const icon = siteIcon(item.icon);
  if (item.items.length > 0) {
    const items = item.items
      .filter((sub) => sub.url && isNavUrlVisible(sub.url))
      .map((sub) => ({
        id: sub.id,
        title: sub.title,
        url: sub.url,
        icon: siteIcon(sub.icon),
        badge: sub.badge,
        newTab: sub.newTab,
      }));
    if (items.length === 0) return null;
    return { id: item.id, title: item.title, icon, badge: item.badge, items };
  }
  if (!item.url || !isNavUrlVisible(item.url)) return null;
  return { id: item.id, title: item.title, url: item.url, icon, badge: item.badge, newTab: item.newTab };
}

export function sidebarGroups(
  variant: PanelVariant,
  doc: SiteDocument | undefined,
  ctx: ResolveContext
): NavGroup[] {
  const groups: NavGroup[] = [];
  let loose: NavGroup | null = null;
  const menu = resolveMenu(variant === "admin" ? "admin_sidebar" : "user_sidebar", doc, ctx);
  for (const item of menu) {
    if (item.kind === "group") {
      loose = null;
      const items = item.items.map(toNavItem).filter((nav): nav is NavItem => nav !== null);
      if (items.length > 0) groups.push({ id: item.id, title: item.title, items });
      continue;
    }
    const nav = toNavItem(item);
    if (!nav) continue;
    if (!loose) {
      loose = { id: `${item.id}-loose`, title: "", items: [] };
      groups.push(loose);
    }
    loose.items.push(nav);
  }
  return groups;
}
