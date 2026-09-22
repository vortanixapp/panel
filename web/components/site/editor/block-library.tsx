"use client";

import { useMemo, useState } from "react";
import { Search } from "lucide-react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useEditor, type LibraryTarget } from "@/components/site/editor/editor-context";
import { BLOCK_GROUPS, BLOCK_TYPES, createBlock, type BlockMeta } from "@/lib/site/blocks";
import { customIdOf, insertBlock, isCustomKey, zoneBlocks } from "@/lib/site/doc";
import { siteIcon } from "@/lib/site/icons";
import { cn } from "@/lib/utils";

export function BlockLibrary({ target, onClose }: { target: LibraryTarget | null; onClose: () => void }) {
  const { editor, t, tr, locale, select } = useEditor();
  const [query, setQuery] = useState("");

  const siteZone = useMemo(() => {
    if (!target || target.zone !== "blocks") return false;
    if (target.page === "/") return true;
    if (!isCustomKey(target.page)) return false;
    const page = editor.doc.custom_pages?.find((item) => item.id === customIdOf(target.page));
    return (page?.layout ?? "site") === "site";
  }, [target, editor.doc]);

  const existing = useMemo(
    () => new Set(target ? zoneBlocks(editor.doc, target).map((block) => block.type) : []),
    [target, editor.doc]
  );

  const needle = query.trim().toLowerCase();
  const groups = BLOCK_GROUPS.map((group) => ({
    group,
    items: BLOCK_TYPES.filter((meta) => {
      if (meta.group !== group) return false;
      if (meta.group === "builtin" && !siteZone) return false;
      if (!needle) return true;
      return `${t(meta.titleKey)} ${t(meta.descriptionKey)}`.toLowerCase().includes(needle);
    }),
  })).filter((entry) => entry.items.length > 0);

  const pick = (meta: BlockMeta) => {
    if (!target) return;
    const block = createBlock(meta.type, tr, locale);
    editor.apply((doc) => insertBlock(doc, target, target.index, block));
    select({ kind: "block", page: target.page, zone: target.zone, id: block.id }, true);
    setQuery("");
    onClose();
  };

  return (
    <Dialog open={target !== null} onOpenChange={(open) => (open ? undefined : onClose())}>
      <DialogContent className="flex max-h-[85vh] flex-col gap-0 p-0 sm:max-w-2xl">
        <DialogHeader className="border-b px-5 pt-5 pb-4">
          <DialogTitle>{t("template.library.title")}</DialogTitle>
          <DialogDescription>{t("template.library.subtitle")}</DialogDescription>
          <div className="relative mt-3">
            <Search className="absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              autoFocus
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t("template.library.search")}
              className="h-9 pl-8"
            />
          </div>
        </DialogHeader>
        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
          <div className="space-y-5 px-5 py-4">
            {groups.map(({ group, items }) => (
              <section key={group}>
                <h3 className="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
                  {t(`template.library.group_${group}`)}
                </h3>
                <div className="grid gap-2 sm:grid-cols-2">
                  {items.map((meta) => {
                    const Icon = siteIcon(meta.icon);
                    const taken = meta.group === "builtin" && existing.has(meta.type);
                    return (
                      <button
                        key={meta.type}
                        type="button"
                        disabled={taken}
                        onClick={() => pick(meta)}
                        className={cn(
                          "flex items-start gap-3 rounded-xl border p-3 text-left transition-colors",
                          taken ? "cursor-not-allowed opacity-50" : "hover:border-primary/50 hover:bg-primary/[0.04]"
                        )}
                      >
                        <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                          {Icon ? <Icon className="size-4" /> : null}
                        </span>
                        <span className="min-w-0">
                          <span className="block text-sm font-medium">{t(meta.titleKey)}</span>
                          <span className="mt-0.5 block text-xs leading-snug text-muted-foreground">
                            {taken ? t("template.library.taken") : t(meta.descriptionKey)}
                          </span>
                        </span>
                      </button>
                    );
                  })}
                </div>
              </section>
            ))}
            {groups.length === 0 ? (
              <p className="py-8 text-center text-sm text-muted-foreground">{t("template.library.empty")}</p>
            ) : null}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
