"use client";

import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { Btn, Notice, Panel, VX_FAINT, VX_MUTED } from "@/components/vx/panel-ui";
import { fetchAgentSandbox, installAgentSandbox, type AgentRow } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { useNodeTask } from "@/hooks/use-node-task";

export type SandboxState = "present" | "missing" | "unknown";

type SandboxNode = Pick<AgentRow, "sandbox" | "is_online">;

export function sandboxState(node: SandboxNode): SandboxState {
  if (!node.sandbox) return "unknown";
  return node.sandbox.available ? "present" : "missing";
}

export type SandboxGaps = { missing: AgentRow[]; unknown: AgentRow[] };

export function sandboxGaps(nodes: AgentRow[]): SandboxGaps {
  const online = nodes.filter((node) => node.is_online);
  return {
    missing: online.filter((node) => sandboxState(node) === "missing"),
    unknown: online.filter((node) => sandboxState(node) === "unknown"),
  };
}

const BADGE_TONE: Record<SandboxState, string> = {
  present: "border-transparent bg-[var(--vx-tint)] text-[var(--vx-fg-strong)]",
  missing: "border-[rgba(232,160,60,0.35)] text-[var(--vx-warn)]",
  unknown: "border-[var(--vx-border-2)] text-[var(--vx-faint)]",
};

export function SandboxBadge({ node }: { node: SandboxNode }) {
  const t = useT();
  const state = sandboxState(node);
  return (
    <span
      data-testid="sandbox-badge"
      data-state={state}
      className={cn("inline-flex items-center rounded-full border px-2.5 py-1 text-[11.5px] whitespace-nowrap", BADGE_TONE[state])}
    >
      {t(`admin.agents.sandbox.badge_${state}`)}
    </span>
  );
}

export function SandboxInstallButton({
  nodeId,
  node,
  canWrite,
  size = "sm",
}: {
  nodeId: string;
  node: SandboxNode;
  canWrite: boolean;
  size?: "sm" | "md";
}) {
  const t = useT();
  const query = useQuery({ queryKey: queryKeys.agentSandbox(nodeId), queryFn: () => fetchAgentSandbox(nodeId) });
  const task = useNodeTask(nodeId, {
    invalidate: [queryKeys.agents, queryKeys.agentSandbox(nodeId)],
    successText: t("admin.agents.sandbox.done"),
    failureText: t("admin.agents.sandbox.failed"),
  });
  const { attach } = task;
  useEffect(() => attach(query.data?.running), [attach, query.data?.running]);

  const supported = node.sandbox?.installer === true || query.data?.supported === true;
  const disabled = !canWrite || !node.is_online || !supported || task.busy;
  return (
    <Btn
      size={size}
      tone="primary"
      disabled={disabled}
      title={!node.is_online ? t("admin.agents.sandbox.offline") : undefined}
      onClick={() => void task.run(() => installAgentSandbox(nodeId))}
    >
      {task.busy ? t("admin.agents.sandbox.installing") : t("admin.agents.sandbox.install")}
    </Btn>
  );
}

export function AgentSandboxPanel({ id, agent, canWrite }: { id: string; agent: AgentRow; canWrite: boolean }) {
  const t = useT();
  const state = sandboxState(agent);
  const query = useQuery({ queryKey: queryKeys.agentSandbox(id), queryFn: () => fetchAgentSandbox(id) });
  const last = query.data?.last;
  const output = last?.result?.output ?? "";

  return (
    <Panel
      title={t("admin.agents.sandbox.title")}
      aside={
        <div className="flex items-center gap-3">
          <SandboxBadge node={agent} />
          {state !== "present" && <SandboxInstallButton nodeId={id} node={agent} canWrite={canWrite} />}
        </div>
      }
    >
      <div className="flex flex-col gap-3">
        {state === "unknown" && <Notice tone="warn">{t("admin.agents.sandbox.unknown_hint")}</Notice>}
        {state === "missing" && (
          <Notice tone="warn">{t("admin.agents.sandbox.missing_hint", { runtime: agent.sandbox?.runtime ?? "runsc" })}</Notice>
        )}
        {state === "present" && (
          <div className={cn("text-[12.5px]", VX_MUTED)}>
            {t("admin.agents.sandbox.present_hint", {
              runtime: agent.sandbox?.runtime ?? "runsc",
              mode: agent.sandbox?.mode ?? "off",
            })}
          </div>
        )}
        {state !== "present" && <div className={cn("text-[12px]", VX_FAINT)}>{t("admin.agents.sandbox.install_hint")}</div>}
        {last?.status === "failed" && output && (
          <div>
            <div className={cn("mb-1 text-[11.5px]", VX_FAINT)}>{t("admin.agents.sandbox.last_output")}</div>
            <pre className="max-h-[220px] overflow-auto rounded-[10px] border border-[var(--vx-border-2)] p-3 font-mono text-[11.5px] whitespace-pre-wrap">
              {output}
            </pre>
          </div>
        )}
      </div>
    </Panel>
  );
}
