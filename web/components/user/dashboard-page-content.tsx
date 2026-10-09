"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { Plus, RotateCcw, Wallet } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { EditableSection } from "@/components/site/editable-section";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { pluralDays } from "@/components/user/panel-parts";
import { AttentionCard, buildAttention } from "@/components/user/dashboard/attention-card";
import { dayPart } from "@/components/user/dashboard/dashboard-utils";
import { LiveCard, QuickStartCard, SpendCard } from "@/components/user/dashboard/extra-cards";
import { ServersCard } from "@/components/user/dashboard/servers-card";
import { BonusCard, NewsCard, RecentCard } from "@/components/user/dashboard/side-cards";
import { BalanceTile, RenewalTile, SpendTile, SupportTile } from "@/components/user/dashboard/stat-tiles";
import { useAccountQuery } from "@/hooks/use-account";
import { useT } from "@/hooks/use-translations";
import { fetchDailyBonus, fetchDashboard } from "@/lib/api";
import { livePollMs } from "@/lib/live-link";

export function DashboardPageContent() {
  const t = useT();
  const dashboard = useQuery({ queryKey: ["dashboard"], queryFn: fetchDashboard, refetchInterval: () => livePollMs(15_000, 60_000) });
  const account = useAccountQuery();
  const bonus = useQuery({ queryKey: ["daily-bonus"], queryFn: fetchDailyBonus, retry: false });

  if (dashboard.isLoading) {
    return (
      <PageShell variant="user">
        <div className="font-panel w-full space-y-6 pb-10">
          <div className="space-y-2">
            <Skeleton className="h-8 w-72 rounded-lg" />
            <Skeleton className="h-4 w-96 max-w-full rounded-md" />
          </div>
          <Skeleton className="h-[168px] w-full rounded-2xl" />
          <div className="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1fr)_340px]">
            <Skeleton className="h-80 w-full rounded-2xl" />
            <Skeleton className="h-80 w-full rounded-2xl" />
          </div>
        </div>
      </PageShell>
    );
  }

  if (dashboard.isError || !dashboard.data) {
    return (
      <PageShell variant="user">
        <div className="font-panel flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-destructive/30 bg-destructive/5 px-5 py-4 text-[13.5px] text-destructive">
          <span>{t("home.load_failed")}</span>
          <Button size="sm" variant="outline" onClick={() => void dashboard.refetch()}>
            <RotateCcw />
            {t("common.retry")}
          </Button>
        </div>
      </PageShell>
    );
  }

  const d = dashboard.data;
  const user = account.data?.user;
  const name = (user?.first_name || user?.display_name || "").trim();
  const part = dayPart();
  const greeting = name ? t(`home.greeting.${part}_name`, { name }) : t(`home.greeting.${part}`);
  const monthly = d.monthly_spend ?? 0;
  const daysLeft = monthly > 0 ? Math.floor(d.balance / (monthly / 30)) : null;
  const summary =
    d.total_servers === 0
      ? t("home.summary.empty")
      : [
          t("home.summary.servers", { running: d.active_servers, total: d.total_servers }),
          daysLeft !== null && daysLeft > 0 ? t("home.summary.days", { days: pluralDays(daysLeft) }) : "",
        ]
          .filter(Boolean)
          .join(" · ");
  const attention = buildAttention(d, user, t);

  return (
    <PageShell variant="user">
      <div className="font-panel srv2 relative w-full space-y-5 pb-10">
        <div className="srv2-glow" aria-hidden />
        <div className="srv2-rise relative flex flex-wrap items-end justify-between gap-5">
          <div className="min-w-0">
            <h1 className="text-[28px] leading-[1.1] font-semibold tracking-[-0.03em] sm:text-[38px]">{greeting}</h1>
            <p className="mt-2 text-[14px] text-muted-foreground">{summary}</p>
          </div>
          <div className="flex flex-wrap gap-2.5">
            <Button asChild variant="outline" className="h-10 rounded-[10px]">
              <Link href="/billing#topup">
                <Wallet />
                {t("home.topup")}
              </Link>
            </Button>
            <Button asChild className="relative h-10 overflow-hidden rounded-[10px] font-semibold">
              <Link href="/rent-server">
                <Plus />
                {t("home.rent")}
                <span className="srv2-sheen pointer-events-none absolute inset-y-0 w-2/5 bg-gradient-to-r from-transparent via-white/40 to-transparent" />
              </Link>
            </Button>
          </div>
        </div>

        {attention.length > 0 && (
          <EditableSection id="dashboard.next_steps">
            <div className="relative">
              <AttentionCard items={attention} />
            </div>
          </EditableSection>
        )}

        <div className="relative grid grid-cols-[repeat(auto-fit,minmax(min(100%,240px),1fr))] gap-3.5">
          <EditableSection id="dashboard.balance">
            <div>
              <BalanceTile d={d} />
            </div>
          </EditableSection>
          <EditableSection id="dashboard.next_charge">
            <div>
              <RenewalTile d={d} />
            </div>
          </EditableSection>
          <EditableSection id="dashboard.spend">
            <div>
              <SpendTile d={d} />
            </div>
          </EditableSection>
          <EditableSection id="dashboard.support">
            <div>
              <SupportTile d={d} />
            </div>
          </EditableSection>
        </div>

        <div className="relative grid grid-cols-1 items-start gap-5 xl:grid-cols-[minmax(0,1fr)_380px]">
          <div className="flex min-w-0 flex-col gap-5">
            <EditableSection id="dashboard.servers">
              <div className="min-w-0">
                <ServersCard d={d} />
              </div>
            </EditableSection>
            {d.total_servers > 0 && <LiveCard d={d} />}
            <SpendCard d={d} />
          </div>
          <div className="flex min-w-0 flex-col gap-5">
            {bonus.data && !bonus.data.prizes_disabled && (
              <EditableSection id="dashboard.bonus">
                <div>
                  <BonusCard bonus={bonus.data} />
                </div>
              </EditableSection>
            )}
            <EditableSection id="dashboard.recent">
              <div>
                <RecentCard d={d} />
              </div>
            </EditableSection>
            <QuickStartCard d={d} account={user} />
            {d.news.length > 0 && (
              <EditableSection id="dashboard.news">
                <div className="min-w-0">
                  <NewsCard news={d.news} />
                </div>
              </EditableSection>
            )}
          </div>
        </div>
      </div>
    </PageShell>
  );
}
