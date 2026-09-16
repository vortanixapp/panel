"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Btn, EmptyState, InfoRow, Panel, VX_INPUT } from "@/components/vx/panel-ui";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchAdminServerCard,
  fetchServerCharges,
  setAdminServerExpiry,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const QUICK_DAYS = [1, 3, 7, 14, 30];

export function AdminServerRentTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();

  const [days, setDays] = useState(7);
  const [exactDate, setExactDate] = useState("");

  const cardQuery = useQuery({
    queryKey: queryKeys.adminServerCard(id),
    queryFn: () => fetchAdminServerCard(id),
    enabled: !!id,
  });
  const chargesQuery = useQuery({
    queryKey: queryKeys.adminServerCharges(id),
    queryFn: () => fetchServerCharges(id),
    enabled: !!id,
  });

  const expiryMutation = useMutation({
    mutationFn: (payload: { days?: number; expires_at?: string }) =>
      setAdminServerExpiry(id, payload),
    onSuccess: () => {
      toast.success(t("servers.admin.rent.saved"));
      setExactDate("");
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminServerCard(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminServerAudit(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminServers });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  if (cardQuery.isLoading || !cardQuery.data) {
    return <Skeleton className="h-[420px] w-full rounded-[14px]" />;
  }
  const card = cardQuery.data;
  const charges = chargesQuery.data?.charges ?? [];

  return (
    <div className="grid gap-[18px] lg:grid-cols-2">
      <Panel title={t("servers.admin.rent.section_state")}>
        <InfoRow
          k={t("servers.admin.list.col_expires")}
          v={
            card.expires_at
              ? new Date(card.expires_at).toLocaleString(localeTag())
              : t("servers.admin.card.no_expiry")
          }
        />
        <InfoRow
          k={t("servers.admin.card.rental_period")}
          v={t("servers.admin.card.days", { days: card.rental_period_days })}
        />
        <InfoRow
          k={t("servers.admin.card.auto_renew")}
          v={card.auto_renew ? t("common.yes") : t("common.no")}
        />
        <InfoRow
          k={t("servers.admin.rent.dunning")}
          v={
            card.dunning_stage > 0
              ? t("servers.admin.rent.dunning_stage", { stage: card.dunning_stage })
              : t("servers.admin.rent.dunning_none")
          }
        />
        <InfoRow k={t("servers.admin.list.col_tariff")} v={card.tariff.name ?? "—"} />
        <InfoRow
          k={t("servers.admin.card.trial")}
          v={card.is_trial ? t("common.yes") : t("common.no")}
        />
      </Panel>

      <Panel title={t("servers.admin.rent.section_grant")}>
        <p className="mb-3 text-[12.5px] text-[var(--vx-muted)]">
          {t("servers.admin.rent.grant_hint")}
        </p>

        <div className="mb-4 flex flex-wrap items-center gap-2">
          {QUICK_DAYS.map((n) => (
            <Btn
              key={n}
              size="sm"
              disabled={expiryMutation.isPending}
              onClick={() => expiryMutation.mutate({ days: n })}
            >
              +{n}
            </Btn>
          ))}
        </div>

        <div className="mb-4 flex flex-wrap items-center gap-2">
          <input
            className={cn(VX_INPUT, "h-[34px] w-[110px]")}
            type="number"
            min={-3650}
            max={3650}
            value={days}
            onChange={(e) => setDays(Number(e.target.value) || 0)}
          />
          <Btn
            tone="primary"
            disabled={expiryMutation.isPending || days === 0}
            onClick={() => expiryMutation.mutate({ days })}
          >
            {t("servers.admin.rent.apply_days")}
          </Btn>
          <span className="text-[11.5px] text-[var(--vx-faint)]">
            {t("servers.admin.rent.negative_hint")}
          </span>
        </div>

        <div className="border-t border-[var(--vx-border)] pt-4">
          <div className="mb-2 text-[12.5px] text-[var(--vx-muted)]">
            {t("servers.admin.rent.exact_date")}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <input
              className={cn(VX_INPUT, "h-[34px]")}
              type="datetime-local"
              value={exactDate}
              onChange={(e) => setExactDate(e.target.value)}
            />
            <Btn
              disabled={expiryMutation.isPending || !exactDate}
              onClick={() =>
                expiryMutation.mutate({
                  expires_at: new Date(exactDate).toISOString(),
                })
              }
            >
              {t("common.save")}
            </Btn>
          </div>
        </div>
      </Panel>

      <div className="lg:col-span-2">
        <Panel
          title={t("servers.admin.rent.section_charges")}
          aside={
            <span className="text-[12px] text-[var(--vx-muted)]">
              {t("servers.admin.rent.charges_total", {
                total: (chargesQuery.data?.total ?? 0).toFixed(2),
              })}
            </span>
          }
        >
          {chargesQuery.isLoading ? (
            <Skeleton className="h-[160px] w-full rounded-[10px]" />
          ) : charges.length === 0 ? (
            <EmptyState>{t("servers.admin.rent.charges_empty")}</EmptyState>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[520px] border-collapse text-[12.5px]">
                <thead>
                  <tr className="border-b border-[var(--vx-border)] text-left text-[var(--vx-muted)]">
                    <th className="py-2 pr-3 font-medium">
                      {t("servers.admin.audit.col_when")}
                    </th>
                    <th className="py-2 pr-3 font-medium">
                      {t("servers.admin.rent.charge_source")}
                    </th>
                    <th className="py-2 pr-3 font-medium">{t("common.description")}</th>
                    <th className="py-2 text-right font-medium">
                      {t("servers.admin.rent.charge_amount")}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {charges.map((charge, index) => (
                    <tr
                      key={`${charge.created_at}-${index}`}
                      className="border-b border-[var(--vx-elevated)]"
                    >
                      <td className="py-2 pr-3 whitespace-nowrap text-[var(--vx-muted)]">
                        {new Date(charge.created_at).toLocaleString(localeTag())}
                      </td>
                      <td className="py-2 pr-3">{charge.source || "—"}</td>
                      <td className="py-2 pr-3">{charge.description || "—"}</td>
                      <td className="py-2 text-right font-mono tabular-nums">
                        {charge.amount.toFixed(2)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Panel>
      </div>
    </div>
  );
}
