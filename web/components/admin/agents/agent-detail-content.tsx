"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  LocationMetricsChart,
  type MetricPoint,
} from "@/components/admin/locations/location-metrics-chart";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import {
  execAdminAgentCommand,
  fetchAdminAgent,
  fetchAdminAgentLogs,
  fetchAdminAgentServers,
  installAdminAgent,
  refreshAdminAgent,
  restartAdminAgent,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import {
  AGENT_STATE_LABEL_KEY,
  agentState,
  agentStateClasses,
} from "./daemons-page-content";
import { useT } from "@/hooks/use-translations";

type Tab = "agent" | "servers";

const SERVER_STATUS_TONE: Record<string, "emerald" | "amber" | "muted"> = {
  running: "emerald",
  online: "emerald",
  active: "emerald",
  installing: "amber",
  starting: "amber",
  pending: "amber",
};

export function AgentDetailContent() {
  const t = useT();
  const params = useParams();
  const id = String(params?.id ?? "");
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<Tab>("agent");
  const [cmd, setCmd] = useState("sudo docker ps --filter name=vortanix-agent");
  const [execOut, setExecOut] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminAgent(id),
    queryFn: () => fetchAdminAgent(id),
    enabled: !!id,
    refetchInterval: 5000,
  });

  const logsQuery = useQuery({
    queryKey: queryKeys.adminAgentLogs(id),
    queryFn: () => fetchAdminAgentLogs(id, 100),
    enabled: !!id && tab === "agent",
    refetchInterval: tab === "agent" ? 5000 : false,
  });

  const serversQuery = useQuery({
    queryKey: queryKeys.adminAgentServers(id),
    queryFn: () => fetchAdminAgentServers(id),
    enabled: !!id && tab === "servers",
  });

  const refreshMut = useMutation({
    mutationFn: () => refreshAdminAgent(id),
    onSuccess: () => {
      toast.success(t("admin.agent.refresh_started"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminAgent(id) });
    },
    onError: (e: Error) => toast.error(e.message || t("admin.agent.refresh_failed")),
  });

  const restartMut = useMutation({
    mutationFn: () => restartAdminAgent(id),
    onSuccess: () => {
      toast.success(t("admin.daemons.restart_started"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminAgent(id) });
    },
    onError: (e: Error) => toast.error(e.message || t("admin.agent.restart_failed")),
  });

  const installMut = useMutation({
    mutationFn: () => installAdminAgent(id),
    onSuccess: () => {
      toast.success(t("admin.agent.install_started"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminAgent(id) });
    },
    onError: (e: Error) => toast.error(e.message || t("admin.agent.install_failed")),
  });

  const execMut = useMutation({
    mutationFn: () => execAdminAgentCommand(id, cmd),
    onSuccess: (res) => {
      setExecOut(res.stdout ?? res.output ?? res.error ?? "");
    },
    onError: (e: Error) => setExecOut(e.message || t("admin.agent.exec_failed")),
  });

  const agent = (data?.agent ?? data?.daemon ?? null) as Record<
    string,
    unknown
  > | null;
  const location = (data?.location ?? null) as Record<string, unknown> | null;
  const agentStatus = String(agent?.status ?? "unknown");
  const containerRunning = Boolean(agent?.container_running);
  const containerFound = logsQuery.data?.container_found !== false;
  const notConfigured =
    agentStatus === "unknown" || agentStatus === "not_installed";
  const state = agentState({
    status: agentStatus,
    is_online: Boolean(agent?.is_online),
  });
  const tone = agentStateClasses(state);

  const cpuMetrics = useMemo(() => {
    const raw = data?.metrics?.agent_cpu_usage ?? [];
    return raw.map((p) => ({
      t: p.measured_at,
      v: p.value,
      measured_at: p.measured_at,
      value: p.value,
    })) as MetricPoint[];
  }, [data?.metrics?.agent_cpu_usage]);

  const ramMetrics = useMemo(() => {
    const raw = data?.metrics?.agent_ram_usage ?? [];
    return raw.map((p) => ({
      t: p.measured_at,
      v: p.value,
      measured_at: p.measured_at,
      value: p.value,
    })) as MetricPoint[];
  }, [data?.metrics?.agent_ram_usage]);

  if (isLoading) {
    return (
      <PageShell variant="admin">
        <div className="w-full space-y-5">
          <Skeleton className="h-16 w-full rounded-2xl" />
          <Skeleton className="h-96 w-full rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  const isOnline = state === "online";
  const busy =
    refreshMut.isPending || restartMut.isPending || installMut.isPending;
  const showInstallBanner =
    notConfigured || !containerFound || (!containerRunning && !isOnline);
  const logs = (logsQuery.data?.logs ?? []).join("\n");
  const locationCode = String(location?.code ?? "—");
  const servers = serversQuery.data?.servers ?? [];

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <div className="flex items-center gap-2.5 text-xs text-muted-foreground">
              <Link href="/admin/daemons" className="hover:text-foreground">
                Vortanix Agent
              </Link>
              <span>/</span>
              <span className="font-mono">{locationCode}</span>
            </div>
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              Vortanix Agent
            </h1>
            <p className="font-mono text-[13px] text-muted-foreground">
              {String(location?.name ?? "—")} ({locationCode}) —{" "}
              {String(location?.ssh_host ?? agent?.host ?? "—")}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2.5">
            <Button variant="outline" asChild className="h-[38px] text-[13px]">
              <Link href="/admin/daemons">← {t("admin.tariffs.to_list")}</Link>
            </Button>
            <Button
              variant="outline"
              className="h-[38px] text-[13px]"
              disabled={busy}
              onClick={() => refreshMut.mutate()}
            >
              {refreshMut.isPending
                ? t("common.updating")
                : t("common.refresh")}
            </Button>
            <Button
              variant="outline"
              className="h-[38px] text-[13px]"
              disabled={busy}
              onClick={() => {
                if (confirm(t("admin.agent.reinstall_confirm"))) {
                  installMut.mutate();
                }
              }}
            >
              {installMut.isPending
                ? t("admin.agent.installing")
                : t("admin.agent.reinstall")}
            </Button>
            <Button
              className="h-[38px] text-[13px]"
              disabled={busy}
              onClick={() => {
                if (confirm(t("admin.agent.restart_confirm")))
                  restartMut.mutate();
              }}
            >
              {t("common.restart")}
            </Button>
          </div>
        </div>

        {showInstallBanner && (
          <div className="flex flex-wrap items-center gap-3.5 rounded-xl border border-amber-500/40 bg-amber-500/5 px-4 py-3.5">
            <span className="size-1.5 shrink-0 rounded-full bg-amber-500" />
            <p className="flex-1 basis-[320px] text-[13px] leading-relaxed text-amber-600 dark:text-amber-200/90">
              {t("admin.agent.banner_before")}{" "}
              <span className="font-mono">vortanix-agent</span>{" "}
              {t("admin.agent.banner_after")}
            </p>
            <div className="flex gap-2">
              <Button
                size="sm"
                className="h-8 bg-amber-500 text-amber-950 hover:bg-amber-400"
                disabled={busy}
                onClick={() => installMut.mutate()}
              >
                {t("admin.agent.install")}
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="h-8 border-amber-500/40"
                disabled={busy}
                onClick={() => refreshMut.mutate()}
              >
                {t("admin.agent.sync")}
              </Button>
              <Button size="sm" variant="outline" asChild className="h-8">
                <Link href={`/admin/locations/${id}/setup`}>
                  {t("admin.agent.setup_wizard")}
                </Link>
              </Button>
            </div>
          </div>
        )}

        <div className="flex w-fit gap-1 rounded-xl border bg-card p-1">
          {(
            [
              { id: "agent" as const, label: "Agent" },
              { id: "servers" as const, label: t("common.servers") },
            ]
          ).map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => setTab(item.id)}
              className={cn(
                "h-8 rounded-lg px-3.5 text-[13px] transition-colors",
                tab === item.id
                  ? "bg-primary font-medium text-primary-foreground"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              {item.label}
            </button>
          ))}
        </div>

        {tab === "agent" ? (
          <div className="space-y-4">
            <div className="grid gap-4 lg:grid-cols-2">
              <Card title={t("common.status")}>
                <Row label={t("admin.agent.state")}>
                  <span
                    className={cn("inline-flex items-center gap-2", tone.text)}
                  >
                    <span className={cn("size-1.5 rounded-full", tone.dot)} />
                    {t(AGENT_STATE_LABEL_KEY[state])}
                  </span>
                </Row>
                <Row label={t("admin.agent.location_ssh")}>
                  <span
                    className={cn(
                      "rounded-md border px-2 py-0.5 text-[11px]",
                      agent?.node_reachable
                        ? "border-emerald-500/40 text-emerald-500"
                        : "text-muted-foreground"
                    )}
                  >
                    {agent?.node_reachable
                      ? t("admin.agent.reachable")
                      : t("admin.agent.no_data")}
                  </span>
                </Row>
                <Row label={t("admin.agent.container")} mono>
                  {String(agent?.container ?? "vortanix-agent")}
                </Row>
                <Row label={t("admin.agent.last_seen")} mono>
                  {String(agent?.last_seen_human ?? "—")}
                </Row>
              </Card>

              <Card title={t("admin.agent.system")}>
                <Row label={t("admin.agent.image_version")} mono>
                  {String(agent?.version || "—")}
                </Row>
                <Row label={t("admin.daemons.platform")} mono>
                  {String(agent?.platform || "—")}
                </Row>
                <Row label="PID" mono>
                  {String(agent?.pid ?? "—")}
                </Row>
                <Row label="Relay" mono>
                  {String(agent?.relay || "—")}
                </Row>
              </Card>
            </div>

            <div className="grid gap-4 lg:grid-cols-2">
              <Card title="CPU agent" hint={t("admin.agent.last_48h")}>
                <LocationMetricsChart title="" points={cpuMetrics} />
              </Card>
              <Card title="RAM agent" hint={t("admin.agent.last_48h")}>
                <LocationMetricsChart title="" points={ramMetrics} />
              </Card>
            </div>

            <Card
              title={t("admin.agent.logs_title")}
              hint={t("admin.agent.logs_hint")}
              action={
                <Button
                  variant="outline"
                  size="sm"
                  className="h-[30px] rounded-md text-xs"
                  disabled={!logs}
                  onClick={() => {
                    void navigator.clipboard
                      ?.writeText(logs)
                      .then(() => toast.success(t("admin.agent.logs_copied")))
                      .catch(() => toast.error(t("common.copy_failed")));
                  }}
                >
                  {t("common.copy")}
                </Button>
              }
            >
              <pre className="max-h-[260px] overflow-auto rounded-xl border bg-muted/30 px-4 py-4 font-mono text-xs leading-relaxed text-muted-foreground">
                {logs ||
                  (logsQuery.isLoading
                    ? t("admin.agent.logs_loading")
                    : t("admin.agent.logs_empty"))}
              </pre>
            </Card>

            <Card
              title={t("admin.agent.ssh_title")}
              description={t("admin.agent.ssh_hint")}
            >
              <div className="flex gap-2.5">
                <Input
                  value={cmd}
                  onChange={(e) => setCmd(e.target.value)}
                  className="h-[38px] rounded-lg font-mono text-[13px] md:text-[13px]"
                />
                <Button
                  className="h-[38px] shrink-0 px-5 text-[13px]"
                  disabled={execMut.isPending || !cmd.trim()}
                  onClick={() => execMut.mutate()}
                >
                  {execMut.isPending
                    ? t("common.processing")
                    : t("admin.agent.exec")}
                </Button>
              </div>
              {execOut && (
                <pre className="max-h-[200px] overflow-auto rounded-xl border bg-muted/30 px-4 py-4 font-mono text-xs leading-relaxed text-muted-foreground">
                  {execOut}
                </pre>
              )}
            </Card>
          </div>
        ) : (
          <div className="overflow-hidden rounded-2xl border bg-card">
            <div className="flex items-baseline justify-between gap-3.5 px-6 py-5">
              <span className="text-[15px] font-semibold">
                {t("admin.agent.servers_title")}
              </span>
              <span className="font-mono text-xs text-muted-foreground">
                {serversQuery.isLoading
                  ? t("common.loading")
                  : t("admin.users.servers_count", { count: servers.length })}
              </span>
            </div>
            {serversQuery.isLoading ? (
              <div className="px-6 pb-6">
                <Skeleton className="h-32 w-full" />
              </div>
            ) : servers.length === 0 ? (
              <div className="border-t px-6 py-10 text-center text-sm text-muted-foreground">
                {t("admin.agent.servers_empty")}
              </div>
            ) : (
              servers.map((s) => {
                const status = String(s.status ?? "—");
                const serverTone = SERVER_STATUS_TONE[status] ?? "muted";
                return (
                  <div
                    key={String(s.id)}
                    className="flex items-center justify-between gap-4 border-t px-6 py-3.5 transition-colors hover:bg-muted/30"
                  >
                    <div className="min-w-0 space-y-1">
                      <div className="truncate text-sm font-medium">
                        {String(s.name)}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {String((s.game as { name?: string })?.name ?? "—")} ·{" "}
                        {String((s.user as { email?: string })?.email ?? "—")}
                      </div>
                    </div>
                    <span
                      className={cn(
                        "inline-flex shrink-0 items-center gap-2 text-xs",
                        serverTone === "emerald" && "text-emerald-500",
                        serverTone === "amber" && "text-amber-500",
                        serverTone === "muted" && "text-muted-foreground"
                      )}
                    >
                      <span
                        className={cn(
                          "size-1.5 rounded-full",
                          serverTone === "emerald" && "bg-emerald-500",
                          serverTone === "amber" && "bg-amber-500",
                          serverTone === "muted" && "bg-muted-foreground"
                        )}
                      />
                      {status}
                    </span>
                  </div>
                );
              })
            )}
          </div>
        )}
      </div>
    </PageShell>
  );
}

function Card({
  title,
  hint,
  description,
  action,
  children,
}: {
  title: string;
  hint?: string;
  description?: string;
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-4 rounded-2xl border bg-card px-5 py-5 sm:px-6">
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <div className="space-y-1">
          <div className="text-[15px] leading-none font-semibold">{title}</div>
          {description && (
            <div className="text-xs text-muted-foreground">{description}</div>
          )}
        </div>
        {hint && !action && (
          <span className="font-mono text-xs text-muted-foreground">
            {hint}
          </span>
        )}
        {action && (
          <div className="flex items-center gap-3">
            {hint && (
              <span className="font-mono text-xs text-muted-foreground">
                {hint}
              </span>
            )}
            {action}
          </div>
        )}
      </div>
      {children}
    </section>
  );
}

function Row({
  label,
  children,
  mono,
}: {
  label: string;
  children: React.ReactNode;
  mono?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-3.5 text-[13px]">
      <span className="text-muted-foreground">{label}</span>
      <span className={cn("truncate text-right", mono && "font-mono")}>
        {children}
      </span>
    </div>
  );
}
