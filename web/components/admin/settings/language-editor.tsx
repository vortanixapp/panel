"use client";

import {
  memo,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
} from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangle,
  ChevronLeft,
  ChevronRight,
  CopyPlus,
  Download,
  RotateCcw,
  Search,
  Upload,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { useT } from "@/hooks/use-translations";
import {
  fetchAdminLanguageMessages,
  saveAdminLanguageMessages,
  type AdminLanguage,
  type ServerPhraseCatalog,
} from "@/lib/api";
import {
  builtinLanguageName,
  CATALOG_KEYS,
  catalogText,
  type BaseLocale,
  type TranslateFn,
} from "@/lib/i18n";
import { cn } from "@/lib/utils";

const PAGE_SIZE = 40;
const PLACEHOLDER = /\{(\w+)\}/g;
const SECTION_ORDER = [
  "common",
  "nav",
  "layout",
  "auth",
  "landing",
  "dashboard",
  "servers",
  "server",
  "monitoring",
  "billing",
  "support",
  "news",
  "settings",
  "errors",
  "mail",
  "notify",
];

type Filter = "all" | "translated" | "empty" | "issues" | "unsaved";

export type PhraseCatalog = {
  keys: string[];
  text: (base: BaseLocale, key: string) => string | undefined;
};

export function buildPhraseCatalog(server?: ServerPhraseCatalog): PhraseCatalog {
  const serverKeys = [
    ...Object.keys(server?.ru ?? {}),
    ...Object.keys(server?.en ?? {}),
  ];
  const keys = Array.from(new Set([...CATALOG_KEYS, ...serverKeys])).sort();
  return {
    keys,
    text: (base, key) =>
      catalogText(base, key) ?? server?.[base]?.[key] ?? server?.ru?.[key],
  };
}

function filled(value: string | undefined): string {
  return value !== undefined && value.trim() !== "" ? value : "";
}

function sectionOf(key: string): string {
  const [first, second] = key.split(".");
  return first === "admin" && second ? `admin.${second}` : first;
}

function placeholders(text: string): string[] {
  return Array.from(
    new Set(Array.from(text.matchAll(PLACEHOLDER), (match) => match[1]))
  );
}

function missingPlaceholders(source: string, value: string): string[] {
  if (!value) return [];
  const present = new Set(placeholders(value));
  return placeholders(source).filter((name) => !present.has(name));
}

function flatten(
  input: Record<string, unknown>,
  prefix = "",
  out: Record<string, string> = {}
): Record<string, string> {
  for (const [key, value] of Object.entries(input)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (typeof value === "string") {
      out[path] = value;
    } else if (value && typeof value === "object" && !Array.isArray(value)) {
      flatten(value as Record<string, unknown>, path, out);
    }
  }
  return out;
}

function buildSections(keys: string[]) {
  const counts = new Map<string, number>();
  for (const key of keys) {
    const id = sectionOf(key);
    counts.set(id, (counts.get(id) ?? 0) + 1);
  }
  const rank = (id: string) => {
    if (id.startsWith("admin.")) return SECTION_ORDER.length + 1;
    const index = SECTION_ORDER.indexOf(id);
    return index === -1 ? SECTION_ORDER.length : index;
  };
  return Array.from(counts, ([id, count]) => ({ id, count })).sort(
    (a, b) => rank(a.id) - rank(b.id) || a.id.localeCompare(b.id)
  );
}

function sectionLabel(t: TranslateFn, id: string): string {
  if (id.startsWith("admin.")) {
    return `${t("admin.language.section.admin")} · ${id.slice("admin.".length)}`;
  }
  const key = `admin.language.section.${id}`;
  const label = t(key);
  return label === key ? id : label;
}

export function LanguageEditor({
  language,
  phrases,
  onDirtyChange,
  onSaved,
}: {
  language: AdminLanguage;
  phrases: PhraseCatalog;
  onDirtyChange: (dirty: boolean) => void;
  onSaved: () => void;
}) {
  const t = useT();
  const qc = useQueryClient();
  const fileRef = useRef<HTMLInputElement>(null);
  const topRef = useRef<HTMLElement>(null);
  const [draft, setDraft] = useState<Record<string, string>>({});
  const [view, setView] = useState<Record<string, string>>({});
  const [search, setSearch] = useState("");
  const [section, setSection] = useState("all");
  const [filter, setFilter] = useState<Filter>("all");
  const [page, setPage] = useState(0);

  const sections = useMemo(() => buildSections(phrases.keys), [phrases.keys]);

  const queryKey = useMemo(
    () => ["admin-language-messages", language.code] as const,
    [language.code]
  );
  const query = useQuery({
    queryKey,
    queryFn: () => fetchAdminLanguageMessages(language.code),
    staleTime: 0,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
  const saved = query.data?.messages;

  useEffect(() => {
    if (!saved) return;
    setDraft(saved);
    setView(saved);
  }, [saved]);

  const changes = useMemo(() => {
    const out: Record<string, string> = {};
    if (!saved) return out;
    for (const key of phrases.keys) {
      const next = filled(draft[key]);
      if (next !== filled(saved[key])) out[key] = next;
    }
    return out;
  }, [draft, saved, phrases.keys]);
  const changeCount = Object.keys(changes).length;

  useEffect(() => {
    onDirtyChange(changeCount > 0);
  }, [changeCount, onDirtyChange]);

  useEffect(() => () => onDirtyChange(false), [onDirtyChange]);

  useEffect(() => {
    if (changeCount === 0) return;
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [changeCount]);

  const translatedCount = useMemo(
    () => phrases.keys.reduce((n, key) => (filled(draft[key]) ? n + 1 : n), 0),
    [draft, phrases.keys]
  );

  const rows = useMemo(() => {
    const q = search.trim().toLowerCase();
    return phrases.keys.filter((key) => {
      if (section !== "all" && sectionOf(key) !== section) return false;
      const value = filled(view[key]);
      if (filter === "translated" && !value) return false;
      if (filter === "empty" && value) return false;
      if (filter === "unsaved" && value === filled(saved?.[key])) return false;
      const source = phrases.text(language.base, key) ?? "";
      if (filter === "issues" && missingPlaceholders(source, value).length === 0) {
        return false;
      }
      if (!q) return true;
      return (
        key.toLowerCase().includes(q) ||
        source.toLowerCase().includes(q) ||
        value.toLowerCase().includes(q)
      );
    });
  }, [view, saved, search, section, filter, language.base, phrases]);

  const pageCount = Math.max(1, Math.ceil(rows.length / PAGE_SIZE));
  const currentPage = Math.min(page, pageCount - 1);
  const visible = rows.slice(
    currentPage * PAGE_SIZE,
    (currentPage + 1) * PAGE_SIZE
  );

  const setValue = useCallback((key: string, value: string) => {
    setDraft((prev) => ({ ...prev, [key]: value }));
  }, []);

  function replaceDraft(next: Record<string, string>) {
    setDraft(next);
    setView(next);
  }

  function changeSearch(value: string) {
    setSearch(value);
    setView(draft);
    setPage(0);
  }

  function changeSection(value: string) {
    setSection(value);
    setView(draft);
    setPage(0);
  }

  function changeFilter(value: Filter) {
    setFilter(value);
    setView(draft);
    setPage(0);
  }

  function goToPage(next: number) {
    setPage(next);
    setView(draft);
    topRef.current?.scrollIntoView({ block: "start", behavior: "smooth" });
  }

  const save = useMutation({
    mutationFn: () => saveAdminLanguageMessages(language.code, changes),
    onSuccess: (data) => {
      qc.setQueryData(queryKey, data);
      onSaved();
      toast.success(t("admin.language.saved"));
    },
    onError: (err) =>
      toast.error(
        err instanceof Error && err.message ? err.message : t("common.save_failed")
      ),
  });

  async function importFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    let parsed: unknown = null;
    try {
      parsed = JSON.parse(await file.text());
    } catch {
      parsed = null;
    }
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      toast.error(t("admin.language.import_invalid"));
      return;
    }
    const known = new Set(phrases.keys);
    const next = { ...draft };
    let applied = 0;
    let skipped = 0;
    for (const [key, value] of Object.entries(
      flatten(parsed as Record<string, unknown>)
    )) {
      if (!known.has(key)) {
        skipped += 1;
        continue;
      }
      next[key] = value === phrases.text(language.base, key) ? "" : value;
      applied += 1;
    }
    replaceDraft(next);
    if (applied > 0) {
      setFilter("unsaved");
      setPage(0);
    }
    toast.success(t("admin.language.imported", { count: applied, skipped }));
  }

  function exportFile() {
    const out: Record<string, string> = {};
    for (const key of phrases.keys) {
      out[key] = filled(draft[key]) || (phrases.text(language.base, key) ?? "");
    }
    const blob = new Blob([`${JSON.stringify(out, null, 2)}\n`], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `vortanix-${language.code}.json`;
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

  function clearAll() {
    if (!window.confirm(t("admin.language.clear_all_confirm"))) return;
    replaceDraft({});
  }

  const baseName = builtinLanguageName(language.base);
  const total = phrases.keys.length;
  const filters: Filter[] = language.builtin
    ? ["all", "translated", "issues", "unsaved"]
    : ["all", "empty", "translated", "issues", "unsaved"];
  const filterLabel = (item: Filter) =>
    language.builtin && item === "translated"
      ? t("admin.language.filter.changed")
      : t(`admin.language.filter.${item}`);

  return (
    <>
      <section
        ref={topRef}
        className="scroll-mt-4 overflow-hidden rounded-2xl border bg-card"
      >
        <div className="flex flex-wrap items-start justify-between gap-4 px-5 py-4">
          <div className="min-w-0 space-y-1">
            <h2 className="text-[15px] font-semibold">
              {t("admin.language.editor_title", { name: language.name })}
            </h2>
            <p className="text-xs text-muted-foreground">
              {language.builtin
                ? t("admin.language.editor_hint_builtin")
                : t("admin.language.editor_hint_custom", { name: baseName })}
            </p>
            <p className="font-mono text-xs text-muted-foreground">
              {language.builtin
                ? t("admin.language.changed_count", { count: translatedCount })
                : t("admin.language.translated_count", {
                    count: translatedCount,
                    total,
                  })}
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            <input
              ref={fileRef}
              type="file"
              accept="application/json,.json"
              className="hidden"
              onChange={importFile}
            />
            <Button
              variant="outline"
              size="sm"
              disabled={!saved || save.isPending}
              onClick={() => fileRef.current?.click()}
            >
              <Upload className="size-4" />
              {t("admin.language.import")}
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={!saved}
              onClick={exportFile}
            >
              <Download className="size-4" />
              {t("admin.language.export")}
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={!saved || translatedCount === 0 || save.isPending}
              onClick={clearAll}
            >
              {t("admin.language.clear_all")}
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2 border-t px-5 py-3">
          <div className="relative min-w-[220px] flex-1">
            <Search className="pointer-events-none absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => changeSearch(e.target.value)}
              placeholder={t("admin.language.search_placeholder")}
              className="h-9 ps-9 text-[13px] md:text-[13px]"
            />
          </div>
          <Select value={section} onValueChange={changeSection}>
            <SelectTrigger className="h-9 w-full text-[13px] sm:w-[240px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="max-h-80">
              <SelectItem value="all">
                {t("admin.language.section_all", { count: total })}
              </SelectItem>
              {sections.map((item) => (
                <SelectItem key={item.id} value={item.id}>
                  {sectionLabel(t, item.id)} · {item.count}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={filter}
            onValueChange={(value) => changeFilter(value as Filter)}
          >
            <SelectTrigger className="h-9 w-full text-[13px] sm:w-[210px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {filters.map((item) => (
                <SelectItem key={item} value={item}>
                  {filterLabel(item)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {query.isLoading ? (
          <div className="space-y-2 border-t p-5">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : !saved ? (
          <div className="border-t py-12 text-center text-[13px] text-muted-foreground">
            {query.error instanceof Error && query.error.message
              ? query.error.message
              : t("common.load_failed")}
          </div>
        ) : visible.length === 0 ? (
          <div className="border-t py-12 text-center text-[13px] text-muted-foreground">
            {t("common.not_found")}
          </div>
        ) : (
          <div>
            <div className="hidden gap-4 border-t bg-muted/40 px-5 py-2.5 font-mono text-[11px] tracking-wider text-muted-foreground uppercase lg:grid lg:grid-cols-[minmax(0,1fr)_minmax(0,1.25fr)_72px]">
              <div>
                {language.builtin
                  ? t("admin.language.col_default")
                  : t("admin.language.col_source", { name: baseName })}
              </div>
              <div>
                {language.builtin
                  ? t("admin.language.col_custom")
                  : t("admin.language.col_translation")}
              </div>
              <div />
            </div>
            {visible.map((key) => (
              <PhraseRow
                key={key}
                phraseKey={key}
                source={phrases.text(language.base, key) ?? ""}
                value={draft[key] ?? ""}
                unsaved={filled(draft[key]) !== filled(saved[key])}
                readOnly={save.isPending}
                onChange={setValue}
              />
            ))}
          </div>
        )}

        {rows.length > PAGE_SIZE && (
          <div className="flex items-center justify-between gap-3 border-t px-5 py-3">
            <span className="font-mono text-xs text-muted-foreground">
              {t("admin.language.range", {
                from: currentPage * PAGE_SIZE + 1,
                to: Math.min(rows.length, (currentPage + 1) * PAGE_SIZE),
                total: rows.length,
              })}
            </span>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="icon"
                className="size-8"
                disabled={currentPage === 0}
                aria-label={t("common.back")}
                onClick={() => goToPage(currentPage - 1)}
              >
                <ChevronLeft className="size-4" />
              </Button>
              <span className="font-mono text-xs text-muted-foreground">
                {currentPage + 1} / {pageCount}
              </span>
              <Button
                variant="outline"
                size="icon"
                className="size-8"
                disabled={currentPage >= pageCount - 1}
                aria-label={t("common.next")}
                onClick={() => goToPage(currentPage + 1)}
              >
                <ChevronRight className="size-4" />
              </Button>
            </div>
          </div>
        )}
      </section>

      {(changeCount > 0 || save.isPending) && (
        <div className="sticky bottom-4 z-10 flex flex-wrap items-center justify-end gap-3 rounded-xl border bg-card/95 px-4 py-3 shadow-sm backdrop-blur">
          <span className="me-auto text-sm text-muted-foreground">
            {t("admin.language.unsaved", { count: changeCount })}
          </span>
          <Button
            variant="outline"
            disabled={save.isPending}
            onClick={() => replaceDraft(saved ?? {})}
          >
            {t("admin.language.discard")}
          </Button>
          <Button disabled={save.isPending} onClick={() => save.mutate()}>
            {save.isPending ? t("common.saving") : t("common.save")}
          </Button>
        </div>
      )}
    </>
  );
}

const PhraseRow = memo(function PhraseRow({
  phraseKey,
  source,
  value,
  unsaved,
  readOnly,
  onChange,
}: {
  phraseKey: string;
  source: string;
  value: string;
  unsaved: boolean;
  readOnly: boolean;
  onChange: (key: string, value: string) => void;
}) {
  const t = useT();
  const missing = missingPlaceholders(source, filled(value));

  return (
    <div
      className={cn(
        "grid gap-2 border-t px-5 py-3 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.25fr)_72px] lg:gap-4",
        unsaved && "bg-amber-500/5"
      )}
    >
      <div className="min-w-0 space-y-1">
        <div className="text-[13px] leading-relaxed break-words whitespace-pre-wrap">
          {source || "—"}
        </div>
        <div
          className="truncate font-mono text-[11px] text-muted-foreground/70"
          title={phraseKey}
        >
          {phraseKey}
        </div>
      </div>
      <div className="min-w-0 space-y-1">
        <Textarea
          rows={1}
          value={value}
          readOnly={readOnly}
          placeholder={source}
          aria-label={phraseKey}
          onChange={(e) => onChange(phraseKey, e.target.value)}
          className="min-h-9 resize-y px-2.5 py-1.5 text-[13px] md:text-[13px]"
        />
        {missing.length > 0 && (
          <p className="flex items-center gap-1.5 text-xs text-amber-500">
            <AlertTriangle className="size-3.5 shrink-0" />
            {t("admin.language.missing_placeholders", {
              list: missing.map((name) => `{${name}}`).join(", "),
            })}
          </p>
        )}
      </div>
      <div className="flex items-start justify-end gap-1">
        <Button
          variant="ghost"
          size="icon"
          className="size-8"
          title={t("admin.language.use_source")}
          aria-label={t("admin.language.use_source")}
          disabled={readOnly || !source}
          onClick={() => onChange(phraseKey, source)}
        >
          <CopyPlus className="size-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="size-8"
          title={t("common.reset")}
          aria-label={t("common.reset")}
          disabled={readOnly || value === ""}
          onClick={() => onChange(phraseKey, "")}
        >
          <RotateCcw className="size-4" />
        </Button>
      </div>
    </div>
  );
});
