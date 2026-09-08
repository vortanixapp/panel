"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMutation, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  ArrowLeft,
  Database,
  Folder,
  Globe,
  HardDrive,
  Mail,
  ShoppingCart,
} from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchBilling,
  fetchHostingRentForm,
  submitHostingRent,
  type HostingPlanSummary,
} from "@/lib/api";
import { formatAmount } from "@/lib/format";
import { planDiskLabel, planFeatureLimit } from "@/lib/hosting-status";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { btnPrimary, fieldClass } from "@/components/user/panel-parts";
import { useT } from "@/hooks/use-translations";

/** Скидка за период в процентах, как её задаёт тариф. */
function periodDiscount(plan: HostingPlanSummary | undefined, period: number): number {
  return plan?.discounts?.[String(period)] ?? 0;
}

function basePrice(plan: HostingPlanSummary | undefined, period: number): number {
  if (!plan) return 0;
  return plan.price_monthly * (period / 30);
}

function planTags(plan: HostingPlanSummary): string[] {
  const tags: string[] = [];
  if (plan.has_ssl) tags.push("SSL");
  if (plan.has_ssh) tags.push("SSH");
  if (plan.has_cron) tags.push("Cron");
  if (plan.has_backup) tags.push(t("billing.hosting.tag_backup"));
  return tags;
}

function PlanCard({
  plan,
  selected,
  onSelect,
}: {
  plan: HostingPlanSummary;
  selected: boolean;
  onSelect: () => void;
}) {
  const inf = planFeatureLimit;
  const specs = [
    { Icon: HardDrive, text: planDiskLabel(plan) },
    {
      Icon: Globe,
      text: t("billing.hosting.rent.sites", { count: inf(plan.features?.sites) }),
    },
    {
      Icon: Database,
      text: t("billing.hosting.rent.databases", {
        count: inf(plan.features?.databases),
      }),
    },
    {
      Icon: Mail,
      text: t("billing.hosting.rent.email", {
        count: inf(plan.features?.email_domains),
      }),
    },
    {
      Icon: Folder,
      text: t("billing.hosting.rent.sftp", {
        count: inf(plan.features?.ftp_accounts),
      }),
    },
  ];

  return (
    <button
      type="button"
      onClick={onSelect}
      aria-pressed={selected}
      className={cn(
        "flex flex-col gap-[14px] rounded-2xl border px-5 py-[18px] text-start transition-colors",
        selected
          ? "border-ring bg-muted/40"
          : "border-border bg-card hover:border-[var(--vx-panel-hover-line)]"
      )}
    >
      <div className="flex flex-wrap items-center gap-3">
        <span
          className={cn(
            "flex h-[18px] w-[18px] flex-none items-center justify-center rounded-full border",
            selected ? "border-primary bg-primary" : "border-ring"
          )}
        >
          {selected && <span className="h-2 w-2 rounded-full bg-primary-foreground" />}
        </span>
        <span className="text-base font-semibold">{plan.name}</span>
        {plan.hosting_server?.panel_type_label && (
          <span className="rounded-full border border-border px-[9px] py-[3px] text-[11.5px] text-muted-foreground">
            {plan.hosting_server.panel_type_label}
          </span>
        )}
        <span className="ms-auto flex items-baseline gap-1.5">
          <span className="font-mono text-[22px] font-bold tracking-[-0.02em]">
            {formatAmount(plan.price_monthly)}
          </span>
          <span className="text-[12.5px] text-muted-foreground">
            {t("billing.hosting.rent.per_month")}
          </span>
        </span>
      </div>

      <div className="grid grid-cols-[repeat(auto-fit,minmax(120px,1fr))] gap-2.5">
        {specs.map(({ Icon, text }) => (
          <span
            key={text}
            className="flex items-center gap-2 text-[12.5px] text-[var(--vx-ink-dim)]"
          >
            <Icon className="h-3.5 w-3.5 flex-none text-muted-foreground" />
            {text}
          </span>
        ))}
      </div>

      <div className="flex flex-wrap items-center gap-1.5 border-t border-border pt-3">
        {planTags(plan).map((t) => (
          <span
            key={t}
            className="rounded-[7px] bg-muted px-[9px] py-[3px] text-[11.5px] text-[var(--vx-ink-dim)]"
          >
            {t}
          </span>
        ))}
        {plan.description && (
          <span className="ms-auto text-xs text-muted-foreground/70">{plan.description}</span>
        )}
      </div>
    </button>
  );
}

export function HostingRentPageContent() {
  // Карточки тарифов зовут t() напрямую, здесь хук нужен ради подписки:
  // без него страница не перерисуется при смене языка.
  useT();
  const router = useRouter();
  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(null);
  const [domain, setDomain] = useState("");
  const [period, setPeriod] = useState(30);
  const [walletId, setWalletId] = useState("");

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.hostingRent,
    queryFn: fetchHostingRentForm,
  });

  const { data: billing } = useQuery({
    queryKey: queryKeys.billing(),
    queryFn: () => fetchBilling(),
  });

  const plans = data?.plans ?? [];
  const wallets = data?.wallets?.length ? data.wallets : (billing?.wallets ?? []);

  useEffect(() => {
    if (wallets.length > 0 && !walletId) setWalletId(wallets[0].id);
  }, [wallets, walletId]);

  // Тариф выбран заранее: боковая панель заказа показывает итог с первой
  // секунды, а не пустует до первого клика.
  useEffect(() => {
    if (!selectedPlanId && plans.length > 0) {
      const first = plans[0];
      setSelectedPlanId(first.id);
      if (first.rental_periods?.length) setPeriod(first.rental_periods[0]);
    }
  }, [plans, selectedPlanId]);

  const selectedPlan = plans.find((p) => p.id === selectedPlanId);
  const periods = selectedPlan?.rental_periods?.length
    ? selectedPlan.rental_periods
    : [30];

  const base = basePrice(selectedPlan, period);
  const discount = periodDiscount(selectedPlan, period);
  const discountValue = base * (discount / 100);
  const total = base - discountValue;

  const selectPlan = useCallback((plan: HostingPlanSummary) => {
    setSelectedPlanId(plan.id);
    // Период сбрасываем на первый доступный: у другого тарифа набор может быть
    // иным, и прежний выбор считался бы по несуществующей скидке.
    if (plan.rental_periods?.length) setPeriod(plan.rental_periods[0]);
  }, []);

  const rentMutation = useMutation({
    mutationFn: () =>
      submitHostingRent({
        plan_id: selectedPlanId!,
        domain,
        period,
        wallet_id: walletId,
      }),
    onSuccess: (res) => {
      toast.success(t("billing.hosting.rent.success"));
      router.push(`/hosting/${res.id}`);
    },
    onError: (err: Error) =>
      toast.error(err.message || t("billing.hosting.rent.error")),
  });

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!selectedPlanId || !domain.trim()) return;
    rentMutation.mutate();
  }

  if (isLoading) {
    return (
      <PageShell variant="user">
        <div className="flex flex-col gap-[22px]">
          <Skeleton className="h-12 w-72 rounded-xl" />
          <div className="grid grid-cols-1 gap-5 lg:grid-cols-[minmax(0,1fr)_340px]">
            <div className="flex flex-col gap-2.5">
              <Skeleton className="h-44 rounded-2xl" />
              <Skeleton className="h-44 rounded-2xl" />
            </div>
            <Skeleton className="h-96 rounded-2xl" />
          </div>
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="user">
      <form onSubmit={onSubmit} className="flex w-full flex-col gap-[22px]">
        <div className="flex items-center gap-[14px]">
          <Link
            href="/hosting/my"
            className="flex h-[34px] w-[34px] flex-none items-center justify-center rounded-[9px] border border-input text-foreground transition-colors hover:bg-accent"
            aria-label={t("billing.hosting.back_aria")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div className="flex flex-col gap-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-[-0.02em]">
              {t("billing.hosting.rent.title")}
            </h1>
            <p className="text-[13.5px] text-muted-foreground">
              {t("billing.hosting.rent.subtitle")}
            </p>
          </div>
        </div>

        {isError && (
          <p className="text-sm text-destructive">
            {t("billing.hosting.rent.load_failed")}
          </p>
        )}

        {plans.length === 0 ? (
          <div className="rounded-2xl border border-border bg-card px-5 py-12 text-center">
            <p className="text-sm font-medium">{t("billing.hosting.rent.no_plans")}</p>
            <p className="mt-1 text-[12.5px] text-muted-foreground">
              {t("billing.hosting.rent.no_plans_hint")}
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,1fr)_340px]">
            <div className="flex flex-col gap-2.5">
              {plans.map((plan) => (
                <PlanCard
                  key={plan.id}
                  plan={plan}
                  selected={selectedPlanId === plan.id}
                  onSelect={() => selectPlan(plan)}
                />
              ))}
            </div>

            <div className="flex flex-col gap-[14px] rounded-2xl border border-border bg-card p-5 lg:sticky lg:top-[88px]">
              <div className="text-[15px] font-semibold">
                {t("billing.hosting.rent.order")}
              </div>

              <label className="flex flex-col gap-[7px]">
                <span className="text-[12.5px] text-muted-foreground">
                  {t("billing.hosting.rent.domain")}
                </span>
                <input
                  type="text"
                  value={domain}
                  onChange={(e) => setDomain(e.target.value)}
                  required
                  placeholder="example.com"
                  className={fieldClass}
                />
                <span className="text-[11.5px] text-muted-foreground/70">
                  {t("billing.hosting.rent.domain_hint")}
                </span>
              </label>

              <div className="flex flex-col gap-[7px]">
                <span className="text-[12.5px] text-muted-foreground">
                  {t("common.period")}
                </span>
                <div className="grid grid-cols-2 gap-1.5">
                  {periods.map((p) => {
                    const d = periodDiscount(selectedPlan, p);
                    const active = period === p;
                    return (
                      <button
                        key={p}
                        type="button"
                        onClick={() => setPeriod(p)}
                        aria-pressed={active}
                        className={cn(
                          "flex flex-col gap-0.5 rounded-[9px] border px-[11px] py-[9px] text-start text-[12.5px] transition-colors",
                          active
                            ? "border-ring bg-muted"
                            : "border-border bg-background hover:border-ring"
                        )}
                      >
                        <span className="font-medium">
                          {t("billing.hosting.days", { days: p })}
                        </span>
                        <span
                          className={cn(
                            "text-[11px]",
                            d ? "text-[var(--vx-info)]" : "text-muted-foreground/70"
                          )}
                        >
                          {d ? `−${d}%` : t("billing.hosting.rent.no_discount")}
                        </span>
                      </button>
                    );
                  })}
                </div>
              </div>

              {wallets.length > 0 && (
                <label className="flex flex-col gap-[7px]">
                  <span className="text-[12.5px] text-muted-foreground">
                    {t("billing.wallet.title")}
                  </span>
                  <select
                    value={walletId}
                    onChange={(e) => setWalletId(e.target.value)}
                    className={cn(fieldClass, "px-2.5")}
                  >
                    {wallets.map((w) => (
                      <option key={w.id} value={w.id}>
                        {w.currency.toUpperCase()} — {formatAmount(w.balance)}
                      </option>
                    ))}
                  </select>
                </label>
              )}

              <div className="h-px bg-border" />

              <div className="flex flex-col gap-2 text-[13px]">
                <div className="flex justify-between text-muted-foreground">
                  <span>
                    {t("billing.hosting.rent.summary_row", {
                      plan: selectedPlan?.name ?? t("billing.hosting.rent.plan_fallback"),
                      days: period,
                    })}
                  </span>
                  <span className="font-mono text-foreground">{formatAmount(base)}</span>
                </div>
                <div className="flex justify-between text-muted-foreground">
                  <span>{t("billing.hosting.rent.period_discount")}</span>
                  <span
                    className={cn(
                      "font-mono",
                      discount ? "text-[var(--vx-info)]" : "text-muted-foreground"
                    )}
                  >
                    {discount ? `−${formatAmount(discountValue)}` : "—"}
                  </span>
                </div>
                <div className="flex items-baseline justify-between border-t border-border pt-2">
                  <span className="text-[12.5px] text-muted-foreground">
                    {t("billing.hosting.rent.total")}
                  </span>
                  <span className="font-mono text-2xl font-bold tracking-[-0.02em]">
                    {formatAmount(total)}
                  </span>
                </div>
              </div>

              <button
                type="submit"
                disabled={rentMutation.isPending || !domain.trim()}
                className={cn(btnPrimary, "h-10 text-[13.5px]")}
              >
                <ShoppingCart className="h-[15px] w-[15px]" />
                {rentMutation.isPending
                  ? t("billing.hosting.rent.processing")
                  : t("billing.hosting.rent.title")}
              </button>
              <div className="text-[11.5px] leading-relaxed text-muted-foreground/70">
                {t("billing.hosting.rent.footnote")}
              </div>
            </div>
          </div>
        )}
      </form>
    </PageShell>
  );
}
