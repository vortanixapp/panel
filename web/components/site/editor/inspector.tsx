"use client";

import { useEffect, useState, type ReactNode } from "react";
import {
  ArrowDown,
  ArrowUp,
  ChevronDown,
  Copy,
  ExternalLink,
  Eye,
  EyeOff,
  Plus,
  RotateCcw,
  Trash2,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useEditor, type Selection } from "@/components/site/editor/editor-context";
import {
  BoolField,
  ChoiceField,
  FieldShell,
  IconField,
  ImageField,
  LTextField,
  UrlField,
} from "@/components/site/editor/fields";
import { RichTextField } from "@/components/site/editor/rich-text";
import { APPEARANCE_FIELDS, blockMeta, cloneBlock, isBuiltinBlock, type FieldSpec } from "@/lib/site/blocks";
import {
  blockProp,
  blockText,
  findBlock,
  insertBlock,
  findMenuItem,
  menuTree,
  removeBlock,
  removeCustomPage,
  removeItem,
  setBlockProp,
  setBlockText,
  setMenuTree,
  setSectionHidden,
  shiftItem,
  updateBlock,
  updateCustomPage,
  updateItem,
  zoneBlocks,
  type ZoneRef,
} from "@/lib/site/doc";
import { siteIcon } from "@/lib/site/icons";
import { builtinIndex, builtinTitle } from "@/lib/site/menu";
import { customPagePath } from "@/lib/site/pages";
import { localized } from "@/lib/site/text";
import type { LText, SiteBlock, SiteItem } from "@/lib/site/types";
import { cn } from "@/lib/utils";

const SLUG = /^[a-z0-9](?:[a-z0-9-]{0,58}[a-z0-9])?$/;
const STAFF_ROLES = ["owner", "admin", "support"] as const;

function ConfirmButton({
  label,
  confirmLabel,
  onConfirm,
}: {
  label: string;
  confirmLabel: string;
  onConfirm: () => void;
}) {
  const [armed, setArmed] = useState(false);
  useEffect(() => {
    if (!armed) return;
    const timer = setTimeout(() => setArmed(false), 3500);
    return () => clearTimeout(timer);
  }, [armed]);
  return (
    <Button
      type="button"
      size="sm"
      variant={armed ? "destructive" : "outline"}
      onClick={() => {
        if (armed) onConfirm();
        else setArmed(true);
      }}
    >
      <Trash2 className="size-3.5" />
      {armed ? confirmLabel : label}
    </Button>
  );
}

function SpecField({
  spec,
  block,
  path,
  update,
}: {
  spec: FieldSpec;
  block: SiteBlock;
  path: string;
  update: (fn: (block: SiteBlock) => SiteBlock, key: string) => void;
}) {
  const { t, locale } = useEditor();
  const label = t(spec.label);
  const value = blockProp(block, path);
  const set = (next: unknown) => update((current) => setBlockProp(current, path, next), path);
  switch (spec.kind) {
    case "text":
    case "long":
      return (
        <LTextField
          label={label}
          value={value as LText | undefined}
          multiline={spec.kind === "long"}
          maxLength={spec.kind === "text" ? 300 : 5000}
          onChange={set}
        />
      );
    case "html":
      return (
        <FieldShell label={label}>
          <RichTextField
            value={blockText(block, path, locale)}
            mode={block.type === "html" ? "html" : "rich"}
            onChange={(html) => update((current) => setBlockText(current, path, locale, html), path)}
          />
        </FieldShell>
      );
    case "url":
      return <UrlField label={label} value={typeof value === "string" ? value : ""} onChange={set} />;
    case "video":
      return <UrlField kind="video" label={label} value={typeof value === "string" ? value : ""} onChange={set} />;
    case "image":
      return <ImageField label={label} value={typeof value === "string" ? value : ""} onChange={set} />;
    case "icon":
      return <IconField label={label} value={typeof value === "string" ? value : ""} onChange={set} />;
    case "bool":
      return <BoolField label={label} checked={value === true} onChange={set} />;
    case "enum":
      return (
        <ChoiceField
          label={label}
          value={typeof value === "string" ? value : spec.options[0].value}
          options={spec.options.map((option) => ({ value: option.value, label: t(option.label) }))}
          onChange={set}
        />
      );
    case "int": {
      const options = [];
      for (let n = spec.min; n <= spec.max; n++) options.push({ value: String(n), label: String(n) });
      return (
        <ChoiceField
          label={label}
          value={String(typeof value === "number" ? value : spec.min)}
          options={options}
          onChange={(next) => set(Number(next))}
        />
      );
    }
    case "list":
      return <ListEditor spec={spec} block={block} path={path} update={update} />;
  }
}

function ListEditor({
  spec,
  block,
  path,
  update,
}: {
  spec: Extract<FieldSpec, { kind: "list" }>;
  block: SiteBlock;
  path: string;
  update: (fn: (block: SiteBlock) => SiteBlock, key: string) => void;
}) {
  const { t, locale, defaultLocale } = useEditor();
  const raw = blockProp(block, path);
  const items = Array.isArray(raw) ? (raw as Record<string, unknown>[]) : [];
  const [open, setOpen] = useState<number | null>(null);
  const setItems = (next: Record<string, unknown>[]) =>
    update((current) => setBlockProp(current, path, next.length ? next : undefined), `${path}:list`);
  const move = (index: number, delta: number) => {
    const to = index + delta;
    if (to < 0 || to >= items.length) return;
    const next = [...items];
    const [moved] = next.splice(index, 1);
    next.splice(to, 0, moved);
    setItems(next);
    setOpen(to);
  };
  return (
    <FieldShell label={t(spec.label)}>
      <div className="space-y-1.5">
        {items.map((item, index) => {
          const title = localized(item[spec.titleField] as LText | undefined, locale, defaultLocale);
          const expanded = open === index;
          return (
            <div key={index} className="rounded-lg border bg-muted/20">
              <div className="flex items-center gap-1 pr-1">
                <button
                  type="button"
                  onClick={() => setOpen(expanded ? null : index)}
                  className="flex min-w-0 flex-1 items-center gap-2 px-2.5 py-2 text-left text-xs"
                >
                  <ChevronDown className={cn("size-3.5 shrink-0 text-muted-foreground transition-transform", !expanded && "-rotate-90")} />
                  <span className="truncate">{title || `${t(spec.itemLabel)} ${index + 1}`}</span>
                </button>
                <button type="button" title={t("template.action.up")} onClick={() => move(index, -1)} className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground">
                  <ArrowUp className="size-3.5" />
                </button>
                <button type="button" title={t("template.action.down")} onClick={() => move(index, 1)} className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground">
                  <ArrowDown className="size-3.5" />
                </button>
                <button
                  type="button"
                  title={t("template.action.remove")}
                  onClick={() => {
                    setItems(items.filter((_, i) => i !== index));
                    setOpen(null);
                  }}
                  className="rounded p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                >
                  <Trash2 className="size-3.5" />
                </button>
              </div>
              {expanded ? (
                <div className="space-y-3 border-t px-2.5 py-3">
                  {spec.item.map((field) => (
                    <SpecField key={field.name} spec={field} block={block} path={`${path}.${index}.${field.name}`} update={update} />
                  ))}
                </div>
              ) : null}
            </div>
          );
        })}
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="w-full"
          disabled={items.length >= spec.max}
          onClick={() => {
            setItems([...items, {}]);
            setOpen(items.length);
          }}
        >
          <Plus className="size-3.5" />
          {t("template.action.add_item")}
        </Button>
      </div>
    </FieldShell>
  );
}

function BlockInspector({ selection }: { selection: Extract<Selection, { kind: "block" }> }) {
  const { editor, t, select } = useEditor();
  const ref: ZoneRef = { page: selection.page, zone: selection.zone };
  const block = findBlock(editor.doc, ref, selection.id);
  if (!block) return <p className="text-sm text-muted-foreground">{t("template.inspector.missing")}</p>;
  const meta = blockMeta(block.type);
  const builtin = isBuiltinBlock(block.type);
  const Icon = siteIcon(meta?.icon);
  const update = (fn: (current: SiteBlock) => SiteBlock, key: string) =>
    editor.apply((doc) => updateBlock(doc, ref, block.id, fn), `block:${block.id}:${key}`);

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-3">
        <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
          {Icon ? <Icon className="size-4" /> : null}
        </span>
        <div className="min-w-0">
          <div className="text-sm font-semibold">{meta ? t(meta.titleKey) : block.type}</div>
          {meta ? <p className="text-xs text-muted-foreground">{t(meta.descriptionKey)}</p> : null}
        </div>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => update((current) => ({ ...current, hidden: !current.hidden || undefined }), "hidden")}
        >
          {block.hidden ? <Eye className="size-3.5" /> : <EyeOff className="size-3.5" />}
          {block.hidden ? t("template.action.show") : t("template.action.hide")}
        </Button>
        {builtin ? null : (
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => {
              const copy = cloneBlock(block);
              const index = zoneBlocks(editor.doc, ref).findIndex((item) => item.id === block.id);
              editor.apply((doc) => insertBlock(doc, ref, index + 1, copy));
              select({ kind: "block", ...ref, id: copy.id });
            }}
          >
            <Copy className="size-3.5" />
            {t("template.action.duplicate")}
          </Button>
        )}
        <ConfirmButton
          label={t("template.action.remove")}
          confirmLabel={t("template.action.confirm_remove")}
          onConfirm={() => {
            editor.apply((doc) => removeBlock(doc, ref, block.id));
            select(null);
          }}
        />
      </div>
      {builtin ? (
        <p className="rounded-lg border border-dashed px-3 py-2.5 text-xs leading-relaxed text-muted-foreground">
          {t("template.inspector.builtin_hint")}
        </p>
      ) : (
        <div className="space-y-4">
          {meta?.fields.map((spec) => (
            <SpecField key={spec.name} spec={spec} block={block} path={spec.name} update={update} />
          ))}
          <div className="space-y-4 border-t pt-4">
            <div className="text-xs font-semibold text-muted-foreground">
              {t("template.inspector.appearance")}
            </div>
            {APPEARANCE_FIELDS.map((spec) => (
              <SpecField key={spec.name} spec={spec} block={block} path={spec.name} update={update} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function SectionInspector({ selection }: { selection: Extract<Selection, { kind: "section" }> }) {
  const { editor, t } = useEditor();
  const hidden = editor.doc.pages?.[selection.page]?.hidden?.includes(selection.id) ?? false;
  const name = t(`template.section.${selection.id}`);
  return (
    <div className="space-y-4">
      <div>
        <div className="text-sm font-semibold">{name}</div>
        <p className="text-xs text-muted-foreground">{t("template.inspector.section_hint")}</p>
      </div>
      <BoolField
        label={t("template.inspector.section_visible")}
        checked={!hidden}
        onChange={(visible) => editor.apply((doc) => setSectionHidden(doc, selection.page, selection.id, !visible))}
      />
    </div>
  );
}

function RoleToggles({ roles, onChange }: { roles: string[]; onChange: (next: string[]) => void }) {
  const { t } = useEditor();
  return (
    <FieldShell label={t("template.menu.roles")} hint={t("template.menu.roles_hint")}>
      <div className="flex flex-wrap gap-1.5">
        {STAFF_ROLES.map((role) => {
          const active = roles.includes(role);
          return (
            <button
              key={role}
              type="button"
              onClick={() => onChange(active ? roles.filter((r) => r !== role) : [...roles, role])}
              className={cn(
                "h-7 rounded-md border px-2.5 text-xs transition-colors",
                active ? "border-primary bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground"
              )}
            >
              {t(`layout.role.${role}`)}
            </button>
          );
        })}
      </div>
    </FieldShell>
  );
}

function MenuItemInspector({ selection }: { selection: Extract<Selection, { kind: "item" }> }) {
  const { editor, t, defaultLocale, select } = useEditor();
  const tree = menuTree(editor.doc, selection.menu, defaultLocale);
  const item = findMenuItem(tree, selection.id);
  if (!item) return <p className="text-sm text-muted-foreground">{t("template.inspector.missing")}</p>;
  const base = item.ref ? builtinIndex(selection.menu).get(item.ref)?.item : undefined;
  const kind = item.kind ?? base?.kind ?? "link";
  const commit = (fn: (items: SiteItem[]) => SiteItem[], key?: string) =>
    editor.apply((doc) => setMenuTree(doc, selection.menu, fn(menuTree(doc, selection.menu, defaultLocale))), key);
  const update = (patch: Partial<SiteItem>, key: string) =>
    commit((items) => updateItem(items, item.id, (current) => clean({ ...current, ...patch })), `item:${item.id}:${key}`);

  return (
    <div className="space-y-4">
      <div>
        <div className="text-sm font-semibold">
          {kind === "group" ? t("template.menu.group") : t("template.menu.link")}
          {base ? <span className="ml-2 text-xs font-normal text-muted-foreground">{t("template.menu.builtin")}</span> : null}
        </div>
        <p className="text-xs text-muted-foreground">{t(`template.menu.hint_${selection.menu}`)}</p>
      </div>
      <LTextField
        label={t("template.menu.title")}
        value={item.label}
        placeholder={base ? builtinTitle(base, t) : undefined}
        maxLength={300}
        onChange={(label) => update({ label }, "label")}
      />
      {kind === "link" ? (
        <UrlField
          label={t("template.menu.url")}
          value={item.url ?? ""}
          placeholder={base?.url}
          onChange={(url) => update({ url: url || undefined }, "url")}
        />
      ) : null}
      <IconField label={t("template.menu.icon")} value={item.icon ?? ""} placeholder={base?.icon} onChange={(icon) => update({ icon: icon || undefined }, "icon")} />
      {kind === "link" ? (
        <BoolField label={t("template.menu.new_tab")} checked={Boolean(item.new_tab)} onChange={(new_tab) => update({ new_tab: new_tab || undefined }, "new_tab")} />
      ) : null}
      <LTextField label={t("template.menu.badge")} value={item.badge} maxLength={40} onChange={(badge) => update({ badge }, "badge")} />
      <ChoiceField
        label={t("template.menu.audience")}
        value={item.audience ?? ""}
        options={[
          { value: "", label: t("template.audience.all") },
          { value: "guests", label: t("template.audience.guests") },
          { value: "users", label: t("template.audience.users") },
          { value: "staff", label: t("template.audience.staff") },
        ]}
        onChange={(audience) => update({ audience: (audience || undefined) as SiteItem["audience"], roles: audience === "staff" ? item.roles : undefined }, "audience")}
      />
      {item.audience === "staff" ? (
        <RoleToggles roles={item.roles ?? []} onChange={(roles) => update({ roles: roles.length ? roles : undefined }, "roles")} />
      ) : null}
      <p className="text-[11px] leading-snug text-muted-foreground">{t("template.menu.audience_note")}</p>
      <BoolField label={t("template.menu.visible")} checked={!item.hidden} onChange={(visible) => update({ hidden: visible ? undefined : true }, "hidden")} />
      <div className="flex flex-wrap gap-2 border-t pt-4">
        <Button type="button" size="sm" variant="outline" onClick={() => commit((items) => shiftItem(items, item.id, -1))}>
          <ArrowUp className="size-3.5" />
          {t("template.action.up")}
        </Button>
        <Button type="button" size="sm" variant="outline" onClick={() => commit((items) => shiftItem(items, item.id, 1))}>
          <ArrowDown className="size-3.5" />
          {t("template.action.down")}
        </Button>
        {base ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() =>
              commit((items) =>
                updateItem(items, item.id, (current) => clean({ id: current.id, ref: current.ref, kind: current.kind, items: current.items }))
              )
            }
          >
            <RotateCcw className="size-3.5" />
            {t("template.menu.reset_item")}
          </Button>
        ) : (
          <ConfirmButton
            label={t("template.action.remove")}
            confirmLabel={t("template.action.confirm_remove")}
            onConfirm={() => {
              commit((items) => removeItem(items, item.id));
              select(null);
            }}
          />
        )}
      </div>
    </div>
  );
}

function clean(item: SiteItem): SiteItem {
  const out = { ...item } as Record<string, unknown>;
  for (const key of Object.keys(out)) {
    const value = out[key];
    if (value === undefined || (value && typeof value === "object" && !Array.isArray(value) && Object.keys(value).length === 0)) {
      delete out[key];
    }
  }
  return out as SiteItem;
}

function PageInspector({ selection }: { selection: Extract<Selection, { kind: "page" }> }) {
  const { editor, t, navigate, select } = useEditor();
  const page = editor.doc.custom_pages?.find((item) => item.id === selection.id);
  const [slug, setSlug] = useState(page?.slug ?? "");
  useEffect(() => {
    setSlug(page?.slug ?? "");
  }, [page?.slug]);
  if (!page) return <p className="text-sm text-muted-foreground">{t("template.inspector.missing")}</p>;
  const taken = (value: string) => (editor.doc.custom_pages ?? []).some((item) => item.id !== page.id && item.slug === value);
  const slugError = !SLUG.test(slug) ? t("template.page.slug_invalid") : taken(slug) ? t("template.page.slug_taken") : undefined;
  const update = (patch: Parameters<typeof updateCustomPage>[2], key: string) =>
    editor.apply((doc) => updateCustomPage(doc, page.id, patch), `page:${page.id}:${key}`);

  return (
    <div className="space-y-4">
      <LTextField label={t("template.page.title")} value={page.title} maxLength={300} onChange={(title) => update({ title }, "title")} />
      <FieldShell label={t("template.page.slug")} error={slugError} hint={`${window.location.origin}${customPagePath(slug || "…")}`}>
        <div className="flex items-center rounded-md border bg-background">
          <span className="pl-2.5 font-mono text-xs text-muted-foreground">/p/</span>
          <Input
            value={slug}
            onChange={(event) => {
              const next = event.target.value.toLowerCase();
              setSlug(next);
              if (SLUG.test(next) && !taken(next)) update({ slug: next }, "slug");
            }}
            className="h-8 border-0 pl-0.5 font-mono text-xs shadow-none focus-visible:ring-0"
          />
        </div>
      </FieldShell>
      <LTextField
        label={t("template.page.description")}
        value={page.description}
        multiline
        maxLength={300}
        onChange={(description) => update({ description }, "description")}
      />
      <ChoiceField
        label={t("template.page.layout")}
        value={page.layout ?? "site"}
        options={[
          { value: "site", label: t("template.page.layout_site") },
          { value: "panel", label: t("template.page.layout_panel") },
        ]}
        onChange={(layout) => update({ layout: layout as "site" | "panel" }, "layout")}
      />
      {page.layout !== "panel" ? (
        <ChoiceField
          label={t("template.page.audience")}
          value={page.audience ?? ""}
          options={[
            { value: "", label: t("template.audience.all") },
            { value: "guests", label: t("template.audience.guests") },
            { value: "users", label: t("template.audience.users") },
          ]}
          onChange={(audience) => update({ audience: audience as "" | "guests" | "users" }, "audience")}
        />
      ) : null}
      <BoolField
        label={t("template.page.published")}
        hint={t("template.page.published_hint")}
        checked={!page.hidden}
        onChange={(visible) => update({ hidden: visible ? undefined : true }, "hidden")}
      />
      <div className="flex flex-wrap gap-2 border-t pt-4">
        <Button type="button" size="sm" variant="outline" onClick={() => navigate(customPagePath(page.slug))}>
          <ExternalLink className="size-3.5" />
          {t("template.page.open")}
        </Button>
        <ConfirmButton
          label={t("template.page.remove")}
          confirmLabel={t("template.action.confirm_remove")}
          onConfirm={() => {
            editor.apply((doc) => removeCustomPage(doc, page.id));
            select(null);
            navigate("/");
          }}
        />
      </div>
    </div>
  );
}

function titleOf(selection: NonNullable<Selection>, t: (key: string) => string): string {
  switch (selection.kind) {
    case "block":
      return t("template.inspector.block");
    case "section":
      return t("template.inspector.section");
    case "item":
      return t("template.inspector.item");
    case "page":
      return t("template.inspector.page");
  }
}

export function Inspector() {
  const { selection, select, t } = useEditor();
  if (!selection) return null;
  let content: ReactNode;
  switch (selection.kind) {
    case "block":
      content = <BlockInspector key={`${selection.page}:${selection.zone}:${selection.id}`} selection={selection} />;
      break;
    case "section":
      content = <SectionInspector selection={selection} />;
      break;
    case "item":
      content = <MenuItemInspector key={`${selection.menu}:${selection.id}`} selection={selection} />;
      break;
    case "page":
      content = <PageInspector key={selection.id} selection={selection} />;
      break;
  }
  return (
    <aside className="absolute inset-y-0 right-0 z-20 flex w-[340px] shrink-0 flex-col border-l bg-background shadow-xl xl:static xl:shadow-none">
      <div className="flex h-11 shrink-0 items-center justify-between border-b pr-2 pl-4">
        <span className="text-sm font-medium">{titleOf(selection, t)}</span>
        <Button type="button" variant="ghost" size="icon" className="size-8" aria-label={t("common.close")} onClick={() => select(null)}>
          <X className="size-4" />
        </Button>
      </div>
      <ScrollArea className="min-h-0 flex-1">
        <div className="p-4">{content}</div>
      </ScrollArea>
    </aside>
  );
}

