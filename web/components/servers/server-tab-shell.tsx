"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { useParams, usePathname } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ConfirmDialog } from "@/components/servers/confirm-dialog";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
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
        <div className="font-panel space-y-4">
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

  const subtitle = [
    server.game?.name ? `${server.game.name} (id: ${server.game_id})` : null,
    server.location?.name,
    server.tariff?.name,
  ]
    .filter(Boolean)
    .join(" · ");

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

  // Пока идёт переход, действия недоступны, но место под ними сохраняется:
  // раньше все кнопки исчезали разом и появлялись через несколько секунд, из-за
  // чего раскладка прыгала дважды за одну операцию.
  const transitioning = isTransitionalStatus(st.category) || powerServer.isPending;

  // Заблокированный сервер запускать нельзя — сервер такой запрос отклонит.
  // Показывать при этом живую кнопку «Запустить» значит обещать то, чего не
  // будет: нажатие возвращало бы отказ и ничего больше. Остановку оставляем:
  // она разрешена и под блокировкой.
  const blocked = Boolean(server.is_blocked);
  const showRestart = !blocked && perms.can_restart !== false && canShowStop(server);
  const showStop = perms.can_stop !== false && canShowStop(server);
  const showStart = !blocked && perms.can_start !== false && canShowStart(server);
  const showReinstall =
    !blocked && perms.can_reinstall !== false && canShowReinstall(server) && installFailed;
  const maintenance = server.location?.maintenance;

  return (
    <PageShell variant={variant}>
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
                <h1 className="m-0 text-[25px] font-medium tracking-[-0.015em]">{server.name}</h1>
                <span
                  className={cn(
                    // Переход по цвету и рамке: смена состояния перестаёт быть
                    // мгновенной подменой и читается как движение.
                    "inline-flex items-center gap-[7px] rounded-full border px-[11px] py-1 text-[11.5px] font-medium transition-colors duration-300",
                    statusTone
                  )}
                >
                  <span
                    className={cn(
                      "h-1.5 w-1.5 rounded-full bg-current",
                      // Пульсация только пока операция идёт — постоянная мигалка
                      // на рабочем сервере превратилась бы в шум.
                      transitioning && "animate-pulse"
                    )}
                  />
                  {statusLabel}
                </span>
              </div>
              {subtitle && (
                <div className={cn("mt-[7px] text-[12.5px]", VX_MUTED)}>{subtitle}</div>
              )}
            </div>

            <div className="flex flex-col items-start gap-3 sm:items-end">
              <button
                type="button"
                onClick={copyAddress}
                className="inline-flex items-center gap-2.5 rounded-[10px] border border-[var(--vx-border-2)] bg-[var(--vx-elevated)] px-3 py-2 font-mono text-[12.5px] font-medium text-[var(--vx-fg)] transition-colors hover:border-[var(--vx-border-strong)]"
              >
                {address}
                <span
                  className={cn("font-sans text-[10.5px] tracking-[0.06em]", VX_FAINT)}
                >
                  {t("servers.shell.copy")}
                </span>
              </button>

              <div className="flex min-h-[34px] flex-wrap justify-end gap-2">
                {transitioning ? (
                  <span
                    className={cn(
                      "inline-flex h-[34px] cursor-default items-center gap-2 rounded-[9px] border border-[var(--vx-border-2)] bg-[var(--vx-elevated)] px-3.5 text-[12.5px] font-medium transition-colors",
                      VX_MUTED
                    )}
                  >
                    <i className="ri-loader-4-line animate-spin text-[14px]" />
                    {st.label}…
                  </span>
                ) : (
                  <>
                    {showRestart && (
                      <Btn onClick={() => onPower("restart")}>
                        {t("common.restart")}
                      </Btn>
                    )}
                    {showStop && (
                      <Btn tone="danger" onClick={() => onPower("stop")}>
                        {t("servers.shell.power_off")}
                      </Btn>
                    )}
                    {showStart && (
                      <Btn tone="primary" onClick={() => onPower("start")}>
                        {t("common.start")}
                      </Btn>
                    )}
                  </>
                )}
                {showReinstall && (
                  <Btn
                    tone="primary"
                    disabled={reinstallMutation.isPending}
                    onClick={() => setConfirmReinstall(true)}
                  >
                    {t("servers.shell.reinstall")}
                  </Btn>
                )}
                {expired && isOwner && (
                  <Link href={serverPath(basePath, id, "/tariff")} className={btnClass("primary")}>
                    {t("servers.shell.extend_rent")}
                  </Link>
                )}
              </div>
            </div>
          </div>

          <div className="border-t border-[var(--vx-border)] px-3.5 py-[9px]">
            <div className="flex items-center gap-1 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
              {tabs.map((tab) => {
                const active = tab.key === currentKey;
                const disabled = isServerTabDisabled(server, tab.key);
                const base =
                  "h-[30px] shrink-0 rounded-[8px] border px-3.5 text-[12.5px] font-medium transition-colors";
                if (disabled) {
                  return (
                    <span
                      key={tab.key}
                      title={t("servers.shell.tab_disabled")}
                      className={cn(base, "cursor-not-allowed border-transparent text-[var(--vx-border-hover)] leading-[30px]")}
                    >
                      {tab.label}
                    </span>
                  );
                }
                return (
                  <Link
                    key={tab.key}
                    href={serverPath(basePath, id, tab.suffix)}
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
        {/* Блокировку страница сервера не показывала вовсе — она была видна
            только в списке. Теперь, когда управление под блокировкой закрыто,
            без этой строки клиент упирался бы в отказ без объяснения. */}
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

        {children}
      </div>
    </PageShell>
  );
}
