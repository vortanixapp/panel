"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  cancelAdminJob,
  cleanupAdminJobs,
  deleteAdminJob,
  fetchAdminJobs,
  retryAdminFailedJobs,
  retryAdminJob,
  type AdminJob,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag, type TranslateFn } from "@/lib/i18n";

const STATUS_OPTIONS = [
  { value: "all", labelKey: "common.all" },
  { value: "active", labelKey: "admin.jobs.filter.active" },
  { value: "pending", labelKey: "admin.jobs.filter.pending" },
  { value: "running", labelKey: "admin.jobs.filter.running" },
  { value: "stuck", labelKey: "admin.jobs.filter.stuck" },
  { value: "failed", labelKey: "admin.jobs.filter.failed" },
  { value: "completed", labelKey: "admin.jobs.filter.completed" },
  { value: "cancelled", labelKey: "admin.jobs.filter.cancelled" },
];

const STATUS_LABEL_KEYS: Record<string, string> = {
  pending: "admin.jobs.status.pending",
  running: "admin.jobs.status.running",
  completed: "admin.jobs.status.completed",
  failed: "admin.jobs.status.failed",
  cancelled: "admin.jobs.status.cancelled",
};

function statusVariant(
  job: AdminJob
): "default" | "secondary" | "destructive" | "outline" {
  if (job.status === "failed") return "destructive";
  if (job.stuck) return "destructive";
  if (job.status === "running") return "default";
  if (job.status === "completed") return "secondary";
  return "outline";
}

function humanAge(seconds: number, t: TranslateFn): string {
  if (seconds < 60)
    return `${Math.max(seconds, 0)} ${t("admin.jobs.unit_seconds")}`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} ${t("admin.jobs.unit_minutes")}`;
  const hours = Math.floor(minutes / 60);
  if (hours < 48) return `${hours} ${t("admin.jobs.unit_hours")}`;
  return `${Math.floor(hours / 24)} ${t("admin.jobs.unit_days")}`;
}

function fmtDateTime(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString(localeTag());
}

function jobTarget(job: AdminJob): string {
  if (job.server_name && job.node_name) {
    return `${job.server_name} · ${job.node_name}`;
  }
  if (job.server_name) return job.server_name;
  if (job.node_name) return job.node_name;
  const action = job.payload?.["action"];
  return typeof action === "string" ? action : "—";
}

function SummaryCard({
  label,
  value,
  hint,
  tone = "normal",
}: {
  label: string;
  value: string | number;
  hint?: string;
  tone?: "normal" | "warn" | "bad";
}) {
  const toneClass =
    tone === "bad"
      ? "text-destructive"
      : tone === "warn"
        ? "text-amber-600 dark:text-amber-500"
        : "";
  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className={`mt-1 text-2xl font-semibold ${toneClass}`}>{value}</div>
      {hint ? (
        <div className="mt-1 text-xs text-muted-foreground">{hint}</div>
      ) : null}
    </div>
  );
}

export function JobsPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const [status, setStatus] = useState("active");
  const [type, setType] = useState("all");
  const [search, setSearch] = useState("");
  const [appliedSearch, setAppliedSearch] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);
  const [cleanupDays, setCleanupDays] = useState("30");

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ["admin-jobs", status, type, appliedSearch],
    queryFn: () =>
      fetchAdminJobs({
        status,
        type,
        search: appliedSearch || undefined,
        limit: 200,
      }),
    refetchInterval: 10_000,
  });

  const jobs = data?.jobs ?? [];
  const summary = data?.summary;
  const types = data?.types ?? [];
  const stuckMinutes = data?.stuck_minutes ?? 30;

  const invalidate = () => qc.invalidateQueries({ queryKey: ["admin-jobs"] });

  const retryMut = useMutation({
    mutationFn: (id: string) => retryAdminJob(id),
    onSuccess: () => {
      toast.success(t("admin.jobs.requeued"));
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message || t("admin.jobs.retry_failed")),
  });

  const cancelMut = useMutation({
    mutationFn: (id: string) => cancelAdminJob(id),
    onSuccess: () => {
      toast.success(t("admin.jobs.cancelled"));
      invalidate();
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.jobs.cancel_failed")),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteAdminJob(id),
    onSuccess: () => {
      toast.success(t("admin.jobs.record_deleted"));
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message || t("common.delete_failed")),
  });

  const retryAllMut = useMutation({
    mutationFn: () => retryAdminFailedJobs(type),
    onSuccess: (res) => {
      toast.success(
        res.count > 0
          ? t("admin.jobs.requeued_count", { count: res.count })
          : t("admin.jobs.no_failed")
      );
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message || t("admin.jobs.retry_failed")),
  });

  const cleanupMut = useMutation({
    mutationFn: () => cleanupAdminJobs(Number(cleanupDays) || 30),
    onSuccess: (res) => {
      toast.success(t("admin.jobs.cleaned_count", { count: res.count }));
      invalidate();
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.jobs.cleanup_failed")),
  });

  return (
    <PageShell variant="admin">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">
          {t("admin.jobs.title")}
        </h1>
        <p className="text-sm text-muted-foreground">
          {t("admin.jobs.subtitle", { minutes: stuckMinutes })}
        </p>
      </div>

      {summary ? (
        <div className="mb-6 grid grid-cols-2 gap-3 md:grid-cols-5">
          <SummaryCard
            label={t("admin.jobs.summary.pending")}
            value={summary.pending}
            hint={
              summary.oldest_pending_seconds > 0
                ? t("admin.jobs.summary.oldest", {
                    age: humanAge(summary.oldest_pending_seconds, t),
                  })
                : undefined
            }
            tone={summary.oldest_pending_seconds > 900 ? "warn" : "normal"}
          />
          <SummaryCard
            label={t("admin.jobs.summary.running")}
            value={summary.running}
          />
          <SummaryCard
            label={t("admin.jobs.summary.stuck")}
            value={summary.stuck}
            tone={summary.stuck > 0 ? "bad" : "normal"}
          />
          <SummaryCard
            label={t("admin.jobs.summary.failed")}
            value={summary.failed}
            tone={summary.failed > 0 ? "bad" : "normal"}
          />
          <SummaryCard
            label={t("admin.jobs.summary.completed")}
            value={summary.completed}
          />
        </div>
      ) : null}

      <div className="mb-4 flex flex-wrap items-end gap-3">
        <div className="space-y-1">
          <Label>{t("common.status")}</Label>
          <Select value={status} onValueChange={setStatus}>
            <SelectTrigger className="w-56">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {STATUS_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {t(o.labelKey)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label>{t("common.type")}</Label>
          <Select value={type} onValueChange={setType}>
            <SelectTrigger className="w-56">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("admin.jobs.all_types")}</SelectItem>
              {types.map((item) => (
                <SelectItem key={item.type} value={item.type}>
                  {item.label} ({item.total})
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label>{t("admin.jobs.search_label")}</Label>
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") setAppliedSearch(search.trim());
            }}
            placeholder={t("admin.jobs.search_placeholder")}
            className="w-72"
          />
        </div>
        <Button variant="outline" onClick={() => setAppliedSearch(search.trim())}>
          {t("common.search")}
        </Button>
        <div className="ml-auto flex flex-wrap items-end gap-2">
          <Button
            variant="outline"
            onClick={() => retryAllMut.mutate()}
            disabled={retryAllMut.isPending || !summary?.failed}
          >
            {t("admin.jobs.retry_failed_all")}
            {type !== "all" ? t("admin.jobs.this_type") : ""}
          </Button>
          <div className="space-y-1">
            <Label className="text-xs">{t("admin.jobs.cleanup_days")}</Label>
            <div className="flex gap-2">
              <Input
                type="number"
                min="1"
                value={cleanupDays}
                onChange={(e) => setCleanupDays(e.target.value)}
                className="w-24"
              />
              <Button
                variant="outline"
                onClick={() => cleanupMut.mutate()}
                disabled={cleanupMut.isPending}
              >
                {t("admin.jobs.cleanup")}
              </Button>
            </div>
          </div>
        </div>
      </div>

      <p className="mb-4 text-xs text-muted-foreground">
        {t("admin.jobs.cleanup_hint")}
      </p>

      <div className="rounded-lg border bg-card">
        {isLoading ? (
          <div className="space-y-3 p-6">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : jobs.length === 0 ? (
          <div className="p-10 text-center text-sm text-muted-foreground">
            {t("admin.jobs.empty")}
          </div>
        ) : (
          <div className={isFetching ? "divide-y opacity-60" : "divide-y"}>
            {jobs.map((job) => {
              const canRetry =
                job.status === "failed" ||
                job.status === "cancelled" ||
                job.stuck;
              const canCancel = job.status === "pending" || job.stuck;
              const canDelete =
                job.status === "completed" ||
                job.status === "failed" ||
                job.status === "cancelled";
              return (
                <div key={job.id} className="p-4">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-medium">{job.type_label}</span>
                        <Badge variant={statusVariant(job)}>
                          {job.stuck
                            ? t("admin.jobs.status.stuck")
                            : STATUS_LABEL_KEYS[job.status]
                              ? t(STATUS_LABEL_KEYS[job.status])
                              : job.status}
                        </Badge>
                        {job.attempts > 0 ? (
                          <Badge variant="outline">
                            {t("admin.jobs.attempts", { count: job.attempts })}
                          </Badge>
                        ) : null}
                      </div>
                      <div className="mt-1 text-sm text-muted-foreground">
                        {jobTarget(job)}
                      </div>
                      {job.error ? (
                        <div className="mt-1 text-sm text-destructive">
                          {job.error}
                        </div>
                      ) : null}
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                      <div className="text-right text-xs text-muted-foreground">
                        <div>
                          {t("admin.jobs.created_ago", {
                            age: humanAge(job.age_seconds, t),
                          })}
                        </div>
                        <div>{fmtDateTime(job.created_at)}</div>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() =>
                          setExpanded(expanded === job.id ? null : job.id)
                        }
                      >
                        {expanded === job.id
                          ? t("common.less")
                          : t("common.details")}
                      </Button>
                      {canRetry ? (
                        <Button
                          size="sm"
                          onClick={() => retryMut.mutate(job.id)}
                          disabled={retryMut.isPending}
                        >
                          {t("admin.jobs.retry")}
                        </Button>
                      ) : null}
                      {canCancel ? (
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => cancelMut.mutate(job.id)}
                          disabled={cancelMut.isPending}
                        >
                          {t("admin.jobs.cancel_job")}
                        </Button>
                      ) : null}
                      {canDelete ? (
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => deleteMut.mutate(job.id)}
                          disabled={deleteMut.isPending}
                        >
                          {t("common.delete")}
                        </Button>
                      ) : null}
                    </div>
                  </div>

                  {expanded === job.id ? (
                    <div className="mt-3 grid gap-3 md:grid-cols-2">
                      <div>
                        <div className="mb-1 text-xs text-muted-foreground">
                          {t("admin.jobs.job_id")}
                        </div>
                        <code className="text-xs break-all">{job.id}</code>
                        <div className="mt-2 mb-1 text-xs text-muted-foreground">
                          Payload
                        </div>
                        <pre className="max-h-64 overflow-auto rounded bg-muted p-2 text-xs">
                          {JSON.stringify(job.payload, null, 2)}
                        </pre>
                      </div>
                      <div>
                        <div className="mb-1 text-xs text-muted-foreground">
                          {t("admin.jobs.updated_at", {
                            when: fmtDateTime(job.updated_at),
                            age: humanAge(job.idle_seconds, t),
                          })}
                        </div>
                        <div className="mt-2 mb-1 text-xs text-muted-foreground">
                          {t("admin.jobs.result")}
                        </div>
                        <pre className="max-h-64 overflow-auto rounded bg-muted p-2 text-xs">
                          {JSON.stringify(job.result, null, 2)}
                        </pre>
                      </div>
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {data && data.total > jobs.length ? (
        <p className="mt-3 text-xs text-muted-foreground">
          {t("admin.jobs.truncated", {
            shown: jobs.length,
            total: data.total,
          })}
        </p>
      ) : null}
    </PageShell>
  );
}
