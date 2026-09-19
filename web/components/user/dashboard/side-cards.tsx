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
    <div className="flex items-center gap-3 border-b px-5 py-3.5">
      <h2 className="text-[15px] font-semibold">{title}</h2>
      {href && link && (
        <Link href={href} className="group ms-auto inline-flex items-center gap-1.5 text-[13px] font-medium text-primary">
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
    <section className="relative isolate overflow-hidden rounded-2xl border bg-card" data-spotlight>
      <div className="pointer-events-none absolute inset-x-0 top-0 h-24 bg-gradient-to-b from-amber-500/[0.08] to-transparent" />
      <div className="relative px-5 py-4">
        <div className="flex items-center gap-3">
          <span className="grid size-9 place-items-center rounded-xl bg-amber-500/10 text-amber-600 dark:text-amber-400">
            <Gift className="size-[18px]" />
          </span>
          <div className="min-w-0">
            <h2 className="text-[15px] font-semibold">{t("home.bonus.title")}</h2>
            {bonus.streak > 0 && (
              <p className="text-[12px] text-muted-foreground">{t("home.bonus.streak", { days: pluralDays(bonus.streak) })}</p>
            )}
          </div>
        </div>
        <p className="mt-3 text-[13px] leading-relaxed text-muted-foreground">
          {bonus.can_spin ? t("home.bonus.available") : t("home.bonus.done")}
        </p>
        <div className="mt-4 flex flex-wrap items-center gap-2">
          <Button
            size="sm"
            disabled={!bonus.can_spin || spin.isPending}
            onClick={() => spin.mutate()}
            className={cn(bonus.can_spin && "bg-amber-500 text-white hover:bg-amber-500/90")}
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
    <section className="overflow-hidden rounded-2xl border bg-card">
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
              <li key={tx.id} className="flex items-center gap-3 px-5 py-3">
                <span
                  className={cn(
                    "grid size-8 shrink-0 place-items-center rounded-lg text-[13px] font-semibold",
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
    <section className="overflow-hidden rounded-2xl border bg-card">
      <CardHeader title={t("home.news.title")} href="/news" link={t("home.news.all")} />
      <div className="grid gap-px bg-border sm:grid-cols-2 lg:grid-cols-3">
        {news.slice(0, 3).map((item) => (
          <Link
            key={item.id}
            href={`/news/${item.slug || item.id}`}
            className="group flex flex-col bg-card p-4 transition-colors hover:bg-muted/30"
          >
            {item.image ? (
              <div className="mb-3 aspect-[16/9] overflow-hidden rounded-xl border bg-muted">
                <img
                  src={item.image}
                  alt=""
                  className="size-full object-cover transition-transform duration-500 group-hover:scale-[1.03]"
                />
              </div>
            ) : (
              <span className="mb-3 grid size-9 place-items-center rounded-xl bg-muted text-muted-foreground">
                <i className="ri-newspaper-line text-[17px]" />
              </span>
            )}
            <div className="text-[11.5px] text-muted-foreground">
              {item.published_at
                ? new Date(item.published_at).toLocaleDateString(localeTag(), { day: "numeric", month: "long" })
                : "—"}
            </div>
            <h3 className="mt-1 line-clamp-2 text-[14px] leading-snug font-semibold">{item.title}</h3>
            {item.excerpt && <p className="mt-1 line-clamp-2 text-[12.5px] leading-snug text-muted-foreground">{item.excerpt}</p>}
          </Link>
        ))}
      </div>
    </section>
  );
}
