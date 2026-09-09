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

const GRID = "grid-cols-[minmax(0,1fr)_repeat(4,64px)] sm:grid-cols-[minmax(0,1fr)_repeat(4,80px)]";

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

  if (isLoading) return <Skeleton className="h-[420px] w-full rounded-[14px]" />;

  return (
    <div className="flex flex-col gap-[18px]">
      <Panel
        title={t("servers.metrics.live_title")}
        aside={
          <span className={cn("font-mono text-[11px]", VX_FAINT)}>
            {t("servers.metrics.refresh_note")}
          </span>
        }
      >
        {metrics.length === 0 ? (
          <EmptyState>{t("servers.overview.no_period_data")}</EmptyState>
        ) : (
          <>
            <svg
              viewBox={`0 0 ${CHART_W} ${CHART_H}`}
              preserveAspectRatio="none"
              className="block h-[190px] w-full"
            >
              {[0.25, 0.5, 0.75].map((f) => (
                <line
                  key={f}
                  x1={0}
                  x2={CHART_W}
                  y1={CHART_H - f * (CHART_H - 6) - 3}
                  y2={CHART_H - f * (CHART_H - 6) - 3}
                  stroke="var(--vx-border)"
                  strokeWidth={0.5}
                />
              ))}
              <polyline points={charts.cpu} fill="none" stroke="var(--vx-fg-strong)" strokeWidth={1.6} />
              <polyline
                points={charts.ram}
                fill="none"
                stroke="var(--vx-faint)"
                strokeWidth={1.6}
                strokeDasharray="5 4"
              />
            </svg>
            <div className={cn("mt-2.5 flex justify-between font-mono text-[10.5px]", "text-[var(--vx-faint)]")}>
              <span>{t("servers.overview.chart_earlier")}</span>
              <span>{t("servers.overview.chart_legend")}</span>
              <span>
                {t("servers.metrics.scale_max", { value: String(charts.scaleMax) })}
              </span>
              <span>{t("servers.overview.chart_now")}</span>
            </div>
          </>
        )}
      </Panel>

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
        <div className="overflow-x-auto px-[18px] pt-1.5 pb-4">
          <div className="min-w-[420px]">
            <div
              className={cn(
                "grid gap-3 py-2.5 text-[10.5px] tracking-[0.08em] uppercase",
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
                    "grid gap-3 py-2.5 font-mono text-[12px] text-[var(--vx-dim)] last:border-b-0",
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
        </div>
      </Panel>
    </div>
  );
}
