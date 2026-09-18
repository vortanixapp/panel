"use client";

import { useMemo, useState } from "react";
import { RotateCcw, Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useEditor } from "@/components/site/editor/editor-context";
import { CATALOG_KEYS } from "@/lib/i18n";
import { cn } from "@/lib/utils";

const LIMIT = 80;

function hiddenKey(key: string): boolean {
  return key.startsWith("mail.") || key.startsWith("notify.") || key.startsWith("template.");
}

export function TextsPanel() {
  const { editor, t, locale, messages, inventory, path, phraseValue, setPhrase } = useEditor();
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState<string | null>(null);
  const draft = editor.doc.texts?.[locale] ?? {};
  const pageKeys = useMemo(
    () => (inventory && inventory.path === path ? inventory.keys.filter((key) => !hiddenKey(key)) : []),
    [inventory, path]
  );
  const needle = query.trim().toLowerCase();

  const keys = useMemo(() => {
    if (!needle) return pageKeys;
    const all = new Set([...CATALOG_KEYS, ...Object.keys(messages)]);
    const out: string[] = [];
    for (const key of all) {
      if (hiddenKey(key)) continue;
      if (key.toLowerCase().includes(needle) || phraseValue(key).toLowerCase().includes(needle)) {
        out.push(key);
        if (out.length >= LIMIT) break;
      }
    }
    return out;
  }, [needle, pageKeys, messages, phraseValue]);

  return (
    <div className="space-y-3">
      <div className="relative">
        <Search className="absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t("template.texts.search")}
          className="h-8 pl-8 text-xs"
        />
      </div>
      <p className="text-[11px] leading-snug text-muted-foreground">
        {needle ? t("template.texts.search_hint", { limit: LIMIT }) : t("template.texts.page_hint")}
      </p>
      {keys.length === 0 ? (
        <p className="rounded-lg border border-dashed px-3 py-6 text-center text-xs text-muted-foreground">
          {needle ? t("template.texts.nothing") : t("template.texts.empty_page")}
        </p>
      ) : null}
      <div className="space-y-1">
        {keys.map((key) => {
          const value = phraseValue(key);
          const changed = draft[key] !== undefined;
          const custom = !changed && Boolean(messages[key]);
          const expanded = open === key;
          return (
            <div key={key} className={cn("rounded-lg border", expanded ? "border-primary/50" : "hover:border-foreground/20")}>
              <button
                type="button"
                onClick={() => setOpen(expanded ? null : key)}
                className="flex w-full flex-col gap-0.5 px-2.5 py-2 text-left"
              >
                <span className="flex items-center gap-1.5">
                  <span
                    className={cn(
                      "size-1.5 shrink-0 rounded-full",
                      changed ? "bg-amber-500" : custom ? "bg-sky-500" : "bg-muted-foreground/30"
                    )}
                  />
                  <span className="truncate font-mono text-[10.5px] text-muted-foreground">{key}</span>
                </span>
                <span className="line-clamp-2 text-xs">{value || <span className="text-muted-foreground">—</span>}</span>
              </button>
              {expanded ? (
                <div className="space-y-2 border-t px-2.5 py-2.5">
                  <Textarea
                    autoFocus
                    value={value}
                    rows={3}
                    onChange={(event) => setPhrase(key, event.target.value)}
                    className="min-h-16 text-xs"
                  />
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-[11px] text-muted-foreground">
                      {changed ? t("template.texts.changed") : custom ? t("template.texts.custom") : t("template.texts.default")}
                    </span>
                    <button
                      type="button"
                      onClick={() => setPhrase(key, "")}
                      className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] text-muted-foreground hover:bg-accent hover:text-foreground"
                    >
                      <RotateCcw className="size-3" />
                      {t("template.texts.reset")}
                    </button>
                  </div>
                </div>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
}
