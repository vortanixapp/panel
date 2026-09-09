"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Btn,
  EmptyState,
  Field,
  InfoRow,
  Panel,
  VX_CODE,
  VX_FAINT,
  VX_INPUT,
  VX_MUTED,
  VX_ROW_LINE,
  VX_SELECT,
  Tile,
  Toggle,
} from "@/components/vx/panel-ui";
import { VxInlineLoader } from "@/components/vx/loader";
import { ServerInstallConsole } from "@/features/servers/install-console";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { serverPath } from "@/lib/panel-paths";
import {
  canShowReinstall,
  canShowUpdate,
  canSwitchVersion,
  isServerExpired,
  isServerProvisioning,
  serverProvisioning,
  serverRuntime,
} from "@/lib/server-lifecycle";
import { isCs16Server } from "@/lib/server-tabs";
import {
  changeServerMap,
  createServerBackup,
  fetchBilling,
  fetchServerBackups,
  fetchServerStatus,
  listServerMapsFolder,
  previewServerRenew,
  reinstallServer,
  renewServer,
  sendServerConsoleCommand,
  setServerAutoStart,
  switchServerVersion,
  updateServerGame,
  type MetricPoint,
} from "@/lib/api";
import { formatAmount } from "@/lib/format";
import { getLocale, t as translate } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useMe, useServerDetail, useServerMetrics } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";

const OVERVIEW_BACKUPS = 3;

function formatBackupSize(bytes: number): string {
  if (!bytes || bytes < 0) return "—";
  if (bytes < 1024 * 1024)
    return translate("servers.unit.kb", {
      value: Math.max(1, Math.round(bytes / 1024)),
    });
  if (bytes < 1024 * 1024 * 1024)
    return translate("servers.unit.mb", {
      value: (bytes / (1024 * 1024)).toFixed(1),
    });
  return translate("servers.unit.gb", {
    value: (bytes / (1024 * 1024 * 1024)).toFixed(2),
  });
}

const MC_SLUGS = ["minecraft", "mcjava", "mcpaper", "mcspigot", "mcforge", "mcfabric", "mcbedrock"];

const CHART_W = 600;
const CHART_H = 150;

function formatDateTime(value: string | null | undefined): string {
  if (!value) return "—";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return "—";
  return d
    .toLocaleString(getLocale() === "en" ? "en-GB" : "ru", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    })
    .replace(",", "");
}

function polyline(values: number[]): string {
  if (values.length === 0) return "";
  if (values.length === 1) values = [values[0], values[0]];
  const stepX = CHART_W / (values.length - 1);
  return values
    .map((v, i) => {
      const clamped = Math.max(0, Math.min(100, Number.isFinite(v) ? v : 0));
      const y = CHART_H - (clamped / 100) * (CHART_H - 6) - 3;
      return `${(i * stepX).toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");
}

function memPercent(p: MetricPoint): number {
  return p.mem_limit_mb > 0 ? (p.mem_used_mb / p.mem_limit_mb) * 100 : 0;
}

export function ServerOverviewTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const { data: server } = useServerDetail(id);
  const { data: me } = useMe();
  const { data: metrics = [], isLoading: metricsLoading } = useServerMetrics(id);

  const [renewOpen, setRenewOpen] = useState(false);
  const [renewPeriod, setRenewPeriod] = useState(30);
  const [renewPromoCode, setRenewPromoCode] = useState("");
  const [selectedVersionId, setSelectedVersionId] = useState("");
  const [changeMapOpen, setChangeMapOpen] = useState(false);
  const [mapActionLoading, setMapActionLoading] = useState("");
  const [playerAction, setPlayerAction] = useState<{
    open: boolean;
    type: "kick" | "ban";
    name: string;
    reason: string;
  }>({ open: false, type: "kick", name: "", reason: "" });

  const { data: liveStatus } = useQuery({
    queryKey: queryKeys.serverStatus(id),
    queryFn: () => fetchServerStatus(id),
    enabled: !!id,
    refetchInterval: (query) => {
      const data = query.state.data;
      const runtime = (data?.runtime_status || server?.runtime_status || "").toLowerCase();
      const prov = (server?.provisioning_status || "").toLowerCase();
      const transitional = ["starting", "stopping", "restarting"].includes(runtime);
      const provisioning = ["pending", "installing", "provisioning", "reinstalling", "updating"].includes(prov);
      return transitional || provisioning ? 3000 : 10000;
    },
  });

  const mapsFolderQuery = useQuery({
    queryKey: queryKeys.serverMapsFolder(id),
    queryFn: () => listServerMapsFolder(id),
    enabled: changeMapOpen && !!id,
  });

  const backupsQuery = useQuery({
    queryKey: ["server-backups", id],
    queryFn: () => fetchServerBackups(id),
    enabled: !!id,
  });

  const allBackups = useMemo(() => {
    const raw = backupsQuery.data?.backups ?? backupsQuery.data?.entries ?? [];
    return raw.filter((b) => !b.is_dir).sort((a, b) => b.name.localeCompare(a.name));
  }, [backupsQuery.data]);
  const latestBackups = allBackups.slice(0, OVERVIEW_BACKUPS);
  const backupsCount = allBackups.length;

  const createBackup = useMutation({
    mutationFn: () => createServerBackup(id),
    onSuccess: () => {
      toast.success(t("servers.overview.backup_created"));
      void queryClient.invalidateQueries({ queryKey: ["server-backups", id] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("servers.overview.backup_create_failed")),
  });

  const { data: billing } = useQuery({
    queryKey: queryKeys.billing(),
    queryFn: () => fetchBilling(),
    enabled: renewOpen,
  });

  const renewPreview = useQuery({
    queryKey: ["server-renew-preview", id, renewPeriod, renewPromoCode],
    queryFn: () => previewServerRenew(id, renewPeriod, renewPromoCode.trim() || undefined),
    enabled: renewOpen && !!id,
  });

  const lifecycleMutation = useMutation({
    mutationFn: async (action: string) => {
      if (action === "reinstall") return reinstallServer(id);
      if (action === "update") return updateServerGame(id);
      if (action.startsWith("version:")) return switchServerVersion(id, action.slice("version:".length));
      throw new Error("unknown action");
    },
    onSuccess: (_data, action) => {
      const labels: Record<string, string> = {
        reinstall: t("servers.shell.reinstall_started"),
        update: t("servers.overview.update_started"),
      };
      toast.success(labels[action] ?? t("servers.overview.version_changed"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const autoStartMutation = useMutation({
    mutationFn: (enabled: boolean) => setServerAutoStart(id, enabled),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) }),
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const renewMutation = useMutation({
    mutationFn: () =>
      renewServer(id, renewPeriod, billing?.selected_wallet?.id, renewPromoCode.trim() || undefined),
    onSuccess: () => {
      toast.success(t("servers.overview.renewed"));
      setRenewOpen(false);
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.overview.renew_error")
      ),
  });

  const consoleMutation = useMutation({
    mutationFn: (command: string) => sendServerConsoleCommand(id, command),
    onSuccess: () => toast.success(t("servers.overview.command_sent")),
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const changeMapMutation = useMutation({
    mutationFn: (map: string) => changeServerMap(id, map),
    onSuccess: () => {
      toast.success(t("servers.overview.map_command_sent"));
      setChangeMapOpen(false);
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverStatus(id) });
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const charts = useMemo(() => {
    const tail = metrics.slice(-60);
    return { cpu: polyline(tail.map((p) => p.cpu_pct)), ram: polyline(tail.map(memPercent)) };
  }, [metrics]);

  if (!server) return null;

  const expired = isServerExpired(server);
  const provisioning = isServerProvisioning(server);
  const prov = serverProvisioning(server);
  const runtime = serverRuntime(server);
  const running = runtime === "running";
  const isInstallFailed = prov === "failed";
  const isOwner = !!server.user_id && server.user_id === me?.user_id;
  const perms = server.viewer_permissions ?? {};
  const versions = server.available_game_versions ?? [];
  const currentVersionId = server.game_version_id ?? "";
  const effectiveVersionId = selectedVersionId || currentVersionId || versions[0]?.id || "";
  const activeVersion =
    versions.find((v) => v.id === effectiveVersionId) ?? versions[0];
  const manualInstall = activeVersion?.manual_install === true;
  const isCs16 = isCs16Server(server);
  const gameSlug = (server.game?.slug || server.game_id || "").toLowerCase();
  const isMinecraft = MC_SLUGS.includes(gameSlug);
  const canConsole = perms.can_console_command !== false;
  const canSettings = perms.can_settings_edit !== false;
  const showPlayerActions = isMinecraft && canConsole;
  const showChangeMap =
    isCs16 && canSettings && !provisioning && !isInstallFailed && !expired && running;
  const showUpdate = canShowUpdate(server);
  const showReinstall = canShowReinstall(server);

  const expiresAt = server.expires_at ? new Date(server.expires_at) : null;
  const daysLeft = expiresAt
    ? Math.max(0, Math.ceil((expiresAt.getTime() - Date.now()) / 86_400_000))
    : 0;

  const playersOnline = liveStatus?.players_online ?? [];
  const onlineCount = liveStatus?.online_players ?? playersOnline.length;
  const maxPlayers = liveStatus?.max_players ?? server.tariff?.slots ?? 0;
  const currentMap = liveStatus?.current_map ?? server.current_map ?? "—";
  const renewalPeriods = server.tariff?.renewal_periods?.length
    ? server.tariff.renewal_periods
    : [15, 30, 60, 180];

  const progress = server.provisioning_progress;
  const rawPercent = progress?.percent;
  const parsedPercent =
    typeof rawPercent === "number" ? rawPercent : parseInt(String(rawPercent ?? "0"), 10);
  const installPct = Number.isFinite(parsedPercent) ? Math.max(0, Math.min(100, parsedPercent)) : 0;
  const installLabel =
    String(progress?.message || "").trim() ||
    String(progress?.stage || "").trim() ||
    t("servers.overview.install_label_default");

  const rawLastMetric = metrics[metrics.length - 1];
  const metricAgeMs = rawLastMetric?.ts ? Date.now() - rawLastMetric.ts * 1000 : Infinity;
  const metricIsFresh = running && metricAgeMs < 90_000;
  const lastMetric = metricIsFresh ? rawLastMetric : undefined;

  const cpuPct = lastMetric?.cpu_pct ?? 0;
  const memLimit =
    lastMetric?.mem_limit_mb ?? Number(server.limits?.memory_mb ?? server.tariff?.ram_mb ?? 0);
  const memUsed = lastMetric?.mem_used_mb ?? 0;
  const memPct = memLimit > 0 ? Math.min(100, (memUsed / memLimit) * 100) : 0;
  const diskTotalMb = Number(server.disk_total_mb ?? server.limits?.disk_mb ?? server.tariff?.disk_mb ?? 0);
  const diskUsedMb = Number(server.disk_used_mb ?? 0);
  const diskPct = diskTotalMb > 0 ? Math.min(100, (diskUsedMb / diskTotalMb) * 100) : 0;

  const gb = (mb: number) => (mb / 1024).toFixed(2);
  const ramSub =
    memLimit > 0 ? `${gb(memUsed)} / ${gb(memLimit)} GB` : `${memUsed.toFixed(0)} MB`;
  const diskSub =
    diskTotalMb > 0
      ? `${diskUsedMb > 0 ? gb(diskUsedMb) : "—"} / ${gb(diskTotalMb)} GB`
      : t("servers.overview.no_limit");
  const cpuLimit = Number(server.limits?.cpu_cores ?? 0);

  const infoRows: [string, string][] = [
    [t("common.game"), server.game?.name || "—"],
    [
      t("servers.overview.row_version"),
      versions.find((v) => v.id === currentVersionId)?.name || "—",
    ],
    [t("servers.overview.row_map"), running ? currentMap || "—" : "—"],
    [t("common.tariff"), server.tariff?.name || "—"],
    [t("common.location"), server.location?.name || "—"],
    [t("common.created_at"), formatDateTime(server.created_at)],
    [t("servers.overview.row_rented_until"), formatDateTime(server.expires_at)],
    [
      t("servers.overview.row_left"),
      t("servers.overview.days", { days: expired ? 0 : daysLeft }),
    ],
    [t("servers.overview.row_uptime"), running ? server.uptime || "—" : "—"],
  ];

  return (
    <div className="flex flex-col gap-[18px]">
      {provisioning && <ServerInstallConsole serverId={id} active />}

      {manualInstall && !provisioning && (
        <div className="rounded-[14px] border border-[rgba(232,160,60,0.28)] bg-[rgba(232,160,60,0.06)] px-5 py-[18px]">
          <div className="text-[12.5px] font-medium text-[var(--vx-warn)]">
            {t("servers.overview.manual_title")}
          </div>
          <p className="mt-2 text-[12.5px] leading-[1.65] text-[var(--vx-ink-muted)]">
            {t("servers.overview.manual_desc", {
              note: activeVersion?.install_note
                ? `: ${activeVersion.install_note}`
                : "",
            })}
          </p>
          <div className="mt-3.5">
            <Link href={serverPath("", id, "/ftp")} className="inline-flex">
              <Btn>{t("servers.overview.open_ftp")}</Btn>
            </Link>
          </div>
        </div>
      )}

      {isInstallFailed && (
        <div className="rounded-[14px] border border-[rgba(224,122,122,0.28)] bg-[rgba(224,122,122,0.06)] px-5 py-[18px]">
          <div className="text-[12.5px] font-medium text-[var(--vx-danger)]">
            {t("servers.overview.install_failed")}
          </div>
          <pre className="m-0 mt-2.5 font-mono text-[12px] whitespace-pre-wrap text-[var(--vx-danger)]">
            {server.provisioning_error || t("servers.overview.reason_unknown")}
          </pre>
          <div className="mt-3.5 flex flex-wrap gap-2">
            {showReinstall && (
              <Btn
                tone="primary"
                disabled={lifecycleMutation.isPending}
                onClick={() => {
                  if (!confirm(t("servers.overview.reinstall_confirm"))) return;
                  lifecycleMutation.mutate("reinstall");
                }}
              >
                {t("servers.shell.reinstall")}
              </Btn>
            )}
            <Btn
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(server.provisioning_error || "");
                  toast.success(t("servers.overview.error_copied"));
                } catch {
                  toast.error(t("common.copy_failed"));
                }
              }}
            >
              {t("servers.overview.copy_error")}
            </Btn>
          </div>
        </div>
      )}

      <div className="grid grid-cols-2 gap-3.5 xl:grid-cols-4">
        <Tile
          label={t("servers.overview.tile_online")}
          value={`${onlineCount} / ${maxPlayers || "—"}`}
          sub={
            running
              ? t("servers.overview.online_polled")
              : t("servers.overview.online_down")
          }
          pct={maxPlayers > 0 ? (onlineCount / maxPlayers) * 100 : 0}
        />
        <Tile
          label="CPU"
          value={metricsLoading ? "…" : running ? `${cpuPct.toFixed(1)}%` : "—"}
          sub={
            !running
              ? t("servers.overview.server_off")
              : cpuLimit > 0
                ? t("servers.overview.cpu_limit", { cores: cpuLimit })
                : t("servers.overview.no_limit")
          }
          pct={cpuPct}
        />
        <Tile
          label="RAM"
          value={metricsLoading ? "…" : running ? `${memPct.toFixed(1)}%` : "—"}
          sub={
            running
              ? ramSub
              : t("servers.overview.ram_limit", { value: gb(memLimit) })
          }
          pct={memPct}
        />
        <Tile
          label={t("servers.overview.tile_disk")}
          value={diskTotalMb > 0 ? `${diskPct.toFixed(0)}%` : "—"}
          sub={diskSub}
          pct={diskPct}
        />
      </div>

      <div className="grid grid-cols-1 items-start gap-[18px] lg:grid-cols-[minmax(0,1fr)_minmax(0,1.35fr)]">
        <div className="flex flex-col gap-[18px]">
          <Panel title={t("servers.overview.panel_summary")} flush>
            <div className="px-[18px] pt-1.5 pb-3.5">
              {infoRows.map(([k, v]) => (
                <InfoRow key={k} k={k} v={v} />
              ))}
            </div>
          </Panel>

          <Panel title={t("servers.overview.panel_version")}>
            <div className="flex flex-col gap-3">
              {versions.length > 0 && (
                <div className="flex gap-2">
                  <select
                    className={cn(VX_SELECT, "flex-1")}
                    value={effectiveVersionId}
                    onChange={(e) => setSelectedVersionId(e.target.value)}
                    disabled={lifecycleMutation.isPending}
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
                      lifecycleMutation.isPending ||
                      !canSwitchVersion(server) ||
                      !effectiveVersionId ||
                      effectiveVersionId === currentVersionId
                    }
                    title={
                      !canSwitchVersion(server)
                        ? t("servers.overview.switch_version_hint")
                        : undefined
                    }
                    onClick={() => {
                      if (!confirm(t("servers.overview.switch_version_confirm")))
                        return;
                      lifecycleMutation.mutate(`version:${effectiveVersionId}`);
                    }}
                  >
                    {t("servers.overview.switch")}
                  </Btn>
                </div>
              )}

              {isOwner && (
                <div className="flex items-center justify-between gap-3 rounded-[10px] border border-[var(--vx-border)] bg-[var(--vx-elevated)] px-3 py-2.5 text-[12.5px]">
                  <span>{t("servers.overview.auto_restart")}</span>
                  <Toggle
                    label={t("servers.overview.auto_restart")}
                    checked={!!server.auto_start_enabled}
                    disabled={autoStartMutation.isPending || expired}
                    onChange={(next) => autoStartMutation.mutate(next)}
                  />
                </div>
              )}

              {showChangeMap && (
                <Btn onClick={() => setChangeMapOpen(true)} disabled={mapsFolderQuery.isFetching}>
                  {mapsFolderQuery.isFetching
                    ? t("servers.overview.maps_loading")
                    : t("servers.overview.change_map")}
                </Btn>
              )}

              {showUpdate && (
                <Btn
                  onClick={() => lifecycleMutation.mutate("update")}
                  disabled={lifecycleMutation.isPending}
                >
                  {lifecycleMutation.isPending
                    ? t("common.updating")
                    : t("servers.overview.update_steam")}
                </Btn>
              )}

              <div className="flex gap-2">
                {showReinstall && !isInstallFailed && (
                  <Btn
                    className="flex-1"
                    disabled={lifecycleMutation.isPending}
                    onClick={() => {
                      if (!confirm(t("servers.overview.reinstall_confirm"))) return;
                      lifecycleMutation.mutate("reinstall");
                    }}
                  >
                    {t("servers.shell.reinstall")}
                  </Btn>
                )}
                {isOwner && (
                  <Btn tone="primary" className="flex-1" onClick={() => setRenewOpen(true)}>
                    {t("servers.shell.extend_rent")}
                  </Btn>
                )}
              </div>
            </div>
          </Panel>
        </div>

        <div className="flex flex-col gap-[18px]">
          <Panel
            title={t("servers.overview.panel_load")}
            aside={<span className={cn("font-mono text-[11px]", VX_FAINT)}>CPU · RAM</span>}
          >
            {metrics.length === 0 ? (
              metricsLoading ? (
                <VxInlineLoader label={t("servers.overview.metrics_loading")} />
              ) : (
                <EmptyState>{t("servers.overview.no_period_data")}</EmptyState>
              )
            ) : (
              <>
                <svg
                  viewBox={`0 0 ${CHART_W} ${CHART_H}`}
                  preserveAspectRatio="none"
                  className="block h-[150px] w-full"
                >
                  <polyline
                    points={charts.cpu}
                    fill="none"
                    stroke="var(--vx-fg-strong)"
                    strokeWidth={1.6}
                    strokeLinejoin="round"
                  />
                  <polyline
                    points={charts.ram}
                    fill="none"
                    stroke="var(--vx-faint)"
                    strokeWidth={1.6}
                    strokeDasharray="5 4"
                    strokeLinejoin="round"
                  />
                </svg>
                <div
                  className={cn(
                    "mt-2.5 flex justify-between font-mono text-[10.5px]",
                    "text-[var(--vx-faint)]"
                  )}
                >
                  <span>{t("servers.overview.chart_earlier")}</span>
                  <span>{t("servers.overview.chart_legend")}</span>
                  <span>{t("servers.overview.chart_now")}</span>
                </div>
              </>
            )}
          </Panel>

          <Panel
            title={t("servers.overview.panel_players")}
            aside={
              <span className={cn("font-mono text-[11px]", VX_FAINT)}>
                {onlineCount} / {maxPlayers || "—"}
              </span>
            }
            flush
          >
            <div className="px-[18px] pt-1.5 pb-3.5">
              <div
                className={cn(
                  "grid items-center gap-3 py-[9px] text-[10.5px] tracking-[0.08em] uppercase",
                  VX_ROW_LINE,
                  VX_FAINT,
                  showPlayerActions
                    ? "grid-cols-[32px_minmax(0,1fr)_70px_120px]"
                    : "grid-cols-[32px_minmax(0,1fr)_70px]"
                )}
              >
                <span>#</span>
                <span>{t("servers.overview.col_nick")}</span>
                <span>
                  {isMinecraft
                    ? t("servers.overview.col_ping")
                    : t("servers.overview.col_frags")}
                </span>
                {showPlayerActions && (
                  <span className="text-right">{t("common.actions")}</span>
                )}
              </div>

              {playersOnline.length === 0 ? (
                <EmptyState>{t("servers.overview.no_data")}</EmptyState>
              ) : (
                playersOnline.map((p, idx) => (
                  <div
                    key={`${p.name}-${idx}`}
                    className={cn(
                      "grid items-center gap-3 py-2.5 text-[12.5px] last:border-b-0",
                      VX_ROW_LINE,
                      showPlayerActions
                        ? "grid-cols-[32px_minmax(0,1fr)_70px_120px]"
                        : "grid-cols-[32px_minmax(0,1fr)_70px]"
                    )}
                  >
                    <span className={cn("font-mono", VX_FAINT)}>{idx + 1}</span>
                    <span className="truncate">{p.name}</span>
                    <span className={cn("font-mono", VX_MUTED)}>
                      {isMinecraft ? (p.ping ?? "—") : (p.score ?? "—")}
                    </span>
                    {showPlayerActions && (
                      <span className="flex justify-end gap-1.5">
                        <Btn
                          size="sm"
                          tone="ghost"
                          className="h-[26px] rounded-[7px] border-[var(--vx-border-2)] px-2.5 text-[11.5px]"
                          onClick={() =>
                            setPlayerAction({ open: true, type: "kick", name: p.name, reason: "" })
                          }
                        >
                          {t("servers.overview.kick")}
                        </Btn>
                        <Btn
                          size="sm"
                          className="h-[26px] rounded-[7px] border-[rgba(224,122,122,0.3)] bg-transparent px-2.5 text-[11.5px] text-[var(--vx-danger)]"
                          onClick={() =>
                            setPlayerAction({ open: true, type: "ban", name: p.name, reason: "" })
                          }
                        >
                          {t("servers.overview.ban")}
                        </Btn>
                      </span>
                    )}
                  </div>
                ))
              )}
            </div>
          </Panel>

          <Panel
            title={t("servers.overview.panel_backups")}
            flush
            aside={
              <Link href={serverPath("", id, "/copies")} className={cn("text-[12px]", VX_MUTED)}>
                {t("servers.overview.all_backups")}
              </Link>
            }
          >
            <div className="px-[18px] pt-2.5 pb-4">
              {backupsQuery.isLoading ? (
                <div className="py-4">
                  <VxInlineLoader />
                </div>
              ) : latestBackups.length === 0 ? (
                <EmptyState>{t("servers.overview.no_backups")}</EmptyState>
              ) : (
                <div className="mb-3 flex flex-col gap-1.5">
                  {latestBackups.map((b) => (
                    <div
                      key={b.name}
                      className="flex items-center justify-between gap-3 rounded-[10px] border border-[var(--vx-border)] bg-[var(--vx-elevated)] px-3.5 py-2.5"
                    >
                      <span className={cn("truncate text-[12.5px]", VX_CODE)}>{b.name}</span>
                      <span className={cn("shrink-0 text-[12px]", VX_MUTED)}>
                        {formatBackupSize(b.size)}
                      </span>
                    </div>
                  ))}
                  {backupsCount > latestBackups.length && (
                    <div className={cn("px-1 pt-0.5 text-[11.5px]", VX_FAINT)}>
                      {t("servers.overview.and_more", {
                        count: backupsCount - latestBackups.length,
                      })}
                    </div>
                  )}
                </div>
              )}
              <Btn
                className="w-full"
                disabled={createBackup.isPending}
                onClick={() => createBackup.mutate()}
              >
                {createBackup.isPending
                  ? t("servers.overview.creating_backup")
                  : t("servers.overview.create_backup")}
              </Btn>
            </div>
          </Panel>
        </div>
      </div>

      <Dialog open={renewOpen} onOpenChange={setRenewOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("servers.overview.renew_title")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className={cn("text-xs", VX_MUTED)}>
              {t("servers.overview.renew_choose_period")}
            </div>
            <div className="grid grid-cols-2 gap-2">
              {renewalPeriods.map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => setRenewPeriod(p)}
                  className={cn(
                    "flex flex-col items-center gap-0.5 rounded-[9px] border px-3 py-2.5 text-[12.5px] font-medium transition-colors",
                    renewPeriod === p
                      ? "border-[var(--vx-fg-strong)] bg-[var(--vx-fg-strong)] text-[var(--vx-on-fill)]"
                      : "border-[var(--vx-border-2)] bg-[var(--vx-inset)] text-[var(--vx-fg)] hover:border-[var(--vx-border-strong)]"
                  )}
                >
                  <span>{t("servers.overview.days", { days: p })}</span>
                  {renewPreview.data && renewPeriod === p && (
                    <span className="text-[11px] opacity-80">
                      {formatAmount(renewPreview.data.price)} {renewPreview.data.currency}
                    </span>
                  )}
                </button>
              ))}
            </div>
            <Field label={t("servers.overview.promo_label")}>
              <input
                className={VX_INPUT}
                value={renewPromoCode}
                onChange={(e) => setRenewPromoCode(e.target.value)}
                placeholder="VORTANIX10"
              />
            </Field>
          </div>
          <DialogFooter>
            <Btn onClick={() => setRenewOpen(false)}>{t("common.cancel")}</Btn>
            <Btn tone="primary" onClick={() => renewMutation.mutate()} disabled={renewMutation.isPending}>
              {renewMutation.isPending
                ? t("servers.overview.paying")
                : t("servers.overview.renew")}
            </Btn>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={playerAction.open}
        onOpenChange={(open) => setPlayerAction((s) => ({ ...s, open }))}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {playerAction.type === "ban"
                ? t("servers.overview.ban_player")
                : t("servers.overview.kick_player")}
              : {playerAction.name}
            </DialogTitle>
          </DialogHeader>
          <Field label={t("servers.overview.reason_label")}>
            <input
              className={VX_INPUT}
              value={playerAction.reason}
              onChange={(e) => setPlayerAction((s) => ({ ...s, reason: e.target.value }))}
            />
          </Field>
          <DialogFooter>
            <Btn onClick={() => setPlayerAction((s) => ({ ...s, open: false }))}>
              {t("common.cancel")}
            </Btn>
            <Btn
              tone={playerAction.type === "ban" ? "danger" : "primary"}
              onClick={() => {
                const reason = playerAction.reason.trim();
                const cmd = reason
                  ? `${playerAction.type} ${playerAction.name} ${reason}`
                  : `${playerAction.type} ${playerAction.name}`;
                consoleMutation.mutate(cmd);
                setPlayerAction((s) => ({ ...s, open: false }));
              }}
            >
              {playerAction.type === "ban"
                ? t("servers.overview.do_ban")
                : t("servers.overview.do_kick")}
            </Btn>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={changeMapOpen} onOpenChange={setChangeMapOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{t("servers.overview.change_map")}</DialogTitle>
          </DialogHeader>
          {mapsFolderQuery.isLoading ? (
            <VxInlineLoader label={t("servers.overview.maps_list_loading")} />
          ) : (mapsFolderQuery.data?.maps ?? []).length === 0 ? (
            <EmptyState>{t("servers.overview.no_maps")}</EmptyState>
          ) : (
            <div className={cn("max-h-[60vh] overflow-auto rounded-[10px]", VX_CODE)}>
              {(mapsFolderQuery.data?.maps ?? []).map((mapName) => (
                <div
                  key={mapName}
                  className={cn(
                    "flex items-center justify-between gap-3 px-3.5 py-2.5 text-[12.5px] last:border-b-0",
                    VX_ROW_LINE
                  )}
                >
                  <span className="font-mono">{mapName}</span>
                  <Btn
                    size="sm"
                    disabled={
                      mapActionLoading === mapName ||
                      changeMapMutation.isPending ||
                      currentMap === mapName
                    }
                    onClick={() => {
                      setMapActionLoading(mapName);
                      changeMapMutation.mutate(mapName, {
                        onSettled: () => setMapActionLoading(""),
                      });
                    }}
                  >
                    {currentMap === mapName
                      ? t("servers.overview.map_active")
                      : mapActionLoading === mapName
                        ? t("servers.overview.map_changing")
                        : t("servers.overview.switch")}
                  </Btn>
                </div>
              ))}
            </div>
          )}
        </DialogContent>
      </Dialog>

    </div>
  );
}
