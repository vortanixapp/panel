"use client";

import { useEffect, useRef, useState } from "react";
import {
  Check,
  ChevronDown,
  CircleArrowUp,
  CircleCheck,
  CircleX,
  CloudOff,
  Download,
  ExternalLink,
  Loader2,
  RotateCw,
  SquareTerminal,
  TriangleAlert,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import type { AdminUpdates, UpdaterJob } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";
import {
  formatClock,
  formatDateTime,
  formatDay,
  formatRelative,
  isNewerVersion,
  reachedStep,
  splitLogLine,
  STEPS,
  validDate,
  type AgentsSummary,
} from "./update-utils";

type Tone = "primary" | "success" | "warning" | "danger" | "muted";

const TONE: Record<Tone, { icon: string; glow: string }> = {
  primary: { icon: "bg-primary/10 text-primary", glow: "from-primary/[0.07]" },
  success: {
    icon: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
    glow: "from-emerald-500/[0.07]",
  },
  warning: {
    icon: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
    glow: "from-amber-500/[0.07]",
  },
  danger: { icon: "bg-rose-500/10 text-rose-600 dark:text-rose-400", glow: "from-rose-500/[0.07]" },
  muted: { icon: "bg-muted text-muted-foreground", glow: "from-muted/40" },
};

function useNow(active: boolean) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!active) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [active]);
  return now;
}

function jobDuration(job: UpdaterJob, now: number): number {
  const start = validDate(job.started_at)?.getTime();
  if (!start) return 0;
  const end = job.state === "running" ? now : (validDate(job.finished_at)?.getTime() ?? now);
  return end - start;
}

export function PanelStatus({
  data,
  liveVersion,
  job,
  restarting,
  starting,
  reloadVersion,
  agents,
  agentsAuto,
  onInstall,
  onManual,
}: {
  data: AdminUpdates;
  liveVersion: string;
  job: UpdaterJob | null;
  restarting: boolean;
  starting: boolean;
  reloadVersion: string | null;
  agents: AgentsSummary | null;
  agentsAuto: boolean | null;
  onInstall: (version: string) => void;
  onManual: () => void;
}) {
  const t = useT();
  const running = restarting || job?.state === "running";
  const failed = !running && job?.state === "failed" && isNewerVersion(job.target, liveVersion);
  const updater = data.updater;
  const canInstall = updater?.available === true;
  const latest = data.latest_version ?? "";
  const newCount = data.releases?.length ?? 0;

  let tone: Tone;
  let icon: React.ReactNode;
  let title: React.ReactNode;
  let hint: React.ReactNode = null;
  let actions: React.ReactNode = null;

  if (running && job) {
    tone = "primary";
    icon = <Loader2 className="size-5 animate-spin" />;
    title = t("admin.updates.hero.installing", { version: job.target });
    hint = restarting ? t("admin.updates.hero.restarting") : `${job.from || liveVersion} → ${job.target}`;
  } else if (reloadVersion) {
    tone = "success";
    icon = <CircleCheck className="size-5" />;
    title = t("admin.updates.hero.updated", { version: reloadVersion });
    hint = t("admin.updates.hero.updated_hint");
    actions = (
      <Button onClick={() => window.location.reload()}>
        <RotateCw />
        {t("admin.updates.hero.reload")}
      </Button>
    );
  } else if (failed && job) {
    tone = "danger";
    icon = <CircleX className="size-5" />;
    title = t("admin.updates.hero.failed", { version: job.target });
    hint = job.error || t("admin.updates.hero.failed_hint");
    actions = canInstall ? (
      <Button onClick={() => onInstall(job.target)} disabled={starting}>
        {starting ? <Loader2 className="animate-spin" /> : <RotateCw />}
        {t("admin.updates.hero.retry")}
      </Button>
    ) : null;
  } else if (data.checks_disabled) {
    tone = "muted";
    icon = <CloudOff className="size-5" />;
    title = t("admin.updates.hero.disabled");
    hint = t("admin.updates.hero.disabled_hint");
  } else if (data.error) {
    tone = "warning";
    icon = <TriangleAlert className="size-5" />;
    title = t("admin.updates.hero.error");
    hint = data.error;
  } else if (data.update_available && latest) {
    tone = "primary";
    icon = <CircleArrowUp className="size-5" />;
    title = t("admin.updates.hero.available", { version: latest });
    hint = (
      <>
        <span className="font-mono">
          {liveVersion} → {latest}
        </span>
        {data.published_at && <span> · {t("admin.updates.hero.published", { date: formatDay(data.published_at) })}</span>}
        {newCount > 1 && <span> · {t("admin.updates.hero.releases", { count: newCount })}</span>}
      </>
    );
    actions = (
      <>
        {canInstall ? (
          <Button onClick={() => onInstall(latest)} disabled={starting}>
            {starting ? <Loader2 className="animate-spin" /> : <Download />}
            {t("admin.updates.hero.install", { version: latest })}
          </Button>
        ) : (
          <Button variant="outline" onClick={onManual}>
            <SquareTerminal />
            {t("admin.updates.hero.manual")}
          </Button>
        )}
        {data.release_url && (
          <Button asChild variant="ghost">
            <a href={data.release_url} target="_blank" rel="noreferrer noopener">
              {t("admin.updates.hero.github")}
              <ExternalLink />
            </a>
          </Button>
        )}
      </>
    );
  } else {
    tone = "success";
    icon = <CircleCheck className="size-5" />;
    title = t("admin.updates.hero.latest");
    hint = t("admin.updates.hero.latest_hint", {
      version: liveVersion || "—",
      when: formatRelative(data.checked_at),
    });
  }

  const showLast = job && !running && !failed && !reloadVersion;

  return (
    <section className="relative overflow-hidden rounded-2xl border bg-card">
      <div className={cn("pointer-events-none absolute inset-x-0 top-0 h-28 bg-gradient-to-b to-transparent", TONE[tone].glow)} />
      <div className="relative flex flex-wrap items-start gap-4 px-5 py-5 sm:px-7 sm:py-6">
        <span className={cn("grid size-11 shrink-0 place-items-center rounded-xl", TONE[tone].icon)}>{icon}</span>
        <div className="min-w-0 flex-1 basis-60">
          <h2 className="text-[18px] leading-tight font-semibold tracking-tight">{title}</h2>
          {hint && <p className="mt-1.5 text-[13.5px] break-words text-muted-foreground">{hint}</p>}
          {!canInstall && data.update_available && !running && !reloadVersion && updater?.reason && (
            <p className="mt-1.5 text-[12.5px] text-amber-600 dark:text-amber-500">
              {t("admin.updates.hero.no_updater", { reason: updater.reason })}
            </p>
          )}
        </div>
        {actions && <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto">{actions}</div>}
      </div>

      {(running || failed) && job && (
        <div className="relative border-t px-5 py-5 sm:px-7">
          <JobProgress job={job} restarting={restarting} defaultOpen={failed === true} />
        </div>
      )}

      <dl className="relative grid grid-cols-2 border-t lg:grid-cols-4">
        <Fact label={t("admin.updates.fact.installed")}>
          <span className="font-mono">{liveVersion || "—"}</span>
        </Fact>
        <Fact label={t("admin.updates.fact.agents")}>
          <a href="#agents" className="hover:underline">
            {agents === null
              ? "—"
              : agents.total === 0
                ? t("admin.updates.fact.agents_none")
                : agents.current === agents.total
                  ? t("admin.updates.fact.agents_all", { count: agents.total })
                  : t("admin.updates.fact.agents_some", { current: agents.current, total: agents.total })}
          </a>
        </Fact>
        <Fact label={t("admin.updates.fact.auto")}>
          {data.auto.enabled && agentsAuto
            ? t("admin.updates.fact.auto_both")
            : data.auto.enabled
              ? t("admin.updates.fact.auto_panel")
              : agentsAuto
                ? t("admin.updates.fact.auto_agents")
                : t("admin.updates.fact.auto_off")}
        </Fact>
        <Fact label={t("admin.updates.fact.mode")}>
          {canInstall ? t(`admin.updates.mode.${updater.mode}`) : t("admin.updates.mode.manual")}
        </Fact>
      </dl>

      {showLast && job && <LastInstall job={job} />}
    </section>
  );
}

function Fact({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="min-w-0 border-border px-4 py-3.5 sm:px-7 [&:nth-child(2n)]:border-s lg:[&:not(:first-child)]:border-s max-lg:[&:nth-child(n+3)]:border-t">
      <dt className="text-[12px] text-muted-foreground">{label}</dt>
      <dd className="mt-1 text-[13.5px] leading-snug font-medium break-words">{children}</dd>
    </div>
  );
}

function LastInstall({ job }: { job: UpdaterJob }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const ok = job.state === "succeeded";
  return (
    <div className="relative border-t px-5 py-3 sm:px-7">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[12.5px] text-muted-foreground">
        <span className="font-medium text-foreground">{t("admin.updates.last.title")}</span>
        <span className="font-mono">
          {job.from || "?"} → {job.target}
        </span>
        <span>{formatDateTime(job.finished_at || job.started_at)}</span>
        <span className={cn(ok ? "text-emerald-600 dark:text-emerald-400" : "text-rose-600 dark:text-rose-400")}>
          {ok ? t("admin.updates.last.succeeded") : t("admin.updates.last.failed")}
        </span>
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="ms-auto inline-flex items-center gap-1 hover:text-foreground"
          aria-expanded={open}
        >
          {open ? t("admin.updates.log.hide") : t("admin.updates.log.show")}
          <ChevronDown className={cn("size-3.5 transition-transform", open && "rotate-180")} />
        </button>
      </div>
      {open && <JobLog lines={job.log} className="mt-3" />}
    </div>
  );
}

function JobProgress({
  job,
  restarting,
  defaultOpen,
}: {
  job: UpdaterJob;
  restarting: boolean;
  defaultOpen: boolean;
}) {
  const t = useT();
  const running = restarting || job.state === "running";
  const now = useNow(running);
  const [open, setOpen] = useState(defaultOpen);
  const reached = reachedStep(job.log);
  const at = Math.max(restarting ? Math.max(reached, 2) : reached, 0);
  const lastLine = job.log.length ? splitLogLine(job.log[job.log.length - 1]).text : "";

  const stepState = (index: number): "done" | "active" | "failed" | "todo" => {
    if (job.state === "succeeded" && !restarting) return "done";
    if (index < at) return "done";
    if (index === at) return job.state === "failed" && !running ? "failed" : "active";
    return "todo";
  };

  return (
    <div className="space-y-4">
      <ol className="grid gap-3 sm:grid-cols-4 sm:gap-0">
        {STEPS.map((step, index) => {
          const state = stepState(index);
          return (
            <li key={step} className="relative flex items-center gap-2.5 sm:flex-col sm:items-start sm:gap-2 sm:pe-4">
              {index < STEPS.length - 1 && (
                <span
                  className={cn(
                    "absolute top-[13px] right-0 left-8 hidden h-px sm:block",
                    state === "done" ? "bg-emerald-500/60" : "bg-border"
                  )}
                />
              )}
              <span
                className={cn(
                  "relative z-[1] grid size-[26px] shrink-0 place-items-center rounded-full border text-[12px] font-semibold",
                  state === "done" && "border-emerald-500 bg-emerald-500 text-white",
                  state === "active" && "border-primary bg-card text-primary",
                  state === "failed" && "border-rose-500 bg-rose-500 text-white",
                  state === "todo" && "border-border bg-card text-muted-foreground"
                )}
              >
                {state === "done" ? (
                  <Check className="size-3.5" />
                ) : state === "active" ? (
                  <Loader2 className="size-3.5 animate-spin" />
                ) : state === "failed" ? (
                  <X className="size-3.5" />
                ) : (
                  index + 1
                )}
              </span>
              <span
                className={cn(
                  "text-[13px]",
                  state === "todo" ? "text-muted-foreground" : "font-medium",
                  state === "failed" && "text-rose-600 dark:text-rose-400"
                )}
              >
                {t(`admin.updates.step.${step}`)}
              </span>
            </li>
          );
        })}
      </ol>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[12.5px] text-muted-foreground">
        <span className="tabular-nums">
          {running
            ? t("admin.updates.elapsed", { time: formatClock(jobDuration(job, now)) })
            : t("admin.updates.duration", { time: formatClock(jobDuration(job, now)) })}
        </span>
        {running && lastLine && <span className="min-w-0 flex-1 truncate">{lastLine}</span>}
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="ms-auto inline-flex items-center gap-1 hover:text-foreground"
          aria-expanded={open}
        >
          {open ? t("admin.updates.log.hide") : t("admin.updates.log.show")}
          <ChevronDown className={cn("size-3.5 transition-transform", open && "rotate-180")} />
        </button>
      </div>

      {open && <JobLog lines={job.log} />}
    </div>
  );
}

function JobLog({ lines, className }: { lines: string[]; className?: string }) {
  const t = useT();
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [lines.length]);
  return (
    <div
      ref={ref}
      className={cn(
        "max-h-[340px] overflow-auto rounded-xl border bg-[var(--vx-surface-2)] px-4 py-3 font-mono text-[12px] leading-relaxed",
        className
      )}
    >
      {lines.length === 0 ? (
        <span className="text-muted-foreground">{t("admin.updates.log.empty")}</span>
      ) : (
        lines.map((line, index) => {
          const { time, text } = splitLogLine(line);
          return (
            <div key={index} className="flex gap-3 whitespace-pre-wrap break-words">
              {time && <span className="shrink-0 text-muted-foreground/70 tabular-nums">{time}</span>}
              <span className="min-w-0">{text}</span>
            </div>
          );
        })
      )}
    </div>
  );
}
