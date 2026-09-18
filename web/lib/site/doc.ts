import { cloneBlock, isBuiltinBlock } from "@/lib/site/blocks";
import { defaultLandingBlocks } from "@/lib/site/defaults";
import { menuItems } from "@/lib/site/menu";
import { withText } from "@/lib/site/text";
import type {
  CustomPage,
  LText,
  MenuName,
  SiteBlock,
  SiteDocument,
  SiteItem,
  SitePage,
  SiteZone,
} from "@/lib/site/types";

export type ZoneRef = { page: string; zone: SiteZone };

export const MAX_MENU_DEPTH = 3;

export function isCustomKey(page: string): boolean {
  return page.startsWith("p:");
}

export function customIdOf(page: string): string {
  return page.slice(2);
}

function cleanPage(page: SitePage): SitePage | null {
  const out: SitePage = {};
  if (page.blocks) out.blocks = page.blocks;
  if (page.top?.length) out.top = page.top;
  if (page.bottom?.length) out.bottom = page.bottom;
  if (page.hidden?.length) out.hidden = page.hidden;
  return Object.keys(out).length ? out : null;
}

function withPage(doc: SiteDocument, key: string, next: SitePage): SiteDocument {
  const pages = { ...(doc.pages ?? {}) };
  const clean = cleanPage(next);
  if (clean) pages[key] = clean;
  else delete pages[key];
  const out: SiteDocument = { ...doc, pages };
  if (Object.keys(pages).length === 0) delete out.pages;
  return out;
}

export function zoneBlocks(doc: SiteDocument, ref: ZoneRef): SiteBlock[] {
  if (isCustomKey(ref.page)) {
    return doc.custom_pages?.find((page) => page.id === customIdOf(ref.page))?.blocks ?? [];
  }
  const page = doc.pages?.[ref.page];
  if (ref.zone === "blocks") return page?.blocks ?? (ref.page === "/" ? defaultLandingBlocks() : []);
  return (ref.zone === "top" ? page?.top : page?.bottom) ?? [];
}

export function setZoneBlocks(doc: SiteDocument, ref: ZoneRef, blocks: SiteBlock[]): SiteDocument {
  if (isCustomKey(ref.page)) {
    const id = customIdOf(ref.page);
    return {
      ...doc,
      custom_pages: (doc.custom_pages ?? []).map((page) => (page.id === id ? { ...page, blocks } : page)),
    };
  }
  const current: SitePage = { ...(doc.pages?.[ref.page] ?? {}) };
  if (ref.zone === "blocks") current.blocks = blocks;
  else if (ref.zone === "top") current.top = blocks;
  else current.bottom = blocks;
  return withPage(doc, ref.page, current);
}

export function findBlock(doc: SiteDocument, ref: ZoneRef, id: string): SiteBlock | undefined {
  return zoneBlocks(doc, ref).find((block) => block.id === id);
}

export function updateBlock(doc: SiteDocument, ref: ZoneRef, id: string, fn: (block: SiteBlock) => SiteBlock): SiteDocument {
  return setZoneBlocks(
    doc,
    ref,
    zoneBlocks(doc, ref).map((block) => (block.id === id ? fn(block) : block))
  );
}

export function insertBlock(doc: SiteDocument, ref: ZoneRef, index: number, block: SiteBlock): SiteDocument {
  const blocks = [...zoneBlocks(doc, ref)];
  if (isBuiltinBlock(block.type) && blocks.some((item) => item.type === block.type)) return doc;
  blocks.splice(Math.max(0, Math.min(index, blocks.length)), 0, block);
  return setZoneBlocks(doc, ref, blocks);
}

export function removeBlock(doc: SiteDocument, ref: ZoneRef, id: string): SiteDocument {
  return setZoneBlocks(
    doc,
    ref,
    zoneBlocks(doc, ref).filter((block) => block.id !== id)
  );
}

export function moveBlock(doc: SiteDocument, ref: ZoneRef, id: string, delta: number): SiteDocument {
  const blocks = [...zoneBlocks(doc, ref)];
  const from = blocks.findIndex((block) => block.id === id);
  const to = from + delta;
  if (from < 0 || to < 0 || to >= blocks.length) return doc;
  const [moved] = blocks.splice(from, 1);
  blocks.splice(to, 0, moved);
  return setZoneBlocks(doc, ref, blocks);
}

export function reorderBlocks(doc: SiteDocument, ref: ZoneRef, from: number, to: number): SiteDocument {
  const blocks = [...zoneBlocks(doc, ref)];
  if (from < 0 || from >= blocks.length || to < 0 || to > blocks.length) return doc;
  const [moved] = blocks.splice(from, 1);
  blocks.splice(to > from ? to - 1 : to, 0, moved);
  return setZoneBlocks(doc, ref, blocks);
}

export function duplicateBlock(doc: SiteDocument, ref: ZoneRef, id: string): { doc: SiteDocument; id: string | null } {
  const blocks = [...zoneBlocks(doc, ref)];
  const index = blocks.findIndex((block) => block.id === id);
  if (index < 0 || isBuiltinBlock(blocks[index].type)) return { doc, id: null };
  const copy = cloneBlock(blocks[index]);
  blocks.splice(index + 1, 0, copy);
  return { doc: setZoneBlocks(doc, ref, blocks), id: copy.id };
}

type PathPart = string | number;

function parsePath(path: string): PathPart[] {
  return path.split(".").map((part) => (/^\d+$/.test(part) ? Number(part) : part));
}

function readAt(root: unknown, parts: PathPart[]): unknown {
  let current: unknown = root;
  for (const part of parts) {
    if (current === null || typeof current !== "object") return undefined;
    current = (current as Record<string | number, unknown>)[part];
  }
  return current;
}

function writeAt(root: unknown, parts: PathPart[], value: unknown): unknown {
  if (parts.length === 0) return value;
  const [head, ...rest] = parts;
  if (typeof head === "number") {
    const list = Array.isArray(root) ? [...root] : [];
    list[head] = writeAt(list[head], rest, value);
    return list;
  }
  const obj = root && typeof root === "object" && !Array.isArray(root) ? { ...(root as Record<string, unknown>) } : {};
  const next = writeAt(obj[head], rest, value);
  if (next === undefined) delete obj[head];
  else obj[head] = next;
  return obj;
}

export function blockText(block: SiteBlock, path: string, locale: string): string {
  const value = readAt(block.props, parsePath(path));
  if (!value || typeof value !== "object" || Array.isArray(value)) return "";
  const text = (value as LText)[locale];
  return typeof text === "string" ? text : "";
}

export function setBlockText(block: SiteBlock, path: string, locale: string, text: string): SiteBlock {
  const parts = parsePath(path);
  const current = readAt(block.props, parts);
  const next = withText(current && typeof current === "object" && !Array.isArray(current) ? (current as LText) : undefined, locale, text);
  return { ...block, props: writeAt(block.props ?? {}, parts, next) as SiteBlock["props"] };
}

export function setBlockProp(block: SiteBlock, path: string, value: unknown): SiteBlock {
  const props = writeAt(block.props ?? {}, parsePath(path), value === "" || value === false ? undefined : value);
  const clean = props && typeof props === "object" && Object.keys(props as object).length ? (props as SiteBlock["props"]) : undefined;
  const out: SiteBlock = { ...block };
  if (clean) out.props = clean;
  else delete out.props;
  return out;
}

export function blockProp(block: SiteBlock, path: string): unknown {
  return readAt(block.props, parsePath(path));
}

export function setSectionHidden(doc: SiteDocument, page: string, id: string, hidden: boolean): SiteDocument {
  const current: SitePage = { ...(doc.pages?.[page] ?? {}) };
  const set = new Set(current.hidden ?? []);
  if (hidden) set.add(id);
  else set.delete(id);
  current.hidden = [...set];
  return withPage(doc, page, current);
}

export function setDraftText(doc: SiteDocument, locale: string, key: string, value: string | null): SiteDocument {
  const texts = { ...(doc.texts ?? {}) };
  const keys = { ...(texts[locale] ?? {}) };
  if (value === null) delete keys[key];
  else keys[key] = value;
  if (Object.keys(keys).length) texts[locale] = keys;
  else delete texts[locale];
  const out: SiteDocument = { ...doc, texts };
  if (Object.keys(texts).length === 0) delete out.texts;
  return out;
}

export function draftText(doc: SiteDocument, locale: string, key: string): string | undefined {
  return doc.texts?.[locale]?.[key];
}

export function menuTree(doc: SiteDocument, name: MenuName, defaultLocale: string): SiteItem[] {
  return menuItems(name, doc, defaultLocale);
}

export function setMenuTree(doc: SiteDocument, name: MenuName, items: SiteItem[]): SiteDocument {
  return { ...doc, menus: { ...(doc.menus ?? {}), [name]: { items } } };
}

export function resetMenu(doc: SiteDocument, name: MenuName): SiteDocument {
  const menus = { ...(doc.menus ?? {}) };
  delete menus[name];
  const out: SiteDocument = { ...doc, menus };
  if (Object.keys(menus).length === 0) delete out.menus;
  return out;
}

export function mapItems(items: SiteItem[], fn: (item: SiteItem) => SiteItem | null): SiteItem[] {
  const out: SiteItem[] = [];
  for (const item of items) {
    const next = fn(item);
    if (!next) continue;
    out.push(next.items ? { ...next, items: mapItems(next.items, fn) } : next);
  }
  return out;
}

export function updateItem(items: SiteItem[], id: string, fn: (item: SiteItem) => SiteItem): SiteItem[] {
  return mapItems(items, (item) => (item.id === id ? fn(item) : item));
}

export function removeItem(items: SiteItem[], id: string): SiteItem[] {
  return mapItems(items, (item) => (item.id === id ? null : item));
}

export function itemDepth(items: SiteItem[], id: string, depth = 1): number {
  for (const item of items) {
    if (item.id === id) return depth;
    const nested = item.items ? itemDepth(item.items, id, depth + 1) : 0;
    if (nested) return nested;
  }
  return 0;
}

export function subtreeHeight(item: SiteItem): number {
  if (!item.items?.length) return 1;
  return 1 + Math.max(...item.items.map(subtreeHeight));
}

function containsId(item: SiteItem, id: string): boolean {
  return item.id === id || Boolean(item.items?.some((child) => containsId(child, id)));
}

export function findMenuItem(items: SiteItem[], id: string): SiteItem | null {
  for (const item of items) {
    if (item.id === id) return item;
    const nested = item.items ? findMenuItem(item.items, id) : null;
    if (nested) return nested;
  }
  return null;
}

export type DropPosition = "before" | "after" | "inside";

export function canDrop(items: SiteItem[], dragId: string, targetId: string, position: DropPosition): boolean {
  const dragged = findMenuItem(items, dragId);
  if (!dragged || dragId === targetId || containsId(dragged, targetId)) return false;
  const targetDepth = itemDepth(items, targetId);
  if (!targetDepth) return false;
  const depth = position === "inside" ? targetDepth + 1 : targetDepth;
  return depth + subtreeHeight(dragged) - 1 <= MAX_MENU_DEPTH;
}

export function moveItem(items: SiteItem[], dragId: string, targetId: string, position: DropPosition): SiteItem[] {
  if (!canDrop(items, dragId, targetId, position)) return items;
  const dragged = findMenuItem(items, dragId);
  if (!dragged) return items;
  const without = removeItem(items, dragId);
  const place = (list: SiteItem[]): SiteItem[] => {
    const out: SiteItem[] = [];
    for (const item of list) {
      if (item.id === targetId) {
        if (position === "before") out.push(dragged, item);
        else if (position === "after") out.push(item, dragged);
        else out.push({ ...item, items: [...(item.items ?? []), dragged] });
        continue;
      }
      out.push(item.items ? { ...item, items: place(item.items) } : item);
    }
    return out;
  };
  return place(without);
}

export function shiftItem(items: SiteItem[], id: string, delta: number): SiteItem[] {
  const index = items.findIndex((item) => item.id === id);
  if (index >= 0) {
    const to = index + delta;
    if (to < 0 || to >= items.length) return items;
    const next = [...items];
    const [moved] = next.splice(index, 1);
    next.splice(to, 0, moved);
    return next;
  }
  return items.map((item) => (item.items ? { ...item, items: shiftItem(item.items, id, delta) } : item));
}

export function addCustomPage(doc: SiteDocument, page: CustomPage): SiteDocument {
  return { ...doc, custom_pages: [...(doc.custom_pages ?? []), page] };
}

export function updateCustomPage(doc: SiteDocument, id: string, patch: Partial<CustomPage>): SiteDocument {
  return {
    ...doc,
    custom_pages: (doc.custom_pages ?? []).map((page) => (page.id === id ? { ...page, ...patch } : page)),
  };
}

export function removeCustomPage(doc: SiteDocument, id: string): SiteDocument {
  const pages = (doc.custom_pages ?? []).filter((page) => page.id !== id);
  const out: SiteDocument = { ...doc, custom_pages: pages };
  if (pages.length === 0) delete out.custom_pages;
  return out;
}

const TRANSLIT: Record<string, string> = {
  а: "a", б: "b", в: "v", г: "g", д: "d", е: "e", ё: "e", ж: "zh", з: "z", и: "i", й: "y", к: "k",
  л: "l", м: "m", н: "n", о: "o", п: "p", р: "r", с: "s", т: "t", у: "u", ф: "f", х: "h", ц: "ts",
  ч: "ch", ш: "sh", щ: "sch", ъ: "", ы: "y", ь: "", э: "e", ю: "yu", я: "ya",
};

export function slugify(text: string): string {
  return text
    .toLowerCase()
    .split("")
    .map((ch) => TRANSLIT[ch] ?? ch)
    .join("")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 60)
    .replace(/-+$/g, "");
}
