"use client";

import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, CheckCircle2, MinusCircle, XCircle } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Btn, EmptyState, Panel, VX_FAINT, VX_MUTED } from "@/components/vx/panel-ui";
import type { AgentTabProps } from "@/components/admin/agents/agent-shell";
import { fetchAgentDiagnostics, startAgentDiagnostics, type AgentCheck } from "@/lib/api";
import { formatDateTime, formatDuration, formatMB } from "@/lib/agents";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { useNodeTask } from "@/hooks/use-node-task";

const ICON = {
  ok: <CheckCircle2 className="h-4 w-4 text-[var(--vx-ok)]" />,
  warn: <AlertTriangle className="h-4 w-4 text-[var(--vx-warn)]" />,
  fail: <XCircle className="h-4 w-4 text-[var(--vx-danger)]" />,
  skip: <MinusCircle className="h-4 w-4 text-[var(--vx-faint)]" />,
};

function num(d: Record<string, unknown> | undefined, k: string): number | null {
  const v = d?.[k];
  return typeof v === "number" ? v : null;
}

function str(d: Record<string, unknown> | undefined, k: string): string {
  const v = d?.[k];
  return v == null ? "" : String(v);
}

function checkValue(c: AgentCheck): string {
  const d = c.data;
  switch (c.id) {
    case "docker":
      return str(d, "version") ? `${str(d, "version")} · ${num(d, "running") ?? 0}/${num(d, "containers") ?? 0}` : "";
    case "disk_space":
      return (num(d, "total_mb") ?? 0) > 0 ? `${formatMB(num(d, "free_mb"))} / ${formatMB(num(d, "total_mb"))}` : "";
    case "inodes":
      return num(d, "used_percent") != null ? `${Math.round(num(d, "used_percent")!)}%` : "";
    case "clock":
      return num(d, "skew_ms") != null ? `${(num(d, "skew_ms")! / 1000).toFixed(1)} s` : "";
    case "firewall":
      return str(d, "iptables");
    case "registry":
      return str(d, "image");
    case "cgroup":
      return str(d, "version") ? `v${str(d, "version")} · ${str(d, "driver")}` : "";
    case "storage_driver":
      return str(d, "driver");
    case "restart_policy":
      return str(d, "policy");
    case "reconnects":
      return `${num(d, "count") ?? 0} · ${formatDuration(num(d, "uptime_sec"))}`;
    case "quota":
      return str(d, "mount");
    case "data_dir_write":
    case "state_dir_write":
      return str(d, "path");
    case "heartbeat":
      return num(d, "age_sec") != null ? formatDuration(num(d, "age_sec")) : "";
    case "relay_rtt":
      return num(d, "rtt_ms") != null ? `${num(d, "rtt_ms")} ms` : "";
    case "agent_version":
      return str(d, "version") ? `${str(d, "version")} → ${str(d, "target")}` : "";
    case "agent_protocol":
      return num(d, "proto") != null ? `v${num(d, "proto")}` : "";
  }
  return "";
}

function CheckRow({ c }: { c: AgentCheck }) {
  const t = useT();
  const value = checkValue(c);
  const showHint = c.status === "warn" || c.status === "fail";
  return (
    <div className="flex items-start gap-3 border-b border-[var(--vx-elevated)] py-2.5 last:border-b-0">
      <span className="mt-0.5 shrink-0">{ICON[c.status]}</span>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-baseline justify-between gap-x-3">
          <span className="text-[12.5px] font-medium">{t(`admin.agents.check.${c.id}`)}</span>
          {value && <span className="truncate font-mono text-[11.5px]">{value}</span>}
        </div>
        {showHint && <div className={cn("mt-0.5 text-[12px]", VX_MUTED)}>{t(`admin.agents.check_hint.${c.id}`)}</div>}
        {c.error && <div className="mt-0.5 font-mono text-[11.5px] break-words text-[var(--vx-danger)]">{c.error}</div>}
      </div>
    </div>
  );
}

export function AgentDiagnosticsTab({ id, agent, viewer }: AgentTabProps) {
  const t = useT();
  const diagQuery = useQuery({ queryKey: queryKeys.agentDiagnostics(id), queryFn: () => fetchAgentDiagnostics(id) });
  const task = useNodeTask<{ checks: AgentCheck[] }>(id, {
    invalidate: [queryKeys.agentDiagnostics(id)],
    successText: t("admin.agents.diag.done"),
    failureText: t("admin.agents.diag.failed"),
  });
  const { attach } = task;
  useEffect(() => attach(diagQuery.data?.running), [attach, diagQuery.data?.running]);

  const last = diagQuery.data?.last;
  const checks = last?.status === "done" ? last.result?.checks ?? [] : [];
  const panelChecks = diagQuery.data?.panel_checks ?? [];
  const canRun = viewer.can_write && agent.state === "online" && agent.caps.includes("diagnostics");
  const stage = task.task?.progress?.stage;
  const counts = checks.reduce(
    (acc, c) => ({ ...acc, [c.status]: (acc[c.status] ?? 0) + 1 }),
    {} as Record<string, number>
  );

  return (
    <div className="flex flex-col gap-[18px]">
      <Panel
        title={t("admin.agents.diag.title")}
        aside={
          <div className="flex items-center gap-3">
            {last?.finished_at && (
              <span className={cn("text-[11.5px]", VX_FAINT)}>
                {t("admin.agents.diag.last_run", { date: formatDateTime(last.finished_at) })}
              </span>
            )}
            <Btn size="sm" tone="primary" disabled={!canRun || task.busy} onClick={() => void task.run(() => startAgentDiagnostics(id))}>
              {task.busy
                ? stage
                  ? t("admin.agents.diag.running_stage", { stage: t(`admin.agents.check.${stage}`) })
                  : t("admin.agents.diag.running")
                : t("admin.agents.diag.run")}
            </Btn>
          </div>
        }
      >
        {diagQuery.isLoading ? (
          <Skeleton className="h-[240px] w-full rounded-[10px]" />
        ) : checks.length === 0 ? (
          <EmptyState>
            {last?.status === "failed" || last?.status === "expired"
              ? last.error || t("admin.agents.diag.failed")
              : canRun
                ? t("admin.agents.diag.never")
                : t("admin.agents.diag.unavailable")}
          </EmptyState>
        ) : (
          <>
            <div className={cn("mb-2 flex flex-wrap gap-4 text-[12px]", VX_MUTED)}>
              <span>{t("admin.agents.diag.count_ok", { count: counts.ok ?? 0 })}</span>
              <span className="text-[var(--vx-warn)]">{t("admin.agents.diag.count_warn", { count: counts.warn ?? 0 })}</span>
              <span className="text-[var(--vx-danger)]">{t("admin.agents.diag.count_fail", { count: counts.fail ?? 0 })}</span>
            </div>
            {checks.map((c) => (
              <CheckRow key={c.id} c={c} />
            ))}
          </>
        )}
      </Panel>
      <Panel title={t("admin.agents.diag.panel_title")}>
        {panelChecks.map((c) => (
          <CheckRow key={c.id} c={c} />
        ))}
      </Panel>
    </div>
  );
}
