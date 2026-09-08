"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  LocationMetricsChart,
  type MetricPoint,
} from "@/components/admin/locations/location-metrics-chart";
import { LocationAgentSetupCard } from "@/components/admin/locations/location-agent-setup-card";
import { LocationCapacityCard } from "@/components/admin/locations/location-capacity-card";
import { LocationIPsCard } from "@/components/admin/locations/location-ips-card";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import {
  fetchAdminLocation,
  pullAdminLocationDaemon,
  testAdminLocationSSH,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

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

type ServiceTone = "emerald" | "rose" | "amber" | "muted";

// t приходит параметром: функция модульная, а язык меняется на лету —
// захваченный при импорте перевод застыл бы на языке загрузки.
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

// Подписи храним ключами: список читается на уровне модуля, и готовый текст
// застыл бы на языке, который стоял в момент загрузки страницы.
const SERVER_INFO_ROWS = [
  ["admin.location.info.os", "os_info"],
  ["admin.location.info.cpu", "cpu_model"],
  ["admin.location.info.ram", "ram_total"],
  ["admin.location.info.disk_total", "disk_total"],
  ["admin.location.info.disk_used", "disk_used"],
  ["admin.location.info.disk_free", "disk_available"],
  ["admin.location.info.uptime", "uptime"],
] as const;

export function LocationDetailContent() {
  const t = useT();
  const params = useParams();
  const id = String(params?.id ?? "");
  const queryClient = useQueryClient();
  const autoSyncStarted = useRef(false);

  const { data, isLoading, isFetching } = useQuery({
    queryKey: queryKeys.adminLocation(id),
    queryFn: () => fetchAdminLocation(id),
    enabled: !!id,
    refetchInterval: (query) =>
      query.state.data?.sync_pending || query.state.data?.metrics_stale
        ? 4000
        : false,
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
      if (res.ok) toast.success(res.output ? `SSH OK: ${res.output}` : "SSH OK");
      else toast.error(res.error || t("admin.location.svc.ssh_error"));
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
          <Skeleton className="h-16 w-full rounded-2xl" />
          <Skeleton className="h-24 w-full rounded-2xl" />
          <Skeleton className="h-96 w-full rounded-2xl" />
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

  const ipPool = Array.isArray(location.ip_pool)
    ? (location.ip_pool as string[])
    : [];
  const awaitingSync = metricsStale || isFetching || pullMut.isPending;
  const pmaHost = String(location.ip_address || location.ssh_host || "");
  const pmaPort = Number(location.phpmyadmin_port ?? 0);
  const pmaUrl = pmaHost && pmaPort > 0 ? `http://${pmaHost}:${pmaPort}` : "";
  const daemon = (data?.daemon ?? {}) as Record<string, unknown>;
  const isOnline = Boolean(daemon.is_online ?? daemon.status === "online");
  const isActive = Boolean(location.is_active);
  const sshUser = String(location.ssh_user || "");
  const sshPort = Number(location.ssh_port || 22);

  const summary = [
    {
      label: t("common.status"),
      value: `${isActive ? t("admin.locations.active") : t("admin.locations.inactive")} / ${isOnline ? t("admin.location.online_lc") : t("admin.location.offline_lc")}`,
      tone: isOnline ? undefined : ("amber" as const),
    },
    {
      label: t("admin.location.placement"),
      value: String(location.city || "—"),
    },
    {
      label: t("admin.location.ip_players"),
      value: String(location.ip_address || t("admin.location.not_set")),
    },
    {
      label: t("admin.location.ssh_host"),
      value: String(location.ssh_host || t("admin.location.not_set")),
    },
    {
      label: t("admin.location.sort_order"),
      value: String(location.sort_order ?? 0),
    },
    {
      label: t("admin.location.ip_pool"),
      value: ipPool.length ? ipPool.join(", ") : "—",
    },
  ];

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-4">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <div className="flex flex-wrap items-center gap-2.5">
              <span className="rounded-md border px-2 py-0.5 font-mono text-xs">
                {String(location.code ?? "—")}
              </span>
              <span
                className={cn(
                  "rounded-md border px-2 py-0.5 text-xs",
                  isActive
                    ? "border-emerald-500/40 text-emerald-500"
                    : "text-muted-foreground"
                )}
              >
                {isActive
                  ? t("admin.locations.active")
                  : t("admin.locations.inactive")}
              </span>
              <span
                className={cn(
                  "rounded-md border px-2 py-0.5 text-xs",
                  isOnline
                    ? "border-emerald-500/40 text-emerald-500"
                    : "border-rose-500/40 text-rose-500"
                )}
              >
                {isOnline
                  ? t("admin.location.agent_online")
                  : t("admin.location.agent_offline")}
              </span>
            </div>
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {String(location.name ?? "")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {[location.city, location.country].filter(Boolean).join(", ") ||
                "—"}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2.5">
            <Button variant="outline" asChild className="h-[38px] text-[13px]">
              <Link href="/admin/locations">← {t("common.back")}</Link>
            </Button>
            <Button variant="outline" asChild className="h-[38px] text-[13px]">
              <Link href={`/admin/locations/${id}/setup`}>
                {t("admin.locations.ssh_setup")}
              </Link>
            </Button>
            <Button variant="outline" asChild className="h-[38px] text-[13px]">
              <Link href={`/admin/locations/${id}/edit`}>
                {t("common.edit")}
              </Link>
            </Button>
            {sshHost && (
              <Button
                className="h-[38px] text-[13px]"
                disabled={pullMut.isPending}
                onClick={() => pullMut.mutate()}
              >
                {pullMut.isPending
                  ? t("common.updating")
                  : t("common.refresh")}
              </Button>
            )}
          </div>
        </div>

        {awaitingSync && sshHost && (
          <div className="flex flex-wrap items-center gap-3.5 rounded-xl border border-amber-500/40 bg-amber-500/5 px-4 py-3.5">
            <span className="size-1.5 shrink-0 rounded-full bg-amber-500" />
            <p className="flex-1 basis-[320px] text-[13px] leading-relaxed text-amber-600 dark:text-amber-200/90">
              {t("admin.location.sync_banner")}
            </p>
            <Button
              variant="outline"
              size="sm"
              className="h-8 border-amber-500/40 text-xs"
              disabled={pullMut.isPending}
              onClick={() => pullMut.mutate()}
            >
              {t("common.refresh")}
            </Button>
          </div>
        )}

        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
          {summary.map((item) => (
            <div
              key={item.label}
              className="flex flex-col gap-2 rounded-xl border bg-card px-4 py-3.5"
            >
              <span className="font-mono text-[10px] tracking-wider text-muted-foreground uppercase">
                {item.label}
              </span>
              <span
                className={cn(
                  "truncate font-mono text-[13px]",
                  item.tone === "amber" && "text-amber-500"
                )}
                title={item.value}
              >
                {item.value}
              </span>
            </div>
          ))}
        </div>

        <LocationAgentSetupCard
          locationId={id}
          agentToken={String(location.agent_token ?? "")}
        />

        <LocationCapacityCard locationId={id} />

        <LocationIPsCard locationId={id} />

        <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)]">
          <Card title={t("admin.location.server_info")}>
            <div className="flex flex-col">
              {SERVER_INFO_ROWS.map(([labelKey, key]) => (
                <div
                  key={key}
                  className="flex items-baseline justify-between gap-4 border-b py-2.5 text-[13px] last:border-0"
                >
                  <span className="text-muted-foreground">{t(labelKey)}</span>
                  <span className="text-right font-mono">
                    {readMetricValue(serverMetrics[key])}
                  </span>
                </div>
              ))}
            </div>
          </Card>

          <div className="flex flex-col gap-4">
            <Card title={t("admin.location.ssh_access")}>
              <Row label={t("admin.location.ssh_user")}>{sshUser || "—"}</Row>
              <Row label={t("common.password")}>
                {location.ssh_password
                  ? "••••••••"
                  : t("admin.location.not_set")}
              </Row>
              <Row label={t("admin.location.port")}>{sshPort}</Row>
              {sshHost && sshUser ? (
                <>
                  <pre className="overflow-x-auto rounded-lg border bg-muted/30 px-3.5 py-3 font-mono text-xs text-muted-foreground">
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
                </>
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
                    className="border-b text-xs text-muted-foreground transition-colors hover:text-foreground"
                  >
                    phpMyAdmin
                  </a>
                ) : null
              }
            >
              <Row label="Host">
                {String(location.mysql_host || location.ip_address || "—")}
              </Row>
              <Row label="Port / User">
                {Number(location.mysql_port || 3306)} /{" "}
                {String(location.mysql_root_username || "—")}
              </Row>
              <Row label="Password">
                {String(location.mysql_root_password_decrypted || "••••••••")}
              </Row>
            </Card>
          </div>
        </div>

        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {LOCATION_SERVICES.map(({ unit, label }) => {
            const svc = serviceStatuses[unit] ?? { state: "unknown", error: null };
            const state = serviceState(
              t,
              String(svc.state),
              awaitingSync && svc.state === "unknown"
            );
            return (
              <div
                key={unit}
                className="flex flex-col gap-2.5 rounded-xl border bg-card px-5 py-4"
              >
                <span className="font-mono text-[10px] tracking-wider text-muted-foreground uppercase">
                  {label}
                </span>
                <span
                  className={cn(
                    "inline-flex items-center gap-2 text-[13px]",
                    TONE_TEXT[state.tone]
                  )}
                >
                  <span
                    className={cn("size-1.5 rounded-full", TONE_DOT[state.tone])}
                  />
                  {state.label}
                </span>
                {svc.error && (
                  <p className="line-clamp-2 text-xs text-destructive">
                    {svc.error}
                  </p>
                )}
              </div>
            );
          })}
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card
            title={t("admin.location.cpu_load")}
            hint={t("admin.agent.last_48h")}
          >
            <LocationMetricsChart points={cpuMetrics} color="var(--chart-1)" />
          </Card>
          <Card
            title={t("admin.location.ram_load")}
            hint={t("admin.agent.last_48h")}
          >
            <LocationMetricsChart points={ramMetrics} color="var(--chart-2)" />
          </Card>
        </div>
      </div>
    </PageShell>
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
        <span className="text-[15px] leading-none font-semibold">{title}</span>
        {hint && (
          <span className="font-mono text-xs text-muted-foreground">{hint}</span>
        )}
        {action}
      </div>
      {children}
    </section>
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
    <div className="flex items-baseline justify-between gap-4 text-[13px]">
      <span className="text-muted-foreground">{label}</span>
      <span className="truncate text-right font-mono">{children}</span>
    </div>
  );
}
