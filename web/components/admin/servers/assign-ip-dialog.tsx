"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import {
  assignServerIP,
  fetchNodeIPs,
  releaseServerIP,
  type NodeIPAddress,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";

export function AssignIPDialog({
  serverId,
  serverName,
  nodeId,
  open,
  onOpenChange,
}: {
  serverId: string;
  serverName: string;
  nodeId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const qc = useQueryClient();
  const [selected, setSelected] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["node-ips", nodeId],
    queryFn: () => fetchNodeIPs(nodeId),
    enabled: open && !!nodeId,
  });

  const list = data?.addresses ?? [];
  const current = list.find((ip) => ip.server_id === serverId);
  const free = list.filter((ip) => ip.status === "free");

  const done = (message: string) => {
    toast.success(message);
    void qc.invalidateQueries({ queryKey: ["node-ips", nodeId] });
    void qc.invalidateQueries({ queryKey: ["admin-servers"] });
  };

  const assignMut = useMutation({
    mutationFn: () => assignServerIP(serverId, selected),
    onSuccess: (res) =>
      done(
        t("admin.assign_ip.assigned", {
          address: `${res.address}:${res.port}`,
        })
      ),
    onError: (e: Error) =>
      toast.error(e.message || t("admin.assign_ip.assign_failed")),
  });

  const releaseMut = useMutation({
    mutationFn: () => releaseServerIP(serverId),
    onSuccess: (res) =>
      done(
        t("admin.assign_ip.released", {
          address: `${res.address}:${res.port}`,
        })
      ),
    onError: (e: Error) =>
      toast.error(e.message || t("admin.assign_ip.release_failed")),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {t("admin.assign_ip.title", { name: serverName })}
          </DialogTitle>
          <DialogDescription>
            {t("admin.assign_ip.description")}
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <Skeleton className="h-24 w-full" />
        ) : current ? (
          <div className="rounded-lg border p-4">
            <div className="text-sm">
              {t("admin.assign_ip.current")}{" "}
              <span className="font-mono">{current.address}</span>
            </div>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("admin.assign_ip.release_hint")}
            </p>
          </div>
        ) : free.length === 0 ? (
          <div className="rounded-lg border p-6 text-center text-sm text-muted-foreground">
            {t("admin.assign_ip.no_free")}
          </div>
        ) : (
          <div className="space-y-2">
            {free.map((ip: NodeIPAddress) => (
              <button
                key={ip.id}
                type="button"
                onClick={() => setSelected(ip.id)}
                className={[
                  "flex w-full items-center justify-between rounded-lg border p-3 text-left transition hover:bg-accent",
                  selected === ip.id ? "border-primary bg-accent" : "",
                ].join(" ")}
              >
                <span className="font-mono text-sm">{ip.address}</span>
                {ip.label ? <Badge variant="outline">{ip.label}</Badge> : null}
              </button>
            ))}
          </div>
        )}

        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            {t("common.close")}
          </Button>
          {current ? (
            <Button
              variant="outline"
              onClick={() => releaseMut.mutate()}
              disabled={releaseMut.isPending}
            >
              {t("admin.assign_ip.release")}
            </Button>
          ) : (
            <Button
              onClick={() => assignMut.mutate()}
              disabled={!selected || assignMut.isPending}
            >
              {t("admin.assign_ip.assign")}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
