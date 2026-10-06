"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { useParams, usePathname } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ConfirmDialog } from "@/components/servers/confirm-dialog";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { ResponsiveTabs } from "@/components/vx/responsive-tabs";
import { Btn, Notice, VX_FAINT, VX_MUTED, btnClass } from "@/components/vx/panel-ui";
import { reinstallServer } from "@/lib/api";
import type { PanelVariant } from "@/lib/panel-paths";
import { serverPath, variantToBasePath } from "@/lib/panel-paths";
import { queryKeys } from "@/lib/query-keys";
import {
  canShowReinstall,
  canShowStart,
  canShowStop,
  isServerExpired,
  isServerProvisioning,
  serverProvisioning,
  serverRuntime,
} from "@/lib/server-lifecycle";
import { getServerStatus, isTransitionalStatus } from "@/lib/server-status";
import {
  serverTabDefs,
  canViewServerTab,
  isCs16Server,
  isServerTabDisabled,
  type ServerTabKey,
} from "@/lib/server-tabs";
import { cn } from "@/lib/utils";
import { useT, useTranslations } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";
import { useMe, usePowerServer, useServerDetail } from "@/hooks/use-queries";

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

export function ServerTabShell({
  variant = "user",
  activeTab,
  children,
}: {
  variant?: PanelVariant;
  activeTab?: ServerTabKey;
  children: React.ReactNode;
}) {
  const t = useT();
  const basePath = variantToBasePath(variant);
  const { id } = useParams<{ id: string }>();
  const pathname = usePathname();
  const queryClient = useQueryClient();
  const { data: server, isLoading } = useServerDetail(id);
  const { data: me } = useMe();
  const powerServer = usePowerServer();
  const [confirmReinstall, setConfirmReinstall] = useState(false);

  const reinstallMutation = useMutation({
    mutationFn: () => reinstallServer(id),
    onSuccess: () => {
      toast.success(t("servers.shell.reinstall_started"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const isOwner = !!server?.user_id && server.user_id === me?.user_id;

  const overrides = useTranslations();
  const tabs = useMemo(() => {
    const defs = serverTabDefs();
    if (!server) return defs;
    const allowed = defs.filter((tab) => canViewServerTab(server, tab.key, isOwner));
    if (allowed.length > 1) return allowed;
    return defs.filter((tab) => tab.key !== "maps" || isCs16Server(server));
  }, [server, isOwner, overrides]);

  const isTabActive = (suffix: string) => {
    const href = serverPath(basePath, id, suffix);
    const base = serverPath(basePath, id);
    return suffix === "" ? pathname === base : pathname === href || pathname.startsWith(`${href}/`);
  };
  const currentKey: ServerTabKey =
    activeTab ?? tabs.find((tab) => isTabActive(tab.suffix))?.key ?? "main";

  if (isLoading || !server) {
    return (
      <PageShell variant={variant}>
        <div className="font-panel srv2 space-y-4">
          <Skeleton className="h-[132px] w-full rounded-[16px]" />
          <Skeleton className="h-[92px] w-full rounded-[14px]" />
          <Skeleton className="h-[420px] w-full rounded-[14px]" />
        </div>
      </PageShell>
    );
  }

  const st = getServerStatus(server);
  const expired = isServerExpired(server);
  const provisioning = isServerProvisioning(server);
  const prov = serverProvisioning(server);
  const runtime = serverRuntime(server);
  const installFailed = prov === "failed";
  const perms = server.viewer_permissions ?? {};

  const statusLabel = expired ? t("servers.shell.rent_expired") : st.label;
  const statusTone = expired ? CHIP.danger : chipTone(st.category);

  const address =
    server.ip_address && server.port
      ? `${server.ip_address}:${server.port}`
      : server.ip_address || "—";

  async function onPower(action: "start" | "stop" | "restart") {
    try {
      await powerServer.mutateAsync({ id, action });
      toast.success(
        action === "start"
          ? t("servers.shell.power_starting")
          : action === "stop"
            ? t("servers.shell.power_stopping")
            : t("servers.shell.power_restarting")
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverStatus(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("common.error"));
    }
  }

  async function copyAddress() {
    try {
      await navigator.clipboard.writeText(address);
      toast.success(t("servers.shell.address_copied"));
    } catch {
      toast.error(t("common.copy_failed"));
    }
  }

  const transitioning = isTransitionalStatus(st.category) || powerServer.isPending;

  const blocked = Boolean(server.is_blocked);
  const showRestart = !blocked && perms.can_restart !== false && canShowStop(server);
  const showStop = perms.can_stop !== false && canShowStop(server);
  const showStart = !blocked && perms.can_start !== false && canShowStart(server);
  const showReinstall =
    !blocked && perms.can_reinstall !== false && canShowReinstall(server) && installFailed;
  const maintenance = server.location?.maintenance;

  return (
    <PageShell variant={variant}>
      <div className="font-panel srv2 relative flex flex-col gap-[18px]">
        <div className="srv2-glow" aria-hidden />
        <section className="relative flex flex-wrap items-end justify-between gap-7 pt-4 pb-2 sm:pt-6">
          <div className="flex min-w-0 flex-col gap-3.5">
            <div className="flex flex-wrap items-center gap-2.5">
              <span
                className={cn(
                  "inline-flex items-center gap-[9px] rounded-full border px-3.5 py-1.5 text-[13px] font-medium transition-colors duration-300",
                  ["running", "active"].includes(st.category) && !expired
                    ? "border-transparent bg-[var(--srv-accent-soft)] text-[var(--srv-accent)]"
                    : statusTone
                )}
              >
                <span
                  className={cn(
                    "relative h-2 w-2 rounded-full bg-current",
                    transitioning
                      ? "animate-pulse"
                      : ["running", "active"].includes(st.category) && !expired && "srv2-ping"
                  )}
                />
                {statusLabel}
                {!expired && ["running", "active"].includes(st.category) && server.uptime
                  ? ` · ${server.uptime}`
                  : ""}
              </span>
              {[server.game?.name, server.location?.name, server.tariff?.name]
                .filter(Boolean)
                .map((chip) => (
                  <span
                    key={chip}
                    className="rounded-full border border-[var(--vx-border)] px-3 py-1.5 text-[13px] text-[var(--vx-muted)]"
                  >
                    {chip}
                  </span>
                ))}
            </div>
            <h1 className="m-0 text-[44px] leading-none font-semibold tracking-[-0.035em] break-words sm:text-[60px] lg:text-[76px]">
              {server.name}
            </h1>
            <div className={cn("font-mono text-[11px] tracking-[0.06em] break-all", VX_FAINT)}>
              {t("servers.shell.eyebrow", { id: server.id })}
            </div>
          </div>

          <div className="flex w-full max-w-full flex-col gap-3 md:w-[400px]">
            <button
              type="button"
              onClick={copyAddress}
              className="flex h-14 items-center justify-between gap-4 rounded-2xl border border-[var(--vx-border-2)] bg-[var(--vx-elevated)] pr-2 pl-5 font-mono text-[16px] text-[var(--vx-fg)] transition-colors hover:border-[var(--srv-accent)]"
            >
              <span className="truncate">{address}</span>
              <span className="flex h-10 shrink-0 items-center gap-2 rounded-[11px] bg-[var(--srv-accent)] px-3.5 font-sans text-[13px] font-semibold text-[var(--srv-accent-on)]">
                <i className="ri-file-copy-line text-[15px]" />
                {t("servers.shell.copy")}
              </span>
            </button>

            <div className="flex min-h-[44px] flex-wrap gap-2.5">
              {transitioning ? (
                <span
                  className={cn(
                    "inline-flex h-11 flex-1 cursor-default items-center justify-center gap-2 rounded-xl border border-[var(--vx-border-2)] bg-[var(--vx-elevated)] px-3.5 text-[13px] font-medium",
                    VX_MUTED
                  )}
                >
                  <i className="ri-loader-4-line animate-spin text-[14px]" />
                  {st.label}…
                </span>
              ) : (
                <>
                  {showRestart && (
                    <Btn className="h-11 flex-1 rounded-xl" onClick={() => onPower("restart")}>
                      <i className="ri-refresh-line text-[16px]" />
                      {t("common.restart")}
                    </Btn>
                  )}
                  {showStop && (
                    <Btn tone="danger" className="h-11 flex-1 rounded-xl" onClick={() => onPower("stop")}>
                      <i className="ri-shut-down-line text-[16px]" />
                      {t("servers.shell.power_off")}
                    </Btn>
                  )}
                  {showStart && (
                    <Btn tone="primary" className="h-11 flex-1 rounded-xl" onClick={() => onPower("start")}>
                      <i className="ri-play-line text-[16px]" />
                      {t("common.start")}
                    </Btn>
                  )}
                </>
              )}
              {showReinstall && (
                <Btn
                  tone="primary"
                  className="h-11 flex-1 rounded-xl"
                  disabled={reinstallMutation.isPending}
                  onClick={() => setConfirmReinstall(true)}
                >
                  {t("servers.shell.reinstall")}
                </Btn>
              )}
              {expired && isOwner && (
                <Link
                  href={serverPath(basePath, id, "/tariff")}
                  className={cn(btnClass("primary"), "h-11 flex-1 rounded-xl")}
                >
                  {t("servers.shell.extend_rent")}
                </Link>
              )}
            </div>
          </div>
        </section>

        <div className="sticky top-2 z-[4] rounded-2xl border border-[var(--vx-border)] bg-[var(--vx-elevated)] p-1">
          <ResponsiveTabs
            pillId={`server-tab-${id}`}
            activeKey={currentKey}
            tabs={tabs.map((tab) => ({
              key: tab.key,
              label: tab.label,
              href: serverPath(basePath, id, tab.suffix),
              disabled: isServerTabDisabled(server, tab.key),
              disabledHint: t("servers.shell.tab_disabled"),
            }))}
          />
        </div>

        <ConfirmDialog
          open={confirmReinstall}
          onOpenChange={setConfirmReinstall}
          title={t("servers.shell.reinstall_title")}
          description={t("servers.shell.reinstall_desc", { name: server.name })}
          confirmLabel={t("servers.shell.reinstall")}
          requirePhrase={server.name}
          pending={reinstallMutation.isPending}
          onConfirm={() => {
            setConfirmReinstall(false);
            reinstallMutation.mutate();
          }}
        />

        {maintenance?.enabled && (
          <Notice>
            {t("servers.shell.maintenance", {
              location: server.location?.name ?? "",
            })}
            {maintenance.reason ? `: ${maintenance.reason}` : ""}
            {maintenance.until
              ? t("servers.shell.maintenance_until", {
                  date: new Date(maintenance.until).toLocaleString(localeTag()),
                })
              : ""}
            {t("servers.shell.maintenance_tail")}
          </Notice>
        )}
        {server.is_blocked && (
          <Notice tone="danger">
            {t("servers.shell.blocked")}
            {server.blocked_reason ? `: ${server.blocked_reason}` : ""}
            {t("servers.shell.blocked_tail")}
          </Notice>
        )}
        {expired && <Notice>{t("servers.shell.expired_notice")}</Notice>}
        {!expired && installFailed && runtime !== "running" && currentKey !== "main" && (
          <Notice>{t("servers.shell.install_failed_notice")}</Notice>
        )}

        <div className="vx-fade-in flex flex-col gap-[18px]">{children}</div>
      </div>
    </PageShell>
  );
}
