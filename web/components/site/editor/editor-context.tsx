"use client";

import { createContext, useContext } from "react";
import type { BaseLocale, LanguageInfo, TranslateFn } from "@/lib/i18n";
import type { MenuName, SiteZone } from "@/lib/site/types";
import type { TemplateEditor } from "@/components/site/editor/use-template-editor";

export type Selection =
  | { kind: "block"; page: string; zone: SiteZone; id: string }
  | { kind: "section"; page: string; id: string }
  | { kind: "item"; menu: MenuName; id: string }
  | { kind: "page"; id: string }
  | null;

export type LibraryTarget = { page: string; zone: SiteZone; index: number };

export type Inventory = {
  path: string;
  keys: string[];
  sections: { id: string; hidden: boolean }[];
};

export type PageLink = { label: string; url: string };

export type EditorContextValue = {
  editor: TemplateEditor;
  t: TranslateFn;
  tr: TranslateFn;
  locale: string;
  defaultLocale: string;
  languages: LanguageInfo[];
  base: BaseLocale;
  messages: Record<string, string>;
  selection: Selection;
  select: (next: Selection, reveal?: boolean) => void;
  setPhrase: (key: string, value: string) => void;
  phraseValue: (key: string) => string;
  path: string;
  pageKey: string;
  navigate: (path: string) => void;
  inventory: Inventory | null;
  openLibrary: (target: LibraryTarget) => void;
  pageLinks: PageLink[];
};

const EditorContext = createContext<EditorContextValue | null>(null);

export const EditorProvider = EditorContext.Provider;

export function useEditor(): EditorContextValue {
  const value = useContext(EditorContext);
  if (!value) throw new Error("editor context is missing");
  return value;
}
