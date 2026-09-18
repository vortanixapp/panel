"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, FileText, LayoutGrid, Loader2, Menu, Type } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { BlockLibrary } from "@/components/site/editor/block-library";
import { BlocksPanel } from "@/components/site/editor/blocks-panel";
import { PublishDialog, VersionsSheet } from "@/components/site/editor/editor-dialogs";
import {
  EditorProvider,
  type EditorContextValue,
  type Inventory,
  type LibraryTarget,
  type PageLink,
  type Selection,
} from "@/components/site/editor/editor-context";
import { EditorTopbar, type ViewerMode } from "@/components/site/editor/editor-topbar";
import { Inspector } from "@/components/site/editor/inspector";
import { MenusPanel } from "@/components/site/editor/menus-panel";
import { PagesPanel } from "@/components/site/editor/pages-panel";
import { PreviewFrame, type Device, type PreviewHandle } from "@/components/site/editor/preview-frame";
import { TextsPanel } from "@/components/site/editor/texts-panel";
import { useTemplateEditor } from "@/components/site/editor/use-template-editor";
import { useLocale } from "@/context/locale-provider";
import { useT } from "@/hooks/use-translations";
import { fetchI18n } from "@/lib/api";
import { catalogText, parseI18nPayload, translator, type BaseLocale } from "@/lib/i18n";
import { cloneBlock, isBuiltinBlock } from "@/lib/site/blocks";
import type { FrameMessage, ParentMessage } from "@/lib/site/bridge";
import {
  findBlock,
  insertBlock,
  moveBlock,
  removeBlock,
  setBlockText,
  setDraftText,
  setSectionHidden,
  updateBlock,
  zoneBlocks,
} from "@/lib/site/doc";
import { REGISTRY_PAGES, customPagePath, customSlugOf, pageKeyOf, pageVariantOf } from "@/lib/site/pages";
import type { EditorMode } from "@/lib/site/store";
import { localized } from "@/lib/site/text";
import type { MenuName, Viewer } from "@/lib/site/types";
import { cn } from "@/lib/utils";

type Tab = "pages" | "blocks" | "menus" | "texts";

const ANCHORS = ["#pricing", "#games", "#hardware", "#faq"];

function editableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  return target.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName);
}

export function TemplateEditor() {
  const t = useT();
  const { languages, defaultLocale } = useLocale();
  const editor = useTemplateEditor();
  const [locale, setLocale] = useState(defaultLocale);
  const [tab, setTab] = useState<Tab>("blocks");
  const [menu, setMenu] = useState<MenuName>("site_header");
  const [device, setDevice] = useState<Device>("desktop");
  const [mode, setMode] = useState<EditorMode>("edit");
  const [viewerMode, setViewerMode] = useState<ViewerMode>("auto");
  const [selection, setSelection] = useState<Selection>(null);
  const [path, setPath] = useState("/");
  const [inventory, setInventory] = useState<Inventory | null>(null);
  const [library, setLibrary] = useState<LibraryTarget | null>(null);
  const [versionsOpen, setVersionsOpen] = useState(false);
  const [publishOpen, setPublishOpen] = useState(false);
  const previewRef = useRef<PreviewHandle | null>(null);

  const i18n = useQuery({
    queryKey: ["template-i18n", locale],
    queryFn: async () => parseI18nPayload(await fetchI18n(locale), locale),
    staleTime: 60_000,
  });
  const base: BaseLocale = i18n.data?.base ?? languages.find((lang) => lang.code === locale)?.base ?? "ru";
  const messages = useMemo(() => i18n.data?.messages ?? {}, [i18n.data]);
  const tr = useMemo(() => translator({ base, messages: {} }), [base]);

  const customPages = editor.doc.custom_pages;
  const pageKey = useMemo(() => {
    const slug = customSlugOf(path);
    if (slug) {
      const page = customPages?.find((item) => item.slug === slug);
      if (page) return `p:${page.id}`;
    }
    return pageKeyOf(path);
  }, [path, customPages]);

  const viewer = useMemo<Viewer | null>(
    () =>
      viewerMode === "guest" ? { loggedIn: false, role: "" } : viewerMode === "client" ? { loggedIn: true, role: "user" } : null,
    [viewerMode]
  );
  const selected = selection?.kind === "block" ? selection.id : selection?.kind === "section" ? `section:${selection.id}` : null;

  const state = useMemo<ParentMessage>(
    () => ({ type: "vx:state", document: editor.doc, mode, viewer, selected, locale }),
    [editor.doc, mode, viewer, selected, locale]
  );

  const navigate = useCallback((next: string) => {
    previewRef.current?.navigate(next);
  }, []);

  const select = useCallback((next: Selection, reveal?: boolean) => {
    setSelection(next);
    if (reveal && next?.kind === "block") previewRef.current?.post({ type: "vx:scroll", block: next.id });
  }, []);

  const draftTexts = editor.doc.texts?.[locale];

  const phraseValue = useCallback(
    (key: string) => {
      const draft = draftTexts?.[key];
      if (draft !== undefined) return draft.trim() ? draft : (catalogText(base, key) ?? "");
      return messages[key] ?? catalogText(base, key) ?? "";
    },
    [draftTexts, messages, base]
  );

  const setPhrase = useCallback(
    (key: string, value: string) => {
      const published = messages[key];
      const catalog = catalogText(base, key) ?? "";
      let next: string | null;
      if (!value.trim() || value === catalog) next = published ? "" : null;
      else if (value === published) next = null;
      else next = value;
      editor.apply((doc) => setDraftText(doc, locale, key, next), `text:${locale}:${key}`);
    },
    [messages, base, locale, editor]
  );

  const blockAction = useCallback(
    (message: Extract<FrameMessage, { type: "vx:block-action" }>) => {
      const ref = { page: message.page, zone: message.zone };
      switch (message.action) {
        case "up":
        case "down":
          editor.apply((doc) => moveBlock(doc, ref, message.block, message.action === "up" ? -1 : 1));
          return;
        case "hide":
        case "show":
          editor.apply((doc) =>
            updateBlock(doc, ref, message.block, (block) => ({ ...block, hidden: message.action === "hide" || undefined }))
          );
          return;
        case "duplicate": {
          const block = findBlock(editor.doc, ref, message.block);
          if (!block || isBuiltinBlock(block.type)) return;
          const copy = cloneBlock(block);
          const index = zoneBlocks(editor.doc, ref).findIndex((item) => item.id === block.id);
          editor.apply((doc) => insertBlock(doc, ref, index + 1, copy));
          setSelection({ kind: "block", ...ref, id: copy.id });
          return;
        }
        case "remove":
          editor.apply((doc) => removeBlock(doc, ref, message.block));
          setSelection(null);
          toast(t("template.toast.block_removed"), {
            action: { label: t("template.toast.undo"), onClick: () => editor.undo() },
          });
      }
    },
    [editor, t]
  );

  const onMessage = useCallback(
    (message: FrameMessage) => {
      switch (message.type) {
        case "vx:ready":
        case "vx:path":
          setPath(message.path);
          return;
        case "vx:select":
          setSelection({ kind: "block", page: message.page, zone: message.zone, id: message.block });
          return;
        case "vx:deselect":
          setSelection(null);
          return;
        case "vx:select-section":
          setSelection({ kind: "section", page: message.page, id: message.section });
          return;
        case "vx:insert":
          setLibrary({ page: message.page, zone: message.zone, index: message.index });
          return;
        case "vx:block-action":
          blockAction(message);
          return;
        case "vx:text":
          if (message.locale === locale) setPhrase(message.key, message.value);
          return;
        case "vx:field":
          editor.apply(
            (doc) =>
              updateBlock(doc, { page: message.page, zone: message.zone }, message.block, (block) =>
                setBlockText(block, message.path, message.locale, message.value)
              ),
            `block:${message.block}:${message.path}`
          );
          return;
        case "vx:section":
          editor.apply((doc) => setSectionHidden(doc, message.page, message.section, message.hidden));
          return;
        case "vx:inventory":
          setInventory({ path: message.path, keys: message.keys, sections: message.sections });
          return;
        case "vx:history":
          if (message.action === "undo") editor.undo();
          else if (message.action === "redo") editor.redo();
          else void editor.saveNow();
      }
    },
    [blockAction, editor, locale, setPhrase]
  );

  useEffect(() => {
    if (!selection || (selection.kind !== "block" && selection.kind !== "section")) return;
    if (selection.page !== pageKey) setSelection(null);
  }, [pageKey, selection]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const mod = event.ctrlKey || event.metaKey;
      if (!mod) return;
      const key = event.key.toLowerCase();
      if (key === "s") {
        event.preventDefault();
        void editor.saveNow();
        return;
      }
      if (editableTarget(event.target)) return;
      if (key === "z") {
        event.preventDefault();
        if (event.shiftKey) editor.redo();
        else editor.undo();
      } else if (key === "y") {
        event.preventDefault();
        editor.redo();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [editor]);

  const pageLinks = useMemo<PageLink[]>(() => {
    const links: PageLink[] = REGISTRY_PAGES.map((page) => ({ label: t(page.titleKey), url: page.path }));
    for (const page of customPages ?? []) {
      links.push({ label: localized(page.title, locale, defaultLocale) || page.slug, url: customPagePath(page.slug) });
    }
    for (const anchor of ANCHORS) links.push({ label: anchor, url: anchor });
    return links;
  }, [customPages, locale, defaultLocale, t]);

  const context = useMemo<EditorContextValue>(
    () => ({
      editor,
      t,
      tr,
      locale,
      defaultLocale,
      languages,
      base,
      messages,
      selection,
      select,
      setPhrase,
      phraseValue,
      path,
      pageKey,
      navigate,
      inventory,
      openLibrary: setLibrary,
      pageLinks,
    }),
    [editor, t, tr, locale, defaultLocale, languages, base, messages, selection, select, setPhrase, phraseValue, path, pageKey, navigate, inventory, pageLinks]
  );

  if (!editor.loaded) {
    return (
      <div className="fixed inset-0 z-40 flex items-center justify-center bg-background">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (editor.loadError) {
    return (
      <div className="fixed inset-0 z-40 flex flex-col items-center justify-center gap-3 bg-background px-6 text-center">
        <AlertTriangle className="size-8 text-destructive" />
        <p className="max-w-md text-sm text-muted-foreground">{editor.loadError}</p>
        <Button type="button" variant="outline" onClick={() => void editor.reload()}>
          {t("template.retry")}
        </Button>
      </div>
    );
  }

  const tabs: { id: Tab; label: string; icon: typeof FileText }[] = [
    { id: "pages", label: t("template.tab.pages"), icon: FileText },
    { id: "blocks", label: t("template.tab.blocks"), icon: LayoutGrid },
    { id: "menus", label: t("template.tab.menus"), icon: Menu },
    { id: "texts", label: t("template.tab.texts"), icon: Type },
  ];

  return (
    <EditorProvider value={context}>
      <div className="fixed inset-0 z-40 flex flex-col bg-background text-foreground">
        <EditorTopbar
          device={device}
          onDevice={setDevice}
          mode={mode}
          onMode={setMode}
          viewerMode={viewerMode}
          onViewerMode={setViewerMode}
          showViewer={pageVariantOf(path) === "site"}
          locale={locale}
          onLocale={setLocale}
          onVersions={() => setVersionsOpen(true)}
          onPublish={() => setPublishOpen(true)}
        />
        {editor.conflict ? (
          <div className="flex flex-wrap items-center gap-3 border-b border-amber-500/30 bg-amber-500/10 px-4 py-2 text-xs">
            <AlertTriangle className="size-4 text-amber-600 dark:text-amber-500" />
            <span className="flex-1">
              {t("template.conflict.text", { name: editor.conflict.draft_updated_by || t("template.conflict.someone") })}
            </span>
            <Button type="button" size="sm" variant="outline" onClick={() => void editor.resolveConflict("theirs")}>
              {t("template.conflict.theirs")}
            </Button>
            <Button type="button" size="sm" onClick={() => void editor.resolveConflict("mine")}>
              {t("template.conflict.mine")}
            </Button>
          </div>
        ) : null}
        <div className="relative flex min-h-0 flex-1">
          <aside className="flex w-[300px] shrink-0 flex-col border-r bg-background">
            <div className="grid shrink-0 grid-cols-4 gap-0.5 border-b p-1.5">
              {tabs.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => setTab(item.id)}
                  className={cn(
                    "flex flex-col items-center gap-1 rounded-md py-1.5 text-[11px] transition-colors",
                    tab === item.id ? "bg-accent font-medium text-foreground" : "text-muted-foreground hover:bg-accent/60 hover:text-foreground"
                  )}
                >
                  <item.icon className="size-4" />
                  {item.label}
                </button>
              ))}
            </div>
            <ScrollArea className="min-h-0 flex-1">
              <div className="p-3">
                {tab === "pages" ? <PagesPanel /> : null}
                {tab === "blocks" ? <BlocksPanel /> : null}
                {tab === "menus" ? <MenusPanel menu={menu} onMenuChange={setMenu} /> : null}
                {tab === "texts" ? <TextsPanel /> : null}
              </div>
            </ScrollArea>
          </aside>
          <PreviewFrame
            initialPath="/"
            device={device}
            state={state}
            onMessage={onMessage}
            onReady={() => setInventory(null)}
            handleRef={previewRef}
          />
          <Inspector />
        </div>
        <BlockLibrary target={library} onClose={() => setLibrary(null)} />
        <VersionsSheet open={versionsOpen} onClose={() => setVersionsOpen(false)} />
        <PublishDialog open={publishOpen} onClose={() => setPublishOpen(false)} />
      </div>
    </EditorProvider>
  );
}
