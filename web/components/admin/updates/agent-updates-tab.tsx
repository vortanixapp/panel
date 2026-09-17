"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Download, Loader2 } from "lucide-react";
import { toast } from "sonner";

import { SettingsCard, ToggleRow } from "@/components/admin/settings/settings-ui";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import {
  fetchAdminAgentUpdates,
  setAdminAgentAutoUpdate,
  startAdminAgentUpdates,
  updateAdminUpdateSettings,
  type AdminAgentUpdateNode,
  type AdminAgentUpdates,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const BUSY = ["pending", "pulling", "restarting"];

function isBusy(node: AdminAgentUpdateNode): boolean {
  return BUSY.includes(node.update?.status ?? "");
}

export function AgentUpdatesTab() {
  const t = useT();
  const queryClient = useQueryClient();

  const agents = useQuery({
    queryKey: queryKeys.adminAgentUpdates,
    queryFn: fetchAdminAgentUpdates,
    refetchInterval: (query) =>
      query.state.data?.nodes.some(isBusy) ? 3000 : 30_000,
  });

  const start = useMutation({
    mutationFn: startAdminAgentUpdates,
    onSuccess: (res) => {
      if (res.started > 0) {
        toast.success(t("admin.updates.agents.started", { count: res.started }));
      }
      const names = new Map(agents.data?.nodes.map((n) => [n.id, n.name]));
      for (const item of res.results) {
        if (!item.ok) {
          toast.error(`${names.get(item.id) ?? item.id}: ${item.error ?? ""}`);
        }
      }
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminAgentUpdates });
    },
    onError: (e: Error) =>
      toast.error(t("admin.updates.action_failed", { error: e.message })),
  });

  const auto = useMutation({
    mutationFn: ({ ids, value }: { ids: string[]; value: boolean }) =>
      setAdminAgentAutoUpdate(ids, value),
    onMutate: ({ ids, value }) => {
      queryClient.setQueryData<AdminAgentUpdates>(queryKeys.adminAgentUpdates, (prev) =>
        prev
          ? {
              ...prev,
              nodes: prev.nodes.map((n) =>
                ids.length === 0 || ids.includes(n.id) ? { ...n, auto_update: value } : n
              ),
            }
          : prev
      );
    },
    onError: (e: Error) =>
      toast.error(t("admin.updates.action_failed", { error: e.message })),
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: queryKeys.adminAgentUpdates }),
  });

  const master = useMutation({
    mutationFn: (value: boolean) =>
      updateAdminUpdateSettings({ agents_auto_enabled: value }),
    onMutate: (value) => {
      queryClient.setQueryData<AdminAgentUpdates>(queryKeys.adminAgentUpdates, (prev) =>
        prev ? { ...prev, auto_enabled: value } : prev
      );
    },
    onSuccess: () => toast.success(t("admin.updates.auto.saved")),
    onError: (e: Error) =>
      toast.error(t("admin.updates.action_failed", { error: e.message })),
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: queryKeys.adminAgentUpdates }),
  });

  if (agents.isLoading) {
    return <Skeleton className="h-40 w-full rounded-xl" />;
  }

  const data = agents.data;
  const nodes = data?.nodes ?? [];
  const versioned = /^\d+\.\d+\.\d+/.test(data?.target_version ?? "");
  const outdated = nodes.filter(
    (n) => n.outdated && !isBusy(n) && (n.online || n.ssh)
  );
  const allAuto = nodes.length > 0 && nodes.every((n) => n.auto_update);

  return (
    <SettingsCard
      title={t("admin.updates.agents.title")}
      description={
        versioned
          ? t("admin.updates.agents.description", { version: data?.target_version ?? "" })
          : t("admin.updates.agents.description_dev", { image: data?.image ?? "" })
      }
      action={
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={nodes.length === 0 || auto.isPending}
            onClick={() => auto.mutate({ ids: [], value: !allAuto })}
          >
            {allAuto
              ? t("admin.updates.agents.auto_all_off")
              : t("admin.updates.agents.auto_all_on")}
          </Button>
          <Button
            size="sm"
            disabled={outdated.length === 0 || start.isPending}
            onClick={() => start.mutate(outdated.map((n) => n.id))}
          >
            {start.isPending ? <Loader2 className="animate-spin" /> : <Download />}
            {t("admin.updates.agents.update_outdated", { count: outdated.length })}
          </Button>
        </div>
      }
    >
      <div className="mb-5">
        <ToggleRow
          label={t("admin.updates.agents.auto_toggle")}
          hint={t("admin.updates.agents.auto_toggle_hint")}
          checked={data?.auto_enabled === true}
          onCheckedChange={(value) => master.mutate(value)}
        />
      </div>
      {nodes.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t("admin.updates.agents.empty")}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="vx-tbl vx-tbl-flat w-full min-w-[680px] text-sm">
            <thead>
              <tr className="border-b text-left text-xs text-muted-foreground">
                <th className="pe-4 pb-2 font-normal">{t("admin.updates.agents.col.location")}</th>
                <th className="pe-4 pb-2 font-normal">{t("admin.updates.agents.col.version")}</th>
                <th className="pe-4 pb-2 font-normal">{t("admin.updates.agents.col.state")}</th>
                <th className="pe-4 pb-2 font-normal">{t("admin.updates.agents.col.auto")}</th>
                <th className="pb-2" />
              </tr>
            </thead>
            <tbody>
              {nodes.map((node) => (
                <AgentRow
                  key={node.id}
                  node={node}
                  autoEnabled={data?.auto_enabled === true}
                  starting={start.isPending}
                  onUpdate={() => start.mutate([node.id])}
                  onAuto={(value) => auto.mutate({ ids: [node.id], value })}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
      <div className="mt-4 space-y-1 text-xs text-muted-foreground">
        <p>{t("admin.updates.agents.auto_hint")}</p>
        <p>{t("admin.updates.agents.old_hint")}</p>
      </div>
    </SettingsCard>
  );
}

function AgentRow({
  node,
  autoEnabled,
  starting,
  onUpdate,
  onAuto,
}: {
  node: AdminAgentUpdateNode;
  autoEnabled: boolean;
  starting: boolean;
  onUpdate: () => void;
  onAuto: (value: boolean) => void;
}) {
  const t = useT();
  const busy = isBusy(node);
  const failed = node.update?.status === "failed";

  let state;
  if (busy) {
    state = (
      <Badge>
        <Loader2 className="size-3 animate-spin" />
        {t(`admin.updates.agents.status.${node.update?.status}`)}
        {node.update?.method === "ssh" && ` · ${t("admin.updates.agents.via_ssh")}`}
      </Badge>
    );
  } else if (failed && node.outdated) {
    state = <Badge variant="destructive">{t("admin.updates.agents.status.failed")}</Badge>;
  } else if (!node.version) {
    state = <Badge variant="secondary">{t("admin.updates.agents.never")}</Badge>;
  } else if (!node.online) {
    state = <Badge variant="secondary">{t("admin.updates.agents.offline")}</Badge>;
  } else if (node.outdated) {
    state = (
      <Badge variant="outline" className="border-amber-500/40 text-amber-600 dark:text-amber-500">
        {t("admin.updates.agents.outdated")}
      </Badge>
    );
  } else {
    state = <Badge variant="secondary">{t("admin.updates.agents.current")}</Badge>;
  }

  const canUpdate =
    !busy && !starting && (node.online || node.ssh) && (node.outdated || !node.version);

  return (
    <tr className="border-b align-top last:border-0">
      <td className="py-3 pe-4" data-cell="full">
        <div className="flex items-center gap-2">
          <span
            className={cn(
              "size-2 shrink-0 rounded-full",
              node.online ? "bg-emerald-500" : "bg-muted-foreground/40"
            )}
          />
          <span className="font-medium">{node.name}</span>
        </div>
        {node.code && (
          <div className="ms-4 font-mono text-xs text-muted-foreground">{node.code}</div>
        )}
      </td>
      <td className="py-3 pe-4 font-mono" data-label={t("admin.updates.agents.col.version")}>{node.version || "—"}</td>
      <td className="py-3 pe-4" data-label={t("admin.updates.agents.col.state")}>
        {state}
        {failed && node.outdated && node.update?.error && (
          <div className="mt-1 max-w-[360px] text-xs text-destructive">{node.update.error}</div>
        )}
      </td>
      <td className="py-3 pe-4" data-label={t("admin.updates.agents.col.auto")}>
        <Switch
          checked={node.auto_update}
          onCheckedChange={onAuto}
          className={cn(!autoEnabled && "opacity-50")}
        />
      </td>
      <td className="py-3 text-end" data-cell="actions">
        <Button variant="outline" size="sm" disabled={!canUpdate} onClick={onUpdate}>
          {t("admin.updates.agents.update")}
        </Button>
      </td>
    </tr>
  );
}
