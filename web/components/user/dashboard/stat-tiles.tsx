"use client";

import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { AnimatedNumber } from "@/components/vx/motion";
import { pluralDays } from "@/components/user/panel-parts";
import type { DashboardData } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";
import { amountText, currencySymbol, daysUntil, money, moneyPrecise, shortDate } from "./dashboard-utils";

function Tile({
  label,
  icon,
  accent,
  tone,
  delay = 0,
  footer,
  children,
}: {
  label: string;
  icon: string;
  accent: string;
  tone?: "warning" | "danger";
  delay?: number;
  footer?: React.ReactNode;
  children: React.ReactNode;
}) {
  const color = tone === "danger" ? "var(--vx-danger)" : tone === "warning" ? "var(--vx-warn)" : accent;
  return (
    <div
      data-spotlight
      style={{ animationDelay: `${delay}ms` }}
      className="srv2-rise relative isolate flex h-full min-h-[200px] flex-col gap-1.5 overflow-hidden rounded-[20px] border bg-card px-5 py-[18px] transition-colors hover:border-[var(--vx-border-strong)]"
    >
      <div
        className="pointer-events-none absolute -top-10 -right-10 size-[120px] rounded-full opacity-[0.12]"
        style={{ background: `radial-gradient(closest-side, ${color}, transparent)` }}
      />
      <div className="flex items-center justify-between gap-3">
        <span className="font-mono text-[11px] tracking-[0.12em] text-muted-foreground uppercase">{label}</span>
        <i className={cn(icon, "text-[17px]")} style={{ color }} />
      </div>
      <div className="min-w-0 flex-1">{children}</div>
      {footer && <div className="mt-1 text-[13px]">{footer}</div>}
    </div>
  );
}

function TileLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <Link href={href} className="group inline-flex items-center gap-1.5 font-medium text-foreground">
      {children}
      <ArrowRight className="size-3.5 transition-transform group-hover:translate-x-0.5" />
    </Link>
  );
}

export function BalanceTile({ d }: { d: DashboardData }) {
  const t = useT();
  const monthly = d.monthly_spend ?? 0;
  const perDay = monthly / 30;
  const daysLeft = perDay > 0 ? Math.floor(d.balance / perDay) : null;
  const low = daysLeft !== null && daysLeft < 7;
  const others = (d.wallets ?? []).slice(1).filter((w) => w.balance !== 0);
  const digits = Number.isInteger(Math.round((Number(d.balance) || 0) * 100) / 100) ? 0 : 2;
  return (
    <Tile
      label={t("home.balance.title")}
      icon="ri-wallet-3-line"
      accent="var(--vx-muted)"
      delay={100}
      tone={low ? "warning" : undefined}
      footer={<TileLink href="/billing#topup">{t("home.balance.topup")}</TileLink>}
    >
      <div className="flex items-baseline gap-1.5">
        <AnimatedNumber
          value={Number(d.balance) || 0}
          format={(value) => amountText(value, digits)}
          className="text-[36px] leading-[1.2] font-semibold tracking-[-0.03em] tabular-nums"
        />
        <span className="text-[15px] font-medium text-muted-foreground">{currencySymbol(d.balance_currency)}</span>
      </div>
      <p className={cn("mt-2 text-[12.5px] leading-snug", low ? "text-amber-600 dark:text-amber-500" : "text-muted-foreground")}>
        {daysLeft !== null
          ? daysLeft <= 0
            ? t("home.balance.empty")
            : t("home.balance.days_left", { days: pluralDays(daysLeft) })
          : t("home.balance.no_spend")}
      </p>
      {others.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1.5">
          {others.map((w) => (
            <span key={w.currency} className="rounded-md border px-1.5 py-0.5 font-mono text-[11.5px] text-muted-foreground">
              {moneyPrecise(w.balance, w.currency)}
            </span>
          ))}
        </div>
      )}
    </Tile>
  );
}

export function RenewalTile({ d }: { d: DashboardData }) {
  const t = useT();
  const renewal = d.next_renewal ?? null;
  const days = renewal ? daysUntil(renewal.expires_at) : null;
  const tone = days !== null && days <= 0 ? "danger" : days !== null && days <= 3 ? "warning" : undefined;
  return (
    <Tile
      label={t("home.renewal.title")}
      icon="ri-calendar-event-line"
      accent="var(--vx-muted)"
      delay={160}
      tone={tone}
      footer={
        renewal ? (
          <TileLink href={`/servers/${renewal.server_id}/tariff`}>{t("home.renewal.open")}</TileLink>
        ) : (
          <TileLink href="/rent-server">{t("home.renewal.rent")}</TileLink>
        )
      }
    >
      {renewal ? (
        <>
          <div
            className={cn(
              "text-[30px] leading-[1.2] font-semibold tracking-[-0.03em]",
              tone === "danger" && "text-rose-600 dark:text-rose-400",
              tone === "warning" && "text-amber-600 dark:text-amber-500"
            )}
          >
            {days !== null && days <= 0 ? t("home.renewal.expired") : shortDate(renewal.expires_at)}
          </div>
          <p className="mt-2 truncate text-[12.5px] text-muted-foreground">
            {renewal.name}
            {days !== null && days > 0 && ` · ${t("home.renewal.in", { days: pluralDays(days) })}`}
          </p>
          <p className="mt-1 text-[12.5px] text-muted-foreground">
            {renewal.auto_renew ? t("home.renewal.auto") : t("home.renewal.manual")}
            {renewal.cost > 0 && ` · ${moneyPrecise(renewal.cost, d.balance_currency)}`}
          </p>
        </>
      ) : (
        <>
          <div className="text-[30px] leading-[1.2] font-semibold tracking-[-0.03em] text-muted-foreground">—</div>
          <p className="mt-2 text-[12.5px] text-muted-foreground">{t("home.renewal.none")}</p>
        </>
      )}
    </Tile>
  );
}

export function SpendTile({ d }: { d: DashboardData }) {
  const t = useT();
  const spending = d.spending;
  const days = spending?.days ?? [];
  const peak = Math.max(1, ...days.map((day) => day.debit));
  const monthly = d.monthly_spend ?? 0;
  return (
    <Tile label={t("home.spend.title")} icon="ri-line-chart-line" accent="var(--vx-muted)" delay={220} footer={<TileLink href="/billing">{t("home.spend.history")}</TileLink>}>
      <div className="text-[30px] leading-[1.2] font-semibold tracking-[-0.03em] tabular-nums">
        {moneyPrecise(spending?.debit ?? 0, spending?.currency || d.balance_currency)}
      </div>
      {days.length > 0 && (
        <div className="mt-3 flex h-[26px] items-end gap-[3px]" aria-hidden>
          {days.map((day) => (
            <span
              key={day.date}
              className={cn("flex-1 rounded-[2px]", day.debit > 0 ? "bg-foreground/70" : "bg-muted")}
              style={{ height: `${day.debit > 0 ? Math.max(12, (day.debit / peak) * 100) : 8}%` }}
              title={`${shortDate(day.date)}: ${moneyPrecise(day.debit, spending?.currency || d.balance_currency)}`}
            />
          ))}
        </div>
      )}
      <p className="mt-2 text-[12.5px] text-muted-foreground">
        {monthly > 0
          ? t("home.spend.monthly", { amount: money(monthly, d.balance_currency) })
          : t("home.spend.none")}
      </p>
    </Tile>
  );
}

export function SupportTile({ d }: { d: DashboardData }) {
  const t = useT();
  const open = d.open_support_tickets_count;
  return (
    <Tile
      label={t("home.support.title")}
      icon="ri-customer-service-2-line"
      accent="var(--vx-muted)"
      delay={280}
      footer={
        open > 0 ? (
          <TileLink href="/support">{t("home.support.open_list")}</TileLink>
        ) : (
          <TileLink href="/support/create">{t("home.support.write")}</TileLink>
        )
      }
    >
      <div className="text-[36px] leading-[1.2] font-semibold tracking-[-0.03em] tabular-nums">{open}</div>
      <p className="mt-2 text-[12.5px] text-muted-foreground">
        {open > 0 ? t("home.support.open_hint") : t("home.support.none")}
      </p>
    </Tile>
  );
}
