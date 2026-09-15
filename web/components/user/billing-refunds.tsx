"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  cancelRefundRequest,
  createRefundRequest,
  fetchRefundRequests,
  type BalanceRefundRequest,
} from "@/lib/api";
import { money } from "@/lib/accounting-period";
import { localeTag, t } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

const CARD = "rounded-[20px] border border-[var(--vx-panel-line)] bg-[var(--vx-panel-card)]";
const INPUT =
  "h-10 w-full rounded-xl border border-[var(--vx-border-2)] bg-[var(--vx-card-2)] px-3.5 text-[13.5px] text-foreground outline-none focus:border-[var(--vx-border-hover)]";
const LABEL = "mb-1.5 block text-[12px] font-semibold text-muted-foreground";
const BUTTON =
  "rounded-full border border-[var(--vx-border-2)] px-4 py-2 text-[12.5px] text-muted-foreground transition-colors hover:border-[var(--vx-border-hover)] hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40";

const KEY = ["billing-refund-requests"];

function fmtDate(iso: string) {
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? iso : date.toLocaleDateString(localeTag());
}

function RequestRow({
  request,
  onCancel,
  busy,
}: {
  request: BalanceRefundRequest;
  onCancel: (id: string) => void;
  busy: boolean;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--vx-inset)] px-4 py-3 last:border-b-0">
      <div className="min-w-0">
        <div className="text-[13.5px] font-medium">
          {t("billing.refund.number", { number: request.number })}
          <span className="ml-2 text-[12px] text-[var(--vx-ink-faint)]">
            {t(`billing.refund.status.${request.status}`)}
          </span>
        </div>
        <div className="mt-0.5 font-mono text-[11.5px] text-[var(--vx-ink-faint)]">
          {t("billing.refund.meta", {
            amount: money(request.amount),
            currency: request.currency,
            date: fmtDate(request.created_at),
          })}
          {request.refunded > 0 &&
            ` · ${t("billing.refund.refunded", { amount: money(request.refunded), currency: request.currency })}`}
        </div>
        {request.admin_note && (
          <div className="mt-1 text-[12px] text-muted-foreground">
            {t("billing.refund.note", { note: request.admin_note })}
          </div>
        )}
      </div>
      {request.status === "pending" && request.refunded === 0 && (
        <button type="button" className={BUTTON} disabled={busy} onClick={() => onCancel(request.id)}>
          {t("billing.refund.cancel")}
        </button>
      )}
    </div>
  );
}

export function BillingRefunds({ currency }: { currency: string }) {
  useT();
  const qc = useQueryClient();
  const { data } = useQuery({ queryKey: KEY, queryFn: fetchRefundRequests });
  const [open, setOpen] = useState(false);
  const [amount, setAmount] = useState("");
  const [method, setMethod] = useState<"original" | "bank">("original");
  const [recipient, setRecipient] = useState("");
  const [account, setAccount] = useState("");
  const [bik, setBik] = useState("");
  const [bankName, setBankName] = useState("");
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");

  const wallet = data?.wallets.find((w) => w.currency.toUpperCase() === currency);
  const balance = Number(wallet?.balance ?? 0);
  const requests = data?.requests ?? [];
  const pendingExists = requests.some((r) => r.status === "pending" && r.currency === currency);

  const createMut = useMutation({
    mutationFn: () =>
      createRefundRequest({
        currency,
        amount: Number(amount.replace(",", ".")),
        method,
        recipient,
        bank_account: account,
        bank_bik: bik,
        bank_name: bankName,
        reason,
      }),
    onSuccess: (res) => {
      qc.setQueryData(KEY, res);
      setOpen(false);
      setError("");
      setReason("");
    },
    onError: (e: Error) => setError(e.message || t("billing.refund.create_failed")),
  });

  const cancelMut = useMutation({
    mutationFn: (id: string) => cancelRefundRequest(id),
    onSuccess: (res) => qc.setQueryData(KEY, res),
    onError: (e: Error) => setError(e.message || t("billing.refund.create_failed")),
  });

  const toggleForm = () => {
    if (!open) {
      setAmount(balance > 0 ? String(Math.floor(balance * 100) / 100) : "");
      setError("");
    }
    setOpen((v) => !v);
  };

  return (
    <div className={cn(CARD, "mt-3.5 px-[26px] py-6")}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="text-base font-semibold">{t("billing.refund.title")}</div>
          <div className="mt-1 text-[12.5px] text-[var(--vx-ink-faint)]">{t("billing.refund.subtitle")}</div>
        </div>
        <button
          type="button"
          className={BUTTON}
          disabled={!open && (balance <= 0 || pendingExists)}
          title={pendingExists ? t("billing.refund.pending_exists") : balance <= 0 ? t("billing.refund.nothing") : undefined}
          onClick={toggleForm}
        >
          {open ? t("billing.refund.close") : t("billing.refund.open")}
        </button>
      </div>

      {error && <p className="mt-3 text-[13px] text-destructive">{error}</p>}

      {open && (
        <form
          className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2"
          onSubmit={(e) => {
            e.preventDefault();
            createMut.mutate();
          }}
        >
          <label>
            <span className={LABEL}>{t("billing.refund.amount", { currency })}</span>
            <input
              className={cn(INPUT, "font-mono")}
              inputMode="decimal"
              value={amount}
              onChange={(e) => setAmount(e.target.value.replace(/[^\d.,]/g, ""))}
            />
            <span className="mt-1 block text-[11.5px] text-[var(--vx-ink-faint)]">
              {t("billing.refund.available", { amount: money(balance), currency })}
            </span>
          </label>
          <div>
            <span className={LABEL}>&nbsp;</span>
            <div className="flex gap-[3px] rounded-full border border-[var(--vx-border-2)] bg-[var(--vx-card-2)] p-[3px]">
              {(["original", "bank"] as const).map((value) => (
                <button
                  key={value}
                  type="button"
                  onClick={() => setMethod(value)}
                  className={cn(
                    "flex-1 rounded-full px-3 py-[7px] text-[12.5px] font-semibold transition-colors",
                    method === value
                      ? "bg-[var(--vx-tint)] text-foreground"
                      : "text-muted-foreground hover:text-foreground"
                  )}
                >
                  {t(`billing.refund.method_${value}`)}
                </button>
              ))}
            </div>
            <span className="mt-1 block text-[11.5px] text-[var(--vx-ink-faint)]">
              {t("billing.refund.method_hint")}
            </span>
          </div>
          {method === "bank" && (
            <>
              <label className="sm:col-span-2">
                <span className={LABEL}>{t("billing.refund.recipient")}</span>
                <input className={INPUT} value={recipient} onChange={(e) => setRecipient(e.target.value)} />
              </label>
              <label>
                <span className={LABEL}>{t("billing.refund.account")}</span>
                <input
                  className={cn(INPUT, "font-mono")}
                  inputMode="numeric"
                  maxLength={20}
                  value={account}
                  onChange={(e) => setAccount(e.target.value.replace(/\D/g, ""))}
                />
              </label>
              <label>
                <span className={LABEL}>{t("billing.refund.bik")}</span>
                <input
                  className={cn(INPUT, "font-mono")}
                  inputMode="numeric"
                  maxLength={9}
                  value={bik}
                  onChange={(e) => setBik(e.target.value.replace(/\D/g, ""))}
                />
              </label>
              <label className="sm:col-span-2">
                <span className={LABEL}>{t("billing.refund.bank_name")}</span>
                <input className={INPUT} value={bankName} onChange={(e) => setBankName(e.target.value)} />
              </label>
            </>
          )}
          <label className="sm:col-span-2">
            <span className={LABEL}>{t("billing.refund.reason")}</span>
            <input className={INPUT} value={reason} onChange={(e) => setReason(e.target.value)} />
          </label>
          <div className="flex flex-wrap items-center gap-3 sm:col-span-2">
            <button
              type="submit"
              disabled={createMut.isPending || !amount}
              className="vx-btn rounded-full px-5 py-[10px] text-[13px] font-semibold disabled:cursor-not-allowed disabled:opacity-50"
            >
              {t("billing.refund.submit")}
            </button>
            <span className="text-[12px] text-[var(--vx-ink-faint)]">{t("billing.refund.terms")}</span>
          </div>
        </form>
      )}

      <div className="mt-4 rounded-[14px] border border-[var(--vx-border-2)]">
        {requests.length === 0 ? (
          <div className="px-4 py-5 text-[13px] text-muted-foreground">{t("billing.refund.empty")}</div>
        ) : (
          requests.map((request) => (
            <RequestRow
              key={request.id}
              request={request}
              busy={cancelMut.isPending}
              onCancel={(id) => cancelMut.mutate(id)}
            />
          ))
        )}
      </div>
    </div>
  );
}
