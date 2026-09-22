"use client";

import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import {
  fetchPayment,
  openPaymentInvoice,
  openPaymentReceipt,
  type PaymentDetail,
  submitCheckoutForm,
} from "@/lib/api";
import { formatAmount } from "@/lib/format";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

const SUCCESS = new Set(["completed", "success", "succeeded", "paid"]);
const FAILURE = new Set(["failed", "cancelled"]);
const PENDING = new Set(["pending", "processing"]);

export function paymentTone(status?: string) {
  if (status && SUCCESS.has(status)) return "success" as const;
  if (status && FAILURE.has(status)) return "failure" as const;
  if (status && PENDING.has(status)) return "pending" as const;
  return "other" as const;
}

export function paymentStatusLabel(status: string) {
  const key = SUCCESS.has(status) ? "completed" : status;
  const label = t(`billing.payment.status.${key}`);
  return label === `billing.payment.status.${key}` ? status : label;
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
    <div className="space-y-3 rounded-[14px] border border-[var(--vx-border-2)] bg-[var(--vx-card-2)] p-4 text-left">
      <div className="text-sm font-semibold">{t("billing.payment.bank_title")}</div>
      <dl className="space-y-2">
        {rows.map((row) => (
          <div key={row.label} className="flex flex-col gap-0.5">
            <dt className="text-[11px] text-muted-foreground">{row.label}</dt>
            <dd className="font-mono text-sm break-all select-all">{row.value}</dd>
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

const ICONS = {
  success: { icon: "ri-check-double-line", box: "bg-emerald-500/10 text-emerald-500" },
  failure: { icon: "ri-close-circle-line", box: "bg-rose-500/10 text-rose-500" },
  pending: { icon: "ri-time-line", box: "bg-amber-500/10 text-amber-500" },
  other: { icon: "ri-question-line", box: "bg-[var(--vx-tint)] text-muted-foreground" },
};

export function BillingPaymentDialog({ id, onClose }: { id: string | null; onClose: () => void }) {
  const t = useT();
  const qc = useQueryClient();
  const { data: payment, isLoading } = useQuery({
    queryKey: queryKeys.billingPayment(id ?? ""),
    queryFn: () => fetchPayment(id ?? ""),
    enabled: Boolean(id),
    retry: false,
    refetchInterval: (query) => (paymentTone(query.state.data?.status) === "pending" ? 5000 : false),
  });

  const tone = payment ? paymentTone(payment.status) : "other";

  useEffect(() => {
    if (tone === "success") {
      void qc.invalidateQueries({ queryKey: ["billing"] });
      void qc.invalidateQueries({ queryKey: queryKeys.billingTopup() });
      void qc.invalidateQueries({ queryKey: ["dashboard"] });
    }
  }, [tone, qc]);

  const currency = (payment?.currency || "RUB").toUpperCase();
  const chargeDiffers =
    payment?.charge_amount != null &&
    (Number(payment.charge_amount) !== Number(payment.amount) ||
      (payment.charge_currency || "").toUpperCase() !== currency);

  const title = !payment
    ? t("billing.payment.not_found_title")
    : tone === "success"
      ? t("billing.payment.success_title")
      : tone === "failure"
        ? t("billing.payment.failed_title")
        : t("billing.payment.pending_title");

  const text = !payment
    ? t("billing.payment.not_found_text")
    : tone === "success"
      ? t("billing.payment.success_text")
      : tone === "failure"
        ? t("billing.payment.failed_text")
        : t("billing.payment.pending_text", { status: paymentStatusLabel(payment.status) });

  return (
    <Dialog open={Boolean(id)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="font-panel max-h-[90vh] overflow-y-auto sm:max-w-md">
        {isLoading ? (
          <div className="flex items-center justify-center py-10 text-muted-foreground">
            <Loader2 className="mr-2 size-5 animate-spin" />
            <DialogTitle className="text-sm font-normal">{t("common.loading")}</DialogTitle>
          </div>
        ) : (
          <div className="space-y-4 text-center">
            <div className={cn("mx-auto flex size-14 items-center justify-center rounded-2xl", ICONS[tone].box)}>
              <i className={cn(ICONS[tone].icon, "text-[26px]")} />
            </div>
            <div className="space-y-1.5">
              <DialogTitle className="text-xl font-bold">{title}</DialogTitle>
              <DialogDescription className="text-sm">{text}</DialogDescription>
            </div>

            {payment && (
              <div className="space-y-1">
                <p className="font-mono text-lg font-semibold">
                  {tone === "success" ? "+" : ""}
                  {formatAmount(
                    tone === "success" ? payment.credited_amount ?? payment.amount : payment.amount
                  )}{" "}
                  {currency}
                  {payment.provider_name && tone !== "success" && (
                    <span className="font-sans text-sm font-normal text-muted-foreground">
                      {" "}
                      {t("billing.payment.via", { provider: payment.provider_name })}
                    </span>
                  )}
                </p>
                {chargeDiffers && tone === "pending" && (
                  <p className="text-xs text-muted-foreground">
                    {t("billing.payment.to_pay", {
                      amount: formatAmount(payment.charge_amount),
                      currency: (payment.charge_currency || "").toUpperCase(),
                    })}
                  </p>
                )}
              </div>
            )}

            {payment && tone === "pending" && payment.instructions && <BankDetails payment={payment} />}

            <div className="flex flex-col gap-2">
              {payment && tone === "pending" && payment.checkout_form?.action && (
                <button
                  type="button"
                  className="vx-btn rounded-full py-3 text-center text-sm font-semibold"
                  onClick={() => payment.checkout_form && submitCheckoutForm(payment.checkout_form)}
                >
                  {t("billing.payment.continue")}
                </button>
              )}
              {payment && tone === "pending" && !payment.checkout_form?.action && payment.checkout_url && (
                <a
                  href={payment.checkout_url}
                  className="vx-btn rounded-full py-3 text-center text-sm font-semibold"
                >
                  {t("billing.payment.continue")}
                </a>
              )}
              {payment && tone === "pending" && payment.provider === "bank" && (
                <button
                  type="button"
                  className="rounded-full border border-[var(--vx-border-2)] py-3 text-sm font-semibold transition-colors hover:border-[var(--vx-border-hover)]"
                  onClick={() => openPaymentInvoice(payment.id).catch((e: Error) => toast.error(e.message))}
                >
                  {t("billing.payment.invoice")}
                </button>
              )}
              {payment && tone === "success" && (
                <button
                  type="button"
                  className="rounded-full border border-[var(--vx-border-2)] py-3 text-sm font-semibold transition-colors hover:border-[var(--vx-border-hover)]"
                  onClick={() => openPaymentReceipt(payment.id).catch((e: Error) => toast.error(e.message))}
                >
                  {t("billing.payment.open_receipt")}
                </button>
              )}
              <button
                type="button"
                onClick={onClose}
                className="rounded-full py-2.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
              >
                {t("common.close")}
              </button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
