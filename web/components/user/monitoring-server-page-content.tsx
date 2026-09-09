"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { GameIcon, MonitoringEmptyState } from "@/components/user/monitoring-page-content";
import {
  Chip,
  CopyAddress,
  LoadBar,
  MON,
  MON_BTN,
  MON_BTN_PRIMARY,
  MON_CARD,
  MON_INNER,
  MON_INPUT,
  MON_ROW,
  MON_TH,
  StatusPill,
  areaPath,
  durationText,
  initials,
  loadPct,
  normalizeIncidents,
  normalizePlayers,
  normalizeServerRow,
  normalizeSettings,
  normalizeUptimeDays,
  pingColor,
  pingText,
  polyPoints,
  secondsToHms,
  shortDate,
  tpsText,
  uptimeText,
} from "@/components/user/monitoring/shared";
import {
  fetchMonitoringServer,
  fetchServerLogs,
  fetchServerMonitoringStats,
  updateMonitoringSettings,
  type MonitoringIncident,
  type MonitoringServerDetail,
  type MonitoringSettings,
  type MonitoringUptimeDay,
} from "@/lib/api";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

type MonTab = "players" | "stats" | "banners" | "console" | "public" | "incidents";

const TABS: { key: MonTab; labelKey: string; icon: string }[] = [
  { key: "players", labelKey: "monitoring.tab.players", icon: "ri-team-line" },
  { key: "stats", labelKey: "monitoring.tab.stats", icon: "ri-line-chart-line" },
  { key: "banners", labelKey: "monitoring.tab.banners", icon: "ri-image-line" },
  { key: "console", labelKey: "monitoring.tab.console", icon: "ri-terminal-box-line" },
  { key: "public", labelKey: "monitoring.tab.public", icon: "ri-global-line" },
  { key: "incidents", labelKey: "monitoring.tab.incidents", icon: "ri-history-line" },
];

export function MonitoringServerPageContent() {
  useT();
  const { id } = useParams<{ id: string }>();
  const [tab, setTab] = useState<MonTab>("players");

  const detail = useQuery({
    queryKey: queryKeys.monitoringServer(id ?? ""),
    queryFn: () => fetchMonitoringServer(id!),
    enabled: !!id,
    refetchInterval: 15_000,
    refetchIntervalInBackground: false,
  });

  if (detail.isLoading && !detail.data) {
    return (
      <PageShell variant="user">
        <div className="flex w-full flex-col gap-4">
          <Skeleton className="h-40 w-full rounded-[14px]" />
          <Skeleton className="h-96 w-full rounded-[14px]" />
        </div>
      </PageShell>
    );
  }

  if (!detail.data) {
    return (
      <PageShell variant="user">
        <MonitoringEmptyState
          icon="ri-error-warning-line"
          title={t("monitoring.detail.not_found_title")}
          text={t("monitoring.detail.not_found_text")}
          action={
            <Link href="/monitoring" className={MON_BTN_PRIMARY}>
              {t("monitoring.detail.back_to_list")}
            </Link>
          }
        />
      </PageShell>
    );
  }

  const raw = detail.data;
  const data: MonitoringServerDetail = {
    ...raw,
    server: normalizeServerRow(raw.server),
    players: normalizePlayers(raw.players),
    peak_24h: Number(raw.peak_24h) || 0,
    avg_24h: Number(raw.avg_24h) || 0,
    uptime_30d: Number(raw.uptime_30d) || 0,
    uptime_days: normalizeUptimeDays(raw.uptime_days),
    incidents: normalizeIncidents(raw.incidents),
    settings: normalizeSettings(raw.settings),
    visits: {
      total: Number(raw.visits?.total) || 0,
      days: Array.isArray(raw.visits?.days) ? raw.visits.days.map((v) => Number(v) || 0) : [],
    },
    public_url: raw.public_url ?? "",
    banner_url: raw.banner_url ?? "",
  };
  const s = data.server;

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-4">
        <Link href="/monitoring" className="text-[12px] text-[var(--vx-muted)] hover:text-white">
          {t("monitoring.detail.back_link")}
        </Link>

        <div className={cn("overflow-hidden", MON_CARD)}>
          <div className="flex flex-wrap items-center gap-3.5 border-b border-[var(--vx-border)] p-[18px]">
            <GameIcon gameId={s.game_id} className="h-11 w-11 rounded-[10px] p-[7px]" />
            <div className="min-w-[200px] flex-1">
              <div className="flex items-center gap-2">
                <h1 className="text-[19px] font-bold tracking-[-0.01em]">{s.name}</h1>
                <span className="font-mono text-[11px] text-[var(--vx-muted)]">ID {s.id}</span>
              </div>
              <div className="mt-[3px] text-[12px] text-[var(--vx-muted)]">
                {[s.game_name || s.game_id, s.version, s.region].filter(Boolean).join(" · ")}
              </div>
            </div>
            <StatusPill status={s.status} />
            <CopyAddress
              address={s.ip}
              onCopied={() => toast.success(t("monitoring.address_copied"))}
              className="h-9 px-3 text-[12px]"
            />
            <button
              type="button"
              className={MON_BTN}
              onClick={() => void detail.refetch()}
              disabled={detail.isFetching}
            >
              <i className="ri-refresh-line text-[15px]" />
              {t("common.refresh")}
            </button>
            {s.public_enabled && (
              <Link href={`/monitoring/public/${s.id}`} className={MON_BTN_PRIMARY}>
                <i className="ri-external-link-line text-[15px]" />
                {t("monitoring.tab.public")}
              </Link>
            )}
          </div>

          <div className="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-px bg-[var(--vx-border)]">
            <MetricTile
              label={t("monitoring.col.online")}
              value={`${s.online} / ${s.slots}`}
              bar={loadPct(s.online, s.slots)}
            />
            <MetricTile
              label={t("monitoring.col.uptime")}
              value={uptimeText(s.uptime)}
              note={t("monitoring.stat.uptime_30d_note")}
              color={MON.ok}
            />
            <MetricTile
              label="CPU"
              value={s.status === "running" ? `${s.cpu}%` : "—"}
              bar={s.status === "running" ? s.cpu : undefined}
            />
            <MetricTile
              label="RAM"
              value={s.status === "running" ? `${s.ram}%` : "—"}
              bar={s.status === "running" ? s.ram : undefined}
              note={s.ram_limit_mb ? `${s.ram_limit_mb} MB` : undefined}
            />
            <MetricTile
              label={t("monitoring.col.ping")}
              value={pingText(s.ping)}
              color={pingColor(s.ping)}
              note={s.region}
            />
            <MetricTile
              label={t("monitoring.col.tps")}
              value={tpsText(s.tps)}
              note={s.tps ? t("monitoring.stat.tps_goal") : undefined}
            />
            <MetricTile
              label={t("monitoring.stat.version")}
              value={s.version || "—"}
              note={s.game_name || s.game_id}
            />
            <MetricTile
              label={t("monitoring.stat.map")}
              value={s.map || "—"}
              note={t("monitoring.stat.map_note")}
            />
          </div>
        </div>

        <div className={cn("overflow-hidden", MON_CARD)}>
          <div className="flex flex-wrap gap-1 border-b border-[var(--vx-border)] px-3 py-2.5">
            {TABS.map((item) => (
              <button
                key={item.key}
                type="button"
                onClick={() => setTab(item.key)}
                className={cn(
                  "flex items-center gap-[7px] rounded-[9px] px-3.5 py-2 text-[13px] font-medium transition-colors",
                  tab === item.key
                    ? "bg-[var(--vx-tint)] text-[var(--vx-fg)]"
                    : "text-[var(--vx-muted)] hover:text-[var(--vx-fg)]"
                )}
              >
                <i className={cn(item.icon, "text-[14px]")} />
                {t(item.labelKey)}
              </button>
            ))}
          </div>

          <div className="p-[18px]">
            {tab === "players" && <PlayersTab data={data} />}
            {tab === "stats" && <StatsTab serverId={s.id} slots={s.slots} />}
            {tab === "banners" && <BannersTab data={data} />}
            {tab === "console" && <ConsoleTab serverId={s.id} />}
            {tab === "public" && <PublicTab data={data} />}
            {tab === "incidents" && (
              <IncidentsTab
                uptime={data.uptime_30d}
                days={data.uptime_days}
                incidents={data.incidents}
              />
            )}
          </div>
        </div>
      </div>
    </PageShell>
  );
}

function MetricTile({
  label,
  value,
  bar,
  note,
  color = MON.fg,
}: {
  label: string;
  value: string;
  bar?: number;
  note?: string;
  color?: string;
}) {
  return (
    <div className="bg-[var(--vx-card)] px-4 py-3.5">
      <div className="text-[11px] uppercase tracking-[0.06em] text-[var(--vx-muted)]">{label}</div>
      <div className="mt-1.5 font-mono text-[17px] font-medium" style={{ color }}>
        {value}
      </div>
      {bar != null && <LoadBar pct={bar} className="mt-2" />}
      {note && <div className="mt-1.5 text-[11px] text-[var(--vx-muted)]">{note}</div>}
    </div>
  );
}

function PlayersTab({ data }: { data: MonitoringServerDetail }) {
  const [query, setQuery] = useState("");
  const q = query.trim().toLowerCase();
  const players = data.players.filter((p) => !q || p.name.toLowerCase().includes(q));

  return (
    <div className="flex flex-col gap-3.5">
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex h-[34px] w-60 items-center gap-2 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-bg)] px-2.5">
          <i className="ri-search-line text-[14px] text-[var(--vx-muted)]" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t("monitoring.players.search_placeholder")}
            className="min-w-0 flex-1 bg-transparent text-[12px] text-[var(--vx-fg)] outline-none placeholder:text-[var(--vx-faint)]"
          />
        </div>
        <span className="mr-auto text-[12px] text-[var(--vx-muted)]">
          {t("monitoring.players.count_summary", {
            count: data.players.length,
            avg: data.avg_24h,
          })}
        </span>
        <span className="text-[11px] text-[var(--vx-muted)]">
          {t("monitoring.players.peak_24h", { count: data.peak_24h })}
        </span>
      </div>

      <div className="overflow-hidden rounded-[10px] border border-[var(--vx-border)]">
        <table className="w-full border-collapse text-left">
          <thead>
            <tr className="bg-[var(--vx-card-2)]">
              <th className={cn(MON_TH, "px-3.5")}>{t("monitoring.players.col_nick")}</th>
              <th className={MON_TH}>{t("monitoring.players.col_frags")}</th>
              <th className={MON_TH}>{t("monitoring.players.col_session")}</th>
              <th className={cn(MON_TH, "px-3.5 text-right")}>{t("monitoring.col.ping")}</th>
            </tr>
          </thead>
          <tbody>
            {players.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-[12px] text-[var(--vx-muted)]">
                  {data.server.status === "running"
                    ? t("monitoring.players.empty_running")
                    : t("monitoring.players.empty_offline")}
                </td>
              </tr>
            ) : (
              players.map((p) => (
                <tr key={p.name} className={MON_ROW}>
                  <td className="px-3.5 py-2.5">
                    <div className="flex items-center gap-2.5">
                      <span className="flex h-6 w-6 items-center justify-center rounded-[6px] bg-[var(--vx-tint)] text-[10px] font-semibold text-[var(--vx-dim)]">
                        {initials(p.name)}
                      </span>
                      <span className="text-[13px]">{p.name}</span>
                    </div>
                  </td>
                  <td className="px-3 py-2.5 font-mono text-[12px]">{p.score}</td>
                  <td className="px-3 py-2.5 font-mono text-[12px] text-[var(--vx-dim)]">
                    {p.duration_sec ? secondsToHms(p.duration_sec) : "—"}
                  </td>
                  <td
                    className="px-3.5 py-2.5 text-right font-mono text-[12px]"
                    style={{ color: pingColor(p.ping) }}
                  >
                    {pingText(p.ping)}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function StatsTab({ serverId, slots }: { serverId: string; slots: number }) {
  const day = useQuery({
    queryKey: queryKeys.monitoringServerStats(serverId, 1),
    queryFn: () => fetchServerMonitoringStats(serverId, 1),
  });
  const week = useQuery({
    queryKey: queryKeys.monitoringServerStats(serverId, 7),
    queryFn: () => fetchServerMonitoringStats(serverId, 7),
  });

  return (
    <div className="flex flex-col gap-4">
      <OnlineChart
        title={t("monitoring.chart.online_24h")}
        subtitle={t("monitoring.chart.step_10m")}
        loading={day.isLoading}
        labels={day.data?.series?.labels ?? []}
        values={(day.data?.series?.online ?? []).map((v) => Number(v ?? 0))}
        cap={day.data?.series?.max_cap ?? slots}
      />
      <OnlineChart
        title={t("monitoring.chart.online_7d")}
        subtitle={t("monitoring.chart.step_1h")}
        loading={week.isLoading}
        labels={week.data?.series?.labels ?? []}
        values={(week.data?.series?.online ?? []).map((v) => Number(v ?? 0))}
        cap={week.data?.series?.max_cap ?? slots}
      />
    </div>
  );
}

function OnlineChart({
  title,
  subtitle,
  labels,
  values,
  cap,
  loading,
}: {
  title: string;
  subtitle: string;
  labels: string[];
  values: number[];
  cap: number | null;
  loading: boolean;
}) {
  const top = Math.max(cap || 0, ...values, 1);
  const slotLine = values.map(() => cap || 0);
  const ticks = useMemo(() => {
    if (labels.length <= 6) return labels;
    const step = Math.floor(labels.length / 6);
    return labels.filter((_, i) => i % step === 0).slice(0, 7);
  }, [labels]);

  const summary = values.length
    ? [
        {
          label: t("monitoring.chart.avg_online"),
          value: String(Math.round(values.reduce((a, b) => a + b, 0) / values.length)),
        },
        { label: t("monitoring.chart.peak_label"), value: String(Math.max(...values)) },
        { label: t("monitoring.chart.min"), value: String(Math.min(...values)) },
        {
          label: t("monitoring.chart.fill"),
          value: cap ? `${Math.round((Math.max(...values) / cap) * 100)}%` : "—",
        },
      ]
    : [];

  return (
    <div className={cn("p-4", MON_INNER)}>
      <div className="flex items-baseline gap-3">
        <span className="text-[13px] font-semibold">{title}</span>
        <span className="text-[11px] text-[var(--vx-muted)]">{subtitle}</span>
        <span className="ml-auto flex items-center gap-3.5 text-[11px] text-[var(--vx-muted)]">
          <span className="flex items-center gap-1.5">
            <span className="h-[2px] w-3.5 bg-[var(--vx-fg-strong)]" />
            {t("monitoring.chart.legend_online")}
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-3.5 border-t-2 border-dashed border-[var(--vx-faint)]" />
            {t("monitoring.chart.legend_slots")}
          </span>
        </span>
      </div>

      {loading ? (
        <Skeleton className="mt-3.5 h-40 w-full" />
      ) : values.length === 0 ? (
        <p className="py-12 text-center text-[12px] text-[var(--vx-muted)]">
          {t("monitoring.chart.empty")}
        </p>
      ) : (
        <>
          <div className="mt-3.5 flex gap-2.5">
            <div className="flex h-40 w-[26px] flex-col justify-between text-right font-mono text-[10px] text-[var(--vx-faint)]">
              <span>{top}</span>
              <span>{Math.round(top / 2)}</span>
              <span>0</span>
            </div>
            <div className="min-w-0 flex-1">
              <svg viewBox="0 0 600 160" preserveAspectRatio="none" className="block h-40 w-full">
                <line x1="0" y1="0.5" x2="600" y2="0.5" stroke="var(--vx-inset)" strokeWidth="1" />
                <line x1="0" y1="80" x2="600" y2="80" stroke="var(--vx-inset)" strokeWidth="1" />
                <line x1="0" y1="159.5" x2="600" y2="159.5" stroke="var(--vx-border)" strokeWidth="1" />
                <path d={areaPath(values, 600, 160, top)} fill="var(--vx-veil-strong)" />
                {!!cap && (
                  <polyline
                    points={polyPoints(slotLine, 600, 160, top)}
                    fill="none"
                    stroke="var(--vx-faint)"
                    strokeWidth="1.5"
                    strokeDasharray="5 4"
                    vectorEffect="non-scaling-stroke"
                  />
                )}
                <polyline
                  points={polyPoints(values, 600, 160, top)}
                  fill="none"
                  stroke="var(--vx-fg-strong)"
                  strokeWidth="1.75"
                  vectorEffect="non-scaling-stroke"
                />
              </svg>
              <div className="mt-2 flex justify-between font-mono text-[10px] text-[var(--vx-faint)]">
                {ticks.map((l, i) => (
                  <span key={`${l}-${i}`}>{l}</span>
                ))}
              </div>
            </div>
          </div>

          <div className="mt-3.5 grid grid-cols-[repeat(auto-fit,minmax(120px,1fr))] gap-3 border-t border-[var(--vx-border)] pt-3.5">
            {summary.map((sm) => (
              <div key={sm.label}>
                <div className="text-[10px] uppercase tracking-[0.06em] text-[var(--vx-muted)]">
                  {sm.label}
                </div>
                <div className="mt-1 font-mono text-[14px]">{sm.value}</div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}

const BANNER_SIZES = ["560x95", "160x250"] as const;

function BannersTab({ data }: { data: MonitoringServerDetail }) {
  const [nonce, setNonce] = useState(0);

  const bannerSrc = (size: string) =>
    `${data.banner_url}/${size}.png${nonce ? `?t=${nonce}` : ""}`;
  const bannerPlain = (size: string) => `${data.banner_url}/${size}.png`;

  return (
    <div className="flex flex-col gap-4">
      {!data.settings.public_enabled && (
        <div className="rounded-[12px] border border-[rgba(232,160,60,0.28)] bg-[rgba(232,160,60,0.06)] px-4 py-3 text-[12px] text-[var(--vx-warn)]">
          {t("monitoring.banners.public_off")}
        </div>
      )}
      <div className="grid grid-cols-[repeat(auto-fit,minmax(320px,1fr))] gap-4">
        {BANNER_SIZES.map((size) => (
          <div key={size} className={cn("flex flex-col gap-3 p-4", MON_INNER)}>
            <div className="flex items-center gap-2">
              <span className="text-[13px] font-semibold">
                {t("monitoring.banners.size_title", { size })}
              </span>
              <span className="font-mono text-[11px] text-[var(--vx-muted)]">
                {t("monitoring.banners.png_note")}
              </span>
              <button
                type="button"
                onClick={() => setNonce(Date.now())}
                className="ml-auto text-[11px] text-[var(--vx-muted)] hover:text-white"
              >
                {t("monitoring.banners.refresh_preview")}
              </button>
            </div>
            <div className="flex min-h-[120px] items-center justify-center rounded-[10px] border border-[var(--vx-border)] bg-[var(--vx-card-2)] p-4">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={bannerSrc(size)}
                alt={t("monitoring.banners.size_title", { size })}
                className="h-auto max-w-full"
                onError={(e) => {
                  e.currentTarget.style.display = "none";
                  const ph = e.currentTarget.nextElementSibling as HTMLElement | null;
                  if (ph) ph.style.display = "flex";
                }}
              />
              <div className="hidden items-center gap-2 text-[11px] text-[var(--vx-muted)]">
                <i className="ri-image-line text-lg" />
                {t("monitoring.banners.unavailable")}
              </div>
            </div>
            <CodeBox
              label={t("monitoring.banners.code_site")}
              value={`<a href="${data.public_url}"><img src="${bannerPlain(size)}" alt="${data.server.name}" /></a>`}
            />
            <CodeBox
              label={t("monitoring.banners.code_forum")}
              value={`[url=${data.public_url}][img]${bannerPlain(size)}[/img][/url]`}
            />
          </div>
        ))}
      </div>
    </div>
  );
}

function CodeBox({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="flex items-center gap-2">
        <span className="text-[11px] font-semibold text-[var(--vx-muted)]">{label}</span>
        <button
          type="button"
          className="ml-auto flex items-center gap-1.5 text-[11px] text-[var(--vx-muted)] hover:text-white"
          onClick={() => {
            void navigator.clipboard?.writeText(value);
            toast.success(t("monitoring.code.copied"));
          }}
        >
          <i className="ri-file-copy-line text-[13px]" />
          {t("monitoring.code.copy")}
        </button>
      </div>
      <textarea
        readOnly
        value={value}
        className="mt-1.5 h-16 w-full resize-none rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-inset)] px-2.5 py-2 font-mono text-[11px] text-[var(--vx-dim)] outline-none"
      />
    </div>
  );
}

type LogLevel = "all" | "INFO" | "WARN" | "ERROR";

type ParsedLine = { time: string; level: Exclude<LogLevel, "all">; text: string };

function parseLogLine(raw: string): ParsedLine {
  const time = raw.match(/\b(\d{2}:\d{2}:\d{2})\b/)?.[1] ?? "";
  const upper = raw.toUpperCase();
  const level: ParsedLine["level"] = upper.includes("ERROR") || upper.includes("SEVERE")
    ? "ERROR"
    : upper.includes("WARN")
      ? "WARN"
      : "INFO";
  const text = raw
    .replace(/^\[?\d{4}-\d{2}-\d{2}[T ]?/, "")
    .replace(/^\[?\d{2}:\d{2}:\d{2}\]?\s*/, "")
    .replace(/^\[?(INFO|WARN|WARNING|ERROR|SEVERE)\]?:?\s*/i, "")
    .trim();
  return { time, level, text: text || raw };
}

const LEVEL_COLOR: Record<ParsedLine["level"], string> = {
  INFO: MON.info,
  WARN: MON.warn,
  ERROR: MON.bad,
};

function ConsoleTab({ serverId }: { serverId: string }) {
  const [level, setLevel] = useState<LogLevel>("all");

  const logs = useQuery({
    queryKey: queryKeys.monitoringLogs(serverId),
    queryFn: () => fetchServerLogs(serverId),
    refetchInterval: 10_000,
    refetchIntervalInBackground: false,
  });

  const lines = useMemo(
    () => (logs.data?.lines ?? []).map(parseLogLine),
    [logs.data]
  );
  const visible = lines.filter((l) => level === "all" || l.level === level);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        {(["all", "INFO", "WARN", "ERROR"] as LogLevel[]).map((l) => (
          <Chip key={l} active={level === l} onClick={() => setLevel(l)}>
            {l === "all" ? t("common.all") : l}
          </Chip>
        ))}
        <span className="ml-auto flex items-center gap-1.5 text-[11px] text-[var(--vx-muted)]">
          <span
            className="h-1.5 w-1.5 rounded-full"
            style={{ background: logs.isError ? MON.bad : MON.ok }}
          />
          {logs.isError
            ? t("monitoring.console.stream_down")
            : t("monitoring.console.stream_live")}
        </span>
      </div>

      <div className="max-h-[380px] overflow-y-auto rounded-[10px] border border-[var(--vx-border)] bg-[var(--vx-code)] px-3.5 py-3 font-mono text-[12px] leading-[1.7]">
        {logs.isLoading ? (
          <Skeleton className="h-40 w-full" />
        ) : visible.length === 0 ? (
          <p className="py-8 text-center text-[var(--vx-faint)]">
            {logs.isError
              ? t("monitoring.console.logs_failed")
              : t("monitoring.console.no_records")}
          </p>
        ) : (
          visible.map((l, i) => (
            <div key={i} className="flex gap-3">
              <span className="shrink-0 text-[var(--vx-faint)]">{l.time || "--:--:--"}</span>
              <span
                className="w-12 shrink-0 font-medium"
                style={{ color: LEVEL_COLOR[l.level] }}
              >
                {l.level}
              </span>
              <span className="break-words text-[var(--vx-dim)]">{l.text}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function PublicTab({ data }: { data: MonitoringServerDetail }) {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<MonitoringSettings>(data.settings);
  const [tagDraft, setTagDraft] = useState("");

  const [dirty, setDirty] = useState(false);
  useEffect(() => {
    if (!dirty) setForm(data.settings);
  }, [data.settings, dirty]);

  const patch = (next: Partial<MonitoringSettings>) => {
    setDirty(true);
    setForm((prev) => ({ ...prev, ...next }));
  };

  const save = useMutation({
    mutationFn: (payload: Partial<MonitoringSettings>) =>
      updateMonitoringSettings(data.server.id, payload),
    onSuccess: (res) => {
      setDirty(false);
      setForm(res.settings);
      toast.success(t("monitoring.settings.saved"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.monitoringServer(data.server.id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.monitoring });
      void queryClient.invalidateQueries({ queryKey: ["monitoring-top"] });
    },
    onError: () => toast.error(t("monitoring.settings.save_failed")),
  });

  const addTag = () => {
    const tag = tagDraft.trim();
    if (!tag || form.tags.includes(tag)) {
      setTagDraft("");
      return;
    }
    patch({ tags: [...form.tags, tag] });
    setTagDraft("");
  };

  const toggles: { key: keyof MonitoringSettings; label: string }[] = [
    { key: "show_players", label: t("monitoring.public_tab.show_players") },
    { key: "show_chart", label: t("monitoring.public_tab.show_chart") },
    { key: "show_incidents", label: t("monitoring.public_tab.show_incidents") },
    { key: "show_address", label: t("monitoring.public_tab.show_address") },
    { key: "show_version", label: t("monitoring.public_tab.show_version") },
  ];

  const maxVisits = Math.max(1, ...(data.visits.days.length ? data.visits.days : [0]));

  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)]">
      <div className="flex flex-col gap-3.5">
        <div className={cn("flex items-center gap-3 p-4", MON_INNER)}>
          <div className="flex-1">
            <div className="text-[13px] font-semibold">
              {t("monitoring.public_tab.state", {
                state: form.public_enabled
                  ? t("monitoring.public_tab.enabled")
                  : t("monitoring.public_tab.disabled"),
              })}
            </div>
            <div className="mt-[3px] text-[12px] text-[var(--vx-muted)]">
              {t("monitoring.public_tab.state_hint")}
            </div>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={form.public_enabled}
            aria-label={t("monitoring.tab.public")}
            onClick={() => patch({ public_enabled: !form.public_enabled })}
            className="relative h-6 w-11 shrink-0 rounded-full transition-colors"
            style={{ background: form.public_enabled ? MON.primary : "var(--vx-border-strong)" }}
          >
            <span
              className="absolute top-[3px] h-[18px] w-[18px] rounded-full transition-[left]"
              style={{
                left: form.public_enabled ? 23 : 3,
                background: form.public_enabled ? "var(--vx-on-fill)" : MON.mut,
              }}
            />
          </button>
        </div>

        <div className={cn("flex flex-col gap-3.5 p-4", MON_INNER)}>
          <div>
            <label className="block text-[11px] font-semibold text-[var(--vx-muted)]">
              {t("monitoring.public_tab.link")}
            </label>
            <button
              type="button"
              onClick={() => {
                void navigator.clipboard?.writeText(data.public_url);
                toast.success(t("monitoring.public_tab.link_copied"));
              }}
              className="mt-1.5 flex w-full items-center gap-2 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-inset)] px-3 py-2.5 text-left font-mono text-[12px] text-[var(--vx-dim)] transition-colors hover:border-[var(--vx-border-hover)]"
            >
              <span className="flex-1 truncate">{data.public_url}</span>
              <i className="ri-file-copy-line text-[13px]" />
            </button>
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-[var(--vx-muted)]">
              {t("common.title")}
            </label>
            <input
              value={form.title}
              onChange={(e) => patch({ title: e.target.value })}
              placeholder={data.server.name}
              className={cn(MON_INPUT, "mt-1.5")}
            />
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-[var(--vx-muted)]">
              {t("monitoring.public_tab.description")}
            </label>
            <textarea
              value={form.description}
              onChange={(e) => patch({ description: e.target.value })}
              className={cn(MON_INPUT, "mt-1.5 h-[88px] resize-none leading-[1.5]")}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-[11px] font-semibold text-[var(--vx-muted)]">Discord</label>
              <input
                value={form.discord}
                onChange={(e) => patch({ discord: e.target.value })}
                placeholder="discord.gg/…"
                className={cn(MON_INPUT, "mt-1.5")}
              />
            </div>
            <div>
              <label className="block text-[11px] font-semibold text-[var(--vx-muted)]">
                {t("monitoring.public_tab.website")}
              </label>
              <input
                value={form.website}
                onChange={(e) => patch({ website: e.target.value })}
                placeholder="example.com"
                className={cn(MON_INPUT, "mt-1.5")}
              />
            </div>
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-[var(--vx-muted)]">
              {t("monitoring.public_tab.tags")}
            </label>
            <div className="mt-2 flex flex-wrap items-center gap-1.5">
              {form.tags.map((tag) => (
                <span
                  key={tag}
                  className="flex items-center gap-1.5 rounded-full border border-[var(--vx-border)] bg-[var(--vx-inset)] px-2.5 py-[5px] text-[12px] text-[var(--vx-dim)]"
                >
                  {tag}
                  <button
                    type="button"
                    aria-label={t("monitoring.public_tab.tag_remove", { tag })}
                    onClick={() => patch({ tags: form.tags.filter((t) => t !== tag) })}
                  >
                    <i className="ri-close-line text-[12px] text-[var(--vx-muted)] hover:text-white" />
                  </button>
                </span>
              ))}
              <input
                value={tagDraft}
                onChange={(e) => setTagDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    addTag();
                  }
                }}
                onBlur={addTag}
                placeholder={t("monitoring.public_tab.tag_placeholder")}
                className="w-24 rounded-full border border-dashed border-[var(--vx-border-strong)] bg-transparent px-2.5 py-[5px] text-[12px] text-[var(--vx-fg)] outline-none placeholder:text-[var(--vx-muted)]"
              />
            </div>
          </div>

          <div className="flex gap-2 pt-1">
            <button
              type="button"
              className={MON_BTN_PRIMARY}
              disabled={save.isPending}
              onClick={() => save.mutate(form)}
            >
              {save.isPending ? t("common.saving") : t("common.save")}
            </button>
            {form.public_enabled && (
              <Link href={`/monitoring/public/${data.server.id}`} className={MON_BTN}>
                {t("monitoring.public_tab.preview")}
              </Link>
            )}
          </div>
        </div>
      </div>

      <div className="flex flex-col gap-3.5">
        <div className={cn("p-4", MON_INNER)}>
          <div className="text-[13px] font-semibold">
            {t("monitoring.public_tab.visible_title")}
          </div>
          <div className="mt-3 flex flex-col gap-2.5">
            {toggles.map((t) => {
              const checked = Boolean(form[t.key]);
              return (
                <button
                  key={t.key}
                  type="button"
                  role="checkbox"
                  aria-checked={checked}
                  onClick={() => patch({ [t.key]: !checked } as Partial<MonitoringSettings>)}
                  className="flex items-center gap-2.5 text-left"
                >
                  <span
                    className="flex h-4 w-4 items-center justify-center rounded-[4px]"
                    style={{ background: checked ? MON.primary : "var(--vx-border-strong)" }}
                  >
                    {checked && <i className="ri-check-line text-[11px] text-[var(--vx-on-fill)]" />}
                  </span>
                  <span className="flex-1 text-[12px] text-[var(--vx-dim)]">{t.label}</span>
                </button>
              );
            })}
          </div>
        </div>

        <div className={cn("p-4", MON_INNER)}>
          <div className="text-[13px] font-semibold">
            {t("monitoring.public_tab.visits")}
          </div>
          <div className="mt-2.5 font-mono text-[26px] font-medium">{data.visits.total}</div>
          <div className="mt-3 flex h-14 items-end gap-1">
            {data.visits.days.map((v, i) => (
              <div
                key={i}
                title={t("monitoring.public_tab.visits_bar", { count: v })}
                className="flex-1 rounded-[3px] bg-[var(--vx-border-strong)]"
                style={{ height: `${Math.max(4, (v / maxVisits) * 100)}%` }}
              />
            ))}
          </div>
        </div>

        <div className={cn("p-4", MON_INNER)}>
          <div className="text-[13px] font-semibold">{t("monitoring.public_tab.votes")}</div>
          <div className="mt-2.5 font-mono text-[26px] font-medium">{form.votes}</div>
          <p className="mt-1.5 text-[11px] text-[var(--vx-muted)]">
            {t("monitoring.public_tab.votes_hint")}
          </p>
        </div>
      </div>
    </div>
  );
}

export function IncidentsTab({
  uptime,
  days,
  incidents,
}: {
  uptime: number;
  days: MonitoringUptimeDay[];
  incidents: MonitoringIncident[];
}) {
  return (
    <div className="flex flex-col gap-4">
      <div className={cn("p-4", MON_INNER)}>
        <div className="flex flex-wrap items-baseline gap-2.5">
          <span className="text-[13px] font-semibold">
            {t("monitoring.incidents.uptime_30d")}
          </span>
          <span className="ml-auto font-mono text-[13px]">{uptimeText(uptime)}</span>
        </div>
        <div className="mt-3.5 flex h-11 gap-[3px]">
          {days.map((d) => (
            <div
              key={d.day}
              title={
                d.has_data
                  ? t("monitoring.incidents.day_tooltip", {
                      day: d.day,
                      uptime: d.uptime.toFixed(2),
                    })
                  : t("monitoring.incidents.day_no_data", { day: d.day })
              }
              className="flex-1 rounded-[3px]"
              style={uptimeDayStyle(d)}
            />
          ))}
        </div>
        <div className="mt-2 flex justify-between font-mono text-[10px] text-[var(--vx-faint)]">
          <span>{t("monitoring.incidents.range_start")}</span>
          <span>{t("monitoring.incidents.range_end")}</span>
        </div>
      </div>

      <div className={cn("overflow-hidden", MON_INNER)}>
        <div className="border-b border-[var(--vx-border)] px-4 py-3.5 text-[13px] font-semibold">
          {t("monitoring.incidents.title")}
        </div>
        {incidents.length === 0 ? (
          <p className="px-4 py-8 text-center text-[12px] text-[var(--vx-muted)]">
            {t("monitoring.incidents.empty")}
          </p>
        ) : (
          incidents.map((inc) => {
            const tone =
              inc.level === "bad" ? MON.bad : inc.level === "warn" ? MON.warn : MON.info;
            return (
              <div key={inc.id} className="flex gap-3.5 border-b border-[var(--vx-divider)] px-4 py-3.5">
                <span
                  className="mt-1.5 h-2 w-2 shrink-0 rounded-full"
                  style={{ background: tone }}
                />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-[13px] font-semibold">{inc.title}</span>
                    <span
                      className="rounded-full bg-[var(--vx-veil)] px-2.5 py-[3px] text-[11px] font-semibold"
                      style={{ color: tone }}
                    >
                      {inc.resolved
                        ? t("monitoring.incidents.resolved")
                        : t("monitoring.incidents.active")}
                    </span>
                  </div>
                  <p className="mt-1.5 text-[12px] text-pretty text-[var(--vx-muted)]">{inc.body}</p>
                </div>
                <div className="shrink-0 text-right font-mono text-[11px] text-[var(--vx-muted)]">
                  <div>{shortDate(inc.started_at)}</div>
                  <div className="mt-[3px]">{durationText(inc.duration_sec)}</div>
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}

function uptimeDayStyle(d: MonitoringUptimeDay): React.CSSProperties {
  if (!d.has_data) return { background: "var(--vx-tint)" };
  if (d.uptime >= 99.5) return { background: MON.ok, opacity: 0.55 };
  if (d.uptime >= 95) return { background: MON.warn, opacity: 0.9 };
  return { background: MON.bad, opacity: 0.9 };
}
