"use client";

import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { pluralDays } from "@/components/user/panel-parts";
import type { AccountUser, DashboardData } from "@/lib/api";
import type { TranslateFn } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";
import { daysUntil, moneyPrecise, shortDate } from "./dashboard-utils";

type Tone = "danger" | "warning" | "info";

export type AttentionItem = {
  id: string;
  tone: Tone;
  icon: string;
  title: string;
  text?: string;
  href?: string;
  action?: string;
};

const TONE: Record<Tone, string> = {
  danger: "bg-rose-500/10 text-rose-600 dark:text-rose-400",
  warning: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  info: "bg-sky-500/10 text-sky-600 dark:text-sky-400",
};

export function buildAttention(d: DashboardData, account: AccountUser | undefined, t: TranslateFn): AttentionItem[] {
  const items: AttentionItem[] = [];
  const currency = d.balance_currency;
  const renewal = d.next_renewal ?? null;

  for (const s of d.recent_servers) {
    if (s.is_blocked) {
      items.push({
        id: `blocked-${s.id}`,
        tone: "danger",
        icon: "ri-lock-line",
        title: t("home.attention.blocked", { name: s.name }),
        text: s.blocked_reason || t("home.attention.blocked_hint"),
        href: `/servers/${s.id}`,
        action: t("home.attention.open"),
      });
      continue;
    }
    if ((s.provisioning_status || "").toLowerCase() === "failed") {
      items.push({
        id: `failed-${s.id}`,
        tone: "danger",
        icon: "ri-error-warning-line",
        title: t("home.attention.install_failed", { name: s.name }),
        text: s.provisioning_error || t("home.attention.install_failed_hint"),
        href: `/servers/${s.id}`,
        action: t("home.attention.open"),
      });
      continue;
    }
    if (s.billing_source === "whmcs") continue;
    const days = daysUntil(s.expires_at);
    if (days === null) continue;
    if (days <= 0) {
      items.push({
        id: `expired-${s.id}`,
        tone: "danger",
        icon: "ri-time-line",
        title: t("home.attention.expired", { name: s.name }),
        text: t("home.attention.expired_hint"),
        href: `/servers/${s.id}/tariff`,
        action: t("home.attention.renew"),
      });
    } else if (days <= 3) {
      const cost = renewal && renewal.server_id === s.id ? renewal.cost : 0;
      const short = s.auto_renew && cost > 0 && d.balance < cost;
      items.push({
        id: `expiring-${s.id}`,
        tone: "warning",
        icon: "ri-time-line",
        title: t("home.attention.expiring", { name: s.name, days: pluralDays(days) }),
        text: s.auto_renew
          ? short
            ? t("home.attention.auto_short", { amount: moneyPrecise(cost - d.balance, currency) })
            : t("home.attention.auto_ok")
          : t("home.attention.manual_hint"),
        href: short ? "/billing#topup" : `/servers/${s.id}/tariff`,
        action: short ? t("home.attention.topup") : t("home.attention.renew"),
      });
    }
  }

  if (renewal && renewal.auto_renew && renewal.cost > d.balance && !items.some((i) => i.id.endsWith(renewal.server_id))) {
    const days = daysUntil(renewal.expires_at);
    if (days !== null && days > 0 && days <= 7) {
      items.push({
        id: `balance-${renewal.server_id}`,
        tone: "warning",
        icon: "ri-wallet-3-line",
        title: t("home.attention.balance_short", {
          amount: moneyPrecise(renewal.cost - d.balance, currency),
          name: renewal.name,
        }),
        text: t("home.attention.balance_short_hint", { date: shortDate(renewal.expires_at) }),
        href: "/billing#topup",
        action: t("home.attention.topup"),
      });
    }
  }

  const seen = new Set<string>();
  for (const s of d.recent_servers) {
    const m = s.location?.maintenance;
    if (!m?.enabled || !s.location) continue;
    const key = s.location.name;
    if (seen.has(key)) continue;
    seen.add(key);
    items.push({
      id: `maintenance-${key}`,
      tone: "info",
      icon: "ri-tools-line",
      title: t("home.attention.maintenance", { location: s.location.city || s.location.name }),
      text: [m.reason, m.until ? t("home.attention.maintenance_until", { date: shortDate(m.until) }) : ""]
        .filter(Boolean)
        .join(" · "),
    });
  }

  if (account && account.email_verified === false && !account.email.endsWith("@telegram.local")) {
    items.push({
      id: "email",
      tone: "info",
      icon: "ri-mail-line",
      title: t("home.attention.email"),
      text: t("home.attention.email_hint"),
      href: "/settings?tab=contacts",
      action: t("home.attention.email_action"),
    });
  }

  const order: Record<Tone, number> = { danger: 0, warning: 1, info: 2 };
  return items.sort((a, b) => order[a.tone] - order[b.tone]);
}

export function AttentionCard({ items }: { items: AttentionItem[] }) {
  const t = useT();
  if (items.length === 0) return null;
  return (
    <section className="overflow-hidden rounded-2xl border bg-card">
      <div className="border-b px-5 py-3.5 sm:px-6">
        <h2 className="text-[15px] font-semibold">{t("home.attention.title")}</h2>
      </div>
      <ul className="divide-y">
        {items.map((item) => (
          <li key={item.id} className="flex flex-wrap items-center gap-x-4 gap-y-2.5 px-5 py-3.5 sm:px-6">
            <span className={cn("grid size-9 shrink-0 place-items-center rounded-xl text-[17px]", TONE[item.tone])}>
              <i className={item.icon} />
            </span>
            <div className="min-w-0 flex-1 basis-56">
              <div className="text-[14px] font-medium">{item.title}</div>
              {item.text && <div className="mt-0.5 text-[12.5px] text-muted-foreground">{item.text}</div>}
            </div>
            {item.href && item.action && (
              <Button asChild size="sm" variant={item.tone === "info" ? "outline" : "default"} className="ms-auto">
                <Link href={item.href}>
                  {item.action}
                  <ArrowRight />
                </Link>
              </Button>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}
