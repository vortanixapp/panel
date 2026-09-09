"use client";

import Link from "next/link";
import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchDailyBonus,
  fetchDashboard,
  spinDailyBonus,
  type DashboardData,
  type DashboardServer,
  type DashboardTransaction,
} from "@/lib/api";
import { formatAmount } from "@/lib/format";
import { localeTag, t } from "@/lib/i18n";
import { getServerStatus } from "@/lib/server-status";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const EMPTY: DashboardData = {
  balance: 0,
  balance_currency: "RUB",
  total_servers: 0,
  active_servers: 0,
  expiring_soon_count: 0,
  open_support_tickets_count: 0,
  next_charge_text: "—",
  recent_servers: [],
  recent_transactions: [],
  news: [],
};

const CARD = "rounded-[22px] border border-[var(--vx-panel-line)] bg-[var(--vx-panel-card)]";
const CARD_RAISED =
  "rounded-[22px] border border-[var(--vx-panel-line-strong)] bg-[var(--vx-panel-card)]";
const CARD_LABEL =
  "text-[13px] font-semibold tracking-[0.02em] text-muted-foreground";

function shortDate(d: Date) {
  return d.toLocaleDateString(localeTag(), { day: "numeric", month: "long" });
}

function relTime(iso: string) {
  const d = new Date(iso);
  const today = new Date();
  const sameDay = d.toDateString() === today.toDateString();
  const yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);
  const time = d.toLocaleTimeString(localeTag(), {
    hour: "2-digit",
    minute: "2-digit",
  });
  if (sameDay) return t("dashboard.rel.today", { time });
  if (d.toDateString() === yesterday.toDateString()) {
    return t("dashboard.rel.yesterday", { time });
  }
  return t("dashboard.rel.date", {
    date: d.toLocaleDateString(localeTag(), { day: "numeric", month: "short" }),
    time,
  });
}

function daysUntil(iso: string | null) {
  if (!iso) return null;
  const ms = new Date(iso).getTime() - Date.now();
  if (!Number.isFinite(ms)) return null;
  return Math.ceil(ms / 86_400_000);
}

function plural(n: number, one: string, few: string, many: string) {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return one;
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return few;
  return many;
}

function monthlyBurn(servers: DashboardServer[]) {
  return servers.reduce((sum, s) => sum + Number(s.tariff?.price_monthly ?? 0), 0);
}

function spendSeries(transactions: DashboardTransaction[]) {
  const days: { debit: number; credit: number; date: Date }[] = [];
  const start = new Date();
  start.setHours(0, 0, 0, 0);
  start.setDate(start.getDate() - 29);

  for (let i = 0; i < 30; i++) {
    const date = new Date(start);
    date.setDate(start.getDate() + i);
    days.push({ debit: 0, credit: 0, date });
  }

  for (const t of transactions) {
    const d = new Date(t.created_at);
    d.setHours(0, 0, 0, 0);
    const idx = Math.round((d.getTime() - start.getTime()) / 86_400_000);
    if (idx < 0 || idx > 29) continue;
    const amount = Math.abs(Number(t.amount) || 0);
    if (t.type === "credit") days[idx].credit += amount;
    else days[idx].debit += amount;
  }

  const peak = Math.max(1, ...days.map((d) => Math.max(d.debit, d.credit)));
  const totalDebit = days.reduce((s, d) => s + d.debit, 0);
  return { days, peak, totalDebit };
}

type NextStep = {
  href: string;
  titleKey: string;
  titleParams: Record<string, string | number>;
  subKey: string;
  subParams: Record<string, string | number>;
  icon: string;
  tone: "neutral" | "info" | "warn" | "danger";
  raised: boolean;
};

function ArrowIcon({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.8}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
    >
      <path d="M5 12h13M13 6.5 18.5 12 13 17.5" />
    </svg>
  );
}

export function DashboardPageContent() {
  useT();
  const qc = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["dashboard"],
    queryFn: fetchDashboard,
  });

  const { data: bonus } = useQuery({
    queryKey: ["daily-bonus"],
    queryFn: fetchDailyBonus,
  });

  const spin = useMutation({
    mutationFn: spinDailyBonus,
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ["daily-bonus"] });
      qc.invalidateQueries({ queryKey: ["dashboard"] });
      if (res.prize_id) toast.success(t("dashboard.bonus.prize_won"));
      else toast.info(t("dashboard.bonus.no_prize"));
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("billing.bonus.spin_failed")
      ),
  });

  const d = data ?? EMPTY;
  const empty = d.total_servers === 0;

  const burn = useMemo(() => {
    const perMonth = monthlyBurn(d.recent_servers);
    const perDay = perMonth / 30;
    const daysLeft = perDay > 0 ? Math.floor(d.balance / perDay) : null;
    return { perMonth, perDay, daysLeft };
  }, [d.recent_servers, d.balance]);

  const spend = useMemo(
    () => spendSeries(d.recent_transactions),
    [d.recent_transactions]
  );

  const soonest = useMemo(() => {
    const withDates = d.recent_servers
      .filter((s) => s.expires_at)
      .map((s) => ({ server: s, days: daysUntil(s.expires_at) ?? Infinity }))
      .sort((a, b) => a.days - b.days);
    return withDates[0] ?? null;
  }, [d.recent_servers]);

  const nextSteps = useMemo<NextStep[]>(() => {
    if (empty) {
      return [
        {
          href: "/rent-server",
          titleKey: "dashboard.step.choose_game",
          titleParams: {},
          subKey: "dashboard.step.choose_game_sub",
          subParams: {},
          icon: "ri-gamepad-line",
          tone: "neutral" as const,
          raised: true,
        },
        {
          href: "/billing/topup",
          titleKey: "billing.history.topup_cta",
          titleParams: {},
          subKey: "dashboard.step.topup_sub",
          subParams: {},
          icon: "ri-wallet-3-line",
          tone: "info" as const,
          raised: false,
        },
        {
          href: "/daily-bonus",
          titleKey: "dashboard.step.bonus",
          titleParams: {},
          subKey: "dashboard.step.bonus_sub",
          subParams: {},
          icon: "ri-gift-line",
          tone: "warn" as const,
          raised: false,
        },
      ];
    }
    const steps: NextStep[] = [];
    if (soonest && soonest.days <= 7) {
      steps.push({
        href: `/servers/${soonest.server.id}`,
        titleKey: "dashboard.step.renew",
        titleParams: { name: soonest.server.name },
        subKey:
          soonest.days <= 0
            ? "dashboard.step.renew_expired"
            : plural(
                soonest.days,
                "dashboard.servers.expires_in_one",
                "dashboard.servers.expires_in_few",
                "dashboard.servers.expires_in_many"
              ),
        subParams: { days: soonest.days },
        icon: "ri-time-line",
        tone: "warn" as const,
        raised: true,
      });
    }
    if (d.open_support_tickets_count > 0) {
      steps.push({
        href: "/support",
        titleKey: "dashboard.step.support",
        titleParams: {},
        subKey: plural(
          d.open_support_tickets_count,
          "dashboard.support.tickets_line_one",
          "dashboard.support.tickets_line_few",
          "dashboard.support.tickets_line_many"
        ),
        subParams: { count: d.open_support_tickets_count },
        icon: "ri-customer-service-2-line",
        tone: "danger" as const,
        raised: steps.length === 0,
      });
    }
    steps.push({
      href: "/billing/topup",
      titleKey: "billing.history.topup_cta",
      titleParams: {},
      subKey:
        burn.daysLeft !== null
          ? plural(
              burn.daysLeft,
              "dashboard.step.topup_days_one",
              "dashboard.step.topup_days_few",
              "dashboard.step.topup_days_many"
            )
          : "dashboard.step.topup_keep",
      subParams: burn.daysLeft !== null ? { days: burn.daysLeft } : {},
      icon: "ri-wallet-3-line",
      tone: "neutral" as const,
      raised: steps.length === 0,
    });
    steps.push({
      href: "/rent-server",
      titleKey: "dashboard.step.rent_more",
      titleParams: {},
      subKey: "dashboard.step.rent_more_sub",
      subParams: {},
      icon: "ri-server-line",
      tone: "neutral" as const,
      raised: false,
    });
    return steps.slice(0, 3);
  }, [empty, soonest, d.open_support_tickets_count, burn.daysLeft]);

  if (isLoading) {
    return (
      <PageShell variant="user">
        <div className="font-panel grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-6">
          <Skeleton className="rounded-[22px] lg:col-span-2 lg:row-span-2 h-[300px]" />
          <Skeleton className="h-[286px] rounded-[22px] lg:col-span-4" />
          <Skeleton className="h-[150px] rounded-[22px] lg:col-span-2" />
          <Skeleton className="h-[150px] rounded-[22px] lg:col-span-2" />
          <Skeleton className="h-[240px] rounded-[22px] lg:col-span-4" />
          <Skeleton className="h-[240px] rounded-[22px] lg:col-span-2" />
          <Skeleton className="h-[280px] rounded-[22px] lg:col-span-3" />
          <Skeleton className="h-[280px] rounded-[22px] lg:col-span-3" />
        </div>
      </PageShell>
    );
  }

  const balancePct =
    burn.daysLeft !== null ? Math.max(0, Math.min(100, (burn.daysLeft / 30) * 100)) : 0;

  return (
    <PageShell variant="user">
      <div className="font-panel">
        <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-6">
          <div
            className={cn(
              CARD_RAISED,
              "flex flex-col p-[26px] sm:col-span-2 lg:row-span-2"
            )}
          >
            <div className="flex items-center justify-between">
              <span className={CARD_LABEL}>{t("dashboard.balance")}</span>
              <i className="ri-wallet-3-line text-lg" />
            </div>
            <div className="mt-[26px] flex items-baseline gap-2">
              <span className="text-[46px] leading-none font-semibold tracking-[-0.045em] sm:text-[54px]">
                {formatAmount(d.balance, 0)}
              </span>
              <span className="text-[17px] font-medium text-[var(--vx-ink-faint)]">
                {d.balance_currency === "RUB" ? "₽" : d.balance_currency}
              </span>
            </div>
            <div className="mt-3 text-sm leading-[1.5] text-muted-foreground">
              {burn.perDay > 0
                ? burn.daysLeft !== null
                  ? t(
                      plural(
                        burn.daysLeft,
                        "dashboard.balance.burn_days_one",
                        "dashboard.balance.burn_days_few",
                        "dashboard.balance.burn_days_many"
                      ),
                      { amount: formatAmount(burn.perDay, 0), days: burn.daysLeft }
                    )
                  : t("dashboard.balance.burn", {
                      amount: formatAmount(burn.perDay, 0),
                    })
                : t("billing.balance.topup_hint")}
            </div>
            <div className="flex-1" />
            <div className="mt-6 h-[5px] overflow-hidden rounded-[3px] bg-[var(--vx-panel-track)]">
              <div
                className="h-[5px] bg-[var(--vx-panel-ink)]"
                style={{ width: `${balancePct}%` }}
              />
            </div>
            <div className="mt-[18px] flex gap-2.5">
              <Link
                href="/billing/topup"
                className="flex-1 rounded-full bg-[var(--vx-panel-track)] border border-[var(--vx-btn-line)] py-[11px] text-center text-[13.5px] font-semibold text-[var(--vx-panel-ink)] transition-colors hover:bg-[var(--vx-btn-hover)]"
              >
                {t("dashboard.topup")}
              </Link>
              <Link
                href="/billing"
                className="rounded-full border border-[var(--vx-panel-line-strong)] px-4 py-[11px] text-[13.5px] font-medium text-[var(--vx-panel-ink-2)] transition-colors hover:border-[var(--vx-panel-hover-line)]"
              >
                {t("dashboard.invoices")}
              </Link>
            </div>
          </div>

          <div className={cn(CARD, "px-[26px] py-6 sm:col-span-2 lg:col-span-4")}>
            <div className="flex flex-wrap items-center gap-3.5">
              <span className="text-base font-semibold">{t("dashboard.servers.title")}</span>
              <span className="font-mono text-xs text-[var(--vx-ink-faint)]">
                {empty
                  ? t("dashboard.servers.zero_active")
                  : t("dashboard.servers.count", {
                      active: d.active_servers,
                      total: d.total_servers,
                    })}
              </span>
              <div className="flex-1" />
              <Link
                href="/servers"
                className="flex items-center gap-[7px] text-[13px] font-medium text-primary"
              >
                {t("common.all")}
                <ArrowIcon className="size-3.5" />
              </Link>
            </div>

            {d.recent_servers.length === 0 ? (
              <div className="mt-[18px] flex flex-col items-start gap-[22px] rounded-2xl border border-dashed border-[var(--vx-panel-line-strong)] px-[26px] py-[34px] sm:flex-row sm:items-center">
                <div className="flex size-[52px] flex-shrink-0 items-center justify-center rounded-[15px] bg-[var(--vx-panel-media)] text-[var(--vx-ink-ghost)]">
                  <i className="ri-server-line text-2xl" />
                </div>
                <div className="flex-1">
                  <div className="text-[17px] font-semibold">
                    {t("dashboard.servers.empty_title")}
                  </div>
                  <div className="mt-[7px] max-w-[420px] text-[13.5px] leading-[1.55] text-muted-foreground">
                    {t("dashboard.servers.empty_text")}
                  </div>
                </div>
                <Link
                  href="/rent-server"
                  className="vx-btn rounded-full px-[22px] py-3 text-[13.5px] font-semibold whitespace-nowrap"
                >
                  {t("dashboard.servers.rent")}
                </Link>
              </div>
            ) : (
              <div className="mt-[18px] grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
                {d.recent_servers.slice(0, 3).map((s, i) => {
                  const st = getServerStatus(s);
                  const online = Number(s.online_players ?? 0);
                  const max = Number(s.max_players ?? 0);
                  const fill = max > 0 ? online / max : 0;
                  const left = daysUntil(s.expires_at);
                  const expiring = left !== null && left <= 7;
                  const running = st.category === "running" || st.category === "active";
                  return (
                    <Link
                      key={s.id}
                      href={`/servers/${s.id}`}
                      className={cn(
                        "rounded-2xl border p-[18px] transition-colors hover:border-[var(--vx-panel-hover-line)]",
                        i === 0
                          ? "border-[var(--vx-panel-line-strong)] bg-[var(--vx-panel-card)]"
                          : "border-[var(--vx-panel-line)] bg-[var(--vx-panel-card-2)]"
                      )}
                    >
                      <div className="flex items-center gap-2.5">
                        <span
                          className={cn(
                            "size-[7px] flex-shrink-0 rounded-full",
                            running ? "bg-[var(--vx-panel-fill)]" : "bg-[var(--vx-panel-fill-muted)]"
                          )}
                        />
                        <span className="truncate text-sm font-semibold tracking-[-0.01em]">
                          {s.name}
                        </span>
                        <div className="flex-1" />
                        <span className="font-mono text-[11px] whitespace-nowrap text-[var(--vx-ink-faint)]">
                          {s.location?.city || s.location?.name || "—"}
                        </span>
                      </div>

                      <div className="mt-3.5 flex h-[38px] items-end gap-[3px]">
                        {Array.from({ length: 14 }).map((_, k) => {
                          const active = k < Math.round(fill * 14);
                          return (
                            <div
                              key={k}
                              className={cn(
                                "flex-1 rounded-[2px]",
                                active ? "bg-[var(--vx-panel-fill)]" : "bg-[var(--vx-panel-track)]"
                              )}
                              style={{ height: active ? `${45 + ((k * 13) % 55)}%` : "18%" }}
                            />
                          );
                        })}
                      </div>

                      <div className="mt-3.5 flex items-baseline justify-between">
                        <span
                          className={cn(
                            "font-mono text-[19px] font-medium",
                            running ? "text-[var(--vx-panel-ink)]" : "text-[var(--vx-ink-faint)]"
                          )}
                        >
                          {online}
                        </span>
                        <span className="text-xs text-[var(--vx-ink-faint)]">
                          {max > 0
                            ? t("dashboard.servers.of_players", { max })
                            : st.label.toLowerCase()}
                        </span>
                      </div>

                      <div className="mt-3.5 flex items-center justify-between border-t border-[var(--vx-panel-line)] pt-[13px] text-xs">
                        <span
                          className={cn(
                            expiring ? "text-[var(--vx-warn)]" : "text-[var(--vx-ink-faint)]"
                          )}
                        >
                          {left === null
                            ? st.label
                            : left <= 0
                              ? t("dashboard.servers.expired")
                              : expiring
                                ? t(
                                    plural(
                                      left,
                                      "dashboard.servers.expires_in_one",
                                      "dashboard.servers.expires_in_few",
                                      "dashboard.servers.expires_in_many"
                                    ),
                                    { days: left }
                                  )
                                : t("dashboard.servers.until", {
                                    date: new Date(s.expires_at!).toLocaleDateString(
                                      localeTag(),
                                      { day: "numeric", month: "long" }
                                    ),
                                  })}
                        </span>
                        <span className="font-semibold text-primary">
                          {expiring
                            ? t("dashboard.servers.renew")
                            : t("dashboard.servers.console")}
                        </span>
                      </div>
                    </Link>
                  );
                })}
              </div>
            )}
          </div>

          <div
            className={cn(
              d.expiring_soon_count > 0 ? CARD_RAISED : CARD,
              "px-6 py-[22px] sm:col-span-1 lg:col-span-2"
            )}
          >
            <div className="flex items-center justify-between">
              <span className={CARD_LABEL}>{t("billing.balance.next_charge")}</span>
              <i
                className={cn(
                  "ri-time-line text-[17px]",
                  d.expiring_soon_count > 0
                    ? "text-[var(--vx-warn)]"
                    : "text-[var(--vx-ink-faint)]"
                )}
              />
            </div>
            <div
              className={cn(
                "mt-4 text-[30px] font-semibold tracking-[-0.03em]",
                d.expiring_soon_count > 0
                  ? "text-[var(--vx-warn)]"
                  : "text-[var(--vx-ink-faint)]"
              )}
            >
              {soonest && soonest.days !== Infinity
                ? soonest.days <= 0
                  ? t("dashboard.next_charge.expired")
                  : t(
                      plural(
                        soonest.days,
                        "dashboard.next_charge.in_one",
                        "dashboard.next_charge.in_few",
                        "dashboard.next_charge.in_many"
                      ),
                      { days: soonest.days }
                    )
                : "—"}
            </div>
            <div className="mt-2 text-[13.5px] text-muted-foreground">
              {soonest && soonest.days !== Infinity
                ? `${soonest.server.name}${soonest.server.tariff?.price_monthly ? ` · ${formatAmount(soonest.server.tariff.price_monthly, 0)} ₽` : ""}`
                : d.next_charge_text !== "—"
                  ? d.next_charge_text
                  : t("billing.balance.no_subscriptions")}
            </div>
          </div>

          <div className={cn(CARD, "flex flex-col px-6 py-[22px] sm:col-span-1 lg:col-span-2")}>
            <div className="flex items-center justify-between">
              <span className={CARD_LABEL}>{t("dashboard.support.title")}</span>
              <i
                className={cn(
                  "ri-customer-service-2-line text-[17px]",
                  d.open_support_tickets_count > 0
                    ? "text-[var(--vx-danger)]"
                    : "text-muted-foreground"
                )}
              />
            </div>
            <div className="mt-4 text-[30px] font-semibold tracking-[-0.03em]">
              {d.open_support_tickets_count > 0
                ? t(
                    plural(
                      d.open_support_tickets_count,
                      "dashboard.support.open_line_one",
                      "dashboard.support.open_line_few",
                      "dashboard.support.open_line_many"
                    ),
                    { count: d.open_support_tickets_count }
                  )
                : t("dashboard.support.none")}
            </div>
            <div className="mt-2 text-[13.5px] text-muted-foreground">
              {d.open_support_tickets_count > 0
                ? t("dashboard.support.waiting")
                : t("dashboard.support.first_reply")}
            </div>
            <div className="flex-1" />
            <Link
              href={d.open_support_tickets_count > 0 ? "/support" : "/support/create"}
              className="mt-4 flex items-center gap-[7px] text-[13px] font-semibold text-primary"
            >
              {d.open_support_tickets_count > 0
                ? t("dashboard.support.open_tickets")
                : t("dashboard.support.write")}
              <ArrowIcon className="size-3.5" />
            </Link>
          </div>

          <div className={cn(CARD, "px-[26px] py-6 sm:col-span-2 lg:col-span-4")}>
            <div className="flex flex-wrap items-center gap-3.5">
              <span className="text-base font-semibold">{t("dashboard.spend.title")}</span>
              <span className="font-mono text-xs text-[var(--vx-ink-faint)]">
                {formatAmount(spend.totalDebit, 0)} ₽
              </span>
              <div className="flex-1" />
              <div className="flex gap-4">
                <div className="flex items-center gap-[7px] text-xs text-muted-foreground">
                  <span className="size-2 rounded-[3px] bg-[var(--vx-panel-fill)]" />
                  {t("billing.tx.debits")}
                </div>
                <div className="flex items-center gap-[7px] text-xs text-muted-foreground">
                  <span className="size-2 rounded-[3px] bg-[var(--vx-panel-fill-muted)]" />
                  {t("billing.tx.credits")}
                </div>
              </div>
            </div>
            <div className="mt-[22px] flex h-[118px] items-end gap-[5px]">
              {spend.days.map((day, i) => (
                <div
                  key={i}
                  className="flex h-full flex-1 items-end gap-px"
                  title={t("dashboard.spend.day_tooltip", {
                    date: shortDate(day.date),
                    debit: formatAmount(day.debit, 0),
                    credit: formatAmount(day.credit, 0),
                  })}
                >
                  <div
                    className="flex-1 rounded-t-[3px] bg-[var(--vx-panel-fill)]"
                    style={{
                      height: `${Math.max(day.debit > 0 ? 4 : 2, (day.debit / spend.peak) * 100)}%`,
                    }}
                  />
                  <div
                    className="flex-1 rounded-t-[3px] bg-[var(--vx-panel-fill-muted)]"
                    style={{
                      height: `${Math.max(day.credit > 0 ? 4 : 2, (day.credit / spend.peak) * 100)}%`,
                    }}
                  />
                </div>
              ))}
            </div>
            <div className="mt-[11px] flex justify-between font-mono text-[11px] text-[var(--vx-ink-ghost)]">
              <span>{shortDate(spend.days[0].date)}</span>
              <span>{shortDate(spend.days[14].date)}</span>
              <span>{shortDate(spend.days[29].date)}</span>
            </div>
            {spend.totalDebit === 0 && (
              <div className="mt-2 text-xs text-[var(--vx-ink-ghost)]">
                {t("dashboard.spend.empty")}
              </div>
            )}
          </div>

          <div className={cn(CARD_RAISED, "flex flex-col px-6 py-[22px] sm:col-span-2")}>
            <div className="flex items-center justify-between">
              <span className="text-[13px] font-semibold tracking-[0.02em] text-[var(--vx-warn)]">
                {t("billing.bonus.title")}
              </span>
              <i className="ri-gift-line text-[17px] text-[var(--vx-warn)]" />
            </div>
            <div className="mt-4 text-[15.5px] leading-[1.5] text-[var(--vx-panel-ink-2)]">
              {bonus?.can_spin
                ? t("dashboard.bonus.available")
                : t("dashboard.bonus.done")}
            </div>
            <div className="flex-1" />
            <button
              type="button"
              disabled={!bonus?.can_spin || spin.isPending}
              onClick={() => spin.mutate()}
              className={cn(
                "mt-5 rounded-full py-3 text-center text-[13.5px] font-semibold transition-colors",
                bonus?.can_spin && !spin.isPending
                  ? "border border-[var(--vx-panel-line-strong)] bg-[var(--vx-panel-tint)] text-[var(--vx-warn)] hover:bg-[var(--vx-btn-hover)]"
                  : "cursor-not-allowed border border-[var(--vx-panel-line-strong)] bg-transparent text-muted-foreground"
              )}
            >
              {spin.isPending
                ? t("billing.bonus.spinning_button")
                : bonus?.can_spin
                  ? t("billing.bonus.spin_button")
                  : t("dashboard.bonus.already")}
            </button>
          </div>

          <div className={cn(CARD, "px-[26px] py-6 sm:col-span-2 lg:col-span-3")}>
            <div className="flex items-center">
              <span className="text-base font-semibold">{t("dashboard.recent.title")}</span>
              <div className="flex-1" />
              <Link
                href="/activity"
                className="flex items-center gap-[7px] text-[13px] font-medium text-primary"
              >
                {t("dashboard.recent.journal")}
                <ArrowIcon className="size-3.5" />
              </Link>
            </div>
            {d.recent_transactions.length === 0 ? (
              <div className="mt-[18px] py-9 text-center">
                <div className="text-sm text-muted-foreground">
                  {t("billing.history.empty_title")}
                </div>
                <div className="mt-[7px] text-[13px] text-[var(--vx-ink-ghost)]">
                  {t("dashboard.recent.empty_text")}
                </div>
              </div>
            ) : (
              <div className="mt-3.5 flex flex-col">
                {d.recent_transactions.slice(0, 4).map((t) => {
                  const credit = t.type === "credit";
                  return (
                    <div
                      key={t.id}
                      className="flex items-center gap-3.5 border-t border-[var(--vx-panel-line)] py-[13px]"
                    >
                      <div
                        className={cn(
                          "flex size-[30px] flex-shrink-0 items-center justify-center rounded-[10px] text-[13px] font-semibold",
                          credit
                            ? "bg-[var(--vx-panel-tint)] text-[var(--vx-panel-ink)]"
                            : "bg-[var(--vx-panel-tint)] text-muted-foreground"
                        )}
                      >
                        {credit ? "+" : "−"}
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-[13.5px] font-medium">
                          {t.description || "—"}
                        </div>
                        <div className="mt-0.5 font-mono text-[11px] text-[var(--vx-ink-ghost)]">
                          {relTime(t.created_at)}
                        </div>
                      </div>
                      <div
                        className={cn(
                          "font-mono text-[13.5px] whitespace-nowrap",
                          credit ? "text-[var(--vx-panel-ink)]" : "text-[var(--vx-panel-ink-2)]"
                        )}
                      >
                        {credit ? "+" : "−"}
                        {formatAmount(Math.abs(Number(t.amount) || 0), 0)} ₽
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          <div className={cn(CARD, "px-[26px] py-6 sm:col-span-2 lg:col-span-3")}>
            <div className="text-base font-semibold">{t("dashboard.next_steps.title")}</div>
            <div className="mt-4 flex flex-col gap-2.5">
              {nextSteps.map((n) => (
                <Link
                  key={n.titleKey}
                  href={n.href}
                  className={cn(
                    "flex items-center gap-3.5 rounded-[14px] border px-4 py-3.5 transition-colors hover:border-[var(--vx-panel-hover-line)]",
                    n.raised
                      ? "border-[var(--vx-panel-line-strong)] bg-[var(--vx-panel-card)]"
                      : "border-[var(--vx-panel-line)] bg-[var(--vx-panel-card-2)]"
                  )}
                >
                  <div
                    className={cn(
                      "flex size-[30px] flex-shrink-0 items-center justify-center rounded-[10px]",
                      n.tone === "warn" && "bg-[rgba(232,160,60,0.13)] text-[var(--vx-warn)]",
                      n.tone === "danger" && "bg-[rgba(224,122,122,0.13)] text-[var(--vx-danger)]",
                      n.tone === "info" && "bg-[rgba(123,179,232,0.13)] text-[var(--vx-info)]",
                      n.tone === "neutral" && "bg-[var(--vx-panel-tint)] text-[var(--vx-panel-ink)]"
                    )}
                  >
                    <i className={cn(n.icon, "text-[15px]")} />
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-[13.5px] font-semibold">
                      {t(n.titleKey, n.titleParams)}
                    </div>
                    <div className="mt-0.5 truncate text-xs text-[var(--vx-ink-faint)]">
                      {t(n.subKey, n.subParams)}
                    </div>
                  </div>
                  <ArrowIcon className="size-[15px] flex-shrink-0 text-[var(--vx-ink-ghost)]" />
                </Link>
              ))}
            </div>
          </div>
        </div>

        {d.news.length > 0 && (
          <div className={cn(CARD, "mt-3.5 px-[26px] py-6")}>
            <div className="flex items-center">
              <span className="text-base font-semibold">{t("news.title")}</span>
              <div className="flex-1" />
              <Link
                href="/news"
                className="flex items-center gap-[7px] text-[13px] font-medium text-primary"
              >
                {t("news.all")}
                <ArrowIcon className="size-3.5" />
              </Link>
            </div>
            <div className="mt-4 grid gap-3.5 sm:grid-cols-2 lg:grid-cols-3">
              {d.news.map((item) => (
                <Link
                  key={item.id}
                  href={`/news/${item.slug || item.id}`}
                  className="group overflow-hidden rounded-2xl border border-[var(--vx-panel-line)] bg-[var(--vx-panel-card-2)] transition-colors hover:border-[var(--vx-panel-hover-line)]"
                >
                  {item.image ? (
                    <div className="aspect-video overflow-hidden bg-[var(--vx-panel-media)]">
                      <img
                        src={item.image}
                        alt=""
                        className="size-full object-cover transition-transform duration-500 group-hover:scale-105"
                      />
                    </div>
                  ) : (
                    <div className="flex aspect-video items-center justify-center bg-[var(--vx-panel-media)] text-[var(--vx-ink-ghost)]">
                      <i className="ri-newspaper-line text-4xl" />
                    </div>
                  )}
                  <div className="p-4">
                    <div className="font-mono text-[11px] text-[var(--vx-ink-faint)]">
                      {item.published_at
                        ? new Date(item.published_at).toLocaleDateString(localeTag())
                        : "—"}
                    </div>
                    <h4 className="mt-2 line-clamp-2 text-[14.5px] font-semibold">
                      {item.title}
                    </h4>
                    <p className="mt-1.5 line-clamp-2 text-[13px] leading-[1.5] text-muted-foreground">
                      {item.excerpt}
                    </p>
                  </div>
                </Link>
              ))}
            </div>
          </div>
        )}
      </div>
    </PageShell>
  );
}
