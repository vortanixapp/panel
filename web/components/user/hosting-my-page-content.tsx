"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import {
  Calendar,
  ExternalLink,
  Globe,
  Plus,
  SlidersHorizontal,
} from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchMyHosting, type HostingAccount } from "@/lib/api";
import {
  formatHostingDate,
  getHostingDisplayDomain,
  getHostingStatusLabel,
  isHostingExpired,
  planDiskLabel,
} from "@/lib/hosting-status";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import {
  StatCell,
  StatStrip,
  StatusPill,
  UsageBar,
  btnGhost,
  btnPrimary,
  daysLeft,
  pluralDays,
  usagePct,
} from "@/components/user/panel-parts";

function AccountCard({ account }: { account: HostingAccount }) {
  const domain = getHostingDisplayDomain(account);
  const statusLabel = getHostingStatusLabel(account.status, account.status_label);
  const expired = isHostingExpired(account.expires_at);
  const plan = account.hosting_plan;

  const sites = account.domains?.length ?? 0;
  const dbs = account.databases?.length ?? 0;
  const left = daysLeft(account.expires_at);

  const limit = (n: number | null | undefined) => (n == null || n <= 0 ? "∞" : String(n));

  return (
    <div className="flex flex-col gap-4 rounded-2xl border border-border bg-card px-5 py-[18px] transition-colors hover:border-[var(--vx-panel-hover-line)]">
      <div className="flex flex-wrap items-center gap-[14px]">
        <span className="flex h-[38px] w-[38px] flex-none items-center justify-center rounded-[10px] border border-border bg-muted text-foreground">
          <Globe className="h-4 w-4" />
        </span>

        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-[10px]">
            <span className="text-[15.5px] font-semibold">{domain}</span>
            <StatusPill status={account.status} label={statusLabel} />
          </div>
          <div className="mt-1 text-[12.5px] text-muted-foreground">
            {[plan?.name, account.hosting_server?.panel_type_label, account.ip_address]
              .filter(Boolean)
              .join(" · ") || "—"}
          </div>
        </div>

        <div className="ms-auto flex items-center gap-2">
          {account.panel_login_url && (
            <a
              href={account.panel_login_url}
              target="_blank"
              rel="noopener noreferrer"
              className={cn(btnGhost, "h-[34px]")}
            >
              <ExternalLink className="h-3.5 w-3.5" />
              {t("billing.hosting.my.panel")}
            </a>
          )}
          <Link href={`/hosting/${account.id}`} className={cn(btnPrimary, "h-[34px] text-[12.5px]")}>
            <SlidersHorizontal className="h-3.5 w-3.5" />
            {t("billing.hosting.my.manage")}
          </Link>
        </div>
      </div>

      <div className="grid grid-cols-[repeat(auto-fit,minmax(190px,1fr))] gap-[14px]">
        <UsageBar label={t("billing.hosting.disk")} value={planDiskLabel(plan)} pct={null} />
        <UsageBar
          label={t("billing.hosting.my.sites")}
          value={`${sites} / ${limit(plan?.max_domains)}`}
          pct={usagePct(sites, plan?.max_domains)}
        />
        <UsageBar
          label={t("billing.hosting.my.databases")}
          value={`${dbs} / ${limit(plan?.max_databases)}`}
          pct={usagePct(dbs, plan?.max_databases)}
        />
        <div>
          <div className="text-xs text-muted-foreground">
            {t("billing.hosting.expires")}
          </div>
          <div
            className={cn(
              "mt-1.5 flex items-center gap-2 text-[13px]",
              expired ? "text-[var(--vx-warn)]" : "text-foreground"
            )}
          >
            <Calendar className="h-3.5 w-3.5" />
            {formatHostingDate(account.expires_at)}
            {left != null && (
              <span className="text-muted-foreground/70">· {pluralDays(left)}</span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export function MyHostingPageContent() {
  useT();
  const { data: accounts = [], isLoading, isError } = useQuery({
    queryKey: queryKeys.hostingMy,
    queryFn: async () => (await fetchMyHosting()).accounts,
  });

  if (isLoading) {
    return (
      <PageShell variant="user">
        <div className="flex flex-col gap-[22px]">
          <Skeleton className="h-[74px] rounded-[14px]" />
          <Skeleton className="h-40 rounded-2xl" />
          <Skeleton className="h-40 rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  const sites = accounts.reduce((n, a) => n + (a.domains?.length ?? 0), 0);
  const dbs = accounts.reduce((n, a) => n + (a.databases?.length ?? 0), 0);
  const soonest = accounts
    .map((a) => daysLeft(a.expires_at))
    .filter((d): d is number => d != null)
    .sort((a, b) => a - b)[0];

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-[22px]">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div className="flex flex-col gap-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-[-0.02em]">
              {t("billing.hosting.my.title")}
            </h1>
            <p className="text-[13.5px] text-muted-foreground">
              {t("billing.hosting.my.subtitle")}
            </p>
          </div>
          <Link href="/hosting/rent" className={cn(btnPrimary, "h-[38px] px-[18px]")}>
            <Plus className="h-[15px] w-[15px]" />
            {t("billing.hosting.rent.title")}
          </Link>
        </div>

        {isError && (
          <p className="text-sm text-destructive">
            {t("billing.hosting.my.load_failed")}
          </p>
        )}

        {accounts.length > 0 && (
          <StatStrip>
            <StatCell label={t("billing.hosting.my.stat_accounts")} value={accounts.length} />
            <StatCell label={t("billing.hosting.my.stat_sites")} value={sites} />
            <StatCell label={t("billing.hosting.my.databases")} value={dbs} />
            <StatCell
              label={t("billing.hosting.my.stat_next_renewal")}
              value={soonest != null ? pluralDays(soonest) : "—"}
              tone={soonest != null && soonest <= 7 ? "warn" : undefined}
            />
          </StatStrip>
        )}

        <div className="flex flex-col gap-2.5">
          {accounts.map((account) => (
            <AccountCard key={account.id} account={account} />
          ))}
        </div>

        <div className="flex items-center gap-[14px] rounded-2xl border border-dashed border-input bg-card px-5 py-[18px]">
          <span className="flex h-[38px] w-[38px] flex-none items-center justify-center rounded-[10px] bg-muted text-muted-foreground">
            <Plus className="h-4 w-4" />
          </span>
          <div className="flex-1">
            <div className="text-sm font-medium">
              {accounts.length === 0
                ? t("billing.hosting.my.empty_title")
                : t("billing.hosting.my.more_title")}
            </div>
            <div className="mt-0.5 text-[12.5px] text-muted-foreground">
              {t("billing.hosting.my.more_hint")}
            </div>
          </div>
          <Link href="/hosting/rent" className={cn(btnGhost, "h-[34px]")}>
            {t("billing.hosting.my.choose_plan")}
          </Link>
        </div>
      </div>
    </PageShell>
  );
}
