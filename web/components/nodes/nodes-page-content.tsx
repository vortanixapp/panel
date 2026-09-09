"use client";

import { useMemo, useState } from "react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { DataTable } from "@/components/data-table";
import { getNodesColumns } from "@/components/nodes/nodes-columns";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import type { PanelVariant } from "@/lib/panel-paths";
import {
  useCreateNode,
  useDeleteNode,
  useNodeInstall,
  useNodes,
} from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";

export function NodesPageContent({ variant = "admin" }: { variant?: PanelVariant }) {
  const t = useT();
  const { data: nodes = [], isLoading } = useNodes();
  const createNode = useCreateNode();
  const deleteNode = useDeleteNode();
  const install = useNodeInstall();

  const [name, setName] = useState("");
  const [fqdn, setFqdn] = useState("");
  const [error, setError] = useState("");
  const [installScript, setInstallScript] = useState("");

  async function onCreate(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await createNode.mutateAsync({ name, fqdn });
      toast.success(t("admin.nodes.created"));
      setName("");
      setFqdn("");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.error"));
    }
  }

  async function onInstall(id: string) {
    setError("");
    try {
      const res = await install.mutateAsync(id);
      setInstallScript(res.script);
      toast.success(t("admin.nodes.script_ready"));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.error"));
    }
  }

  async function onDelete(id: string) {
    if (!confirm(t("admin.nodes.delete_confirm"))) return;
    await deleteNode.mutateAsync(id);
    toast.success(t("admin.nodes.deleted"));
  }

  const columns = useMemo(
    () =>
      getNodesColumns(
        {
          onInstall,
          onDelete,
          installPending: install.isPending,
        },
        t
      ),
    [install.isPending, t]
  );

  return (
    <PageShell variant={variant}>
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">{t("layout.nodes")}</h1>
        <p className="text-sm text-muted-foreground">{t("admin.nodes.subtitle")}</p>
      </div>

      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="text-base">{t("admin.dashboard.add_node")}</CardTitle>
        </CardHeader>
        <CardContent>
          {error && <p className="mb-3 text-sm text-destructive">{error}</p>}
          <form
            onSubmit={onCreate}
            className="flex flex-col gap-3 sm:flex-row sm:items-end"
          >
            <div className="flex-1 space-y-1">
              <Input
                placeholder={t("common.name")}
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>
            <div className="flex-1 space-y-1">
              <Input
                placeholder={t("admin.nodes.fqdn")}
                value={fqdn}
                onChange={(e) => setFqdn(e.target.value)}
                required
              />
            </div>
            <Button type="submit" disabled={createNode.isPending}>
              {createNode.isPending ? t("common.creating") : t("common.create")}
            </Button>
          </form>
        </CardContent>
      </Card>

      {installScript && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle className="text-base">{t("admin.nodes.script_title")}</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="overflow-x-auto rounded-md bg-muted p-4 text-xs">
              {installScript}
            </pre>
          </CardContent>
        </Card>
      )}

      {isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <DataTable
          columns={columns}
          data={nodes}
          searchPlaceholder={t("admin.nodes.search_placeholder")}
          searchKey="name"
          filters={[
            {
              columnId: "status",
              title: t("common.status"),
              options: [
                { label: "Online", value: "online" },
                { label: "Offline", value: "offline" },
                { label: "Pending", value: "pending" },
              ],
            },
          ]}
          emptyMessage={t("admin.nodes.empty")}
        />
      )}
    </PageShell>
  );
}
