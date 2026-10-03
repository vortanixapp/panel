"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { usePublicSettings } from "@/context/brand-provider";
import { useT } from "@/hooks/use-translations";
import { fetchTaxRates, saveTaxRates, type TaxRate } from "@/lib/api";
import { localeTag } from "@/lib/i18n";
import { countryName } from "@/lib/countries";
import { cn } from "@/lib/utils";

const KEY = ["admin-accounting-tax-rates"];

type Draft = Record<number, { rate: string; enabled: boolean }>;

export function AccountingTaxes() {
  const t = useT();
  const qc = useQueryClient();
  const { accountingRegions } = usePublicSettings();
  const query = useQuery({ queryKey: KEY, queryFn: fetchTaxRates });
  const [draft, setDraft] = useState<Draft>({});
  const [search, setSearch] = useState("");
  const [region, setRegion] = useState("all");

  const rates = useMemo(() => query.data?.rates ?? [], [query.data]);
  const tag = localeTag();

  const visible = useMemo(() => {
    const needle = search.trim().toLowerCase();
    return rates
      .filter((item) => region === "all" || item.region === region)
      .filter((item) => accountingRegions.includes(item.region))
      .filter((item) => {
        if (!needle) return true;
        const name = countryName(item.country, tag).toLowerCase();
        return (
          name.includes(needle) ||
          item.country.toLowerCase().includes(needle) ||
          item.subdivision.toLowerCase().includes(needle)
        );
      })
      .sort((a, b) =>
        `${countryName(a.country, tag)}${a.subdivision}`.localeCompare(
          `${countryName(b.country, tag)}${b.subdivision}`,
          tag
        )
      );
  }, [rates, region, search, accountingRegions, tag]);

  const valueOf = (item: TaxRate) =>
    draft[item.id] ?? { rate: String(item.rate), enabled: item.enabled };

  const changed = rates.filter((item) => {
    const d = draft[item.id];
    return d && (d.enabled !== item.enabled || Number(d.rate) !== item.rate);
  });

  const setDraftFor = (item: TaxRate, patch: Partial<{ rate: string; enabled: boolean }>) =>
    setDraft((prev) => ({ ...prev, [item.id]: { ...valueOf(item), ...patch } }));

  const save = useMutation({
    mutationFn: () =>
      saveTaxRates(
        changed.map((item) => ({
          id: item.id,
          rate: Number(valueOf(item).rate),
          enabled: valueOf(item).enabled,
        }))
      ),
    onSuccess: () => {
      toast.success(t("admin.accounting.taxes.saved"));
      setDraft({});
      void qc.invalidateQueries({ queryKey: KEY });
    },
    onError: (e: Error) => toast.error(e.message || t("admin.accounting.save_failed")),
  });

  const invalid = changed.some((item) => {
    const n = Number(valueOf(item).rate);
    return !Number.isFinite(n) || n < 0 || n > 100;
  });

  if (query.isLoading) return <Skeleton className="h-96 w-full" />;

  const regionTabs = ["all", ...accountingRegions];

  return (
    <div className="space-y-4">
      <div className="rounded-lg border bg-card p-4 text-sm text-muted-foreground">
        {t("admin.accounting.taxes.hint", {
          country: countryName(query.data?.seller_country ?? "RU", tag),
        })}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t("admin.accounting.taxes.search")}
          className="max-w-xs"
        />
        <div className="flex flex-wrap gap-1">
          {regionTabs.map((id) => (
            <Button
              key={id}
              size="sm"
              variant={region === id ? "default" : "outline"}
              onClick={() => setRegion(id)}
            >
              {id === "all" ? t("common.all") : t(`accounting.region.${id}`)}
            </Button>
          ))}
        </div>
        <div className="ms-auto flex items-center gap-2">
          {changed.length > 0 && (
            <Badge variant="outline">{t("admin.accounting.taxes.changed", { count: changed.length })}</Badge>
          )}
          <Button
            disabled={changed.length === 0 || invalid || save.isPending}
            onClick={() => save.mutate()}
          >
            {t("common.save")}
          </Button>
        </div>
      </div>

      {visible.length === 0 ? (
        <div className="rounded-lg border bg-card p-8 text-center text-sm text-muted-foreground">
          {t("admin.accounting.taxes.empty")}
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border bg-card">
          <div className="grid grid-cols-[minmax(0,1fr)_110px_80px] gap-3 border-b px-4 py-2 text-xs text-muted-foreground sm:grid-cols-[minmax(0,1fr)_140px_100px]">
            <span>{t("admin.accounting.taxes.col_country")}</span>
            <span>{t("admin.accounting.taxes.col_rate")}</span>
            <span>{t("admin.accounting.taxes.col_enabled")}</span>
          </div>
          <div className="max-h-[60vh] overflow-y-auto">
            {visible.map((item) => {
              const v = valueOf(item);
              const bad = !Number.isFinite(Number(v.rate)) || Number(v.rate) < 0 || Number(v.rate) > 100;
              return (
                <div
                  key={item.id}
                  className={cn(
                    "grid grid-cols-[minmax(0,1fr)_110px_80px] items-center gap-3 border-b px-4 py-2 last:border-b-0 sm:grid-cols-[minmax(0,1fr)_140px_100px]",
                    v.enabled && "bg-primary/5"
                  )}
                >
                  <div className="min-w-0">
                    <div className="truncate text-sm font-medium">
                      {countryName(item.country, tag)}
                      {item.subdivision ? ` · ${item.subdivision}` : ""}
                    </div>
                    <div className="text-xs text-muted-foreground">{item.label}</div>
                  </div>
                  <Input
                    value={v.rate}
                    inputMode="decimal"
                    aria-invalid={bad || undefined}
                    className={cn("h-8 font-mono", bad && "border-destructive")}
                    onChange={(e) => setDraftFor(item, { rate: e.target.value.replace(",", ".") })}
                  />
                  <Switch
                    checked={v.enabled}
                    onCheckedChange={(enabled) => setDraftFor(item, { enabled })}
                    aria-label={t("admin.accounting.taxes.col_enabled")}
                  />
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
