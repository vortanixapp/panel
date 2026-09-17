"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchReceiptOffsets,
  markReceiptOffsetSent,
  retryReceiptOffset,
  type ReceiptOffset,
} from "@/lib/api";
import { money } from "@/lib/accounting-period";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

const FILTERS = ["active", "pending", "failed", "manual", "sent", "skipped"] as const;
const STATUSES: ReceiptOffset["status"][] = ["pending", "failed", "manual", "sent", "skipped"];

function statusVariant(status: ReceiptOffset["status"]): "secondary" | "outline" | "destructive" {
  if (status === "failed") return "destructive";
  if (status === "sent" || status === "skipped") return "outline";
  return "secondary";
}

export function AccountingOffsets() {
  const t = useT();
  const qc = useQueryClient();
  const [filter, setFilter] = useState<string>("active");
  const [busy, setBusy] = useState("");
  const query = useQuery({
    queryKey: ["admin-receipt-offsets", filter],
    queryFn: () => fetchReceiptOffsets(filter === "active" ? "" : filter),
    refetchInterval: 30_000,
  });

  const run = async (id: string, action: () => Promise<unknown>) => {
    setBusy(id);
    try {
      await action();
      toast.success(t("admin.accounting.offsets.done"));
      void qc.invalidateQueries({ queryKey: ["admin-receipt-offsets"] });
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : t("admin.accounting.offsets.failed"));
    } finally {
      setBusy("");
    }
  };

  const markSent = (offset: ReceiptOffset) => {
    const reference = window.prompt(t("admin.accounting.offsets.reference_prompt"));
    if (reference === null || !reference.trim()) return;
    void run(offset.id, () => markReceiptOffsetSent(offset.id, reference.trim()));
  };

  const offsets = query.data?.offsets ?? [];
  const counts = query.data?.counts ?? {};

  return (
    <div className="space-y-4">
      <p className="max-w-3xl text-sm text-muted-foreground">{t("admin.accounting.offsets.hint")}</p>
      <div className="flex flex-wrap items-center gap-3">
        <Select value={filter} onValueChange={setFilter}>
          <SelectTrigger className="w-[240px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {FILTERS.map((value) => (
              <SelectItem key={value} value={value}>
                {value === "active"
                  ? t("admin.accounting.offsets.filter.active")
                  : t(`admin.accounting.offsets.status.${value}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <div className="flex flex-wrap gap-2">
          {STATUSES.map((status) => (
            <Badge key={status} variant={statusVariant(status)}>
              {t(`admin.accounting.offsets.status.${status}`)}: {counts[status] ?? 0}
            </Badge>
          ))}
        </div>
      </div>

      <div className="vx-tbl-wrap rounded-lg border bg-card">
        {query.isLoading ? (
          <div className="p-4">
            <Skeleton className="h-32 w-full" />
          </div>
        ) : offsets.length === 0 ? (
          <div className="p-8 text-center text-sm text-muted-foreground">{t("admin.accounting.offsets.empty")}</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="vx-tbl w-full text-sm">
              <thead>
                <tr className="text-xs text-muted-foreground">
                  <th className="px-4 py-2 text-left font-normal">{t("admin.accounting.offsets.col_date")}</th>
                  <th className="px-4 py-2 text-left font-normal">{t("admin.accounting.offsets.col_client")}</th>
                  <th className="px-4 py-2 text-left font-normal">{t("admin.accounting.offsets.col_invoice")}</th>
                  <th className="px-4 py-2 text-left font-normal">{t("admin.accounting.offsets.col_provider")}</th>
                  <th className="px-4 py-2 text-left font-normal">{t("admin.accounting.offsets.col_service")}</th>
                  <th className="px-4 py-2 text-right font-normal">{t("admin.accounting.offsets.col_amount")}</th>
                  <th className="px-4 py-2 text-left font-normal">{t("admin.accounting.offsets.col_status")}</th>
                  <th className="px-4 py-2" />
                </tr>
              </thead>
              <tbody>
                {offsets.map((offset) => (
                  <tr key={offset.id} className="border-t align-top">
                    <td className="px-4 py-2 whitespace-nowrap" data-label={t("admin.accounting.offsets.col_date")}>
                      {new Date(offset.charged_at ?? offset.created_at).toLocaleDateString(localeTag())}
                    </td>
                    <td className="px-4 py-2 font-medium" data-cell="full">{offset.user_email || "—"}</td>
                    <td className="px-4 py-2 font-mono" data-label={t("admin.accounting.offsets.col_invoice")}>{offset.invoice_no || "—"}</td>
                    <td className="px-4 py-2" data-label={t("admin.accounting.offsets.col_provider")}>{offset.provider_name || "—"}</td>
                    <td className="px-4 py-2" data-label={t("admin.accounting.offsets.col_service")}>{offset.service}</td>
                    <td className="px-4 py-2 text-right font-mono" data-label={t("admin.accounting.offsets.col_amount")}>{money(offset.amount)}</td>
                    <td className="px-4 py-2" data-label={t("admin.accounting.offsets.col_status")}>
                      <Badge variant={statusVariant(offset.status)}>
                        {t(`admin.accounting.offsets.status.${offset.status}`)}
                      </Badge>
                      {offset.reference ? (
                        <div className="mt-1 font-mono text-xs text-muted-foreground">{offset.reference}</div>
                      ) : null}
                      {offset.error ? <div className="mt-1 text-xs text-destructive">{offset.error}</div> : null}
                    </td>
                    <td className="px-4 py-2" data-cell="actions">
                      <div className="flex flex-wrap justify-end gap-2">
                        {offset.status === "failed" ? (
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={busy !== ""}
                            onClick={() => void run(offset.id, () => retryReceiptOffset(offset.id))}
                          >
                            {t("admin.accounting.offsets.retry")}
                          </Button>
                        ) : null}
                        {offset.status === "manual" || offset.status === "failed" ? (
                          <Button size="sm" variant="ghost" disabled={busy !== ""} onClick={() => markSent(offset)}>
                            {t("admin.accounting.offsets.mark_sent")}
                          </Button>
                        ) : null}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
