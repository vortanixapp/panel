"use client";

import { useState } from "react";
import { ArrowRight, EyeOff, FilePlus2, Lock, Settings2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useEditor } from "@/components/site/editor/editor-context";
import { ChoiceField, FieldShell } from "@/components/site/editor/fields";
import { createBlock, newId } from "@/lib/site/blocks";
import { addCustomPage, slugify } from "@/lib/site/doc";
import { REGISTRY_PAGES, customPagePath, type PageGroup } from "@/lib/site/pages";
import { localized } from "@/lib/site/text";
import { cn } from "@/lib/utils";

const SLUG = /^[a-z0-9](?:[a-z0-9-]{0,58}[a-z0-9])?$/;
const GROUPS: PageGroup[] = ["site", "auth", "cabinet", "admin"];

function CreatePageDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { editor, t, tr, locale, select, navigate } = useEditor();
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [slugTouched, setSlugTouched] = useState(false);
  const [layout, setLayout] = useState<"site" | "panel">("site");
  const taken = (editor.doc.custom_pages ?? []).some((page) => page.slug === slug);
  const slugError = slug && !SLUG.test(slug) ? t("template.page.slug_invalid") : taken ? t("template.page.slug_taken") : undefined;
  const canCreate = title.trim() !== "" && SLUG.test(slug) && !taken;

  const reset = () => {
    setTitle("");
    setSlug("");
    setSlugTouched(false);
    setLayout("site");
  };

  const create = () => {
    if (!canCreate) return;
    const heading = createBlock("heading", tr, locale);
    heading.props = { ...(heading.props ?? {}), title: { [locale]: title.trim() }, size: "xl" };
    const text = createBlock("text", tr, locale);
    const page = {
      id: newId("p"),
      slug,
      title: { [locale]: title.trim() },
      layout,
      blocks: [heading, text],
    };
    editor.apply((doc) => addCustomPage(doc, page));
    navigate(customPagePath(slug));
    select({ kind: "page", id: page.id });
    reset();
    onClose();
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          reset();
          onClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("template.page.create_title")}</DialogTitle>
          <DialogDescription>{t("template.page.create_hint")}</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <FieldShell label={t("template.page.title")}>
            <Input
              autoFocus
              value={title}
              maxLength={300}
              onChange={(event) => {
                setTitle(event.target.value);
                if (!slugTouched) setSlug(slugify(event.target.value));
              }}
              className="h-9"
            />
          </FieldShell>
          <FieldShell label={t("template.page.slug")} error={slugError}>
            <div className="flex items-center rounded-md border bg-background">
              <span className="pl-2.5 font-mono text-xs text-muted-foreground">/p/</span>
              <Input
                value={slug}
                onChange={(event) => {
                  setSlugTouched(true);
                  setSlug(event.target.value.toLowerCase());
                }}
                className="h-9 border-0 pl-0.5 font-mono text-xs shadow-none focus-visible:ring-0"
              />
            </div>
          </FieldShell>
          <ChoiceField
            label={t("template.page.layout")}
            value={layout}
            options={[
              { value: "site", label: t("template.page.layout_site") },
              { value: "panel", label: t("template.page.layout_panel") },
            ]}
            onChange={(next) => setLayout(next as "site" | "panel")}
          />
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              reset();
              onClose();
            }}
          >
            {t("common.cancel")}
          </Button>
          <Button type="button" disabled={!canCreate} onClick={create}>
            {t("template.page.create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function PagesPanel() {
  const { editor, t, locale, defaultLocale, pageKey, navigate, select, selection } = useEditor();
  const [creating, setCreating] = useState(false);
  const [address, setAddress] = useState("");
  const custom = editor.doc.custom_pages ?? [];

  const row = (active: boolean) =>
    cn(
      "flex w-full items-center gap-2 rounded-lg px-2.5 py-1.5 text-left text-xs transition-colors",
      active ? "bg-primary/10 font-medium text-foreground" : "text-foreground/85 hover:bg-accent"
    );

  return (
    <div className="space-y-5">
      {GROUPS.map((group) => (
        <section key={group} className="space-y-1">
          <h3 className="mb-1.5 text-xs font-medium tracking-wide text-muted-foreground uppercase">
            {t(`template.pages.group_${group}`)}
          </h3>
          {REGISTRY_PAGES.filter((page) => page.group === group).map((page) => (
            <button key={page.key} type="button" onClick={() => navigate(page.path)} className={row(pageKey === page.key)}>
              <span className="min-w-0 flex-1 truncate">{t(page.titleKey)}</span>
              <span className="shrink-0 font-mono text-[10.5px] text-muted-foreground">{page.path}</span>
            </button>
          ))}
          {group === "site" ? (
            <>
              {custom.map((page) => {
                const key = `p:${page.id}`;
                const title = localized(page.title, locale, defaultLocale) || page.slug;
                const settingsActive = selection?.kind === "page" && selection.id === page.id;
                return (
                  <div key={page.id} className="flex items-center gap-1">
                    <button type="button" onClick={() => navigate(customPagePath(page.slug))} className={row(pageKey === key)}>
                      <span className="min-w-0 flex-1 truncate">{title}</span>
                      {page.hidden ? <EyeOff className="size-3 shrink-0 text-muted-foreground" /> : null}
                      {page.layout === "panel" || page.audience === "users" ? (
                        <Lock className="size-3 shrink-0 text-muted-foreground" />
                      ) : null}
                      <span className="shrink-0 font-mono text-[10.5px] text-muted-foreground">/p/{page.slug}</span>
                    </button>
                    <button
                      type="button"
                      title={t("template.page.settings")}
                      aria-label={t("template.page.settings")}
                      onClick={() => select({ kind: "page", id: page.id })}
                      className={cn(
                        "rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground",
                        settingsActive && "bg-accent text-foreground"
                      )}
                    >
                      <Settings2 className="size-3.5" />
                    </button>
                  </div>
                );
              })}
              <Button type="button" size="sm" variant="outline" className="mt-1 w-full border-dashed" onClick={() => setCreating(true)}>
                <FilePlus2 className="size-3.5" />
                {t("template.page.create")}
              </Button>
            </>
          ) : null}
        </section>
      ))}
      <section className="space-y-1.5">
        <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">{t("template.pages.address")}</h3>
        <form
          className="flex gap-1.5"
          onSubmit={(event) => {
            event.preventDefault();
            const path = address.trim();
            if (path.startsWith("/") && !path.startsWith("//")) navigate(path);
          }}
        >
          <Input
            value={address}
            onChange={(event) => setAddress(event.target.value)}
            placeholder="/servers"
            className="h-8 font-mono text-xs"
          />
          <Button type="submit" size="sm" variant="outline" className="h-8 px-2.5" aria-label={t("template.pages.open")}>
            <ArrowRight className="size-3.5" />
          </Button>
        </form>
      </section>
      <CreatePageDialog open={creating} onClose={() => setCreating(false)} />
    </div>
  );
}
