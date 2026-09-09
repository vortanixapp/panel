"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { createTopup, fetchTopupForm } from "@/lib/api";
import {
  BILLING_PROVIDERS,
  formatBillingDate,
  paymentStatusBadgeCls,
  providerDisplayName,
} from "@/lib/billing-providers";
import { formatAmount } from "@/lib/format";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

function resolveApiError(err: unknown): string {
  if (err && typeof err === "object" && "message" in err) {
    return String((err as { message: string }).message);
  }
  return t("billing.topup.create_error");
}

export function BillingTopupPageContent() {
  const t = useT();
  const router = useRouter();
  const [selectedWalletId, setSelectedWalletId] = useState("");
  const [providerCode, setProviderCode] = useState("");
  const [amount, setAmount] = useState("100");
  const [promoCode, setPromoCode] = useState("");
  const [paymentMethodId, setPaymentMethodId] = useState("");
  const [showAll, setShowAll] = useState(false);
  const [error, setError] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.billingTopup(),
    queryFn: fetchTopupForm,
  });

  const wallets = data?.wallets ?? [];
  const payments = data?.payments ?? [];
  const enabledProviders = data?.enabled_providers ?? [];
  const freekassaMethods = data?.freekassa_methods ?? [];
  const providersByCode = useMemo(() => {
    const map = new Map<string, { id: string; code: string; name: string }>();
    for (const p of data?.providers ?? []) {
      map.set(p.code, p);
    }
    return map;
  }, [data?.providers]);

  const activeWalletId =
    wallets.find((w) => w.id === selectedWalletId)?.id ??
    data?.selected_wallet?.id ??
    wallets[0]?.id ??
    "";

  const provider =
    enabledProviders.includes(providerCode) ? providerCode : enabledProviders[0] ?? "";

  const selectedWallet = wallets.find((w) => w.id === activeWalletId);
  const currency = (selectedWallet?.currency || "RUB").toUpperCase();

  const topupMutation = useMutation({
    mutationFn: async () => {
      const providerRow = providersByCode.get(provider);
      if (!providerRow) {
        throw new Error(t("billing.topup.select_provider"));
      }
      return createTopup({
        walletId: activeWalletId || undefined,
        providerId: providerRow.id,
        providerCode: provider,
        amount,
        promoCode: promoCode || undefined,
        paymentMethodId:
          provider === "freekassa" && paymentMethodId ? paymentMethodId : undefined,
      });
    },
    onSuccess: (res) => {
      if (res.redirect_url) {
        window.location.href = res.redirect_url;
        return;
      }
      router.push(`/billing/topup/payment/${res.payment_id}`);
    },
    onError: (err) => setError(resolveApiError(err)),
  });

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    topupMutation.mutate();
  };

  if (isLoading) {
    return (
      <PageShell variant="user">
        <div className="flex items-center justify-center py-24 text-muted-foreground">
          <Loader2 className="mr-2 h-6 w-6 animate-spin" />
          {t("common.loading")}
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="user">
      <div className="space-y-6">
        <div className="relative overflow-hidden rounded-2xl border border-border bg-gradient-to-br from-primary/10 via-card to-card p-6">
          <div className="pointer-events-none absolute top-0 right-0 h-64 w-64 translate-x-1/2 -translate-y-1/2 rounded-full bg-primary/10 blur-3xl" />
          <div className="relative flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-primary to-primary/70 text-primary-foreground shadow-lg shadow-primary/25">
                <i className="ri-wallet-3-line text-2xl" />
              </div>
              <div>
                <h1 className="text-xl font-bold text-foreground lg:text-2xl">
                  {t("billing.topup.heading")}
                </h1>
                <p className="text-sm text-muted-foreground">
                  {t("billing.topup.subtitle")}
                </p>
              </div>
            </div>
            <Link
              href="/billing"
              className="inline-flex items-center justify-center gap-2 rounded-xl border border-border bg-card px-4 py-2 text-xs font-semibold text-muted-foreground transition hover:bg-muted hover:text-foreground"
            >
              <i className="ri-history-line" /> {t("billing.history.title")}
            </Link>
          </div>
        </div>

        <div className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
          <div className="border-b border-border bg-muted/30 px-6 py-4">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold text-foreground">
                {t("billing.topup.my_topups")}
              </h2>
              <span className="rounded-full border border-border bg-muted/40 px-3 py-1 text-xs text-muted-foreground">
                {t("billing.topup.last_20")}
              </span>
            </div>
          </div>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-border">
              <thead className="bg-muted/30">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium tracking-wider text-muted-foreground uppercase">
                    {t("common.date")}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium tracking-wider text-muted-foreground uppercase">
                    {t("common.id")}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium tracking-wider text-muted-foreground uppercase">
                    {t("common.status")}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium tracking-wider text-muted-foreground uppercase">
                    {t("common.amount")}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {payments.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="px-6 py-12 text-center">
                      <div className="flex flex-col items-center gap-2">
                        <i className="ri-file-list-3-line text-4xl text-muted-foreground" />
                        <p className="text-sm text-muted-foreground">
                          {t("billing.topup.no_requests")}
                        </p>
                      </div>
                    </td>
                  </tr>
                ) : (
                  payments.map((p) => (
                    <tr key={p.id} className="transition hover:bg-muted/30">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="text-sm text-foreground">
                          {formatBillingDate(p.created_at)}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <Link
                          href={`/billing/topup/payment/${p.id}`}
                          className="inline-flex items-center gap-1 text-sm font-medium text-primary"
                        >
                          #{String(p.id).slice(0, 8)}
                        </Link>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span
                          className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${paymentStatusBadgeCls(p.status)}`}
                        >
                          {p.status}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="text-sm font-semibold text-foreground">
                          {formatAmount(p.amount)} {(p.currency || "RUB").toUpperCase()}
                        </div>
                        {p.credited_amount != null &&
                          Number(p.credited_amount) > Number(p.amount) && (
                            <div className="text-xs text-emerald-500">
                              +
                              {formatAmount(
                                Number(p.credited_amount) - Number(p.amount)
                              )}{" "}
                              {t("billing.topup.bonus_suffix")}
                            </div>
                          )}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        <form onSubmit={submit}>
          <div className="grid gap-6 lg:grid-cols-3">
            <div className="space-y-6 lg:col-span-2">
              <div className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
                <div className="border-b border-border bg-muted/30 px-6 py-4">
                  <h2 className="text-base font-semibold text-foreground">
                    {t("billing.topup.params_title")}
                  </h2>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {t("billing.topup.params_hint")}
                  </p>
                </div>
                <div className="grid gap-6 p-6 md:grid-cols-3">
                  <div className="space-y-2">
                    <label className="block text-xs font-medium text-muted-foreground">
                      {t("billing.wallet.title")}
                    </label>
                    <select
                      value={activeWalletId}
                      onChange={(e) => setSelectedWalletId(e.target.value)}
                      className="block w-full rounded-xl border border-border bg-muted/50 px-4 py-3 text-sm text-foreground shadow-sm transition outline-none hover:border-primary/30 focus:border-primary/40 focus:ring-2 focus:ring-primary/20"
                    >
                      {wallets.map((w) => (
                        <option key={w.id} value={w.id}>
                          {w.currency.toUpperCase()} — {formatAmount(w.balance)}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div className="space-y-2">
                    <label className="block text-xs font-medium text-muted-foreground">
                      {t("billing.amount_currency", { currency })}
                    </label>
                    <input
                      type="number"
                      min="1"
                      step="0.01"
                      required
                      value={amount}
                      onChange={(e) => setAmount(e.target.value)}
                      className="block w-full rounded-xl border border-border bg-muted/50 px-4 py-3 text-sm text-foreground shadow-sm transition outline-none placeholder:text-muted-foreground hover:border-primary/30 focus:border-primary/40 focus:ring-2 focus:ring-primary/20"
                      placeholder="100.00"
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="block text-xs font-medium text-muted-foreground">
                      {t("billing.topup.promo")}{" "}
                      <span className="text-muted-foreground/70">
                        {t("billing.topup.promo_optional")}
                      </span>
                    </label>
                    <input
                      type="text"
                      value={promoCode}
                      onChange={(e) => setPromoCode(e.target.value)}
                      className="block w-full rounded-xl border border-border bg-muted/50 px-4 py-3 text-sm text-foreground shadow-sm transition outline-none placeholder:text-muted-foreground hover:border-primary/30 focus:border-primary/40 focus:ring-2 focus:ring-primary/20"
                      placeholder="BONUS2024"
                    />
                  </div>
                </div>
              </div>

              <div className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
                <div className="border-b border-border bg-muted/30 px-6 py-4">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <h2 className="text-base font-semibold text-foreground">
                      {t("billing.topup.select_provider")}
                    </h2>
                    <span className="text-xs text-muted-foreground">
                      {t("billing.topup.providers_count", {
                        count: enabledProviders.length,
                      })}
                    </span>
                  </div>
                </div>
                <div className="p-6">
                  <div className="mb-4 flex items-center justify-end">
                    <button
                      type="button"
                      onClick={() => setShowAll(!showAll)}
                      className="inline-flex items-center gap-2 rounded-xl border border-border bg-muted/40 px-3 py-2 text-xs font-semibold text-muted-foreground transition hover:bg-muted hover:text-foreground"
                    >
                      {showAll
                        ? t("billing.topup.hide_disabled")
                        : t("billing.topup.show_all")}
                    </button>
                  </div>
                  <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                    {BILLING_PROVIDERS.filter(
                      (p) => showAll || enabledProviders.includes(p.key)
                    ).map((p) => {
                      const isEnabled = enabledProviders.includes(p.key);
                      const isSelected = provider === p.key;
                      return (
                        <label key={p.key} className="group relative cursor-pointer">
                          <input
                            type="radio"
                            name="provider"
                            value={p.key}
                            checked={isSelected}
                            onChange={() => setProviderCode(p.key)}
                            disabled={!isEnabled}
                            className="peer sr-only"
                          />
                          <div
                            className={`flex h-full flex-col justify-between rounded-xl border border-border bg-card p-4 shadow-sm transition hover:border-primary/30 hover:bg-muted/30 ${isSelected ? "border-primary bg-primary/5" : ""} ${!isEnabled ? "opacity-50 grayscale" : ""}`}
                          >
                            <div className="mb-3 flex items-start justify-between">
                              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted/50 text-xl">
                                💳
                              </div>
                              <div
                                className={`flex h-5 w-5 items-center justify-center rounded-full border-2 transition ${isSelected ? "border-primary bg-primary" : "border-border"}`}
                              >
                                {isSelected && (
                                  <svg
                                    className="h-3 w-3 text-primary-foreground"
                                    fill="currentColor"
                                    viewBox="0 0 12 12"
                                  >
                                    <path d="M10 3L4.5 8.5 2 6" />
                                  </svg>
                                )}
                              </div>
                            </div>
                            <div>
                              <div className="text-sm font-semibold text-foreground">
                                {p.name}
                              </div>
                              <div className="mt-1 text-xs text-muted-foreground">
                                {t(p.descKey)}
                              </div>
                              {!isEnabled && (
                                <div className="mt-2 inline-flex rounded-full border border-border bg-muted/40 px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
                                  {t("billing.topup.provider_disabled")}
                                </div>
                              )}
                            </div>
                          </div>
                        </label>
                      );
                    })}
                  </div>

                  {provider === "freekassa" && (
                    <div className="mt-6 border-t border-border pt-6">
                      <div className="mb-3 text-sm font-semibold text-foreground">
                        {t("billing.topup.freekassa_method")}
                      </div>
                      {freekassaMethods.length === 0 ? (
                        <div className="mb-4 rounded-xl border border-amber-500/20 bg-amber-500/10 p-4 text-xs text-amber-500">
                          {t("billing.topup.freekassa_empty")}
                        </div>
                      ) : (
                        <select
                          value={paymentMethodId}
                          onChange={(e) => setPaymentMethodId(e.target.value)}
                          required
                          className="block w-full rounded-xl border border-border bg-muted/50 px-4 py-3 text-sm text-foreground shadow-sm transition outline-none focus:border-primary/40 focus:ring-2 focus:ring-primary/20"
                        >
                          <option value="">{t("billing.topup.freekassa_select")}</option>
                          {freekassaMethods.map((m) => (
                            <option key={m.id} value={m.id}>
                              {m.name}
                            </option>
                          ))}
                        </select>
                      )}
                    </div>
                  )}

                  {enabledProviders.length === 0 && (
                    <div className="mt-4 rounded-xl border border-amber-500/20 bg-amber-500/10 p-4 text-xs text-amber-600 dark:text-amber-500">
                      {t("billing.topup.no_active_providers")}
                    </div>
                  )}

                  {error && <div className="mt-2 text-xs text-rose-500">{error}</div>}
                </div>
              </div>
            </div>

            <div className="lg:col-span-1">
              <div className="space-y-6 lg:sticky lg:top-6">
                <div className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
                  <div className="border-b border-border bg-muted/30 px-6 py-4">
                    <div className="flex items-center justify-between gap-3">
                      <div className="text-sm font-semibold text-foreground">
                        {t("billing.topup.info")}
                      </div>
                      <div className="text-[11px] text-muted-foreground">
                        {providerDisplayName(provider)}
                      </div>
                    </div>
                  </div>
                  <div className="space-y-4 p-6">
                    <div className="rounded-xl border border-border bg-muted/40 p-4">
                      <div className="text-xs text-muted-foreground">
                        {t("common.amount")}
                      </div>
                      <div className="mt-1 text-xl font-bold text-foreground">
                        {amount || "—"} {currency}
                      </div>
                    </div>
                    <div className="grid gap-3">
                      <div className="flex items-center justify-between text-xs">
                        <span className="text-muted-foreground">
                          {t("billing.topup.promo")}
                        </span>
                        <span className="font-semibold text-foreground">
                          {promoCode || "—"}
                        </span>
                      </div>
                      {provider === "freekassa" && (
                        <div className="flex items-center justify-between text-xs">
                          <span className="text-muted-foreground">
                            {t("billing.topup.freekassa_method_short")}
                          </span>
                          <span className="font-semibold text-foreground">
                            {paymentMethodId ? `ID ${paymentMethodId}` : "—"}
                          </span>
                        </div>
                      )}
                    </div>
                    <div className="rounded-xl border border-border bg-muted/40 p-4 text-xs text-muted-foreground">
                      {t("billing.topup.auto_credit")}
                    </div>
                  </div>
                </div>
                <div className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
                  <div className="p-6">
                    <button
                      type="submit"
                      disabled={
                        topupMutation.isPending ||
                        enabledProviders.length === 0 ||
                        (provider === "freekassa" && freekassaMethods.length === 0)
                      }
                      className="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-primary px-6 py-3 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                      <i className="ri-arrow-right-line" />
                      {topupMutation.isPending
                        ? t("billing.topup.processing")
                        : t("billing.topup.go_to_payment")}
                    </button>
                    <div className="mt-3 text-[11px] text-muted-foreground">
                      {t("billing.topup.promo_note")}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </form>
      </div>
    </PageShell>
  );
}
