"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/servers/confirm-dialog";
import { Bar, Btn, EmptyState, Notice, Panel, VX_FAINT, VX_MUTED } from "@/components/vx/panel-ui";
import { CleanupDialog } from "@/components/admin/agents/cleanup-dialog";
import type { AgentTabProps } from "@/components/admin/agents/agent-shell";
import {
  fetchAgentContainers,
  fetchAgentDisk,
  powerServer,
  startAgentDisk,
  type AgentContainer,
  type AgentDiskReport,
  type PanelServerRef,
} from "@/lib/api";
import { TONE_CLASS, agentErrorText, formatBytes, formatDateTime, formatMB } from "@/lib/agents";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { useNodeTask } from "@/hooks/use-node-task";

function DiskPanel({ id, agent, viewer }: AgentTabProps) {
  const t = useT();
  const [cleanupOpen, setCleanupOpen] = useState(false);
  const diskQuery = useQuery({ queryKey: queryKeys.agentDisk(id), queryFn: () => fetchAgentDisk(id) });
  const task = useNodeTask<{ report: AgentDiskReport }>(id, {
    invalidate: [queryKeys.agentDisk(id)],
    silent: true,
  });
  const { attach } = task;
  useEffect(() => attach(diskQuery.data?.running), [attach, diskQuery.data?.running]);
  const report = diskQuery.data?.last?.status === "done" ? diskQuery.data.last.result?.report : undefined;
  const lastError = diskQuery.data?.last?.status !== "done" ? diskQuery.data?.last?.error : undefined;
  const canRun = agent.state === "online" && agent.caps.includes("disk_usage");
  const progress = task.task?.progress;
  const usedPct = report ? report.used_percent : null;
  const d = report?.docker;

  return (
    <Panel
      title={t("admin.agents.disk.title")}
      aside={
        <div className="flex flex-wrap gap-2">
          {viewer.can_write && (
            <Btn
              size="sm"
              disabled={!agent.caps.includes("cleanup") || agent.state !== "online"}
              onClick={() => setCleanupOpen(true)}
            >
              {t("admin.agents.cleanup.button")}
            </Btn>
          )}
          {viewer.can_write && (
            <Btn size="sm" disabled={!canRun || task.busy} onClick={() => void task.run(() => startAgentDisk(id))}>
              {task.busy
                ? progress?.total
                  ? t("admin.agents.disk.progress", { done: progress.done ?? 0, total: progress.total })
                  : t("admin.agents.disk.running")
                : t("admin.agents.disk.recalc")}
            </Btn>
          )}
        </div>
      }
    >
      {diskQuery.isLoading ? (
        <Skeleton className="h-[120px] w-full rounded-[10px]" />
      ) : !report ? (
        <div className={cn("text-[12.5px]", VX_MUTED)}>
          {lastError ? agentErrorText(diskQuery.data?.last?.error_code ?? "", lastError) : canRun ? t("admin.agents.disk.never") : t("admin.agents.disk.unavailable")}
        </div>
      ) : (
        <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
          <div className="grid gap-3">
            <div>
              <div className="flex justify-between text-[12.5px]">
                <span className={VX_MUTED}>{report.data_dir}</span>
                <span className="font-mono">
                  {report.total_mb > 0
                    ? t("admin.agents.tile.disk_free", { free: formatMB(report.free_mb), total: formatMB(report.total_mb) })
                    : "—"}
                </span>
              </div>
              <Bar pct={usedPct ?? 0} className="mt-2" barClassName={usedPct != null && usedPct > 90 ? "bg-[var(--vx-danger)]" : undefined} />
            </div>
            <div className={cn("grid grid-cols-2 gap-2 text-[12px]", VX_MUTED)}>
              <span>{t("admin.agents.disk.inodes", { value: Math.round(report.inodes_percent) })}</span>
              <span>{report.quota ? t("admin.agents.disk.quota_on") : t("admin.agents.disk.quota_off")}</span>
              <span>{t("admin.agents.disk.plugin_cache", { value: formatBytes(report.plugin_cache_bytes) })}</span>
              <span>
                {t("admin.agents.disk.measured", {
                  date: formatDateTime(diskQuery.data?.last?.finished_at),
                })}
              </span>
            </div>
            {d ? (
              <div className="grid gap-1 text-[12px]">
                {(
                  [
                    ["images", d.images_bytes, d.images_reclaimable_bytes],
                    ["containers", d.containers_bytes, d.containers_reclaimable_bytes],
                    ["volumes", d.volumes_bytes, d.volumes_reclaimable_bytes],
                    ["build_cache", d.build_cache_bytes, d.build_cache_reclaimable_bytes],
                  ] as const
                ).map(([key, total, free]) => (
                  <div key={key} className="flex justify-between gap-3 border-b border-[var(--vx-elevated)] py-1 last:border-b-0">
                    <span className={VX_MUTED}>{t(`admin.agents.disk.docker.${key}`)}</span>
                    <span className="font-mono">
                      {formatBytes(total)}
                      <span className={VX_FAINT}> · {t("admin.agents.disk.reclaimable", { value: formatBytes(free) })}</span>
                    </span>
                  </div>
                ))}
              </div>
            ) : report.docker_error ? (
              <div className="text-[12px] text-[var(--vx-danger)]">{report.docker_error}</div>
            ) : null}
          </div>
          <div>
            <div className={cn("mb-2 text-[11.5px]", VX_FAINT)}>{t("admin.agents.disk.by_server")}</div>
            {report.servers.length === 0 ? (
              <EmptyState>{t("admin.agents.disk.no_servers")}</EmptyState>
            ) : (
              <div className="max-h-[240px] overflow-auto">
                {report.servers.map((s) => (
                  <div key={s.server_id} className="flex items-center justify-between gap-3 border-b border-[var(--vx-elevated)] py-1.5 text-[12px] last:border-b-0">
                    {s.known ? (
                      <Link href={`/admin/servers/${s.server_id}`} className="truncate font-mono hover:underline">
                        {s.server_id.slice(0, 8)}
                      </Link>
                    ) : (
                      <span className="truncate font-mono text-[var(--vx-warn)]" title={t("admin.agents.disk.orphan_hint")}>
                        {s.server_id.slice(0, 8)} · {t("admin.agents.disk.orphan")}
                      </span>
                    )}
                    <span className="font-mono">{formatMB(s.used_mb)}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
      <CleanupDialog id={id} open={cleanupOpen} onOpenChange={setCleanupOpen} />
    </Panel>
  );
}

function stateTone(c: AgentContainer) {
  if (c.state === "running") return c.health === "unhealthy" ? "warn" : "ok";
  if (c.state === "restarting") return "warn";
  if (c.state === "exited" && c.exit_code !== 0) return "danger";
  return "muted";
}

function ContainerRow({
  c,
  canPower,
  onPower,
}: {
  c: AgentContainer;
  canPower: boolean;
  onPower: (server: PanelServerRef, action: "start" | "stop") => void;
}) {
  const t = useT();
  const running = c.state === "running" || c.state === "restarting";
  return (
    <tr className="border-b border-[var(--vx-elevated)] last:border-b-0">
      <td className="px-3 py-2.5" data-cell="full">
        {c.server ? (
          <Link href={`/admin/servers/${c.server.id}`} className="block truncate font-medium hover:underline">
            {c.server.name}
          </Link>
        ) : (
          <span className="block truncate font-medium">{c.name}</span>
        )}
        <span className={cn("block truncate font-mono text-[11px]", VX_FAINT)}>
          {c.server ? c.name : c.image}
        </span>
      </td>
      <td className="px-3 py-2.5" data-label={t("admin.agents.containers.col.state")}>
        <span className="inline-flex flex-wrap items-center justify-end gap-1 md:justify-start">
          <span className={cn("rounded-full border px-2 py-px text-[11px]", TONE_CLASS[stateTone(c)])}>
            {c.state}
            {c.state === "exited" ? ` (${c.exit_code})` : ""}
          </span>
          {c.status_mismatch && (
            <span className={cn("rounded-full border px-1.5 py-px text-[10.5px]", TONE_CLASS.warn)} title={t("admin.agents.containers.mismatch_hint", { status: c.server?.status ?? "" })}>
              {t("admin.agents.containers.mismatch")}
            </span>
          )}
          {c.abandoned && (
            <span className={cn("rounded-full border px-1.5 py-px text-[10.5px]", TONE_CLASS.danger)}>
              {t("admin.agents.containers.abandoned")}
            </span>
          )}
        </span>
      </td>
      <td className="px-3 py-2.5 font-mono text-[12px]" data-label={t("admin.agents.containers.col.load")}>
        {running ? `${c.cpu_pct.toFixed(1)}% · ${formatMB(c.mem_used_mb)}` : "—"}
      </td>
      <td className="px-3 py-2.5 font-mono text-[12px]" data-label={t("admin.agents.containers.col.restarts")}>
        <span className={cn(c.restarts > 3 && "text-[var(--vx-warn)]")}>{c.restarts}</span>
      </td>
      <td className="px-3 py-2.5 font-mono text-[11.5px]" data-label={t("admin.agents.containers.col.ports")}>
        {c.ports.length ? c.ports.slice(0, 3).join(", ") + (c.ports.length > 3 ? ` +${c.ports.length - 3}` : "") : "—"}
      </td>
      <td className="px-3 py-2.5 text-right" data-cell="actions">
        {c.server && canPower && (
          <Btn size="sm" tone={running ? "danger" : "default"} onClick={() => onPower(c.server!, running ? "stop" : "start")}>
            {running ? t("admin.agents.containers.stop") : t("admin.agents.containers.start")}
          </Btn>
        )}
      </td>
    </tr>
  );
}

function Group({
  title,
  items,
  canPower,
  onPower,
}: {
  title: string;
  items: AgentContainer[];
  canPower: boolean;
  onPower: (server: PanelServerRef, action: "start" | "stop") => void;
}) {
  const t = useT();
  if (!items.length) return null;
  return (
    <Panel title={`${title} · ${items.length}`} flush>
      <div className="vx-tbl-wrap overflow-x-auto px-[6px] pb-2 md:px-0">
        <table className="vx-tbl vx-tbl-flat w-full min-w-[720px] text-[12.5px]">
          <thead className="border-b border-[var(--vx-border)]">
            <tr className={cn("text-left text-[10.5px] tracking-[0.08em] uppercase", VX_FAINT)}>
              <th className="px-3 py-2 font-normal">{t("admin.agents.containers.col.name")}</th>
              <th className="px-3 py-2 font-normal">{t("admin.agents.containers.col.state")}</th>
              <th className="px-3 py-2 font-normal">{t("admin.agents.containers.col.load")}</th>
              <th className="px-3 py-2 font-normal">{t("admin.agents.containers.col.restarts")}</th>
              <th className="px-3 py-2 font-normal">{t("admin.agents.containers.col.ports")}</th>
              <th className="w-24" />
            </tr>
          </thead>
          <tbody>
            {items.map((c) => (
              <ContainerRow key={c.id} c={c} canPower={canPower} onPower={onPower} />
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

export function AgentContainersTab(props: AgentTabProps) {
  const t = useT();
  const { id, viewer } = props;
  const qc = useQueryClient();
  const [power, setPower] = useState<{ server: PanelServerRef; action: "start" | "stop" } | null>(null);
  const listQuery = useQuery({
    queryKey: queryKeys.agentContainers(id),
    queryFn: () => fetchAgentContainers(id),
    refetchInterval: 20000,
  });
  const powerMutation = useMutation({
    mutationFn: (p: { server: PanelServerRef; action: "start" | "stop" }) => powerServer(p.server.id, p.action),
    onSuccess: () => {
      toast.success(t("admin.agents.containers.power_sent"));
      setPower(null);
      setTimeout(() => void qc.invalidateQueries({ queryKey: queryKeys.agentContainers(id) }), 2500);
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : t("common.error")),
  });

  const data = listQuery.data;
  const list = data?.containers ?? [];
  const servers = list.filter((c) => c.kind === "server" && !c.abandoned);
  const service = list.filter((c) => ["agent", "agent_previous", "mysql", "helper"].includes(c.kind));
  const abandoned = list.filter((c) => c.abandoned);
  const other = list.filter((c) => c.kind === "other");
  const onPower = (server: PanelServerRef, action: "start" | "stop") => setPower({ server, action });

  return (
    <div className="flex flex-col gap-[18px]">
      <DiskPanel {...props} />

      {listQuery.isLoading ? (
        <Skeleton className="h-[260px] w-full rounded-[14px]" />
      ) : data?.limited ? (
        <>
          <Notice tone="warn">
            {agentErrorText(data.reason ?? "", data.error ?? t("admin.agents.containers.limited"))}{" "}
            {t("admin.agents.containers.limited_tail")}
          </Notice>
          <Panel title={t("admin.agents.containers.panel_servers")}>
            {(data.servers ?? []).length === 0 ? (
              <EmptyState>{t("admin.agents.containers.no_servers")}</EmptyState>
            ) : (
              (data.servers ?? []).map((s) => (
                <div key={s.id} className="flex justify-between gap-3 border-b border-[var(--vx-elevated)] py-2 text-[12.5px] last:border-b-0">
                  <Link href={`/admin/servers/${s.id}`} className="truncate hover:underline">
                    {s.name}
                  </Link>
                  <span className={VX_MUTED}>{s.status}</span>
                </div>
              ))
            )}
          </Panel>
        </>
      ) : (
        <>
          <Group title={t("admin.agents.containers.group.servers")} items={servers} canPower={viewer.can_power} onPower={onPower} />
          {(data?.missing.length ?? 0) > 0 && (
            <Panel title={`${t("admin.agents.containers.group.missing")} · ${data!.missing.length}`}>
              <div className={cn("mb-2 text-[12px]", VX_MUTED)}>{t("admin.agents.containers.missing_hint")}</div>
              {data!.missing.map((s) => (
                <div key={s.id} className="flex justify-between gap-3 border-b border-[var(--vx-elevated)] py-2 text-[12.5px] last:border-b-0">
                  <Link href={`/admin/servers/${s.id}`} className="truncate hover:underline">
                    {s.name}
                  </Link>
                  <span className={VX_MUTED}>{s.status}</span>
                </div>
              ))}
            </Panel>
          )}
          <Group title={t("admin.agents.containers.group.abandoned")} items={abandoned} canPower={false} onPower={onPower} />
          <Group title={t("admin.agents.containers.group.service")} items={service} canPower={false} onPower={onPower} />
          <Group title={t("admin.agents.containers.group.other")} items={other} canPower={false} onPower={onPower} />
          {list.length === 0 && (
            <Panel>
              <EmptyState>{t("admin.agents.containers.empty")}</EmptyState>
            </Panel>
          )}
        </>
      )}

      <ConfirmDialog
        open={power != null}
        onOpenChange={(open) => !open && setPower(null)}
        tone={power?.action === "stop" ? "danger" : "primary"}
        title={power?.action === "stop" ? t("admin.agents.containers.stop_title") : t("admin.agents.containers.start_title")}
        description={t("admin.agents.containers.power_body", { name: power?.server.name ?? "" })}
        confirmLabel={power?.action === "stop" ? t("admin.agents.containers.stop") : t("admin.agents.containers.start")}
        pending={powerMutation.isPending}
        onConfirm={() => power && powerMutation.mutate(power)}
      />
    </div>
  );
}
