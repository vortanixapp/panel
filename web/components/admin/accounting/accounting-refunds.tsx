"use client";

import { useState } from "react";
import Link from "next/link";
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
  completeAdminRefundRequest,
  fetchAdminRefundRequests,
  rejectAdminRefundRequest,
  type BalanceRefundRequest,
} from "@/lib/api";
import { money } from "@/lib/accounting-period";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

export function AccountingRefunds() {
  const t = useT();
  const qc = useQueryClient();
  const [status, setStatus] = useState("pending");
  const [busy, setBusy] = useState("");
  const query = useQuery({
    queryKey: ["admin-refund-requests", status],
    queryFn: () => fetchAdminRefundRequests(status),
  });

  const invalidate = () => void qc.invalidateQueries({ queryKey: ["admin-refund-requests"] });

  const run = async (id: string, action: () => Promise<void>) => {
    setBusy(id);
    try {
      await action();
      invalidate();
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : t("admin.accounting.refunds.failed"));
    } finally {
      setBusy("");
    }
  };

  const viaGateway = (request: BalanceRefundRequest) => {
    const left = request.amount - request.refunded;
    if (!window.confirm(t("admin.accounting.refunds.gateway_confirm", { amount: money(left), currency: request.currency }))) {
      return;
    }
    void run(request.id, async () => {
      const res = await completeAdminRefundRequest(request.id, { mode: "gateway" });
      toast.success(res.message || t("admin.accounting.refunds.done"));
    });
  };

  const manual = (request: BalanceRefundRequest) => {
    const reference = window.prompt(t("admin.accounting.refunds.reference_prompt"));
    if (reference === null) return;
    if (!reference.trim()) {
      toast.error(t("admin.accounting.refunds.reference_required"));
      return;
    }
    void run(request.id, async () => {
      await completeAdminRefundRequest(request.id, { mode: "manual", reference: reference.trim() });
      toast.success(t("admin.accounting.refunds.done"));
    });
  };

  const reject = (request: BalanceRefundRequest) => {
    const note = window.prompt(t("admin.accounting.refunds.reject_prompt"));
    if (note === null || !note.trim()) return;
    void run(request.id, async () => {
      await rejectAdminRefundRequest(request.id, note.trim());
      toast.success(t("admin.accounting.refunds.rejected"));
    });
  };

  const requests = query.data?.requests ?? [];

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="max-w-3xl text-sm text-muted-foreground">{t("admin.accounting.refunds.hint")}</p>
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-[200px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="pending">{t("admin.accounting.refunds.filter.pending")}</SelectItem>
            <SelectItem value="all">{t("admin.accounting.refunds.filter.all")}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-lg border bg-card">
        {query.isLoading ? (
          <div className="p-4">
            <Skeleton className="h-32 w-full" />
          </div>
        ) : requests.length === 0 ? (
          <div className="p-8 text-center text-sm text-muted-foreground">{t("admin.accounting.refunds.empty")}</div>
        ) : (
          <div className="divide-y">
            {requests.map((request) => (
              <div key={request.id} className="flex flex-wrap items-start justify-between gap-4 p-4">
                <div className="min-w-0 space-y-1 text-sm">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-medium">{t("admin.accounting.refunds.number", { number: request.number })}</span>
                    <Badge variant={request.status === "pending" ? "secondary" : "outline"}>
                      {t(`admin.accounting.refunds.status.${request.status}`)}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {new Date(request.created_at).toLocaleString(localeTag())}
                    </span>
                  </div>
                  <div>
                    <Link href={`/admin/users/${request.user_id}`} className="hover:underline">
                      {request.user_name}
                    </Link>
                    <span className="text-muted-foreground"> · {request.user_email}</span>
                  </div>
                  <div className="font-mono">
                    {t("admin.accounting.refunds.amount", { amount: money(request.amount), currency: request.currency })}
                    {request.refunded > 0
                      ? ` · ${t("admin.accounting.refunds.refunded", { amount: money(request.refunded) })}`
                      : ""}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {t(`admin.accounting.refunds.method.${request.method}`)}
                    {request.method === "bank"
                      ? ` · ${t("admin.accounting.refunds.bank_details", {
                          recipient: request.recipient,
                          account: request.bank_account,
                          bik: request.bank_bik,
                        })}${request.bank_name ? `, ${request.bank_name}` : ""}`
                      : ""}
                  </div>
                  {request.reason ? <div className="text-xs">{request.reason}</div> : null}
                  {request.status === "pending" ? (
                    <div className="text-xs text-muted-foreground">
                      {t("admin.accounting.refunds.balance", {
                        balance: money(request.balance),
                        gateway: money(request.gateway_refundable),
                      })}
                    </div>
                  ) : null}
                  {request.admin_note || request.reference ? (
                    <div className="text-xs text-muted-foreground">
                      {[request.reference, request.admin_note].filter(Boolean).join(" · ")}
                    </div>
                  ) : null}
                </div>
                {request.status === "pending" ? (
                  <div className="flex flex-wrap gap-2">
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={busy !== "" || request.gateway_refundable <= 0}
                      onClick={() => viaGateway(request)}
                    >
                      {t("admin.accounting.refunds.gateway")}
                    </Button>
                    <Button size="sm" variant="outline" disabled={busy !== ""} onClick={() => manual(request)}>
                      {t("admin.accounting.refunds.manual")}
                    </Button>
                    <Button size="sm" variant="ghost" disabled={busy !== ""} onClick={() => reject(request)}>
                      {t("admin.accounting.refunds.reject")}
                    </Button>
                  </div>
                ) : null}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
