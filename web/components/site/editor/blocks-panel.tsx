"use client";

import { useState } from "react";
import { Eye, EyeOff, GripVertical, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useEditor } from "@/components/site/editor/editor-context";
import { blockMeta } from "@/lib/site/blocks";
import {
  customIdOf,
  isCustomKey,
  reorderBlocks,
  setSectionHidden,
  updateBlock,
  zoneBlocks,
  type ZoneRef,
} from "@/lib/site/doc";
import { siteIcon } from "@/lib/site/icons";
import { localized } from "@/lib/site/text";
import type { LText, SiteBlock, SiteZone } from "@/lib/site/types";
import { cn } from "@/lib/utils";

function snippet(block: SiteBlock, locale: string, defaultLocale: string): string {
  const props = block.props ?? {};
  for (const name of ["title", "q", "eyebrow", "text", "caption"]) {
    const value = props[name];
    if (value && typeof value === "object" && !Array.isArray(value)) {
      const text = localized(value as LText, locale, defaultLocale);
      if (text) return text;
    }
  }
  const html = props.html;
  if (html && typeof html === "object") {
    const text = localized(html as LText, locale, defaultLocale).replace(/<[^>]+>/g, " ").replace(/\s+/g, " ").trim();
    if (text) return text;
  }
  return "";
}

function ZoneList({ title, zoneRef, hint }: { title: string; zoneRef: ZoneRef; hint?: string }) {
  const { editor, t, locale, defaultLocale, selection, select, openLibrary } = useEditor();
  const blocks = zoneBlocks(editor.doc, zoneRef);
  const [drag, setDrag] = useState<number | null>(null);
  const [over, setOver] = useState<{ index: number; after: boolean } | null>(null);

  const drop = (to: number) => {
    if (drag === null) return;
    editor.apply((doc) => reorderBlocks(doc, zoneRef, drag, to));
    setDrag(null);
    setOver(null);
  };

  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">{title}</h3>
        <span className="text-[11px] text-muted-foreground">{blocks.length}</span>
      </div>
      {hint ? <p className="text-[11px] leading-snug text-muted-foreground">{hint}</p> : null}
      <div className="space-y-1">
        {blocks.map((block, index) => {
          const meta = blockMeta(block.type);
          const Icon = siteIcon(meta?.icon);
          const active = selection?.kind === "block" && selection.id === block.id && selection.page === zoneRef.page;
          const text = snippet(block, locale, defaultLocale);
          return (
            <div
              key={block.id}
              draggable
              onDragStart={(event) => {
                setDrag(index);
                event.dataTransfer.effectAllowed = "move";
                event.dataTransfer.setData("text/plain", block.id);
              }}
              onDragEnd={() => {
                setDrag(null);
                setOver(null);
              }}
              onDragOver={(event) => {
                if (drag === null) return;
                event.preventDefault();
                const rect = event.currentTarget.getBoundingClientRect();
                setOver({ index, after: event.clientY > rect.top + rect.height / 2 });
              }}
              onDrop={(event) => {
                event.preventDefault();
                if (over) drop(over.after ? over.index + 1 : over.index);
              }}
              className={cn(
                "group relative flex items-center gap-2 rounded-lg border bg-background px-2 py-1.5 transition-colors",
                active ? "border-primary bg-primary/[0.05]" : "hover:border-foreground/20",
                drag === index && "opacity-40",
                block.hidden && "opacity-60"
              )}
            >
              {over?.index === index ? (
                <span
                  className={cn(
                    "pointer-events-none absolute inset-x-1 h-0.5 rounded-full bg-primary",
                    over.after ? "-bottom-[3px]" : "-top-[3px]"
                  )}
                />
              ) : null}
              <GripVertical className="size-3.5 shrink-0 cursor-grab text-muted-foreground/60" />
              <button
                type="button"
                onClick={() => select({ kind: "block", ...zoneRef, id: block.id }, true)}
                className="flex min-w-0 flex-1 items-center gap-2 text-left"
              >
                <span className="flex size-6 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
                  {Icon ? <Icon className="size-3.5" /> : null}
                </span>
                <span className="min-w-0">
                  <span className="block truncate text-xs font-medium">{meta ? t(meta.titleKey) : block.type}</span>
                  {text ? <span className="block truncate text-[11px] text-muted-foreground">{text}</span> : null}
                </span>
              </button>
              <button
                type="button"
                title={block.hidden ? t("template.action.show") : t("template.action.hide")}
                aria-label={block.hidden ? t("template.action.show") : t("template.action.hide")}
                onClick={() =>
                  editor.apply((doc) => updateBlock(doc, zoneRef, block.id, (current) => ({ ...current, hidden: !current.hidden || undefined })))
                }
                className="rounded p-1 text-muted-foreground opacity-70 transition-opacity group-hover:opacity-100 hover:bg-accent hover:text-foreground"
              >
                {block.hidden ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
              </button>
            </div>
          );
        })}
      </div>
      <Button
        type="button"
        size="sm"
        variant="outline"
        className="w-full border-dashed"
        onClick={() => openLibrary({ ...zoneRef, index: blocks.length })}
      >
        <Plus className="size-3.5" />
        {t("template.blocks.add")}
      </Button>
    </section>
  );
}

function SectionsList() {
  const { editor, t, pageKey, inventory, path, selection, select } = useEditor();
  const sections = inventory && inventory.path === path ? inventory.sections : [];
  if (sections.length === 0) return null;
  const hidden = new Set(editor.doc.pages?.[pageKey]?.hidden ?? []);
  return (
    <section className="space-y-2">
      <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">{t("template.blocks.sections")}</h3>
      <p className="text-[11px] leading-snug text-muted-foreground">{t("template.blocks.sections_hint")}</p>
      <div className="space-y-1">
        {sections.map((section) => {
          const isHidden = hidden.has(section.id);
          const active = selection?.kind === "section" && selection.id === section.id;
          return (
            <div
              key={section.id}
              className={cn(
                "flex items-center gap-2 rounded-lg border px-2.5 py-1.5",
                active ? "border-amber-500/60 bg-amber-500/[0.06]" : "hover:border-foreground/20",
                isHidden && "opacity-60"
              )}
            >
              <button
                type="button"
                onClick={() => select({ kind: "section", page: pageKey, id: section.id })}
                className="min-w-0 flex-1 truncate text-left text-xs"
              >
                {t(`template.section.${section.id}`)}
              </button>
              <button
                type="button"
                title={isHidden ? t("template.action.show") : t("template.action.hide")}
                aria-label={isHidden ? t("template.action.show") : t("template.action.hide")}
                onClick={() => editor.apply((doc) => setSectionHidden(doc, pageKey, section.id, !isHidden))}
                className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
              >
                {isHidden ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
              </button>
            </div>
          );
        })}
      </div>
    </section>
  );
}

export function BlocksPanel() {
  const { editor, t, pageKey } = useEditor();
  const custom = isCustomKey(pageKey) ? editor.doc.custom_pages?.find((page) => page.id === customIdOf(pageKey)) : undefined;

  if (isCustomKey(pageKey) && !custom) {
    return <p className="text-xs text-muted-foreground">{t("template.inspector.missing")}</p>;
  }

  if (pageKey === "/" || custom) {
    return (
      <div className="space-y-6">
        <ZoneList title={t("template.blocks.page_blocks")} zoneRef={{ page: pageKey, zone: "blocks" }} />
      </div>
    );
  }

  const zones: { zone: SiteZone; title: string }[] = [
    { zone: "top", title: t("template.blocks.top") },
    { zone: "bottom", title: t("template.blocks.bottom") },
  ];
  return (
    <div className="space-y-6">
      <p className="text-[11px] leading-snug text-muted-foreground">{t("template.blocks.slots_hint")}</p>
      {zones.map((item) => (
        <ZoneList key={item.zone} title={item.title} zoneRef={{ page: pageKey, zone: item.zone }} />
      ))}
      <SectionsList />
    </div>
  );
}
