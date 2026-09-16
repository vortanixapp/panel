"use client";

import Link from "next/link";
import { useState } from "react";
import { AnimatePresence, m } from "motion/react";
import { ArrowRight, Check } from "lucide-react";
import { landingPricing } from "@/components/landing/landing-content";
import { EASE_OUT, Reveal, SectionLabel, WordsReveal } from "@/components/landing/motion";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";

type Period = "monthly" | "hourly";

export function PricingSection({ index }: { index: string }) {
  const t = useT();
  const plans = landingPricing(t);
  const [period, setPeriod] = useState<Period>("monthly");

  const formatPrice = (plan: (typeof plans)[number]) =>
    period === "monthly"
      ? plan.monthly.toLocaleString(localeTag())
      : plan.hourly.toLocaleString(localeTag(), { minimumFractionDigits: 2, maximumFractionDigits: 2 });

  return (
    <section id="pricing" className="scroll-mt-16 border-b border-border">
      <div className="mx-auto max-w-[1240px] px-5 py-20 sm:px-8 lg:py-28">
        <div className="grid gap-8 lg:grid-cols-[1fr_minmax(0,440px)] lg:items-end">
          <div>
            <SectionLabel index={index}>{t("landing.pricing.label")}</SectionLabel>
            <h2 className="vx-display mt-5 text-[2rem] leading-[1.08] font-semibold tracking-[-0.03em] text-balance sm:text-[2.6rem]">
              <WordsReveal text={t("landing.pricing.title")} />
            </h2>
          </div>
          <Reveal className="flex flex-col gap-6">
            <p className="text-[16px] leading-[1.6] text-muted-foreground">{t("landing.pricing.text")}</p>
            <div
              role="radiogroup"
              aria-label={t("landing.pricing.label")}
              className="inline-flex w-fit rounded-full border border-border p-1"
            >
              {(["monthly", "hourly"] as const).map((value) => {
                const active = period === value;
                return (
                  <button
                    key={value}
                    type="button"
                    role="radio"
                    aria-checked={active}
                    onClick={() => setPeriod(value)}
                    className={cn(
                      "relative rounded-full px-4 py-1.5 text-[13.5px] font-medium transition-colors",
                      active ? "text-background" : "text-muted-foreground hover:text-foreground"
                    )}
                  >
                    {active && (
                      <m.span
                        layoutId="pricing-period"
                        className="absolute inset-0 rounded-full bg-foreground"
                        transition={{ type: "spring", stiffness: 420, damping: 34 }}
                      />
                    )}
                    <span className="relative">{t(`landing.pricing.${value}`)}</span>
                  </button>
                );
              })}
            </div>
          </Reveal>
        </div>

        <div className="mt-14 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {plans.map((plan, i) => (
            <Reveal key={plan.id} delay={i * 0.07} className="min-w-0">
              <div
                className={cn(
                  "group relative flex h-full flex-col rounded-2xl border p-6 transition-transform duration-500 ease-[cubic-bezier(0.22,1,0.36,1)] hover:-translate-y-1.5",
                  plan.highlighted
                    ? "border-foreground bg-foreground text-background"
                    : "border-border bg-card"
                )}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="vx-display text-[1.2rem] font-semibold tracking-[-0.02em]">{plan.name}</div>
                  {plan.highlighted && (
                    <span className="rounded-full bg-background/15 px-2.5 py-1 text-[11.5px] font-medium whitespace-nowrap">
                      {t("landing.pricing.popular")}
                    </span>
                  )}
                </div>
                <p
                  className={cn(
                    "mt-2 min-h-[2.9em] text-[14px] leading-[1.45]",
                    plan.highlighted ? "text-background/70" : "text-muted-foreground"
                  )}
                >
                  {plan.audience}
                </p>

                <div className="mt-6 flex items-baseline gap-1.5">
                  <span className="relative inline-flex overflow-hidden">
                    <AnimatePresence mode="popLayout" initial={false}>
                      <m.span
                        key={`${plan.id}-${period}`}
                        initial={{ y: "70%", opacity: 0, filter: "blur(4px)" }}
                        animate={{ y: "0%", opacity: 1, filter: "blur(0px)" }}
                        exit={{ y: "-70%", opacity: 0, filter: "blur(4px)" }}
                        transition={{ duration: 0.45, ease: EASE_OUT }}
                        className="vx-display block text-[2.6rem] leading-[1.1] font-semibold tracking-[-0.04em] tabular-nums"
                      >
                        {formatPrice(plan)}
                      </m.span>
                    </AnimatePresence>
                  </span>
                  <span className={cn("text-[14px]", plan.highlighted ? "text-background/70" : "text-muted-foreground")}>
                    {t(period === "monthly" ? "landing.pricing.per_month" : "landing.pricing.per_hour")}
                  </span>
                </div>

                <ul className={cn("mt-6 flex flex-col gap-2.5 border-t pt-6", plan.highlighted ? "border-background/15" : "border-border")}>
                  {plan.specs.map((spec) => (
                    <li key={spec} className="flex gap-2.5 text-[14px] leading-[1.45]">
                      <Check
                        className={cn(
                          "mt-[3px] size-3.5 shrink-0",
                          plan.highlighted ? "text-background" : "text-emerald-500"
                        )}
                      />
                      <span className={plan.highlighted ? "text-background/85" : undefined}>{spec}</span>
                    </li>
                  ))}
                </ul>

                <Link
                  href="/register"
                  className={cn(
                    "mt-8 inline-flex items-center justify-between rounded-full py-2.5 pr-2.5 pl-5 text-[14px] font-semibold transition-colors",
                    plan.highlighted
                      ? "bg-background text-foreground"
                      : "border border-border hover:border-foreground hover:bg-foreground hover:text-background"
                  )}
                >
                  {t("landing.pricing.choose")}
                  <span
                    className={cn(
                      "flex size-7 items-center justify-center rounded-full transition-transform duration-300 group-hover:translate-x-0.5",
                      plan.highlighted ? "bg-foreground text-background" : "bg-muted text-foreground"
                    )}
                  >
                    <ArrowRight className="size-3.5" />
                  </span>
                </Link>
              </div>
            </Reveal>
          ))}
        </div>

        <Reveal>
          <p className="mt-8 max-w-[720px] text-[14px] leading-[1.6] text-muted-foreground">{t("landing.pricing.note")}</p>
        </Reveal>
      </div>
    </section>
  );
}
