"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Btn,
  EmptyState,
  Field,
  Panel,
  VX_FAINT,
  VX_INPUT_MONO,
  VX_ROW_LINE,
  Toggle,
} from "@/components/vx/panel-ui";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import {
  createServerCronJob,
  deleteServerCronJob,
  fetchServerCron,
  toggleServerCronJob,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const GRID = "grid-cols-[130px_minmax(0,1fr)_70px_70px] sm:grid-cols-[150px_minmax(0,1fr)_90px_80px]";

export function ServerCronTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [schedule, setSchedule] = useState("");
  const [command, setCommand] = useState("");

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: ["server-cron", id],
    queryFn: () => fetchServerCron(id),
    enabled: !!id,
  });

  const createMutation = useMutation({
    mutationFn: () => createServerCronJob(id, schedule, command),
    onSuccess: () => {
      toast.success(t("servers.cron.added"));
      setCreateOpen(false);
      setSchedule("");
      setCommand("");
      void queryClient.invalidateQueries({ queryKey: ["server-cron", id] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.cron.create_error")
      ),
  });

  const deleteMutation = useMutation({
    mutationFn: (jobId: string) => deleteServerCronJob(id, jobId),
    onSuccess: () => {
      toast.success(t("servers.cron.deleted"));
      void queryClient.invalidateQueries({ queryKey: ["server-cron", id] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.cron.delete_error")
      ),
  });

  const toggleMutation = useMutation({
    mutationFn: ({ jobId, enabled }: { jobId: string; enabled: boolean }) =>
      toggleServerCronJob(id, jobId, enabled),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["server-cron", id] }),
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.cron.update_error")
      ),
  });

  if (isLoading) return <Skeleton className="h-[320px] w-full rounded-[14px]" />;

  const jobs = data?.jobs ?? [];

  return (
    <>
      <Panel
        title={t("servers.cron.title")}
        aside={
          <div className="flex gap-2">
            <Btn size="sm" onClick={() => refetch()} disabled={isFetching}>
              {isFetching ? t("common.updating") : t("common.refresh")}
            </Btn>
            <Btn size="sm" tone="primary" onClick={() => setCreateOpen(true)}>
              {t("common.add")}
            </Btn>
          </div>
        }
        flush
      >
        <div className="overflow-x-auto px-[18px] pt-1.5 pb-4">
          <div className="min-w-[440px]">
            <div
              className={cn(
                "grid gap-3 py-2.5 text-[10.5px] tracking-[0.08em] uppercase",
                GRID,
                VX_ROW_LINE,
                VX_FAINT
              )}
            >
              <span>{t("servers.cron.col_schedule")}</span>
              <span>{t("servers.cron.col_command")}</span>
              <span>{t("servers.cron.col_enabled")}</span>
              <span />
            </div>

            {jobs.length === 0 ? (
              <EmptyState>{t("servers.cron.empty")}</EmptyState>
            ) : (
              jobs.map((job) => (
                <div
                  key={job.id}
                  className={cn(
                    "grid items-center gap-3 py-2.5 text-[12.5px] last:border-b-0",
                    GRID,
                    VX_ROW_LINE
                  )}
                >
                  <span className="font-mono text-[12px]">{job.schedule}</span>
                  <span className="truncate font-mono text-[12px] text-[var(--vx-dim)]">
                    {job.command}
                  </span>
                  <Toggle
                    label={t("servers.cron.job_toggle")}
                    checked={job.enabled}
                    disabled={toggleMutation.isPending}
                    onChange={(enabled) => toggleMutation.mutate({ jobId: job.id, enabled })}
                  />
                  <button
                    type="button"
                    disabled={deleteMutation.isPending}
                    onClick={() => {
                      if (!confirm(t("servers.cron.delete_confirm"))) return;
                      deleteMutation.mutate(job.id);
                    }}
                    className="justify-self-end text-[11.5px] text-[var(--vx-danger)] transition-opacity hover:opacity-80 disabled:opacity-40"
                  >
                    {t("common.delete")}
                  </button>
                </div>
              ))
            )}
          </div>
        </div>
      </Panel>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("servers.cron.new_title")}</DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-3">
            <Field label={t("servers.cron.col_schedule")}>
              <input
                className={VX_INPUT_MONO}
                placeholder="0 */6 * * *"
                value={schedule}
                onChange={(e) => setSchedule(e.target.value)}
              />
            </Field>
            <Field label={t("servers.cron.col_command")}>
              <input
                className={VX_INPUT_MONO}
                placeholder="./restart.sh"
                value={command}
                onChange={(e) => setCommand(e.target.value)}
              />
            </Field>
          </div>
          <DialogFooter>
            <Btn onClick={() => setCreateOpen(false)}>{t("common.cancel")}</Btn>
            <Btn
              tone="primary"
              onClick={() => createMutation.mutate()}
              disabled={createMutation.isPending || !schedule || !command}
            >
              {createMutation.isPending ? t("common.saving") : t("common.add")}
            </Btn>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
