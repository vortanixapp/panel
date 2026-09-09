"use client";

import Image from "next/image";
import Link from "next/link";
import { useState } from "react";
import { cn } from "@/lib/utils";
import { DashboardPreview } from "@/components/landing/dashboard-preview";
import {
  LANDING_DEPLOY_LOG,
  LANDING_FEATURED_GAMES,
  LANDING_MORE_GAMES,
  LANDING_PANEL_PREVIEW_LOG,
  LANDING_TPS_BARS,
  landingBento,
  landingFaq,
  landingGamesTitle,
  landingHeroBadge,
  landingHeroGauges,
  landingHeroStats,
  landingLocations,
  landingPanelMetrics,
  landingPanelPoints,
  landingPricing,
  landingSteps,
} from "@/components/landing/landing-content";
import { useLandingReveal } from "@/components/landing/use-landing-reveal";
import { useBrand } from "@/context/brand-provider";
import { useT } from "@/hooks/use-translations";

function LandingBlock({
  name,
  blocks,
  children,
}: {
  name: string;
  blocks: Record<string, boolean>;
  children: React.ReactNode;
}) {
  if (blocks[name] === false) return null;
  return <>{children}</>;
}

const LANDING_COLOR_TOKENS: Record<string, string> = {
  primary: "--primary",
  secondary: "--secondary",
  accent: "--accent",
  background: "--background",
  foreground: "--foreground",
  card: "--card",
  muted: "--muted",
  border: "--border",
};

function landingColorStyle(colors: Record<string, string>): React.CSSProperties {
  const style: Record<string, string> = {};
  for (const [key, token] of Object.entries(LANDING_COLOR_TOKENS)) {
    const value = colors[key];
    if (typeof value === "string" && /^#([\da-f]{3}|[\da-f]{6})$/i.test(value.trim())) {
      style[token] = value.trim();
    }
  }
  return style as React.CSSProperties;
}

function SectionKicker({ children }: { children: React.ReactNode }) {
  return (
    <h2 className="mb-8 text-[15px] font-semibold tracking-[0.1em] text-muted-foreground uppercase">
      {children}
    </h2>
  );
}

export function LandingBody() {
  const t = useT();
  const [openFaq, setOpenFaq] = useState<number | null>(0);
  const rootRef = useLandingReveal<HTMLDivElement>();
  const { templateBlocks: blocks, templateColors } = useBrand();

  const heroGauges = landingHeroGauges(t);
  const heroStats = landingHeroStats(t);
  const steps = landingSteps(t);
  const bento = landingBento(t);
  const panelPoints = landingPanelPoints(t);
  const panelMetrics = landingPanelMetrics(t);
  const locations = landingLocations(t);
  const faq = landingFaq(t);
  const pricing = landingPricing(t);

  return (
    <div ref={rootRef} style={landingColorStyle(templateColors)}>
      <noscript>
        <style>{".vx-reveal{opacity:1;transform:none}"}</style>
      </noscript>

      <LandingBlock name="hero" blocks={blocks}>
      <section className="relative overflow-hidden border-b border-border px-4 pt-16 pb-16 sm:px-10 sm:pt-[88px] sm:pb-[72px]">
        <div className="landing-glow pointer-events-none absolute -top-[300px] left-1/2 h-[600px] w-[1100px] max-w-none -translate-x-1/2 rounded-full bg-[var(--vx-glow)] blur-[110px]" />

        <div className="relative mx-auto grid max-w-[1240px] items-center gap-12 lg:grid-cols-[1.05fr_0.95fr] lg:gap-14">
          <div>
            <div
              data-reveal
              className="vx-reveal inline-flex items-center gap-2.5 rounded-full border border-[var(--vx-hairline)] bg-[var(--vx-surface-2)] px-3 py-1.5 font-mono text-[11.5px] tracking-[0.08em] text-primary uppercase"
            >
              <span className="vx-pulse size-1.5 rounded-full bg-primary" />
              {landingHeroBadge(t)}
            </div>

            <h1
              data-reveal
              style={{ transitionDelay: "60ms" }}
              className="vx-reveal mt-6 text-[2.6rem] leading-[0.97] font-bold tracking-[-0.042em] text-balance sm:text-[3.2rem] lg:text-[4.25rem]"
            >
              {t("landing.hero.title")}
            </h1>
            <p
              data-reveal
              style={{ transitionDelay: "120ms" }}
              className="vx-reveal mt-5 max-w-[520px] text-[17.5px] leading-[1.62] text-muted-foreground"
            >
              {t("landing.hero.subtitle")}
            </p>

            <div
              data-reveal
              style={{ transitionDelay: "180ms" }}
              className="vx-reveal mt-8 flex flex-wrap gap-3"
            >
              <Link
                href="/register"
                className="vx-btn vx-sheen relative overflow-hidden rounded-[10px] px-[26px] py-[15px] text-[15px] font-semibold shadow-[0_0_44px_var(--vx-glow)]"
              >
                <span className="relative">{t("landing.hero.cta_primary")}</span>
              </Link>
              <button
                type="button"
                onClick={() =>
                  document
                    .getElementById("pricing")
                    ?.scrollIntoView({ behavior: "smooth" })
                }
                className="vx-btn-ghost rounded-[10px] px-[26px] py-[15px] text-[15px] font-medium"
              >
                {t("landing.hero.cta_secondary")}
              </button>
            </div>

            <div
              data-reveal
              style={{ transitionDelay: "240ms" }}
              className="vx-reveal mt-[30px] flex flex-wrap items-center gap-x-[22px] gap-y-2 text-[13px] text-[var(--vx-ink-faint)]"
            >
              <span>{t("landing.hero.perk_free")}</span>
              <span className="size-1 rounded-full bg-[var(--vx-dot)]" />
              <span>{t("landing.hero.perk_no_card")}</span>
              <span className="size-1 rounded-full bg-[var(--vx-dot)]" />
              <span>{t("landing.hero.perk_migration")}</span>
            </div>
          </div>

          <div className="landing-float">
            <div className="overflow-hidden rounded-[14px] border border-[var(--vx-line-strong)] bg-[var(--vx-surface-3)] shadow-[var(--vx-shadow-hero)]">
              <div className="flex items-center gap-2.5 border-b border-border bg-[var(--vx-surface)] px-4 py-3">
                <span className="vx-pulse size-[9px] rounded-full bg-primary" />
                <span className="font-mono text-[11.5px] text-[var(--vx-ink-faint)]">
                  deploy · survival.vortanix.net
                </span>
                <span className="ml-auto font-mono text-[11px] text-primary">
                  40,2 c
                </span>
              </div>
              <div className="flex flex-col gap-1.5 p-4 font-mono text-[12px] leading-[1.5]">
                {LANDING_DEPLOY_LOG.map((l, i) => (
                  <div
                    key={l.text}
                    className="vx-rise flex gap-2.5"
                    style={{ animationDelay: `${i * 0.25}s` }}
                  >
                    <span className="flex-shrink-0 text-[var(--vx-ink-ghost)]">
                      {l.t}
                    </span>
                    <span className={l.color}>{l.text}</span>
                  </div>
                ))}
                <div className="flex gap-2.5 text-primary">
                  <span className="flex-shrink-0 text-[var(--vx-ink-ghost)]">
                    40,2
                  </span>
                  <span>
                    ${" "}
                    <span className="vx-caret inline-block h-[13px] w-[7px] translate-y-[2px] bg-primary" />
                  </span>
                </div>
              </div>
              <div className="grid grid-cols-3 gap-px border-t border-border bg-border">
                {heroGauges.map((g) => (
                  <div key={g.label} className="bg-[var(--vx-surface-4)] p-4">
                    <div className="text-[11.5px] text-[var(--vx-ink-faint)]">
                      {g.label}
                    </div>
                    <div className="mt-1.5 font-mono text-xl font-bold text-primary">
                      {g.value}
                    </div>
                    <div className="mt-2.5 h-[3px] overflow-hidden rounded-full bg-border">
                      <div
                        data-bar={g.pct}
                        className="vx-grow h-[3px] bg-primary"
                      />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        <div className="relative mx-auto mt-16 grid max-w-[1240px] grid-cols-2 gap-px overflow-hidden rounded-xl border border-border bg-border sm:grid-cols-4">
          {heroStats.map((s, i) => (
            <div
              key={s.label}
              data-reveal
              style={{ transitionDelay: `${i * 60}ms` }}
              className="vx-reveal bg-[var(--vx-surface)] px-[22px] py-6"
            >
              <div className="font-mono text-[28px] font-bold tracking-[-0.03em] text-primary sm:text-[34px]">
                {s.value}
              </div>
              <div className="mt-1.5 text-[13px] leading-[1.45] text-muted-foreground">
                {s.label}
              </div>
            </div>
          ))}
        </div>
      </section>
      </LandingBlock>

      <LandingBlock name="games" blocks={blocks}>
      <section
        id="games"
        className="border-b border-border px-4 py-16 sm:px-10 sm:py-[76px]"
      >
        <div className="mx-auto max-w-[1180px]">
          <div className="flex flex-wrap items-baseline justify-between gap-6">
            <div>
              <h2 className="text-[28px] font-bold tracking-[-0.03em] sm:text-[34px]">
                {landingGamesTitle(t)}
              </h2>
              <p className="mt-3 text-[15px] text-muted-foreground">
                {t("landing.games.subtitle")}
              </p>
            </div>
            <Link
              href="/games"
              className="text-[13.5px] whitespace-nowrap text-primary hover:opacity-75"
            >
              {t("landing.games.all_link")}
            </Link>
          </div>

          <div className="mt-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {LANDING_FEATURED_GAMES.map((g, i) => (
              <Link
                key={g.name}
                href="/games"
                data-reveal
                style={{ transitionDelay: `${(i % 3) * 60}ms` }}
                className="vx-reveal block overflow-hidden rounded-[14px] border border-border bg-[var(--vx-surface)] transition-[border-color,transform] duration-200 hover:-translate-y-1 hover:border-[var(--vx-btn-line)]"
              >
                <div className="relative h-[172px] bg-[var(--vx-surface-4)]">
                  <Image
                    src={g.image}
                    alt={g.name}
                    fill
                    unoptimized={g.image.endsWith(".svg")}
                    sizes="(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 380px"
                    className="object-cover"
                  />
                  <div className="vx-scrim-b pointer-events-none absolute inset-0" />
                  <div className="vx-scrim-chip pointer-events-none absolute top-3 left-3 flex size-[38px] items-center justify-center rounded-[9px] border border-[var(--vx-hairline)] font-mono text-[11px] font-bold text-primary">
                    {g.tag}
                  </div>
                </div>
                <div className="flex items-center justify-between gap-3 px-[18px] py-4">
                  <div>
                    <div className="text-[15px] font-semibold tracking-[-0.01em]">
                      {g.name}
                    </div>
                    <div className="mt-[3px] font-mono text-xs text-[var(--vx-ink-faint)]">
                      {t("landing.games.price_from", { price: String(g.from) })}
                    </div>
                  </div>
                  <span className="text-base text-[var(--vx-ink-ghost)]">→</span>
                </div>
              </Link>
            ))}
          </div>

          <div className="mt-4 grid grid-cols-1 gap-px overflow-hidden rounded-xl border border-border bg-border sm:grid-cols-2 lg:grid-cols-3">
            {LANDING_MORE_GAMES.map((g) => (
              <Link
                key={g.name}
                href="/games"
                className="flex items-center gap-3.5 bg-[var(--vx-surface)] px-5 py-[18px] transition-colors hover:bg-[var(--vx-surface-2)]"
              >
                <span className="flex size-[34px] flex-shrink-0 items-center justify-center rounded-[9px] border border-[var(--vx-hairline)] bg-[var(--vx-surface-2)] font-mono text-[10.5px] font-bold text-primary">
                  {g.tag}
                </span>
                <span className="min-w-0">
                  <span className="block text-[14.5px] font-medium">
                    {g.name}
                  </span>
                  <span className="mt-0.5 block font-mono text-[11.5px] text-[var(--vx-ink-faint)]">
                    {t("landing.games.price_from", { price: String(g.from) })}
                  </span>
                </span>
              </Link>
            ))}
          </div>
        </div>
      </section>
      </LandingBlock>

      <LandingBlock name="steps" blocks={blocks}>
      <section className="border-b border-border px-4 py-16 sm:px-10 sm:py-[76px]">
        <div className="mx-auto max-w-[1180px]">
          <SectionKicker>{t("landing.steps.kicker")}</SectionKicker>
          <div className="grid gap-5 md:grid-cols-3">
            {steps.map((s, i) => (
              <div
                key={s.n}
                data-reveal
                style={{ transitionDelay: `${i * 60}ms` }}
                className="vx-reveal relative overflow-hidden rounded-xl border border-border bg-[var(--vx-surface)] px-[26px] pt-7 pb-[30px] transition-[border-color,transform] duration-200 hover:-translate-y-1 hover:border-[var(--vx-btn-line)]"
              >
                <div className="font-mono text-[52px] leading-none font-bold tracking-[-0.04em] text-[var(--vx-numeral)]">
                  {s.n}
                </div>
                <div className="mt-4 text-lg font-semibold">{s.title}</div>
                <div className="mt-2.5 text-sm leading-[1.62] text-muted-foreground">
                  {s.body}
                </div>
                <div className="mt-4 font-mono text-[11.5px] text-primary">
                  {s.meta}
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>
      </LandingBlock>

      <LandingBlock name="hardware" blocks={blocks}>
      <section
        id="hardware"
        className="border-b border-border px-4 py-16 sm:px-10 sm:py-20"
      >
        <div className="mx-auto max-w-[1180px]">
          <div className="grid items-end gap-10 sm:gap-16 lg:grid-cols-[340px_1fr]">
            <h2 className="text-[32px] leading-[1.08] font-bold tracking-[-0.03em] sm:text-[40px]">
              {t("landing.hardware.title")}
            </h2>
            <p className="max-w-[560px] text-[15.5px] leading-[1.65] text-muted-foreground">
              {t("landing.hardware.text")}
            </p>
          </div>
          <div className="mt-9 grid auto-rows-[minmax(150px,auto)] grid-cols-1 gap-3.5 sm:grid-cols-3">
            <div
              data-reveal
              className="vx-reveal vx-raise relative overflow-hidden rounded-[14px] border border-[var(--vx-line-strong)] p-[30px] sm:col-span-2 sm:row-span-2"
            >
              <div className="font-mono text-[11.5px] tracking-[0.08em] text-primary">
                {t("landing.hardware.tps_label")}
              </div>
              <div className="mt-3.5 text-2xl font-bold tracking-[-0.025em] sm:text-[26px]">
                {t("landing.hardware.tps_median")}
              </div>
              <div className="mt-2.5 max-w-[420px] text-sm leading-[1.6] text-muted-foreground">
                {t("landing.hardware.tps_note")}
              </div>
              <div className="mt-[26px] flex h-[120px] items-end gap-1">
                {LANDING_TPS_BARS.map((h, i) => (
                  <div
                    key={i}
                    data-bar-h={h}
                    style={{ transitionDelay: `${i * 0.03}s` }}
                    className={cn(
                      "vx-grow-h flex-1 rounded-t-sm",
                      [18, 19, 20, 21].includes(i) ? "bg-[#7E9A2E]" : "bg-primary"
                    )}
                  />
                ))}
              </div>
              <div className="mt-2 flex justify-between font-mono text-[10.5px] text-[var(--vx-ink-ghost)]">
                <span>00:00</span>
                <span>08:00</span>
                <span>16:00</span>
                <span>23:00</span>
              </div>
            </div>
            {bento.map((f, i) => (
              <div
                key={f.title}
                data-reveal
                style={{ transitionDelay: `${i * 60}ms` }}
                className="vx-reveal flex flex-col rounded-[14px] border border-border bg-[var(--vx-surface)] p-6 transition-[border-color,transform] duration-200 hover:-translate-y-[3px] hover:border-[var(--vx-btn-line)]"
              >
                <div className="font-mono text-[11px] tracking-[0.08em] text-primary">
                  {f.kicker}
                </div>
                <div className="mt-3 text-[16.5px] font-semibold tracking-[-0.01em]">
                  {f.title}
                </div>
                <div className="mt-2 text-[13.5px] leading-[1.6] text-muted-foreground">
                  {f.body}
                </div>
                <div className="mt-auto pt-4 font-mono text-[19px] font-bold text-foreground">
                  {f.metric}
                </div>
              </div>
            ))}
          </div>

          <div
            data-reveal
            className="vx-reveal relative mt-3.5 h-[220px] overflow-hidden rounded-[14px] border border-border bg-[var(--vx-surface-4)] sm:h-[300px]"
          >
            <Image
              src="/landing/datacenter.svg"
              alt={t("landing.hardware.photo_alt")}
              fill
              unoptimized
              sizes="(max-width: 1180px) 100vw, 1180px"
              className="vx-photo object-cover"
            />
            <div className="vx-scrim-l pointer-events-none absolute inset-0" />
            <div className="pointer-events-none absolute bottom-6 left-6 max-w-[420px] sm:left-[30px]">
              <div className="font-mono text-[11px] tracking-[0.08em] text-primary">
                {t("landing.hardware.rack_label")}
              </div>
              <div className="mt-2.5 text-lg font-semibold tracking-[-0.02em] sm:text-xl">
                {t("landing.hardware.rack_text")}
              </div>
            </div>
          </div>
        </div>
      </section>
      </LandingBlock>

      <LandingBlock name="panel" blocks={blocks}>
      <section className="border-b border-border px-4 py-16 sm:px-10 sm:py-20">
        <div className="mx-auto grid max-w-[1180px] items-center gap-14 lg:grid-cols-[0.85fr_1.15fr]">
          <div>
            <h2 className="text-[32px] leading-[1.08] font-bold tracking-[-0.03em] sm:text-[38px]">
              {t("landing.panel.title")}
            </h2>
            <p className="mt-[18px] text-[15.5px] leading-[1.65] text-muted-foreground">
              {t("landing.panel.text")}
            </p>
            <div className="mt-[26px] flex flex-col gap-3">
              {panelPoints.map((p) => (
                <div key={p} className="flex gap-3 text-sm leading-[1.5] text-[var(--vx-spec)]">
                  <span className="text-primary">/</span>
                  <span>{p}</span>
                </div>
              ))}
            </div>
            <Link
              href="/dashboard"
              className="vx-btn-ghost mt-7 inline-block w-fit rounded-[10px] px-6 py-[13px] text-[14.5px] font-medium"
            >
              {t("landing.panel.demo_link")}
            </Link>
          </div>

          <div data-reveal className="vx-reveal flex flex-col gap-3.5">
            <Link
              href="/dashboard"
              className="block overflow-hidden rounded-[14px] border border-[var(--vx-line-strong)] bg-[var(--vx-surface-3)] shadow-[var(--vx-shadow-panel)]"
            >
              <div className="flex items-center gap-2.5 border-b border-border bg-[var(--vx-surface)] px-4 py-[13px]">
                <span className="vx-pulse size-[7px] rounded-full bg-primary" />
                <span className="text-[12.5px] font-semibold">
                  survival.vortanix.net
                </span>
                <span className="ml-auto font-mono text-[11px] text-[var(--vx-ink-faint)]">
                  Paper 1.21.4 · FRA-2
                </span>
              </div>
              <div className="grid grid-cols-2 gap-px bg-border sm:grid-cols-4">
                {panelMetrics.map((m) => (
                  <div key={m.label} className="bg-[var(--vx-surface-4)] p-3.5">
                    <div className="text-[11px] text-[var(--vx-ink-faint)]">
                      {m.label}
                    </div>
                    <div className="mt-[5px] font-mono text-[17px] font-bold text-primary">
                      {m.value}
                    </div>
                    <div className="mt-2 h-[3px] overflow-hidden rounded-full bg-border">
                      <div
                        data-bar={m.pct}
                        className="vx-grow h-[3px] bg-primary"
                      />
                    </div>
                  </div>
                ))}
              </div>
              <div className="flex flex-col gap-[5px] border-t border-border px-4 py-3.5 font-mono text-[11.5px] leading-[1.5]">
                {LANDING_PANEL_PREVIEW_LOG.map((l) => (
                  <div key={l.text} className="flex gap-2.5">
                    <span className="flex-shrink-0 text-[var(--vx-ink-ghost)]">
                      {l.time}
                    </span>
                    <span className={l.color}>{l.text}</span>
                  </div>
                ))}
              </div>
            </Link>

            <div className="relative h-[230px] overflow-hidden rounded-[14px] border border-border bg-[var(--vx-surface-4)]">
              <DashboardPreview className="w-[182%] origin-top-left scale-[0.55] rounded-none border-0 shadow-none ring-0" />
              <div className="vx-scrim-b-hard pointer-events-none absolute inset-x-0 bottom-0 h-20" />
            </div>
          </div>
        </div>
      </section>
      </LandingBlock>

      <LandingBlock name="locations" blocks={blocks}>
      <section className="border-b border-border px-4 py-16 sm:px-10 sm:py-20">
        <div className="mx-auto max-w-[1180px]">
          <SectionKicker>{t("landing.locations.kicker")}</SectionKicker>
          <div className="overflow-hidden overflow-x-auto rounded-xl border border-border">
            <div className="grid min-w-[680px] grid-cols-[1.4fr_1fr_1fr_1.2fr_0.8fr] bg-[var(--vx-surface-2)] px-[22px] py-[13px] font-mono text-[11px] tracking-[0.08em] text-[var(--vx-ink-faint)] uppercase">
              <div>{t("landing.locations.col_location")}</div>
              <div>{t("landing.locations.col_ping")}</div>
              <div>{t("landing.locations.col_uplink")}</div>
              <div>{t("landing.locations.col_capacity")}</div>
              <div>{t("landing.locations.col_status")}</div>
            </div>
            {locations.map((l) => (
              <div
                key={l.city}
                className="grid min-w-[680px] grid-cols-[1.4fr_1fr_1fr_1.2fr_0.8fr] items-center border-t border-border px-[22px] py-4 text-sm transition-colors hover:bg-[var(--vx-surface-2)]"
              >
                <div className="font-medium">{l.city}</div>
                <div className="font-mono text-primary">{l.ping}</div>
                <div className="font-mono text-[var(--vx-ink-dim)]">
                  {l.uplink}
                </div>
                <div className="flex items-center gap-2.5">
                  <span className="h-1 max-w-[110px] flex-1 overflow-hidden rounded-full bg-border">
                    <span
                      data-bar={l.pct}
                      className={cn(
                        "vx-grow block h-1",
                        l.pct > 85 ? "bg-[var(--vx-warn)]" : "bg-primary"
                      )}
                    />
                  </span>
                  <span className="font-mono text-[12.5px] text-[var(--vx-ink-dim)]">
                    {l.load}
                  </span>
                </div>
                <div
                  className={cn(
                    "text-xs",
                    l.ok ? "text-primary" : "text-[var(--vx-warn)]"
                  )}
                >
                  {l.status}
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>
      </LandingBlock>

      <LandingBlock name="faq" blocks={blocks}>
      <section
        id="faq"
        className="border-b border-border px-4 py-16 sm:px-10 sm:py-20"
      >
        <div className="mx-auto max-w-[900px]">
          <SectionKicker>{t("landing.faq.kicker")}</SectionKicker>
          <div className="overflow-hidden rounded-xl border border-border">
            {faq.map((f, i) => {
              const isOpen = openFaq === i;
              return (
                <div
                  key={f.q}
                  className="border-t border-border bg-[var(--vx-surface)] first:border-t-0"
                >
                  <button
                    type="button"
                    onClick={() => setOpenFaq(isOpen ? null : i)}
                    aria-expanded={isOpen}
                    className="flex w-full items-center gap-5 px-[22px] py-5 text-left transition-colors hover:bg-[var(--vx-surface-2)]"
                  >
                    <span className="flex-1 text-base font-medium">{f.q}</span>
                    <span
                      className={cn(
                        "flex-shrink-0 font-mono text-lg text-primary transition-transform duration-250",
                        isOpen && "rotate-45"
                      )}
                    >
                      +
                    </span>
                  </button>
                  <div
                    className={cn(
                      "grid transition-[grid-template-rows,opacity] duration-300",
                      isOpen
                        ? "grid-rows-[1fr] opacity-100"
                        : "grid-rows-[0fr] opacity-0"
                    )}
                  >
                    <div className="overflow-hidden">
                      <div className="max-w-[680px] px-[22px] pb-[22px] text-[14.5px] leading-[1.65] text-muted-foreground">
                        {f.a}
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </section>
      </LandingBlock>

      <LandingBlock name="pricing" blocks={blocks}>
      <section
        id="pricing"
        className="border-b border-border px-4 py-16 sm:px-10 sm:py-20"
      >
        <div className="mx-auto max-w-[1180px]">
          <SectionKicker>{t("landing.pricing.kicker")}</SectionKicker>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {pricing.map((p, i) => (
              <div
                key={p.id}
                data-reveal
                style={{ transitionDelay: `${i * 60}ms` }}
                className={cn(
                  "vx-reveal flex flex-col rounded-[14px] border p-6 transition-[border-color,transform] duration-200 hover:-translate-y-[5px] hover:border-[var(--vx-btn-line)]",
                  p.highlighted
                    ? "vx-raise border-[var(--vx-btn-line)]"
                    : "border-border bg-[var(--vx-surface)]"
                )}
              >
                <div className="flex items-center justify-between">
                  <div className="text-[17px] font-bold">{p.name}</div>
                  <div className="font-mono text-[10.5px] tracking-[0.08em] text-primary uppercase">
                    {p.badge}
                  </div>
                </div>
                <div className="mt-[18px] flex items-baseline gap-1.5">
                  <span className="font-mono text-[38px] font-bold tracking-[-0.03em]">
                    {p.price}
                  </span>
                  <span className="text-[13px] text-[var(--vx-ink-faint)]">
                    {t("landing.pricing.per_month")}
                  </span>
                </div>
                <div className="mt-1.5 font-mono text-[12.5px] text-[var(--vx-ink-ghost)]">
                  {t("landing.pricing.per_hour", { price: String(p.hourly) })}
                </div>
                <div className="mt-[22px] flex flex-col gap-2.5">
                  {p.specs.map((s) => (
                    <div key={s} className="flex gap-2.5 text-[13.5px] text-[var(--vx-spec)]">
                      <span className="text-primary">/</span>
                      <span>{s}</span>
                    </div>
                  ))}
                </div>
                <div className="mt-auto pt-[26px]">
                  <Link
                    href="/register"
                    className={cn(
                      "block rounded-[9px] py-3 text-center text-sm font-semibold",
                      p.highlighted ? "vx-btn" : "vx-btn-ghost"
                    )}
                  >
                    {t("landing.pricing.choose")}
                  </Link>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>
      </LandingBlock>

      <LandingBlock name="cta" blocks={blocks}>
      <section className="px-4 py-16 sm:px-10 sm:pt-20 sm:pb-25">
        <div
          data-reveal
          className="vx-reveal vx-raise relative mx-auto flex max-w-[1180px] flex-col items-start gap-8 overflow-hidden rounded-2xl border border-[var(--vx-hairline)] px-7 py-12 sm:flex-row sm:items-center sm:justify-between sm:px-12 sm:py-14"
        >
          <div className="landing-glow vx-glow-radial pointer-events-none absolute -top-[160px] -right-[80px] size-[420px] rounded-full" />
          <div className="relative">
            <h2 className="text-[32px] leading-[1.1] font-bold tracking-[-0.03em] sm:text-[38px]">
              {t("landing.cta.title")}
            </h2>
            <p className="mt-3.5 max-w-[520px] text-[15.5px] text-[var(--vx-ink-dim)]">
              {t("landing.cta.text")}
            </p>
          </div>
          <Link
            href="/register"
            className="vx-btn vx-sheen relative overflow-hidden rounded-[10px] px-[30px] py-4 text-[15px] font-semibold whitespace-nowrap"
          >
            <span className="relative">{t("landing.cta.button")}</span>
          </Link>
        </div>
      </section>
      </LandingBlock>
    </div>
  );
}
