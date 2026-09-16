"use client";

import { useMemo } from "react";
import { m } from "motion/react";
import { TPS_SERIES, landingSpecs } from "@/components/landing/landing-content";
import { CountUp, EASE_OUT, Reveal, SectionLabel, WordsReveal } from "@/components/landing/motion";
import { useT } from "@/hooks/use-translations";

const CHART_W = 640;
const CHART_H = 200;
const MIN_TPS = 19.1;
const MAX_TPS = 20.05;

function chartPath(values: readonly number[]) {
  const step = CHART_W / (values.length - 1);
  const points = values.map((value, i) => ({
    x: i * step,
    y: CHART_H - ((value - MIN_TPS) / (MAX_TPS - MIN_TPS)) * CHART_H,
  }));
  let line = `M ${points[0].x} ${points[0].y}`;
  for (let i = 1; i < points.length; i++) {
    const prev = points[i - 1];
    const cur = points[i];
    const midX = (prev.x + cur.x) / 2;
    line += ` C ${midX} ${prev.y}, ${midX} ${cur.y}, ${cur.x} ${cur.y}`;
  }
  const area = `${line} L ${CHART_W} ${CHART_H} L 0 ${CHART_H} Z`;
  return { line, area, points };
}

export function HardwareSection({ index }: { index: string }) {
  const t = useT();
  const specs = landingSpecs(t);
  const { line, area, points } = useMemo(() => chartPath(TPS_SERIES), []);
  const dip = points.reduce((low, point) => (point.y > low.y ? point : low), points[0]);

  return (
    <section id="hardware" className="scroll-mt-16 border-b border-border">
      <div className="mx-auto max-w-[1240px] px-5 py-20 sm:px-8 lg:py-28">
        <div className="grid gap-8 lg:grid-cols-[1fr_minmax(0,460px)] lg:items-end">
          <div>
            <SectionLabel index={index}>{t("landing.hardware.label")}</SectionLabel>
            <h2 className="vx-display mt-5 text-[2rem] leading-[1.08] font-semibold tracking-[-0.03em] text-balance sm:text-[2.6rem]">
              <WordsReveal text={t("landing.hardware.title")} />
            </h2>
          </div>
          <Reveal>
            <p className="text-[16px] leading-[1.6] text-muted-foreground">{t("landing.hardware.text")}</p>
          </Reveal>
        </div>

        <div className="mt-14 grid gap-12 lg:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)] lg:gap-16">
          <dl className="flex flex-col">
            {specs.map((spec, i) => (
              <div key={spec.key} className="relative py-6">
                <m.span
                  className="absolute inset-x-0 top-0 h-px origin-left bg-border"
                  initial={{ scaleX: 0 }}
                  whileInView={{ scaleX: 1 }}
                  viewport={{ once: true, margin: "0px 0px -10% 0px" }}
                  transition={{ duration: 1, ease: EASE_OUT, delay: i * 0.08 }}
                />
                <Reveal delay={0.1 + i * 0.08} y={16} className="grid gap-2 sm:grid-cols-[140px_1fr] sm:gap-6">
                  <dt className="font-mono text-[12px] text-muted-foreground sm:pt-2">{spec.label}</dt>
                  <dd>
                    <div className="vx-display text-[1.45rem] leading-tight font-semibold tracking-[-0.02em]">
                      {spec.value}
                    </div>
                    <p className="mt-1.5 text-[14.5px] leading-[1.55] text-muted-foreground">{spec.note}</p>
                  </dd>
                </Reveal>
              </div>
            ))}
          </dl>

          <Reveal>
            <div className="rounded-2xl border border-border bg-card p-6 sm:p-8">
              <div className="font-mono text-[12px] text-muted-foreground">{t("landing.hardware.tps_label")}</div>
              <div className="mt-3 flex items-baseline gap-2">
                <span className="vx-display text-[3rem] leading-none font-semibold tracking-[-0.03em]">
                  <CountUp value={19.9} decimals={1} />
                </span>
                <span className="text-[15px] text-muted-foreground">{t("landing.hardware.tps_of")}</span>
              </div>
              <p className="mt-3 max-w-[460px] text-[14px] leading-[1.55] text-muted-foreground">
                {t("landing.hardware.tps_note")}
              </p>

              <div className="relative mt-8">
                <svg
                  viewBox={`0 0 ${CHART_W} ${CHART_H}`}
                  className="h-auto w-full overflow-visible"
                  role="img"
                  aria-label={t("landing.hardware.tps_label")}
                >
                  {[0.25, 0.5, 0.75].map((f) => (
                    <line
                      key={f}
                      x1={0}
                      x2={CHART_W}
                      y1={CHART_H * f}
                      y2={CHART_H * f}
                      className="stroke-border"
                      strokeDasharray="3 6"
                    />
                  ))}
                  <m.path
                    d={area}
                    className="fill-foreground/[0.06]"
                    initial={{ opacity: 0 }}
                    whileInView={{ opacity: 1 }}
                    viewport={{ once: true }}
                    transition={{ duration: 1.2, delay: 1.1 }}
                  />
                  <m.path
                    d={line}
                    fill="none"
                    className="stroke-foreground"
                    strokeWidth={2}
                    strokeLinecap="round"
                    initial={{ pathLength: 0 }}
                    whileInView={{ pathLength: 1 }}
                    viewport={{ once: true, margin: "0px 0px -10% 0px" }}
                    transition={{ duration: 1.8, ease: EASE_OUT }}
                  />
                  <m.g
                    initial={{ opacity: 0, y: 6 }}
                    whileInView={{ opacity: 1, y: 0 }}
                    viewport={{ once: true }}
                    transition={{ duration: 0.5, delay: 1.7 }}
                  >
                    <circle cx={dip.x} cy={dip.y} r={5} className="fill-amber-500" />
                    <circle cx={dip.x} cy={dip.y} r={11} className="fill-amber-500/20" />
                  </m.g>
                </svg>
                <m.div
                  initial={{ opacity: 0 }}
                  whileInView={{ opacity: 1 }}
                  viewport={{ once: true }}
                  transition={{ duration: 0.5, delay: 1.9 }}
                  className="pointer-events-none absolute rounded-md border border-border bg-background px-2 py-1 font-mono text-[11px] whitespace-nowrap text-muted-foreground"
                  style={{
                    left: `${(dip.x / CHART_W) * 100}%`,
                    top: `${(dip.y / CHART_H) * 100}%`,
                    transform: "translate(calc(-100% - 16px), -50%)",
                  }}
                >
                  {t("landing.hardware.tps_peak")}
                </m.div>
                <div className="mt-3 flex justify-between font-mono text-[11px] text-muted-foreground">
                  <span>00:00</span>
                  <span>06:00</span>
                  <span>12:00</span>
                  <span>18:00</span>
                  <span>23:00</span>
                </div>
              </div>
            </div>
          </Reveal>
        </div>
      </div>
    </section>
  );
}
