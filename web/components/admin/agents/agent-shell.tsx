"use client";

import Link from "next/link";
import { useState } from "react";
import { useParams, usePathname, useRouter, useSearchParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ChevronDown, RefreshCw, RotateCw } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { ConfirmDialog } from "@/components/servers/confirm-dialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { Btn, Notice, VX_CARD, VX_FAINT, VX_MUTED } from "@/components/vx/panel-ui";
import { LabelChips, StatePill, UpdateStage } from "@/components/admin/agents/agents-table";
import { AgentOverviewTab } from "@/components/admin/agents/agent-overview-tab";
import { AgentContainersTab } from "@/components/admin/agents/agent-containers-tab";
import { AgentLogsTab } from "@/components/admin/agents/agent-logs-tab";
import { AgentEventsTab } from "@/components/admin/agents/agent-events-tab";
import { AgentDiagnosticsTab } from "@/components/admin/agents/agent-diagnostics-tab";
import { AgentMaintenanceTab } from "@/components/admin/agents/agent-maintenance-tab";
import {
  agentSSHInstall,
  apiErrorCode,
  fetchAgentView,
  restartAgent,
  updateAgent,
  type AgentView,
  type AgentViewer,
} from "@/lib/api";
import { agentErrorText, formatDuration } from "@/lib/agents";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { useNodeTask } from "@/hooks/use-node-task";

const TABS = ["overview", "containers", "logs", "events", "diagnostics", "maintenance"] as const;
type AgentTab = (typeof TABS)[number];

function isTab(v: string | null): v is AgentTab {
  return v != null && (TABS as readonly string[]).includes(v);
}

export type AgentTabProps = { id: string; agent: AgentView; viewer: AgentViewer };

export function AgentShell() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const qc = useQueryClient();
  const tabParam = params.get("tab");
  const tab: AgentTab = isTab(tabParam) ? tabParam : "overview";
  const [restartOpen, setRestartOpen] = useState(false);
  const [forceRestart, setForceRestart] = useState(false);
  const [reinstallOpen, setReinstallOpen] = useState(false);

  const viewQuery = useQuery({
    queryKey: queryKeys.agent(id),
    queryFn: () => fetchAgentView(id),
    enabled: !!id,
    refetchInterval: (q) => {
      const a = q.state.data?.agent_view;
      if (!a) return 15000;
      return a.labels.includes("updating") || a.labels.includes("restarting") ? 3000 : 10000;
    },
  });

  const invalidateAll = [queryKeys.agent(id), queryKeys.agents, queryKeys.agentEvents(id, "")];

  const restartTask = useNodeTask(id, {
    invalidate: invalidateAll,
    successText: t("admin.agents.restart.done"),
    failureText: t("admin.agents.restart.failed"),
    onDone: (task) => {
      if (task.error_code === "agent_busy") {
        setForceRestart(true);
        setRestartOpen(true);
      }
    },
  });
  const reinstallTask = useNodeTask(id, {
    invalidate: invalidateAll,
    successText: t("admin.agents.reinstall.done"),
    failureText: t("admin.agents.reinstall.failed"),
  });

  const updateMutation = useMutation({
    mutationFn: () => updateAgent(id),
    onSuccess: () => {
      toast.success(t("admin.agents.update.sent"));
      for (const key of invalidateAll) void qc.invalidateQueries({ queryKey: key });
    },
    onError: (e) => toast.error(agentErrorText(apiErrorCode(e), e instanceof Error ? e.message : "")),
  });

  function selectTab(next: AgentTab) {
    const q = new URLSearchParams(params.toString());
    if (next === "overview") q.delete("tab");
    else q.set("tab", next);
    const qs = q.toString();
    router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
  }

  if (viewQuery.isError) {
    return (
      <PageShell variant="admin">
        <Notice>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <span>{t("admin.agents.load_error")}</span>
            <Btn size="sm" onClick={() => viewQuery.refetch()}>
              {t("common.retry")}
            </Btn>
          </div>
        </Notice>
      </PageShell>
    );
  }

  if (!viewQuery.data) {
    return (
      <PageShell variant="admin">
        <div className="font-panel space-y-4">
          <Skeleton className="h-[148px] w-full rounded-[16px]" />
          <Skeleton className="h-[92px] w-full rounded-[14px]" />
          <Skeleton className="h-[420px] w-full rounded-[14px]" />
        </div>
      </PageShell>
    );
  }

  const { agent_view: agent, viewer } = viewQuery.data;
  const facts = agent.facts;
  const online = agent.state === "online";
  const subtitle = [
    agent.version ? `v${agent.version.replace(/^v/, "")}` : null,
    online && agent.uptime_sec != null ? t("admin.agents.uptime_value", { value: formatDuration(agent.uptime_sec) }) : null,
    facts?.os,
    facts?.docker?.version ? `Docker ${facts.docker.version}` : null,
  ]
    .filter(Boolean)
    .join(" · ");
  const canRestart = viewer.can_write && (online || agent.ssh);
  const updating = agent.labels.includes("updating");
  const restarting = agent.labels.includes("restarting") || restartTask.busy;

  const tabProps: AgentTabProps = { id, agent, viewer };

  return (
    <PageShell variant="admin">
      <div className="font-panel flex flex-col gap-[18px]">
        <div className={cn("rounded-[16px]", VX_CARD)}>
          <div className="flex flex-wrap items-start justify-between gap-5 px-[22px] py-5">
            <div className="min-w-[260px]">
              <div className={cn("font-mono text-[10.5px] font-medium tracking-[0.1em] uppercase", VX_FAINT)}>
                <Link href="/admin/daemons" className="hover:text-[var(--vx-fg)]">
                  {t("admin.agents.eyebrow")}
                </Link>
                {agent.code ? ` · ${agent.code}` : ""}
              </div>
              <div className="mt-2 flex flex-wrap items-center gap-3">
                <h1 className="m-0 text-[25px] font-medium tracking-[-0.015em]">{agent.name}</h1>
                <StatePill state={agent.state} />
                <LabelChips labels={agent.labels} />
              </div>
              {subtitle && <div className={cn("mt-[7px] text-[12.5px]", VX_MUTED)}>{subtitle}</div>}
              <div className="mt-1">
                <UpdateStage row={agent} />
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              {agent.outdated && viewer.can_write && (
                <Btn
                  tone="primary"
                  disabled={updateMutation.isPending || updating}
                  onClick={() => updateMutation.mutate()}
                >
                  <RefreshCw className="h-3.5 w-3.5" />
                  {t("admin.agents.update.to", { version: agent.target_version })}
                </Btn>
              )}
              {canRestart && (
                <Btn disabled={restarting} onClick={() => setRestartOpen(true)}>
                  <RotateCw className={cn("h-3.5 w-3.5", restarting && "animate-spin")} />
                  {restarting ? t("admin.agents.restart.running") : t("admin.agents.restart.button")}
                </Btn>
              )}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button size="sm" variant="outline">
                    {t("admin.agents.menu.more")}
                    <ChevronDown className="ml-2 h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-60">
                  <DropdownMenuItem onClick={() => selectTab("diagnostics")}>
                    {t("admin.agents.tab.diagnostics")}
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild>
                    <Link href={`/admin/locations/${id}`}>{t("admin.agents.menu.location")}</Link>
                  </DropdownMenuItem>
                  {viewer.can_write && agent.ssh && (
                    <>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem variant="destructive" onClick={() => setReinstallOpen(true)}>
                        {t("admin.agents.reinstall.menu")}
                      </DropdownMenuItem>
                    </>
                  )}
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>

          <div className="border-t border-[var(--vx-border)] px-3.5 py-[9px]">
            <div className="flex items-center gap-1 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
              {TABS.map((key) => {
                const active = key === tab;
                return (
                  <button
                    key={key}
                    type="button"
                    onClick={() => selectTab(key)}
                    className={cn(
                      "inline-flex h-[30px] shrink-0 items-center rounded-[8px] border px-3.5 text-[12.5px] font-medium transition-colors",
                      active
                        ? "border-[var(--vx-border-strong)] bg-[var(--vx-tint)] text-[var(--vx-fg-strong)]"
                        : "border-transparent text-[var(--vx-muted)] hover:text-[var(--vx-fg)]"
                    )}
                  >
                    {t(`admin.agents.tab.${key}`)}
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        {!online && (
          <Notice>
            {agent.state === "never_connected" ? t("admin.agents.notice.never") : t("admin.agents.notice.offline")}
            {agent.ssh ? ` ${t("admin.agents.notice.offline_ssh")}` : ` ${t("admin.agents.notice.offline_no_ssh")}`}
          </Notice>
        )}
        {agent.labels.includes("legacy") && <Notice tone="warn">{t("admin.agents.notice.legacy")}</Notice>}
        {updating && (
          <Notice tone="warn">
            {t("admin.agents.notice.updating", { version: agent.update?.target ?? agent.target_version })}
          </Notice>
        )}
        {agent.update?.status === "failed" && (
          <Notice>{t("admin.agents.notice.update_failed", { error: agent.update.error ?? "" })}</Notice>
        )}

        {tab === "overview" && <AgentOverviewTab {...tabProps} />}
        {tab === "containers" && <AgentContainersTab {...tabProps} />}
        {tab === "logs" && <AgentLogsTab {...tabProps} />}
        {tab === "events" && <AgentEventsTab {...tabProps} />}
        {tab === "diagnostics" && <AgentDiagnosticsTab {...tabProps} />}
        {tab === "maintenance" && <AgentMaintenanceTab {...tabProps} />}
      </div>

      <ConfirmDialog
        open={restartOpen}
        onOpenChange={(open) => {
          setRestartOpen(open);
          if (!open) setForceRestart(false);
        }}
        tone={forceRestart ? "danger" : "primary"}
        title={t("admin.agents.restart.title")}
        description={
          <div className="grid gap-2">
            <span>{online && agent.proto >= 2 ? t("admin.agents.restart.body_relay") : t("admin.agents.restart.body_ssh")}</span>
            {forceRestart && <span className="text-[var(--vx-warn)]">{t("admin.agents.restart.busy")}</span>}
          </div>
        }
        confirmLabel={forceRestart ? t("admin.agents.restart.force") : t("admin.agents.restart.button")}
        pending={restartTask.busy}
        onConfirm={() => {
          const force = forceRestart;
          setRestartOpen(false);
          setForceRestart(false);
          void restartTask.run(() => restartAgent(id, force));
        }}
      />

      <ConfirmDialog
        open={reinstallOpen}
        onOpenChange={setReinstallOpen}
        title={t("admin.agents.reinstall.title")}
        description={t("admin.agents.reinstall.body")}
        confirmLabel={t("admin.agents.reinstall.confirm")}
        requirePhrase={agent.code || agent.name}
        pending={reinstallTask.busy}
        onConfirm={() => {
          setReinstallOpen(false);
          void reinstallTask.run(() => agentSSHInstall(id));
        }}
      />
    </PageShell>
  );
}
