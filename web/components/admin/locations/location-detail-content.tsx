"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Copy } from "lucide-react";
import {
  LocationMetricsChart,
  type MetricPoint,
} from "@/components/admin/locations/location-metrics-chart";
import { LocationAgentSetupCard } from "@/components/admin/locations/location-agent-setup-card";
import {
  LocationCapacityCard,
  fmtMB,
} from "@/components/admin/locations/location-capacity-card";
import { LocationIPsCard } from "@/components/admin/locations/location-ips-card";
import { MaintenanceDialog } from "@/components/admin/locations/maintenance-dialog";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import {
  fetchAdminLocation,
  fetchNodeCapacity,
  pullAdminLocationDaemon,
  resetAdminLocationHostKey,
  testAdminLocationSSH,
  type AdminLocationListItem,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import { localeTag, type TranslateFn } from "@/lib/i18n";

function readMetricValue(metric: unknown): string {
  if (metric == null) return "—";
  if (typeof metric === "string") return metric.trim() || "—";
  if (typeof metric === "number") return String(metric);
  if (typeof metric === "object") {
    const item = metric as {
      text_value?: unknown;
      textValue?: unknown;
      value?: unknown;
    };
    const text = item.text_value ?? item.textValue;
    if (typeof text === "string" && text.trim()) return text;
    if (item.value != null && item.value !== "") return String(item.value);
  }
  return "—";
}

function readMetricNumber(metric: unknown): number | null {
  const n = Number.parseFloat(readMetricValue(metric));
  return Number.isFinite(n) && n >= 0 ? n : null;
}

type ServiceTone = "emerald" | "rose" | "amber" | "muted";

function serviceState(
  t: TranslateFn,
  state: string,
  awaitingSync = false
): { label: string; tone: ServiceTone } {
  switch (state) {
    case "active":
      return { label: t("admin.location.svc.active"), tone: "emerald" };
    case "inactive":
    case "failed":
      return { label: t("admin.location.svc.stopped"), tone: "rose" };
    case "unconfigured":
      return {
        label: t("admin.locations.ssh_not_configured"),
        tone: "amber",
      };
    case "ssh_failed":
      return { label: t("admin.location.svc.ssh_error"), tone: "rose" };
    case "error":
      return { label: t("common.error"), tone: "rose" };
    default:
      return awaitingSync
        ? { label: t("admin.location.svc.syncing"), tone: "amber" }
        : { label: t("common.unknown"), tone: "muted" };
  }
}

const TONE_TEXT: Record<ServiceTone, string> = {
  emerald: "text-emerald-500",
  rose: "text-rose-500",
  amber: "text-amber-500",
  muted: "text-muted-foreground",
};

const TONE_DOT: Record<ServiceTone, string> = {
  emerald: "bg-emerald-500",
  rose: "bg-rose-500",
  amber: "bg-amber-500",
  muted: "bg-muted-foreground",
};

type LocationRecord = Record<string, unknown>;

const LOCATION_SERVICES = [
  { unit: "docker", label: "Docker" },
  { unit: "mysql", label: "MySQL" },
  { unit: "vortanix-sftp", label: "SFTP" },
  { unit: "vortanix-agent", label: "Vortanix Agent" },
] as const;

const HARDWARE_ROWS = [
  ["admin.location.info.os", "os_info", "text"],
  ["admin.location.info.cpu", "cpu_model", "text"],
  ["admin.location.info.ram", "ram_total", "bytes"],
  ["admin.location.info.disk_total", "disk_total", "bytes"],
] as const;

function formatMetric(
  t: TranslateFn,
  kind: "text" | "bytes" | "uptime",
  raw: string
): string {
  if (raw === "—" || kind === "text") return raw;
  const n = Number.parseFloat(raw);
  if (!Number.isFinite(n) || n < 0) return raw;
  if (kind === "bytes") {
    const gb = n / 1024 ** 3;
    if (gb >= 1) {
      const value = gb.toLocaleString(localeTag(), {
        maximumFractionDigits: gb >= 100 ? 0 : 1,
      });
      return `${value} ${t("admin.infra.unit_gb")}`;
    }
    return `${Math.round(n / 1024 ** 2)} ${t("admin.infra.unit_mb")}`;
  }
  const total = Math.floor(n);
  const d = Math.floor(total / 86400);
  const h = Math.floor((total % 86400) / 3600);
  const m = Math.floor((total % 3600) / 60);
  if (d > 0) return t("admin.location.uptime.days", { d, h });
  if (h > 0) return t("admin.location.uptime.hours", { h, m });
  return t("admin.location.uptime.minutes", { m });
}

function pointValue(point: MetricPoint | undefined): number | null {
  if (!point) return null;
  const value = Number(point.v ?? point.value);
  return Number.isFinite(value) ? value : null;
}

function lastValue(points: MetricPoint[]): number | null {
  return pointValue(points[points.length - 1]);
}

function averageValue(points: MetricPoint[]): number | null {
  const values = points
    .map(pointValue)
    .filter((v): v is number => v !== null);
  if (values.length === 0) return null;
  return values.reduce((sum, v) => sum + v, 0) / values.length;
}

function formatPercent(value: number | null): string {
  if (value === null) return "—";
  return `${value.toLocaleString(localeTag(), { maximumFractionDigits: value < 10 ? 1 : 0 })}%`;
}

type MysqlInstance = {
  key?: string;
  name?: string;
  container?: string;
  port?: number | string;
  root_password?: string;
};

export function LocationDetailContent() {
  const t = useT();
  const params = useParams();
  const id = String(params?.id ?? "");
  const queryClient = useQueryClient();
  const autoSyncStarted = useRef(false);
  const [maintenanceOpen, setMaintenanceOpen] = useState(false);

  const { data, isLoading, isFetching } = useQuery({
    queryKey: queryKeys.adminLocation(id),
    queryFn: () => fetchAdminLocation(id),
    enabled: !!id,
    refetchInterval: (query) =>
      query.state.data?.sync_pending || query.state.data?.metrics_stale
        ? 4000
        : false,
  });

  const capacityQuery = useQuery({
    queryKey: ["node-capacity", id],
    queryFn: () => fetchNodeCapacity(id),
    enabled: !!id,
  });

  const pullMut = useMutation({
    mutationFn: () => pullAdminLocationDaemon(id),
    onSuccess: () => {
      toast.success(t("admin.location.sync_started"));
      void queryClient.invalidateQueries({
        queryKey: queryKeys.adminLocation(id),
      });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const sshTestMut = useMutation({
    mutationFn: () => testAdminLocationSSH(id),
    onSuccess: (res) => {
      if (res.ok) {
        toast.success(res.output ? `SSH OK: ${res.output}` : "SSH OK");
        void queryClient.invalidateQueries({ queryKey: queryKeys.adminLocation(id) });
      } else {
        toast.error(res.error || t("admin.location.svc.ssh_error"));
      }
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const hostKeyResetMut = useMutation({
    mutationFn: () => resetAdminLocationHostKey(id),
    onSuccess: () => {
      toast.success(t("admin.location.host_key_reset_done"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminLocation(id) });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const metricsStale = Boolean(data?.metrics_stale ?? data?.sync_pending);
  const sshHost = String(
    (data?.location as LocationRecord | undefined)?.ssh_host ?? ""
  );

  useEffect(() => {
    autoSyncStarted.current = false;
  }, [id]);

  useEffect(() => {
    if (!id || !sshHost || autoSyncStarted.current || !metricsStale) return;
    autoSyncStarted.current = true;
    pullMut.mutate();
  }, [id, sshHost, metricsStale, pullMut]);

  if (isLoading) {
    return (
      <PageShell variant="admin">
        <div className="w-full space-y-4">
          <Skeleton className="h-20 w-full rounded-2xl" />
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-24 rounded-xl" />
            ))}
          </div>
          <Skeleton className="h-16 w-full rounded-xl" />
          <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(320px,400px)]">
            <Skeleton className="h-[520px] rounded-2xl" />
            <Skeleton className="h-[520px] rounded-2xl" />
          </div>
        </div>
      </PageShell>
    );
  }

  const location = (data?.location ?? null) as LocationRecord | null;
  if (!location) {
    return (
      <PageShell variant="admin">
        <p className="py-20 text-center text-muted-foreground">
          {t("admin.location.not_found")}
        </p>
      </PageShell>
    );
  }

  const serverMetrics = (data?.serverMetrics ?? {}) as Record<string, unknown>;
  const serviceStatuses = data?.serviceStatuses ?? {};
  const cpuMetrics = (data?.metrics?.cpu_usage ??
    (data as { cpuMetrics?: MetricPoint[] })?.cpuMetrics ??
    []) as MetricPoint[];
  const ramMetrics = (data?.metrics?.ram_usage ??
    (data as { ramMetrics?: MetricPoint[] })?.ramMetrics ??
    []) as MetricPoint[];

  const mysqlInstances = (
    Array.isArray(location.mysql_instances) ? location.mysql_instances : []
  ) as MysqlInstance[];
  const mysqlHost = String(
    location.mysql_host || location.ip_address || sshHost || "—"
  );

  const awaitingSync = metricsStale || isFetching || pullMut.isPending;
  const pmaHost = String(location.ip_address || location.ssh_host || "");
  const pmaPort = Number(location.phpmyadmin_port ?? 0);
  const pmaUrl = pmaHost && pmaPort > 0 ? `https://${pmaHost}:${pmaPort}` : "";
  const daemon = (data?.daemon ?? {}) as Record<string, unknown>;
  const isOnline = Boolean(daemon.is_online ?? daemon.status === "online");
  const isActive = Boolean(location.is_active);
  const sshUser = String(location.ssh_user || "");
  const hostKey = String(location.ssh_host_key || "");
  const sshPort = Number(location.ssh_port || 22);
  const playersIp = String(location.ip_address || "");
  const maintenance = Boolean(location.maintenance_mode);
  const maintenanceReason = String(location.maintenance_reason || "");
  const maintenanceUntil =
    typeof location.maintenance_until === "string" ? location.maintenance_until : "";

  const capacity = capacityQuery.data;
  const cpuNow = lastValue(cpuMetrics);
  const ramNow = lastValue(ramMetrics);
  const cpuAvg = averageValue(cpuMetrics);
  const ramAvg = averageValue(ramMetrics);
  const diskUsed = readMetricNumber(serverMetrics.disk_used);
  const diskTotal = readMetricNumber(serverMetrics.disk_total);
  const diskPercent =
    diskUsed !== null && diskTotal ? (diskUsed / diskTotal) * 100 : null;
  const uptime = formatMetric(t, "uptime", readMetricValue(serverMetrics.uptime));

  const maintenanceTarget: AdminLocationListItem = {
    id,
    name: String(location.name ?? ""),
    country: String(location.country ?? ""),
    code: String(location.code ?? ""),
    is_active: isActive,
    servers_count: capacity?.servers ?? 0,
    tariffs_count: 0,
    maintenance_mode: maintenance,
    maintenance_reason: maintenanceReason,
    maintenance_until: maintenanceUntil || null,
  };

  async function copyPlayersIp() {
    try {
      await navigator.clipboard.writeText(playersIp);
      toast.success(t("common.copied"));
    } catch {
      toast.error(t("common.copy_failed"));
    }
  }

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-4">
        <header className="flex flex-wrap items-start justify-between gap-x-6 gap-y-4">
          <div className="min-w-0 space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <Badge tone="muted" mono>
                {String(location.code ?? "—")}
              </Badge>
              <Badge tone={isActive ? "emerald" : "muted"}>
                {isActive
                  ? t("admin.locations.active")
                  : t("admin.locations.inactive")}
              </Badge>
              <Badge tone={isOnline ? "emerald" : "rose"}>
                {isOnline
                  ? t("admin.location.agent_online")
                  : t("admin.location.agent_offline")}
              </Badge>
              {maintenance && (
                <Badge tone="amber">{t("admin.locations.maintenance_active")}</Badge>
              )}
            </div>
            <h1 className="truncate text-[26px] leading-tight font-bold tracking-tight">
              {String(location.name ?? "")}
            </h1>
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
              <span>
                {[location.city, location.country].filter(Boolean).join(", ") ||
                  "—"}
              </span>
              <span className="text-border">·</span>
              <span className="inline-flex items-center gap-1.5">
                {t("admin.location.ip_players")}:
                <span className="font-mono text-foreground">
                  {playersIp || t("admin.location.not_set")}
                </span>
                {playersIp && (
                  <button
                    type="button"
                    title={t("common.copy")}
                    className="rounded p-0.5 transition-colors hover:text-foreground"
                    onClick={() => void copyPlayersIp()}
                  >
                    <Copy className="size-3.5" />
                  </button>
                )}
              </span>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Button variant="ghost" asChild className="h-9 text-[13px]">
              <Link href="/admin/locations">← {t("common.back")}</Link>
            </Button>
            <Button
              variant="outline"
              className={cn(
                "h-9 text-[13px]",
                maintenance && "border-amber-500/60 text-amber-600 dark:text-amber-400"
              )}
              onClick={() => setMaintenanceOpen(true)}
            >
              {maintenance
                ? t("admin.locations.maintenance_active")
                : t("admin.locations.maintenance")}
            </Button>
            <Button variant="outline" asChild className="h-9 text-[13px]">
              <Link href={`/admin/locations/${id}/setup`}>
                {t("admin.locations.ssh_setup")}
              </Link>
            </Button>
            <Button variant="outline" asChild className="h-9 text-[13px]">
              <Link href={`/admin/images?node=${id}`}>{t("admin.images.title")}</Link>
            </Button>
            <Button variant="outline" asChild className="h-9 text-[13px]">
              <Link href={`/admin/locations/${id}/edit`}>
                {t("common.edit")}
              </Link>
            </Button>
            {sshHost && (
              <Button
                className="h-9 text-[13px]"
                disabled={pullMut.isPending}
                onClick={() => pullMut.mutate()}
              >
                {pullMut.isPending ? t("common.updating") : t("common.refresh")}
              </Button>
            )}
          </div>
        </header>

        {maintenance && (
          <Banner tone="amber">
            {t("admin.locations.maintenance_active")}
            {maintenanceReason ? `: ${maintenanceReason}` : ""}
            {maintenanceUntil
              ? ` · ${t("admin.location.maintenance_until", {
                  date: new Date(maintenanceUntil).toLocaleString(localeTag()),
                })}`
              : ""}
          </Banner>
        )}

        {awaitingSync && sshHost && (
          <Banner tone="amber">{t("admin.location.sync_banner")}</Banner>
        )}

        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-6">
          <StatTile
            label="CPU"
            value={formatPercent(cpuNow)}
            sub={
              cpuAvg !== null
                ? t("admin.location.kpi.avg", { value: formatPercent(cpuAvg) })
                : undefined
            }
            percent={cpuNow}
          />
          <StatTile
            label="RAM"
            value={formatPercent(ramNow)}
            sub={
              ramAvg !== null
                ? t("admin.location.kpi.avg", { value: formatPercent(ramAvg) })
                : undefined
            }
            percent={ramNow}
          />
          <StatTile
            label={t("admin.location.kpi.disk")}
            value={formatPercent(diskPercent)}
            sub={
              diskUsed !== null && diskTotal !== null
                ? t("admin.location.kpi.disk_of", {
                    used: formatMetric(t, "bytes", String(diskUsed)),
                    total: formatMetric(t, "bytes", String(diskTotal)),
                  })
                : undefined
            }
            percent={diskPercent}
          />
          <StatTile
            label={t("admin.location.kpi.servers")}
            value={capacity ? String(capacity.servers) : "—"}
            sub={
              capacity
                ? capacity.max_servers !== null
                  ? t("admin.location.kpi.servers_limit", { max: capacity.max_servers })
                  : t("admin.capacity.no_limit")
                : undefined
            }
            percent={
              capacity && capacity.max_servers
                ? (capacity.servers / capacity.max_servers) * 100
                : null
            }
          />
          <StatTile
            label={t("admin.capacity.free_for_new")}
            value={capacity?.metrics_known ? fmtMB(t, capacity.free_ram_mb) : "—"}
            sub={
              capacity?.metrics_known
                ? capacity.free_ram_mb < 0
                  ? t("admin.capacity.oversubscribed")
                  : t("admin.location.kpi.ram_of", {
                      total: fmtMB(t, capacity.node_ram_mb),
                    })
                : undefined
            }
            tone={capacity?.metrics_known && capacity.free_ram_mb < 0 ? "rose" : undefined}
          />
          <StatTile label={t("admin.location.info.uptime")} value={uptime} />
        </div>

        <section className="grid gap-px overflow-hidden rounded-xl border bg-border sm:grid-cols-2 xl:grid-cols-4">
          {LOCATION_SERVICES.map(({ unit, label }) => {
            const svc = serviceStatuses[unit] ?? { state: "unknown", error: null };
            const state = serviceState(
              t,
              String(svc.state),
              awaitingSync && svc.state === "unknown"
            );
            return (
              <div key={unit} className="flex min-w-0 flex-col gap-1 bg-card px-4 py-3">
                <div className="flex items-center justify-between gap-3">
                  <span className="truncate text-[13px] font-medium">{label}</span>
                  <span
                    className={cn(
                      "inline-flex shrink-0 items-center gap-1.5 text-xs",
                      TONE_TEXT[state.tone]
                    )}
                  >
                    <span className={cn("size-1.5 rounded-full", TONE_DOT[state.tone])} />
                    {state.label}
                  </span>
                </div>
                {svc.error && (
                  <p className="line-clamp-2 text-xs text-destructive" title={svc.error}>
                    {svc.error}
                  </p>
                )}
              </div>
            );
          })}
        </section>

        <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(320px,400px)]">
          <div className="flex min-w-0 flex-col gap-4">
            <Card title={t("admin.location.load")} hint={t("admin.agent.last_48h")}>
              <div className="grid gap-6 md:grid-cols-2">
                <ChartBlock
                  label={t("admin.location.cpu_load")}
                  now={cpuNow}
                  points={cpuMetrics}
                  color="var(--chart-1)"
                />
                <ChartBlock
                  label={t("admin.location.ram_load")}
                  now={ramNow}
                  points={ramMetrics}
                  color="var(--chart-2)"
                />
              </div>
            </Card>

            <LocationCapacityCard locationId={id} />

            <LocationIPsCard locationId={id} />
          </div>

          <div className="flex min-w-0 flex-col gap-4">
            <Card title={t("admin.location.server_info")}>
              <div className="flex flex-col">
                {HARDWARE_ROWS.map(([labelKey, key, kind]) => (
                  <Row key={key} label={t(labelKey)}>
                    {formatMetric(t, kind, readMetricValue(serverMetrics[key]))}
                  </Row>
                ))}
              </div>
            </Card>

            <Card title={t("admin.location.ssh_access")}>
              <div className="flex flex-col">
                <Row label={t("admin.location.ssh_host")}>
                  {sshHost || t("admin.location.not_set")}
                </Row>
                <Row label={t("admin.location.ssh_user")}>{sshUser || "—"}</Row>
                <Row label={t("admin.location.port")}>{sshPort}</Row>
                <Row label={t("common.password")}>
                  {location.ssh_password_set ? "••••••••" : t("admin.location.not_set")}
                </Row>
                <Row label={t("admin.location.host_key")}>
                  {hostKey ? (
                    <span className="font-mono text-[11px] break-all">{hostKey}</span>
                  ) : (
                    t("admin.location.host_key_empty")
                  )}
                </Row>
              </div>
              {sshHost && sshUser ? (
                <div className="flex flex-col gap-2.5">
                  <pre className="overflow-x-auto rounded-lg border bg-muted/30 px-3 py-2.5 font-mono text-xs text-muted-foreground">
                    ssh {sshUser}@{sshHost} -p {sshPort}
                  </pre>
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-8 w-fit text-xs"
                    disabled={sshTestMut.isPending}
                    onClick={() => sshTestMut.mutate()}
                  >
                    {sshTestMut.isPending
                      ? t("admin.settings.storage.testing")
                      : t("admin.location.test_ssh")}
                  </Button>
                  {hostKey && (
                    <div className="flex flex-col gap-1.5">
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-8 w-fit text-xs"
                        disabled={hostKeyResetMut.isPending}
                        onClick={() => hostKeyResetMut.mutate()}
                      >
                        {t("admin.location.host_key_reset")}
                      </Button>
                      <p className="text-[11px] text-muted-foreground">
                        {t("admin.location.host_key_hint")}
                      </p>
                    </div>
                  )}
                </div>
              ) : (
                <p className="text-xs text-amber-500">
                  {t("admin.locations.ssh_not_configured")} —{" "}
                  <Link
                    href={`/admin/locations/${id}/edit`}
                    className="underline underline-offset-4"
                  >
                    {t("admin.location.add_in_settings")}
                  </Link>
                </p>
              )}
            </Card>

            <Card
              title={t("admin.location.mysql_title")}
              action={
                pmaUrl ? (
                  <a
                    href={pmaUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-xs text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline"
                  >
                    phpMyAdmin ↗
                  </a>
                ) : null
              }
            >
              {mysqlInstances.length === 0 ? (
                <p className="text-xs text-muted-foreground">
                  {t("admin.location.mysql_empty")}
                </p>
              ) : (
                <div className="flex flex-col gap-3">
                  {mysqlInstances.map((inst, index) => (
                    <MysqlInstanceRow
                      key={inst.key || inst.container || index}
                      instance={inst}
                      host={mysqlHost}
                    />
                  ))}
                </div>
              )}
            </Card>
          </div>
        </div>

        <LocationAgentSetupCard
          locationId={id}
          agentToken=""
        />
      </div>

      {maintenanceOpen && (
        <MaintenanceDialog
          location={maintenanceTarget}
          open
          onOpenChange={(open) => {
            if (open) return;
            setMaintenanceOpen(false);
            void queryClient.invalidateQueries({
              queryKey: queryKeys.adminLocation(id),
            });
          }}
        />
      )}
    </PageShell>
  );
}

const BADGE_TONE: Record<ServiceTone, string> = {
  emerald: "border-emerald-500/40 text-emerald-600 dark:text-emerald-400",
  rose: "border-rose-500/40 text-rose-600 dark:text-rose-400",
  amber: "border-amber-500/50 text-amber-600 dark:text-amber-400",
  muted: "text-muted-foreground",
};

function Badge({
  tone,
  mono,
  children,
}: {
  tone: ServiceTone;
  mono?: boolean;
  children: React.ReactNode;
}) {
  return (
    <span
      className={cn(
        "rounded-md border px-2 py-0.5 text-xs",
        mono && "font-mono",
        BADGE_TONE[tone]
      )}
    >
      {children}
    </span>
  );
}

function Banner({
  tone,
  children,
}: {
  tone: "amber";
  children: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "flex items-center gap-3 rounded-xl border px-4 py-3",
        tone === "amber" && "border-amber-500/40 bg-amber-500/5"
      )}
    >
      <span className="size-1.5 shrink-0 rounded-full bg-amber-500" />
      <p className="text-[13px] leading-relaxed text-amber-600 dark:text-amber-200/90">
        {children}
      </p>
    </div>
  );
}

function barTone(percent: number): string {
  if (percent >= 90) return "bg-rose-500";
  if (percent >= 70) return "bg-amber-500";
  return "bg-emerald-500";
}

function StatTile({
  label,
  value,
  sub,
  percent,
  tone,
}: {
  label: string;
  value: string;
  sub?: string;
  percent?: number | null;
  tone?: "rose";
}) {
  const width =
    percent === undefined || percent === null
      ? null
      : Math.max(0, Math.min(100, percent));
  return (
    <div className="flex min-w-0 flex-col gap-1 rounded-xl border bg-card px-4 py-3.5">
      <span className="truncate font-mono text-[10px] tracking-wider text-muted-foreground uppercase">
        {label}
      </span>
      <span
        className={cn(
          "truncate text-xl leading-tight font-semibold tabular-nums",
          tone === "rose" && "text-rose-500"
        )}
        title={value}
      >
        {value}
      </span>
      <span className="truncate text-xs text-muted-foreground" title={sub}>
        {sub ?? " "}
      </span>
      <div
        className={cn(
          "mt-1.5 h-1 overflow-hidden rounded-full",
          width !== null && "bg-muted"
        )}
      >
        {width !== null && (
          <div
            className={cn("h-full rounded-full transition-[width]", barTone(width))}
            style={{ width: `${width}%` }}
          />
        )}
      </div>
    </div>
  );
}

function ChartBlock({
  label,
  now,
  points,
  color,
}: {
  label: string;
  now: number | null;
  points: MetricPoint[];
  color: string;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-[13px] text-muted-foreground">{label}</span>
        <span className="font-mono text-[13px] tabular-nums">{formatPercent(now)}</span>
      </div>
      <LocationMetricsChart points={points} color={color} height={180} />
    </div>
  );
}

function Card({
  title,
  hint,
  action,
  children,
}: {
  title: string;
  hint?: string;
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-4 rounded-2xl border bg-card px-5 py-5 sm:px-6">
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <h2 className="text-[15px] leading-none font-semibold">{title}</h2>
        {hint && (
          <span className="font-mono text-xs text-muted-foreground">{hint}</span>
        )}
        {action}
      </div>
      {children}
    </section>
  );
}

function MysqlInstanceRow({
  instance,
  host,
}: {
  instance: MysqlInstance;
  host: string;
}) {
  const t = useT();
  const [shown, setShown] = useState(false);
  const password = String(instance.root_password ?? "");
  const title = instance.name || instance.container || instance.key || "MySQL";

  async function copyPassword() {
    try {
      await navigator.clipboard.writeText(password);
      toast.success(t("admin.location.mysql_copied"));
    } catch {
      toast.error(t("common.copy_failed"));
    }
  }

  return (
    <div className="flex flex-col gap-1 rounded-xl border bg-muted/20 px-3.5 py-3">
      <div className="flex items-center justify-between gap-3">
        <span className="truncate text-[13px] font-medium">{title}</span>
        <span className="shrink-0 font-mono text-xs text-muted-foreground">
          :{Number(instance.port || 3306)}
        </span>
      </div>
      <Row label="Host">{host}</Row>
      <Row label="User">root</Row>
      <Row label={t("common.password")}>
        {password
          ? shown
            ? password
            : "••••••••"
          : t("admin.location.not_set")}
      </Row>
      {password && (
        <div className="mt-1 flex flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            className="h-7 text-xs"
            onClick={() => setShown((v) => !v)}
          >
            {shown ? t("admin.location.mysql_hide") : t("admin.location.mysql_show")}
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="h-7 text-xs"
            onClick={() => void copyPassword()}
          >
            {t("admin.location.mysql_copy")}
          </Button>
        </div>
      )}
    </div>
  );
}

function Row({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-baseline justify-between gap-4 border-b py-2 text-[13px] last:border-0">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className="min-w-0 text-right font-mono break-words">{children}</span>
    </div>
  );
}
