import type { TranslateFn } from "@/lib/i18n";
import { isStaffRole } from "@/lib/rbac";
import {
  DEFAULT_MENUS,
  SECTION_ANCHORS,
  defaultLandingBlocks,
  type BuiltinItem,
} from "@/lib/site/defaults";
import { localized, pickText } from "@/lib/site/text";
import type { ItemAudience, MenuName, SiteDocument, SiteItem, Viewer } from "@/lib/site/types";

export type BuiltinEntry = {
  item: BuiltinItem;
  parent: string | null;
  siblings: string[];
};

const indexes = new Map<MenuName, Map<string, BuiltinEntry>>();

export function builtinIndex(name: MenuName): Map<string, BuiltinEntry> {
  const cached = indexes.get(name);
  if (cached) return cached;
  const out = new Map<string, BuiltinEntry>();
  const walk = (items: BuiltinItem[], parent: string | null) => {
    const siblings = items.map((item) => item.ref);
    for (const item of items) {
      out.set(item.ref, { item, parent, siblings });
      if (item.items) walk(item.items, item.ref);
    }
  };
  walk(DEFAULT_MENUS[name], null);
  indexes.set(name, out);
  return out;
}

export function builtinTitle(item: BuiltinItem | undefined, t: TranslateFn): string {
  if (!item) return "";
  if (item.labelKey) return t(item.labelKey);
  return item.labelText ?? "";
}

export function materializeBuiltin(item: BuiltinItem, locale: string, skip?: Set<string>): SiteItem {
  const out: SiteItem = { id: item.ref, ref: item.ref };
  if (item.kind === "group") out.kind = "group";
  if (item.badge) out.badge = { [locale]: item.badge };
  const children = (item.items ?? [])
    .filter((child) => !skip?.has(child.ref))
    .map((child) => materializeBuiltin(child, locale, skip));
  if (children.length > 0) out.items = children;
  return out;
}

export function defaultMenuItems(name: MenuName, locale: string): SiteItem[] {
  return DEFAULT_MENUS[name].map((item) => materializeBuiltin(item, locale));
}

export function collectRefs(items: SiteItem[], out = new Set<string>()): Set<string> {
  for (const item of items) {
    if (item.ref) out.add(item.ref);
    if (item.items) collectRefs(item.items, out);
  }
  return out;
}

export function findItem(items: SiteItem[], match: (item: SiteItem) => boolean): SiteItem | null {
  for (const item of items) {
    if (match(item)) return item;
    const nested = item.items ? findItem(item.items, match) : null;
    if (nested) return nested;
  }
  return null;
}

function insertPosition(list: SiteItem[], entry: BuiltinEntry): number {
  const own = entry.siblings.indexOf(entry.item.ref);
  for (let i = own - 1; i >= 0; i--) {
    const at = list.findIndex((item) => item.ref === entry.siblings[i]);
    if (at >= 0) return at + 1;
  }
  for (let i = own + 1; i < entry.siblings.length; i++) {
    const at = list.findIndex((item) => item.ref === entry.siblings[i]);
    if (at >= 0) return at;
  }
  return list.length;
}

function cloneItems(items: SiteItem[]): SiteItem[] {
  return items.map((item) => ({
    ...item,
    ...(item.items ? { items: cloneItems(item.items) } : {}),
  }));
}

export function mergeMenu(name: MenuName, config: SiteItem[], locale: string): SiteItem[] {
  const index = builtinIndex(name);
  const tree = cloneItems(config);
  const present = collectRefs(tree);
  for (const [ref, entry] of index) {
    if (present.has(ref)) continue;
    const node = materializeBuiltin(entry.item, locale, present);
    const parent = entry.parent ? findItem(tree, (item) => item.ref === entry.parent) : null;
    let list = tree;
    if (parent) {
      parent.items = parent.items ?? [];
      list = parent.items;
    }
    list.splice(insertPosition(list, entry), 0, node);
    collectRefs([node], present);
  }
  return tree;
}

export function menuItems(name: MenuName, doc: SiteDocument | undefined, locale: string): SiteItem[] {
  const config = doc?.menus?.[name];
  if (!config) return defaultMenuItems(name, locale);
  return mergeMenu(name, config.items ?? [], locale);
}

export function visibleFor(audience: ItemAudience | undefined, roles: string[] | undefined, viewer: Viewer): boolean {
  switch (audience) {
    case "guests":
      return !viewer.loggedIn;
    case "users":
      return viewer.loggedIn;
    case "staff":
      return viewer.loggedIn && isStaffRole(viewer.role) && (!roles || roles.length === 0 || roles.includes(viewer.role));
    default:
      return true;
  }
}

export function landingAnchors(doc: SiteDocument | undefined): Set<string> {
  const blocks = doc?.pages?.["/"]?.blocks ?? defaultLandingBlocks();
  const out = new Set<string>();
  for (const block of blocks) {
    if (!block.hidden && block.type.startsWith("landing.")) out.add(block.type.slice("landing.".length));
  }
  return out;
}

export type ResolvedItem = {
  id: string;
  kind: "link" | "group";
  title: string;
  url: string;
  icon?: string;
  newTab: boolean;
  badge?: string;
  items: ResolvedItem[];
};

export type ResolveContext = {
  t: TranslateFn;
  locale: string;
  defaultLocale: string;
  viewer: Viewer;
  anchors?: Set<string>;
};

function resolveItem(item: SiteItem, index: Map<string, BuiltinEntry>, ctx: ResolveContext): ResolvedItem | null {
  const base = item.ref ? index.get(item.ref)?.item : undefined;
  if (item.ref && !base) return null;
  if (item.hidden || !visibleFor(item.audience, item.roles, ctx.viewer)) return null;
  const kind = item.kind ?? base?.kind ?? "link";
  const url = (item.url || base?.url || "").trim();
  const anchor = SECTION_ANCHORS[url];
  if (anchor && ctx.anchors && !ctx.anchors.has(anchor)) return null;
  const items = resolveItems(item.items ?? [], index, ctx);
  if (kind === "group" && items.length === 0) return null;
  if (kind === "link" && !url && items.length === 0) return null;
  const title = base
    ? pickText(item.label, ctx.locale) || builtinTitle(base, ctx.t)
    : localized(item.label, ctx.locale, ctx.defaultLocale) || (kind === "link" ? url : "");
  const badge = localized(item.badge, ctx.locale, ctx.defaultLocale);
  return {
    id: item.id,
    kind,
    title,
    url,
    icon: item.icon || base?.icon,
    newTab: Boolean(item.new_tab),
    badge: badge || undefined,
    items,
  };
}

function resolveItems(items: SiteItem[], index: Map<string, BuiltinEntry>, ctx: ResolveContext): ResolvedItem[] {
  const out: ResolvedItem[] = [];
  for (const item of items) {
    const resolved = resolveItem(item, index, ctx);
    if (resolved) out.push(resolved);
  }
  return out;
}

export function resolveMenu(name: MenuName, doc: SiteDocument | undefined, ctx: ResolveContext): ResolvedItem[] {
  return resolveItems(menuItems(name, doc, ctx.defaultLocale), builtinIndex(name), ctx);
}

export function isExternalUrl(url: string): boolean {
  return /^(https?:)?\/\//i.test(url) || /^(mailto|tel):/i.test(url);
}
