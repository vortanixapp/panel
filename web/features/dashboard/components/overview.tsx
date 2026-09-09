"use client";

import { useMemo } from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { Server } from "@/lib/api";

type OverviewProps = {
  servers: Server[];
};

type ChartRow = {
  name: string;
  total: number;
  fill: string;
};

const STATUS_COLORS = {
  running: "var(--chart-2)",
  stopped: "var(--chart-3)",
  transitional: "var(--chart-4)",
} as const;

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
      <p className="text-muted-foreground">
        {value}{" "}
        {value === 1 ? "сервер" : value < 5 ? "сервера" : "серверов"}
      </p>
    </div>
  );
}

export function Overview({ servers }: OverviewProps) {
  const data = useMemo<ChartRow[]>(() => {
    const counts = {
      running: 0,
      stopped: 0,
      transitional: 0,
    };
    for (const server of servers) {
      if (server.status === "running") counts.running += 1;
      else if (server.status === "stopped") counts.stopped += 1;
      else counts.transitional += 1;
    }
    return [
      {
        name: "Запущены",
        total: counts.running,
        fill: STATUS_COLORS.running,
      },
      {
        name: "Остановлены",
        total: counts.stopped,
        fill: STATUS_COLORS.stopped,
      },
      {
        name: "Переход",
        total: counts.transitional,
        fill: STATUS_COLORS.transitional,
      },
    ];
  }, [servers]);

  const yMax = useMemo(
    () => Math.max(1, ...data.map((d) => d.total)),
    [data]
  );

  if (servers.length === 0) {
    return (
      <p className="py-16 text-center text-sm text-muted-foreground">
        Нет серверов для отображения статистики
      </p>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={320}>
      <BarChart
        data={data}
        margin={{ top: 8, right: 8, left: -8, bottom: 0 }}
        barCategoryGap="28%"
      >
        <CartesianGrid
          vertical={false}
          stroke="var(--border)"
          strokeDasharray="4 4"
        />
        <XAxis
          dataKey="name"
          tickLine={false}
          axisLine={false}
          tickMargin={10}
          tick={{ fill: "var(--muted-foreground)", fontSize: 12 }}
        />
        <YAxis
          allowDecimals={false}
          domain={[0, yMax]}
          tickLine={false}
          axisLine={false}
          tickMargin={8}
          width={32}
          tick={{ fill: "var(--muted-foreground)", fontSize: 12 }}
        />
        <Tooltip
          cursor={{ fill: "var(--muted)", opacity: 0.35 }}
          content={<ChartTooltip />}
        />
        <Bar dataKey="total" radius={[6, 6, 0, 0]} maxBarSize={56}>
          {data.map((entry) => (
            <Cell key={entry.name} fill={entry.fill} />
          ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  );
}
