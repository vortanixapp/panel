"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchAdminLocation,
  installAdminLocationDaemon,
  refreshAdminLocationDaemon,
  restartAdminLocationDaemon,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

export function LocationDaemonContent() {
  const t = useT();
  const params = useParams();
  const id = String(params?.id ?? "");
  const queryClient = useQueryClient();
  const [log, setLog] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminLocation(id),
    queryFn: () => fetchAdminLocation(id),
    enabled: !!id,
  });

  const refreshMut = useMutation({
    mutationFn: () => refreshAdminLocationDaemon(id),
    onSuccess: () => {
      toast.success(t("admin.daemons.refreshed"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminLocation(id) });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const restartMut = useMutation({
    mutationFn: () => restartAdminLocationDaemon(id),
    onSuccess: () => {
      toast.success(t("admin.daemons.restart_started"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminLocation(id) });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const installMut = useMutation({
    mutationFn: () => installAdminLocationDaemon(id),
    onSuccess: (res) => {
      setLog(res.log ?? t("admin.location_daemon.installed"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminLocation(id) });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  if (isLoading) {
    return (
      <PageShell variant="admin">
        <Skeleton className="h-24 w-full" />
      </PageShell>
    );
  }

  const location = (data?.location ?? null) as Record<string, unknown> | null;
  const daemon = (data?.daemon ?? null) as Record<string, unknown> | null;

  if (!location) {
    return (
      <PageShell variant="admin">
        <p className="py-20 text-center text-muted-foreground">
          {t("admin.location.not_found")}
        </p>
      </PageShell>
    );
  }

  const isOnline = Boolean(daemon?.is_online ?? daemon?.status === "online");
  const busy = refreshMut.isPending || restartMut.isPending || installMut.isPending;

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Vortanix Agent</h1>
          <p className="text-sm text-muted-foreground">
            {String(location.name)} ({String(location.code)}) — {String(location.ssh_host ?? "—")}
          </p>
        </div>
        <Button variant="outline" asChild>
          <Link href={`/admin/locations/${id}`}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            {t("admin.location_daemon.back")}
          </Link>
        </Button>
      </div>

      {daemon && (daemon.status !== "unknown" || daemon.version) ? (
        <>
          <div className="mb-6 grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">
                  {t("common.status")}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">
                    {t("admin.agent.state")}
                  </span>
                  <Badge variant={isOnline ? "secondary" : "destructive"}>
                    {isOnline
                      ? t("admin.infra.online")
                      : t("admin.infra.offline")}
                  </Badge>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">
                    {t("admin.agent.last_seen")}
                  </span>
                  <span className="text-sm">
                    {String(daemon.last_seen_human ?? daemon.last_seen_at ?? "—")}
                  </span>
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-base">
                  {t("admin.agent.system")}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">
                    {t("admin.daemons.platform")}
                  </span>
                  <span>{String(daemon.platform ?? "—")}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">
                    {t("admin.maps.version")}
                  </span>
                  <span>{String(daemon.version ?? "—")}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">PID</span>
                  <span>{String(daemon.pid ?? "—")}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">
                    {t("admin.location_daemon.uptime")}
                  </span>
                  <span>
                    {daemon.uptime_sec
                      ? `${Math.round((Number(daemon.uptime_sec) / 3600) * 10) / 10} ${t("admin.jobs.unit_hours")}`
                      : "—"}
                  </span>
                </div>
              </CardContent>
            </Card>
          </div>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">
                {t("admin.dashboard.manage")}
              </CardTitle>
            </CardHeader>
            <CardContent className="flex flex-wrap gap-3">
              <Button disabled={busy} onClick={() => refreshMut.mutate()}>
                <RefreshCw className={`mr-2 h-4 w-4 ${refreshMut.isPending ? "animate-spin" : ""}`} />
                {t("admin.location_daemon.refresh")}
              </Button>
              <Button
                variant="secondary"
                disabled={busy}
                onClick={() => {
                  if (confirm(t("admin.daemons.restart_confirm"))) {
                    restartMut.mutate();
                  }
                }}
              >
                {t("common.restart")}
              </Button>
            </CardContent>
          </Card>
        </>
      ) : (
        <Card>
          <CardContent className="flex flex-col items-center py-12 text-center">
            <h2 className="text-xl font-bold">
              {t("admin.location_daemon.not_configured")}
            </h2>
            <p className="mt-2 max-w-md text-muted-foreground">
              {t("admin.location_daemon.not_installed")}
            </p>
            <div className="mt-6 flex flex-wrap justify-center gap-3">
              <Button disabled={busy} onClick={() => installMut.mutate()}>
                {installMut.isPending
                  ? t("admin.agent.installing")
                  : t("admin.location_daemon.install")}
              </Button>
              <Button variant="outline" asChild>
                <Link href={`/admin/locations/${id}/setup`}>
                  {t("admin.location_daemon.go_setup")}
                </Link>
              </Button>
            </div>
            {log ? (
              <pre className="mt-8 max-h-96 w-full overflow-auto rounded-md bg-muted p-4 text-left text-xs">
                {log}
              </pre>
            ) : null}
          </CardContent>
        </Card>
      )}
    </PageShell>
  );
}
