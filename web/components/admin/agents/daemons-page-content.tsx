"use client";

import Link from "next/link";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import {
  fetchAdminAgents,
  refreshAdminAgent,
  restartAdminAgent,
  type AdminAgentListItem,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import { localeTag, type TranslateFn } from "@/lib/i18n";

type AgentState = "online" | "offline" | "not_installed";

export function agentState(agent: {
  status?: string;
  is_online?: boolean;
}): AgentState {
  if (agent.is_online || agent.status === "online") return "online";
  if (agent.status === "not_installed" || agent.status === "unknown") {
    return "not_installed";
  }
  return "offline";
}

export const AGENT_STATE_LABEL_KEY: Record<AgentState, string> = {
  online: "admin.infra.online",
  offline: "admin.infra.offline",
  not_installed: "admin.daemons.not_installed",
};

export function agentStateClasses(state: AgentState) {
  return {
    text:
      state === "online"
        ? "text-emerald-500"
        : state === "offline"
          ? "text-rose-500"
          : "text-muted-foreground",
    dot:
      state === "online"
        ? "bg-emerald-500"
        : state === "offline"
          ? "bg-rose-500"
          : "bg-muted-foreground",
    border:
      state === "online"
        ? "border-emerald-500/40"
        : state === "offline"
          ? "border-rose-500/40"
          : "border-border",
  };
}

function uptimeText(sec: number | undefined, t: TranslateFn) {
  if (!sec) return "—";
  const hours = sec / 3600;
  if (hours < 1)
    return `${Math.round(sec / 60)} ${t("admin.jobs.unit_minutes")}`;
  return `${hours.toLocaleString(localeTag(), { maximumFractionDigits: 1 })} ${t("admin.jobs.unit_hours")}`;
}

export function DaemonsPageContent() {
  const t = useT();
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState<Record<string, boolean>>({});
  const [refreshingAll, setRefreshingAll] = useState(false);

  const { data: agents = [], isLoading } = useQuery({
    queryKey: queryKeys.adminAgents,
    queryFn: fetchAdminAgents,
    refetchInterval: 5000,
  });

  const actionMut = useMutation({
    mutationFn: async ({
      id,
      action,
    }: {
      id: string;
      action: "refresh" | "restart";
    }) => {
      if (action === "refresh") return refreshAdminAgent(id);
      return restartAdminAgent(id);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminAgents });
    },
  });

  async function runAction(id: string, action: "refresh" | "restart") {
    if (
      action === "restart" &&
      !confirm(t("admin.daemons.restart_confirm"))
    ) {
      return;
    }
    setBusy((p) => ({ ...p, [id]: true }));
    try {
      await actionMut.mutateAsync({ id, action });
      toast.success(
        action === "refresh"
          ? t("admin.daemons.refreshed")
          : t("admin.daemons.restart_started")
      );
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("common.error"));
    } finally {
      setBusy((p) => ({ ...p, [id]: false }));
    }
  }

  async function refreshAll() {
    if (agents.length === 0) return;
    setRefreshingAll(true);
    try {
      const results = await Promise.allSettled(
        agents.map((a) => refreshAdminAgent(a.id))
      );
      const failed = results.filter((r) => r.status === "rejected").length;
      await queryClient.invalidateQueries({ queryKey: queryKeys.adminAgents });
      if (failed === 0) toast.success(t("admin.daemons.all_refreshed"));
      else
        toast.warning(
          t("admin.daemons.partial_refreshed", {
            done: agents.length - failed,
            total: agents.length,
          })
        );
    } finally {
      setRefreshingAll(false);
    }
  }

  const online = agents.filter((a) => agentState(a) === "online").length;

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              Vortanix Agent
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("admin.daemons.subtitle")}
            </p>
          </div>
          <div className="flex items-center gap-3.5">
            <span className="font-mono text-xs text-muted-foreground">
              {t("admin.daemons.autorefresh")}
            </span>
            <Button
              type="button"
              variant="outline"
              className="h-[38px] text-[13px]"
              onClick={refreshAll}
              disabled={refreshingAll || agents.length === 0}
            >
              {refreshingAll
                ? t("common.updating")
                : t("admin.daemons.refresh_all")}
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap gap-2.5">
          <StatCard label={t("admin.daemons.total")} value={agents.length} />
          <StatCard
            label={t("admin.infra.online")}
            value={online}
            tone="emerald"
          />
          <StatCard
            label={t("admin.daemons.need_attention")}
            value={agents.length - online}
            tone="amber"
          />
        </div>

        {isLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            <Skeleton className="h-64 w-full rounded-2xl" />
            <Skeleton className="h-64 w-full rounded-2xl" />
            <Skeleton className="h-64 w-full rounded-2xl" />
          </div>
        ) : agents.length === 0 ? (
          <div className="rounded-2xl border bg-card py-16 text-center text-sm text-muted-foreground">
            {t("admin.daemons.empty")}{" "}
            <Link
              href="/admin/locations"
              className="text-foreground underline underline-offset-4"
            >
              {t("admin.daemons.empty_link")}
            </Link>
            .
          </div>
        ) : (
          <div className="grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(340px,1fr))]">
            {agents.map((agent) => (
              <AgentCard
                key={agent.id}
                agent={agent}
                loading={!!busy[agent.id]}
                onRefresh={() => runAction(agent.id, "refresh")}
                onRestart={() => runAction(agent.id, "restart")}
              />
            ))}
          </div>
        )}
      </div>
    </PageShell>
  );
}

function AgentCard({
  agent,
  loading,
  onRefresh,
  onRestart,
}: {
  agent: AdminAgentListItem;
  loading: boolean;
  onRefresh: () => void;
  onRestart: () => void;
}) {
  const t = useT();
  const loc = agent.location;
  const state = agentState(agent);
  const tone = agentStateClasses(state);
  const name = loc?.name || agent.name || "—";
  const code = loc?.code || agent.code || "—";

  return (
    <div
      className={cn(
        "flex flex-col gap-4.5 rounded-2xl border bg-card px-6 py-5",
        loading && "opacity-70"
      )}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 space-y-1">
          <div className="font-mono text-[11px] tracking-wider text-muted-foreground uppercase">
            {loc?.region || agent.region || t("common.location")}
          </div>
          <div className="truncate text-base font-semibold">
            {name} ({code})
          </div>
          <div className="font-mono text-xs text-muted-foreground">
            {agent.host || "—"}
          </div>
        </div>
        <span
          className={cn(
            "inline-flex shrink-0 items-center gap-2 rounded-full border px-2.5 py-1 text-xs",
            tone.border,
            tone.text
          )}
        >
          <span className={cn("size-1.5 rounded-full", tone.dot)} />
          {t(AGENT_STATE_LABEL_KEY[state])}
        </span>
      </div>

      <div className="flex flex-col gap-2.5 rounded-xl border bg-muted/30 px-4 py-3.5 text-xs">
        <InfoRow
          label={t("admin.daemons.platform")}
          value={agent.platform || "—"}
        />
        <InfoRow label="PID" value={agent.pid ? String(agent.pid) : "—"} />
        <InfoRow label="Uptime" value={uptimeText(agent.uptime_sec, t)} />
        <InfoRow
          label={t("admin.daemons.contact")}
          value={agent.last_seen_human || "—"}
          tone={state === "online" ? "emerald" : "amber"}
        />
      </div>

      <div className="flex gap-2">
        <Button
          variant="outline"
          asChild
          className="h-[34px] flex-1 text-[13px]"
        >
          <Link href={`/admin/daemons/${agent.id}`}>
            {t("admin.locations.view")}
          </Link>
        </Button>
        <Button
          variant="outline"
          className="h-[34px] flex-1 text-[13px]"
          disabled={loading}
          onClick={onRefresh}
        >
          {t("common.refresh")}
        </Button>
        <Button
          variant="outline"
          className="h-[34px] flex-1 text-[13px]"
          disabled={loading}
          onClick={onRestart}
        >
          {t("common.restart")}
        </Button>
      </div>
    </div>
  );
}

function InfoRow({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: "emerald" | "amber";
}) {
  return (
    <div className="flex justify-between gap-3">
      <span className="text-muted-foreground">{label}</span>
      <span
        className={cn(
          "truncate font-mono",
          tone === "emerald" && "text-emerald-500",
          tone === "amber" && "text-amber-500"
        )}
      >
        {value}
      </span>
    </div>
  );
}

function StatCard({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone?: "emerald" | "amber";
}) {
  return (
    <div className="flex min-w-[180px] flex-1 flex-col gap-1.5 rounded-xl border bg-card px-4 py-3.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span
        className={cn(
          "font-mono text-xl",
          tone === "emerald" && "text-emerald-500",
          tone === "amber" && "text-amber-500"
        )}
      >
        {value}
      </span>
    </div>
  );
}
