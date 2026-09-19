"use client";

import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { RefreshCw, RotateCcw } from "lucide-react";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchAdminAgentUpdates,
  fetchAdminPanelUpdateStatus,
  fetchAdminUpdates,
  startAdminPanelUpdate,
  type UpdaterJob,
  type UpdaterStatus,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";
import { AgentsSection } from "./agents-section";
import { AutoUpdateCard } from "./auto-update-card";
import { InstallCard } from "./install-card";
import { PanelStatus } from "./panel-status";
import { agentBusy, formatRelative, summarizeAgents } from "./update-utils";
import { WhatsNewCard } from "./whats-new-card";

export function UpdatesPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const [watching, setWatching] = useState(false);
  const [checking, setChecking] = useState(false);
  const [confirmVersion, setConfirmVersion] = useState<string | null>(null);
  const [manualOpen, setManualOpen] = useState<boolean | null>(null);
  const pageVersion = useRef<string | null>(null);

  const updates = useQuery({
    queryKey: queryKeys.adminUpdates,
    queryFn: () => fetchAdminUpdates(),
    staleTime: 5 * 60_000,
  });
  const agents = useQuery({
    queryKey: queryKeys.adminAgentUpdates,
    queryFn: fetchAdminAgentUpdates,
    refetchInterval: (query) => (query.state.data?.nodes.some(agentBusy) ? 3000 : 30_000),
  });

  const data = updates.data;
  if (data && pageVersion.current === null) pageVersion.current = data.current_version;

  const jobRunning = data?.updater?.job?.state === "running";
  const status = useQuery({
    queryKey: queryKeys.adminPanelUpdateStatus,
    queryFn: fetchAdminPanelUpdateStatus,
    enabled: watching || jobRunning,
    refetchInterval: 2500,
    retry: false,
  });

  const statusJobState = status.data?.updater?.job?.state;
  useEffect(() => {
    if (statusJobState && statusJobState !== "running") {
      setWatching(false);
      void qc.invalidateQueries({ queryKey: queryKeys.adminUpdates });
      void qc.invalidateQueries({ queryKey: queryKeys.adminAgentUpdates });
    }
  }, [statusJobState, qc]);

  const start = useMutation({
    mutationFn: (version: string) => startAdminPanelUpdate(version),
    onSuccess: (res) => {
      setConfirmVersion(null);
      toast.success(t("admin.updates.started"));
      qc.setQueryData(queryKeys.adminPanelUpdateStatus, {
        current_version: data?.current_version ?? "",
        updater: { ...(data?.updater as UpdaterStatus), job: res.job },
      });
      setWatching(true);
    },
    onError: (e: Error) => toast.error(t("admin.updates.action_failed", { error: e.message })),
  });

  const recheck = async () => {
    setChecking(true);
    try {
      qc.setQueryData(queryKeys.adminUpdates, await fetchAdminUpdates(true));
      void agents.refetch();
    } catch (e) {
      toast.error(t("admin.updates.action_failed", { error: e instanceof Error ? e.message : "" }));
    } finally {
      setChecking(false);
    }
  };

  const updater = data?.updater;
  const job: UpdaterJob | null = status.data?.updater?.job ?? updater?.job ?? null;
  const restarting = (watching || jobRunning) && status.isError;
  const liveVersion = status.data?.current_version ?? data?.current_version ?? "";
  const reloadVersion =
    job?.state === "succeeded" && pageVersion.current !== null && liveVersion && liveVersion !== pageVersion.current
      ? liveVersion
      : null;
  const agentsSummary = agents.data ? summarizeAgents(agents.data.nodes) : null;
  const manualVisible = manualOpen ?? (updater ? !updater.available : false);

  const openManual = () => {
    setManualOpen(true);
    window.setTimeout(() => document.getElementById("manual")?.scrollIntoView({ behavior: "smooth", block: "start" }), 50);
  };

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-6 pb-10">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div className="space-y-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-tight">{t("admin.updates.title")}</h1>
            <p className="text-sm text-muted-foreground">{t("admin.updates.subtitle")}</p>
          </div>
          {data && !data.checks_disabled && (
            <div className="flex items-center gap-3">
              {data.checked_at && (
                <span className="text-[12.5px] text-muted-foreground">
                  {t("admin.updates.checked", { when: formatRelative(data.checked_at) })}
                </span>
              )}
              <Button variant="outline" size="sm" onClick={() => void recheck()} disabled={checking}>
                <RefreshCw className={cn(checking && "animate-spin")} />
                {t("admin.updates.recheck")}
              </Button>
            </div>
          )}
        </div>

        {updates.isLoading ? (
          <div className="space-y-6">
            <Skeleton className="h-52 w-full rounded-2xl" />
            <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
              <Skeleton className="h-72 w-full rounded-2xl" />
              <Skeleton className="h-72 w-full rounded-2xl" />
            </div>
          </div>
        ) : updates.isError || !data ? (
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-destructive/30 bg-destructive/5 px-5 py-4 text-[13.5px] text-destructive">
            <span>{updates.error instanceof Error && updates.error.message ? updates.error.message : t("admin.updates.load_failed")}</span>
            <Button size="sm" variant="outline" onClick={() => void updates.refetch()}>
              <RotateCcw />
              {t("common.retry")}
            </Button>
          </div>
        ) : (
          <>
            <PanelStatus
              data={data}
              liveVersion={liveVersion}
              job={job}
              restarting={restarting}
              starting={start.isPending}
              reloadVersion={reloadVersion}
              agents={agentsSummary}
              agentsAuto={agents.data ? agents.data.auto_enabled : null}
              onInstall={(version) => setConfirmVersion(version)}
              onManual={openManual}
            />

            <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
              <WhatsNewCard data={data} />
              <div className="space-y-6">
                <AutoUpdateCard
                  auto={data.auto}
                  agentsAuto={agents.data ? agents.data.auto_enabled : null}
                  updaterAvailable={updater?.available === true}
                />
                <InstallCard data={data} manualOpen={manualVisible} onManualOpenChange={setManualOpen} />
              </div>
            </div>

            <AgentsSection
              data={agents.data}
              isLoading={agents.isLoading}
              isError={agents.isError}
              onRetry={() => void agents.refetch()}
            />
          </>
        )}
      </div>

      <ConfirmDialog
        open={confirmVersion !== null}
        onOpenChange={(open) => !open && !start.isPending && setConfirmVersion(null)}
        title={t("admin.updates.confirm.title", { version: confirmVersion ?? "" })}
        desc={
          <ul className="mt-1 list-disc space-y-1 ps-5 text-start">
            <li>{t("admin.updates.confirm.backup")}</li>
            <li>{t("admin.updates.confirm.pull")}</li>
            <li>{t("admin.updates.confirm.restart")}</li>
            <li>{t("admin.updates.confirm.servers")}</li>
          </ul>
        }
        cancelBtnText={t("common.cancel")}
        confirmText={t("admin.updates.confirm.ok")}
        isLoading={start.isPending}
        handleConfirm={() => confirmVersion && start.mutate(confirmVersion)}
      />
    </PageShell>
  );
}
