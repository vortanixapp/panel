"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Btn,
  EmptyState,
  Field,
  Panel,
  VX_FAINT,
  VX_INPUT,
  VX_INSET,
  VX_MUTED,
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
  createServerBackup,
  deleteRemoteBackup,
  deleteServerBackup,
  fetchRemoteBackups,
  fetchServerBackupSchedule,
  fetchServerBackups,
  restoreRemoteBackup,
  restoreServerBackup,
  saveServerBackupSchedule,
  type BackupEntry,
  type RemoteBackup,
  type ServerBackupSchedule,
} from "@/lib/api";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

type ScheduleForm = Omit<ServerBackupSchedule, "last_run_at" | "last_error">;

const WEEKDAY_INDEXES = [0, 1, 2, 3, 4, 5, 6];

function formatSize(bytes: number) {
  if (!bytes || bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let size = bytes;
  let i = 0;
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024;
    i += 1;
  }
  return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

export function ServerCopiesTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [backupName, setBackupName] = useState("");
  const [restoreTarget, setRestoreTarget] = useState<string | null>(null);
  const [schedule, setSchedule] = useState<ScheduleForm | null>(null);

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: ["server-backups", id],
    queryFn: () => fetchServerBackups(id),
    enabled: !!id,
  });

  const scheduleQuery = useQuery({
    queryKey: ["server-backup-schedule", id],
    queryFn: () => fetchServerBackupSchedule(id),
    enabled: !!id,
  });

  useEffect(() => {
    const s = scheduleQuery.data;
    if (!s) return;
    setSchedule({
      enabled: s.enabled,
      frequency: s.frequency,
      hour_utc: s.hour_utc,
      day_of_week: s.day_of_week,
      keep_count: s.keep_count,
    });
  }, [scheduleQuery.data]);

  const scheduleMutation = useMutation({
    mutationFn: (form: ScheduleForm) => saveServerBackupSchedule(id, form),
    onSuccess: () => {
      toast.success(t("servers.copies.schedule_saved"));
      void queryClient.invalidateQueries({ queryKey: ["server-backup-schedule", id] });
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.save_failed")),
  });

  const createMutation = useMutation({
    mutationFn: (name: string) => createServerBackup(id, name),
    onSuccess: () => {
      toast.success(t("servers.copies.created"));
      setCreateOpen(false);
      setBackupName("");
      void queryClient.invalidateQueries({ queryKey: ["server-backups", id] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.copies.create_error")
      ),
  });

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteServerBackup(id, name),
    onSuccess: () => {
      toast.success(t("servers.copies.deleted"));
      void queryClient.invalidateQueries({ queryKey: ["server-backups", id] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.copies.delete_error")
      ),
  });

  const restoreMutation = useMutation({
    mutationFn: (name: string) => restoreServerBackup(id, name),
    onSuccess: () => {
      toast.success(t("servers.copies.restored"));
      setRestoreTarget(null);
      void queryClient.invalidateQueries({ queryKey: ["server-backups", id] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.copies.restore_error")
      ),
  });

  const remoteQuery = useQuery({
    queryKey: ["server-backups-remote", id],
    queryFn: () => fetchRemoteBackups(id),
    enabled: !!id,
    refetchInterval: 15_000,
  });
  const remote = remoteQuery.data?.backups ?? [];
  const remoteAvailable = remoteQuery.data?.available ?? false;

  const remoteRestoreMutation = useMutation({
    mutationFn: (backupId: string) => restoreRemoteBackup(id, backupId, true),
    onSuccess: () => {
      toast.success(t("servers.copies.remote_restoring"));
      void queryClient.invalidateQueries({ queryKey: ["server-backups-remote", id] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.copies.restore_error")
      ),
  });

  const remoteDeleteMutation = useMutation({
    mutationFn: (backupId: string) => deleteRemoteBackup(id, backupId),
    onSuccess: () => {
      toast.success(t("servers.copies.remote_deleted"));
      void queryClient.invalidateQueries({ queryKey: ["server-backups-remote", id] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.copies.delete_error")
      ),
  });

  const entries = ((data?.entries ?? data?.backups ?? []) as BackupEntry[]).filter(
    (e) => !e.is_dir && e.name
  );

  if (isLoading) return <Skeleton className="h-[320px] w-full rounded-[14px]" />;

  return (
    <>
      <Panel
        title={t("servers.copies.auto_title")}
        bodyClassName="flex flex-col gap-3 px-[18px] py-3.5"
      >
        {!schedule ? (
          <Skeleton className="h-[92px] w-full rounded-[10px]" />
        ) : (
          <>
            <label className="flex items-center gap-2.5">
              <input
                type="checkbox"
                checked={schedule.enabled}
                onChange={(e) =>
                  setSchedule({ ...schedule, enabled: e.target.checked })
                }
                className="size-4 accent-[var(--vx-fg-strong)]"
              />
              <span className="text-[12.5px]">
                {t("servers.copies.auto_enable")}
              </span>
            </label>

            <div className="flex flex-wrap items-end gap-3">
              <Field label={t("servers.copies.frequency")}>
                <select
                  className={VX_INPUT}
                  value={schedule.frequency}
                  disabled={!schedule.enabled}
                  onChange={(e) =>
                    setSchedule({
                      ...schedule,
                      frequency: e.target.value as ScheduleForm["frequency"],
                    })
                  }
                >
                  <option value="daily">{t("servers.copies.daily")}</option>
                  <option value="weekly">{t("servers.copies.weekly")}</option>
                </select>
              </Field>

              {schedule.frequency === "weekly" && (
                <Field label={t("servers.copies.weekday_label")}>
                  <select
                    className={VX_INPUT}
                    value={schedule.day_of_week}
                    disabled={!schedule.enabled}
                    onChange={(e) =>
                      setSchedule({ ...schedule, day_of_week: Number(e.target.value) })
                    }
                  >
                    {WEEKDAY_INDEXES.map((idx) => (
                      <option key={idx} value={idx}>
                        {t(`servers.copies.weekday.${idx}`)}
                      </option>
                    ))}
                  </select>
                </Field>
              )}

              <Field label={t("servers.copies.hour_utc")}>
                <select
                  className={VX_INPUT}
                  value={schedule.hour_utc}
                  disabled={!schedule.enabled}
                  onChange={(e) =>
                    setSchedule({ ...schedule, hour_utc: Number(e.target.value) })
                  }
                >
                  {Array.from({ length: 24 }, (_, h) => (
                    <option key={h} value={h}>
                      {String(h).padStart(2, "0")}:00
                    </option>
                  ))}
                </select>
              </Field>

              <Field label={t("servers.copies.keep_count")}>
                <select
                  className={VX_INPUT}
                  value={schedule.keep_count}
                  disabled={!schedule.enabled}
                  onChange={(e) =>
                    setSchedule({ ...schedule, keep_count: Number(e.target.value) })
                  }
                >
                  {[3, 5, 7, 14, 30].map((n) => (
                    <option key={n} value={n}>
                      {n}
                    </option>
                  ))}
                </select>
              </Field>

              <Btn
                size="sm"
                tone="primary"
                onClick={() => schedule && scheduleMutation.mutate(schedule)}
                disabled={scheduleMutation.isPending}
              >
                {scheduleMutation.isPending
                  ? t("common.saving")
                  : t("common.save")}
              </Btn>
            </div>

            <p className={cn("text-[11.5px] leading-[1.6]", VX_FAINT)}>
              {t("servers.copies.auto_hint_prefix")}{" "}
              <span className="font-mono">auto-…</span>{" "}
              {t("servers.copies.auto_hint_suffix")}
            </p>

            {scheduleQuery.data?.last_error && (
              <p className="text-[11.5px] leading-[1.6] text-[var(--vx-danger)]">
                {t("servers.copies.last_error", {
                  error: scheduleQuery.data.last_error,
                })}
              </p>
            )}
            {scheduleQuery.data?.last_run_at && !scheduleQuery.data.last_error && (
              <p className={cn("text-[11.5px]", VX_FAINT)}>
                {t("servers.copies.last_run", {
                  date: new Date(scheduleQuery.data.last_run_at).toLocaleString(
                    localeTag()
                  ),
                })}
              </p>
            )}
          </>
        )}
      </Panel>

      <Panel
        title={t("servers.copies.list_title")}
        aside={
          <div className="flex gap-2">
            <Btn size="sm" onClick={() => refetch()} disabled={isFetching}>
              {isFetching ? t("common.updating") : t("common.refresh")}
            </Btn>
            <Btn size="sm" tone="primary" onClick={() => setCreateOpen(true)}>
              {t("servers.copies.create")}
            </Btn>
          </div>
        }
        bodyClassName="flex flex-col gap-2 px-[18px] py-3.5"
      >
        {entries.length === 0 ? (
          <div className="py-10 text-center">
            <div className={cn("text-[12.5px]", VX_MUTED)}>
              {t("servers.copies.empty")}
            </div>
            <Btn size="sm" tone="primary" className="mt-3.5" onClick={() => setCreateOpen(true)}>
              {t("servers.copies.create_first")}
            </Btn>
          </div>
        ) : (
          entries.map((entry) => (
            <div
              key={entry.name}
              className={cn(
                "flex flex-wrap items-center justify-between gap-3 rounded-[11px] px-[15px] py-3",
                VX_INSET
              )}
            >
              <div className="min-w-0">
                <div className="truncate font-mono text-[12.5px]">{entry.name}</div>
                <div className={cn("mt-[3px] text-[11.5px]", VX_FAINT)}>
                  {formatSize(entry.size)}
                </div>
              </div>
              <div className="flex shrink-0 gap-2">
                <Btn
                  size="sm"
                  disabled={restoreMutation.isPending}
                  onClick={() => setRestoreTarget(entry.name)}
                >
                  {t("servers.copies.restore")}
                </Btn>
                <Btn
                  size="sm"
                  className="border-[rgba(224,122,122,0.3)] bg-transparent text-[var(--vx-danger)]"
                  disabled={deleteMutation.isPending}
                  onClick={() => {
                    if (
                      !confirm(
                        t("servers.copies.delete_confirm", { name: entry.name })
                      )
                    )
                      return;
                    deleteMutation.mutate(entry.name);
                  }}
                >
                  {t("common.delete")}
                </Btn>
              </div>
            </div>
          ))
        )}
      </Panel>

      {(remoteAvailable || remote.length > 0) && (
        <Panel
          title={t("servers.copies.remote_title")}
          bodyClassName="flex flex-col gap-2 px-[18px] py-3.5"
        >
          <p className={cn("text-[11.5px] leading-[1.6]", VX_FAINT)}>
            {t("servers.copies.remote_hint")}
          </p>
          {remote.length === 0 ? (
            <div className={cn("py-6 text-center text-[12.5px]", VX_MUTED)}>
              {t("servers.copies.remote_empty")}
            </div>
          ) : (
            remote.map((backup: RemoteBackup) => (
              <div
                key={backup.id}
                className={cn(
                  "flex flex-wrap items-center justify-between gap-3 rounded-[11px] px-[15px] py-3",
                  VX_INSET
                )}
              >
                <div className="min-w-0">
                  <div className="truncate font-mono text-[12.5px]">
                    {backup.filename}
                  </div>
                  <div className={cn("mt-[3px] text-[11.5px]", VX_FAINT)}>
                    {t(`servers.copies.remote_status.${backup.status}`)}
                    {backup.remote_size > 0
                      ? ` · ${formatSize(backup.remote_size)}`
                      : ""}
                    {backup.uploaded_at
                      ? ` · ${new Date(backup.uploaded_at).toLocaleString(localeTag())}`
                      : ""}
                  </div>
                  {backup.error && (
                    <div className="mt-[3px] text-[11.5px] text-[var(--vx-danger)]">
                      {backup.error}
                    </div>
                  )}
                </div>
                {backup.status === "uploaded" && (
                  <div className="flex shrink-0 gap-2">
                    <Btn
                      size="sm"
                      disabled={remoteRestoreMutation.isPending}
                      onClick={() => {
                        if (
                          !confirm(
                            t("servers.copies.remote_restore_confirm", {
                              name: backup.filename,
                            })
                          )
                        )
                          return;
                        remoteRestoreMutation.mutate(backup.id);
                      }}
                    >
                      {t("servers.copies.restore")}
                    </Btn>
                    <Btn
                      size="sm"
                      className="border-[rgba(224,122,122,0.3)] bg-transparent text-[var(--vx-danger)]"
                      disabled={remoteDeleteMutation.isPending}
                      onClick={() => {
                        if (
                          !confirm(
                            t("servers.copies.remote_delete_confirm", {
                              name: backup.filename,
                            })
                          )
                        )
                          return;
                        remoteDeleteMutation.mutate(backup.id);
                      }}
                    >
                      {t("common.delete")}
                    </Btn>
                  </div>
                )}
              </div>
            ))
          )}
        </Panel>
      )}

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("servers.copies.new_title")}</DialogTitle>
          </DialogHeader>
          <Field label={t("servers.copies.file_name")}>
            <input
              className={VX_INPUT}
              value={backupName}
              onChange={(e) => setBackupName(e.target.value)}
              placeholder="manual-backup"
            />
          </Field>
          <p className={cn("text-xs", VX_FAINT)}>
            {t("servers.copies.archive_hint")}
          </p>
          <DialogFooter>
            <Btn onClick={() => setCreateOpen(false)}>{t("common.cancel")}</Btn>
            <Btn
              tone="primary"
              onClick={() => createMutation.mutate(backupName)}
              disabled={createMutation.isPending}
            >
              {createMutation.isPending
                ? t("common.creating")
                : t("common.create")}
            </Btn>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={restoreTarget !== null} onOpenChange={(open) => !open && setRestoreTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("servers.copies.restore_title")}</DialogTitle>
          </DialogHeader>
          <p className={cn("text-[12.5px] leading-[1.65]", VX_MUTED)}>
            {t("servers.copies.restore_desc_prefix")}{" "}
            <span className="font-mono text-[var(--vx-fg)]">{restoreTarget}</span>
            {t("servers.copies.restore_desc_suffix")}
          </p>
          <DialogFooter>
            <Btn onClick={() => setRestoreTarget(null)}>{t("common.cancel")}</Btn>
            <Btn
              tone="danger"
              onClick={() => restoreTarget && restoreMutation.mutate(restoreTarget)}
              disabled={restoreMutation.isPending || !restoreTarget}
            >
              {restoreMutation.isPending
                ? t("servers.copies.restoring")
                : t("servers.copies.restore")}
            </Btn>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
