"use client";

import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { fetchAdminLanguage, updateAdminLanguage } from "@/lib/api";
import { OVERRIDABLE_STRINGS } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";

type Row = {
  key: string;
  value: string;
  fallback: string;
  groupKey: string;
};

const REGISTRY_GROUP_ORDER = Array.from(
  new Set(OVERRIDABLE_STRINGS.map((s) => s.groupKey))
);
const OTHER_GROUP_KEY = "admin.language.group.other";

const COLUMNS = "grid-cols-[minmax(220px,0.9fr)_minmax(260px,1.1fr)_90px]";

export function LanguagePageContent() {
  const t = useT();
  const [locale] = useState("ru");
  const [search, setSearch] = useState("");
  const [rows, setRows] = useState<Row[]>([]);

  const languageQuery = useQuery({
    queryKey: ["admin-language", locale],
    queryFn: () => fetchAdminLanguage(locale),
  });

  const saveMutation = useMutation({
    mutationFn: (payload: Record<string, string>) =>
      updateAdminLanguage(locale, payload),
    onSuccess: () => toast.success(t("admin.language.saved")),
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.save_failed")),
  });

  useEffect(() => {
    const saved = new Map(
      (languageQuery.data?.messages ?? []).map((m) => [m.key, m.value])
    );
    const known: Row[] = OVERRIDABLE_STRINGS.map((s) => ({
      key: s.key,
      value: saved.get(s.key) ?? "",
      fallback: s.default,
      groupKey: s.groupKey,
    }));
    const extra: Row[] = [];
    for (const [key, value] of saved) {
      if (OVERRIDABLE_STRINGS.some((s) => s.key === key)) continue;
      extra.push({ key, value, fallback: "", groupKey: OTHER_GROUP_KEY });
    }
    setRows([...known, ...extra]);
  }, [languageQuery.data?.messages]);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return rows;
    return rows.filter(
      (r) =>
        r.key.toLowerCase().includes(q) ||
        r.value.toLowerCase().includes(q) ||
        r.fallback.toLowerCase().includes(q)
    );
  }, [rows, search]);

  const grouped = useMemo(() => {
    const order = [...REGISTRY_GROUP_ORDER, OTHER_GROUP_KEY];
    const map = new Map<string, Row[]>();
    for (const row of filtered) {
      const list = map.get(row.groupKey);
      if (list) list.push(row);
      else map.set(row.groupKey, [row]);
    }
    return order
      .filter((g) => map.has(g))
      .map((g) => ({ groupKey: g, rows: map.get(g)! }));
  }, [filtered]);

  const overriddenCount = rows.filter((r) => r.value.trim() !== "").length;

  function setValue(key: string, value: string) {
    setRows((prev) => prev.map((r) => (r.key === key ? { ...r, value } : r)));
  }

  function resetAll() {
    if (!overriddenCount) return;
    if (!confirm(t("admin.language.reset_all_confirm"))) return;
    setRows((prev) => prev.map((r) => ({ ...r, value: "" })));
  }

  function saveAll() {
    const payload: Record<string, string> = {};
    for (const row of rows) {
      payload[row.key] = row.value;
    }
    saveMutation.mutate(payload);
  }

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {t("admin.language.title")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("admin.language.subtitle")}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <span
              className={cn(
                "font-mono text-xs",
                overriddenCount ? "text-amber-500" : "text-muted-foreground"
              )}
            >
              {overriddenCount
                ? t("admin.language.overridden", { count: overriddenCount })
                : t("admin.language.overridden_none")}
            </span>
            <Button
              variant="outline"
              className="h-[38px] text-[13px]"
              onClick={resetAll}
              disabled={!overriddenCount || saveMutation.isPending}
            >
              {t("admin.language.reset_all")}
            </Button>
            <Button
              className="h-[38px] text-[13px]"
              onClick={saveAll}
              disabled={saveMutation.isPending}
            >
              {saveMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t("admin.language.filter_placeholder")}
            className="h-9 max-w-[360px] flex-1 basis-[300px] rounded-lg text-[13px] md:text-[13px]"
          />
          <span className="text-xs text-muted-foreground">
            {t("admin.language.registry_hint")}
          </span>
        </div>

        {languageQuery.isLoading ? (
          <Skeleton className="h-96 w-full rounded-2xl" />
        ) : (
          <div className="space-y-4">
            {grouped.map(({ groupKey, rows: groupRows }) => {
              const changedInGroup = groupRows.filter((r) =>
                r.value.trim()
              ).length;
              return (
                <section
                  key={groupKey}
                  className="overflow-hidden rounded-2xl border bg-card"
                >
                  <div className="flex items-center justify-between gap-3.5 px-6 py-4">
                    <span className="text-[15px] font-semibold">
                      {t(groupKey)}
                    </span>
                    <span className="font-mono text-xs text-muted-foreground">
                      {changedInGroup
                        ? t("admin.language.group_overridden", {
                            changed: changedInGroup,
                            total: groupRows.length,
                          })
                        : t("admin.language.group_rows", {
                            count: groupRows.length,
                          })}
                    </span>
                  </div>

                  {groupKey === OTHER_GROUP_KEY && (
                    <p className="border-t bg-amber-500/5 px-6 py-2.5 text-xs text-amber-500">
                      {t("admin.language.other_group_hint")}
                    </p>
                  )}

                  <div className="overflow-x-auto">
                    <div className="min-w-[640px]">
                      <div
                        className={cn(
                          "grid gap-4 border-t bg-muted/40 px-6 py-2.5 font-mono text-[11px] tracking-wider text-muted-foreground uppercase",
                          COLUMNS
                        )}
                      >
                        <div>{t("admin.language.col_default")}</div>
                        <div>{t("admin.language.col_custom")}</div>
                        <div />
                      </div>

                      {groupRows.map((row) => {
                        const changed = row.value.trim() !== "";
                        return (
                          <div
                            key={row.key}
                            className={cn(
                              "grid items-center gap-4 border-t px-6 py-3 transition-colors hover:bg-muted/30",
                              COLUMNS
                            )}
                          >
                            <div className="min-w-0 space-y-1">
                              <div className="truncate text-[13px]">
                                {row.fallback || "—"}
                              </div>
                              <div className="truncate font-mono text-[11px] text-muted-foreground/70">
                                {row.key}
                              </div>
                            </div>
                            <Input
                              value={row.value}
                              placeholder={row.fallback}
                              onChange={(e) => setValue(row.key, e.target.value)}
                              className={cn(
                                "h-9 rounded-lg text-[13px] md:text-[13px]",
                                !changed && "text-muted-foreground"
                              )}
                            />
                            <div className="flex justify-end">
                              <Button
                                variant="outline"
                                size="sm"
                                className="h-7 rounded-md px-2.5 text-xs"
                                disabled={!changed}
                                onClick={() => setValue(row.key, "")}
                              >
                                {t("common.reset")}
                              </Button>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                </section>
              );
            })}

            {grouped.length === 0 && (
              <div className="rounded-2xl border bg-card py-12 text-center text-[13px] text-muted-foreground">
                {t("common.not_found")}
              </div>
            )}

            <p className="text-xs text-muted-foreground">
              {t("admin.language.reload_hint")}
            </p>
          </div>
        )}
      </div>
    </PageShell>
  );
}
