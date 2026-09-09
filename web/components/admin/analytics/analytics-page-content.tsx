"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Area,
  AreaChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  downloadAdminAnalyticsCsv,
  fetchAdminAnalytics,
  type AnalyticsBreakdownRow,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

const PERIODS = [
  { value: "7", labelKey: "admin.analytics.period.7" },
  { value: "30", labelKey: "admin.analytics.period.30" },
  { value: "90", labelKey: "admin.analytics.period.90" },
  { value: "365", labelKey: "admin.analytics.period.365" },
];

const SERIES = {
  charges: "var(--vx-series-1)",
  topups: "var(--vx-series-2)",
};

function money(value: number): string {
  return new Intl.NumberFormat(localeTag(), { maximumFractionDigits: 0 }).format(
    Math.round(value)
  );
}

function fmtDay(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime())
    ? iso
    : d.toLocaleDateString(localeTag(), { day: "numeric", month: "short" });
}

function Stat({
  label,
  value,
  hint,
  change,
}: {
  label: string;
  value: string;
  hint?: string;
  change?: number;
}) {
  const arrow = change === undefined ? null : change >= 0 ? "▲" : "▼";
  const tone =
    change === undefined
      ? ""
      : change >= 0
        ? "text-emerald-600 dark:text-emerald-500"
        : "text-destructive";
  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-2xl font-semibold">{value}</div>
      <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
        {change !== undefined && (
          <span className={tone}>
            {arrow} {Math.abs(Math.round(change))}%
          </span>
        )}
        {hint}
      </div>
    </div>
  );
}

function BreakdownTable({
  title,
  rows,
  emptyText,
}: {
  title: string;
  rows: AnalyticsBreakdownRow[];
  emptyText: string;
}) {
  const t = useT();
  return (
    <div className="rounded-lg border bg-card">
      <div className="border-b px-4 py-3 text-sm font-medium">{title}</div>
      {rows.length === 0 ? (
        <div className="p-6 text-center text-sm text-muted-foreground">
          {emptyText}
        </div>
      ) : (
        <table className="w-full text-sm">
          <thead>
            <tr className="text-xs text-muted-foreground">
              <th className="px-4 py-2 text-left font-normal">
                {t("common.name")}
              </th>
              <th className="px-4 py-2 text-right font-normal">
                {t("admin.analytics.col_servers")}
              </th>
              <th className="px-4 py-2 text-right font-normal">
                {t("admin.analytics.col_per_month")}
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.label} className="border-t">
                <td className="px-4 py-2">{row.label}</td>
                <td className="px-4 py-2 text-right font-mono">{row.servers}</td>
                <td className="px-4 py-2 text-right font-mono">
                  {money(row.monthly)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

export function AnalyticsPageContent() {
  const t = useT();
  const [days, setDays] = useState("30");
  const { data, isLoading } = useQuery({
    queryKey: ["admin-analytics", days],
    queryFn: () => fetchAdminAnalytics(Number(days)),
  });

  const chartData = (data?.series ?? []).map((p) => ({
    ...p,
    label: fmtDay(p.date),
  }));

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("admin.analytics.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.analytics.subtitle")}
          </p>
        </div>
        <div className="flex items-end gap-2">
          <div className="space-y-1">
            <Label>{t("common.period")}</Label>
            <Select value={days} onValueChange={setDays}>
              <SelectTrigger className="w-36">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PERIODS.map((p) => (
                  <SelectItem key={p.value} value={p.value}>
                    {t(p.labelKey)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <Button
            variant="outline"
            onClick={() =>
              downloadAdminAnalyticsCsv(Number(days)).catch((e: Error) =>
                toast.error(e.message)
              )
            }
          >
            {t("admin.analytics.export_csv")}
          </Button>
        </div>
      </div>

      {isLoading || !data ? (
        <div className="space-y-4">
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-72 w-full" />
        </div>
      ) : (
        <>
          <div className="mb-6 grid grid-cols-2 gap-3 md:grid-cols-5">
            <Stat
              label={t("admin.analytics.stat_earned")}
              value={`${money(data.revenue.charges)} ₽`}
              change={data.revenue.charges_change_percent}
              hint={t("admin.analytics.vs_previous")}
            />
            <Stat
              label={t("admin.analytics.stat_topups")}
              value={`${money(data.revenue.topups)} ₽`}
              change={data.revenue.topups_change_percent}
              hint={t("admin.analytics.payers", {
                count: data.revenue.payers,
              })}
            />
            <Stat
              label={t("admin.analytics.stat_expected")}
              value={`${money(data.recurring.monthly)} ₽`}
              hint={t("admin.analytics.active_servers", {
                count: data.recurring.active_servers,
              })}
            />
            <Stat
              label={t("admin.analytics.stat_arpu")}
              value={`${money(data.revenue.arpu)} ₽`}
              hint={t("admin.analytics.arpu_hint")}
            />
            <Stat
              label={t("admin.analytics.stat_expiring")}
              value={String(data.recurring.expiring_7d)}
              hint={t("admin.analytics.expiring_hint")}
            />
          </div>

          <div className="mb-6 rounded-lg border bg-card p-4">
            <div className="mb-4 text-sm font-medium">
              {t("admin.analytics.chart_title")}
            </div>
            <div className="h-72 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart
                  data={chartData}
                  margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
                >
                  <defs>
                    <linearGradient id="fillCharges" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={SERIES.charges} stopOpacity={0.28} />
                      <stop offset="100%" stopColor={SERIES.charges} stopOpacity={0.02} />
                    </linearGradient>
                    <linearGradient id="fillTopups" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={SERIES.topups} stopOpacity={0.28} />
                      <stop offset="100%" stopColor={SERIES.topups} stopOpacity={0.02} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid
                    strokeDasharray="3 3"
                    stroke="var(--border)"
                    vertical={false}
                  />
                  <XAxis
                    dataKey="label"
                    tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
                    axisLine={false}
                    tickLine={false}
                    minTickGap={24}
                  />
                  <YAxis
                    tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
                    axisLine={false}
                    tickLine={false}
                    width={64}
                    tickFormatter={(v: number) => money(v)}
                  />
                  <Tooltip
                    contentStyle={{
                      background: "var(--card)",
                      border: "1px solid var(--border)",
                      borderRadius: 8,
                      fontSize: 12,
                      color: "var(--foreground)",
                    }}
                    formatter={(value, name) => [
                      `${money(Number(value) || 0)} ₽`,
                      String(name),
                    ]}
                  />
                  <Legend
                    wrapperStyle={{ fontSize: 12, color: "var(--muted-foreground)" }}
                  />
                  <Area
                    type="monotone"
                    dataKey="charges"
                    name={t("admin.analytics.series_charges")}
                    stroke={SERIES.charges}
                    strokeWidth={2}
                    fill="url(#fillCharges)"
                    dot={false}
                    activeDot={{ r: 4 }}
                  />
                  <Area
                    type="monotone"
                    dataKey="topups"
                    name={t("admin.analytics.series_topups")}
                    stroke={SERIES.topups}
                    strokeWidth={2}
                    fill="url(#fillTopups)"
                    dot={false}
                    activeDot={{ r: 4 }}
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>

          <div className="mb-6 grid gap-4 lg:grid-cols-2">
            <div className="rounded-lg border bg-card">
              <div className="border-b px-4 py-3 text-sm font-medium">
                {t("admin.analytics.sources_title")}
              </div>
              {data.sources.length === 0 ? (
                <div className="p-6 text-center text-sm text-muted-foreground">
                  {t("admin.analytics.sources_empty")}
                </div>
              ) : (
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-xs text-muted-foreground">
                      <th className="px-4 py-2 text-left font-normal">
                        {t("admin.analytics.col_operation")}
                      </th>
                      <th className="px-4 py-2 text-right font-normal">
                        {t("admin.analytics.col_count")}
                      </th>
                      <th className="px-4 py-2 text-right font-normal">
                        {t("admin.analytics.col_amount")}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.sources.map((s) => (
                      <tr key={s.source} className="border-t">
                        <td className="px-4 py-2">{s.label}</td>
                        <td className="px-4 py-2 text-right font-mono">{s.count}</td>
                        <td className="px-4 py-2 text-right font-mono">
                          {money(s.amount)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>

            <div className="rounded-lg border bg-card p-4">
              <div className="mb-3 text-sm font-medium">
                {t("admin.analytics.funnel_title")}
              </div>
              <div className="space-y-3">
                <FunnelRow
                  label={t("admin.analytics.funnel_registered")}
                  value={data.customers.registered}
                  total={data.customers.registered}
                />
                <FunnelRow
                  label={t("admin.analytics.funnel_server")}
                  value={data.customers.with_server}
                  total={data.customers.registered}
                  percent={data.customers.server_conversion}
                />
                <FunnelRow
                  label={t("admin.analytics.funnel_paid")}
                  value={data.customers.with_payment}
                  total={data.customers.registered}
                  percent={data.customers.payment_conversion}
                />
              </div>
              <div className="mt-4 border-t pt-3 text-xs text-muted-foreground">
                {t("admin.analytics.churn", {
                  total: data.customers.total,
                  expired: data.churn.expired,
                  renewed: data.churn.renewed,
                  suspended: data.churn.suspended,
                })}
              </div>
            </div>
          </div>

          <div className="grid gap-4 lg:grid-cols-3">
            <BreakdownTable
              title={t("admin.analytics.by_games")}
              rows={data.breakdown.games}
              emptyText={t("admin.analytics.no_active_servers")}
            />
            <BreakdownTable
              title={t("admin.analytics.by_locations")}
              rows={data.breakdown.locations}
              emptyText={t("admin.analytics.no_active_servers")}
            />
            <BreakdownTable
              title={t("admin.analytics.by_tariffs")}
              rows={data.breakdown.tariffs}
              emptyText={t("admin.analytics.no_active_servers")}
            />
          </div>

          <p className="mt-4 text-xs text-muted-foreground">
            {t("admin.analytics.breakdown_hint")}
          </p>
        </>
      )}
    </PageShell>
  );
}

function FunnelRow({
  label,
  value,
  total,
  percent,
}: {
  label: string;
  value: number;
  total: number;
  percent?: number;
}) {
  const width = total > 0 ? Math.max((value / total) * 100, 2) : 0;
  return (
    <div>
      <div className="flex items-baseline justify-between text-sm">
        <span>{label}</span>
        <span className="font-mono">
          {value}
          {percent !== undefined ? (
            <span className="ml-2 text-xs text-muted-foreground">
              {Math.round(percent)}%
            </span>
          ) : null}
        </span>
      </div>
      <div className="mt-1 h-2 rounded-full bg-muted">
        <div
          className="h-2 rounded-full"
          style={{ width: `${width}%`, background: "var(--vx-series-2)" }}
        />
      </div>
    </div>
  );
}
