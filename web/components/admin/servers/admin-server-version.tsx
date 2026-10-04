"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Btn, Panel, VX_FAINT, VX_MUTED, VX_SELECT } from "@/components/vx/panel-ui";
import { confirmAction } from "@/components/action-dialog";
import {
  fetchServerRuntime,
  setServerRuntime,
  switchServerVersion,
  type DashboardServer,
} from "@/lib/api";
import { isServerExpired, isServerProvisioning, serverProvisioning, serverRuntime } from "@/lib/server-lifecycle";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

export function AdminServerVersion({ serverId, server }: { serverId: string; server: DashboardServer }) {
  const t = useT();
  const queryClient = useQueryClient();
  const versions = server.available_game_versions ?? [];
  const currentVersionId = server.game_version_id ?? "";
  const failed = serverProvisioning(server) === "failed";
  const [selectedVersionId, setSelectedVersionId] = useState("");
  const [keepData, setKeepData] = useState(!failed);
  const [selectedRuntime, setSelectedRuntime] = useState<string | null>(null);

  const effectiveVersionId = selectedVersionId || currentVersionId || versions[0]?.id || "";
  const stopped = ["offline", "stopped", "missing"].includes(serverRuntime(server));
  const canSwitch =
    versions.length > 0 && !isServerExpired(server) && !isServerProvisioning(server) && (stopped || failed);

  const versionMutation = useMutation({
    mutationFn: () => switchServerVersion(serverId, effectiveVersionId, keepData),
    onSuccess: () => {
      toast.success(t("servers.overview.version_changed"));
      setSelectedVersionId("");
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(serverId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminServerCard(serverId) });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const runtimeQuery = useQuery({
    queryKey: queryKeys.serverRuntime(serverId),
    queryFn: () => fetchServerRuntime(serverId),
    enabled: !!serverId,
  });
  const runtimeInfo = runtimeQuery.data;
  const runtimeCurrent = runtimeInfo?.current ?? "";
  const runtimeValue = selectedRuntime ?? runtimeCurrent;

  const runtimeMutation = useMutation({
    mutationFn: (version: string) => setServerRuntime(serverId, version),
    onSuccess: () => {
      toast.success(t("servers.overview.runtime_saved"));
      setSelectedRuntime(null);
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverRuntime(serverId) });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const hasRuntime = !!runtimeInfo?.supported && (runtimeInfo.versions?.length ?? 0) > 0;
  if (versions.length === 0 && !hasRuntime) return null;

  return (
    <Panel title={t("servers.overview.panel_version")}>
      <div className="flex flex-col gap-3">
        {versions.length > 0 && (
          <>
            <div className="flex flex-wrap gap-2">
              <select
                className={cn(VX_SELECT, "min-w-[150px] flex-1")}
                value={effectiveVersionId}
                onChange={(e) => setSelectedVersionId(e.target.value)}
                disabled={versionMutation.isPending}
              >
                {versions.map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.name}
                    {v.source_type ? ` · ${v.source_type}` : ""}
                  </option>
                ))}
              </select>
              <Btn
                disabled={
                  versionMutation.isPending ||
                  !canSwitch ||
                  !effectiveVersionId ||
                  effectiveVersionId === currentVersionId
                }
                title={!canSwitch ? t("servers.overview.switch_version_hint") : undefined}
                onClick={async () => {
                  const message = keepData
                    ? t("servers.overview.switch_version_keep_confirm")
                    : t("servers.overview.switch_version_confirm");
                  if (!(await confirmAction(message))) return;
                  versionMutation.mutate();
                }}
              >
                {t("servers.overview.switch")}
              </Btn>
            </div>
            <label className="flex items-start gap-2 text-[12px] leading-[1.35]">
              <input
                type="checkbox"
                className="mt-0.5"
                checked={keepData}
                onChange={(e) => setKeepData(e.target.checked)}
                disabled={versionMutation.isPending}
              />
              <span>
                {t("servers.overview.keep_data")}
                <span className={cn("block", VX_FAINT)}>{t("servers.overview.keep_data_hint")}</span>
              </span>
            </label>
          </>
        )}

        {hasRuntime && (
          <div className="flex flex-col gap-2">
            <span className={cn("text-[12px]", VX_MUTED)}>
              {runtimeInfo?.kind === "php" ? t("servers.overview.runtime_php") : t("servers.overview.runtime_java")}
            </span>
            <div className="flex flex-wrap gap-2">
              <select
                className={cn(VX_SELECT, "min-w-[150px] flex-1")}
                value={runtimeValue}
                onChange={(e) => setSelectedRuntime(e.target.value)}
                disabled={runtimeMutation.isPending || runtimeQuery.isLoading}
              >
                <option value="">{t("servers.overview.runtime_auto")}</option>
                {(runtimeInfo?.versions ?? []).map((v) => (
                  <option key={v} value={v}>
                    {v}
                  </option>
                ))}
              </select>
              <Btn
                disabled={runtimeMutation.isPending || runtimeValue === runtimeCurrent}
                onClick={() => runtimeMutation.mutate(runtimeValue)}
              >
                {t("common.save")}
              </Btn>
            </div>
            <span className={cn("text-[11.5px]", VX_FAINT)}>{t("servers.overview.runtime_hint")}</span>
          </div>
        )}
      </div>
    </Panel>
  );
}
