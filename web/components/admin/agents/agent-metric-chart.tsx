"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Skeleton } from "@/components/ui/skeleton";
import { Panel, SubTabs, VX_FAINT } from "@/components/vx/panel-ui";
import { fetchAgentMetrics, type AgentMetricPoint, type AgentMetricRange } from "@/lib/api";
import { formatDateTime, formatMB } from "@/lib/agents";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

type Series = { key: string; label: string; color: string; points: AgentMetricPoint[] };

const W = 720;
const H = 200;
const PAD_L = 44;
const PAD_R = 8;
const PAD_T = 10;
const PAD_B = 22;

function niceMax(v: number, percent: boolean): number {
  if (v <= 0) return percent ? 10 : 1;
  const padded = v * 1.15;
  if (percent) return Math.min(100, Math.max(5, Math.ceil(padded / 5) * 5));
  const mag = 10 ** Math.floor(Math.log10(padded));
  return Math.ceil(padded / mag) * mag;
}

function segments(points: AgentMetricPoint[], x: (i: number) => number, y: (v: number) => number): string[] {
  const out: string[] = [];
  let cur = "";
  points.forEach((p, i) => {
    if (p.v == null) {
      if (cur) out.push(cur);
      cur = "";
      return;
    }
    cur += `${cur ? "L" : "M"}${x(i).toFixed(1)},${y(p.v).toFixed(1)}`;
  });
  if (cur) out.push(cur);
  return out;
}

function Chart({ series, percent, empty }: { series: Series[]; percent: boolean; empty: string }) {
  const [hover, setHover] = useState<number | null>(null);
  const len = series[0]?.points.length ?? 0;
  const max = useMemo(() => {
    let m = 0;
    for (const s of series) for (const p of s.points) if (p.v != null && p.v > m) m = p.v;
    return niceMax(m, percent);
  }, [series, percent]);
  const hasData = series.some((s) => s.points.some((p) => p.v != null));
  if (!hasData || len < 2) {
    return <div className={cn("grid h-[200px] place-items-center text-[12.5px]", VX_FAINT)}>{empty}</div>;
  }
  const x = (i: number) => PAD_L + (i / (len - 1)) * (W - PAD_L - PAD_R);
  const y = (v: number) => PAD_T + (1 - v / max) * (H - PAD_T - PAD_B);
  const fmt = (v: number) => (percent ? `${Math.round(v)}%` : formatMB(v));
  const ticks = [0, 0.25, 0.5, 0.75, 1].map((f) => max * f);
  const times = series[0].points;
  const labelIdx = [0, Math.floor((len - 1) / 2), len - 1];

  return (
    <div className="relative">
      <svg
        viewBox={`0 0 ${W} ${H}`}
        className="h-[200px] w-full"
        preserveAspectRatio="none"
        onMouseLeave={() => setHover(null)}
        onMouseMove={(e) => {
          const rect = e.currentTarget.getBoundingClientRect();
          const rel = ((e.clientX - rect.left) / rect.width) * W;
          const i = Math.round(((rel - PAD_L) / (W - PAD_L - PAD_R)) * (len - 1));
          setHover(i >= 0 && i < len ? i : null);
        }}
      >
        {ticks.map((v) => (
          <g key={v}>
            <line x1={PAD_L} x2={W - PAD_R} y1={y(v)} y2={y(v)} stroke="var(--vx-border)" strokeWidth={1} vectorEffect="non-scaling-stroke" />
            <text x={PAD_L - 6} y={y(v) + 3} textAnchor="end" fontSize={9.5} fill="var(--vx-faint)" fontFamily="var(--font-mono, monospace)">
              {fmt(v)}
            </text>
          </g>
        ))}
        {series.map((s) =>
          segments(s.points, x, y).map((d, i) => (
            <path key={`${s.key}-${i}`} d={d} fill="none" stroke={s.color} strokeWidth={1.6} vectorEffect="non-scaling-stroke" strokeLinejoin="round" />
          ))
        )}
        {hover != null && (
          <line x1={x(hover)} x2={x(hover)} y1={PAD_T} y2={H - PAD_B} stroke="var(--vx-border-strong)" strokeWidth={1} vectorEffect="non-scaling-stroke" />
        )}
        {labelIdx.map((i) => (
          <text
            key={i}
            x={x(i)}
            y={H - 6}
            textAnchor={i === 0 ? "start" : i === len - 1 ? "end" : "middle"}
            fontSize={9.5}
            fill="var(--vx-faint)"
          >
            {formatDateTime(times[i]?.t).slice(0, -3)}
          </text>
        ))}
      </svg>
      {hover != null && (
        <div className="pointer-events-none absolute top-1 right-2 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-elevated)] px-2.5 py-1.5 text-[11px] shadow-sm">
          <div className={VX_FAINT}>{formatDateTime(times[hover]?.t)}</div>
          {series.map((s) => {
            const v = s.points[hover]?.v;
            return (
              <div key={s.key} className="flex items-center gap-2 font-mono">
                <span className="h-1.5 w-1.5 rounded-full" style={{ background: s.color }} />
                {s.label}: {v == null ? "—" : fmt(v)}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export function AgentMetricChart({ id }: { id: string }) {
  const t = useT();
  const [range, setRange] = useState<AgentMetricRange>("24h");
  const [scope, setScope] = useState<"node" | "agent">("node");
  const metricsQuery = useQuery({
    queryKey: queryKeys.agentMetrics(id, range),
    queryFn: () => fetchAgentMetrics(id, range),
    refetchInterval: range === "1h" ? 30000 : 120000,
  });
  const s = metricsQuery.data?.series ?? {};
  const nodeSeries: Series[] = [
    { key: "cpu", label: "CPU", color: "var(--vx-info)", points: s.cpu_usage ?? [] },
    { key: "ram", label: "RAM", color: "var(--vx-violet)", points: s.ram_usage ?? [] },
    { key: "disk", label: t("admin.agents.disk_short"), color: "var(--vx-warn)", points: s.disk_usage ?? [] },
  ];
  const agentSeries: Series[] = [
    { key: "acpu", label: "CPU", color: "var(--vx-info)", points: s.agent_cpu_usage ?? [] },
  ];
  const agentMem: Series[] = [
    { key: "amem", label: "RAM", color: "var(--vx-violet)", points: s.agent_ram_mb ?? [] },
  ];

  return (
    <Panel
      title={t("admin.agents.chart.title")}
      aside={
        <div className="flex flex-wrap items-center gap-2">
          <SubTabs
            items={[
              { id: "node", title: t("admin.agents.chart.node") },
              { id: "agent", title: t("admin.agents.chart.agent") },
            ]}
            active={scope}
            onSelect={(v) => setScope(v as "node" | "agent")}
          />
          <SubTabs
            items={[
              { id: "1h", title: t("admin.agents.chart.1h") },
              { id: "24h", title: t("admin.agents.chart.24h") },
              { id: "7d", title: t("admin.agents.chart.7d") },
            ]}
            active={range}
            onSelect={(v) => setRange(v as AgentMetricRange)}
          />
        </div>
      }
    >
      {metricsQuery.isLoading ? (
        <Skeleton className="h-[200px] w-full rounded-[10px]" />
      ) : scope === "node" ? (
        <Chart series={nodeSeries} percent empty={t("admin.agents.chart.empty")} />
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          <Chart series={agentSeries} percent empty={t("admin.agents.chart.empty")} />
          <Chart series={agentMem} percent={false} empty={t("admin.agents.chart.empty")} />
        </div>
      )}
      <div className={cn("mt-3 flex flex-wrap gap-4 text-[11.5px]", VX_FAINT)}>
        {(scope === "node" ? nodeSeries : [...agentSeries, ...agentMem]).map((ser) => (
          <span key={ser.key} className="inline-flex items-center gap-1.5">
            <span className="h-1.5 w-3 rounded-full" style={{ background: ser.color }} />
            {ser.label}
          </span>
        ))}
      </div>
    </Panel>
  );
}
