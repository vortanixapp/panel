"use client";

import { useEffect, useRef, useState } from "react";
import { AnimatePresence, m, useInView, useReducedMotionConfig } from "motion/react";
import { landingLocations } from "@/components/landing/landing-content";
import { EASE_OUT, Reveal, SectionLabel, WordsReveal } from "@/components/landing/motion";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

const COUNTRY: Record<string, string> = {
  moscow: "RU",
  spb: "RU",
  frankfurt: "DE",
  amsterdam: "NL",
  warsaw: "PL",
  new_york: "US",
  singapore: "SG",
};

function pingTone(ms: number) {
  if (ms < 40) return "bg-emerald-500";
  if (ms < 110) return "bg-amber-500";
  return "bg-rose-500";
}

export function LocationsSection({ index }: { index: string }) {
  const t = useT();
  const locations = landingLocations(t);
  const listRef = useRef<HTMLDivElement>(null);
  const inView = useInView(listRef, { amount: 0.3 });
  const reduced = useReducedMotionConfig();
  const baseline = locations.map((location) => location.ms).join(",");
  const [jitter, setJitter] = useState<number[]>(() => locations.map(() => 0));

  useEffect(() => {
    if (!inView || reduced) return;
    const limits = baseline.split(",").map((ms) => Math.max(1, Math.round(Number(ms) * 0.12)));
    const timer = setInterval(() => {
      setJitter((current) =>
        current.map((value, i) => {
          if (Math.random() < 0.55) return value;
          const next = value + Math.round(Math.random() * 4 - 2);
          return Math.max(-limits[i], Math.min(limits[i], next));
        })
      );
    }, 1700);
    return () => clearInterval(timer);
  }, [inView, reduced, baseline]);

  return (
    <section className="border-b border-border">
      <div className="mx-auto grid max-w-[1240px] gap-12 px-5 py-20 sm:px-8 lg:grid-cols-[minmax(0,0.75fr)_minmax(0,1.25fr)] lg:gap-16 lg:py-28">
        <div className="lg:pt-2">
          <SectionLabel index={index}>{t("landing.locations.label")}</SectionLabel>
          <h2 className="vx-display mt-5 text-[2rem] leading-[1.08] font-semibold tracking-[-0.03em] text-balance sm:text-[2.6rem]">
            <WordsReveal text={t("landing.locations.title")} />
          </h2>
          <Reveal>
            <p className="mt-6 text-[16px] leading-[1.6] text-muted-foreground">{t("landing.locations.text")}</p>
            <p className="mt-8 flex items-center gap-2.5 font-mono text-[12px] text-muted-foreground">
              <span className="relative flex size-2">
                <span className="absolute inset-0 animate-ping rounded-full bg-emerald-500/50 [animation-duration:2.2s]" />
                <span className="relative size-2 rounded-full bg-emerald-500" />
              </span>
              {t("landing.locations.note")}
            </p>
          </Reveal>
        </div>

        <div ref={listRef}>
          <div className="hidden grid-cols-[minmax(0,1fr)_96px_110px_minmax(0,180px)] gap-6 border-b border-border pb-3 font-mono text-[11.5px] text-muted-foreground sm:grid">
            <span>{t("landing.locations.col_city")}</span>
            <span>{t("landing.locations.col_ping")}</span>
            <span>{t("landing.locations.col_uplink")}</span>
            <span>{t("landing.locations.col_load")}</span>
          </div>
          <ul>
            {locations.map((location, i) => {
              const ms = Math.max(1, location.ms + jitter[i]);
              return (
                <m.li
                  key={location.key}
                  initial={{ opacity: 0, y: 14 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  viewport={{ once: true, margin: "0px 0px -8% 0px" }}
                  transition={{ duration: 0.6, ease: EASE_OUT, delay: i * 0.06 }}
                  data-reveal
                  className="group grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-6 gap-y-3 border-b border-border py-4 sm:grid-cols-[minmax(0,1fr)_96px_110px_minmax(0,180px)] sm:py-5"
                >
                  <div className="flex min-w-0 items-baseline gap-3">
                    <span className="truncate text-[17px] font-medium tracking-[-0.01em] transition-transform duration-300 group-hover:translate-x-1">
                      {location.city}
                    </span>
                    <span className="font-mono text-[11px] text-muted-foreground">{COUNTRY[location.key]}</span>
                  </div>

                  <div className="flex items-center gap-2 justify-self-end font-mono text-[14px] tabular-nums sm:justify-self-start">
                    <span className={cn("size-1.5 rounded-full", pingTone(location.ms))} />
                    <span className="relative inline-flex h-[1.4em] min-w-[2.2ch] overflow-hidden">
                      <AnimatePresence mode="popLayout" initial={false}>
                        <m.span
                          key={ms}
                          initial={{ y: "100%", opacity: 0 }}
                          animate={{ y: "0%", opacity: 1 }}
                          exit={{ y: "-100%", opacity: 0 }}
                          transition={{ duration: 0.35, ease: EASE_OUT }}
                          className="block leading-[1.4em]"
                        >
                          {ms}
                        </m.span>
                      </AnimatePresence>
                    </span>
                    <span className="text-[12px] text-muted-foreground">{t("landing.locations.ms")}</span>
                  </div>

                  <span className="hidden font-mono text-[13px] text-muted-foreground sm:block">
                    {t("landing.locations.uplink", { gbit: location.gbit })}
                  </span>

                  <div className="col-span-2 flex items-center gap-3 sm:col-span-1">
                    <div className="relative h-1 flex-1 overflow-hidden rounded-full bg-muted">
                      <m.div
                        className={cn(
                          "absolute inset-y-0 left-0 w-full origin-left rounded-full",
                          location.busy ? "bg-amber-500" : "bg-foreground/80"
                        )}
                        initial={{ scaleX: 0 }}
                        whileInView={{ scaleX: location.load / 100 }}
                        viewport={{ once: true }}
                        transition={{ duration: 1.2, ease: EASE_OUT, delay: 0.2 + i * 0.06 }}
                      />
                    </div>
                    <span
                      className={cn(
                        "w-[92px] shrink-0 text-right font-mono text-[12px] sm:w-auto",
                        location.busy ? "text-amber-600 dark:text-amber-400" : "text-muted-foreground"
                      )}
                    >
                      {location.busy ? t("landing.locations.busy") : `${location.load}%`}
                    </span>
                  </div>
                </m.li>
              );
            })}
          </ul>
        </div>
      </div>
    </section>
  );
}
