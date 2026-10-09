"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowRight, Gift, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { countdown } from "@/components/user/account/shared";
import { pluralDays } from "@/components/user/panel-parts";
import { spinDailyBonus, type DailyBonusResponse, type DashboardData, type DashboardNews } from "@/lib/api";
import { localeTag } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";
import { moneyPrecise, relativeTime } from "./dashboard-utils";

function CardHeader({ title, href, link }: { title: string; href?: string; link?: string }) {
  return (
    <div className="flex items-center gap-3 border-b px-5 py-4">
      <h2 className="text-[16px] font-semibold">{title}</h2>
      {href && link && (
        <Link href={href} className="group ms-auto inline-flex items-center gap-1.5 text-[13px] font-medium">
          {link}
          <ArrowRight className="size-3.5 transition-transform group-hover:translate-x-0.5" />
        </Link>
      )}
    </div>
  );
}

export function BonusCard({ bonus }: { bonus: DailyBonusResponse }) {
  const t = useT();
  const qc = useQueryClient();
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (bonus.can_spin || !bonus.next_spin_at) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [bonus.can_spin, bonus.next_spin_at]);

  const [deg, setDeg] = useState(0);
  const spin = useMutation({
    mutationFn: spinDailyBonus,
    onSuccess: (res) => {
      void qc.invalidateQueries({ queryKey: ["daily-bonus"] });
      void qc.invalidateQueries({ queryKey: ["dashboard"] });
      if (res.prize_id) toast.success(t("home.bonus.won", { prize: res.prize?.label || "" }));
      else toast.info(t("home.bonus.no_prize"));
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const remaining = countdown(bonus.next_spin_at, now);
  const left = remaining && remaining !== "00:00:00" ? remaining : "";

  return (
    <section
      className="srv2-rise relative isolate overflow-hidden rounded-[22px] border border-amber-500/30 bg-card"
      style={{ animationDelay: "300ms" }}
      data-spotlight
    >
      <div className="pointer-events-none absolute inset-0 bg-gradient-to-br from-amber-500/[0.12] to-transparent" />
      <div className="relative px-[22px] py-5">
        <div className="flex items-center gap-3.5">
          <div className="relative size-24 shrink-0">
            <div className="absolute top-[-6px] left-1/2 z-[2] -ml-[7px] h-0 w-0 border-x-[7px] border-t-[14px] border-x-transparent border-t-foreground" />
            <div
              className="size-24 rounded-full border-[3px] border-card shadow-[0_0_0_1px_var(--border),0_0_28px_rgba(245,158,11,0.25)]"
              style={{
                background:
                  "repeating-conic-gradient(rgba(245,158,11,0.75) 0deg 45deg, var(--muted) 45deg 90deg)",
                transform: `rotate(${deg}deg)`,
                transition: "transform 4s cubic-bezier(.12,.7,.1,1)",
              }}
            />
            <div className="absolute inset-9 rounded-full border bg-card" />
          </div>
          <div className="min-w-0">
            <h2 className="text-[16px] font-semibold">{t("home.bonus.title")}</h2>
            {bonus.streak > 0 && (
              <p className="text-[12px] text-muted-foreground">{t("home.bonus.streak", { days: pluralDays(bonus.streak) })}</p>
            )}
            <p className="mt-1 text-[13px] leading-snug text-muted-foreground">
              {bonus.can_spin ? t("home.bonus.available") : t("home.bonus.done")}
            </p>
          </div>
        </div>
        <div className="mt-4 flex flex-wrap items-center gap-2.5">
          <Button
            size="sm"
            disabled={!bonus.can_spin || spin.isPending}
            onClick={() => {
              setDeg((value) => value + 1440 + Math.round(Math.random() * 300));
              spin.mutate();
            }}
            className={cn("rounded-full", bonus.can_spin && "bg-amber-500 text-white hover:bg-amber-500/90")}
          >
            {spin.isPending && <Loader2 className="animate-spin" />}
            {bonus.can_spin ? t("home.bonus.spin") : left ? t("home.bonus.next", { time: left }) : t("home.bonus.tomorrow")}
          </Button>
          <Button asChild size="sm" variant="ghost">
            <Link href="/daily-bonus">{t("home.bonus.more")}</Link>
          </Button>
        </div>
      </div>
    </section>
  );
}

export function RecentCard({ d }: { d: DashboardData }) {
  const t = useT();
  const items = d.recent_transactions.slice(0, 6);
  return (
    <section className="srv2-rise overflow-hidden rounded-[22px] border bg-card" style={{ animationDelay: "400ms" }}>
      <CardHeader title={t("home.recent.title")} href="/billing" link={t("home.recent.all")} />
      {items.length === 0 ? (
        <div className="px-5 py-8 text-center">
          <p className="text-[13.5px] text-muted-foreground">{t("home.recent.empty")}</p>
        </div>
      ) : (
        <ul className="divide-y">
          {items.map((tx) => {
            const credit = tx.type === "credit";
            const amount = Math.abs(Number(tx.amount) || 0);
            return (
              <li key={tx.id} className="flex items-center gap-3 px-5 py-3 transition-colors hover:bg-muted/30">
                <span
                  className={cn(
                    "grid size-8 shrink-0 place-items-center rounded-full text-[14px] font-semibold",
                    credit ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400" : "bg-muted text-muted-foreground"
                  )}
                >
                  {credit ? "+" : "−"}
                </span>
                <div className="min-w-0 flex-1">
                  <div className="truncate text-[13px] font-medium">{tx.description || t("home.recent.no_description")}</div>
                  <div className="mt-0.5 text-[11.5px] text-muted-foreground">{relativeTime(tx.created_at)}</div>
                </div>
                <span
                  className={cn(
                    "shrink-0 font-mono text-[13px] tabular-nums",
                    credit ? "text-emerald-600 dark:text-emerald-400" : "text-foreground"
                  )}
                >
                  {credit ? "+" : "−"}
                  {moneyPrecise(amount, d.balance_currency)}
                </span>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

export function NewsCard({ news }: { news: DashboardNews[] }) {
  const t = useT();
  if (news.length === 0) return null;
  return (
    <section className="srv2-rise overflow-hidden rounded-[22px] border bg-card" style={{ animationDelay: "600ms" }}>
      <CardHeader title={t("home.news.title")} href="/news" link={t("home.news.all")} />
      <div className="px-5 pb-2">
        {news.slice(0, 3).map((item) => (
          <Link
            key={item.id}
            href={`/news/${item.slug || item.id}`}
            className="block border-b py-[11px] last:border-b-0 hover:opacity-80"
          >
            <div className="font-mono text-[11px] text-muted-foreground">
              {item.published_at
                ? new Date(item.published_at).toLocaleDateString(localeTag(), { day: "numeric", month: "long" })
                : "—"}
            </div>
            <div className="mt-[3px] line-clamp-2 text-[13.5px] leading-snug">{item.title}</div>
          </Link>
        ))}
      </div>
    </section>
  );
}
