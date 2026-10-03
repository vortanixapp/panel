"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useT } from "@/hooks/use-translations";
import { fetchAdminPerformance, type AdminPerfChart } from "@/lib/api";
import { localeTag, type TranslateFn } from "@/lib/i18n";
import { pollMs } from "@/lib/public-settings";

const RANGES = ["1h", "6h", "24h", "7d"] as const;

const SECTIONS: { id: string; charts: string[] }[] = [
  { id: "api", charts: ["api_rps", "api_routes_rps", "api_p95", "api_routes_p95", "api_5xx"] },
  { id: "db", charts: ["db_p95", "db_slow", "db_pool", "db_pool_wait"] },
  { id: "runtime", charts: ["goroutines", "memory"] },
  { id: "relay", charts: ["relay_rps", "relay_p95"] },
];

const COLORS = [
  "#6366f1",
  "#22c55e",
  "#f59e0b",
  "#ef4444",
  "#06b6d4",
  "#a855f7",
  "#84cc16",
  "#ec4899",
];

function formatValue(value: number, unit: string, t: TranslateFn): string {
  if (!Number.isFinite(value)) return "—";
  switch (unit) {
    case "s":
      return value < 1
        ? `${(value * 1000).toFixed(value < 0.01 ? 1 : 0)} ${t("admin.perf.unit.ms")}`
        : `${value.toFixed(2)} ${t("admin.perf.unit.s")}`;
    case "reqps":
      return `${value.toFixed(value < 10 ? 2 : 1)}/${t("admin.perf.unit.s")}`;
    case "perMin":
      return `${value.toFixed(value < 10 ? 2 : 1)}/${t("admin.perf.unit.min")}`;
    case "bytes": {
      const mb = value / (1024 * 1024);
      return mb >= 1024
        ? `${(mb / 1024).toFixed(2)} ${t("admin.perf.unit.gb")}`
        : `${mb.toFixed(0)} ${t("admin.perf.unit.mb")}`;
    }
    default:
      return value >= 100 ? value.toFixed(0) : value.toFixed(2);
  }
}

function chartRows(chart: AdminPerfChart) {
  const rows = new Map<number, Record<string, number>>();
  chart.series.forEach((series, index) => {
    for (const [ts, value] of series.points) {
      const row = rows.get(ts) ?? { ts };
      row[`s${index}`] = value;
      rows.set(ts, row);
    }
  });
  return [...rows.values()].sort((a, b) => a.ts - b.ts);
}

function PerfChart({ chart, span }: { chart: AdminPerfChart; span: string }) {
  const t = useT();
  const tag = localeTag();
  const rows = useMemo(() => chartRows(chart), [chart]);
  const long = span === "24h" || span === "7d";
  const tick = (ts: number) =>
    new Date(ts * 1000).toLocaleString(tag, long ? { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" } : { hour: "2-digit", minute: "2-digit" });

  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="mb-3 text-sm font-medium">{t(`admin.perf.chart.${chart.id}`)}</div>
      {rows.length === 0 ? (
        <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">
          {t("admin.perf.no_data")}
        </div>
      ) : (
        <div className="h-48 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={rows} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
              <XAxis
                dataKey="ts"
                type="number"
                domain={["dataMin", "dataMax"]}
                tickFormatter={tick}
                tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
                axisLine={false}
                tickLine={false}
                minTickGap={40}
              />
              <YAxis
                tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
                axisLine={false}
                tickLine={false}
                width={64}
                tickFormatter={(v: number) => formatValue(v, chart.unit, t)}
              />
              <Tooltip
                labelFormatter={(ts) => tick(Number(ts))}
                formatter={(value, name) => [formatValue(Number(value), chart.unit, t), name]}
                contentStyle={{ fontSize: 12 }}
              />
              {chart.series.length > 1 && <Legend wrapperStyle={{ fontSize: 11 }} />}
              {chart.series.map((series, index) => (
                <Line
                  key={`${series.name}-${index}`}
                  type="monotone"
                  dataKey={`s${index}`}
                  name={series.name || t(`admin.perf.chart.${chart.id}`)}
                  stroke={COLORS[index % COLORS.length]}
                  strokeWidth={1.75}
                  dot={false}
                  isAnimationActive={false}
                  connectNulls
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}

export function PerformancePageContent() {
  const t = useT();
  const [range, setRange] = useState<(typeof RANGES)[number]>("1h");
  const query = useQuery({
    queryKey: ["admin-performance", range],
    queryFn: () => fetchAdminPerformance(range),
    refetchInterval: pollMs(30_000),
    retry: false,
  });

  const byId = useMemo(() => {
    const map = new Map<string, AdminPerfChart>();
    for (const chart of query.data?.charts ?? []) map.set(chart.id, chart);
    return map;
  }, [query.data]);

  return (
    <PageShell variant="admin">
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <h1 className="text-2xl font-semibold tracking-tight">{t("admin.perf.title")}</h1>
        <div className="ms-auto flex gap-1">
          {RANGES.map((id) => (
            <Button
              key={id}
              size="sm"
              variant={range === id ? "default" : "outline"}
              onClick={() => setRange(id)}
            >
              {t(`admin.perf.range.${id}`)}
            </Button>
          ))}
        </div>
      </div>

      {query.isLoading ? (
        <Skeleton className="h-96 w-full" />
      ) : !query.data?.available ? (
        <div className="rounded-lg border bg-card p-8 text-center">
          <div className="text-base font-medium">{t("admin.perf.unavailable_title")}</div>
          <p className="mx-auto mt-2 max-w-xl text-sm text-muted-foreground">
            {t("admin.perf.unavailable_body")}
          </p>
          <code className="mx-auto mt-4 block max-w-full overflow-x-auto rounded-md bg-muted px-3 py-2 text-xs">
            {t("admin.perf.unavailable_command")}
          </code>
        </div>
      ) : (
        <div className="space-y-8">
          {SECTIONS.map((section) => (
            <section key={section.id}>
              <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
                {t(`admin.perf.section.${section.id}`)}
              </h2>
              <div className="grid gap-4 lg:grid-cols-2">
                {section.charts.map((id) => {
                  const chart = byId.get(id);
                  return chart ? <PerfChart key={id} chart={chart} span={range} /> : null;
                })}
              </div>
            </section>
          ))}
        </div>
      )}
    </PageShell>
  );
}
