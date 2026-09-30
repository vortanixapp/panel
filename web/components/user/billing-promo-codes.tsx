"use client";

import Link from "next/link";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { fetchPromoCodes, type UserPromoCode } from "@/lib/api";
import { formatAmount } from "@/lib/format";
import { localeTag, t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

const CARD = "rounded-[20px] border border-[var(--vx-panel-line)] bg-[var(--vx-panel-card)]";

const STATUS_ORDER: Record<UserPromoCode["status"], number> = {
  active: 0,
  upcoming: 1,
  used: 2,
  expired: 3,
  disabled: 4,
};

const STATUS_TONE: Record<UserPromoCode["status"], string> = {
  active: "border-emerald-500/30 bg-emerald-500/10 text-emerald-500",
  upcoming: "border-[var(--vx-border-2)] text-muted-foreground",
  used: "border-[var(--vx-border-2)] text-[var(--vx-ink-faint)]",
  expired: "border-[var(--vx-border-2)] text-[var(--vx-ink-faint)]",
  disabled: "border-[var(--vx-border-2)] text-[var(--vx-ink-faint)]",
};

function fmtDate(iso: string) {
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? iso : date.toLocaleDateString(localeTag());
}

function benefit(promo: UserPromoCode, symbol: string): string {
  const parts: string[] = [];
  if (promo.discount_value > 0) {
    parts.push(
      promo.discount_type === "fixed"
        ? t("billing.promos.discount_fixed", {
            value: `${formatAmount(promo.discount_value)} ${symbol}`,
          })
        : t("billing.promos.discount_percent", { value: formatAmount(promo.discount_value, 0) })
    );
  }
  if (promo.bonus_percent > 0) {
    parts.push(t("billing.promos.bonus_percent", { value: formatAmount(promo.bonus_percent, 0) }));
  }
  if (promo.bonus_fixed > 0) {
    parts.push(
      t("billing.promos.bonus_fixed", { value: `${formatAmount(promo.bonus_fixed)} ${symbol}` })
    );
  }
  return parts.join(" · ");
}

function scopes(promo: UserPromoCode): string {
  if (promo.applies_to.length === 0) return t("billing.promos.scope_all");
  return promo.applies_to.map((scope) => t(`billing.promos.scope_${scope}`)).join(", ");
}

function validity(promo: UserPromoCode): string {
  if (promo.status === "upcoming" && promo.starts_at) {
    return t("billing.promos.from", { date: fmtDate(promo.starts_at) });
  }
  return promo.ends_at
    ? t("billing.promos.until", { date: fmtDate(promo.ends_at) })
    : t("billing.promos.endless");
}

async function copyCode(code: string) {
  try {
    await navigator.clipboard.writeText(code);
    toast.success(t("common.copied"));
  } catch {
    toast.error(t("common.copy_failed"));
  }
}

function PromoRow({
  promo,
  symbol,
  onUse,
}: {
  promo: UserPromoCode;
  symbol: string;
  onUse?: (code: string) => void;
}) {
  const live = promo.status === "active";
  const canUse =
    live && onUse && (promo.applies_to.length === 0 || promo.applies_to.includes("topup"));
  const summary = benefit(promo, symbol);
  return (
    <div
      className={cn(
        "flex flex-col gap-2.5 border-b border-[var(--vx-inset)] px-5 py-4 last:border-b-0 sm:px-[26px]",
        !live && "opacity-60"
      )}
    >
      <div className="flex flex-wrap items-center justify-between gap-2.5">
        <button
          type="button"
          onClick={() => void copyCode(promo.code)}
          title={t("billing.promos.copy_code")}
          className="inline-flex items-center gap-2 rounded-[10px] border border-dashed border-[var(--vx-border-strong)] bg-[var(--vx-card-2)] px-3 py-1.5 font-mono text-[14px] tracking-[0.06em] transition-colors hover:border-[var(--vx-border-hover)]"
        >
          {promo.code}
          <i className="ri-file-copy-line text-[14px] text-[var(--vx-ink-faint)]" />
        </button>
        <span
          className={cn(
            "rounded-full border px-2.5 py-0.5 text-[11px] font-medium",
            STATUS_TONE[promo.status]
          )}
        >
          {t(`billing.promos.status_${promo.status}`)}
        </span>
      </div>

      <div className="min-w-0">
        <div className="truncate text-[13.5px] font-medium">{promo.title}</div>
        {summary && <div className="mt-0.5 text-[13px] text-foreground">{summary}</div>}
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2.5">
        <span className="font-mono text-[11px] text-[var(--vx-ink-faint)]">
          {[
            scopes(promo),
            promo.min_amount
              ? t("billing.promos.min_amount", {
                  amount: `${formatAmount(promo.min_amount, 0)} ${symbol}`,
                })
              : "",
            validity(promo),
          ]
            .filter(Boolean)
            .join(" · ")}
        </span>
        {canUse && (
          <button
            type="button"
            onClick={() => onUse(promo.code)}
            className="rounded-full border border-[var(--vx-border-2)] px-3.5 py-1.5 text-[12.5px] text-muted-foreground transition-colors hover:border-[var(--vx-border-hover)] hover:text-foreground"
          >
            {t("billing.promos.use")}
          </button>
        )}
      </div>
    </div>
  );
}

export function BillingPromoCodes({
  symbol,
  onUse,
}: {
  symbol: string;
  onUse?: (code: string) => void;
}) {
  useT();
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.billingPromoCodes(),
    queryFn: fetchPromoCodes,
  });

  const promos = useMemo(
    () =>
      [...(data?.promo_codes ?? [])].sort(
        (a, b) => STATUS_ORDER[a.status] - STATUS_ORDER[b.status]
      ),
    [data]
  );

  if (isLoading) return null;

  return (
    <div className={cn(CARD, "overflow-hidden")}>
      <div className="flex flex-wrap items-center gap-3.5 border-b border-[var(--vx-panel-line)] px-5 py-5 sm:px-[26px]">
        <span className="text-base font-semibold">{t("billing.promos.title")}</span>
        <span className="text-xs text-[var(--vx-ink-faint)]">{t("billing.promos.hint")}</span>
      </div>
      {promos.length === 0 ? (
        <div className="flex flex-wrap items-center justify-between gap-3 px-5 py-5 text-[13px] text-muted-foreground sm:px-[26px]">
          <span className="flex items-center gap-2.5">
            <i className="ri-coupon-3-line text-[17px] text-[var(--vx-ink-faint)]" />
            {t("billing.promos.empty")}
          </span>
          <Link
            href="/daily-bonus"
            className="rounded-full border border-[var(--vx-border-2)] px-3.5 py-1.5 text-[12.5px] transition-colors hover:border-[var(--vx-border-hover)] hover:text-foreground"
          >
            {t("billing.promos.empty_cta")}
          </Link>
        </div>
      ) : (
        promos.map((promo) => (
          <PromoRow key={promo.id} promo={promo} symbol={symbol} onUse={onUse} />
        ))
      )}
    </div>
  );
}
