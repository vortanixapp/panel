"use client";

import { useMemo } from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

export type MetricPoint = {
  t?: string;
  measured_at?: string;
  v?: number;
  value?: number;
};

type ChartRow = {
  label: string;
  value: number;
};

type LocationMetricsChartProps = {
  title?: string;
  points: MetricPoint[];
  color?: string;
};

type TooltipContentProps = {
  active?: boolean;
  payload?: Array<{ value?: number }>;
  label?: string;
};

function ChartTooltip({ active, payload, label }: TooltipContentProps) {
  if (!active || !payload?.length) return null;

  const value = payload[0]?.value ?? 0;

  return (
    <div className="rounded-lg border border-border bg-popover px-3 py-2 text-sm shadow-md">
      <p className="font-medium text-popover-foreground">{label}</p>
      <p className="text-muted-foreground">{value.toFixed(1)}%</p>
    </div>
  );
}

function formatTimestamp(raw: string): string {
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return raw;
  return d.toLocaleString(localeTag(), {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function toChartRows(points: MetricPoint[]): ChartRow[] {
  return points.map((p) => {
    const raw = p.t ?? p.measured_at ?? "";
    const value = Number(p.v ?? p.value ?? 0);
    return {
      label: raw ? formatTimestamp(raw) : "—",
      value: Number.isFinite(value) ? value : 0,
    };
  });
}

export function LocationMetricsChart({
  title,
  points,
  color = "var(--chart-1)",
}: LocationMetricsChartProps) {
  const t = useT();
  const data = useMemo(() => toChartRows(points), [points]);

  return (
    <div className="space-y-3">
      {title ? <p className="text-base font-semibold">{title}</p> : null}
      {data.length > 0 ? (
        <ResponsiveContainer width="100%" height={220}>
          <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
            <CartesianGrid
              vertical={false}
              stroke="var(--border)"
              strokeDasharray="4 4"
            />
            <XAxis
              dataKey="label"
              tickLine={false}
              axisLine={false}
              minTickGap={32}
              tick={{ fill: "var(--muted-foreground)", fontSize: 11 }}
            />
            <YAxis
              domain={[0, 100]}
              tickLine={false}
              axisLine={false}
              width={36}
              tickFormatter={(v) => `${v}%`}
              tick={{ fill: "var(--muted-foreground)", fontSize: 11 }}
            />
            <Tooltip content={<ChartTooltip />} />
            <Line
              type="monotone"
              dataKey="value"
              stroke={color}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
            />
          </LineChart>
        </ResponsiveContainer>
      ) : (
        <p className="py-8 text-center text-sm text-muted-foreground">
          {t("admin.metrics.no_data")}
        </p>
      )}
    </div>
  );
}
