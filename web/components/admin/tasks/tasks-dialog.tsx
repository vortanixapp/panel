"use client";

import Link from "next/link";
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  ArrowLeft,
  ArrowUpRight,
  BellRing,
  CheckCircle2,
  ClipboardList,
  History,
  Inbox,
  Loader2,
  RotateCw,
  XCircle,
} from "lucide-react";

import { confirmAction } from "@/components/action-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import {
  adminTasksKey,
  useAdminTaskDetail,
  useAdminTaskEvents,
  useAdminTaskFeed,
} from "@/hooks/use-admin-tasks";
import { useT } from "@/hooks/use-translations";
import { cancelAdminJob, retryAdminJob, type AdminTask, type AdminTaskEvent } from "@/lib/api";
import type { TranslateFn } from "@/lib/i18n";
import { cn } from "@/lib/utils";

import {
  formatTaskDuration,
  formatTaskTime,
  stringifyJson,
  taskStatusLabel,
  taskStatusVariant,
} from "./task-utils";

type Tab = "tasks" | "history" | "events";
type TaskRef = Pick<AdminTask, "source" | "id">;

const TAB_ICONS = {
  tasks: ClipboardList,
  history: History,
  events: BellRing,
} as const;

function StatusBadge({ task, t }: { task: AdminTask; t: TranslateFn }) {
  return (
    <Badge variant={taskStatusVariant(task)} className="gap-1">
      {task.status === "running" && !task.stuck ? (
        <Loader2 className="animate-spin" />
      ) : null}
      {taskStatusLabel(task, t)}
    </Badge>
  );
}

function EmptyBlock({ text, icon: Icon = Inbox }: { text: string; icon?: typeof Inbox }) {
  return (
    <div className="flex h-full min-h-48 flex-col items-center justify-center gap-3 text-muted-foreground">
      <Icon className="size-10 opacity-60" />
      <span className="text-sm">{text}</span>
    </div>
  );
}

function ListSkeleton() {
  return (
    <div className="flex flex-col gap-2 p-4">
      {[0, 1, 2].map((i) => (
        <Skeleton key={i} className="h-16 w-full" />
      ))}
    </div>
  );
}

function TaskRow({
  task,
  t,
  onOpen,
  onCancel,
  onRetry,
  busy,
}: {
  task: AdminTask;
  t: TranslateFn;
  onOpen: () => void;
  onCancel: () => void;
  onRetry: () => void;
  busy: boolean;
}) {
  const active = task.status === "queued" || task.status === "running";
  const canCancel = task.source === "job" && (task.status === "queued" || task.stuck);
  const canRetry = task.source === "job" && (task.status === "failed" || task.status === "cancelled");
  return (
    <div className="flex flex-col gap-2 rounded-lg border bg-card px-3.5 py-3">
      <div className="flex items-start justify-between gap-3">
        <button type="button" onClick={onOpen} className="min-w-0 flex-1 text-start">
          <div className="flex flex-wrap items-center gap-2">
            <span className="truncate text-sm font-medium">{task.label}</span>
            <StatusBadge task={task} t={t} />
          </div>
          <div className="mt-1 truncate text-xs text-muted-foreground">
            {task.target || t("admin.tasks.no_target")}
          </div>
        </button>
        <div className="flex shrink-0 items-center gap-1.5">
          {canCancel && (
            <Button size="sm" variant="outline" disabled={busy} onClick={onCancel}>
              {t("common.cancel")}
            </Button>
          )}
          {canRetry && (
            <Button size="sm" variant="outline" disabled={busy} onClick={onRetry}>
              <RotateCw className="size-3.5" />
              {t("admin.tasks.retry")}
            </Button>
          )}
        </div>
      </div>
      {active && task.percent !== undefined && (
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
          <div
            className="h-full rounded-full bg-primary transition-[width] duration-500"
            style={{ width: `${Math.max(2, Math.min(100, task.percent))}%` }}
          />
        </div>
      )}
      {active && task.status === "running" && task.percent === undefined && (
        <div className="relative h-1.5 w-full overflow-hidden rounded-full bg-muted">
          <div className="absolute inset-y-0 w-1/3 animate-pulse rounded-full bg-primary/70" />
        </div>
      )}
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <span className="min-w-0 truncate">
          {task.error || task.message || formatTaskTime(task.created_at)}
        </span>
        <span className="shrink-0">{formatTaskDuration(task.duration_sec, t)}</span>
      </div>
    </div>
  );
}

function JsonBlock({ title, value }: { title: string; value: unknown }) {
  const text = stringifyJson(value);
  if (!text) return null;
  return (
    <details className="rounded-lg border bg-card">
      <summary className="cursor-pointer select-none px-3.5 py-2.5 text-sm font-medium">
        {title}
      </summary>
      <pre className="max-h-56 overflow-auto border-t bg-muted/40 p-3 font-mono text-xs leading-relaxed whitespace-pre-wrap break-all">
        {text}
      </pre>
    </details>
  );
}

function TaskDetail({
  taskRef,
  t,
  onBack,
}: {
  taskRef: TaskRef;
  t: TranslateFn;
  onBack: () => void;
}) {
  const detail = useAdminTaskDetail(taskRef);
  const task = detail.data?.task;
  const log = detail.data?.log ?? [];

  const rows: [string, string][] = task
    ? [
        [t("admin.tasks.detail.target"), task.target || "—"],
        [t("admin.tasks.detail.created"), formatTaskTime(task.created_at)],
        [t("admin.tasks.detail.finished"), formatTaskTime(task.finished_at)],
        [t("admin.tasks.detail.duration"), formatTaskDuration(task.duration_sec, t)],
        ...(task.attempts ? ([[t("admin.tasks.detail.attempts"), String(task.attempts)]] as [string, string][]) : []),
        [t("admin.tasks.detail.id"), task.id],
      ]
    : [];

  return (
    <div className="flex h-full flex-col gap-3 overflow-y-auto p-4">
      <div className="flex items-center gap-2">
        <Button size="sm" variant="ghost" onClick={onBack}>
          <ArrowLeft className="size-4" />
          {t("admin.tasks.back")}
        </Button>
      </div>
      {detail.isLoading || !task ? (
        <ListSkeleton />
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-base font-semibold">{task.label}</h3>
            <StatusBadge task={task} t={t} />
          </div>
          <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 rounded-lg border bg-card px-3.5 py-3 text-sm">
            {rows.map(([k, v]) => (
              <div key={k} className="contents">
                <dt className="text-muted-foreground">{k}</dt>
                <dd className="min-w-0 truncate text-end font-mono text-xs leading-5">{v}</dd>
              </div>
            ))}
          </dl>
          {task.error && (
            <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-3.5 py-3 text-sm text-destructive">
              <div className="mb-1 flex items-center gap-1.5 font-medium">
                <XCircle className="size-4" />
                {t("admin.tasks.detail.error")}
              </div>
              <pre className="font-mono text-xs whitespace-pre-wrap break-words">{task.error}</pre>
            </div>
          )}
          <div className="flex flex-col gap-1.5">
            <div className="text-sm font-medium">{t("admin.tasks.detail.log")}</div>
            {log.length === 0 ? (
              <div className="rounded-lg border bg-muted/30 px-3.5 py-3 text-xs text-muted-foreground">
                {task.message || t("admin.tasks.detail.log_empty")}
              </div>
            ) : (
              <pre className="max-h-72 overflow-auto rounded-lg border bg-muted/40 p-3 font-mono text-xs leading-relaxed whitespace-pre-wrap break-all">
                {log.join("\n")}
              </pre>
            )}
          </div>
          <JsonBlock title={t("admin.tasks.detail.input")} value={task.input} />
          <JsonBlock title={t("admin.tasks.detail.output")} value={task.output} />
        </>
      )}
    </div>
  );
}

function eventSummary(event: AdminTaskEvent): string {
  const data = event.data ?? {};
  const message = data.message ?? data.error ?? data.reason ?? data.status;
  if (typeof message === "string" && message) return message;
  const text = stringifyJson(data);
  return text ? text.replace(/\s+/g, " ").slice(0, 160) : "";
}

function EventsList({ t }: { t: TranslateFn }) {
  const events = useAdminTaskEvents(true);
  const items = events.data?.events ?? [];
  if (events.isLoading) return <ListSkeleton />;
  if (items.length === 0) return <EmptyBlock text={t("admin.tasks.events_empty")} />;
  return (
    <div className="flex flex-col gap-2 p-4">
      {items.map((event) => (
        <div key={event.id} className="flex items-start gap-3 rounded-lg border bg-card px-3.5 py-3">
          <span
            className={cn(
              "mt-1.5 size-2 shrink-0 rounded-full",
              event.level === "error" && "bg-destructive",
              event.level === "warn" && "bg-amber-500",
              event.level === "success" && "bg-emerald-500",
              event.level === "info" && "bg-muted-foreground"
            )}
          />
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center justify-between gap-x-3">
              <span className="truncate text-sm font-medium">{event.kind}</span>
              <span className="shrink-0 text-xs text-muted-foreground">
                {formatTaskTime(event.created_at)}
              </span>
            </div>
            <div className="truncate text-xs text-muted-foreground">
              {[event.node, eventSummary(event)].filter(Boolean).join(" · ")}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

export function AdminTasksDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<Tab>("tasks");
  const [selected, setSelected] = useState<TaskRef | null>(null);

  const active = useAdminTaskFeed("active", true, open);
  const history = useAdminTaskFeed("history", open && tab === "history", open);

  const refresh = () => void queryClient.invalidateQueries({ queryKey: adminTasksKey });

  const cancelMutation = useMutation({
    mutationFn: (id: string) => cancelAdminJob(id),
    onSuccess: () => {
      toast.success(t("admin.tasks.cancelled"));
      refresh();
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const retryMutation = useMutation({
    mutationFn: (id: string) => retryAdminJob(id),
    onSuccess: () => {
      toast.success(t("admin.tasks.retried"));
      refresh();
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const busy = cancelMutation.isPending || retryMutation.isPending;
  const activeCount = active.data?.counts.active ?? 0;

  const tabs: { id: Tab; label: string; count?: number }[] = [
    { id: "tasks", label: t("admin.tasks.tab.tasks"), count: activeCount },
    { id: "history", label: t("admin.tasks.tab.history") },
    { id: "events", label: t("admin.tasks.tab.events") },
  ];

  const askCancel = async (task: AdminTask) => {
    if (!(await confirmAction(t("admin.tasks.cancel_confirm", { name: task.label })))) return;
    cancelMutation.mutate(task.id);
  };

  const renderList = (items: AdminTask[] | undefined, loading: boolean, empty: string) => {
    if (loading) return <ListSkeleton />;
    if (!items || items.length === 0) {
      return <EmptyBlock text={empty} icon={tab === "tasks" ? CheckCircle2 : Inbox} />;
    }
    return (
      <div className="flex flex-col gap-2 p-4">
        {items.map((task) => (
          <TaskRow
            key={`${task.source}:${task.id}`}
            task={task}
            t={t}
            busy={busy}
            onOpen={() => setSelected({ source: task.source, id: task.id })}
            onCancel={() => void askCancel(task)}
            onRetry={() => retryMutation.mutate(task.id)}
          />
        ))}
      </div>
    );
  };

  let body;
  if (selected) {
    body = <TaskDetail taskRef={selected} t={t} onBack={() => setSelected(null)} />;
  } else if (tab === "tasks") {
    body = renderList(active.data?.tasks, active.isLoading, t("admin.tasks.empty_active"));
  } else if (tab === "history") {
    body = renderList(history.data?.tasks, history.isLoading, t("admin.tasks.empty_history"));
  } else {
    body = <EventsList t={t} />;
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) setSelected(null);
        onOpenChange(next);
      }}
    >
      <DialogContent
        className="h-[min(680px,88vh)] grid-rows-[auto_minmax(0,1fr)] gap-0 overflow-hidden p-0 sm:max-w-4xl"
        aria-describedby={undefined}
      >
        <DialogTitle className="sr-only">{t("admin.tasks.title")}</DialogTitle>
        <DialogDescription className="sr-only">{t("admin.tasks.description")}</DialogDescription>
        <div className="flex items-center justify-between gap-3 border-b px-5 py-3.5 pe-12">
          <div className="text-base font-semibold">{t("admin.tasks.title")}</div>
          <Button asChild variant="ghost" size="sm">
            <Link href="/admin/jobs" onClick={() => onOpenChange(false)}>
              {t("admin.tasks.open_queue")}
              <ArrowUpRight className="size-4" />
            </Link>
          </Button>
        </div>
        <div className="grid min-h-0 grid-cols-1 grid-rows-[auto_minmax(0,1fr)] sm:grid-cols-[200px_minmax(0,1fr)] sm:grid-rows-1">
          <nav className="flex gap-1 overflow-x-auto border-b p-2 sm:flex-col sm:overflow-visible sm:border-e sm:border-b-0">
            {tabs.map(({ id, label, count }) => {
              const Icon = TAB_ICONS[id];
              const current = tab === id;
              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => {
                    setTab(id);
                    setSelected(null);
                  }}
                  className={cn(
                    "flex items-center gap-2 rounded-md px-3 py-2 text-start text-sm whitespace-nowrap transition-colors",
                    current
                      ? "bg-accent font-medium text-accent-foreground"
                      : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                  )}
                >
                  <Icon className="size-4 shrink-0" />
                  <span className="truncate">
                    {label}
                    {count !== undefined ? ` (${count})` : ""}
                  </span>
                </button>
              );
            })}
          </nav>
          <div className="min-h-0 overflow-y-auto">{body}</div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
