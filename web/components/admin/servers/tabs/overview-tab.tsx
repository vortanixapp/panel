"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { EmptyState, InfoRow, Panel } from "@/components/vx/panel-ui";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchAdminServerAudit, fetchAdminServerCard } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { localeTag, type TranslateFn } from "@/lib/i18n";
import { serverPath } from "@/lib/panel-paths";
import { useT } from "@/hooks/use-translations";
import { useServerDetail } from "@/hooks/use-queries";
import { getServerStatus } from "@/lib/server-status";
import {
  auditActionLabel,
  cpuLimitLabel,
  ownerStatusLabel,
} from "@/components/admin/servers/labels";

function formatDate(value: string | null | undefined, fallback: string): string {
  if (!value) return fallback;
  return new Date(value).toLocaleString(localeTag());
}

function formatMb(value: unknown, t: TranslateFn): string | null {
  const num = Number(value);
  if (!Number.isFinite(num) || num <= 0) return null;
  if (num >= 1024) {
    const gb = num / 1024;
    return t("servers.unit.gb", { value: Number.isInteger(gb) ? gb : gb.toFixed(1) });
  }
  return t("servers.unit.mb", { value: num });
}

export function AdminServerOverviewTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const { data: server } = useServerDetail(id);
  const cardQuery = useQuery({
    queryKey: queryKeys.adminServerCard(id),
    queryFn: () => fetchAdminServerCard(id),
    enabled: !!id,
  });
  const auditQuery = useQuery({
    queryKey: [...queryKeys.adminServerAudit(id), "recent"],
    queryFn: () => fetchAdminServerAudit(id, { limit: 8 }),
    enabled: !!id,
  });

  if (cardQuery.isLoading || !cardQuery.data) {
    return <Skeleton className="h-[420px] w-full rounded-[14px]" />;
  }
  const card = cardQuery.data;
  const dash = "—";
  const limits = card.limits as Record<string, unknown>;

  return (
    <div className="grid gap-[18px] lg:grid-cols-2">
      <Panel title={t("servers.admin.card.section_state")}>
        <InfoRow k={t("common.status")} v={server ? getServerStatus(server).label : dash} />
        <InfoRow
          k={t("servers.admin.card.provisioning")}
          v={t(`servers.admin.card.prov_${card.provisioning_status}`)}
        />
        <InfoRow
          k={t("servers.admin.card.created")}
          v={formatDate(card.created_at, dash)}
        />
        <InfoRow
          k={t("servers.admin.list.col_expires")}
          v={formatDate(card.expires_at, t("servers.admin.card.no_expiry"))}
        />
        <InfoRow
          k={t("servers.admin.card.blocked")}
          v={
            card.is_blocked
              ? `${formatDate(card.blocked_at, dash)}${card.blocked_reason ? ` · ${card.blocked_reason}` : ""}`
              : t("common.no")
          }
        />
        <InfoRow
          k={t("servers.admin.card.suspended")}
          v={card.suspended_at ? formatDate(card.suspended_at, dash) : t("common.no")}
        />
        <InfoRow
          k={t("servers.admin.card.trial")}
          v={card.is_trial ? t("common.yes") : t("common.no")}
        />
      </Panel>

      <Panel title={t("servers.admin.card.section_owner")}>
        {card.owner ? (
          <>
            <InfoRow
              k={t("common.email")}
              v={
                <Link
                  href={`/admin/users/${card.owner.id}`}
                  className="text-primary hover:underline"
                >
                  {card.owner.email}
                </Link>
              }
            />
            <InfoRow k={t("common.status")} v={ownerStatusLabel(card.owner.status)} />
            <InfoRow
              k={t("servers.admin.owner.registered")}
              v={formatDate(card.owner.created_at, dash)}
            />
            <InfoRow
              k={t("servers.admin.owner.balance")}
              v={
                card.owner.balances.length === 0
                  ? dash
                  : card.owner.balances
                      .map((b) => `${b.balance.toFixed(2)} ${b.currency}`)
                      .join(" · ")
              }
            />
            <InfoRow
              k={t("servers.admin.owner.servers")}
              v={String(card.owner.server_count)}
            />
          </>
        ) : (
          <EmptyState>{t("servers.admin.owner.none")}</EmptyState>
        )}
      </Panel>

      <Panel title={t("servers.admin.card.section_resources")}>
        <InfoRow k={t("servers.admin.list.col_tariff")} v={card.tariff.name ?? dash} />
        <InfoRow k="RAM" v={formatMb(limits.memory_mb, t) ?? dash} />
        <InfoRow k={t("servers.admin.card.disk")} v={formatMb(limits.disk_mb, t) ?? dash} />
        <InfoRow k="CPU" v={cpuLimitLabel(limits) ?? dash} />
        <InfoRow k={t("servers.admin.card.slots")} v={limits.slots ? String(limits.slots) : dash} />
        <InfoRow
          k={t("servers.admin.card.rental_period")}
          v={t("servers.admin.card.days", { days: card.rental_period_days })}
        />
        <InfoRow
          k={t("servers.admin.card.auto_renew")}
          v={card.auto_renew ? t("common.yes") : t("common.no")}
        />
      </Panel>

      <Panel title={t("servers.admin.card.section_links")}>
        <InfoRow k={t("servers.admin.card.node")} v={card.node.name || dash} />
        <InfoRow k={t("servers.admin.card.friends")} v={String(card.counts.friends)} />
        <InfoRow k={t("servers.admin.card.backups")} v={String(card.counts.backups)} />
        <InfoRow
          k={t("servers.admin.card.tickets")}
          v={
            card.counts.tickets > 0 ? (
              <Link href="/admin/support" className="text-primary hover:underline">
                {card.counts.tickets}
              </Link>
            ) : (
              "0"
            )
          }
        />
        <InfoRow
          k={t("servers.admin.card.abuse")}
          v={
            card.counts.abuse_cases > 0 ? (
              <Link href="/admin/abuse" className="text-primary hover:underline">
                {card.counts.abuse_cases}
              </Link>
            ) : (
              "0"
            )
          }
        />
        <InfoRow k={t("servers.admin.card.billing_source")} v={card.billing_source} />
      </Panel>

      <div className="lg:col-span-2">
        <Panel
          title={t("servers.admin.card.section_recent")}
          aside={
            <Link
              href={serverPath("/admin", id, "/audit")}
              className="text-[12px] text-primary hover:underline"
            >
              {t("servers.admin.card.open_audit")}
            </Link>
          }
        >
          {auditQuery.data && auditQuery.data.entries.length > 0 ? (
            <div className="flex flex-col">
              {auditQuery.data.entries.map((entry) => (
                <InfoRow
                  key={entry.id}
                  k={new Date(entry.created_at).toLocaleString(localeTag())}
                  v={
                    <span>
                      {auditActionLabel(entry.action)}
                      {entry.actor_email ? (
                        <span className="text-[var(--vx-muted)]"> · {entry.actor_email}</span>
                      ) : null}
                    </span>
                  }
                />
              ))}
            </div>
          ) : (
            <EmptyState>{t("servers.admin.audit.empty")}</EmptyState>
          )}
        </Panel>
      </div>
    </div>
  );
}
