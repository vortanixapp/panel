"use client";

import { useState } from "react";
import Link from "next/link";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowRight, Download, Loader2, RotateCcw } from "lucide-react";
import { toast } from "sonner";
import { SettingsCard } from "@/components/admin/settings/settings-ui";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import {
  setAdminAgentAutoUpdate,
  startAdminAgentUpdates,
  type AdminAgentUpdateNode,
  type AdminAgentUpdates,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";
import {
  agentBusy,
  agentFailed,
  agentMatches,
  agentReachable,
  formatRelative,
  summarizeAgents,
  type AgentFilter,
} from "./update-utils";

const FILTERS: AgentFilter[] = ["all", "outdated", "updating", "failed", "offline"];

export function AgentsSection({
  data,
  isLoading,
  isError,
  onRetry,
}: {
  data: AdminAgentUpdates | undefined;
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
}) {
  const t = useT();
  const qc = useQueryClient();
  const [filter, setFilter] = useState<AgentFilter>("all");

  const start = useMutation({
    mutationFn: startAdminAgentUpdates,
    onSuccess: (res) => {
      if (res.started > 0) toast.success(t("admin.updates.agents.started", { count: res.started }));
      const names = new Map(data?.nodes.map((n) => [n.id, n.name]));
      for (const item of res.results) {
        if (!item.ok) toast.error(`${names.get(item.id) ?? item.id}: ${item.error ?? ""}`);
      }
      void qc.invalidateQueries({ queryKey: queryKeys.adminAgentUpdates });
    },
    onError: (e: Error) => toast.error(t("admin.updates.action_failed", { error: e.message })),
  });

  const auto = useMutation({
    mutationFn: ({ ids, value }: { ids: string[]; value: boolean }) => setAdminAgentAutoUpdate(ids, value),
    onMutate: ({ ids, value }) => {
      qc.setQueryData<AdminAgentUpdates>(queryKeys.adminAgentUpdates, (prev) =>
        prev
          ? {
              ...prev,
              nodes: prev.nodes.map((n) => (ids.length === 0 || ids.includes(n.id) ? { ...n, auto_update: value } : n)),
            }
          : prev
      );
    },
    onError: (e: Error) => toast.error(t("admin.updates.action_failed", { error: e.message })),
    onSettled: () => qc.invalidateQueries({ queryKey: queryKeys.adminAgentUpdates }),
  });

  const nodes = data?.nodes ?? [];
  const summary = summarizeAgents(nodes);
  const versioned = /^\d+\.\d+\.\d+/.test(data?.target_version ?? "");
  const updatable = nodes.filter((n) => agentMatches(n, "outdated") && agentReachable(n) && n.version);
  const allAuto = nodes.length > 0 && nodes.every((n) => n.auto_update);
  const counts: Record<AgentFilter, number> = {
    all: summary.total,
    outdated: summary.outdated,
    updating: summary.updating,
    failed: summary.failed,
    offline: summary.offline,
  };
  const active = counts[filter] > 0 || filter === "all" ? filter : "all";
  const visible = nodes.filter((n) => agentMatches(n, active));

  return (
    <div id="agents" className="scroll-mt-24">
      <SettingsCard
        title={t("admin.updates.agents.title")}
        description={
          versioned
            ? t("admin.updates.agents.description", { version: data?.target_version ?? "" })
            : data
              ? t("admin.updates.agents.description_dev", { image: data.image })
              : undefined
        }
        action={
          nodes.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={auto.isPending}
                onClick={() => auto.mutate({ ids: [], value: !allAuto })}
              >
                {allAuto ? t("admin.updates.agents.auto_all_off") : t("admin.updates.agents.auto_all_on")}
              </Button>
              <Button size="sm" disabled={updatable.length === 0 || start.isPending} onClick={() => start.mutate(updatable.map((n) => n.id))}>
                {start.isPending ? <Loader2 className="animate-spin" /> : <Download />}
                {t("admin.updates.agents.update_outdated", { count: updatable.length })}
              </Button>
            </div>
          ) : undefined
        }
      >
        {isLoading ? (
          <div className="space-y-2.5">
            <Skeleton className="h-9 w-72 rounded-lg" />
            <Skeleton className="h-16 w-full rounded-xl" />
            <Skeleton className="h-16 w-full rounded-xl" />
          </div>
        ) : isError ? (
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-[13px] text-destructive">
            <span>{t("admin.updates.agents.load_failed")}</span>
            <Button size="sm" variant="outline" onClick={onRetry}>
              <RotateCcw />
              {t("common.retry")}
            </Button>
          </div>
        ) : nodes.length === 0 ? (
          <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed px-6 py-10 text-center">
            <p className="text-[13.5px] text-muted-foreground">{t("admin.updates.agents.empty")}</p>
            <Button asChild variant="outline" size="sm">
              <Link href="/admin/locations">{t("admin.updates.agents.add_location")}</Link>
            </Button>
          </div>
        ) : (
          <>
            <div role="tablist" aria-label={t("admin.updates.agents.title")} className="mb-4 flex flex-wrap gap-1.5">
              {FILTERS.filter((f) => f === "all" || counts[f] > 0).map((f) => (
                <button
                  key={f}
                  type="button"
                  role="tab"
                  aria-selected={active === f}
                  onClick={() => setFilter(f)}
                  className={cn(
                    "inline-flex h-8 items-center gap-1.5 rounded-lg border px-3 text-[12.5px] transition-colors",
                    active === f
                      ? "border-primary bg-primary text-primary-foreground"
                      : "border-border text-muted-foreground hover:text-foreground"
                  )}
                >
                  {t(`admin.updates.agents.filter.${f}`)}
                  <span className={cn("tabular-nums", active === f ? "opacity-80" : "opacity-70")}>{counts[f]}</span>
                </button>
              ))}
            </div>
            <ul className="divide-y rounded-xl border">
              {visible.map((node) => (
                <AgentRow
                  key={node.id}
                  node={node}
                  target={data?.target_version ?? ""}
                  masterAuto={data?.auto_enabled === true}
                  starting={start.isPending}
                  onUpdate={() => start.mutate([node.id])}
                  onAuto={(value) => auto.mutate({ ids: [node.id], value })}
                />
              ))}
            </ul>
            <div className="mt-4 space-y-1 text-[12px] leading-relaxed text-muted-foreground">
              <p>{t("admin.updates.agents.auto_hint")}</p>
              <p>{t("admin.updates.agents.old_hint")}</p>
            </div>
          </>
        )}
      </SettingsCard>
    </div>
  );
}

function AgentRow({
  node,
  target,
  masterAuto,
  starting,
  onUpdate,
  onAuto,
}: {
  node: AdminAgentUpdateNode;
  target: string;
  masterAuto: boolean;
  starting: boolean;
  onUpdate: () => void;
  onAuto: (value: boolean) => void;
}) {
  const t = useT();
  const busy = agentBusy(node);
  const failed = agentFailed(node) && !busy;
  const canUpdate = !busy && !starting && agentReachable(node) && (node.outdated || !node.version);

  let state: React.ReactNode;
  if (busy) {
    state = (
      <Badge>
        <Loader2 className="size-3 animate-spin" />
        {t(`admin.updates.agents.status.${node.update?.status}`)}
        {node.update?.method === "ssh" && ` · ${t("admin.updates.agents.via_ssh")}`}
      </Badge>
    );
  } else if (failed) {
    state = <Badge variant="destructive">{t("admin.updates.agents.status.failed")}</Badge>;
  } else if (!node.version) {
    state = <Badge variant="secondary">{t("admin.updates.agents.never")}</Badge>;
  } else if (node.outdated) {
    state = (
      <Badge variant="outline" className="border-amber-500/40 text-amber-600 dark:text-amber-500">
        {t("admin.updates.agents.outdated")}
      </Badge>
    );
  } else {
    state = (
      <Badge variant="secondary" className="text-emerald-600 dark:text-emerald-400">
        {t("admin.updates.agents.current")}
      </Badge>
    );
  }

  return (
    <li className="flex flex-wrap items-center gap-x-4 gap-y-2.5 px-4 py-3.5">
      <div className="flex min-w-0 flex-1 basis-56 items-start gap-2.5">
        <span
          className={cn("mt-1.5 size-2 shrink-0 rounded-full", node.online ? "bg-emerald-500" : "bg-muted-foreground/40")}
          aria-hidden
        />
        <div className="min-w-0">
          <div className="truncate text-[14px] font-medium">{node.name}</div>
          <div className="mt-0.5 flex flex-wrap gap-x-2.5 text-[12px] text-muted-foreground">
            {node.code && <span className="font-mono">{node.code}</span>}
            {!node.online && (
              <span>
                {node.last_seen
                  ? t("admin.updates.agents.seen", { when: formatRelative(node.last_seen) })
                  : t("admin.updates.agents.offline")}
              </span>
            )}
          </div>
        </div>
      </div>

      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <span className="inline-flex items-center gap-1.5 font-mono text-[12.5px]">
          <span className={cn(node.outdated && "text-amber-600 dark:text-amber-500")}>{node.version || "—"}</span>
          {node.outdated && target && (
            <>
              <ArrowRight className="size-3 text-muted-foreground" />
              <span className="text-muted-foreground">{target}</span>
            </>
          )}
        </span>
        {state}
      </div>

      <div className="ms-auto flex items-center gap-3">
        <label className={cn("flex items-center gap-2 text-[12.5px] text-muted-foreground", !masterAuto && "opacity-60")}>
          {t("admin.updates.agents.auto")}
          <Switch
            checked={node.auto_update}
            onCheckedChange={onAuto}
            aria-label={t("admin.updates.agents.auto_aria", { name: node.name })}
          />
        </label>
        <Button
          variant="outline"
          size="sm"
          disabled={!canUpdate}
          onClick={onUpdate}
          title={!agentReachable(node) ? t("admin.updates.agents.unreachable") : undefined}
        >
          {failed ? t("admin.updates.agents.retry") : t("admin.updates.agents.update")}
        </Button>
      </div>

      {failed && node.update?.error && (
        <p className="w-full rounded-lg border border-destructive/20 bg-destructive/5 px-3 py-2 text-[12.5px] text-destructive">
          {node.update.error}
        </p>
      )}
    </li>
  );
}
