"use client";

import Link from "next/link";
import { useState } from "react";
import { useParams, usePathname, useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  ArrowRightLeft,
  ChevronDown,
  Lock,
  LockOpen,
  Network,
  Play,
  RefreshCw,
  RotateCw,
  Square,
  Trash2,
  Zap,
} from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { ConfirmDialog } from "@/components/servers/confirm-dialog";
import { AssignIPDialog } from "@/components/admin/servers/assign-ip-dialog";
import { MigrateServerDialog } from "@/components/admin/servers/migrate-server-dialog";
import { BlockServerDialog } from "@/components/admin/servers/block-server-dialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { Notice, VX_FAINT, VX_MUTED } from "@/components/vx/panel-ui";
import {
  adminToggleServerBlock,
  deleteServer,
  fetchAdminServerCard,
  powerServer,
  reinstallServer,
} from "@/lib/api";
import { serverPath, serversListPath } from "@/lib/panel-paths";
import { queryKeys } from "@/lib/query-keys";
import { localeTag } from "@/lib/i18n";
import {
  adminServerTabDefs,
  isAdminServerTabDisabled,
  type AdminServerTabKey,
} from "@/lib/server-tabs";
import { getServerStatus, isTransitionalStatus } from "@/lib/server-status";
import { isServerExpired, serverProvisioning } from "@/lib/server-lifecycle";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { useServerDetail } from "@/hooks/use-queries";
import { wheelScrollX } from "@/lib/wheel-scroll-x";

const CHIP: Record<string, string> = {
  running: "border-[var(--vx-border-strong)] bg-[var(--vx-veil-strong)] text-[var(--vx-fg-strong)]",
  stopped: "border-[var(--vx-tint-hover)] bg-transparent text-[var(--vx-muted)]",
  warn: "border-[rgba(232,160,60,0.3)] bg-[rgba(232,160,60,0.07)] text-[var(--vx-warn)]",
  danger: "border-[rgba(224,122,122,0.3)] bg-[rgba(224,122,122,0.07)] text-[var(--vx-danger)]",
};

function chipTone(category: string): string {
  if (["running", "active"].includes(category)) return CHIP.running;
  if (
    ["starting", "stopping", "installing", "reinstalling", "updating", "suspended"].includes(
      category
    )
  ) {
    return CHIP.warn;
  }
  if (["failed", "missing"].includes(category)) return CHIP.danger;
  return CHIP.stopped;
}

export function AdminServerShell({
  activeTab,
  children,
}: {
  activeTab?: AdminServerTabKey;
  children: React.ReactNode;
}) {
  const t = useT();
  const router = useRouter();
  const pathname = usePathname();
  const queryClient = useQueryClient();
  const { id } = useParams<{ id: string }>();

  const { data: server, isLoading } = useServerDetail(id);
  const cardQuery = useQuery({
    queryKey: queryKeys.adminServerCard(id),
    queryFn: () => fetchAdminServerCard(id),
    enabled: !!id,
  });
  const card = cardQuery.data;

  const [blockOpen, setBlockOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [reinstallOpen, setReinstallOpen] = useState(false);
  const [migrateOpen, setMigrateOpen] = useState(false);
  const [ipOpen, setIpOpen] = useState(false);

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
    void queryClient.invalidateQueries({ queryKey: queryKeys.adminServerCard(id) });
    void queryClient.invalidateQueries({ queryKey: queryKeys.adminServerAudit(id) });
  }

  function reportError(err: unknown) {
    toast.error(err instanceof Error ? err.message : t("common.error"));
  }

  const powerMutation = useMutation({
    mutationFn: (action: string) => powerServer(id, action),
    onSuccess: (_res, action) => {
      toast.success(t("servers.admin.command_sent", { action }));
      refresh();
    },
    onError: reportError,
  });

  const blockMutation = useMutation({
    mutationFn: ({ blocked, reason }: { blocked: boolean; reason: string }) =>
      adminToggleServerBlock(id, blocked, reason),
    onSuccess: (_res, vars) => {
      toast.success(
        vars.blocked ? t("servers.admin.blocked") : t("servers.admin.unblocked")
      );
      setBlockOpen(false);
      refresh();
    },
    onError: reportError,
  });

  const reinstallMutation = useMutation({
    mutationFn: () => reinstallServer(id),
    onSuccess: () => {
      toast.success(t("servers.shell.reinstall_started"));
      setReinstallOpen(false);
      refresh();
    },
    onError: reportError,
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteServer(id),
    onSuccess: (res) => {
      if (res.cleanup === "deferred") {
        toast.warning(t("servers.settings.server_deleted_deferred"));
      } else {
        toast.success(t("servers.settings.server_deleted"));
      }
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminServers });
      router.push(serversListPath("/admin"));
    },
    onError: reportError,
  });

  if (isLoading || !server) {
    return (
      <PageShell variant="admin">
        <div className="font-panel space-y-4">
          <Skeleton className="h-[148px] w-full rounded-[16px]" />
          <Skeleton className="h-[92px] w-full rounded-[14px]" />
          <Skeleton className="h-[420px] w-full rounded-[14px]" />
        </div>
      </PageShell>
    );
  }

  const tabs = adminServerTabDefs(server);
  const isTabActive = (suffix: string) => {
    const href = serverPath("/admin", id, suffix);
    const base = serverPath("/admin", id);
    return suffix === "" ? pathname === base : pathname === href || pathname.startsWith(`${href}/`);
  };
  const currentKey: AdminServerTabKey =
    activeTab ?? tabs.find((tab) => isTabActive(tab.suffix))?.key ?? "main";

  const st = getServerStatus(server);
  const expired = isServerExpired(server);
  const installFailed = serverProvisioning(server) === "failed";
  const transitioning = isTransitionalStatus(st.category) || powerMutation.isPending;
  const blocked = Boolean(server.is_blocked ?? card?.is_blocked);
  const suspended = Boolean(card?.suspended_at);
  const maintenance = server.location?.maintenance;

  const address =
    server.ip_address && server.port
      ? `${server.ip_address}:${server.port}`
      : server.ip_address || "—";

  const subtitle = [
    server.game?.name,
    server.location?.name,
    card?.node.name ? t("servers.admin.card.node_short", { node: card.node.name }) : null,
    server.tariff?.name,
  ]
    .filter(Boolean)
    .join(" · ");

  async function copyAddress() {
    try {
      await navigator.clipboard.writeText(address);
      toast.success(t("servers.shell.address_copied"));
    } catch {
      toast.error(t("common.copy_failed"));
    }
  }

  return (
    <PageShell variant="admin">
      <div className="font-panel flex flex-col gap-[18px]">
        <div className="rounded-[16px] border border-[var(--vx-border)] bg-[var(--vx-card)]">
          <div className="flex flex-wrap items-start justify-between gap-5 px-[22px] py-5">
            <div className="min-w-[260px]">
              <div
                className={cn(
                  "font-mono text-[10.5px] font-medium tracking-[0.1em] uppercase",
                  VX_FAINT
                )}
              >
                {t("servers.shell.eyebrow", { id: server.id })}
              </div>
              <div className="mt-2 flex flex-wrap items-center gap-3">
                <h1 className="m-0 text-[25px] font-medium tracking-[-0.015em]">
                  {server.name}
                </h1>
                <span
                  className={cn(
                    "inline-flex items-center gap-[7px] rounded-full border px-[11px] py-1 text-[11.5px] font-medium transition-colors duration-300",
                    expired ? CHIP.danger : chipTone(st.category)
                  )}
                >
                  <span
                    className={cn(
                      "h-1.5 w-1.5 rounded-full bg-current",
                      transitioning && "animate-pulse"
                    )}
                  />
                  {expired ? t("servers.shell.rent_expired") : st.label}
                </span>
              </div>
              {subtitle && <div className={cn("mt-[7px] text-[12.5px]", VX_MUTED)}>{subtitle}</div>}
              {card?.owner && (
                <div className="mt-2 text-[12.5px]">
                  <span className={VX_MUTED}>{t("servers.admin.col_owner")}: </span>
                  <Link
                    href={`/admin/users/${card.owner.id}`}
                    className="text-primary hover:underline"
                  >
                    {card.owner.email}
                  </Link>
                </div>
              )}
            </div>

            <div className="flex flex-col items-start gap-3 sm:items-end">
              <button
                type="button"
                onClick={copyAddress}
                className="inline-flex items-center gap-2.5 rounded-[10px] border border-[var(--vx-border-2)] bg-[var(--vx-elevated)] px-3 py-2 font-mono text-[12.5px] font-medium text-[var(--vx-fg)] transition-colors hover:border-[var(--vx-border-strong)]"
              >
                {address}
                <span className={cn("font-sans text-[10.5px] tracking-[0.06em]", VX_FAINT)}>
                  {t("servers.shell.copy")}
                </span>
              </button>

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button size="sm" variant="outline">
                    {t("servers.admin.actions.menu")}
                    <ChevronDown className="ml-2 h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-56">
                  <DropdownMenuItem onClick={() => powerMutation.mutate("start")}>
                    <Play className="mr-2 h-4 w-4" />
                    {t("common.start")}
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => powerMutation.mutate("stop")}>
                    <Square className="mr-2 h-4 w-4" />
                    {t("servers.shell.power_off")}
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => powerMutation.mutate("restart")}>
                    <RotateCw className="mr-2 h-4 w-4" />
                    {t("common.restart")}
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => powerMutation.mutate("kill")}>
                    <Zap className="mr-2 h-4 w-4" />
                    {t("servers.admin.actions.kill")}
                  </DropdownMenuItem>

                  <DropdownMenuSeparator />
                  {blocked ? (
                    <DropdownMenuItem
                      onClick={() => blockMutation.mutate({ blocked: false, reason: "" })}
                    >
                      <LockOpen className="mr-2 h-4 w-4" />
                      {t("servers.admin.actions.unblock")}
                    </DropdownMenuItem>
                  ) : (
                    <DropdownMenuItem onClick={() => setBlockOpen(true)}>
                      <Lock className="mr-2 h-4 w-4" />
                      {t("servers.admin.actions.block")}
                    </DropdownMenuItem>
                  )}

                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={() => setMigrateOpen(true)}>
                    <ArrowRightLeft className="mr-2 h-4 w-4" />
                    {t("servers.admin.migrate")}
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setIpOpen(true)}>
                    <Network className="mr-2 h-4 w-4" />
                    {t("servers.admin.actions.ip")}
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setReinstallOpen(true)}>
                    <RefreshCw className="mr-2 h-4 w-4" />
                    {t("servers.shell.reinstall")}
                  </DropdownMenuItem>

                  <DropdownMenuSeparator />
                  <DropdownMenuItem variant="destructive" onClick={() => setDeleteOpen(true)}>
                    <Trash2 className="mr-2 h-4 w-4" />
                    {t("common.delete")}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>

          <div className="border-t border-[var(--vx-border)] px-3.5 py-[9px]">
            <div ref={wheelScrollX} className="flex items-center gap-1 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
              {tabs.map((tab) => {
                const active = tab.key === currentKey;
                const disabled = isAdminServerTabDisabled(server, tab.key);
                const base =
                  "h-[30px] shrink-0 rounded-[8px] border px-3.5 text-[12.5px] font-medium transition-colors";
                if (disabled) {
                  return (
                    <span
                      key={tab.key}
                      title={t("servers.shell.tab_disabled")}
                      className={cn(
                        base,
                        "cursor-not-allowed border-transparent leading-[30px] text-[var(--vx-border-hover)]"
                      )}
                    >
                      {tab.label}
                    </span>
                  );
                }
                return (
                  <Link
                    key={tab.key}
                    href={serverPath("/admin", id, tab.suffix)}
                    className={cn(
                      base,
                      "inline-flex items-center",
                      active
                        ? "border-[var(--vx-border-strong)] bg-[var(--vx-tint)] text-[var(--vx-fg-strong)]"
                        : "border-transparent text-[var(--vx-muted)] hover:text-[var(--vx-fg)]"
                    )}
                  >
                    {tab.label}
                  </Link>
                );
              })}
            </div>
          </div>
        </div>

        {maintenance?.enabled && (
          <Notice>
            {t("servers.shell.maintenance", { location: server.location?.name ?? "" })}
            {maintenance.reason ? `: ${maintenance.reason}` : ""}
            {maintenance.until
              ? t("servers.shell.maintenance_until", {
                  date: new Date(maintenance.until).toLocaleString(localeTag()),
                })
              : ""}
            {t("servers.shell.maintenance_tail")}
          </Notice>
        )}
        {blocked && (
          <Notice tone="danger">
            {t("servers.admin.card.blocked_notice")}
            {card?.blocked_reason ? `: ${card.blocked_reason}` : ""}
          </Notice>
        )}
        {suspended && card?.suspended_at && (
          <Notice>
            {t("servers.admin.card.suspended_notice", {
              date: new Date(card.suspended_at).toLocaleString(localeTag()),
            })}
          </Notice>
        )}
        {expired && !suspended && <Notice>{t("servers.admin.card.expired_notice")}</Notice>}
        {installFailed && (
          <Notice tone="danger">
            {t("servers.admin.card.install_failed_notice")}
            {card?.provisioning_error ? `: ${card.provisioning_error}` : ""}
          </Notice>
        )}

        {children}
      </div>

      <BlockServerDialog
        open={blockOpen}
        serverName={server.name}
        pending={blockMutation.isPending}
        onOpenChange={setBlockOpen}
        onConfirm={(reason) => blockMutation.mutate({ blocked: true, reason })}
      />

      <ConfirmDialog
        open={reinstallOpen}
        onOpenChange={setReinstallOpen}
        title={t("servers.shell.reinstall_title")}
        description={t("servers.shell.reinstall_desc", { name: server.name })}
        confirmLabel={t("servers.shell.reinstall")}
        requirePhrase={server.name}
        pending={reinstallMutation.isPending}
        onConfirm={() => reinstallMutation.mutate()}
      />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("servers.admin.delete.title")}
        description={t("servers.admin.delete.description", {
          name: server.name,
          owner: card?.owner?.email ?? "—",
        })}
        confirmLabel={t("common.delete")}
        requirePhrase={server.name}
        pending={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />

      {ipOpen && card && (
        <AssignIPDialog
          serverId={id}
          serverName={server.name}
          nodeId={card.node.id}
          open
          onOpenChange={setIpOpen}
        />
      )}

      {migrateOpen && (
        <MigrateServerDialog
          serverId={id}
          serverName={server.name}
          open
          onOpenChange={setMigrateOpen}
        />
      )}
    </PageShell>
  );
}
