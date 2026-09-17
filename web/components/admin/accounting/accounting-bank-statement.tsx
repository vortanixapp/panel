"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { applyBankStatement, previewBankStatement, type BankStatementLine } from "@/lib/api";
import { money } from "@/lib/accounting-period";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

function statusVariant(status: BankStatementLine["status"]): "secondary" | "outline" | "destructive" {
  if (status === "match") return "secondary";
  if (status === "unmatched") return "destructive";
  return "outline";
}

export function AccountingBankStatement() {
  const t = useT();
  const [lines, setLines] = useState<BankStatementLine[]>([]);
  const [summary, setSummary] = useState<{ documents: number; outgoing: number } | null>(null);
  const [selected, setSelected] = useState<Record<string, boolean>>({});
  const [loading, setLoading] = useState(false);
  const [applying, setApplying] = useState(false);

  const onFile = async (file: File | undefined) => {
    if (!file) return;
    setLoading(true);
    try {
      const res = await previewBankStatement(file);
      setLines(res.lines);
      setSummary({ documents: res.documents, outgoing: res.outgoing });
      setSelected(
        Object.fromEntries(res.lines.filter((line) => line.status === "match").map((line) => [line.fingerprint, true]))
      );
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : t("admin.accounting.statement.preview_failed"));
    } finally {
      setLoading(false);
    }
  };

  const chosen = lines.filter((line) => line.status === "match" && selected[line.fingerprint]);

  const apply = async () => {
    setApplying(true);
    try {
      const res = await applyBankStatement(chosen);
      toast.success(t("admin.accounting.statement.applied", { applied: res.applied, skipped: res.skipped }));
      res.failures.forEach((failure) => toast.error(failure));
      setLines([]);
      setSummary(null);
      setSelected({});
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : t("admin.accounting.statement.apply_failed"));
    } finally {
      setApplying(false);
    }
  };

  return (
    <div className="space-y-4">
      <p className="max-w-3xl text-sm text-muted-foreground">{t("admin.accounting.statement.hint")}</p>
      <div className="flex flex-wrap items-end gap-3 rounded-lg border bg-card p-4">
        <div className="space-y-1">
          <Label>{t("admin.accounting.statement.file")}</Label>
          <Input
            type="file"
            accept=".txt,text/plain"
            disabled={loading}
            onChange={(e) => {
              void onFile(e.target.files?.[0]);
              e.target.value = "";
            }}
          />
        </div>
        {loading ? <span className="text-sm text-muted-foreground">{t("admin.accounting.statement.loading")}</span> : null}
        {summary ? (
          <span className="text-sm text-muted-foreground">
            {t("admin.accounting.statement.summary", {
              documents: summary.documents,
              incoming: lines.length,
              outgoing: summary.outgoing,
            })}
          </span>
        ) : null}
      </div>

      {summary ? (
        <div className="vx-tbl-wrap rounded-lg border bg-card">
          {lines.length === 0 ? (
            <div className="p-8 text-center text-sm text-muted-foreground">{t("admin.accounting.statement.empty")}</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="vx-tbl w-full text-sm">
                <thead>
                  <tr className="text-xs text-muted-foreground">
                    <th className="px-4 py-2" />
                    <th className="px-4 py-2 text-left font-normal">{t("admin.accounting.statement.col_date")}</th>
                    <th className="px-4 py-2 text-left font-normal">{t("admin.accounting.statement.col_payer")}</th>
                    <th className="px-4 py-2 text-right font-normal">{t("admin.accounting.statement.col_amount")}</th>
                    <th className="px-4 py-2 text-left font-normal">{t("admin.accounting.statement.col_purpose")}</th>
                    <th className="px-4 py-2 text-left font-normal">{t("admin.accounting.statement.col_match")}</th>
                  </tr>
                </thead>
                <tbody>
                  {lines.map((line) => (
                    <tr key={line.fingerprint} className="border-t align-top">
                      <td className="px-4 py-2" data-cell="lead">
                        {line.status === "match" ? (
                          <Checkbox
                            checked={Boolean(selected[line.fingerprint])}
                            onCheckedChange={(value) =>
                              setSelected((prev) => ({ ...prev, [line.fingerprint]: value === true }))
                            }
                          />
                        ) : null}
                      </td>
                      <td className="px-4 py-2 whitespace-nowrap" data-label={t("admin.accounting.statement.col_date")}>
                        {line.date ? new Date(`${line.date}T00:00:00`).toLocaleDateString(localeTag()) : "—"}
                        <div className="font-mono text-xs text-muted-foreground">№ {line.number}</div>
                      </td>
                      <td className="px-4 py-2 font-medium" data-cell="full">
                        {line.payer_name || "—"}
                        {line.payer_inn ? <div className="font-mono text-xs text-muted-foreground">{line.payer_inn}</div> : null}
                      </td>
                      <td className="px-4 py-2 text-right font-mono whitespace-nowrap" data-label={t("admin.accounting.statement.col_amount")}>{money(line.amount)}</td>
                      <td className="max-w-[360px] px-4 py-2 text-xs" data-cell="block" data-label={t("admin.accounting.statement.col_purpose")}>{line.purpose}</td>
                      <td className="px-4 py-2" data-label={t("admin.accounting.statement.col_match")}>
                        <Badge variant={statusVariant(line.status)}>
                          {t(`admin.accounting.statement.status.${line.status}`, { invoice: line.invoice_no ?? "" })}
                        </Badge>
                        {line.user_email ? <div className="mt-1 text-xs text-muted-foreground">{line.user_email}</div> : null}
                        {line.note ? <div className="mt-1 text-xs text-muted-foreground">{line.note}</div> : null}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      ) : null}

      {chosen.length > 0 ? (
        <Button disabled={applying} onClick={() => void apply()}>
          {t("admin.accounting.statement.apply", { count: chosen.length })}
        </Button>
      ) : null}
    </div>
  );
}
