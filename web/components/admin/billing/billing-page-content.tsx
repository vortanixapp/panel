"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
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
  adjustAdminUserBalance,
  fetchAdminBilling,
  fetchRefundSupport,
  openPaymentReceipt,
  refundAdminPayment,
  type AdminPayment,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

const STATUS_OPTIONS = [
  { value: "all", labelKey: "admin.billing.status.all" },
  { value: "completed", labelKey: "admin.billing.status.completed" },
  { value: "pending", labelKey: "admin.billing.status.pending" },
  { value: "processing", labelKey: "admin.billing.status.processing" },
  { value: "failed", labelKey: "admin.billing.status.failed" },
  { value: "cancelled", labelKey: "admin.billing.status.cancelled" },
  { value: "refunded", labelKey: "admin.billing.status.refunded" },
];

const PERIOD_OPTIONS = [
  { value: "0", labelKey: "admin.billing.period.all" },
  { value: "1", labelKey: "admin.billing.period.day" },
  { value: "7", labelKey: "admin.billing.period.week" },
  { value: "30", labelKey: "admin.billing.period.month" },
];

const PAID = new Set(["completed", "success", "succeeded", "paid"]);

function fmtDateTime(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString(localeTag());
}

export function BillingPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const [status, setStatus] = useState("all");
  const [days, setDays] = useState("0");
  const [q, setQ] = useState("");
  const [appliedQ, setAppliedQ] = useState("");

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ["admin-billing", status, days, appliedQ],
    queryFn: async () =>
      (
        await fetchAdminBilling({
          limit: 200,
          status: status === "all" ? undefined : status,
          days: Number(days) || undefined,
          q: appliedQ || undefined,
        })
      ).payments,
  });
  const payments = data ?? [];

  const total = payments
    .filter((p) => PAID.has(p.status))
    .reduce((sum, p) => sum + p.amount, 0);

  const refundSupport = useQuery({
    queryKey: ["refund-support"],
    queryFn: fetchRefundSupport,
    staleTime: 60 * 60 * 1000,
  });

  const [refundFor, setRefundFor] = useState<AdminPayment | null>(null);
  const [refundAmount, setRefundAmount] = useState("");
  const [refundReason, setRefundReason] = useState("");

  const refundMut = useMutation({
    mutationFn: () =>
      refundAdminPayment(refundFor!.id, {
        amount: Number(refundAmount) || undefined,
        reason: refundReason,
      }),
    onSuccess: (res) => {
      toast.success(
        t("admin.billing.refund_done", {
          amount: res.refunded_amount.toFixed(2),
        })
      );
      setRefundFor(null);
      setRefundAmount("");
      setRefundReason("");
      qc.invalidateQueries({ queryKey: ["admin-billing"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.billing.refund_failed")),
  });

  const [adjustFor, setAdjustFor] = useState<{
    userId: string;
    email: string;
    currency: string;
  } | null>(null);
  const [amount, setAmount] = useState("");
  const [comment, setComment] = useState("");

  const adjustMut = useMutation({
    mutationFn: () =>
      adjustAdminUserBalance(adjustFor!.userId, {
        amount: Number(amount),
        currency: adjustFor!.currency,
        comment,
      }),
    onSuccess: (res) => {
      toast.success(
        t("admin.billing.balance_changed", {
          balance: `${res.balance} ${res.currency}`,
        })
      );
      setAdjustFor(null);
      setAmount("");
      setComment("");
      qc.invalidateQueries({ queryKey: ["admin-billing"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.billing.balance_failed")),
  });

  return (
    <PageShell variant="admin">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">
          {t("admin.billing.title")}
        </h1>
        <p className="text-sm text-muted-foreground">
          {t("admin.billing.subtitle")}
        </p>
      </div>

      <div className="mb-4 flex flex-wrap items-end gap-3">
        <div className="space-y-1">
          <Label>{t("common.status")}</Label>
          <Select value={status} onValueChange={setStatus}>
            <SelectTrigger className="w-48">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {STATUS_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {t(o.labelKey)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label>{t("common.period")}</Label>
          <Select value={days} onValueChange={setDays}>
            <SelectTrigger className="w-44">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PERIOD_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {t(o.labelKey)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label>{t("admin.billing.search_label")}</Label>
          <Input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") setAppliedQ(q.trim());
            }}
            placeholder="user@example.com"
            className="w-72"
          />
        </div>
        <Button variant="outline" onClick={() => setAppliedQ(q.trim())}>
          {t("common.search")}
        </Button>
        <div className="ml-auto text-sm text-muted-foreground">
          {t("admin.billing.paid_total")} <b>{total.toFixed(2)}</b>
        </div>
      </div>

      {refundFor && (
        <div className="mb-6 space-y-3 rounded-lg border bg-card p-4">
          <div className="text-sm font-medium">
            {t("admin.billing.refund_title", {
              user: refundFor.user_email || refundFor.user_id,
            })}
          </div>
          <p className="text-xs text-muted-foreground">
            {t("admin.billing.refund_hint")}
          </p>
          <div className="flex flex-wrap items-end gap-3">
            <div className="space-y-1">
              <Label>
                {t("admin.billing.amount_label", {
                  currency: refundFor.currency || "RUB",
                })}
              </Label>
              <Input
                type="number"
                step="0.01"
                value={refundAmount}
                onChange={(e) => setRefundAmount(e.target.value)}
                className="w-40"
              />
            </div>
            <div className="flex-1 space-y-1">
              <Label>{t("admin.maintenance.reason")}</Label>
              <Input
                value={refundReason}
                onChange={(e) => setRefundReason(e.target.value)}
                placeholder={t("admin.billing.refund_reason_placeholder")}
              />
            </div>
            <Button
              onClick={() => refundMut.mutate()}
              disabled={refundMut.isPending || !Number(refundAmount)}
            >
              {t("admin.billing.refund_action")}
            </Button>
            <Button variant="ghost" onClick={() => setRefundFor(null)}>
              {t("common.cancel")}
            </Button>
          </div>
        </div>
      )}

      {adjustFor && (
        <div className="mb-6 space-y-3 rounded-lg border bg-card p-4">
          <div className="text-sm font-medium">
            {t("admin.billing.adjust_title", {
              user: adjustFor.email || adjustFor.userId,
            })}
          </div>
          <div className="flex flex-wrap items-end gap-3">
            <div className="space-y-1">
              <Label>
                {t("admin.billing.amount_label", {
                  currency: adjustFor.currency,
                })}
              </Label>
              <Input
                type="number"
                step="0.01"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                placeholder={t("admin.billing.amount_placeholder")}
                className="w-40"
              />
            </div>
            <div className="flex-1 space-y-1">
              <Label>{t("admin.maintenance.reason")}</Label>
              <Input
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                placeholder={t("admin.billing.adjust_reason_placeholder")}
              />
            </div>
            <Button
              onClick={() => adjustMut.mutate()}
              disabled={
                adjustMut.isPending || !Number(amount) || !comment.trim()
              }
            >
              {t("common.apply")}
            </Button>
            <Button variant="ghost" onClick={() => setAdjustFor(null)}>
              {t("common.cancel")}
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">
            {t("admin.billing.adjust_hint")}
          </p>
        </div>
      )}

      <div className="rounded-lg border bg-card">
        {isLoading ? (
          <div className="space-y-3 p-6">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : payments.length === 0 ? (
          <div className="p-10 text-center text-sm text-muted-foreground">
            {t("admin.billing.empty")}
          </div>
        ) : (
          <div className={isFetching ? "divide-y opacity-60" : "divide-y"}>
            {payments.map((p) => (
              <div
                key={p.id}
                className="flex flex-wrap items-center justify-between gap-4 p-4"
              >
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-medium">
                      {p.amount.toFixed(2)} {p.currency || "RUB"}
                    </span>
                    <span className="text-sm text-muted-foreground">
                      {p.user_email || p.user_id}
                    </span>
                  </div>
                  <div className="mt-1 font-mono text-xs text-muted-foreground">
                    {p.provider || "—"} ·{" "}
                    {p.provider_payment_id || t("admin.billing.no_id")} ·{" "}
                    {fmtDateTime(p.created_at)}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {PAID.has(p.status) ? (
                    <Badge className="bg-emerald-500/10 text-emerald-600 ring-1 ring-inset ring-emerald-500/20">
                      {p.status}
                    </Badge>
                  ) : (
                    <Badge variant="outline">{p.status}</Badge>
                  )}
                  {PAID.has(p.status) ? (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() =>
                        openPaymentReceipt(p.id).catch((e: Error) =>
                          toast.error(e.message)
                        )
                      }
                    >
                      {t("admin.billing.receipt")}
                    </Button>
                  ) : null}
                  {PAID.has(p.status) &&
                  refundSupport.data?.providers[p.provider ?? ""] &&
                  (p.refunded_amount ?? 0) < p.amount ? (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setRefundFor(p);
                        setRefundAmount(
                          (p.amount - (p.refunded_amount ?? 0)).toFixed(2)
                        );
                        setRefundReason("");
                      }}
                    >
                      {t("admin.billing.refund")}
                    </Button>
                  ) : null}
                  {(p.refunded_amount ?? 0) > 0 ? (
                    <Badge variant="outline">
                      {t("admin.billing.refunded", {
                        amount: (p.refunded_amount ?? 0).toFixed(2),
                      })}
                    </Badge>
                  ) : null}
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      setAdjustFor({
                        userId: p.user_id,
                        email: p.user_email || "",
                        currency: p.currency || "RUB",
                      });
                      setAmount("");
                      setComment("");
                    }}
                  >
                    {t("admin.billing.adjust")}
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </PageShell>
  );
}
