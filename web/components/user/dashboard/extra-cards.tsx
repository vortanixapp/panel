"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { ArrowRight, Check } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { fetchAccountReferrals, type AccountUser, type DashboardData } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";
import { moneyPrecise, shortDate } from "./dashboard-utils";

const LIVE_POINTS = 30;
const CHART_W = 600;
const CHART_H = 160;

function linePath(values: number[], max: number) {
  const pts = values.length === 1 ? [values[0], values[0]] : values;
  const stepX = CHART_W / Math.max(1, pts.length - 1);
  const line = pts
    .map((v, i) => {
      const y = CHART_H - 8 - (Math.max(0, v) / max) * (CHART_H - 16);
      return `${i === 0 ? "M" : "L"}${(i * stepX).toFixed(1)} ${y.toFixed(1)}`;
    })
    .join("");
  return { line, area: `${line}L${CHART_W} ${CHART_H}L0 ${CHART_H}Z` };
}

export function LiveCard({ d }: { d: DashboardData }) {
  const t = useT();
  const total = useMemo(
    () => d.recent_servers.reduce((sum, server) => sum + (server.online_players ?? 0), 0),
    [d.recent_servers]
  );
  const buffer = useRef<number[]>([]);
  const [samples, setSamples] = useState<number[]>([total, total]);

  useEffect(() => {
    buffer.current = [...buffer.current, total].slice(-LIVE_POINTS);
    setSamples(buffer.current.length > 1 ? buffer.current : [total, total]);
  }, [d, total]);

  const max = Math.max(5, ...samples);
  const { line, area } = linePath(samples, max);

  return (
    <section className="srv2-rise rounded-[22px] border bg-card px-6 py-[22px]" style={{ animationDelay: "400ms" }}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-[16px] font-semibold">{t("home.live.title")}</h2>
          <div className="mt-0.5 text-[12.5px] text-muted-foreground">{t("home.live.hint")}</div>
        </div>
        <span className="inline-flex items-center gap-2 font-mono text-[13px]">
          <span className="srv2-glow-dot size-2 rounded-full bg-foreground" />
          {t("home.live.online", { count: total })}
        </span>
      </div>
      <svg viewBox={`0 0 ${CHART_W} ${CHART_H}`} preserveAspectRatio="none" className="mt-3.5 block h-[170px] w-full">
        <defs>
          <linearGradient id="srv2-live" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="currentColor" stopOpacity="0.22" />
            <stop offset="1" stopColor="currentColor" stopOpacity="0" />
          </linearGradient>
        </defs>
        <g className="stroke-border">
          {[20, 60, 100, 140].map((y) => (
            <line key={y} x1="0" x2={CHART_W} y1={y} y2={y} />
          ))}
        </g>
        <path d={area} fill="url(#srv2-live)" className="text-foreground" />
        <path
          d={line}
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinejoin="round"
          vectorEffect="non-scaling-stroke"
          className="text-foreground"
        />
      </svg>
      <div className="mt-2 flex justify-between font-mono text-[11px] text-muted-foreground/70">
        <span>{t("home.live.earlier")}</span>
        <span>{t("home.live.now")}</span>
      </div>
    </section>
  );
}

export function SpendCard({ d }: { d: DashboardData }) {
  const t = useT();
  const days = d.spending?.days ?? [];
  const currency = d.spending?.currency || d.balance_currency;
  const peak = Math.max(1, ...days.map((day) => day.debit));
  return (
    <section className="srv2-rise rounded-[22px] border bg-card px-6 py-[22px]" style={{ animationDelay: "500ms" }}>
      <div className="flex items-center justify-between">
        <h2 className="text-[16px] font-semibold">{t("home.spend30.title")}</h2>
        <span className="font-mono text-[13px] text-muted-foreground">{moneyPrecise(d.spending?.debit ?? 0, currency)}</span>
      </div>
      {days.length === 0 || (d.spending?.debit ?? 0) <= 0 ? (
        <div className="py-10 text-center text-[13px] text-muted-foreground">{t("home.spend30.empty")}</div>
      ) : (
        <>
          <div className="mt-[18px] flex h-[150px] items-end gap-[3px] sm:gap-[5px]">
            {days.map((day) => (
              <div key={day.date} className="flex h-full flex-1 flex-col justify-end" title={`${shortDate(day.date)}: ${moneyPrecise(day.debit, currency)}`}>
                <div
                  className={cn("min-h-[2px] rounded-t-[4px] rounded-b-[1px]", day.debit > 0 ? "bg-foreground/70" : "bg-muted")}
                  style={{ height: `${day.debit > 0 ? Math.max(4, (day.debit / peak) * 100) : 1}%` }}
                />
              </div>
            ))}
          </div>
          <div className="mt-2 flex justify-between font-mono text-[11px] text-muted-foreground/70">
            <span>{shortDate(days[0].date)}</span>
            <span>{shortDate(days[days.length - 1].date)}</span>
          </div>
        </>
      )}
    </section>
  );
}

export function QuickStartCard({ d, account }: { d: DashboardData; account?: AccountUser }) {
  const t = useT();
  const referrals = useQuery({ queryKey: ["account-referrals"], queryFn: fetchAccountReferrals, retry: false });
  const steps = [
    {
      title: t("home.quick.topup"),
      done: d.balance > 0 || d.recent_transactions.some((tx) => tx.type === "credit"),
      href: "/billing#topup",
    },
    { title: t("home.quick.rent"), done: d.total_servers > 0, href: "/rent-server" },
    { title: t("home.quick.email"), done: !account || account.email_verified !== false, href: "/settings?tab=contacts" },
    ...(referrals.data?.enabled
      ? [{ title: t("home.quick.invite"), done: referrals.data.stats.invited > 0, href: "/settings?tab=referrals" }]
      : []),
  ];
  const doneCount = steps.filter((step) => step.done).length;
  if (doneCount === steps.length) return null;
  return (
    <section className="srv2-rise rounded-[22px] border bg-card px-[22px] py-5" style={{ animationDelay: "500ms" }}>
      <h2 className="mb-3.5 text-[16px] font-semibold">{t("home.quick.title")}</h2>
      {steps.map((step, index) => (
        <Link key={step.title} href={step.href} className="group flex items-center gap-3 py-[9px]">
          <span
            className={cn(
              "grid size-[26px] shrink-0 place-items-center rounded-full border-[1.5px] text-[12px] font-bold",
              step.done ? "border-foreground bg-foreground text-background" : "border-border text-muted-foreground"
            )}
          >
            {step.done ? <Check className="size-3.5" /> : index + 1}
          </span>
          <span className={cn("flex-1", step.done && "text-muted-foreground")}>{step.title}</span>
          {!step.done && <ArrowRight className="size-3.5 text-muted-foreground transition-transform group-hover:translate-x-0.5" />}
        </Link>
      ))}
      <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
        <div className="h-full bg-foreground transition-[width] duration-300" style={{ width: `${(doneCount / steps.length) * 100}%` }} />
      </div>
    </section>
  );
}
