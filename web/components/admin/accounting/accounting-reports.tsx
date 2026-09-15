"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  downloadAccountingReport,
  fetchAccountingSummary,
  type AccountingReportKind,
  type AccountingSummary,
} from "@/lib/api";
import {
  PERIOD_PRESETS,
  money,
  monthTitle,
  presetRange,
  type PeriodPreset,
} from "@/lib/accounting-period";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";
import { cn } from "@/lib/utils";

const REPORTS: AccountingReportKind[] = ["kudir", "payments", "refunds", "services", "balances", "acts", "offsets"];

function taxLine(t: TranslateFn, s: AccountingSummary): string {
  const currency = s.currency;
  switch (s.tax_system) {
    case "usn_income":
    case "usn_income_outcome":
    case "patent":
    case "npd":
      return t(`admin.accounting.tax.${s.tax_system}`, { amount: money(s.net_income), currency });
    case "osn":
      return t("admin.accounting.tax.osn", {
        revenue: money(s.services.amount - s.services_vat),
        vat: money(s.services_vat),
        currency,
      });
  }
  return "";
}

export function AccountingReports() {
  const t = useT();
  const [preset, setPreset] = useState<PeriodPreset | "custom">("month");
  const [range, setRange] = useState(() => presetRange("month"));
  const [currency, setCurrency] = useState("");
  const [downloading, setDownloading] = useState<AccountingReportKind | null>(null);

  const summaryQuery = useQuery({
    queryKey: ["admin-accounting-summary", range.from, range.to, currency],
    queryFn: () => fetchAccountingSummary({ ...range, currency }),
    enabled: range.from !== "" && range.to !== "",
    placeholderData: (prev) => prev,
  });
  const s = summaryQuery.data;
  const activeCurrency = currency || s?.currency || "";

  const applyPreset = (value: string) => {
    if (value === "custom") {
      setPreset("custom");
      return;
    }
    const next = value as PeriodPreset;
    setPreset(next);
    setRange(presetRange(next));
  };

  const download = async (kind: AccountingReportKind) => {
    setDownloading(kind);
    try {
      await downloadAccountingReport(kind, { ...range, currency: activeCurrency });
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : t("admin.accounting.download_failed"));
    } finally {
      setDownloading(null);
    }
  };

  const cards = s
    ? [
        {
          label: t("admin.accounting.card.income"),
          value: money(s.income.amount),
          note: t("admin.accounting.card.income_note", { count: s.income.count }),
        },
        {
          label: t("admin.accounting.card.refunds"),
          value: money(s.refunds.amount),
          note: t("admin.accounting.card.refunds_note", { count: s.refunds.count }),
        },
        {
          label: t("admin.accounting.card.net"),
          value: money(s.net_income),
          note: taxLine(t, s),
        },
        {
          label: t("admin.accounting.card.services"),
          value: money(s.services.amount),
          note:
            s.vat !== "none"
              ? t("admin.accounting.card.services_vat", {
                  count: s.services.count,
                  vat: money(s.services_vat),
                })
              : t("admin.accounting.card.services_note", { count: s.services.count }),
        },
        {
          label: t("admin.accounting.card.bonuses"),
          value: money(s.bonuses),
          note: "",
        },
        {
          label: t("admin.accounting.card.balances"),
          value: money(s.balance_close),
          note: t("admin.accounting.card.balances_note", {
            open: money(s.balance_open),
            close: money(s.balance_close),
          }),
        },
      ]
    : [];

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end gap-3 rounded-lg border bg-card p-4">
        <div className="space-y-1">
          <Label>{t("admin.accounting.period")}</Label>
          <Select value={preset} onValueChange={applyPreset}>
            <SelectTrigger className="w-[190px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PERIOD_PRESETS.map((p) => (
                <SelectItem key={p} value={p}>
                  {t(`admin.accounting.preset.${p}`)}
                </SelectItem>
              ))}
              <SelectItem value="custom">{t("admin.accounting.preset.custom")}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label>{t("admin.accounting.from")}</Label>
          <Input
            type="date"
            className="w-[160px]"
            value={range.from}
            onChange={(e) => {
              setPreset("custom");
              setRange((prev) => ({ ...prev, from: e.target.value }));
            }}
          />
        </div>
        <div className="space-y-1">
          <Label>{t("admin.accounting.to")}</Label>
          <Input
            type="date"
            className="w-[160px]"
            value={range.to}
            onChange={(e) => {
              setPreset("custom");
              setRange((prev) => ({ ...prev, to: e.target.value }));
            }}
          />
        </div>
        <div className="space-y-1">
          <Label>{t("admin.accounting.currency")}</Label>
          <Select value={activeCurrency} onValueChange={setCurrency}>
            <SelectTrigger className="w-[110px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(s?.currencies ?? (activeCurrency ? [activeCurrency] : [])).map((c) => (
                <SelectItem key={c} value={c}>
                  {c}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {summaryQuery.isError ? (
        <div className="rounded-lg border border-destructive/40 bg-destructive/5 p-4 text-sm">
          {(summaryQuery.error as Error)?.message || t("admin.accounting.load_failed")}
        </div>
      ) : null}

      {s && !s.requisites_ready ? (
        <div className="rounded-lg border border-amber-500/40 bg-amber-500/5 p-4 text-sm">
          {t("admin.accounting.requisites_warning")}
        </div>
      ) : null}

      {!s ? (
        <Skeleton className="h-40 w-full" />
      ) : (
        <div className={cn("grid gap-3 sm:grid-cols-2 lg:grid-cols-3", summaryQuery.isFetching && "opacity-70")}>
          {cards.map((card) => (
            <div key={card.label} className="rounded-lg border bg-card p-4">
              <div className="text-xs text-muted-foreground">{card.label}</div>
              <div className="mt-2 font-mono text-2xl font-semibold">
                {card.value} <span className="text-sm font-normal text-muted-foreground">{s.currency}</span>
              </div>
              {card.note ? <div className="mt-1 text-xs text-muted-foreground">{card.note}</div> : null}
            </div>
          ))}
        </div>
      )}

      {s ? (
        <div className="grid gap-4 lg:grid-cols-2">
          <div className="rounded-lg border bg-card">
            <div className="border-b p-4 text-sm font-medium">{t("admin.accounting.months_title")}</div>
            {s.months.length === 0 ? (
              <div className="p-6 text-center text-sm text-muted-foreground">{t("admin.accounting.no_data")}</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-xs text-muted-foreground">
                      <th className="px-4 py-2 text-left font-normal">{t("admin.accounting.col_month")}</th>
                      <th className="px-4 py-2 text-right font-normal">{t("admin.accounting.col_income")}</th>
                      <th className="px-4 py-2 text-right font-normal">{t("admin.accounting.col_refunds")}</th>
                      <th className="px-4 py-2 text-right font-normal">{t("admin.accounting.col_services")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {s.months.map((row) => (
                      <tr key={row.month} className="border-t">
                        <td className="px-4 py-2">{monthTitle(row.month)}</td>
                        <td className="px-4 py-2 text-right font-mono">{money(row.income)}</td>
                        <td className="px-4 py-2 text-right font-mono">{money(row.refunds)}</td>
                        <td className="px-4 py-2 text-right font-mono">{money(row.services)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          <div className="rounded-lg border bg-card">
            <div className="border-b p-4 text-sm font-medium">{t("admin.accounting.providers_title")}</div>
            {s.providers.length === 0 ? (
              <div className="p-6 text-center text-sm text-muted-foreground">{t("admin.accounting.no_data")}</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-xs text-muted-foreground">
                      <th className="px-4 py-2 text-left font-normal">{t("admin.accounting.col_provider")}</th>
                      <th className="px-4 py-2 text-right font-normal">{t("admin.accounting.col_count")}</th>
                      <th className="px-4 py-2 text-right font-normal">{t("admin.accounting.col_amount")}</th>
                      <th className="px-4 py-2 text-right font-normal">{t("admin.accounting.col_refunds")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {s.providers.map((row) => (
                      <tr key={row.provider} className="border-t">
                        <td className="px-4 py-2">{row.name}</td>
                        <td className="px-4 py-2 text-right font-mono">{row.count}</td>
                        <td className="px-4 py-2 text-right font-mono">{money(row.amount)}</td>
                        <td className="px-4 py-2 text-right font-mono">{money(row.refunds)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      ) : null}

      <div>
        <h2 className="text-lg font-semibold">{t("admin.accounting.reports_title")}</h2>
        <p className="mb-3 text-sm text-muted-foreground">
          {t("admin.accounting.reports_hint", { timezone: s?.timezone ?? "Europe/Moscow" })}
        </p>
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {REPORTS.map((kind) => (
            <div key={kind} className="flex flex-col justify-between gap-3 rounded-lg border bg-card p-4">
              <div>
                <div className="text-sm font-medium">{t(`admin.accounting.report.${kind}`)}</div>
                <div className="mt-1 text-xs text-muted-foreground">
                  {t(`admin.accounting.report.${kind}_hint`)}
                </div>
              </div>
              <Button
                variant="outline"
                size="sm"
                className="self-start"
                disabled={downloading !== null || !range.from || !range.to}
                onClick={() => void download(kind)}
              >
                {downloading === kind ? t("common.loading") : t("admin.accounting.download")}
              </Button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
