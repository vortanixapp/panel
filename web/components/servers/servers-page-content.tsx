"use client";

import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { MigrateServerDialog } from "@/components/admin/servers/migrate-server-dialog";
import { AssignIPDialog } from "@/components/admin/servers/assign-ip-dialog";
import { DataTable } from "@/components/data-table";
import { getServersColumns } from "@/components/servers/servers-columns";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import type { PanelBasePath, PanelVariant } from "@/lib/panel-paths";
import { variantToBasePath } from "@/lib/panel-paths";
import { isStaffRole } from "@/lib/rbac";
import { reinstallServer } from "@/lib/api";
import { getTranslationsVersion } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";
import {
  useAdminServers,
  useAdminToggleServerBlock,
  useCreateServer,
  useDeleteServer,
  useMe,
  useNodes,
  usePowerServer,
  useServers,
} from "@/hooks/use-queries";

type ServersPageContentProps = {
  variant?: PanelVariant;
  allowCreate?: boolean;
};

export function ServersPageContent({
  variant = "user",
  allowCreate = true,
}: ServersPageContentProps) {
  const t = useT();
  const basePath: PanelBasePath = variantToBasePath(variant);
  const { data: me } = useMe();
  const isStaff = isStaffRole(me?.role);
  const canManage = variant === "admin" || isStaff;
  const serversQuery = useServers();
  const adminServersQuery = useAdminServers(variant === "admin");
  const servers = (variant === "admin" ? adminServersQuery.data : serversQuery.data) ?? [];
  const isLoading = variant === "admin" ? adminServersQuery.isLoading : serversQuery.isLoading;
  const { data: nodes = [], isError: nodesError } = useNodes(
    canManage && allowCreate
  );
  const createServer = useCreateServer();
  const deleteServer = useDeleteServer();
  const powerServer = usePowerServer();
  const blockMutation = useAdminToggleServerBlock();

  const [name, setName] = useState("");
  const [nodeId, setNodeId] = useState("");
  const [error, setError] = useState("");
  const [migrateTarget, setMigrateTarget] = useState<{
    id: string;
    name: string;
  } | null>(null);
  const [ipTarget, setIpTarget] = useState<{
    id: string;
    name: string;
    nodeId: string;
  } | null>(null);

  useEffect(() => {
    if (!nodeId && nodes[0]) setNodeId(nodes[0].id);
  }, [nodes, nodeId]);

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await createServer.mutateAsync({ nodeId, name, gameId: "test" });
      toast.success(t("servers.admin.created"));
      setName("");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.error"));
    }
  }

  async function onPower(id: string, action: string) {
    setError("");
    try {
      await powerServer.mutateAsync({ id, action });
      toast.success(t("servers.admin.command_sent", { action }));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.error"));
    }
  }

  async function onReinstall(id: string) {
    if (!confirm(t("servers.admin.reinstall_confirm"))) return;
    try {
      await reinstallServer(id);
      toast.success(t("servers.shell.reinstall_started"));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.error"));
    }
  }

  async function onToggleBlock(id: string, blocked: boolean) {
    const reason =
      blocked && variant === "admin"
        ? (prompt(t("servers.admin.block_reason"), "") ?? "")
        : "";
    try {
      await blockMutation.mutateAsync({ id, blocked, reason });
      toast.success(
        blocked ? t("servers.admin.blocked") : t("servers.admin.unblocked")
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.error"));
    }
  }

  async function onDelete(id: string) {
    if (!confirm(t("servers.admin.delete_confirm"))) return;
    await deleteServer.mutateAsync(id);
    toast.success(t("servers.settings.server_deleted"));
  }

  const columns = useMemo(
    () =>
      getServersColumns({
        basePath,
        onPower,
        onDelete,
        onReinstall,
        onToggleBlock,
        onMigrate: (id: string, serverName: string) =>
          setMigrateTarget({ id, name: serverName }),
        onAssignIP: (id: string, serverName: string, nodeId: string) =>
          setIpTarget({ id, name: serverName, nodeId }),
        powerPending: powerServer.isPending,
        canManage,
        showAdminOps: variant === "admin",
      }),
    // Версия переводов в зависимостях: сама t() стабильна, и без неё
    // заголовки колонок остались бы на языке первой отрисовки.
    [basePath, canManage, powerServer.isPending, variant, getTranslationsVersion()]
  );

  return (
    <PageShell variant={variant}>
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">
          {t("common.servers")}
        </h1>
        <p className="text-sm text-muted-foreground">
          {variant === "admin"
            ? t("servers.admin.subtitle_all")
            : t("servers.admin.subtitle_own")}
        </p>
      </div>

      {allowCreate && canManage && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle className="text-base">
              {t("servers.admin.create_title")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {error && <p className="mb-3 text-sm text-destructive">{error}</p>}
            {nodesError && (
              <p className="mb-3 text-sm text-muted-foreground">
                {t("servers.admin.no_nodes_hint")}
              </p>
            )}
            <form
              onSubmit={onCreate}
              className="flex flex-col gap-3 sm:flex-row sm:items-end"
            >
              <Input
                placeholder={t("common.name")}
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className="sm:flex-1"
              />
              <select
                value={nodeId}
                onChange={(e) => setNodeId(e.target.value)}
                required
                disabled={nodes.length === 0}
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm sm:flex-1"
              >
                {nodes.length === 0 ? (
                  <option value="">{t("servers.admin.no_nodes")}</option>
                ) : (
                  nodes.map((n) => (
                    <option key={n.id} value={n.id}>
                      {n.name}
                    </option>
                  ))
                )}
              </select>
              <Button
                type="submit"
                disabled={createServer.isPending || !nodeId}
              >
                {t("common.create")}
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      {isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <DataTable
          columns={columns}
          data={servers}
          searchPlaceholder={t("servers.admin.search_placeholder")}
          searchKey="name"
          filters={[
            {
              columnId: "status",
              title: t("common.status"),
              options: [
                { label: "Running", value: "running" },
                { label: "Stopped", value: "stopped" },
                { label: "Starting", value: "starting" },
              ],
            },
          ]}
          emptyMessage={t("servers.admin.empty")}
        />
      )}

      {ipTarget ? (
        <AssignIPDialog
          serverId={ipTarget.id}
          serverName={ipTarget.name}
          nodeId={ipTarget.nodeId}
          open
          onOpenChange={(open) => {
            if (!open) setIpTarget(null);
          }}
        />
      ) : null}

      {migrateTarget ? (
        <MigrateServerDialog
          serverId={migrateTarget.id}
          serverName={migrateTarget.name}
          open
          onOpenChange={(open) => {
            if (!open) setMigrateTarget(null);
          }}
        />
      ) : null}
    </PageShell>
  );
}
