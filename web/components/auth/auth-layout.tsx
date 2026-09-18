"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { AnimatePresence, animate, m, useInView, useReducedMotionConfig } from "motion/react";
import { ArrowLeft } from "lucide-react";
import { BrandLogo } from "@/components/brand-logo";
import { landingFontVariables } from "@/components/landing/fonts";
import { HERO_GAMES } from "@/components/landing/landing-content";
import { EASE_OUT, MotionRoot, WordsReveal } from "@/components/landing/motion";
import { EditableSection, useSectionHidden } from "@/components/site/editable-section";
import { PageSlot } from "@/components/site/page-slot";
import { useBrand } from "@/context/brand-provider";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

type Scene = "login" | "register" | "forgot" | "reset" | "two_factor";

const SCENES: Record<string, Scene> = {
  "/login": "login",
  "/register": "register",
  "/forgot-password": "forgot",
  "/reset-password": "reset",
  "/two-factor-challenge": "two_factor",
};

const SHORT_NAMES: Record<string, string> = {
  cs2: "CS2",
};

const START_DURATION = 2.4;
const ONLINE_HOLD = 3200;

export function AuthLayout({ children }: { children: ReactNode }) {
  const t = useT();
  const pathname = usePathname();
  const { name } = useBrand();
  const scene = SCENES[pathname] ?? "login";
  const showcaseHidden = useSectionHidden("auth.showcase");

  return (
    <MotionRoot>
      <div className={cn(landingFontVariables, "font-landing min-h-screen bg-background text-foreground antialiased")}>
        <div className={cn("grid min-h-screen", !showcaseHidden && "lg:grid-cols-[minmax(0,1fr)_minmax(0,1.08fr)]")}>
          <div className="flex min-h-screen min-w-0 flex-col px-5 sm:px-10">
            <header className="flex h-20 shrink-0 items-center justify-between gap-4">
              <Link href="/" aria-label={t("landing.header.home_aria")} className="flex items-center">
                <BrandLogo size="sm" className="h-8 max-w-[176px]" priority />
              </Link>
              <Link
                href="/"
                className="group inline-flex items-center gap-1.5 text-[14px] text-muted-foreground transition-colors hover:text-foreground"
              >
                <ArrowLeft className="size-4 transition-transform duration-300 group-hover:-translate-x-0.5" />
                {t("auth.layout.home")}
              </Link>
            </header>

            <main className="flex flex-1 items-center justify-center py-8">
              <m.div
                key={pathname}
                initial={{ opacity: 0, y: 16 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.6, ease: EASE_OUT }}
                className="w-full max-w-[420px]"
              >
                <PageSlot position="top" />
                {children}
                <PageSlot position="bottom" />
              </m.div>
            </main>

            <footer className="flex h-16 shrink-0 items-center text-[12.5px] text-muted-foreground">
              © {new Date().getFullYear()} {name}
            </footer>
          </div>

          <EditableSection id="auth.showcase">
            <aside className="hidden p-3 lg:block">
              <AuthScene scene={scene} />
            </aside>
          </EditableSection>
        </div>
      </div>
    </MotionRoot>
  );
}

function AuthScene({ scene }: { scene: Scene }) {
  const t = useT();
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref);
  const [gameIndex, setGameIndex] = useState(0);
  const game = HERO_GAMES[gameIndex];

  return (
    <div
      ref={ref}
      className="sticky top-3 flex h-[calc(100vh-24px)] min-h-[640px] flex-col overflow-hidden rounded-[28px] bg-[#0b0c0f] p-10 text-white xl:p-14 dark:bg-[#141519] dark:ring-1 dark:ring-white/[0.06]"
    >
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 opacity-[0.12] [mask-image:radial-gradient(ellipse_80%_70%_at_70%_20%,black,transparent)]"
        style={{
          backgroundImage: "radial-gradient(circle at center, currentColor 1.1px, transparent 1.3px)",
          backgroundSize: "24px 24px",
        }}
      />
      <m.div
        aria-hidden
        className="pointer-events-none absolute -top-40 -right-40 size-[520px] rounded-full bg-white/[0.07] blur-3xl"
        animate={{ x: [0, -90, 30, 0], y: [0, 70, 140, 0] }}
        transition={{ duration: 18, repeat: Infinity, ease: "easeInOut" }}
      />

      <div className="relative flex items-center gap-2.5 font-mono text-[12px] text-white/60">
        <span className="relative flex size-2">
          <span className="absolute inset-0 animate-ping rounded-full bg-emerald-400/60 [animation-duration:2.4s]" />
          <span className="relative size-2 rounded-full bg-emerald-400" />
        </span>
        {t("auth.scene.label")}
      </div>

      <div className="relative mt-auto">
        <AnimatePresence mode="wait" initial={false}>
          <m.div
            key={scene}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0, y: -12, transition: { duration: 0.25 } }}
            transition={{ duration: 0.3 }}
          >
            {scene === "register" ? (
              <h2 className="vx-display text-[clamp(2.3rem,3.5vw,3.5rem)] leading-[1.02] font-semibold tracking-[-0.035em]">
                <WordsReveal immediate text={t("auth.scene.register_before")} />
                <span className="relative block h-[1.1em] overflow-hidden">
                  <AnimatePresence mode="popLayout" initial={false}>
                    <m.span
                      key={game.id}
                      initial={{ y: "100%", opacity: 0, filter: "blur(6px)" }}
                      animate={{ y: "0%", opacity: 1, filter: "blur(0px)" }}
                      exit={{ y: "-100%", opacity: 0, filter: "blur(6px)" }}
                      transition={{ duration: 0.6, ease: EASE_OUT }}
                      className="block text-emerald-400"
                    >
                      {SHORT_NAMES[game.id] ?? game.name}
                    </m.span>
                  </AnimatePresence>
                </span>
                <WordsReveal immediate delay={0.15} text={t("auth.scene.register_after")} />
              </h2>
            ) : (
              <h2 className="vx-display text-[clamp(2.3rem,3.5vw,3.5rem)] leading-[1.02] font-semibold tracking-[-0.035em] text-balance">
                <WordsReveal immediate text={t(`auth.scene.${scene}_title`)} />
              </h2>
            )}
            <m.p
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.7, ease: EASE_OUT, delay: 0.25 }}
              className="mt-6 max-w-[460px] text-[16.5px] leading-[1.6] text-white/65"
            >
              {t(`auth.scene.${scene}_text`)}
            </m.p>
          </m.div>
        </AnimatePresence>
      </div>

      <div className="relative mt-12">
        <MiniServer
          active={inView}
          gameIndex={gameIndex}
          onNext={() => setGameIndex((index) => (index + 1) % HERO_GAMES.length)}
        />
        <ul className="mt-8 grid grid-cols-3 border-t border-white/15 pt-6 text-[13px] leading-[1.45] text-white/60">
          {[1, 2, 3].map((n) => (
            <li key={n} className="border-l border-white/15 px-4 first:border-l-0 first:pl-0">
              {t(`auth.scene.fact${n}`)}
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

function MiniServer({
  active,
  gameIndex,
  onNext,
}: {
  active: boolean;
  gameIndex: number;
  onNext: () => void;
}) {
  const t = useT();
  const reduced = useReducedMotionConfig();
  const game = HERO_GAMES[gameIndex];
  const [progress, setProgress] = useState(100);
  const [players, setPlayers] = useState(Math.round(game.slots * 0.4));
  const onNextRef = useRef(onNext);
  onNextRef.current = onNext;

  useEffect(() => {
    const target = Math.max(1, Math.round(game.slots * 0.4));
    if (reduced || !active) {
      setProgress(100);
      setPlayers(target);
      return;
    }
    setProgress(0);
    setPlayers(0);
    let hold: ReturnType<typeof setTimeout> | undefined;
    let joining: { stop: () => void } | undefined;
    const starting = animate(0, 100, {
      duration: START_DURATION,
      ease: [0.4, 0, 0.2, 1],
      onUpdate: (value) => setProgress(value),
      onComplete: () => {
        joining = animate(0, target, {
          duration: 1.6,
          ease: EASE_OUT,
          onUpdate: (value) => setPlayers(Math.round(value)),
        });
        hold = setTimeout(() => onNextRef.current(), ONLINE_HOLD);
      },
    });
    return () => {
      starting.stop();
      joining?.stop();
      if (hold) clearTimeout(hold);
    };
  }, [game.slots, gameIndex, active, reduced]);

  const online = progress >= 100;

  return (
    <div className="rounded-2xl border border-white/15 bg-white/[0.04] p-5 backdrop-blur-sm">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <AnimatePresence mode="popLayout" initial={false}>
            <m.div
              key={game.id}
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.3 }}
              className="flex min-w-0 items-baseline gap-2.5"
            >
              <span className="truncate font-mono text-[14px]">{game.server}</span>
              <span className="truncate font-mono text-[12px] text-white/50">{game.build}</span>
            </m.div>
          </AnimatePresence>
        </div>
        <span
          className={cn(
            "inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2.5 py-1 text-[12px] font-medium transition-colors duration-300",
            online ? "border-emerald-400/40 text-emerald-300" : "border-amber-400/40 text-amber-300"
          )}
        >
          <span className={cn("size-1.5 rounded-full", online ? "bg-emerald-400" : "animate-pulse bg-amber-400")} />
          {online ? t("landing.hero.window_online") : t("landing.hero.window_starting")}
        </span>
      </div>

      <div className="mt-4 h-1 overflow-hidden rounded-full bg-white/15">
        <div
          className={cn("h-full rounded-full", online ? "bg-emerald-400" : "bg-white")}
          style={{ width: `${progress}%` }}
        />
      </div>

      <div className="mt-4 flex items-center justify-between gap-4 font-mono text-[12.5px] text-white/60">
        <span className="truncate">203.0.113.24:{game.port}</span>
        <span className="shrink-0 tabular-nums">
          {players}/{game.slots}
        </span>
      </div>
    </div>
  );
}
