"use client";

import { useEffect, useRef, useState } from "react";
import { AnimatePresence, animate, m, useInView, useReducedMotionConfig, useScroll } from "motion/react";
import { Check } from "lucide-react";
import { landingSteps } from "@/components/landing/landing-content";
import { EASE_OUT, Reveal, SectionLabel, WordsReveal } from "@/components/landing/motion";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export function StepsSection({ index }: { index: string }) {
  const t = useT();
  const steps = landingSteps(t);
  const [active, setActive] = useState(0);
  const railRef = useRef<HTMLDivElement>(null);
  const { scrollYProgress } = useScroll({ target: railRef, offset: ["start center", "end center"] });

  return (
    <section className="border-b border-border">
      <div className="mx-auto max-w-[1240px] px-5 py-20 sm:px-8 lg:py-28">
        <div className="lg:hidden">
          <SectionLabel index={index}>{t("landing.steps.label")}</SectionLabel>
          <h2 className="vx-display mt-5 text-[2rem] leading-[1.08] font-semibold tracking-[-0.03em] text-balance sm:text-[2.6rem]">
            <WordsReveal text={t("landing.steps.title")} />
          </h2>
        </div>

        <div ref={railRef} className="grid gap-16 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)] lg:gap-20">
          <div className="hidden lg:block">
            <div className="sticky top-28">
              <SectionLabel index={index}>{t("landing.steps.label")}</SectionLabel>
              <h2 className="vx-display mt-5 text-[2.6rem] leading-[1.08] font-semibold tracking-[-0.03em] text-balance">
                <WordsReveal text={t("landing.steps.title")} />
              </h2>

              <div className="relative mt-12 pl-8">
                <div className="absolute top-1 bottom-1 left-[5px] w-px bg-border" />
                <m.div
                  className="absolute top-1 bottom-1 left-[5px] w-px origin-top bg-foreground"
                  style={{ scaleY: scrollYProgress }}
                />
                <ol className="flex flex-col gap-8">
                  {steps.map((step, i) => {
                    const current = i === active;
                    return (
                      <li key={step.n} className="relative">
                        <span
                          className={cn(
                            "absolute top-[7px] -left-8 size-[11px] rounded-full border-2 transition-colors duration-500",
                            i <= active ? "border-foreground bg-foreground" : "border-border bg-background"
                          )}
                        />
                        <div className="flex items-baseline gap-3">
                          <span className="font-mono text-[12px] text-muted-foreground">{step.n}</span>
                          <span
                            className={cn(
                              "text-[19px] font-semibold tracking-[-0.01em] transition-colors duration-500",
                              current ? "text-foreground" : "text-muted-foreground"
                            )}
                          >
                            {step.title}
                          </span>
                        </div>
                        <AnimatePresence initial={false}>
                          {current && (
                            <m.div
                              initial={{ height: 0, opacity: 0 }}
                              animate={{ height: "auto", opacity: 1 }}
                              exit={{ height: 0, opacity: 0 }}
                              transition={{ duration: 0.45, ease: EASE_OUT }}
                              className="overflow-hidden"
                            >
                              <p className="max-w-[420px] pt-2.5 text-[15px] leading-[1.6] text-muted-foreground">
                                {step.body}
                              </p>
                              <p className="pt-2 font-mono text-[12px] text-foreground">{step.meta}</p>
                            </m.div>
                          )}
                        </AnimatePresence>
                      </li>
                    );
                  })}
                </ol>
              </div>
            </div>
          </div>

          <div className="flex flex-col gap-6 lg:gap-[18vh] lg:py-[8vh]">
            {steps.map((step, i) => (
              <StepPanel key={step.n} index={i} onActive={setActive}>
                <div className="mb-5 lg:hidden">
                  <div className="flex items-baseline gap-3">
                    <span className="font-mono text-[12px] text-muted-foreground">{step.n}</span>
                    <span className="text-[19px] font-semibold">{step.title}</span>
                  </div>
                  <p className="mt-2 text-[15px] leading-[1.6] text-muted-foreground">{step.body}</p>
                </div>
                {i === 0 && <ConfigureScene />}
                {i === 1 && <InstallScene />}
                {i === 2 && <OnlineScene />}
              </StepPanel>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

function StepPanel({
  index,
  onActive,
  children,
}: {
  index: number;
  onActive: (index: number) => void;
  children: React.ReactNode;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, { amount: 0.6 });

  useEffect(() => {
    if (inView) onActive(index);
  }, [inView, index, onActive]);

  return (
    <Reveal>
      <div ref={ref}>{children}</div>
    </Reveal>
  );
}

function SceneFrame({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <div className={cn("overflow-hidden rounded-2xl border border-border bg-card p-6 sm:p-8", className)}>
      {children}
    </div>
  );
}

function useSceneStart() {
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, { once: true, amount: 0.5 });
  return { ref, started: inView };
}

function ConfigureScene() {
  const t = useT();
  const reduced = useReducedMotionConfig();
  const { ref, started } = useSceneStart();
  const [ram, setRam] = useState(2);

  useEffect(() => {
    if (!started) return;
    if (reduced) {
      setRam(6);
      return;
    }
    const controls = animate(2, 6, {
      duration: 1.6,
      delay: 0.4,
      ease: EASE_OUT,
      onUpdate: (value) => setRam(Math.round(value)),
    });
    return () => controls.stop();
  }, [started, reduced]);

  const price = 99 + ram * 82;

  return (
    <div ref={ref}>
      <SceneFrame>
        <div className="text-[12.5px] text-muted-foreground">{t("landing.steps.scene_game")}</div>
        <div className="mt-2 flex flex-wrap gap-2">
          {["Minecraft", "Rust", "CS2"].map((name, i) => (
            <span
              key={name}
              className={cn(
                "rounded-full border px-3.5 py-1.5 text-[13px]",
                i === 0 ? "border-foreground bg-foreground text-background" : "border-border text-muted-foreground"
              )}
            >
              {name}
            </span>
          ))}
        </div>

        <div className="mt-7 flex items-baseline justify-between">
          <span className="text-[12.5px] text-muted-foreground">RAM</span>
          <span className="font-mono text-[14px] tabular-nums">{t("landing.unit.gb", { value: ram })}</span>
        </div>
        <div className="relative mt-3 h-1.5 rounded-full bg-muted">
          <m.div
            className="absolute inset-y-0 left-0 rounded-full bg-foreground"
            style={{ width: `${((ram - 1) / 11) * 100}%` }}
          />
          <m.div
            className="absolute top-1/2 size-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-foreground bg-background"
            style={{ left: `${((ram - 1) / 11) * 100}%` }}
          />
        </div>

        <div className="mt-7 grid grid-cols-2 gap-3 text-[13px]">
          <div className="rounded-xl border border-border px-4 py-3">
            <div className="text-muted-foreground">{t("landing.steps.scene_location")}</div>
            <div className="mt-1 font-medium">{t("landing.locations.frankfurt")}</div>
          </div>
          <div className="rounded-xl border border-border px-4 py-3">
            <div className="text-muted-foreground">{t("landing.steps.scene_total")}</div>
            <div className="mt-1 font-mono font-medium tabular-nums">
              {t("landing.unit.rub_month", { price: price.toLocaleString(localeTag()) })}
            </div>
          </div>
        </div>
      </SceneFrame>
    </div>
  );
}

function InstallScene() {
  const t = useT();
  const reduced = useReducedMotionConfig();
  const { ref, started } = useSceneStart();
  const [progress, setProgress] = useState(0);
  const stages = [
    t("landing.steps.stage_download"),
    t("landing.steps.stage_java"),
    t("landing.steps.stage_world"),
    t("landing.steps.stage_start"),
  ];

  useEffect(() => {
    if (!started) return;
    if (reduced) {
      setProgress(100);
      return;
    }
    const controls = animate(0, 100, {
      duration: 3.4,
      delay: 0.3,
      ease: [0.4, 0, 0.2, 1],
      onUpdate: (value) => setProgress(value),
    });
    return () => controls.stop();
  }, [started, reduced]);

  const stageIndex = Math.min(stages.length - 1, Math.floor(progress / (100 / stages.length)));

  return (
    <div ref={ref}>
      <SceneFrame>
        <div className="flex items-baseline justify-between gap-4">
          <span className="font-mono text-[13px]">survival</span>
          <span className="vx-display text-[2.2rem] leading-none font-semibold tabular-nums">
            {Math.round(progress)}%
          </span>
        </div>
        <div className="mt-5 h-1.5 overflow-hidden rounded-full bg-muted">
          <div className="h-full rounded-full bg-foreground" style={{ width: `${progress}%` }} />
        </div>
        <ul className="mt-6 flex flex-col gap-2.5">
          {stages.map((stage, i) => {
            const done = progress >= 100 || i < stageIndex;
            const current = i === stageIndex && progress < 100;
            return (
              <li
                key={stage}
                className={cn(
                  "flex items-center gap-3 text-[14px] transition-colors duration-300",
                  done || current ? "text-foreground" : "text-muted-foreground/60"
                )}
              >
                <span
                  className={cn(
                    "flex size-5 items-center justify-center rounded-full border",
                    done ? "border-emerald-500 bg-emerald-500 text-white" : "border-border"
                  )}
                >
                  {done ? (
                    <Check className="size-3" />
                  ) : current ? (
                    <span className="size-1.5 animate-pulse rounded-full bg-foreground" />
                  ) : null}
                </span>
                {stage}
              </li>
            );
          })}
        </ul>
      </SceneFrame>
    </div>
  );
}

function OnlineScene() {
  const t = useT();
  const { ref, started } = useSceneStart();
  const players = ["Nikita_QQ", "zhenya.exe", "Aurora", "kot_v_sapogah"];

  return (
    <div ref={ref}>
      <SceneFrame>
        <div className="flex items-center justify-between gap-4">
          <span className="inline-flex items-center gap-2 rounded-full border border-emerald-500/30 px-3 py-1 text-[12.5px] font-medium text-emerald-600 dark:text-emerald-400">
            <span className="size-1.5 rounded-full bg-emerald-500" />
            {t("landing.hero.window_online")}
          </span>
          <span className="font-mono text-[12px] text-muted-foreground">TPS 20.0</span>
        </div>
        <div className="vx-display mt-6 text-[1.55rem] leading-tight font-semibold tracking-[-0.02em] break-all sm:text-[1.9rem]">
          203.0.113.24:25565
        </div>
        <div className="mt-6 text-[12.5px] text-muted-foreground">{t("landing.steps.scene_players")}</div>
        <ul className="mt-3 flex flex-col">
          {players.map((name, i) => (
            <m.li
              key={name}
              initial={{ opacity: 0, x: -12 }}
              animate={started ? { opacity: 1, x: 0 } : undefined}
              transition={{ duration: 0.5, ease: EASE_OUT, delay: 0.4 + i * 0.45 }}
              className="flex items-center justify-between border-b border-border py-2.5 text-[14px] last:border-b-0"
            >
              <span className="flex items-center gap-3">
                <span className="flex size-7 items-center justify-center rounded-md bg-muted font-mono text-[11px] uppercase">
                  {name.slice(0, 2)}
                </span>
                {name}
              </span>
              <span className="font-mono text-[12px] text-muted-foreground">{12 + i * 7} ms</span>
            </m.li>
          ))}
        </ul>
      </SceneFrame>
    </div>
  );
}
