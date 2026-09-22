"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Copy } from "lucide-react";
import { LocationAgentCard } from "@/components/admin/locations/location-agent-card";
import { LocationCapacityCard } from "@/components/admin/locations/location-capacity-card";
import { LocationIPsCard } from "@/components/admin/locations/location-ips-card";
import { MaintenanceDialog } from "@/components/admin/locations/maintenance-dialog";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import {
  fetchAdminLocation,
  fetchNodeCapacity,
  resetAdminLocationHostKey,
  testAdminLocationSSH,
  type AdminLocationListItem,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

type BadgeTone = "emerald" | "amber" | "muted";

type LocationRecord = Record<string, unknown>;

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
  const [maintenanceOpen, setMaintenanceOpen] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminLocation(id),
    queryFn: () => fetchAdminLocation(id),
    enabled: !!id,
  });

  const capacityQuery = useQuery({
    queryKey: ["node-capacity", id],
    queryFn: () => fetchNodeCapacity(id),
    enabled: !!id,
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

  if (isLoading) {
    return (
      <PageShell variant="admin">
        <div className="w-full space-y-4">
          <Skeleton className="h-20 w-full rounded-2xl" />
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

  const sshHost = String(location.ssh_host ?? "");
  const mysqlInstances = (
    Array.isArray(location.mysql_instances) ? location.mysql_instances : []
  ) as MysqlInstance[];
  const mysqlHost = String(
    location.mysql_host || location.ip_address || sshHost || "—"
  );
  const pmaHost = String(location.ip_address || location.ssh_host || "");
  const pmaPort = Number(location.phpmyadmin_port ?? 0);
  const pmaUrl = pmaHost && pmaPort > 0 ? `https://${pmaHost}:${pmaPort}` : "";
  const isActive = Boolean(location.is_active);
  const sshUser = String(location.ssh_user || "");
  const hostKey = String(location.ssh_host_key || "");
  const sshPort = Number(location.ssh_port || 22);
  const playersIp = String(location.ip_address || "");
  const maintenance = Boolean(location.maintenance_mode);
  const maintenanceReason = String(location.maintenance_reason || "");
  const maintenanceUntil =
    typeof location.maintenance_until === "string" ? location.maintenance_until : "";

  const maintenanceTarget: AdminLocationListItem = {
    id,
    name: String(location.name ?? ""),
    country: String(location.country ?? ""),
    code: String(location.code ?? ""),
    is_active: isActive,
    servers_count: capacityQuery.data?.servers ?? 0,
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
          </div>
        </header>

        {maintenance && (
          <Banner>
            {t("admin.locations.maintenance_active")}
            {maintenanceReason ? `: ${maintenanceReason}` : ""}
            {maintenanceUntil
              ? ` · ${t("admin.location.maintenance_until", {
                  date: new Date(maintenanceUntil).toLocaleString(localeTag()),
                })}`
              : ""}
          </Banner>
        )}

        <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(320px,400px)]">
          <div className="flex min-w-0 flex-col gap-4">
            <LocationCapacityCard locationId={id} />

            <LocationIPsCard locationId={id} />
          </div>

          <div className="flex min-w-0 flex-col gap-4">
            <LocationAgentCard locationId={id} />

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

const BADGE_TONE: Record<BadgeTone, string> = {
  emerald: "border-emerald-500/40 text-emerald-600 dark:text-emerald-400",
  amber: "border-amber-500/50 text-amber-600 dark:text-amber-400",
  muted: "text-muted-foreground",
};

function Badge({
  tone,
  mono,
  children,
}: {
  tone: BadgeTone;
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

function Banner({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-3 rounded-xl border border-amber-500/40 bg-amber-500/5 px-4 py-3">
      <span className="size-1.5 shrink-0 rounded-full bg-amber-500" />
      <p className="text-[13px] leading-relaxed text-amber-600 dark:text-amber-200/90">
        {children}
      </p>
    </div>
  );
}

function Card({
  title,
  action,
  children,
}: {
  title: string;
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-4 rounded-2xl border bg-card px-5 py-5 sm:px-6">
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <h2 className="text-[15px] leading-none font-semibold">{title}</h2>
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
