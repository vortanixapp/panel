"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Chip,
  CopyAddress,
  LoadBar,
  MON,
  MON_BTN,
  MON_BTN_PRIMARY,
  MON_CARD,
  MON_INNER,
  MON_ROW,
  MON_TH,
  Sparkline,
  StatusPill,
  gameIconSrc,
  loadPct,
  normalizeServerRow,
  pingColor,
  pingText,
  relativeUpdate,
  tpsText,
  uptimeText,
} from "@/components/user/monitoring/shared";
import { fetchMonitoring, fetchMonitoringTop, type MonitoringServerRow } from "@/lib/api";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

type ViewMode = "table" | "cards";

const ALL_GAMES = "all";

export function MonitoringPageContent() {
  useT();
  const [view, setView] = useState<ViewMode>("table");
  const [gameFilter, setGameFilter] = useState<string>(ALL_GAMES);
  const [query, setQuery] = useState("");

  const monitoring = useQuery({
    queryKey: queryKeys.monitoring,
    queryFn: fetchMonitoring,
    refetchInterval: 15_000,
    refetchIntervalInBackground: false,
  });

  const top = useQuery({
    queryKey: queryKeys.monitoringTop(),
    queryFn: () => fetchMonitoringTop(),
    staleTime: 60_000,
  });

  const servers = useMemo(
    () => (monitoring.data?.servers ?? []).map(normalizeServerRow),
    [monitoring.data]
  );

  const games = useMemo(() => {
    const names = new Set<string>();
    for (const s of servers) names.add(s.game_name || s.game_id);
    return Array.from(names);
  }, [servers]);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return servers.filter((s) => {
      const gameLabel = s.game_name || s.game_id;
      if (gameFilter !== ALL_GAMES && gameLabel !== gameFilter) return false;
      if (!q) return true;
      return s.name.toLowerCase().includes(q) || s.ip.toLowerCase().includes(q);
    });
  }, [servers, gameFilter, query]);

  const copied = () => toast.success(t("monitoring.address_copied"));

  if (monitoring.isLoading) {
    return (
      <PageShell variant="user">
        <div className="flex w-full flex-col gap-5">
          <Skeleton className="h-16 w-full rounded-[14px]" />
          <Skeleton className="h-24 w-full rounded-[14px]" />
          <Skeleton className="h-96 w-full rounded-[14px]" />
        </div>
      </PageShell>
    );
  }

  if (monitoring.isError) {
    return (
      <PageShell variant="user">
        <MonitoringEmptyState
          icon="ri-error-warning-line"
          title={t("monitoring.error.title")}
          text={t("monitoring.error.text")}
          action={
            <button type="button" className={MON_BTN_PRIMARY} onClick={() => void monitoring.refetch()}>
              {t("common.retry")}
            </button>
          }
        />
      </PageShell>
    );
  }

  if (servers.length === 0) {
    return (
      <PageShell variant="user">
        <MonitoringEmptyState
          icon="ri-pulse-line"
          title={t("monitoring.empty.title")}
          text={t("monitoring.empty.text")}
          action={
            <Link href="/rent-server" className={MON_BTN_PRIMARY}>
              {t("monitoring.empty.rent_cta")}
            </Link>
          }
        />
      </PageShell>
    );
  }

  const data = {
    ...monitoring.data!,
    total: monitoring.data!.total ?? servers.length,
    online: monitoring.data!.online ?? 0,
    players: monitoring.data!.players ?? 0,
    slots: monitoring.data!.slots ?? 0,
    avg_uptime: monitoring.data!.avg_uptime ?? 0,
    incidents_7d: monitoring.data!.incidents_7d ?? 0,
    updated_at: monitoring.data!.updated_at ?? new Date().toISOString(),
  };

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-5">
        <div className="flex flex-wrap items-end gap-4">
          <div className="min-w-[260px] flex-1">
            <h1 className="text-[26px] font-bold tracking-[-0.02em]">
              {t("monitoring.title")}
            </h1>
            <p className={cn("mt-1 text-[13px]", "text-[var(--vx-muted)]")}>
              {t("monitoring.subtitle")}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <div className="flex h-9 items-center gap-1.5 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-card)] px-3 text-[12px] text-[var(--vx-muted)]">
              <span
                className="h-1.5 w-1.5 rounded-full"
                style={{ background: monitoring.isFetching ? MON.warn : MON.ok }}
              />
              <span className="font-mono">{relativeUpdate(data.updated_at)}</span>
            </div>
            <button
              type="button"
              className={MON_BTN}
              onClick={() => void monitoring.refetch()}
              disabled={monitoring.isFetching}
            >
              <i className="ri-refresh-line text-[15px]" />
              {t("common.refresh")}
            </button>
            <Link href="/rent-server" className={MON_BTN_PRIMARY}>
              <i className="ri-add-line text-[15px]" />
              {t("monitoring.add_server")}
            </Link>
          </div>
        </div>

        <div className="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-3">
          <KpiCard
            label={t("monitoring.kpi.total")}
            icon="ri-server-line"
            value={String(data.total)}
            sub={t("monitoring.kpi.online_sub", { count: data.online })}
          />
          <KpiCard
            label={t("monitoring.kpi.players")}
            icon="ri-user-line"
            value={String(data.players)}
            sub={t("monitoring.kpi.slots_sub", { count: data.slots })}
          />
          <KpiCard
            label={t("monitoring.kpi.avg_uptime")}
            icon="ri-pulse-line"
            value={uptimeText(data.avg_uptime)}
            sub={t("monitoring.kpi.for_30d")}
            color={data.avg_uptime >= 99 ? MON.ok : data.avg_uptime > 0 ? MON.warn : MON.fg}
          />
          <KpiCard
            label={t("monitoring.kpi.incidents")}
            icon="ri-error-warning-line"
            value={String(data.incidents_7d)}
            sub={t("monitoring.kpi.for_7d")}
            color={data.incidents_7d > 0 ? MON.warn : MON.fg}
          />
        </div>

        <div className={cn("overflow-hidden", MON_CARD)}>
          <div className="flex flex-wrap items-center gap-3 border-b border-[var(--vx-border)] px-4 py-3.5">
            <div className="mr-auto flex items-center gap-2">
              <h2 className="text-[15px] font-semibold">{t("monitoring.my_servers")}</h2>
              <span className="font-mono text-[12px] text-[var(--vx-muted)]">
                {t("monitoring.count_of", {
                  shown: visible.length,
                  total: servers.length,
                })}
              </span>
            </div>
            <div className="flex h-[34px] w-[220px] items-center gap-2 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-bg)] px-2.5">
              <i className="ri-search-line text-[14px] text-[var(--vx-muted)]" />
              <input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={t("monitoring.search_placeholder")}
                className="min-w-0 flex-1 bg-transparent text-[12px] text-[var(--vx-fg)] outline-none placeholder:text-[var(--vx-faint)]"
              />
            </div>
            <div className="flex gap-1 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-bg)] p-[3px]">
              <Chip active={gameFilter === ALL_GAMES} onClick={() => setGameFilter(ALL_GAMES)}>
                {t("common.all")}
              </Chip>
              {games.map((g) => (
                <Chip key={g} active={gameFilter === g} onClick={() => setGameFilter(g)}>
                  {g}
                </Chip>
              ))}
            </div>
            <div className="flex gap-1 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-bg)] p-[3px]">
              <Chip active={view === "table"} onClick={() => setView("table")}>
                <i className="ri-list-check text-[14px]" />
                {t("monitoring.view.table")}
              </Chip>
              <Chip active={view === "cards"} onClick={() => setView("cards")}>
                <i className="ri-layout-grid-line text-[14px]" />
                {t("monitoring.view.cards")}
              </Chip>
            </div>
          </div>

          {visible.length === 0 ? (
            <div className="px-4 py-12 text-center">
              <div className="text-[14px] font-semibold">{t("common.not_found")}</div>
              <p className="mt-1.5 text-[12px] text-[var(--vx-muted)]">
                {t("monitoring.filter_empty_hint")}
              </p>
            </div>
          ) : view === "table" ? (
            <ServerTable servers={visible} onCopied={copied} />
          ) : (
            <ServerCards servers={visible} onCopied={copied} />
          )}
        </div>

        <div className={cn("overflow-hidden", MON_CARD)}>
          <div className="flex flex-wrap items-center gap-3 border-b border-[var(--vx-border)] px-4 py-3.5">
            <div className="mr-auto">
              <h2 className="text-[15px] font-semibold">{t("monitoring.public.title")}</h2>
              <p className="mt-0.5 text-[12px] text-[var(--vx-muted)]">
                {t("monitoring.public.subtitle")}
              </p>
            </div>
            <Link
              href="/monitoring/top"
              className="flex items-center gap-1.5 text-[12px] text-[var(--vx-muted)] transition-colors hover:text-white"
            >
              {t("monitoring.public.all_ranking")}
              <i className="ri-arrow-right-line text-[14px]" />
            </Link>
          </div>
          {(top.data?.items ?? []).length === 0 ? (
            <p className="px-4 py-8 text-center text-[12px] text-[var(--vx-muted)]">
              {t("monitoring.public.rating_empty")}
            </p>
          ) : (
            <div>
              {(top.data?.items ?? []).slice(0, 5).map((t) => (
                <Link
                  key={t.server_id}
                  href={`/monitoring/public/${t.server_id}`}
                  className={cn("flex items-center gap-3 px-4 py-2.5", MON_ROW)}
                >
                  <span className="w-[22px] text-right font-mono text-[12px] text-[var(--vx-muted)]">
                    {t.rank}
                  </span>
                  <GameIcon gameId={t.game_id} className="h-6 w-6 p-[3px]" />
                  <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{t.name}</span>
                  <span className="w-[110px] truncate text-[12px] text-[var(--vx-muted)]">
                    {t.game_name}
                  </span>
                  <span className="w-[88px] text-right font-mono text-[12px]">
                    {t.online} / {t.slots}
                  </span>
                  <span className="w-[72px] text-right font-mono text-[12px] text-[var(--vx-muted)]">
                    {t.votes}
                  </span>
                </Link>
              ))}
            </div>
          )}
        </div>
      </div>
    </PageShell>
  );
}

function KpiCard({
  label,
  icon,
  value,
  sub,
  color = MON.fg,
}: {
  label: string;
  icon: string;
  value: string;
  sub: string;
  color?: string;
}) {
  return (
    <div className={cn("px-[18px] py-4", MON_CARD)}>
      <div className="flex items-center justify-between gap-2">
        <span className="text-[12px] font-medium text-[var(--vx-muted)]">{label}</span>
        <i className={cn(icon, "text-[15px] text-[var(--vx-faint)]")} />
      </div>
      <div className="mt-2.5 flex items-baseline gap-2">
        <span className="text-[28px] font-bold tracking-[-0.02em]" style={{ color }}>
          {value}
        </span>
        <span className="text-[12px] text-[var(--vx-muted)]">{sub}</span>
      </div>
    </div>
  );
}

export function GameIcon({ gameId, className }: { gameId: string; className?: string }) {
  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src={gameIconSrc(gameId)}
      alt=""
      className={cn("shrink-0 rounded-[6px] bg-[var(--vx-inset)] object-contain", className)}
      onError={(e) => {
        e.currentTarget.style.visibility = "hidden";
      }}
    />
  );
}

function ServerTable({
  servers,
  onCopied,
}: {
  servers: MonitoringServerRow[];
  onCopied: () => void;
}) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse text-left">
        <thead>
          <tr className="bg-[var(--vx-card-2)]">
            <th className={cn(MON_TH, "px-4")}>{t("monitoring.col.server")}</th>
            <th className={MON_TH}>{t("common.status")}</th>
            <th className={MON_TH}>{t("monitoring.col.online")}</th>
            <th className={MON_TH}>{t("monitoring.col.day")}</th>
            <th className={MON_TH}>{t("monitoring.col.cpu_ram")}</th>
            <th className={MON_TH}>{t("monitoring.col.ping")}</th>
            <th className={MON_TH}>{t("monitoring.col.tps")}</th>
            <th className={MON_TH}>{t("monitoring.col.uptime")}</th>
            <th className={MON_TH}>{t("monitoring.col.address")}</th>
            <th className={cn(MON_TH, "px-4 text-right")} />
          </tr>
        </thead>
        <tbody>
          {servers.map((s) => (
            <tr key={s.id} className={MON_ROW}>
              <td className="px-4 py-3">
                <div className="flex items-center gap-2.5">
                  <GameIcon gameId={s.game_id} className="h-7 w-7 p-1" />
                  <div className="min-w-0">
                    <Link
                      href={`/monitoring/${s.id}`}
                      className="block text-[13px] font-semibold text-[var(--vx-fg)] hover:text-white"
                    >
                      {s.name}
                    </Link>
                    <div className="text-[11px] text-[var(--vx-muted)]">
                      {[s.game_name || s.game_id, s.version].filter(Boolean).join(" · ")}
                    </div>
                  </div>
                </div>
              </td>
              <td className="p-3">
                <StatusPill status={s.status} />
              </td>
              <td className="p-3">
                <div className="font-mono text-[13px]">
                  {s.online} / {s.slots}
                </div>
                <LoadBar pct={loadPct(s.online, s.slots)} className="mt-1.5 w-[72px]" />
              </td>
              <td className="p-3">
                <Sparkline values={s.spark} cap={s.slots} className="h-[30px] w-[96px]" />
              </td>
              <td className="p-3 font-mono text-[12px] text-[var(--vx-dim)]">
                {s.status === "running" ? `${s.cpu}% / ${s.ram}%` : "— / —"}
              </td>
              <td className="p-3 font-mono text-[12px]" style={{ color: pingColor(s.ping) }}>
                {pingText(s.ping)}
              </td>
              <td className="p-3 font-mono text-[12px] text-[var(--vx-dim)]">{tpsText(s.tps)}</td>
              <td className="p-3 font-mono text-[12px] text-[var(--vx-dim)]">{uptimeText(s.uptime)}</td>
              <td className="p-3">
                <CopyAddress address={s.ip} onCopied={onCopied} />
              </td>
              <td className="whitespace-nowrap px-4 py-3 text-right">
                <Link
                  href={`/monitoring/${s.id}`}
                  className="text-[12px] text-[var(--vx-muted)] hover:text-white"
                >
                  {t("monitoring.details")}
                </Link>
                {s.public_enabled && (
                  <>
                    <span className="mx-2 text-[var(--vx-border-strong)]">·</span>
                    <Link
                      href={`/monitoring/public/${s.id}`}
                      className="text-[12px] text-[var(--vx-muted)] hover:text-white"
                    >
                      {t("monitoring.public_link")}
                    </Link>
                  </>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ServerCards({
  servers,
  onCopied,
}: {
  servers: MonitoringServerRow[];
  onCopied: () => void;
}) {
  return (
    <div className="grid grid-cols-[repeat(auto-fill,minmax(320px,1fr))] gap-3 p-4">
      {servers.map((s) => (
        <div key={s.id} className={cn("flex flex-col gap-3.5 p-4", MON_INNER)}>
          <div className="flex items-start gap-2.5">
            <GameIcon gameId={s.game_id} className="h-9 w-9 rounded-[8px] p-1.5" />
            <div className="min-w-0 flex-1">
              <Link href={`/monitoring/${s.id}`} className="block text-[14px] font-semibold">
                {s.name}
              </Link>
              <div className="text-[11px] text-[var(--vx-muted)]">
                {[s.game_name || s.game_id, s.version].filter(Boolean).join(" · ")}
              </div>
            </div>
            <StatusPill status={s.status} />
          </div>
          <Sparkline values={s.spark} cap={s.slots} className="h-12 w-full" />
          <div className="grid grid-cols-4 gap-2">
            <CardStat label={t("monitoring.col.online")} value={`${s.online} / ${s.slots}`} />
            <CardStat
              label={t("monitoring.col.ping")}
              value={pingText(s.ping)}
              color={pingColor(s.ping)}
            />
            <CardStat label={t("monitoring.col.tps")} value={tpsText(s.tps)} />
            <CardStat label={t("monitoring.col.uptime")} value={uptimeText(s.uptime)} />
          </div>
          <div className="flex items-center gap-2">
            <CopyAddress address={s.ip} onCopied={onCopied} className="flex-1 justify-between" />
            <Link href={`/monitoring/${s.id}`} className={MON_BTN_PRIMARY}>
              {t("monitoring.details")}
            </Link>
          </div>
        </div>
      ))}
    </div>
  );
}

function CardStat({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div>
      <div className="text-[10px] uppercase tracking-[0.06em] text-[var(--vx-muted)]">{label}</div>
      <div className="mt-[3px] font-mono text-[13px]" style={color ? { color } : undefined}>
        {value}
      </div>
    </div>
  );
}

export function MonitoringEmptyState({
  icon,
  title,
  text,
  action,
}: {
  icon: string;
  title: string;
  text: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="w-full">
      <div className="mb-5">
        <h1 className="text-[26px] font-bold tracking-[-0.02em]">{t("monitoring.title")}</h1>
        <p className="mt-1 text-[13px] text-[var(--vx-muted)]">
          {t("monitoring.subtitle_short")}
        </p>
      </div>
      <div
        className={cn(
          "mt-6 w-full rounded-[16px] px-8 py-16 text-center",
          MON_CARD
        )}
      >
        <div className="mx-auto mb-[18px] flex h-14 w-14 items-center justify-center rounded-[14px] bg-[var(--vx-inset)]">
          <i className={cn(icon, "text-2xl text-[var(--vx-muted)]")} />
        </div>
        <h2 className="text-[18px] font-semibold">{title}</h2>
        <p className="mx-auto mt-2 max-w-[520px] text-[13px] text-pretty text-[var(--vx-muted)]">
          {text}
        </p>
        {action && <div className="mt-6 flex justify-center gap-2.5">{action}</div>}
        <div className="mt-7 flex justify-center gap-6 border-t border-[var(--vx-border)] pt-5 text-[11px] text-[var(--vx-muted)]">
          <span>{t("monitoring.empty.feat_online")}</span>
          <span>{t("monitoring.empty.feat_uptime")}</span>
          <span>{t("monitoring.empty.feat_banners")}</span>
        </div>
      </div>
    </div>
  );
}
