"use client";

import { InfoRow, Notice, Panel, VX_MUTED, btnClass } from "@/components/vx/panel-ui";
import type { DashboardServer } from "@/lib/api";
import { dateLocaleTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

export function isWhmcsBilled(server: DashboardServer | null | undefined): boolean {
  return server?.billing_source === "whmcs";
}

export function whmcsDueLabel(value: string | null | undefined): string {
  if (!value) return "—";
  const date = new Date(`${value}T00:00:00`);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString(dateLocaleTag());
}

export function WhmcsBillingPanel({ server }: { server: DashboardServer }) {
  const t = useT();
  const whmcs = server.whmcs;

  return (
    <Panel title={t("servers.whmcs.title")} bodyClassName="flex flex-col gap-3.5 p-[18px]">
      <p className={cn("text-[12.5px] leading-[1.6]", VX_MUTED)}>{t("servers.whmcs.desc")}</p>
      {whmcs?.status === "suspended" && <Notice>{t("servers.whmcs.suspended_notice")}</Notice>}
      <div>
        <InfoRow k={t("common.tariff")} v={server.tariff?.name ?? "—"} />
        {whmcs && <InfoRow k={t("servers.whmcs.service")} v={`#${whmcs.service_id}`} />}
        {whmcs && (
          <InfoRow k={t("servers.whmcs.status")} v={t(`servers.whmcs.status_${whmcs.status}`)} />
        )}
        <InfoRow k={t("servers.whmcs.next_due")} v={whmcsDueLabel(whmcs?.next_due_date)} />
      </div>
      {whmcs?.manage_url && (
        <a
          href={whmcs.manage_url}
          target="_blank"
          rel="noopener noreferrer"
          className={cn(btnClass("primary"), "h-9 w-full")}
        >
          {t("servers.whmcs.manage")}
        </a>
      )}
    </Panel>
  );
}
