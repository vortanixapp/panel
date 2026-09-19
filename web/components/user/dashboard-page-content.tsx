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
import { ServersCard } from "@/components/user/dashboard/servers-card";
import { BonusCard, NewsCard, RecentCard } from "@/components/user/dashboard/side-cards";
import { BalanceTile, RenewalTile, SpendTile, SupportTile } from "@/components/user/dashboard/stat-tiles";
import { useAccountQuery } from "@/hooks/use-account";
import { useT } from "@/hooks/use-translations";
import { fetchDailyBonus, fetchDashboard } from "@/lib/api";

export function DashboardPageContent() {
  const t = useT();
  const dashboard = useQuery({ queryKey: ["dashboard"], queryFn: fetchDashboard });
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
      <div className="font-panel w-full space-y-6 pb-10">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div className="min-w-0 space-y-1.5">
            <h1 className="text-[26px] leading-tight font-bold tracking-tight">{greeting}</h1>
            <p className="text-[13.5px] text-muted-foreground">{summary}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button asChild variant="outline">
              <Link href="/billing#topup">
                <Wallet />
                {t("home.topup")}
              </Link>
            </Button>
            <Button asChild>
              <Link href="/rent-server">
                <Plus />
                {t("home.rent")}
              </Link>
            </Button>
          </div>
        </div>

        {attention.length > 0 && (
          <EditableSection id="dashboard.next_steps">
            <div>
              <AttentionCard items={attention} />
            </div>
          </EditableSection>
        )}

        <div className="vx-stagger grid grid-cols-1 gap-px overflow-hidden rounded-2xl border bg-border sm:grid-cols-2 xl:grid-cols-4">
          <EditableSection id="dashboard.balance">
            <div className="bg-card">
              <BalanceTile d={d} />
            </div>
          </EditableSection>
          <EditableSection id="dashboard.next_charge">
            <div className="bg-card">
              <RenewalTile d={d} />
            </div>
          </EditableSection>
          <EditableSection id="dashboard.spend">
            <div className="bg-card">
              <SpendTile d={d} />
            </div>
          </EditableSection>
          <EditableSection id="dashboard.support">
            <div className="bg-card">
              <SupportTile d={d} />
            </div>
          </EditableSection>
        </div>

        <div className="grid grid-cols-1 items-start gap-6 xl:grid-cols-[minmax(0,1fr)_340px] xl:grid-rows-[auto_1fr]">
          <EditableSection id="dashboard.servers">
            <div className="min-w-0 xl:col-start-1 xl:row-start-1">
              <ServersCard d={d} />
            </div>
          </EditableSection>
          <div className="min-w-0 space-y-6 xl:col-start-2 xl:row-span-2 xl:row-start-1">
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
          </div>
          {d.news.length > 0 && (
            <EditableSection id="dashboard.news">
              <div className="min-w-0 xl:col-start-1 xl:row-start-2">
                <NewsCard news={d.news} />
              </div>
            </EditableSection>
          )}
        </div>
      </div>
    </PageShell>
  );
}
