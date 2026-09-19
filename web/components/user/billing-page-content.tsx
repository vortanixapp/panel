"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { BillingDocuments } from "@/components/user/billing-documents";
import {
  BillingPaymentDialog,
  paymentStatusLabel,
  paymentTone,
} from "@/components/user/billing-payment-dialog";
import { BillingRefunds } from "@/components/user/billing-refunds";
import { IdentificationNotice } from "@/components/user/identification-notice";
import {
  createTopup,
  createWallet,
  fetchBilling,
  fetchDashboard,
  fetchTopupForm,
  type BillingData,
  type TopupProvider,
  type Transaction,
} from "@/lib/api";
import { BILLING_PROVIDERS } from "@/lib/billing-providers";
import { formatAmount } from "@/lib/format";
import { localeTag, t } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";

const EMPTY_BILLING: BillingData = {
  wallets: [],
  selected_wallet: null,
  transactions: [],
  credits_total: 0,
  debits_total: 0,
  available_currencies: [],
};

const CARD = "rounded-[20px] border border-[var(--vx-panel-line)] bg-[var(--vx-panel-card)]";
const CARD_RAISED =
  "rounded-[20px] border border-[var(--vx-panel-line-strong)] bg-[var(--vx-elevated)]";
const CARD_WARN =
  "rounded-[20px] border border-[var(--vx-panel-line-strong)] bg-[var(--vx-elevated)]";

const PRESETS = [500, 1000, 2000, 5000, 10000];

type Filter = "all" | "in" | "out";

function fmtMoney(v: number) {
  return Math.round(v).toLocaleString(localeTag());
}

function fmtRowDate(iso: string) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return `${d.toLocaleDateString(localeTag(), { day: "2-digit", month: "short" })}, ${d.toLocaleTimeString(localeTag(), { hour: "2-digit", minute: "2-digit" })}`;
}

function providerMeta(code: string) {
  const def = BILLING_PROVIDERS.find((p) => p.key === code);
  return { name: def?.name ?? code, note: def ? t(def.descKey) : "" };
}

function resolveApiError(err: unknown): string {
  if (err && typeof err === "object" && "message" in err) {
    return String((err as { message: string }).message);
  }
  return t("billing.topup.create_error");
}

type Quote = {
  chargeAmount: number | null;
  chargeCurrency: string;
  feePercent: number;
  converted: boolean;
};

function buildQuote(
  amount: number,
  walletCurrency: string,
  provider: TopupProvider,
  fxFeePercent: number,
  rates: Record<string, number> | null | undefined
): Quote {
  const chargeCurrency = (provider.currency || walletCurrency).toUpperCase();
  const feePercent = provider.fee_percent ?? 0;
  const converted = chargeCurrency !== walletCurrency;
  let value = amount;
  if (converted) {
    const from = rates?.[walletCurrency];
    const to = rates?.[chargeCurrency];
    if (!from || !to) return { chargeAmount: null, chargeCurrency, feePercent, converted };
    value = value * (to / from) * (1 + fxFeePercent / 100);
  }
  value = value * (1 + feePercent / 100);
  return {
    chargeAmount: Math.ceil(value * 100 - 1e-6) / 100,
    chargeCurrency,
    feePercent,
    converted,
  };
}

function externalRedirect(url?: string) {
  if (!url || !/^https?:\/\//i.test(url)) return null;
  try {
    const target = new URL(url);
    if (target.origin === window.location.origin && /^\/billing(\/|$)/.test(target.pathname)) {
      return null;
    }
    return target.toString();
  } catch {
    return null;
  }
}

const STATUS_BADGE = {
  success: "border-emerald-500/30 text-emerald-500",
  failure: "border-rose-500/30 text-rose-500",
  pending: "border-amber-500/30 text-amber-500",
  other: "border-[var(--vx-border-2)] text-muted-foreground",
};

function withRunningBalance(transactions: Transaction[], currentBalance: number) {
  let running = currentBalance;
  return transactions.map((t) => {
    const after = running;
    const amount = Math.abs(Number(t.amount) || 0);
    running += t.type === "credit" ? -amount : amount;
    return { tx: t, after };
  });
}

function spendBreakdown(transactions: Transaction[]) {
  const other = t("billing.balance.other_charges");
  const groups = new Map<string, number>();
  for (const t of transactions) {
    if (t.type === "credit") continue;
    const key = t.description?.trim() || other;
    groups.set(key, (groups.get(key) ?? 0) + Math.abs(Number(t.amount) || 0));
  }
  const rows = [...groups.entries()]
    .map(([name, amount]) => ({ name, amount }))
    .sort((a, b) => b.amount - a.amount)
    .slice(0, 5);
  const total = rows.reduce((s, r) => s + r.amount, 0);
  return rows.map((r, i) => ({
    ...r,
    pct: total > 0 ? (r.amount / total) * 100 : 0,
    bar: ["var(--vx-fg)", "var(--vx-dim)", "var(--vx-muted)", "var(--vx-faint)", "var(--vx-ghost)"][i] ?? "var(--vx-ghost)",
  }));
}

function toCsv(rows: { tx: Transaction; after: number }[], currency: string) {
  const head = [
    t("common.date"),
    t("common.type"),
    t("common.description"),
    t("common.amount"),
    t("billing.history.csv.balance_after"),
    t("billing.history.csv.currency"),
  ];
  const body = rows.map(({ tx, after }) => [
    new Date(tx.created_at).toISOString(),
    tx.type === "credit" ? t("billing.tx.credit") : t("billing.tx.debit"),
    (tx.description || "").replace(/"/g, '""'),
    (tx.type === "credit" ? "" : "-") + Math.abs(Number(tx.amount) || 0),
    after,
    currency,
  ]);
  return [head, ...body]
    .map((r) => r.map((c) => `"${String(c)}"`).join(";"))
    .join("\r\n");
}

function ArrowIcon({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.8}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
    >
      <path d="M5 12h13M13 6.5 18.5 12 13 17.5" />
    </svg>
  );
}

export function BillingPageContent() {
  useT();
  const qc = useQueryClient();
  const router = useRouter();
  const searchParams = useSearchParams();
  const paymentId = searchParams.get("payment");

  const [selectedWalletId, setSelectedWalletId] = useState<string | null>(null);
  const [amount, setAmount] = useState(1000);
  const [method, setMethod] = useState("");
  const [promoOpen, setPromoOpen] = useState(false);
  const [promoCode, setPromoCode] = useState("");
  const [freekassaMethod, setFreekassaMethod] = useState("");
  const [filter, setFilter] = useState<Filter>("all");
  const [visible, setVisible] = useState(8);
  const [paymentsVisible, setPaymentsVisible] = useState(5);
  const [topupError, setTopupError] = useState("");

  const openPayment = (id: string) =>
    router.replace(`/billing?payment=${encodeURIComponent(id)}`, { scroll: false });
  const closePayment = () => router.replace("/billing", { scroll: false });

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.billing(selectedWalletId),
    queryFn: () => fetchBilling(selectedWalletId),
    placeholderData: (prev) => prev,
  });

  const { data: topupForm } = useQuery({
    queryKey: queryKeys.billingTopup(),
    queryFn: fetchTopupForm,
  });

  const { data: dash } = useQuery({
    queryKey: ["dashboard"],
    queryFn: fetchDashboard,
  });

  const createWalletMutation = useMutation({
    mutationFn: (currency: string) => createWallet(currency),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["billing"] }),
  });

  const ready = Boolean(data);
  useEffect(() => {
    if (ready && window.location.hash === "#topup") {
      document.getElementById("topup")?.scrollIntoView({ block: "start" });
    }
  }, [ready]);

  const d = data ?? EMPTY_BILLING;
  const wallet =
    d.selected_wallet ??
    d.wallets.find((w) => w.id === selectedWalletId) ??
    d.wallets[0] ??
    null;
  const currency = (wallet?.currency || "RUB").toUpperCase();
  const symbol = currency === "RUB" ? "₽" : currency;
  const balance = wallet?.balance ?? 0;
  const activeWalletId = selectedWalletId ?? d.selected_wallet?.id ?? wallet?.id ?? "";

  const providerRows = topupForm?.providers ?? [];
  const enabledProviders = topupForm?.enabled_providers ?? [];
  const freekassaMethods = topupForm?.freekassa_methods ?? [];
  const payments = topupForm?.payments ?? [];
  const activeMethod = enabledProviders.includes(method) ? method : enabledProviders[0] ?? "";
  const activeProvider = providerRows.find((p) => p.code === activeMethod);
  const needsFreekassaMethod =
    activeMethod === "freekassa" && freekassaMethods.length > 0 && !freekassaMethod;
  const quote =
    activeProvider && amount > 0
      ? buildQuote(amount, currency, activeProvider, topupForm?.fx?.fee_percent ?? 0, topupForm?.fx?.rates)
      : null;

  const topupMutation = useMutation({
    mutationFn: async () => {
      if (!activeProvider) throw new Error(t("billing.topup.provider_unavailable"));
      if (!amount || amount <= 0) throw new Error(t("billing.topup.amount_required"));
      if (needsFreekassaMethod) throw new Error(t("billing.topup.freekassa_select"));
      return createTopup({
        walletId: activeWalletId || undefined,
        providerId: activeProvider.id,
        providerCode: activeMethod,
        amount,
        promoCode: promoCode.trim() || undefined,
        paymentMethodId:
          activeMethod === "freekassa" && freekassaMethod ? freekassaMethod : undefined,
      });
    },
    onSuccess: (res) => {
      void qc.invalidateQueries({ queryKey: queryKeys.billingTopup() });
      const target = externalRedirect(res.redirect_url);
      if (target) {
        window.location.href = target;
        return;
      }
      openPayment(res.payment_id);
    },
    onError: (err) => setTopupError(resolveApiError(err)),
  });

  const rowsWithBalance = useMemo(
    () => withRunningBalance(d.transactions, balance),
    [d.transactions, balance]
  );

  const filtered = useMemo(
    () =>
      rowsWithBalance.filter(({ tx }) =>
        filter === "all" ? true : filter === "in" ? tx.type === "credit" : tx.type !== "credit"
      ),
    [rowsWithBalance, filter]
  );

  const breakdown = useMemo(() => spendBreakdown(d.transactions), [d.transactions]);

  const downloadCsv = () => {
    const blob = new Blob(["﻿" + toCsv(filtered, currency)], {
      type: "text/csv;charset=utf-8",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `vortanix-billing-${currency}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const nextCharge = dash?.next_charge_text && dash.next_charge_text !== "—" ? dash.next_charge_text : null;
  const dailyBurn = useMemo(() => {
    const perMonth = (dash?.recent_servers ?? []).reduce(
      (s, srv) => s + Number(srv.tariff?.price_monthly ?? 0),
      0
    );
    return perMonth / 30;
  }, [dash?.recent_servers]);
  const daysLeft = dailyBurn > 0 ? Math.floor(balance / dailyBurn) : null;

  const cards = [
    {
      label: t("billing.balance.current"),
      value: fmtMoney(balance),
      unit: symbol,
      note:
        daysLeft !== null
          ? t("billing.balance.days_left", { days: daysLeft })
          : t("billing.balance.topup_hint"),
      icon: "ri-wallet-3-line",
      raised: true as const,
      tone: "" as const,
    },
    {
      label: t("billing.tx.credits"),
      value: `+${fmtMoney(d.credits_total)}`,
      unit: t("billing.balance.unit_total", { symbol }),
      note:
        d.credits_total > 0
          ? t("billing.balance.credits_note")
          : t("billing.balance.no_operations"),
      icon: "ri-arrow-down-line",
      raised: false as const,
      tone: "" as const,
    },
    {
      label: t("billing.tx.debits"),
      value: `−${fmtMoney(d.debits_total)}`,
      unit: t("billing.balance.unit_total", { symbol }),
      note:
        dailyBurn > 0
          ? t("billing.balance.daily_burn", { amount: fmtMoney(dailyBurn), symbol })
          : t("billing.balance.no_subscriptions"),
      icon: "ri-arrow-up-line",
      raised: false as const,
      tone: "" as const,
    },
    {
      label: t("billing.balance.next_charge"),
      value: nextCharge ?? "—",
      unit: "",
      note: nextCharge
        ? t("billing.balance.next_charge_note")
        : t("billing.balance.next_charge_empty"),
      icon: "ri-time-line",
      raised: false as const,
      tone: nextCharge ? ("warn" as const) : ("" as const),
    },
  ];

  if (isLoading && !data) {
    return (
      <PageShell variant="user">
        <div className="font-panel flex items-center justify-center py-24 text-muted-foreground">
          <Loader2 className="mr-2 size-6 animate-spin" />
          {t("common.loading")}
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="user">
      <div className="font-panel">
        {isError && (
          <p className="mb-4 text-sm text-destructive">
            {t("billing.balance.load_failed")}
          </p>
        )}

        <IdentificationNotice className="mb-3.5" />

        <div className="vx-stagger grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-4">
          {cards.map((c) => (
            <div
              key={c.label}
              data-spotlight
              className={cn(
                c.tone === "warn" ? CARD_WARN : c.raised ? CARD_RAISED : CARD,
                "relative isolate px-6 py-[22px]"
              )}
            >
              <div className="flex items-center justify-between">
                <span className="text-[13px] font-semibold text-muted-foreground">
                  {c.label}
                </span>
                <i
                  className={cn(
                    c.icon,
                    "text-[17px]",
                    c.tone === "warn" ? "text-[var(--vx-warn)]" : "text-muted-foreground"
                  )}
                />
              </div>
              <div className="mt-[18px] flex items-baseline gap-[7px]">
                <span
                  className={cn(
                    "font-mono text-[29px] font-medium tracking-[-0.035em]",
                    c.tone === "warn" ? "text-[var(--vx-warn)]" : "text-foreground"
                  )}
                >
                  {c.value}
                </span>
                {c.unit && (
                  <span className="text-[13px] text-[var(--vx-ink-faint)]">{c.unit}</span>
                )}
              </div>
              <div className="mt-2.5 text-[13px] text-muted-foreground">{c.note}</div>
            </div>
          ))}
        </div>

        <div className="mt-3.5 grid grid-cols-1 items-start gap-3.5 lg:grid-cols-2">
          <form
            id="topup"
            className={cn(CARD, "scroll-mt-24 px-5 py-6 sm:px-[26px]")}
            onSubmit={(e) => {
              e.preventDefault();
              setTopupError("");
              topupMutation.mutate();
            }}
          >
            <div className="flex items-center justify-between gap-3.5">
              <span className="text-base font-semibold">{t("billing.topup.title")}</span>
              <span className="font-mono text-xs text-[var(--vx-ink-faint)]">
                {t("billing.topup.instant")}
              </span>
            </div>

            <div className="mt-[18px] flex flex-wrap gap-2">
              {PRESETS.map((v) => {
                const on = amount === v;
                return (
                  <button
                    key={v}
                    type="button"
                    onClick={() => setAmount(v)}
                    className={cn(
                      "rounded-xl border px-[18px] py-[11px] font-mono text-sm transition-colors hover:border-[var(--vx-border-hover)]",
                      on
                        ? "border-[var(--vx-border-hover)] bg-[var(--vx-tint)] text-foreground"
                        : "border-[var(--vx-border-2)] bg-[var(--vx-card-2)] text-[var(--vx-ink-dim)]"
                    )}
                  >
                    {v.toLocaleString(localeTag())} {symbol}
                  </button>
                );
              })}
            </div>

            <label className="mt-3.5 flex items-center gap-3 rounded-[14px] border border-[var(--vx-border-2)] bg-[var(--vx-card-2)] px-4 py-3.5">
              <span className="text-[13px] whitespace-nowrap text-muted-foreground">
                {t("billing.topup.custom_amount")}
              </span>
              <input
                inputMode="numeric"
                value={amount ? amount.toLocaleString(localeTag()) : ""}
                onChange={(e) => {
                  const v = parseInt(e.target.value.replace(/\D/g, ""), 10);
                  setAmount(Number.isNaN(v) ? 0 : v);
                }}
                className="min-w-0 flex-1 border-none bg-transparent font-mono text-[19px] text-foreground outline-none"
              />
              <span className="font-mono text-[15px] text-[var(--vx-ink-faint)]">
                {symbol}
              </span>
            </label>

            {promoOpen ? (
              <label className="mt-2.5 flex items-center gap-3 rounded-[14px] border border-[var(--vx-border-2)] bg-[var(--vx-card-2)] px-4 py-3">
                <span className="text-[13px] whitespace-nowrap text-muted-foreground">
                  {t("billing.topup.promo")}
                </span>
                <input
                  value={promoCode}
                  onChange={(e) => setPromoCode(e.target.value)}
                  autoComplete="off"
                  className="min-w-0 flex-1 border-none bg-transparent font-mono text-[15px] text-foreground uppercase outline-none"
                />
              </label>
            ) : (
              <button
                type="button"
                onClick={() => setPromoOpen(true)}
                className="mt-2.5 text-[12.5px] text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline"
              >
                {t("billing.topup.promo_toggle")}
              </button>
            )}

            <div className="mt-[18px] text-[12.5px] font-semibold tracking-[0.06em] text-[var(--vx-ink-faint)] uppercase">
              {t("billing.topup.method")}
            </div>
            {enabledProviders.length === 0 ? (
              <div className="mt-3 rounded-[14px] border border-dashed border-[var(--vx-border-2)] px-4 py-5 text-[13px] text-muted-foreground">
                {t("billing.topup.no_providers")}
              </div>
            ) : (
              <div className="mt-3 grid grid-cols-2 gap-2.5 sm:grid-cols-3">
                {enabledProviders.map((code) => {
                  const on = activeMethod === code;
                  const meta = providerMeta(code);
                  const row = providerRows.find((p) => p.code === code);
                  const fee = row?.fee_percent ?? 0;
                  return (
                    <button
                      key={code}
                      type="button"
                      onClick={() => setMethod(code)}
                      className={cn(
                        "rounded-[14px] border p-3.5 text-left transition-colors hover:border-[var(--vx-border-hover)]",
                        on
                          ? "border-[var(--vx-border-hover)] bg-[var(--vx-tint)]"
                          : "border-[var(--vx-border-2)] bg-[var(--vx-card-2)]"
                      )}
                    >
                      <div
                        className={cn(
                          "text-[13.5px] font-semibold break-words",
                          on ? "text-foreground" : "text-[var(--vx-ink-dim)]"
                        )}
                      >
                        {row?.name || meta.name}
                      </div>
                      <div className="mt-[5px] text-[11.5px] text-[var(--vx-ink-faint)]">
                        {meta.note}
                      </div>
                      {fee > 0 && (
                        <div className="mt-2 inline-flex rounded-full border border-[var(--vx-border-2)] px-2 py-0.5 text-[10.5px] text-muted-foreground">
                          {t("billing.topup.fee", { fee })}
                        </div>
                      )}
                    </button>
                  );
                })}
              </div>
            )}

            {activeMethod === "freekassa" && freekassaMethods.length > 0 && (
              <select
                value={freekassaMethod}
                onChange={(e) => setFreekassaMethod(e.target.value)}
                className="mt-3 h-11 w-full rounded-[14px] border border-[var(--vx-border-2)] bg-[var(--vx-card-2)] px-4 text-[13px] text-foreground outline-none focus:border-[var(--vx-border-hover)]"
              >
                <option value="">{t("billing.topup.freekassa_select")}</option>
                {freekassaMethods.map((m) => (
                  <option key={m.id} value={String(m.id)}>
                    {m.name}
                  </option>
                ))}
              </select>
            )}

            {quote && (quote.feePercent > 0 || quote.converted) && (
              <div className="mt-3 flex flex-wrap items-center justify-between gap-x-3 gap-y-1 rounded-[14px] border border-[var(--vx-border-2)] bg-[var(--vx-card-2)] px-4 py-3 text-[12.5px]">
                <span className="text-muted-foreground">
                  {[
                    quote.feePercent > 0 ? t("billing.topup.fee", { fee: quote.feePercent }) : "",
                    quote.converted
                      ? t("billing.topup.conversion", { currency: quote.chargeCurrency })
                      : "",
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </span>
                <span className="font-mono font-medium text-foreground">
                  {quote.chargeAmount != null
                    ? t("billing.payment.to_pay", {
                        amount: formatAmount(quote.chargeAmount),
                        currency: quote.chargeCurrency,
                      })
                    : t("billing.topup.rate_pending", { currency: quote.chargeCurrency })}
                </span>
              </div>
            )}

            {topupError && (
              <p className="mt-3 text-[13px] text-destructive">{topupError}</p>
            )}

            <div className="mt-[18px] flex flex-col items-stretch gap-2.5 sm:flex-row sm:items-center sm:gap-3">
              <button
                type="submit"
                disabled={
                  topupMutation.isPending || !activeMethod || amount <= 0 || needsFreekassaMethod
                }
                className="vx-btn flex-1 rounded-full px-4 py-[13px] text-center text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-50"
              >
                {topupMutation.isPending
                  ? t("billing.topup.creating")
                  : activeProvider?.manual
                    ? t("billing.topup.bank_submit")
                    : t("billing.topup.submit_amount", {
                        amount: amount.toLocaleString(localeTag()),
                        symbol,
                      })}
              </button>
              <div className="text-center text-[12.5px] leading-[1.4] text-[var(--vx-ink-faint)] sm:max-w-[150px] sm:text-left">
                {t("billing.topup.receipt_hint")}
              </div>
            </div>
          </form>

          <div className="flex flex-col gap-3.5">
            <div className={cn(CARD, "px-[26px] py-6")}>
              <div className="flex items-center justify-between gap-3.5">
                <span className="text-base font-semibold">{t("billing.spend.title")}</span>
                <span className="font-mono text-xs text-[var(--vx-ink-faint)]">
                  {d.debits_total > 0
                    ? t("billing.spend.total", { amount: fmtMoney(d.debits_total), symbol })
                    : t("billing.spend.no_data")}
                </span>
              </div>
              {breakdown.length === 0 ? (
                <div className="mt-[18px] py-6 text-[13.5px] text-muted-foreground">
                  {t("billing.spend.empty")}
                </div>
              ) : (
                <div className="mt-[18px] flex flex-col gap-3.5">
                  {breakdown.map((b) => (
                    <div key={b.name}>
                      <div className="flex items-baseline justify-between gap-3">
                        <span className="truncate text-[13.5px] font-medium">{b.name}</span>
                        <span className="font-mono text-[13px] whitespace-nowrap text-[var(--vx-ink-dim)]">
                          {fmtMoney(b.amount)} {symbol}
                        </span>
                      </div>
                      <div className="mt-2 h-[5px] overflow-hidden rounded-[3px] bg-[var(--vx-tint)]">
                        <div
                          className="h-[5px]"
                          style={{ width: `${b.pct}%`, background: b.bar }}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className={cn(CARD, "flex flex-wrap items-center gap-4 px-[26px] py-[22px]")}>
              <div className="flex size-[38px] flex-shrink-0 items-center justify-center rounded-xl bg-[var(--vx-tint)] text-muted-foreground">
                <i className="ri-exchange-funds-line text-lg" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="text-[14.5px] font-semibold">{t("billing.wallet.title")}</div>
                <div className="mt-1 text-[12.5px] leading-[1.45] text-muted-foreground">
                  {t("billing.wallet.hint")}
                </div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <select
                  value={activeWalletId}
                  onChange={(e) => setSelectedWalletId(e.target.value || null)}
                  className="h-10 rounded-full border border-[var(--vx-border-2)] bg-[var(--vx-card-2)] px-4 text-[13px] text-foreground outline-none focus:border-[var(--vx-border-hover)]"
                >
                  {d.wallets.map((w) => (
                    <option key={w.id} value={w.id}>
                      {w.currency.toUpperCase()} — {formatAmount(w.balance)}
                    </option>
                  ))}
                </select>
                {d.available_currencies.map((c) => (
                  <button
                    key={c}
                    type="button"
                    disabled={createWalletMutation.isPending}
                    onClick={() => createWalletMutation.mutate(c)}
                    className="h-10 rounded-full border border-[var(--vx-border-2)] px-3.5 text-[12.5px] text-muted-foreground transition-colors hover:border-[var(--vx-border-hover)] hover:text-foreground disabled:opacity-50"
                  >
                    + {c}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>

        {payments.length > 0 && (
          <div className={cn(CARD, "mt-3.5 overflow-hidden")}>
            <div className="flex flex-wrap items-center gap-3.5 border-b border-[var(--vx-panel-line)] px-5 py-5 sm:px-[26px]">
              <span className="text-base font-semibold">{t("billing.topup.my_topups")}</span>
              <span className="font-mono text-xs text-[var(--vx-ink-faint)]">
                {t("billing.topup.last_20")}
              </span>
            </div>
            {payments.slice(0, paymentsVisible).map((p) => {
              const tone = paymentTone(p.status);
              const bonus =
                p.credited_amount != null && Number(p.credited_amount) > Number(p.amount)
                  ? Number(p.credited_amount) - Number(p.amount)
                  : 0;
              return (
                <button
                  key={p.id}
                  type="button"
                  onClick={() => openPayment(p.id)}
                  className="flex w-full items-center gap-3 border-b border-[var(--vx-inset)] px-5 py-3.5 text-left transition-colors last:border-b-0 hover:bg-[var(--vx-elevated)] sm:px-[26px]"
                >
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-[13.5px] font-medium">
                      {p.provider_name || (p.provider ? providerMeta(p.provider).name : "—")}
                    </div>
                    <div className="mt-1 truncate font-mono text-[11px] text-[var(--vx-ink-faint)]">
                      {fmtRowDate(p.created_at)} · #{String(p.id).slice(0, 8)}
                    </div>
                  </div>
                  <div className="flex flex-shrink-0 flex-col items-end gap-1">
                    <span className="font-mono text-sm">
                      {formatAmount(p.amount)} {(p.currency || "RUB").toUpperCase()}
                    </span>
                    {bonus > 0 && (
                      <span className="font-mono text-[11px] text-emerald-500">
                        +{formatAmount(bonus)} {t("billing.topup.bonus_suffix")}
                      </span>
                    )}
                    <span
                      className={cn(
                        "rounded-full border px-2 py-0.5 text-[10.5px] font-semibold",
                        STATUS_BADGE[tone]
                      )}
                    >
                      {paymentStatusLabel(p.status)}
                    </span>
                  </div>
                </button>
              );
            })}
            {payments.length > paymentsVisible && (
              <div className="flex justify-end px-5 py-3 sm:px-[26px]">
                <button
                  type="button"
                  onClick={() => setPaymentsVisible(payments.length)}
                  className="flex items-center gap-2 rounded-full border border-[var(--vx-border-2)] px-4 py-2 text-[12.5px] text-muted-foreground transition-colors hover:border-[var(--vx-border-hover)] hover:text-foreground"
                >
                  {t("billing.history.show_more")}
                  <ArrowIcon className="size-3.5" />
                </button>
              </div>
            )}
          </div>
        )}

        <div className={cn(CARD, "mt-3.5 overflow-hidden")}>
          <div className="flex flex-wrap items-center gap-3.5 border-b border-[var(--vx-panel-line)] px-[26px] py-5">
            <span className="text-base font-semibold">{t("billing.history.title")}</span>
            <span className="font-mono text-xs text-[var(--vx-ink-faint)]">
              {t("billing.history.shown", {
                shown: Math.min(visible, filtered.length),
                total: filtered.length,
              })}
            </span>
            <div className="flex-1" />
            <div className="flex gap-[3px] rounded-full border border-[var(--vx-border-2)] bg-[var(--vx-card-2)] p-[3px]">
              {(
                [
                  ["all", t("common.all")],
                  ["in", t("billing.tx.credits")],
                  ["out", t("billing.tx.debits")],
                ] as const
              ).map(([key, label]) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => {
                    setFilter(key);
                    setVisible(8);
                  }}
                  className={cn(
                    "rounded-full px-[15px] py-[7px] text-[12.5px] font-semibold transition-colors",
                    filter === key
                      ? "bg-[var(--vx-tint)] text-foreground"
                      : "text-muted-foreground hover:text-foreground"
                  )}
                >
                  {label}
                </button>
              ))}
            </div>
            <button
              type="button"
              onClick={downloadCsv}
              disabled={filtered.length === 0}
              className="flex items-center gap-2 rounded-full border border-[var(--vx-border-2)] px-3.5 py-2 text-[12.5px] text-muted-foreground transition-colors hover:border-[var(--vx-border-hover)] hover:text-foreground disabled:opacity-40"
            >
              <i className="ri-file-list-3-line" />
              {t("billing.history.export_csv")}
            </button>
          </div>

          {filtered.length === 0 ? (
            <div className="flex flex-col items-center gap-3.5 px-[26px] py-[74px]">
              <div className="flex size-[52px] items-center justify-center rounded-2xl border border-[var(--vx-border-2)] bg-[var(--vx-inset)] text-[var(--vx-ghost)]">
                <i className="ri-file-list-3-line text-2xl" />
              </div>
              <div className="text-base font-semibold">
                {t("billing.history.empty_title")}
              </div>
              <div className="max-w-[340px] text-center text-[13.5px] leading-[1.55] text-[var(--vx-ink-faint)]">
                {t("billing.history.empty_hint")}
              </div>
              <a
                href="#topup"
                className="vx-btn mt-1 rounded-full px-[22px] py-[11px] text-[13.5px] font-semibold"
              >
                {t("billing.history.topup_cta")}
              </a>
            </div>
          ) : (
            <>
              <div className="hidden md:block">
                <div className="grid grid-cols-[150px_150px_1fr_130px_120px] gap-4 border-b border-[var(--vx-panel-line)] bg-[var(--vx-card-2)] px-[26px] py-[13px] font-mono text-[11px] tracking-[0.1em] text-[var(--vx-ink-faint)] uppercase">
                  <div>{t("common.date")}</div>
                  <div>{t("common.type")}</div>
                  <div>{t("common.description")}</div>
                  <div className="text-right">{t("common.amount")}</div>
                  <div className="text-right">{t("billing.history.balance")}</div>
                </div>
                {filtered.slice(0, visible).map(({ tx, after }) => {
                  const credit = tx.type === "credit";
                  return (
                    <div
                      key={tx.id}
                      className="grid grid-cols-[150px_150px_1fr_130px_120px] items-center gap-4 border-b border-[var(--vx-inset)] px-[26px] py-[15px] transition-colors hover:bg-[var(--vx-elevated)]"
                    >
                      <div className="font-mono text-[12.5px] text-muted-foreground">
                        {fmtRowDate(tx.created_at)}
                      </div>
                      <div>
                        <span
                          className={cn(
                            "inline-flex items-center gap-[7px] rounded-full border px-[11px] py-1 text-[11.5px] font-semibold",
                            credit
                              ? "border-[var(--vx-border-hover)] text-foreground"
                              : "border-[var(--vx-border-strong)] text-[var(--vx-ink-dim)]"
                          )}
                        >
                          <span
                            className={cn(
                              "size-[5px] rounded-full",
                              credit ? "bg-[var(--vx-fg)]" : "bg-[var(--vx-muted)]"
                            )}
                          />
                          {credit ? t("billing.tx.credit") : t("billing.tx.debit")}
                        </span>
                      </div>
                      <div className="min-w-0">
                        <div className="truncate text-[13.5px] font-medium">
                          {tx.description || "—"}
                        </div>
                      </div>
                      <div
                        className={cn(
                          "text-right font-mono text-sm",
                          credit ? "text-foreground" : "text-[var(--vx-dim)]"
                        )}
                      >
                        {credit ? "+" : "−"}
                        {fmtMoney(Math.abs(Number(tx.amount) || 0))} {symbol}
                      </div>
                      <div className="text-right font-mono text-[12.5px] text-[var(--vx-ink-faint)]">
                        {fmtMoney(after)} {symbol}
                      </div>
                    </div>
                  );
                })}
              </div>

              <div className="md:hidden">
                {filtered.slice(0, visible).map(({ tx, after }) => {
                  const credit = tx.type === "credit";
                  return (
                    <div
                      key={tx.id}
                      className="flex items-start justify-between gap-3 border-b border-[var(--vx-inset)] px-5 py-4"
                    >
                      <div className="min-w-0">
                        <div className="truncate text-[13.5px] font-medium">
                          {tx.description || "—"}
                        </div>
                        <div className="mt-1 font-mono text-[11px] text-[var(--vx-ink-faint)]">
                          {t("billing.history.row_meta", {
                            date: fmtRowDate(tx.created_at),
                            balance: fmtMoney(after),
                            symbol,
                          })}
                        </div>
                      </div>
                      <div
                        className={cn(
                          "flex-shrink-0 font-mono text-sm",
                          credit ? "text-foreground" : "text-[var(--vx-dim)]"
                        )}
                      >
                        {credit ? "+" : "−"}
                        {fmtMoney(Math.abs(Number(tx.amount) || 0))} {symbol}
                      </div>
                    </div>
                  );
                })}
              </div>

              <div className="flex items-center gap-3.5 px-[26px] py-4">
                <span className="text-[12.5px] text-[var(--vx-ink-faint)]">
                  {t("billing.history.wallet_note", { currency })}
                </span>
                <div className="flex-1" />
                {visible < filtered.length && (
                  <button
                    type="button"
                    onClick={() => setVisible((v) => v + 8)}
                    className="flex items-center gap-2 rounded-full border border-[var(--vx-border-2)] px-4 py-2 text-[12.5px] text-muted-foreground transition-colors hover:border-[var(--vx-border-hover)] hover:text-foreground"
                  >
                    {t("billing.history.show_more")}
                    <ArrowIcon className="size-3.5" />
                  </button>
                )}
              </div>
            </>
          )}
        </div>

        <BillingDocuments currency={currency} />
        <BillingRefunds currency={currency} />
      </div>
      <BillingPaymentDialog id={paymentId} onClose={closePayment} />
    </PageShell>
  );
}
