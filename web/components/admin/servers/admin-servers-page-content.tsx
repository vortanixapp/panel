"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Plus } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { DataTable, DataTableBulkActions } from "@/components/data-table";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/servers/confirm-dialog";
import { AssignIPDialog } from "@/components/admin/servers/assign-ip-dialog";
import { MigrateServerDialog } from "@/components/admin/servers/migrate-server-dialog";
import { BlockServerDialog } from "@/components/admin/servers/block-server-dialog";
import { CreateServerDialog } from "@/components/admin/servers/create-server-dialog";
import { getAdminServersColumns } from "@/components/admin/servers/admin-servers-columns";
import {
  adminServersBulk,
  adminToggleServerBlock,
  deleteServer,
  fetchAdminServers,
  powerServer,
  reinstallServer,
  type AdminServerBulkAction,
  type AdminServerListItem,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { isStaffRole } from "@/lib/rbac";
import { getTranslationsVersion } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { useMe } from "@/hooks/use-queries";

type BlockTarget = { ids: string[]; name?: string };

function StatCard({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone?: "warn" | "danger";
}) {
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs tracking-wide text-muted-foreground uppercase">{label}</div>
      <div
        className={cn(
          "mt-1 text-2xl font-semibold tabular-nums",
          tone === "danger" && "text-destructive",
          tone === "warn" && "text-amber-600 dark:text-amber-500"
        )}
      >
        {value}
      </div>
    </div>
  );
}

export function AdminServersPageContent() {
  const t = useT();
  const queryClient = useQueryClient();
  const { data: me } = useMe();
  const canWrite = isStaffRole(me?.role);

  const serversQuery = useQuery({
    queryKey: queryKeys.adminServers,
    queryFn: fetchAdminServers,
    refetchInterval: 15_000,
  });
  const data = serversQuery.data;
  const servers = useMemo(() => data?.servers ?? [], [data]);

  const [createOpen, setCreateOpen] = useState(false);
  const [blockTarget, setBlockTarget] = useState<BlockTarget | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AdminServerListItem | null>(null);
  const [reinstallTarget, setReinstallTarget] = useState<AdminServerListItem | null>(null);
  const [migrateTarget, setMigrateTarget] = useState<AdminServerListItem | null>(null);
  const [ipTarget, setIpTarget] = useState<AdminServerListItem | null>(null);

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: queryKeys.adminServers });
  }

  function reportError(err: unknown) {
    toast.error(err instanceof Error ? err.message : t("common.error"));
  }

  const powerMutation = useMutation({
    mutationFn: ({ id, action }: { id: string; action: string }) =>
      powerServer(id, action),
    onSuccess: (_res, vars) => {
      toast.success(t("servers.admin.command_sent", { action: vars.action }));
      refresh();
    },
    onError: reportError,
  });

  const blockMutation = useMutation({
    mutationFn: ({ ids, blocked, reason }: { ids: string[]; blocked: boolean; reason: string }) =>
      ids.length === 1
        ? adminToggleServerBlock(ids[0], blocked, reason).then(() => ({
            done: 1,
            failures: [] as { id: string; error: string }[],
          }))
        : adminServersBulk(ids, blocked ? "block" : "unblock", reason),
    onSuccess: (res, vars) => {
      if (res.failures.length > 0) {
        toast.warning(
          t("servers.admin.bulk.partial", { done: res.done, failed: res.failures.length })
        );
      } else {
        toast.success(
          vars.blocked ? t("servers.admin.blocked") : t("servers.admin.unblocked")
        );
      }
      setBlockTarget(null);
      refresh();
    },
    onError: reportError,
  });

  const bulkMutation = useMutation({
    mutationFn: ({ ids, action }: { ids: string[]; action: AdminServerBulkAction }) =>
      adminServersBulk(ids, action),
    onSuccess: (res) => {
      if (res.failures.length > 0) {
        toast.warning(
          t("servers.admin.bulk.partial", { done: res.done, failed: res.failures.length })
        );
      } else {
        toast.success(t("servers.admin.bulk.done", { count: res.done }));
      }
      refresh();
    },
    onError: reportError,
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteServer(id),
    onSuccess: (res) => {
      if (res.cleanup === "deferred") {
        toast.warning(t("servers.settings.server_deleted_deferred"));
      } else {
        toast.success(t("servers.settings.server_deleted"));
      }
      setDeleteTarget(null);
      refresh();
    },
    onError: reportError,
  });

  const reinstallMutation = useMutation({
    mutationFn: (id: string) => reinstallServer(id),
    onSuccess: () => {
      toast.success(t("servers.shell.reinstall_started"));
      setReinstallTarget(null);
      refresh();
    },
    onError: reportError,
  });

  const gameOptions = useMemo(() => {
    const names = new Set<string>();
    servers.forEach((s) => {
      if (s.game?.name) names.add(s.game.name);
    });
    return [...names].sort().map((name) => ({ label: name, value: name }));
  }, [servers]);

  const locationOptions = useMemo(() => {
    const names = new Set<string>();
    servers.forEach((s) => {
      if (s.location?.name) names.add(s.location.name);
    });
    return [...names].sort().map((name) => ({ label: name, value: name }));
  }, [servers]);

  const columns = useMemo(
    () =>
      getAdminServersColumns({
        canWrite,
        onPower: (server, action) =>
          powerMutation.mutate({ id: server.id, action }),
        onBlock: (server) => setBlockTarget({ ids: [server.id], name: server.name }),
        onUnblock: (server) =>
          blockMutation.mutate({ ids: [server.id], blocked: false, reason: "" }),
        onMigrate: setMigrateTarget,
        onAssignIP: setIpTarget,
        onReinstall: setReinstallTarget,
        onDelete: setDeleteTarget,
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canWrite, getTranslationsVersion()]
  );

  const statusOptions = [
    { label: t("servers.status.running"), value: "running" },
    { label: t("servers.status.stopped"), value: "stopped" },
    { label: t("servers.status.starting"), value: "starting" },
    { label: t("servers.status.stopping"), value: "stopping" },
    { label: t("servers.status.installing"), value: "installing" },
  ];

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("common.servers")}</h1>
          <p className="text-sm text-muted-foreground">
            {t("servers.admin.subtitle_all")}
          </p>
        </div>
        {canWrite && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("servers.admin.create.title")}
          </Button>
        )}
      </div>

      <div className="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label={t("servers.list.stat_total")} value={data?.total ?? 0} />
        <StatCard label={t("servers.list.stat_running")} value={data?.active_count ?? 0} />
        <StatCard
          label={t("servers.list.stat_expiring")}
          value={data?.expiring_soon ?? 0}
          tone={data?.expiring_soon ? "warn" : undefined}
        />
        <StatCard
          label={t("servers.admin.list.stat_blocked")}
          value={data?.blocked_count ?? 0}
          tone={data?.blocked_count ? "danger" : undefined}
        />
      </div>

      {serversQuery.isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <DataTable
          columns={columns}
          data={servers}
          searchPlaceholder={t("servers.admin.search_placeholder")}
          searchKey="name"
          emptyMessage={t("servers.admin.empty")}
          filters={[
            { columnId: "status", title: t("common.status"), options: statusOptions },
            { columnId: "game", title: t("common.game"), options: gameOptions },
            {
              columnId: "location",
              title: t("servers.admin.list.col_location"),
              options: locationOptions,
            },
          ]}
          bulkActions={(table) => {
            if (!canWrite) return null;
            const selected = table
              .getFilteredSelectedRowModel()
              .rows.map((row) => row.original);
            const ids = selected.map((s) => s.id);
            const run = (action: AdminServerBulkAction) => {
              bulkMutation.mutate({ ids, action });
              table.resetRowSelection();
            };
            return (
              <DataTableBulkActions table={table} entityName={t("common.servers")}>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={bulkMutation.isPending}
                  onClick={() => run("start")}
                >
                  {t("common.start")}
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={bulkMutation.isPending}
                  onClick={() => run("stop")}
                >
                  {t("servers.shell.power_off")}
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={bulkMutation.isPending}
                  onClick={() => run("kill")}
                >
                  {t("servers.admin.actions.kill")}
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={bulkMutation.isPending}
                  onClick={() => {
                    setBlockTarget({ ids });
                    table.resetRowSelection();
                  }}
                >
                  {t("servers.admin.actions.block")}
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={bulkMutation.isPending}
                  onClick={() => run("unblock")}
                >
                  {t("servers.admin.actions.unblock")}
                </Button>
              </DataTableBulkActions>
            );
          }}
        />
      )}

      <CreateServerDialog open={createOpen} onOpenChange={setCreateOpen} />

      <BlockServerDialog
        open={!!blockTarget}
        serverName={blockTarget?.name}
        count={blockTarget?.ids.length}
        pending={blockMutation.isPending}
        onOpenChange={(open) => {
          if (!open) setBlockTarget(null);
        }}
        onConfirm={(reason) => {
          if (!blockTarget) return;
          blockMutation.mutate({ ids: blockTarget.ids, blocked: true, reason });
        }}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
        title={t("servers.admin.delete.title")}
        description={t("servers.admin.delete.description", {
          name: deleteTarget?.name ?? "",
          owner: deleteTarget?.owner_email ?? "—",
        })}
        confirmLabel={t("common.delete")}
        requirePhrase={deleteTarget?.name}
        pending={deleteMutation.isPending}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
      />

      <ConfirmDialog
        open={!!reinstallTarget}
        onOpenChange={(open) => {
          if (!open) setReinstallTarget(null);
        }}
        title={t("servers.shell.reinstall_title")}
        description={t("servers.shell.reinstall_desc", {
          name: reinstallTarget?.name ?? "",
        })}
        confirmLabel={t("servers.shell.reinstall")}
        requirePhrase={reinstallTarget?.name}
        pending={reinstallMutation.isPending}
        onConfirm={() => reinstallTarget && reinstallMutation.mutate(reinstallTarget.id)}
      />

      {ipTarget && (
        <AssignIPDialog
          serverId={ipTarget.id}
          serverName={ipTarget.name}
          nodeId={ipTarget.node_id}
          open
          onOpenChange={(open) => {
            if (!open) setIpTarget(null);
          }}
        />
      )}

      {migrateTarget && (
        <MigrateServerDialog
          serverId={migrateTarget.id}
          serverName={migrateTarget.name}
          open
          onOpenChange={(open) => {
            if (!open) setMigrateTarget(null);
          }}
        />
      )}
    </PageShell>
  );
}
