"use client";

import { useState } from "react";
import { ChevronRight, ExternalLink, EyeOff, FolderPlus, GripVertical, Link2, RotateCcw, Shield, UserCheck, UserX } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useEditor } from "@/components/site/editor/editor-context";
import { newId } from "@/lib/site/blocks";
import {
  canDrop,
  findMenuItem,
  menuTree,
  moveItem,
  resetMenu,
  setMenuTree,
  updateItem,
  type DropPosition,
} from "@/lib/site/doc";
import { siteIcon } from "@/lib/site/icons";
import { builtinIndex, builtinTitle, isExternalUrl } from "@/lib/site/menu";
import { pickText, localized } from "@/lib/site/text";
import { MENU_NAMES, type MenuName, type SiteItem } from "@/lib/site/types";
import { cn } from "@/lib/utils";

const PREVIEW_PATH: Record<MenuName, string> = {
  site_header: "/",
  site_footer: "/",
  user_sidebar: "/dashboard",
  admin_sidebar: "/admin/dashboard",
};

function matchesPreview(menu: MenuName, path: string): boolean {
  if (menu === "admin_sidebar") return path.startsWith("/admin");
  if (menu === "user_sidebar") return !path.startsWith("/admin") && !path.startsWith("/p/") && path !== "/";
  return path === "/" || path.startsWith("/p/");
}

export function MenusPanel({ menu, onMenuChange }: { menu: MenuName; onMenuChange: (menu: MenuName) => void }) {
  const { editor, t, locale, defaultLocale, selection, select, navigate, path } = useEditor();
  const items = menuTree(editor.doc, menu, defaultLocale);
  const index = builtinIndex(menu);
  const configured = Boolean(editor.doc.menus?.[menu]);
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());
  const [drag, setDrag] = useState<string | null>(null);
  const [over, setOver] = useState<{ id: string; position: DropPosition } | null>(null);

  const commit = (next: SiteItem[]) => editor.apply((doc) => setMenuTree(doc, menu, next));

  const titleOf = (item: SiteItem) => {
    const base = item.ref ? index.get(item.ref)?.item : undefined;
    if (base) return pickText(item.label, locale) || builtinTitle(base, t);
    return localized(item.label, locale, defaultLocale) || item.url || t("template.menu.untitled");
  };

  const choose = (next: MenuName) => {
    onMenuChange(next);
    if (!matchesPreview(next, path)) navigate(PREVIEW_PATH[next]);
  };

  const add = (kind: "link" | "group") => {
    const item: SiteItem =
      kind === "group"
        ? { id: newId("g"), kind: "group", label: { [locale]: t("template.menu.new_group") }, items: [] }
        : { id: newId("m"), kind: "link", label: { [locale]: t("template.menu.new_link") }, url: "/" };
    const target = selection?.kind === "item" && selection.menu === menu ? findMenuItem(items, selection.id) : null;
    if (kind === "link" && target && (target.kind ?? index.get(target.ref ?? "")?.item.kind) === "group") {
      commit(updateItem(items, target.id, (entry) => ({ ...entry, items: [...(entry.items ?? []), item] })));
    } else {
      commit([...items, item]);
    }
    select({ kind: "item", menu, id: item.id });
  };

  const positionFor = (event: React.DragEvent<HTMLDivElement>, item: SiteItem): DropPosition => {
    const rect = event.currentTarget.getBoundingClientRect();
    const y = (event.clientY - rect.top) / rect.height;
    const kind = item.kind ?? index.get(item.ref ?? "")?.item.kind ?? "link";
    if (y < 0.3) return "before";
    if (y > 0.7) return "after";
    return kind === "group" || (item.items?.length ?? 0) > 0 ? "inside" : y < 0.5 ? "before" : "after";
  };

  const renderItems = (list: SiteItem[], depth: number) =>
    list.map((item) => {
      const base = item.ref ? index.get(item.ref)?.item : undefined;
      const kind = item.kind ?? base?.kind ?? "link";
      const Icon = siteIcon(item.icon || base?.icon);
      const children = item.items ?? [];
      const isCollapsed = collapsed.has(item.id);
      const active = selection?.kind === "item" && selection.menu === menu && selection.id === item.id;
      const url = item.url || base?.url || "";
      const marker = over?.id === item.id ? over.position : null;
      return (
        <div key={item.id}>
          <div
            draggable
            onDragStart={(event) => {
              setDrag(item.id);
              event.dataTransfer.effectAllowed = "move";
              event.dataTransfer.setData("text/plain", item.id);
            }}
            onDragEnd={() => {
              setDrag(null);
              setOver(null);
            }}
            onDragOver={(event) => {
              if (!drag) return;
              const position = positionFor(event, item);
              if (!canDrop(items, drag, item.id, position)) {
                setOver(null);
                return;
              }
              event.preventDefault();
              if (over?.id !== item.id || over.position !== position) setOver({ id: item.id, position });
            }}
            onDrop={(event) => {
              event.preventDefault();
              if (drag && over) commit(moveItem(items, drag, over.id, over.position));
              setDrag(null);
              setOver(null);
            }}
            className={cn(
              "group relative flex items-center gap-1.5 rounded-lg border py-1 pr-1.5 transition-colors",
              active ? "border-primary bg-primary/[0.05]" : "border-transparent hover:border-border hover:bg-accent/40",
              drag === item.id && "opacity-40",
              marker === "inside" && "border-primary border-dashed bg-primary/[0.06]",
              item.hidden && "opacity-60"
            )}
            style={{ paddingLeft: 6 + depth * 16 }}
          >
            {marker === "before" || marker === "after" ? (
              <span
                className={cn(
                  "pointer-events-none absolute right-1 h-0.5 rounded-full bg-primary",
                  marker === "after" ? "-bottom-[2px]" : "-top-[2px]"
                )}
                style={{ left: 6 + depth * 16 }}
              />
            ) : null}
            <GripVertical className="size-3.5 shrink-0 cursor-grab text-muted-foreground/60" />
            {children.length > 0 ? (
              <button
                type="button"
                aria-label={isCollapsed ? t("template.menu.expand") : t("template.menu.collapse")}
                onClick={() =>
                  setCollapsed((current) => {
                    const next = new Set(current);
                    if (next.has(item.id)) next.delete(item.id);
                    else next.add(item.id);
                    return next;
                  })
                }
                className="rounded p-0.5 text-muted-foreground hover:bg-accent"
              >
                <ChevronRight className={cn("size-3.5 transition-transform", !isCollapsed && "rotate-90")} />
              </button>
            ) : (
              <span className="w-[18px]" />
            )}
            <button
              type="button"
              onClick={() => select({ kind: "item", menu, id: item.id })}
              className="flex min-w-0 flex-1 items-center gap-2 text-left"
            >
              {Icon ? <Icon className="size-3.5 shrink-0 text-muted-foreground" /> : null}
              <span className={cn("truncate text-xs", kind === "group" && "font-semibold tracking-wide uppercase text-muted-foreground")}>
                {titleOf(item)}
              </span>
            </button>
            {item.hidden ? <EyeOff className="size-3 shrink-0 text-muted-foreground" /> : null}
            {item.audience === "guests" ? <UserX className="size-3 shrink-0 text-muted-foreground" /> : null}
            {item.audience === "users" ? <UserCheck className="size-3 shrink-0 text-muted-foreground" /> : null}
            {item.audience === "staff" ? <Shield className="size-3 shrink-0 text-muted-foreground" /> : null}
            {kind === "link" && (item.new_tab || isExternalUrl(url)) ? (
              <ExternalLink className="size-3 shrink-0 text-muted-foreground" />
            ) : null}
          </div>
          {children.length > 0 && !isCollapsed ? renderItems(children, depth + 1) : null}
        </div>
      );
    });

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-1 rounded-lg bg-muted/60 p-0.5">
        {MENU_NAMES.map((name) => (
          <button
            key={name}
            type="button"
            onClick={() => choose(name)}
            className={cn(
              "h-8 rounded-md px-2 text-xs transition-colors",
              menu === name ? "bg-background font-medium text-foreground shadow-xs" : "text-muted-foreground hover:text-foreground"
            )}
          >
            {t(`template.menu.name_${name}`)}
          </button>
        ))}
      </div>
      <p className="text-[11px] leading-snug text-muted-foreground">{t(`template.menu.hint_${menu}`)}</p>
      <div
        className="space-y-0.5"
        onDragLeave={(event) => {
          if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setOver(null);
        }}
      >
        {renderItems(items, 0)}
      </div>
      <div className="grid grid-cols-2 gap-2">
        <Button type="button" size="sm" variant="outline" onClick={() => add("link")}>
          <Link2 className="size-3.5" />
          {t("template.menu.add_link")}
        </Button>
        <Button type="button" size="sm" variant="outline" onClick={() => add("group")}>
          <FolderPlus className="size-3.5" />
          {t("template.menu.add_group")}
        </Button>
      </div>
      {configured ? (
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="w-full text-muted-foreground"
          onClick={() => {
            editor.apply((doc) => resetMenu(doc, menu));
            if (selection?.kind === "item" && selection.menu === menu) select(null);
          }}
        >
          <RotateCcw className="size-3.5" />
          {t("template.menu.reset")}
        </Button>
      ) : null}
    </div>
  );
}
