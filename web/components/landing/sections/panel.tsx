"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { AnimatePresence, m, useInView, useReducedMotionConfig } from "motion/react";
import { ArrowUpRight, File, Folder, RotateCcw } from "lucide-react";
import {
  PANEL_FILES,
  PANEL_LOG,
  landingPanelPoints,
  landingPanelTabs,
} from "@/components/landing/landing-content";
import { EASE_OUT, Reveal, SectionLabel, WordsReveal } from "@/components/landing/motion";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

const TAB_DURATION = 5600;

const TONE: Record<string, string> = {
  ok: "text-emerald-600 dark:text-emerald-400",
  warn: "text-amber-600 dark:text-amber-400",
  danger: "text-rose-600 dark:text-rose-400",
  muted: "text-muted-foreground",
};

export function PanelSection({ index }: { index: string }) {
  const t = useT();
  const points = landingPanelPoints(t);

  return (
    <section className="border-b border-border">
      <div className="mx-auto grid max-w-[1240px] gap-14 px-5 py-20 sm:px-8 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)] lg:items-center lg:gap-16 lg:py-28">
        <div>
          <SectionLabel index={index}>{t("landing.panel.label")}</SectionLabel>
          <h2 className="vx-display mt-5 text-[2rem] leading-[1.08] font-semibold tracking-[-0.03em] text-balance sm:text-[2.6rem]">
            <WordsReveal text={t("landing.panel.title")} />
          </h2>
          <Reveal>
            <p className="mt-6 text-[16px] leading-[1.6] text-muted-foreground">{t("landing.panel.text")}</p>
          </Reveal>
          <ul className="mt-8 flex flex-col">
            {points.map((point, i) => (
              <Reveal as="li" key={point} delay={i * 0.07} y={12} className="flex gap-4 border-t border-border py-3.5 text-[15px] last:border-b">
                <span className="font-mono text-[12px] leading-[1.7] text-muted-foreground">0{i + 1}</span>
                <span>{point}</span>
              </Reveal>
            ))}
          </ul>
          <Reveal>
            <Link href="/dashboard" className="group mt-8 inline-flex items-center gap-1.5 text-[15px] font-medium">
              {t("landing.panel.open_link")}
              <ArrowUpRight className="size-4 transition-transform duration-300 group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
            </Link>
          </Reveal>
        </div>

        <Reveal>
          <PanelWindow />
        </Reveal>
      </div>
    </section>
  );
}

function PanelWindow() {
  const t = useT();
  const tabs = landingPanelTabs(t);
  const reduced = useReducedMotionConfig();
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, { amount: 0.4 });
  const [active, setActive] = useState(0);
  const [auto, setAuto] = useState(true);
  const [paused, setPaused] = useState(false);
  const running = auto && !paused && inView && !reduced;

  useEffect(() => {
    if (!running) return;
    const timer = setTimeout(() => setActive((i) => (i + 1) % tabs.length), TAB_DURATION);
    return () => clearTimeout(timer);
  }, [running, active, tabs.length]);

  return (
    <div
      ref={ref}
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      className="overflow-hidden rounded-2xl border border-border bg-card shadow-[0_30px_80px_-40px_rgba(0,0,0,0.5)]"
    >
      <div className="flex items-center justify-between gap-4 border-b border-border px-5 py-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <span className="size-2 rounded-full bg-emerald-500" />
          <span className="truncate font-mono text-[13px]">survival</span>
          <span className="hidden font-mono text-[12px] text-muted-foreground sm:inline">Paper 1.21.4</span>
        </div>
        <span className="font-mono text-[12px] text-muted-foreground">17 / 50</span>
      </div>

      <div role="tablist" className="no-scrollbar flex overflow-x-auto border-b border-border px-2">
        {tabs.map((tab, i) => {
          const current = i === active;
          return (
            <button
              key={tab.id}
              type="button"
              role="tab"
              aria-selected={current}
              onClick={() => {
                setAuto(false);
                setActive(i);
              }}
              className={cn(
                "relative shrink-0 px-3.5 py-3 text-[13.5px] font-medium transition-colors",
                current ? "text-foreground" : "text-muted-foreground hover:text-foreground"
              )}
            >
              {tab.label}
              {current && (
                <m.span
                  layoutId="panel-tab-line"
                  className="absolute inset-x-2 -bottom-px h-px bg-border"
                  transition={{ type: "spring", stiffness: 380, damping: 32 }}
                >
                  <m.span
                    key={`${active}-${running}`}
                    className="absolute inset-y-0 left-0 bg-foreground"
                    initial={{ width: running ? "0%" : "100%" }}
                    animate={{ width: "100%" }}
                    transition={{ duration: running ? TAB_DURATION / 1000 : 0, ease: "linear" }}
                  />
                </m.span>
              )}
            </button>
          );
        })}
      </div>

      <div className="relative h-[300px] overflow-hidden">
        <AnimatePresence mode="wait" initial={false}>
          <m.div
            key={tabs[active].id}
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -12 }}
            transition={{ duration: 0.35, ease: EASE_OUT }}
            className="absolute inset-0"
          >
            {tabs[active].id === "console" && <ConsoleTab />}
            {tabs[active].id === "files" && <FilesTab />}
            {tabs[active].id === "backups" && <BackupsTab />}
            {tabs[active].id === "schedule" && <ScheduleTab />}
          </m.div>
        </AnimatePresence>
      </div>
    </div>
  );
}

function ConsoleTab() {
  const t = useT();
  return (
    <div className="flex h-full flex-col">
      <div className="flex-1 overflow-hidden bg-background/40 px-5 py-4 font-mono text-[12.5px] leading-[1.8]">
        {PANEL_LOG.map((line, i) => (
          <m.div
            key={line.time}
            initial={{ opacity: 0, x: -6 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.3, delay: i * 0.08 }}
            className="flex gap-3 truncate"
          >
            <span className="shrink-0 text-muted-foreground/60">{line.time}</span>
            <span className={cn("truncate", TONE[line.tone])}>{line.text}</span>
          </m.div>
        ))}
      </div>
      <div className="flex items-center gap-3 border-t border-border px-5 py-3 font-mono text-[12.5px]">
        <span className="text-muted-foreground">&gt;</span>
        <TypedText text={t("landing.panel.command")} />
      </div>
    </div>
  );
}

function TypedText({ text }: { text: string }) {
  const reduced = useReducedMotionConfig();
  const [count, setCount] = useState(0);

  useEffect(() => {
    if (reduced) {
      setCount(text.length);
      return;
    }
    setCount(0);
    let i = 0;
    const timer = setInterval(() => {
      i += 1;
      setCount(i);
      if (i >= text.length) clearInterval(timer);
    }, 45);
    return () => clearInterval(timer);
  }, [text, reduced]);

  return (
    <span className="truncate">
      {text.slice(0, count)}
      <span className="ml-px inline-block h-[13px] w-[7px] translate-y-[2px] animate-pulse bg-foreground/70" />
    </span>
  );
}

function FilesTab() {
  const t = useT();
  return (
    <div className="h-full overflow-hidden">
      <div className="grid grid-cols-[1fr_90px] border-b border-border px-5 py-2 font-mono text-[11.5px] text-muted-foreground sm:grid-cols-[1fr_90px_120px]">
        <span>{t("landing.panel.col_name")}</span>
        <span className="text-right">{t("landing.panel.col_size")}</span>
        <span className="hidden text-right sm:block">{t("landing.panel.col_changed")}</span>
      </div>
      {PANEL_FILES.map((file, i) => (
        <m.div
          key={file.name}
          initial={{ opacity: 0, y: 6 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: i * 0.05 }}
          className="grid grid-cols-[1fr_90px] items-center border-b border-border/60 px-5 py-2.5 text-[13.5px] last:border-b-0 sm:grid-cols-[1fr_90px_120px]"
        >
          <span className="flex min-w-0 items-center gap-2.5">
            {file.dir ? (
              <Folder className="size-4 shrink-0 text-muted-foreground" />
            ) : (
              <File className="size-4 shrink-0 text-muted-foreground" />
            )}
            <span className="truncate font-mono text-[13px]">{file.name}</span>
          </span>
          <span className="text-right font-mono text-[12px] text-muted-foreground">{file.size}</span>
          <span className="hidden text-right text-[12px] text-muted-foreground sm:block">
            {t(i < 2 ? "landing.panel.changed_today" : "landing.panel.changed_yesterday")}
          </span>
        </m.div>
      ))}
    </div>
  );
}

function BackupsTab() {
  const t = useT();
  const rows = [
    { label: t("landing.panel.backup_now"), size: "3.1 GB", auto: true },
    { label: t("landing.panel.backup_6h"), size: "3.1 GB", auto: true },
    { label: t("landing.panel.backup_manual"), size: "2.9 GB", auto: false },
    { label: t("landing.panel.backup_yesterday"), size: "2.8 GB", auto: true },
  ];
  return (
    <div className="h-full overflow-hidden px-5 py-3">
      {rows.map((row, i) => (
        <m.div
          key={row.label}
          initial={{ opacity: 0, y: 6 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: i * 0.06 }}
          className="flex items-center justify-between gap-4 border-b border-border/60 py-3.5 last:border-b-0"
        >
          <div className="min-w-0">
            <div className="truncate text-[14px]">{row.label}</div>
            <div className="mt-0.5 font-mono text-[11.5px] text-muted-foreground">
              {row.size} · {row.auto ? t("landing.panel.backup_auto") : t("landing.panel.backup_by_hand")}
            </div>
          </div>
          <span className="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-border px-3 py-1 text-[12px]">
            <RotateCcw className="size-3.5" />
            {t("landing.panel.backup_restore")}
          </span>
        </m.div>
      ))}
    </div>
  );
}

function ScheduleTab() {
  const t = useT();
  const rows = [
    { title: t("landing.panel.task_restart"), when: "0 6 * * *", on: true },
    { title: t("landing.panel.task_backup"), when: "0 */6 * * *", on: true },
    { title: t("landing.panel.task_warn"), when: "55 5 * * *", on: true },
    { title: t("landing.panel.task_update"), when: "0 4 * * 1", on: false },
  ];
  return (
    <div className="h-full overflow-hidden px-5 py-3">
      {rows.map((row, i) => (
        <m.div
          key={row.title}
          initial={{ opacity: 0, y: 6 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: i * 0.06 }}
          className="flex items-center justify-between gap-4 border-b border-border/60 py-3.5 last:border-b-0"
        >
          <div className="min-w-0">
            <div className="truncate text-[14px]">{row.title}</div>
            <div className="mt-0.5 font-mono text-[11.5px] text-muted-foreground">{row.when}</div>
          </div>
          <span
            className={cn(
              "flex h-5 w-9 shrink-0 items-center rounded-full p-0.5 transition-colors",
              row.on ? "bg-foreground" : "bg-muted"
            )}
          >
            <m.span
              initial={false}
              animate={{ x: row.on ? 16 : 0 }}
              className={cn("size-4 rounded-full", row.on ? "bg-background" : "bg-muted-foreground/60")}
            />
          </span>
        </m.div>
      ))}
    </div>
  );
}
