"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchServerMigrations,
  fetchServerMigrationTargets,
  migrateServer,
  type MigrationTargetNode,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

const STAGE_LABEL_KEYS: Record<string, string> = {
  queued: "admin.migrate.stage.queued",
  stopping: "admin.migrate.stage.stopping",
  transferring: "admin.migrate.stage.transferring",
  switching: "admin.migrate.stage.switching",
  starting: "admin.migrate.stage.starting",
  cleanup: "admin.migrate.stage.cleanup",
  done: "admin.migrate.stage.done",
};

function fmtSize(mb: number, t: TranslateFn): string {
  if (!mb || mb <= 0) return "—";
  if (mb >= 1024)
    return `${(mb / 1024).toFixed(1)} ${t("admin.infra.unit_gb")}`;
  return `${Math.round(mb)} ${t("admin.infra.unit_mb")}`;
}

function fmtBytes(bytes: number, t: TranslateFn): string {
  if (!bytes) return "0";
  const gb = bytes / (1024 * 1024 * 1024);
  if (gb >= 1) return `${gb.toFixed(2)} ${t("admin.infra.unit_gb")}`;
  return `${(bytes / (1024 * 1024)).toFixed(0)} ${t("admin.infra.unit_mb")}`;
}

export function MigrateServerDialog({
  serverId,
  serverName,
  open,
  onOpenChange,
}: {
  serverId: string;
  serverName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const qc = useQueryClient();
  const [selected, setSelected] = useState<string>("");
  const [removeSource, setRemoveSource] = useState(true);

  const targetsQuery = useQuery({
    queryKey: ["server-migration-targets", serverId],
    queryFn: () => fetchServerMigrationTargets(serverId),
    enabled: open,
  });

  const historyQuery = useQuery({
    queryKey: ["server-migrations", serverId],
    queryFn: () => fetchServerMigrations(serverId),
    enabled: open,
    refetchInterval: open ? 5_000 : false,
  });

  const active = historyQuery.data?.migrations.find(
    (m) => m.status === "pending" || m.status === "running"
  );
  const last = historyQuery.data?.migrations[0];

  const migrateMut = useMutation({
    mutationFn: () => migrateServer(serverId, selected, removeSource),
    onSuccess: (res) => {
      toast.success(t("admin.migrate.queued", { node: res.to_node }));
      void qc.invalidateQueries({ queryKey: ["server-migrations", serverId] });
      void qc.invalidateQueries({ queryKey: ["admin-jobs"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.migrate.failed")),
  });

  const nodes = targetsQuery.data?.nodes ?? [];
  const needRAM = targetsQuery.data?.need_ram_mb ?? 0;
  const needDisk = targetsQuery.data?.need_disk_mb ?? 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            {t("admin.migrate.title", { name: serverName })}
          </DialogTitle>
          <DialogDescription>
            {t("admin.migrate.description")}
          </DialogDescription>
        </DialogHeader>

        {active ? (
          <div className="rounded-lg border bg-card p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="font-medium">
                  {STAGE_LABEL_KEYS[active.stage]
                    ? t(STAGE_LABEL_KEYS[active.stage])
                    : active.stage || t("common.processing")}
                </div>
                <div className="text-sm text-muted-foreground">
                  {active.from_node} → {active.to_node}
                </div>
              </div>
              <Badge>
                {t("admin.migrate.transferred", {
                  size: fmtBytes(active.bytes, t),
                })}
              </Badge>
            </div>
            <p className="mt-2 text-xs text-muted-foreground">
              {t("admin.migrate.background_hint")}
            </p>
          </div>
        ) : (
          <>
            {needRAM > 0 || needDisk > 0 ? (
              <p className="text-sm text-muted-foreground">
                {t("admin.migrate.needs")}{" "}
                {needRAM > 0 ? `${fmtSize(needRAM, t)} RAM` : ""}
                {needRAM > 0 && needDisk > 0 ? " · " : ""}
                {needDisk > 0
                  ? t("admin.migrate.need_disk", {
                      size: fmtSize(needDisk, t),
                    })
                  : ""}
              </p>
            ) : null}

            {targetsQuery.isLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-16 w-full" />
                <Skeleton className="h-16 w-full" />
              </div>
            ) : nodes.length === 0 ? (
              <div className="rounded-lg border p-6 text-center text-sm text-muted-foreground">
                {t("admin.migrate.no_nodes")}
              </div>
            ) : (
              <div className="space-y-2">
                {nodes.map((node: MigrationTargetNode) => {
                  const isSelected = selected === node.id;
                  return (
                    <button
                      key={node.id}
                      type="button"
                      disabled={!node.fits}
                      onClick={() => setSelected(node.id)}
                      className={[
                        "w-full rounded-lg border p-3 text-left transition",
                        isSelected ? "border-primary bg-accent" : "",
                        node.fits
                          ? "hover:bg-accent"
                          : "cursor-not-allowed opacity-60",
                      ].join(" ")}
                    >
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="font-medium">{node.name}</div>
                        {node.fits ? (
                          <Badge variant="outline">
                            {t("admin.migrate.node_servers", {
                              count: node.servers,
                            })}
                          </Badge>
                        ) : (
                          <Badge variant="destructive">{node.reason}</Badge>
                        )}
                      </div>
                      <div className="mt-1 text-xs text-muted-foreground">
                        {node.fqdn}
                        {node.metrics_known
                          ? t("admin.migrate.node_metrics", {
                              ram: fmtSize(node.ram_free_mb, t),
                              disk: fmtSize(node.disk_free_mb, t),
                              cpu: Math.round(node.cpu_percent),
                            })
                          : t("admin.migrate.node_no_metrics")}
                      </div>
                    </button>
                  );
                })}
              </div>
            )}

            <div className="flex items-start gap-2 rounded-lg border p-3">
              <Checkbox
                id="remove-source"
                checked={removeSource}
                onCheckedChange={(v) => setRemoveSource(v === true)}
              />
              <div>
                <Label htmlFor="remove-source">
                  {t("admin.migrate.remove_source")}
                </Label>
                <p className="text-xs text-muted-foreground">
                  {t("admin.migrate.remove_source_hint")}
                </p>
              </div>
            </div>
          </>
        )}

        {last && !active ? (
          <div className="text-xs text-muted-foreground">
            {t("admin.migrate.last", {
              from: last.from_node,
              to: last.to_node,
            })}{" "}
            {last.status === "completed"
              ? t("admin.migrate.last_ok", {
                  size: fmtBytes(last.bytes, t),
                })
              : last.status === "failed"
                ? t("admin.migrate.last_failed", { error: last.error })
                : last.status}
          </div>
        ) : null}

        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            {t("common.close")}
          </Button>
          {!active ? (
            <Button
              onClick={() => migrateMut.mutate()}
              disabled={!selected || migrateMut.isPending}
            >
              {t("admin.migrate.start")}
            </Button>
          ) : null}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
