"use client";

import type { MetricPoint } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

export function MetricsChart({ points }: { points: MetricPoint[] }) {
  const t = useT();
  if (points.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        {t("servers.metrics.chart_empty")}
      </p>
    );
  }

  const maxCpu = Math.max(10, ...points.map((p) => p.cpu_pct ?? 0));
  const w = 100 / Math.max(points.length - 1, 1);

  const cpuPath = points
    .map((p, i) => {
      const x = i * w;
      const y = 100 - ((p.cpu_pct ?? 0) / maxCpu) * 100;
      return `${i === 0 ? "M" : "L"}${x},${y}`;
    })
    .join(" ");

  const last = points[points.length - 1];
  const memUsed = last.mem_used_mb ?? 0;
  const memLimit = last.mem_limit_mb ?? 0;
  const memDisplay =
    memUsed === 0 && memLimit > 0
      ? "< 1"
      : String(memUsed);

  return (
    <div>
      <div className="mb-3 flex flex-wrap gap-6 text-sm">
        <span>
          CPU: <strong className="text-foreground">{(last.cpu_pct ?? 0).toFixed(1)}%</strong>
        </span>
        <span>
          RAM:{" "}
          <strong className="text-foreground">
            {memDisplay}
          </strong>
          {" / "}
          {memLimit} MB
        </span>
      </div>
      <svg
        viewBox="0 0 100 100"
        preserveAspectRatio="none"
        className="h-28 w-full rounded-md bg-muted/20"
      >
        <path
          d={cpuPath}
          fill="none"
          stroke="hsl(var(--primary))"
          strokeWidth="1.5"
          vectorEffect="non-scaling-stroke"
        />
      </svg>
    </div>
  );
}
