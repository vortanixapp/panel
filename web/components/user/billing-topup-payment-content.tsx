"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { toast } from "sonner";
import {
  fetchPayment,
  openPaymentInvoice,
  openPaymentReceipt,
  type PaymentDetail,
} from "@/lib/api";
import { formatAmount } from "@/lib/format";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

function isSuccessStatus(status?: string) {
  return status === "completed" || status === "success" || status === "succeeded";
}

function isFailureStatus(status?: string) {
  return status === "failed" || status === "cancelled";
}

function isPendingStatus(status?: string) {
  return status === "pending" || status === "processing";
}

function BankDetails({ payment }: { payment: PaymentDetail }) {
  const t = useT();
  const info = payment.instructions ?? {};
  const rows = [
    { label: t("billing.payment.bank_name"), value: info.bank_name },
    { label: t("billing.payment.bank_holder"), value: info.account_holder },
    { label: t("billing.payment.bank_account"), value: info.account_number },
    { label: t("billing.payment.bank_swift"), value: info.swift_bic },
    {
      label: t("billing.payment.bank_reference"),
      value: info.reference
        ? t("billing.payment.bank_reference_value", { reference: info.reference })
        : undefined,
    },
  ].filter((row) => row.value);

  return (
    <div className="space-y-3 rounded-xl border border-border bg-muted/30 p-4 text-left">
      <div className="text-sm font-semibold text-foreground">
        {t("billing.payment.bank_title")}
      </div>
      <dl className="space-y-2">
        {rows.map((row) => (
          <div key={row.label} className="flex flex-col gap-0.5">
            <dt className="text-[11px] text-muted-foreground">{row.label}</dt>
            <dd className="font-mono text-sm break-all text-foreground select-all">{row.value}</dd>
          </div>
        ))}
      </dl>
      {info.instructions && (
        <p className="text-xs whitespace-pre-line text-muted-foreground">{info.instructions}</p>
      )}
      <p className="text-xs text-muted-foreground">{t("billing.payment.bank_hint")}</p>
    </div>
  );
}

export function BillingTopupPaymentContent({ id }: { id: string }) {
  const t = useT();
  const { data: payment, isLoading } = useQuery({
    queryKey: queryKeys.billingPayment(id),
    queryFn: () => fetchPayment(id),
    enabled: Boolean(id),
    refetchInterval: (query) =>
      isPendingStatus(query.state.data?.status) ? 5000 : false,
  });

  const chargeDiffers =
    payment?.charge_amount != null &&
    (Number(payment.charge_amount) !== Number(payment.amount) ||
      (payment.charge_currency || "").toUpperCase() !== (payment.currency || "").toUpperCase());

  return (
    <PageShell variant="user">
      <div className="flex min-h-[60vh] items-center justify-center p-4">
        {isLoading ? (
          <div className="flex items-center text-muted-foreground">
            <Loader2 className="mr-2 h-6 w-6 animate-spin" />
            {t("common.loading")}
          </div>
        ) : (
          <div className="w-full max-w-md">
            <div className="space-y-6 rounded-2xl border border-border bg-card p-8 text-center shadow-xl">
              {isSuccessStatus(payment?.status) ? (
                <>
                  <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-emerald-500/10">
                    <i className="ri-check-double-line text-3xl text-emerald-500" />
                  </div>
                  <h1 className="text-2xl font-bold text-foreground">
                    {t("billing.payment.success_title")}
                  </h1>
                  <p className="text-sm text-muted-foreground">
                    {t("billing.payment.success_text")}
                  </p>
                  {payment?.amount != null && (
                    <p className="text-lg font-semibold text-foreground">
                      +{formatAmount(payment.credited_amount ?? payment.amount)}{" "}
                      {(payment.currency || "RUB").toUpperCase()}
                    </p>
                  )}
                  <button
                    type="button"
                    className="text-sm text-primary underline-offset-4 hover:underline"
                    onClick={() =>
                      openPaymentReceipt(id).catch((e: Error) => toast.error(e.message))
                    }
                  >
                    {t("billing.payment.open_receipt")}
                  </button>
                </>
              ) : isFailureStatus(payment?.status) ? (
                <>
                  <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-rose-500/10">
                    <i className="ri-close-circle-line text-3xl text-rose-500" />
                  </div>
                  <h1 className="text-2xl font-bold text-foreground">
                    {t("billing.payment.failed_title")}
                  </h1>
                  <p className="text-sm text-muted-foreground">
                    {t("billing.payment.failed_text")}
                  </p>
                </>
              ) : payment ? (
                <>
                  <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-amber-500/10">
                    <i className="ri-time-line text-3xl text-amber-500" />
                  </div>
                  <h1 className="text-2xl font-bold text-foreground">
                    {t("billing.payment.pending_title")}
                  </h1>
                  <p className="text-sm text-muted-foreground">
                    {t("billing.payment.pending_text", { status: payment.status })}
                  </p>
                  <div className="space-y-1">
                    <p className="text-sm font-medium text-foreground">
                      {formatAmount(payment.amount)} {(payment.currency || "RUB").toUpperCase()}
                      {payment.provider_name && (
                        <span className="text-muted-foreground">
                          {" "}
                          {t("billing.payment.via", { provider: payment.provider_name })}
                        </span>
                      )}
                    </p>
                    {chargeDiffers && (
                      <p className="text-xs text-muted-foreground">
                        {t("billing.payment.to_pay", {
                          amount: formatAmount(payment.charge_amount),
                          currency: (payment.charge_currency || "").toUpperCase(),
                        })}
                      </p>
                    )}
                  </div>
                  {payment.instructions && <BankDetails payment={payment} />}
                  {payment.provider === "bank" && (
                    <button
                      type="button"
                      className="inline-flex w-full items-center justify-center rounded-xl border border-border py-3 text-sm font-semibold text-foreground transition hover:bg-muted"
                      onClick={() =>
                        openPaymentInvoice(id).catch((e: Error) => toast.error(e.message))
                      }
                    >
                      {t("billing.payment.invoice")}
                    </button>
                  )}
                  {payment.checkout_url && (
                    <a
                      href={payment.checkout_url}
                      className="inline-flex w-full items-center justify-center rounded-xl border border-primary/40 py-3 text-sm font-semibold text-primary transition hover:bg-primary/5"
                    >
                      {t("billing.payment.continue")}
                    </a>
                  )}
                </>
              ) : (
                <>
                  <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-muted">
                    <i className="ri-question-line text-3xl text-muted-foreground" />
                  </div>
                  <h1 className="text-2xl font-bold text-foreground">
                    {t("billing.payment.not_found_title")}
                  </h1>
                  <p className="text-sm text-muted-foreground">
                    {t("billing.payment.not_found_text")}
                  </p>
                </>
              )}
              <div className="flex flex-col gap-3 pt-4">
                <Link
                  href="/billing"
                  className="w-full rounded-xl bg-primary py-3 text-center text-sm font-semibold text-primary-foreground transition-all hover:shadow-lg hover:shadow-primary/25"
                >
                  {t("billing.payment.to_billing")}
                </Link>
                <Link href="/dashboard" className="text-sm text-primary hover:underline">
                  {t("billing.payment.to_dashboard")}
                </Link>
              </div>
            </div>
          </div>
        )}
      </div>
    </PageShell>
  );
}
