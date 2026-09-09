"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { Skeleton } from "@/components/ui/skeleton";
import { fetchAdminDashboardSeries, type AdminSeriesPoint } from "@/lib/api";
import type { TranslateFn } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

function buildRanges(t: TranslateFn) {
  return [
    { hours: 24, label: t("admin.load_chart.range_24h") },
    { hours: 72, label: t("admin.load_chart.range_3d") },
    { hours: 168, label: t("admin.load_chart.range_7d") },
  ];
}

function polylines(points: AdminSeriesPoint[], key: "cpu" | "ram"): string[] {
  const step = 100 / Math.max(points.length - 1, 1);
  const segments: string[] = [];
  let current: string[] = [];
  points.forEach((p, i) => {
    const v = p[key];
    if (v === null || v === undefined) {
      if (current.length > 1) segments.push(current.join(" "));
      current = [];
      return;
    }
    const x = i * step;
    const y = 100 - Math.min(100, Math.max(0, v));
    current.push(`${current.length === 0 ? "M" : "L"}${x.toFixed(2)},${y.toFixed(2)}`);
  });
  if (current.length > 1) segments.push(current.join(" "));
  return segments;
}

function lastValue(points: AdminSeriesPoint[], key: "cpu" | "ram"): number | null {
  for (let i = points.length - 1; i >= 0; i--) {
    const v = points[i][key];
    if (v !== null && v !== undefined) return v;
  }
  return null;
}

export function AdminLoadChart({ className }: { className?: string }) {
  const t = useT();
  const [hours, setHours] = useState(24);
  const [nodeId, setNodeId] = useState("");
  const ranges = buildRanges(t);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-dashboard-series", hours, nodeId],
    queryFn: () => fetchAdminDashboardSeries({ hours, nodeId: nodeId || undefined }),
    refetchInterval: 60_000,
  });

  const points = data?.points ?? [];
  const nodes = data?.nodes ?? [];
  const cpuNow = lastValue(points, "cpu");
  const ramNow = lastValue(points, "ram");

  return (
    <div className={className}>
      <div className="flex flex-wrap items-center gap-3">
        <span className="text-base font-semibold">
          {t("admin.load_chart.title")}
        </span>
        <div className="flex-1" />
        {nodes.length > 1 && (
          <select
            value={nodeId}
            onChange={(e) => setNodeId(e.target.value)}
            className="h-8 rounded-[8px] border border-[var(--vx-border-2)] bg-transparent px-2 text-[13px]"
          >
            <option value="">{t("admin.load_chart.all_locations")}</option>
            {nodes.map((n) => (
              <option key={n.id} value={n.id}>
                {n.name}
              </option>
            ))}
          </select>
        )}
        <div className="flex gap-1">
          {ranges.map((r) => (
            <button
              key={r.hours}
              type="button"
              onClick={() => setHours(r.hours)}
              className={cn(
                "h-8 rounded-[8px] px-2.5 text-[13px] font-medium transition-colors",
                hours === r.hours
                  ? "bg-[var(--vx-fg-strong)] text-[var(--vx-on-fill)]"
                  : "text-[var(--vx-muted)] hover:bg-[var(--vx-tint)]"
              )}
            >
              {r.label}
            </button>
          ))}
        </div>
      </div>

      <div className="mt-3 flex flex-wrap gap-5 text-[13px]">
        <span className="text-muted-foreground">
          CPU:{" "}
          <b className="text-foreground">
            {cpuNow === null ? "—" : `${cpuNow.toFixed(1)}%`}
          </b>
        </span>
        <span className="text-muted-foreground">
          RAM:{" "}
          <b className="text-foreground">
            {ramNow === null ? "—" : `${ramNow.toFixed(1)}%`}
          </b>
        </span>
      </div>

      {isLoading ? (
        <Skeleton className="mt-4 h-[160px] w-full rounded-[14px]" />
      ) : points.length === 0 ? (
        <div className="py-14 text-center">
          <div className="text-sm text-muted-foreground">
            {t("admin.load_chart.empty")}
          </div>
          <div className="mt-[7px] text-[13px] text-[var(--vx-ink-ghost)]">
            {t("admin.load_chart.empty_hint")}
          </div>
        </div>
      ) : (
        <svg
          viewBox="0 0 100 100"
          preserveAspectRatio="none"
          className="mt-4 h-[160px] w-full"
          aria-label={t("admin.load_chart.aria")}
        >
          {[25, 50, 75].map((y) => (
            <line
              key={y}
              x1="0"
              x2="100"
              y1={y}
              y2={y}
              stroke="currentColor"
              strokeWidth="0.3"
              className="text-[var(--vx-panel-line)]"
              vectorEffect="non-scaling-stroke"
            />
          ))}
          {polylines(points, "ram").map((d, i) => (
            <path
              key={`ram-${i}`}
              d={d}
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              className="text-[var(--vx-ink-ghost)]"
              vectorEffect="non-scaling-stroke"
            />
          ))}
          {polylines(points, "cpu").map((d, i) => (
            <path
              key={`cpu-${i}`}
              d={d}
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              className="text-[var(--vx-fg-strong)]"
              vectorEffect="non-scaling-stroke"
            />
          ))}
        </svg>
      )}
      <div className="mt-2 flex gap-4 text-[11px] text-[var(--vx-ink-ghost)]">
        <span className="flex items-center gap-1.5">
          <span className="h-[2px] w-4 bg-[var(--vx-fg-strong)]" /> CPU
        </span>
        <span className="flex items-center gap-1.5">
          <span className="h-[2px] w-4 bg-[var(--vx-ink-ghost)]" /> RAM
        </span>
      </div>
    </div>
  );
}
