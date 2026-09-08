"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  createAdminMysqlInstance,
  deleteAdminMysqlInstance,
  fetchAdminMysqlInstances,
  fetchNodes,
  updateAdminMysqlInstance,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";

export function AdminMysqlPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const [nodeId, setNodeId] = useState("");
  const [key, setKey] = useState("");
  const [name, setName] = useState("");
  const [container, setContainer] = useState("");
  const [port, setPort] = useState("3306");
  const [rootPassword, setRootPassword] = useState("");
  // Правка инстанса шла четырьмя окнами prompt подряд: отменить на середине
  // было нельзя, а пустой ответ молча затирал значение.
  const [editing, setEditing] = useState<{
    node_id: string;
    key: string;
    name: string;
    container: string;
    port: string;
    root_password: string;
  } | null>(null);

  const nodesQuery = useQuery({
    queryKey: ["admin-mysql-nodes"],
    queryFn: fetchNodes,
  });
  const mysqlQuery = useQuery({
    queryKey: ["admin-mysql-instances"],
    queryFn: fetchAdminMysqlInstances,
  });

  const createMut = useMutation({
    mutationFn: createAdminMysqlInstance,
    onSuccess: () => {
      toast.success(t("admin.mysql.created"));
      setKey("");
      setName("");
      setContainer("");
      setPort("3306");
      setRootPassword("");
      void qc.invalidateQueries({ queryKey: ["admin-mysql-instances"] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("admin.mysql.create_failed")
      ),
  });
  const updateMut = useMutation({
    mutationFn: updateAdminMysqlInstance,
    onSuccess: () => {
      toast.success(t("admin.mysql.updated"));
      setEditing(null);
      void qc.invalidateQueries({ queryKey: ["admin-mysql-instances"] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("admin.mysql.update_failed")
      ),
  });
  const deleteMut = useMutation({
    mutationFn: ({ nodeId, key }: { nodeId: string; key: string }) =>
      deleteAdminMysqlInstance(nodeId, key),
    onSuccess: () => {
      toast.success(t("admin.mysql.deleted"));
      void qc.invalidateQueries({ queryKey: ["admin-mysql-instances"] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("admin.mysql.delete_failed")
      ),
  });

  const nodes = useMemo(() => nodesQuery.data ?? [], [nodesQuery.data]);
  const instances = mysqlQuery.data?.instances ?? [];

  function createInstance() {
    if (!nodeId || !key.trim() || !container.trim()) {
      toast.error(t("admin.mysql.required_fields"));
      return;
    }
    createMut.mutate({
      node_id: nodeId,
      key: key.trim(),
      name: name.trim() || undefined,
      container: container.trim(),
      port: Number(port) || 3306,
      root_password: rootPassword || undefined,
    });
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">MySQL</h1>
        <p className="text-sm text-muted-foreground">{t("admin.mysql.subtitle")}</p>
      </div>

      <div className="mb-6 grid gap-3 rounded-2xl border bg-card p-4 md:grid-cols-6">
        <div className="space-y-1 md:col-span-2">
          <Label className="text-xs text-muted-foreground">{t("admin.mysql.node")}</Label>
          <select
            value={nodeId}
            onChange={(e) => setNodeId(e.target.value)}
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm"
          >
            <option value="">{t("admin.mysql.select_node")}</option>
            {nodes.map((n) => (
              <option key={n.id} value={n.id}>
                {n.name}
              </option>
            ))}
          </select>
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">Key</Label>
          <Input value={key} onChange={(e) => setKey(e.target.value)} placeholder="mysql80-main" />
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">Name</Label>
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="MySQL 8 main" />
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">Container</Label>
          <Input value={container} onChange={(e) => setContainer(e.target.value)} placeholder="vortanix-mysql80-3306" />
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">Port</Label>
          <Input value={port} onChange={(e) => setPort(e.target.value)} placeholder="3306" />
        </div>
        <div className="space-y-1 md:col-span-5">
          <Label className="text-xs text-muted-foreground">Root password (optional)</Label>
          <Input value={rootPassword} onChange={(e) => setRootPassword(e.target.value)} placeholder="root password" />
        </div>
        <div className="flex items-end">
          <Button className="w-full" onClick={createInstance} disabled={createMut.isPending}>
            {createMut.isPending ? t("common.adding") : t("admin.mysql.add")}
          </Button>
        </div>
      </div>

      {mysqlQuery.isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <div className="overflow-auto rounded-2xl border bg-card">
          <table className="w-full text-sm">
            <thead className="bg-muted/50 text-xs uppercase text-muted-foreground">
              <tr>
                <th className="px-3 py-2 text-left">{t("admin.mysql.node")}</th>
                <th className="px-3 py-2 text-left">Key</th>
                <th className="px-3 py-2 text-left">Container</th>
                <th className="px-3 py-2 text-left">Port</th>
                <th className="px-3 py-2 text-left">Name</th>
                <th className="px-3 py-2 text-left">{t("common.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {instances.map((inst) => (
                <tr key={`${inst.node_id}:${inst.key}`} className="border-t">
                  <td className="px-3 py-2">{inst.node_name}</td>
                  <td className="px-3 py-2 font-mono text-xs">{inst.key}</td>
                  <td className="px-3 py-2">{inst.container}</td>
                  <td className="px-3 py-2">{inst.port}</td>
                  <td className="px-3 py-2">{inst.name || "—"}</td>
                  <td className="px-3 py-2">
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() =>
                          setEditing({
                            node_id: inst.node_id,
                            key: inst.key,
                            name: inst.name || "",
                            container: inst.container,
                            port: String(inst.port),
                            root_password: "",
                          })
                        }
                        disabled={updateMut.isPending}
                      >
                        {t("admin.infra.edit")}
                      </Button>
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => {
                          if (
                            !confirm(
                              t("admin.mysql.delete_confirm", { key: inst.key })
                            )
                          )
                            return;
                          deleteMut.mutate({ nodeId: inst.node_id, key: inst.key });
                        }}
                        disabled={deleteMut.isPending}
                      >
                        {t("common.delete")}
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
              {instances.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-3 py-8 text-center text-muted-foreground">
                    {t("admin.mysql.empty")}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      <Dialog open={editing !== null} onOpenChange={(open) => !open && setEditing(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {t("admin.mysql.dialog_title", { key: editing?.key ?? "" })}
            </DialogTitle>
            <DialogDescription>{t("admin.mysql.dialog_hint")}</DialogDescription>
          </DialogHeader>
          {editing && (
            <div className="grid gap-3">
              <div className="space-y-1">
                <Label className="text-xs text-muted-foreground">Container</Label>
                <Input
                  value={editing.container}
                  onChange={(e) => setEditing({ ...editing, container: e.target.value })}
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs text-muted-foreground">Port</Label>
                <Input
                  value={editing.port}
                  onChange={(e) => setEditing({ ...editing, port: e.target.value })}
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs text-muted-foreground">Name</Label>
                <Input
                  value={editing.name}
                  onChange={(e) => setEditing({ ...editing, name: e.target.value })}
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs text-muted-foreground">
                  {t("admin.mysql.root_password_keep")}
                </Label>
                <Input
                  type="password"
                  value={editing.root_password}
                  onChange={(e) => setEditing({ ...editing, root_password: e.target.value })}
                />
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="ghost" onClick={() => setEditing(null)}>
              {t("common.cancel")}
            </Button>
            <Button
              onClick={() => {
                if (!editing) return;
                if (!editing.container.trim()) {
                  toast.error(t("admin.mysql.container_required"));
                  return;
                }
                updateMut.mutate({
                  node_id: editing.node_id,
                  key: editing.key,
                  name: editing.name.trim() || undefined,
                  container: editing.container.trim(),
                  port: Number(editing.port) || undefined,
                  root_password: editing.root_password || undefined,
                });
              }}
              disabled={updateMut.isPending}
            >
              {updateMut.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageShell>
  );
}
