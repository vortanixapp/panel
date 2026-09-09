"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import {
  ListEmpty,
  MON,
  MON_CARD,
  MON_INNER,
  countdown,
  formatDate,
  formatMoney,
  plural,
} from "@/components/user/account/shared";
import {
  fetchDailyBonus,
  spinDailyBonus,
  type BonusPrize,
  type DailyBonusSpinResponse,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const WEEKDAY_KEYS = [
  "billing.bonus.weekday.sun",
  "billing.bonus.weekday.mon",
  "billing.bonus.weekday.tue",
  "billing.bonus.weekday.wed",
  "billing.bonus.weekday.thu",
  "billing.bonus.weekday.fri",
  "billing.bonus.weekday.sat",
];
const SPIN_DURATION_MS = 2600;

export function DailyBonusPageContent() {
  const t = useT();
  const queryClient = useQueryClient();
  const [now, setNow] = useState(() => Date.now());
  const [rotation, setRotation] = useState(18);
  const [spinning, setSpinning] = useState(false);
  const [result, setResult] = useState<DailyBonusSpinResponse | null>(null);
  const settleTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const bonus = useQuery({ queryKey: ["daily-bonus"], queryFn: fetchDailyBonus });

  const nextSpinAt = bonus.data?.next_spin_at ?? null;
  useEffect(() => {
    if (!nextSpinAt) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [nextSpinAt]);

  useEffect(() => () => {
    if (settleTimer.current) clearTimeout(settleTimer.current);
  }, []);

  const prizes = bonus.data?.prizes ?? [];

  const spin = useMutation({
    mutationFn: spinDailyBonus,
    onMutate: () => {
      setSpinning(true);
      setResult(null);
    },
    onSuccess: (res) => {
      setRotation((prev) => prev + 360 * 5 + sectorAngle(prizes, res.prize_id, prev));
      settleTimer.current = setTimeout(() => {
        setSpinning(false);
        setResult(res);
        toast.success(
          res.credited > 0
            ? t("billing.bonus.win_credited", { amount: formatMoney(res.credited) })
            : res.promo_code
              ? t("billing.bonus.your_promo", { code: res.promo_code })
              : t("billing.bonus.you_won", {
                  prize: res.prize?.label ?? t("billing.bonus.prize_fallback"),
                })
        );
        void queryClient.invalidateQueries({ queryKey: ["daily-bonus"] });
        void queryClient.invalidateQueries({ queryKey: ["billing"] });
        void queryClient.invalidateQueries({ queryKey: ["notifications"] });
        void queryClient.invalidateQueries({ queryKey: ["notifications-unread"] });
      }, SPIN_DURATION_MS);
    },
    onError: (err) => {
      setSpinning(false);
      toast.error(
        err instanceof Error ? err.message : t("billing.bonus.spin_failed")
      );
      void queryClient.invalidateQueries({ queryKey: ["daily-bonus"] });
    },
  });

  const wheelGradient = useMemo(() => buildWheelGradient(prizes), [prizes]);
  const timeLeft = countdown(nextSpinAt, now);
  const canSpin = Boolean(bonus.data?.can_spin) && !spinning && !spin.isPending;

  if (bonus.isLoading) {
    return (
      <PageShell variant="user">
        <div className="grid w-full gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,0.85fr)]">
          <Skeleton className="h-[520px] rounded-[16px]" />
          <Skeleton className="h-[520px] rounded-[14px]" />
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-[18px]">
        <div>
          <h1 className="text-[26px] font-bold tracking-[-0.02em]">
            {t("billing.bonus.title")}
          </h1>
          <p className="mt-1 text-[13px] text-[var(--vx-muted)]">
            {t("billing.bonus.subtitle")}
          </p>
        </div>

        {bonus.data?.prizes_disabled ? (
          <div className={cn(MON_CARD)}>
            <ListEmpty
              icon="ri-gift-line"
              title={t("billing.bonus.no_prizes_title")}
              text={t("billing.bonus.no_prizes_text")}
            />
          </div>
        ) : (
          <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,0.85fr)]">
            <div
              className={cn(
                "flex flex-col items-center gap-[22px] rounded-[16px] p-7",
                MON_CARD
              )}
            >
              <div className="relative h-[260px] w-[260px]">
                <div
                  className="absolute inset-0 rounded-full"
                  style={{
                    background: wheelGradient,
                    transform: `rotate(${rotation}deg)`,
                    transition: spinning
                      ? `transform ${SPIN_DURATION_MS}ms cubic-bezier(.15,.6,.15,1)`
                      : undefined,
                  }}
                />
                <div className="pointer-events-none absolute inset-0 rounded-full shadow-[inset_0_0_0_1px_var(--vx-veil-strong)]" />
                <div className="absolute inset-[78px] flex flex-col items-center justify-center gap-0.5 rounded-full border border-[var(--vx-border)] bg-[var(--vx-card)]">
                  <i className="ri-gift-line text-[22px] text-[var(--vx-warn)]" />
                  <span className="font-mono text-[12px] text-[var(--vx-muted)]">
                    {spinning
                      ? "…"
                      : bonus.data?.can_spin
                        ? t("billing.bonus.hours_24")
                        : timeLeft || t("billing.bonus.hours_zero")}
                  </span>
                </div>
                <div
                  className="absolute -top-1 left-1/2 -ml-1.5 h-4 w-3 bg-[var(--vx-fg-strong)]"
                  style={{ clipPath: "polygon(50% 100%, 0 0, 100% 0)" }}
                />
              </div>

              <div className="text-center">
                <div className="text-[14px] font-semibold">
                  {spinning
                    ? t("billing.bonus.spinning")
                    : bonus.data?.can_spin
                      ? t("billing.bonus.not_spun_today")
                      : t("billing.bonus.claimed")}
                </div>
                <p className="mt-1.5 text-[12px] text-[var(--vx-muted)]">
                  {spinning
                    ? t("billing.bonus.determining")
                    : bonus.data?.can_spin
                      ? t("billing.bonus.one_per_day")
                      : t("billing.bonus.next_spin_in", { time: timeLeft || "—" })}
                </p>
              </div>

              <button
                type="button"
                onClick={() => spin.mutate()}
                disabled={!canSpin}
                className={cn(
                  "min-w-[200px] rounded-[10px] px-[22px] py-[13px] text-[14px] font-semibold transition-colors",
                  canSpin
                    ? "cursor-pointer bg-[var(--vx-fg-strong)] text-[var(--vx-on-fill)] hover:bg-white"
                    : "cursor-default bg-[var(--vx-tint)] text-[var(--vx-muted)]"
                )}
              >
                {spinning
                  ? t("billing.bonus.spinning_button")
                  : bonus.data?.can_spin
                    ? t("billing.bonus.spin_button")
                    : t("billing.bonus.unavailable", { time: timeLeft || "—" })}
              </button>

              {result && (
                <div className={cn("flex w-full items-center gap-3 px-4 py-3.5", MON_INNER)}>
                  <span className="flex h-[34px] w-[34px] items-center justify-center rounded-[10px] bg-[rgba(232,160,60,0.12)] text-[var(--vx-warn)]">
                    <i className="ri-sparkling-line text-[16px]" />
                  </span>
                  <div className="flex-1">
                    <div className="text-[13px] font-semibold">{result.prize?.label}</div>
                    <div className="mt-0.5 text-[11px] text-[var(--vx-muted)]">
                      {result.credited > 0
                        ? t("billing.bonus.credited", {
                            amount: formatMoney(result.credited),
                          })
                        : result.promo_code
                          ? t("billing.bonus.promo_once")
                          : t("billing.bonus.prize_activated")}
                    </div>
                    {result.promo_code && (
                      <div className="mt-1 font-mono text-[13px] font-semibold tracking-wide">
                        {result.promo_code}
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>

            <div className="flex flex-col gap-4">
              <div className={cn("p-[18px]", MON_CARD)}>
                <div className="flex items-baseline gap-2.5">
                  <span className="text-[14px] font-semibold">
                    {t("billing.bonus.streak")}
                  </span>
                  <span className="ml-auto font-mono text-[13px] text-[var(--vx-warn)]">
                    {bonus.data?.streak ?? 0}{" "}
                    {plural(
                      bonus.data?.streak ?? 0,
                      t("billing.bonus.day_one"),
                      t("billing.bonus.day_few"),
                      t("billing.bonus.day_many")
                    )}{" "}
                    {t("billing.bonus.streak_suffix")}
                  </span>
                </div>
                <div className="mt-3.5 flex gap-1.5">
                  {(bonus.data?.streak_days ?? []).map((d) => (
                    <div key={d.day} className="flex flex-1 flex-col items-center gap-1.5">
                      <div
                        className={cn(
                          "flex aspect-square w-full items-center justify-center rounded-[10px] border",
                          d.claimed
                            ? "border-[var(--vx-border)] bg-[rgba(232,160,60,0.14)] text-[var(--vx-warn)]"
                            : d.today
                              ? "border-[var(--vx-border-strong)] bg-[var(--vx-tint)] text-[var(--vx-fg)]"
                              : "border-[var(--vx-border)] bg-[var(--vx-card-2)] text-[var(--vx-ghost)]"
                        )}
                      >
                        <i
                          className={cn(
                            "text-[13px]",
                            d.claimed
                              ? "ri-check-line"
                              : d.today
                                ? "ri-gift-line"
                                : "ri-lock-line"
                          )}
                        />
                      </div>
                      <span className="text-[10px] text-[var(--vx-muted)]">
                        {t(WEEKDAY_KEYS[d.weekday])}
                      </span>
                    </div>
                  ))}
                </div>
                <p className="mt-3.5 text-[12px] text-pretty text-[var(--vx-muted)]">
                  {t("billing.bonus.streak_hint")}
                </p>
              </div>

              <div className={cn("overflow-hidden", MON_CARD)}>
                <div className="border-b border-[var(--vx-border)] px-4 py-3.5 text-[14px] font-semibold">
                  {t("billing.bonus.prizes_title")}
                </div>
                {prizes.map((p) => (
                  <div
                    key={p.id}
                    className="flex items-center gap-3 border-b border-[var(--vx-divider)] px-4 py-2.5 last:border-b-0"
                  >
                    <span
                      className="h-2 w-2 shrink-0 rounded-[2px]"
                      style={{ background: p.color }}
                    />
                    <span className="flex-1 text-[13px]">{p.label}</span>
                    <span className="font-mono text-[12px] text-[var(--vx-muted)]">{p.chance}%</span>
                  </div>
                ))}
              </div>

              <div className={cn("overflow-hidden", MON_CARD)}>
                <div className="border-b border-[var(--vx-border)] px-4 py-3.5 text-[14px] font-semibold">
                  {t("billing.bonus.history_title")}
                </div>
                {(bonus.data?.history ?? []).length === 0 ? (
                  <p className="px-4 py-8 text-center text-[12px] text-[var(--vx-muted)]">
                    {t("billing.bonus.history_empty")}
                  </p>
                ) : (
                  (bonus.data?.history ?? []).map((h, i) => (
                    <div
                      key={`${h.created_at}-${i}`}
                      className="flex items-center gap-3 border-b border-[var(--vx-divider)] px-4 py-2.5 last:border-b-0"
                    >
                      <span className="flex-1 text-[13px]">{h.prize}</span>
                      <span className="font-mono text-[11px] text-[var(--vx-muted)]">
                        {formatDate(h.created_at)}
                      </span>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </PageShell>
  );
}

function buildWheelGradient(prizes: BonusPrize[]): string {
  if (prizes.length === 0) return `conic-gradient(${MON.faint} 0deg 360deg)`;
  const step = 360 / prizes.length;
  const stops = prizes.map(
    (p, i) => `${p.color || MON.faint} ${i * step}deg ${(i + 1) * step}deg`
  );
  return `conic-gradient(${stops.join(", ")})`;
}

function sectorAngle(prizes: BonusPrize[], prizeId: string, currentRotation: number): number {
  const index = prizes.findIndex((p) => p.id === prizeId);
  if (index < 0 || prizes.length === 0) return 0;
  const step = 360 / prizes.length;
  const target = (360 - (index * step + step / 2)) % 360;
  const current = ((currentRotation % 360) + 360) % 360;
  return (target - current + 360) % 360;
}
