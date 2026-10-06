"use client";

import { useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import {
  EmptyState,
  Panel,
  VX_FAINT,
  VX_ROW_LINE,
  VX_SELECT,
} from "@/components/vx/panel-ui";
import { VxInlineLoader } from "@/components/vx/loader";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchServerMonitoringStats, type MetricPoint } from "@/lib/api";
import { cn } from "@/lib/utils";
import { useServerMetrics } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";

const HISTORY_PERIODS = [1, 7, 14, 30, 60, 90, 180] as const;

const CHART_W = 600;
const CHART_H = 170;

const GRID = "grid-cols-[minmax(0,1fr)_repeat(4,50px)] sm:grid-cols-[minmax(0,1fr)_repeat(4,80px)]";

function chartScaleMax(...series: number[][]): number {
  let peak = 0;
  for (const values of series) {
    for (const v of values) {
      if (Number.isFinite(v) && v > peak) peak = v;
    }
  }
  if (peak <= 0) return 10;
  return Math.min(100, Math.max(10, Math.ceil(peak / 10) * 10));
}

function polyline(values: number[], scaleMax: number): string {
  if (values.length === 0) return "";
  const pts = values.length === 1 ? [values[0], values[0]] : values;
  const stepX = CHART_W / (pts.length - 1);
  return pts
    .map((v, i) => {
      const clamped = Math.max(0, Math.min(scaleMax, Number.isFinite(v) ? v : 0));
      const y = CHART_H - (clamped / scaleMax) * (CHART_H - 6) - 3;
      return `${(i * stepX).toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");
}

function memPercent(p: MetricPoint): number {
  return p.mem_limit_mb > 0 ? (p.mem_used_mb / p.mem_limit_mb) * 100 : 0;
}

function pctText(value: number | null): string {
  return value === null || !Number.isFinite(value) ? "—" : `${Math.round(value)}%`;
}


function areaPoints(values: number[], scaleMax: number): string {
  const line = polyline(values, scaleMax);
  return line ? `0,${CHART_H} ${line} ${CHART_W},${CHART_H}` : "";
}

function MetricChart({
  title,
  big,
  bigSub,
  values,
  scaleMax,
  color,
  gradientId,
}: {
  title: string;
  big: string;
  bigSub: string;
  values: number[];
  scaleMax: number;
  color: string;
  gradientId: string;
}) {
  const t = useT();
  return (
    <Panel title={title}>
      <div className="text-[30px] leading-none font-semibold tracking-[-0.03em] text-[var(--vx-fg-strong)]">
        {big}
        <span className="ml-2.5 text-[13px] font-normal tracking-normal text-[var(--vx-faint)]">
          {bigSub}
        </span>
      </div>
      {values.length === 0 ? (
        <EmptyState>{t("servers.overview.no_period_data")}</EmptyState>
      ) : (
        <>
          <svg
            viewBox={`0 0 ${CHART_W} ${CHART_H}`}
            preserveAspectRatio="none"
            className="mt-3 block h-[150px] w-full"
          >
            <defs>
              <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0" stopColor={color} stopOpacity="0.18" />
                <stop offset="1" stopColor={color} stopOpacity="0" />
              </linearGradient>
            </defs>
            {[0.25, 0.5, 0.75].map((f) => (
              <line
                key={f}
                x1={0}
                x2={CHART_W}
                y1={CHART_H - f * (CHART_H - 6) - 3}
                y2={CHART_H - f * (CHART_H - 6) - 3}
                stroke="var(--vx-border)"
                strokeWidth={1}
              />
            ))}
            <polygon points={areaPoints(values, scaleMax)} fill={`url(#${gradientId})`} />
            <polyline
              points={polyline(values, scaleMax)}
              fill="none"
              stroke={color}
              strokeWidth={2}
              strokeLinejoin="round"
              vectorEffect="non-scaling-stroke"
            />
          </svg>
          <div className="mt-1.5 flex justify-between font-mono text-[11px] text-[var(--vx-ghost)]">
            <span>{t("servers.overview.chart_earlier")}</span>
            <span>{t("servers.overview.chart_now")}</span>
          </div>
        </>
      )}
    </Panel>
  );
}

export function ServerMetricsTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const [historyPeriod, setHistoryPeriod] = useState<number>(7);
  const { data: metrics = [], isLoading } = useServerMetrics(id);

  const historyQuery = useQuery({
    queryKey: ["server-monitoring-stats", id, historyPeriod],
    queryFn: () => fetchServerMonitoringStats(id, historyPeriod),
    enabled: !!id,
  });

  const charts = useMemo(() => {
    const tail = metrics.slice(-90);
    const cpuValues = tail.map((p) => p.cpu_pct);
    const ramValues = tail.map(memPercent);
    const scaleMax = chartScaleMax(cpuValues, ramValues);
    return {
      cpu: polyline(cpuValues, scaleMax),
      ram: polyline(ramValues, scaleMax),
      scaleMax,
    };
  }, [metrics]);

  const series = historyQuery.data?.ok ? historyQuery.data.series : null;
  const historyRows =
    series?.labels.map((label, i) => ({
      label,
      online: series.online[i] ?? null,
      cpu: series.cpu[i] ?? null,
      ram: series.ram[i] ?? null,
      disk: series.disk[i] ?? null,
    })) ?? [];

  const cpuValues = metrics.slice(-90).map((p) => p.cpu_pct);
  const ramValues = metrics.slice(-90).map(memPercent);
  const lastMetric = metrics[metrics.length - 1];
  const diskValues = historyRows.flatMap((r) => (r.disk === null ? [] : [r.disk]));
  const onlineValues = historyRows.flatMap((r) => (r.online === null ? [] : [r.online]));
  const onlineMax = Math.max(5, ...onlineValues);

  if (isLoading) return <Skeleton className="h-[420px] w-full rounded-[14px]" />;

  return (
    <div className="flex flex-col gap-[18px]">
      <div className="grid grid-cols-1 gap-[18px] lg:grid-cols-2">
        <MetricChart
          title="CPU"
          big={cpuValues.length ? `${cpuValues[cpuValues.length - 1].toFixed(1)}%` : "—"}
          bigSub={t("servers.metrics.refresh_note")}
          values={cpuValues}
          scaleMax={charts.scaleMax}
          color="var(--srv-green)"
          gradientId="srv2-m-cpu"
        />
        <MetricChart
          title="RAM"
          big={ramValues.length ? `${ramValues[ramValues.length - 1].toFixed(1)}%` : "—"}
          bigSub={
            lastMetric && lastMetric.mem_limit_mb > 0
              ? `${(lastMetric.mem_used_mb / 1024).toFixed(2)} / ${(lastMetric.mem_limit_mb / 1024).toFixed(2)} GB`
              : ""
          }
          values={ramValues}
          scaleMax={100}
          color="var(--srv-warn)"
          gradientId="srv2-m-ram"
        />
        <MetricChart
          title={t("servers.metrics.col_disk")}
          big={diskValues.length ? `${diskValues[diskValues.length - 1].toFixed(0)}%` : "—"}
          bigSub={t("servers.metrics.period_days", { days: historyPeriod })}
          values={diskValues}
          scaleMax={100}
          color="var(--srv-info)"
          gradientId="srv2-m-disk"
        />
        <MetricChart
          title={t("servers.metrics.col_online")}
          big={onlineValues.length ? String(Math.round(onlineValues[onlineValues.length - 1])) : "—"}
          bigSub={t("servers.metrics.period_days", { days: historyPeriod })}
          values={onlineValues}
          scaleMax={onlineMax}
          color="var(--srv-violet)"
          gradientId="srv2-m-online"
        />
      </div>

      <Panel
        title={t("servers.metrics.history")}
        aside={
          <select
            className={cn(VX_SELECT, "h-[30px] rounded-[8px] text-[12px]")}
            value={historyPeriod}
            onChange={(e) => setHistoryPeriod(Number(e.target.value))}
          >
            {HISTORY_PERIODS.map((p) => (
              <option key={p} value={p}>
                {t("servers.metrics.period_days", { days: p })}
              </option>
            ))}
          </select>
        }
        flush
      >
        <div className="px-4 pt-1.5 pb-4 sm:px-6">
          <div
            className={cn(
              "grid gap-2 py-2.5 text-[10px] tracking-[0.04em] uppercase sm:gap-3 sm:text-[10.5px] sm:tracking-[0.08em]",
              GRID,
              VX_ROW_LINE,
              VX_FAINT
            )}
          >
            <span>{t("servers.metrics.col_date")}</span>
            <span>{t("servers.metrics.col_online")}</span>
            <span>CPU</span>
            <span>RAM</span>
            <span>{t("servers.metrics.col_disk")}</span>
          </div>

          {historyQuery.isLoading ? (
            <VxInlineLoader label={t("servers.metrics.history_loading")} />
          ) : historyRows.length === 0 ? (
            <EmptyState>{t("servers.metrics.history_empty")}</EmptyState>
          ) : (
            historyRows.map((r) => (
              <div
                key={r.label}
                className={cn(
                  "grid gap-2 py-2.5 font-mono text-[12px] text-[var(--vx-dim)] last:border-b-0 sm:gap-3",
                  GRID,
                  VX_ROW_LINE
                )}
              >
                <span>{r.label}</span>
                <span>{r.online ?? "—"}</span>
                <span>{pctText(r.cpu)}</span>
                <span>{pctText(r.ram)}</span>
                <span>{pctText(r.disk)}</span>
              </div>
            ))
          )}
        </div>
      </Panel>
    </div>
  );
}
