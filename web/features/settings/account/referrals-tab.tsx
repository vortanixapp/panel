"use client";

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Check, Copy, Share2, TriangleAlert } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useT } from "@/hooks/use-translations";
import { fetchAccountReferrals, type MoneyAmount } from "@/lib/api";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { formatDate, QueryState, SettingsSection, useCopy } from "./ui";

function formatMoney(amount: number, currency: string) {
  try {
    return new Intl.NumberFormat(localeTag(), { style: "currency", currency, maximumFractionDigits: 2 }).format(amount);
  } catch {
    return `${amount.toFixed(2)} ${currency}`;
  }
}

function moneyList(list: MoneyAmount[]) {
  return list.filter((item) => item.amount > 0).map((item) => formatMoney(item.amount, item.currency));
}

function Stat({ label, className, children }: { label: string; className?: string; children: React.ReactNode }) {
  return (
    <div className={cn("rounded-xl border border-border px-4 py-3", className)}>
      <div className="text-[12px] text-muted-foreground">{label}</div>
      <div className="mt-1 text-[18px] leading-tight font-semibold tabular-nums">{children}</div>
    </div>
  );
}

export function ReferralsTab() {
  const t = useT();
  const { copied, copy } = useCopy();
  const query = useQuery({ queryKey: ["account-referrals"], queryFn: fetchAccountReferrals });
  const [origin, setOrigin] = useState("");
  const [canShare, setCanShare] = useState(false);

  useEffect(() => {
    setOrigin(window.location.origin);
    setCanShare(typeof navigator !== "undefined" && typeof navigator.share === "function");
  }, []);

  const data = query.data;
  const link = data?.code && origin ? `${origin}/register?ref=${data.code}` : "";
  const earned = data ? moneyList(data.stats.earned) : [];

  const share = async () => {
    if (!link) return;
    try {
      await navigator.share({ title: t("settings.referrals.share_title"), text: t("settings.referrals.share_text"), url: link });
    } catch {
      return;
    }
  };

  const terms = data
    ? [
        t("settings.referrals.term_percent", { percent: data.percent }),
        data.months > 0
          ? t("settings.referrals.term_months", { months: data.months })
          : t("settings.referrals.term_forever"),
        ...(data.min_payment > 0
          ? [t("settings.referrals.term_min", { amount: formatMoney(data.min_payment, data.currency || "RUB") })]
          : []),
        t("settings.referrals.term_balance"),
        t("settings.referrals.term_refund"),
        t("settings.referrals.term_self"),
      ]
    : [];

  return (
    <>
      <SettingsSection
        title={t("settings.referrals.title")}
        description={
          data?.enabled
            ? t("settings.referrals.hint", { percent: data.percent })
            : t("settings.referrals.hint_off")
        }
      >
        <QueryState isLoading={query.isLoading} isError={query.isError} error={query.error} onRetry={() => void query.refetch()}>
          {data && (
            <div className="space-y-5">
              {!data.enabled && (
                <div className="flex items-start gap-2.5 rounded-xl border border-amber-500/30 bg-amber-500/5 px-4 py-3 text-[13px] text-amber-700 dark:text-amber-400">
                  <TriangleAlert className="mt-0.5 size-4 shrink-0" />
                  <span>{t("settings.referrals.paused")}</span>
                </div>
              )}

              {data.enabled && link && (
                <div className="space-y-2">
                  <div className="text-[13px] font-medium">{t("settings.referrals.link")}</div>
                  <div className="flex flex-wrap items-stretch gap-2">
                    <code className="min-w-0 flex-1 truncate rounded-lg border border-border bg-muted/40 px-3 py-2 font-mono text-[13px] select-all">
                      {link}
                    </code>
                    <Button type="button" variant="outline" onClick={() => void copy(link, "link")}>
                      {copied === "link" ? <Check className="size-4" /> : <Copy className="size-4" />}
                      {t("settings.common.copy")}
                    </Button>
                    {canShare && (
                      <Button type="button" variant="outline" onClick={() => void share()}>
                        <Share2 className="size-4" />
                        <span className="max-sm:sr-only">{t("settings.referrals.share")}</span>
                      </Button>
                    )}
                  </div>
                  <p className="text-[12px] text-muted-foreground">
                    {t("settings.referrals.code_hint", { code: data.code })}
                  </p>
                </div>
              )}

              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                <Stat label={t("settings.referrals.stat_invited")}>{data.stats.invited}</Stat>
                <Stat label={t("settings.referrals.stat_paid")}>{data.stats.paid}</Stat>
                <Stat label={t("settings.referrals.stat_earned")} className="col-span-2 sm:col-span-1">
                  {earned.length > 0 ? (
                    <span className="flex flex-col">
                      {earned.map((line) => (
                        <span key={line}>{line}</span>
                      ))}
                    </span>
                  ) : (
                    formatMoney(0, data.currency || "RUB")
                  )}
                </Stat>
              </div>
            </div>
          )}
        </QueryState>
      </SettingsSection>

      {data && (
        <SettingsSection title={t("settings.referrals.terms_title")}>
          <ul className="space-y-2 text-[13px]">
            {terms.map((line) => (
              <li key={line} className="flex items-start gap-2.5">
                <Check className="mt-0.5 size-4 shrink-0 text-emerald-500" />
                <span>{line}</span>
              </li>
            ))}
          </ul>
        </SettingsSection>
      )}

      {data && (
        <SettingsSection
          title={t("settings.referrals.list_title")}
          description={data.stats.invited > data.referrals.length ? t("settings.referrals.list_limited", { count: data.referrals.length }) : undefined}
        >
          {data.referrals.length === 0 ? (
            <p className="text-[13px] text-muted-foreground">{t("settings.referrals.list_empty")}</p>
          ) : (
            <ul className="divide-y divide-border rounded-xl border border-border">
              {data.referrals.map((item, index) => {
                const amounts = moneyList(item.earned);
                return (
                  <li key={`${item.email}-${item.joined_at}-${index}`} className="flex flex-wrap items-center gap-3 px-4 py-3">
                    <i className="ri-user-3-line text-lg text-muted-foreground" />
                    <div className="min-w-0 flex-1">
                      <div className={cn("truncate text-[13.5px] font-medium", item.deleted && "text-muted-foreground")}>
                        {item.deleted ? t("settings.referrals.deleted") : item.email}
                      </div>
                      <div className="mt-0.5 flex flex-wrap gap-x-3 text-[12px] text-muted-foreground">
                        <span>{t("settings.referrals.joined", { date: formatDate(item.joined_at) })}</span>
                        {item.active_until && (
                          <span className={cn(!item.active && "text-muted-foreground/70")}>
                            {item.active
                              ? t("settings.referrals.active_until", { date: formatDate(item.active_until) })
                              : t("settings.referrals.expired")}
                          </span>
                        )}
                      </div>
                    </div>
                    {amounts.length > 0 ? (
                      <Badge variant="secondary" className="text-emerald-600 dark:text-emerald-400">
                        +{amounts.join(" · ")}
                      </Badge>
                    ) : (
                      <Badge variant="outline" className="font-normal text-muted-foreground">
                        {t("settings.referrals.no_payments")}
                      </Badge>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
        </SettingsSection>
      )}
    </>
  );
}
