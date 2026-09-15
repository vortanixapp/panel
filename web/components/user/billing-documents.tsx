"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  fetchBillingDocuments,
  openBillingAct,
  openBillingReconciliation,
  saveBillingPayer,
  type BillingPayerInput,
  type BillingPayerType,
} from "@/lib/api";
import { money, monthTitle, yearToDate } from "@/lib/accounting-period";
import { t } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

const CARD = "rounded-[20px] border border-[var(--vx-panel-line)] bg-[var(--vx-panel-card)]";
const INPUT =
  "h-10 w-full rounded-xl border border-[var(--vx-border-2)] bg-[var(--vx-card-2)] px-3.5 text-[13.5px] text-foreground outline-none focus:border-[var(--vx-border-hover)]";
const LABEL = "mb-1.5 block text-[12px] font-semibold text-muted-foreground";
const BUTTON =
  "rounded-full border border-[var(--vx-border-2)] px-4 py-2 text-[12.5px] text-muted-foreground transition-colors hover:border-[var(--vx-border-hover)] hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40";

const PAYER_TYPES: BillingPayerType[] = ["person", "ip", "company"];

const EMPTY_PAYER: BillingPayerInput = {
  payer_type: "person",
  legal_name: "",
  inn: "",
  kpp: "",
  ogrn: "",
  address: "",
};

export function BillingDocuments({ currency }: { currency: string }) {
  useT();
  const qc = useQueryClient();
  const queryKey = ["billing-documents", currency];
  const { data, isLoading } = useQuery({
    queryKey,
    queryFn: () => fetchBillingDocuments(currency),
  });

  const [payer, setPayer] = useState<BillingPayerInput>(EMPTY_PAYER);
  const [dirty, setDirty] = useState(false);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [period, setPeriod] = useState(yearToDate);

  useEffect(() => {
    if (!data || dirty) return;
    const p = data.payer;
    setPayer({
      payer_type: p.payer_type,
      legal_name: p.legal_name,
      inn: p.inn,
      kpp: p.kpp,
      ogrn: p.ogrn,
      address: p.address,
    });
  }, [data, dirty]);

  const saveMutation = useMutation({
    mutationFn: () => saveBillingPayer(payer),
    onSuccess: (res) => {
      qc.setQueryData(queryKey, res);
      setDirty(false);
      setError("");
      setNotice(t("billing.documents.saved"));
    },
    onError: (e: Error) => {
      setNotice("");
      setError(e.message || t("billing.documents.save_failed"));
    },
  });

  const update = (patch: Partial<BillingPayerInput>) => {
    setPayer((prev) => ({ ...prev, ...patch }));
    setDirty(true);
    setNotice("");
  };

  const run = async (key: string, action: () => Promise<void>) => {
    setBusy(key);
    setError("");
    try {
      await action();
    } catch (e) {
      setError(e instanceof Error && e.message ? e.message : t("billing.documents.open_failed"));
    } finally {
      setBusy("");
    }
  };

  const docCurrency = data?.currency || currency;
  const months = data?.months ?? [];
  const business = payer.payer_type !== "person";

  return (
    <div className={cn(CARD, "mt-3.5 px-[26px] py-6")}>
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <span className="text-base font-semibold">{t("billing.documents.title")}</span>
        <span className="text-[12.5px] text-[var(--vx-ink-faint)]">
          {t("billing.documents.subtitle")}
        </span>
      </div>

      {data && !data.company.ready && (
        <p className="mt-3 text-[12.5px] text-[var(--vx-warn)]">
          {t("billing.documents.company_incomplete")}
        </p>
      )}
      {error && <p className="mt-3 text-[13px] text-destructive">{error}</p>}

      <div className="mt-5 grid grid-cols-1 gap-6 lg:grid-cols-2">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            saveMutation.mutate();
          }}
        >
          <div className="text-[14px] font-semibold">{t("billing.documents.payer_title")}</div>
          <p className="mt-1 text-[12.5px] leading-[1.45] text-muted-foreground">
            {t("billing.documents.payer_hint")}
          </p>

          <div className="mt-3.5 flex flex-wrap gap-[3px] rounded-full border border-[var(--vx-border-2)] bg-[var(--vx-card-2)] p-[3px]">
            {PAYER_TYPES.map((type) => (
              <button
                key={type}
                type="button"
                onClick={() => update({ payer_type: type })}
                className={cn(
                  "flex-1 rounded-full px-3 py-[7px] text-[12.5px] font-semibold transition-colors",
                  payer.payer_type === type
                    ? "bg-[var(--vx-tint)] text-foreground"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                {t(`billing.documents.type.${type}`)}
              </button>
            ))}
          </div>

          <div className="mt-3.5 grid grid-cols-1 gap-3 sm:grid-cols-2">
            {business && (
              <label className="sm:col-span-2">
                <span className={LABEL}>{t("billing.documents.legal_name")}</span>
                <input
                  className={INPUT}
                  value={payer.legal_name}
                  placeholder={t(`billing.documents.legal_name_placeholder_${payer.payer_type}`)}
                  onChange={(e) => update({ legal_name: e.target.value })}
                />
              </label>
            )}
            <label>
              <span className={LABEL}>{t("billing.documents.inn")}</span>
              <input
                className={cn(INPUT, "font-mono")}
                inputMode="numeric"
                maxLength={12}
                value={payer.inn}
                onChange={(e) => update({ inn: e.target.value.replace(/\D/g, "") })}
              />
            </label>
            {payer.payer_type === "company" && (
              <label>
                <span className={LABEL}>{t("billing.documents.kpp")}</span>
                <input
                  className={cn(INPUT, "font-mono")}
                  maxLength={9}
                  value={payer.kpp}
                  onChange={(e) => update({ kpp: e.target.value.toUpperCase() })}
                />
              </label>
            )}
            {business && (
              <label>
                <span className={LABEL}>
                  {payer.payer_type === "ip"
                    ? t("billing.documents.ogrnip")
                    : t("billing.documents.ogrn")}
                </span>
                <input
                  className={cn(INPUT, "font-mono")}
                  inputMode="numeric"
                  maxLength={payer.payer_type === "ip" ? 15 : 13}
                  value={payer.ogrn}
                  onChange={(e) => update({ ogrn: e.target.value.replace(/\D/g, "") })}
                />
              </label>
            )}
            <label className="sm:col-span-2">
              <span className={LABEL}>{t("billing.documents.address")}</span>
              <input
                className={INPUT}
                value={payer.address}
                onChange={(e) => update({ address: e.target.value })}
              />
            </label>
          </div>

          <div className="mt-4 flex items-center gap-3">
            <button
              type="submit"
              disabled={saveMutation.isPending || isLoading}
              className="vx-btn rounded-full px-5 py-[10px] text-[13px] font-semibold disabled:cursor-not-allowed disabled:opacity-50"
            >
              {saveMutation.isPending ? t("billing.documents.saving") : t("billing.documents.save")}
            </button>
            {notice && <span className="text-[12.5px] text-muted-foreground">{notice}</span>}
          </div>
        </form>

        <div className="flex flex-col gap-6">
          <div>
            <div className="text-[14px] font-semibold">{t("billing.documents.acts_title")}</div>
            <p className="mt-1 text-[12.5px] leading-[1.45] text-muted-foreground">
              {t("billing.documents.acts_hint")}
            </p>
            {months.length === 0 ? (
              <div className="mt-3 rounded-[14px] border border-dashed border-[var(--vx-border-2)] px-4 py-5 text-[13px] text-muted-foreground">
                {isLoading
                  ? t("common.loading")
                  : t("billing.documents.acts_empty", { currency: docCurrency })}
              </div>
            ) : (
              <div className="mt-3 flex max-h-[280px] flex-col overflow-y-auto rounded-[14px] border border-[var(--vx-border-2)]">
                {months.map((m) => (
                  <div
                    key={m.period}
                    className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--vx-inset)] px-4 py-3 last:border-b-0"
                  >
                    <div className="min-w-0">
                      <div className="text-[13.5px] font-medium">
                        {monthTitle(m.period)}
                        {m.number !== null && (
                          <span className="ml-2 font-mono text-[12px] text-[var(--vx-ink-faint)]">
                            {t("billing.documents.act_number", { number: m.number })}
                          </span>
                        )}
                      </div>
                      <div className="mt-0.5 font-mono text-[11.5px] text-[var(--vx-ink-faint)]">
                        {t("billing.documents.act_meta", {
                          count: m.count,
                          amount: money(m.amount),
                          currency: docCurrency,
                        })}
                      </div>
                    </div>
                    {m.closed ? (
                      <button
                        type="button"
                        className={BUTTON}
                        disabled={busy !== ""}
                        onClick={() =>
                          void run(`act-${m.period}`, async () => {
                            await openBillingAct(m.period, docCurrency);
                            await qc.invalidateQueries({ queryKey });
                          })
                        }
                      >
                        {t("billing.documents.act_open")}
                      </button>
                    ) : (
                      <span className="text-[12px] text-[var(--vx-ink-faint)]">
                        {t("billing.documents.act_pending")}
                      </span>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>

          <div>
            <div className="text-[14px] font-semibold">
              {t("billing.documents.reconciliation_title")}
            </div>
            <p className="mt-1 text-[12.5px] leading-[1.45] text-muted-foreground">
              {t("billing.documents.reconciliation_hint")}
            </p>
            <div className="mt-3 flex flex-wrap items-end gap-2.5">
              <label>
                <span className={LABEL}>{t("billing.documents.from")}</span>
                <input
                  type="date"
                  className={cn(INPUT, "w-[160px]")}
                  value={period.from}
                  onChange={(e) => setPeriod((prev) => ({ ...prev, from: e.target.value }))}
                />
              </label>
              <label>
                <span className={LABEL}>{t("billing.documents.to")}</span>
                <input
                  type="date"
                  className={cn(INPUT, "w-[160px]")}
                  value={period.to}
                  onChange={(e) => setPeriod((prev) => ({ ...prev, to: e.target.value }))}
                />
              </label>
              <button
                type="button"
                className={cn(BUTTON, "h-10")}
                disabled={busy !== "" || !period.from || !period.to}
                onClick={() =>
                  void run("reconciliation", () =>
                    openBillingReconciliation(period.from, period.to, docCurrency)
                  )
                }
              >
                {t("billing.documents.reconciliation_open")}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
