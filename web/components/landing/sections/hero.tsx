"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { AnimatePresence, m, useReducedMotionConfig } from "motion/react";
import { ArrowRight, Check, Copy } from "lucide-react";
import { HERO_GAMES } from "@/components/landing/landing-content";
import { EASE_OUT, Magnetic, WordsReveal } from "@/components/landing/motion";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";

const LINE_INTERVAL = 520;
const AUTOPLAY_DELAY = 5200;

export function HeroSection({ showPricingLink }: { showPricingLink: boolean }) {
  const t = useT();

  const scrollToPricing = () =>
    document.getElementById("pricing")?.scrollIntoView({ behavior: "smooth", block: "start" });

  return (
    <section className="relative overflow-hidden border-b border-border">
      <div className="mx-auto grid max-w-[1240px] items-center gap-14 px-5 pt-14 pb-16 sm:px-8 sm:pt-20 lg:grid-cols-[1.05fr_0.95fr] lg:gap-16 lg:pt-24 lg:pb-24">
        <div className="min-w-0">
          <m.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, ease: EASE_OUT }}
            className="flex items-center gap-2.5 font-mono text-[12px] text-muted-foreground"
          >
            <span className="relative flex size-2">
              <span className="absolute inset-0 animate-ping rounded-full bg-emerald-500/60 [animation-duration:2.4s]" />
              <span className="relative size-2 rounded-full bg-emerald-500" />
            </span>
            {t("landing.hero.badge")}
          </m.div>

          <h1 className="vx-display mt-6 text-[2.35rem] leading-[1.04] font-semibold tracking-[-0.035em] text-balance sm:text-[3.1rem] lg:text-[3.7rem]">
            <WordsReveal immediate delay={0.1} text={t("landing.hero.title")} />
          </h1>

          <m.p
            initial={{ opacity: 0, y: 14 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8, ease: EASE_OUT, delay: 0.45 }}
            className="mt-6 max-w-[540px] text-[17px] leading-[1.6] text-muted-foreground"
          >
            {t("landing.hero.subtitle")}
          </m.p>

          <m.div
            initial={{ opacity: 0, y: 14 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8, ease: EASE_OUT, delay: 0.6 }}
            className="mt-9 flex flex-wrap items-center gap-x-7 gap-y-4"
          >
            <Magnetic>
              <Link
                href="/register"
                className="group inline-flex items-center gap-3 rounded-full bg-primary py-3.5 pr-3.5 pl-6 text-[15px] font-semibold text-primary-foreground transition-[transform,opacity] active:scale-[0.97]"
              >
                {t("landing.hero.cta_primary")}
                <span className="flex size-8 items-center justify-center rounded-full bg-primary-foreground/10 transition-transform duration-300 group-hover:translate-x-0.5">
                  <ArrowRight className="size-4" />
                </span>
              </Link>
            </Magnetic>
            {showPricingLink && (
              <button
                type="button"
                onClick={scrollToPricing}
                className="group relative text-[15px] font-medium text-foreground"
              >
                {t("landing.hero.cta_secondary")}
                <span className="absolute -bottom-1 left-0 h-px w-full origin-left scale-x-100 bg-foreground/40 transition-transform duration-300 group-hover:scale-x-0" />
                <span className="absolute -bottom-1 left-0 h-px w-full origin-right scale-x-0 bg-foreground transition-transform delay-150 duration-300 group-hover:origin-left group-hover:scale-x-100" />
              </button>
            )}
          </m.div>

          <m.ul
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.8, delay: 0.8 }}
            className="mt-9 flex flex-wrap gap-x-6 gap-y-2 text-[13.5px] text-muted-foreground"
          >
            {["perk_free", "perk_no_card", "perk_migration"].map((key) => (
              <li key={key} className="flex items-center gap-2">
                <Check className="size-3.5 text-emerald-500" />
                {t(`landing.hero.${key}`)}
              </li>
            ))}
          </m.ul>
        </div>

        <m.div
          initial={{ opacity: 0, y: 40, rotate: 1.5 }}
          animate={{ opacity: 1, y: 0, rotate: 0 }}
          transition={{ duration: 1.1, ease: EASE_OUT, delay: 0.25 }}
          className="min-w-0"
        >
          <ServerWindow />
        </m.div>
      </div>
    </section>
  );
}

function ServerWindow() {
  const t = useT();
  const reduced = useReducedMotionConfig();
  const [index, setIndex] = useState(0);
  const [lines, setLines] = useState(0);
  const [autoplay, setAutoplay] = useState(true);
  const [hovered, setHovered] = useState(false);
  const [copied, setCopied] = useState(false);
  const copyTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const game = HERO_GAMES[index];
  const online = lines >= game.log.length;
  const players = online ? Math.max(1, Math.round(game.slots * 0.35)) : 0;
  const address = `203.0.113.24:${game.port}`;

  useEffect(() => {
    if (reduced) {
      setLines(game.log.length);
      return;
    }
    setLines(0);
    const timers = game.log.map((_, i) =>
      setTimeout(() => setLines(i + 1), LINE_INTERVAL * (i + 1))
    );
    return () => timers.forEach(clearTimeout);
  }, [index, game.log, reduced]);

  useEffect(() => {
    if (reduced || !autoplay || hovered || !online) return;
    const timer = setTimeout(
      () => setIndex((current) => (current + 1) % HERO_GAMES.length),
      AUTOPLAY_DELAY
    );
    return () => clearTimeout(timer);
  }, [online, autoplay, hovered, reduced]);

  useEffect(() => () => {
    if (copyTimer.current) clearTimeout(copyTimer.current);
  }, []);

  const pick = (next: number) => {
    setAutoplay(false);
    setIndex(next);
  };

  const copyAddress = async () => {
    try {
      await navigator.clipboard.writeText(address);
      setCopied(true);
      if (copyTimer.current) clearTimeout(copyTimer.current);
      copyTimer.current = setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  };

  return (
    <div onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)}>
      <div
        role="tablist"
        aria-label={t("landing.hero.window_pick")}
        className="no-scrollbar -mx-1 flex gap-1 overflow-x-auto px-1 pb-4"
      >
        {HERO_GAMES.map((item, i) => {
          const active = i === index;
          return (
            <button
              key={item.id}
              type="button"
              role="tab"
              aria-selected={active}
              onClick={() => pick(i)}
              className={cn(
                "relative shrink-0 rounded-full px-3.5 py-1.5 text-[13px] font-medium transition-colors",
                active ? "text-primary-foreground" : "text-muted-foreground hover:text-foreground"
              )}
            >
              {active && (
                <m.span
                  layoutId="hero-game-pill"
                  className="absolute inset-0 rounded-full bg-primary"
                  transition={{ type: "spring", stiffness: 420, damping: 34 }}
                />
              )}
              <span className="relative">{item.name}</span>
            </button>
          );
        })}
      </div>

      <div className="relative">
        <div className="absolute inset-x-6 -bottom-3 h-full rounded-2xl border border-border bg-card/60" />
        <div className="relative overflow-hidden rounded-2xl border border-border bg-card shadow-[0_30px_80px_-30px_rgba(0,0,0,0.45)]">
          <div className="flex items-center gap-3 border-b border-border px-5 py-3.5">
            <div className="min-w-0 flex-1">
              <AnimatePresence mode="wait" initial={false}>
                <m.div
                  key={game.id}
                  initial={{ opacity: 0, y: 6 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -6 }}
                  transition={{ duration: 0.25 }}
                  className="flex min-w-0 items-baseline gap-2"
                >
                  <span className="truncate font-mono text-[13px] text-foreground">{game.server}</span>
                  <span className="truncate font-mono text-[11.5px] text-muted-foreground">{game.build}</span>
                </m.div>
              </AnimatePresence>
            </div>
            <StatusBadge online={online} />
          </div>

          <div className="h-[184px] overflow-hidden bg-background/40 px-5 py-4 font-mono text-[12.5px] leading-[1.75]">
            <AnimatePresence initial={false}>
              {game.log.slice(0, lines).map((line, i) => (
                <m.div
                  key={`${game.id}-${i}`}
                  initial={{ opacity: 0, x: -6 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ duration: 0.3 }}
                  className={cn(
                    "truncate",
                    i === game.log.length - 1 ? "text-emerald-500" : "text-muted-foreground"
                  )}
                >
                  <span className="mr-3 text-muted-foreground/50">{String(i + 1).padStart(2, "0")}</span>
                  {line}
                </m.div>
              ))}
            </AnimatePresence>
            {!online && (
              <span className="mt-1 inline-block h-[14px] w-[7px] translate-y-[3px] animate-pulse bg-foreground/70" />
            )}
          </div>

          <div className="grid grid-cols-3 border-t border-border">
            <Stat label={t("landing.hero.window_players")} value={`${players}/${game.slots}`} />
            <Stat label="RAM" value={t("landing.unit.gb", { value: game.ramGb })} />
            <Stat
              label={t("landing.hero.window_price")}
              value={t("landing.unit.rub_month", { price: game.price.toLocaleString(localeTag()) })}
            />
          </div>

          <div className="flex items-center justify-between gap-3 border-t border-border px-5 py-3">
            <span className="truncate font-mono text-[12.5px] text-muted-foreground">{address}</span>
            <button
              type="button"
              onClick={() => void copyAddress()}
              disabled={!online}
              className="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-border px-3 py-1 text-[12px] font-medium text-foreground transition-[opacity,background-color] hover:bg-accent disabled:opacity-40"
            >
              {copied ? <Check className="size-3.5 text-emerald-500" /> : <Copy className="size-3.5" />}
              {copied ? t("landing.hero.window_copied") : t("landing.hero.window_copy")}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function StatusBadge({ online }: { online: boolean }) {
  const t = useT();
  return (
    <m.span
      layout
      className={cn(
        "inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2.5 py-1 text-[11.5px] font-medium",
        online
          ? "border-emerald-500/30 text-emerald-600 dark:text-emerald-400"
          : "border-amber-500/30 text-amber-600 dark:text-amber-400"
      )}
    >
      <span
        className={cn(
          "size-1.5 rounded-full",
          online ? "bg-emerald-500" : "animate-pulse bg-amber-500"
        )}
      />
      {online ? t("landing.hero.window_online") : t("landing.hero.window_starting")}
    </m.span>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="border-r border-border px-5 py-3.5 last:border-r-0">
      <div className="text-[11.5px] text-muted-foreground">{label}</div>
      <AnimatePresence mode="popLayout" initial={false}>
        <m.div
          key={value}
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -8 }}
          transition={{ duration: 0.28 }}
          className="mt-1 truncate font-mono text-[14px] text-foreground tabular-nums"
        >
          {value}
        </m.div>
      </AnimatePresence>
    </div>
  );
}
