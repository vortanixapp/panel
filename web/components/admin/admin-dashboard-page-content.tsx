"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { PageShell } from "@/components/layout/page-shell";
import { AdminLoadChart } from "@/components/admin/admin-load-chart";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchAdminBilling,
  fetchAdminDashboard,
  fetchAdminDashboardNodes,
  fetchAdminLogsFiltered,
  fetchAdminReadiness,
  fetchAdminSupport,
  fetchServers,
  fetchUsers,
  type AdminDashboardNode,
  type AdminPayment,
  type PanelUser,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { isStaffRole, roleLabel } from "@/lib/rbac";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { localeTag, type TranslateFn } from "@/lib/i18n";

const CARD = "rounded-[20px] border border-[var(--vx-panel-line)] bg-[var(--vx-panel-card)]";
const CARD_INNER = "rounded-[14px] border border-[var(--vx-tint)] bg-[var(--vx-panel-card-2)]";
const MONO_LABEL =
  "font-mono text-[11px] tracking-[0.08em] uppercase text-[var(--vx-ink-ghost)]";

const MS_HOUR = 3_600_000;
const MS_DAY = 86_400_000;
const PAID_STATUSES = new Set(["success", "succeeded", "completed", "paid"]);

type Range = "24ч" | "7д" | "30д";

// Подписи храним ключами: список читается на уровне модуля, и готовый текст
// застыл бы на языке, который стоял в момент загрузки страницы.
const RANGES: { id: Range; labelKey: string }[] = [
  { id: "24ч", labelKey: "admin.dashboard.range_24h" },
  { id: "7д", labelKey: "admin.dashboard.range_7d" },
  { id: "30д", labelKey: "admin.dashboard.range_30d" },
];

function nf(value: number) {
  return Math.round(value).toLocaleString(localeTag());
}

function money(value: number, t: TranslateFn) {
  if (!Number.isFinite(value)) return "0";
  if (Math.abs(value) >= 1_000_000)
    return `${(value / 1_000_000).toFixed(2)} ${t("admin.dashboard.unit_million")}`;
  return nf(value);
}

function bytesLabel(value: number, t: TranslateFn) {
  const gb = t("admin.infra.unit_gb");
  if (!Number.isFinite(value) || value <= 0) return `0 ${gb}`;
  const tb = value / 1024 ** 4;
  if (tb >= 1) return `${tb.toFixed(1)} ${t("admin.dashboard.unit_tb")}`;
  return `${Math.round(value / 1024 ** 3)} ${gb}`;
}

function parseBytes(value: string | undefined) {
  if (!value) return 0;
  const num = Number(String(value).trim());
  return Number.isFinite(num) ? num : 0;
}

function pct(part: number, total: number) {
  if (total <= 0) return 0;
  return Math.max(0, Math.min(100, (part / total) * 100));
}

function plural(n: number, one: string, few: string, many: string) {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return one;
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return few;
  return many;
}

// Формы числительных лежат в словаре тремя ключами <base>_one/_few/_many:
// в русском они разные, в английском единственное и множественное совпадают
// в двух последних.
function countWord(t: TranslateFn, base: string, n: number) {
  return plural(n, t(`${base}_one`), t(`${base}_few`), t(`${base}_many`));
}

function clockLabel(d: Date) {
  return d.toLocaleTimeString(localeTag(), { hour: "2-digit", minute: "2-digit" });
}

function eventTime(iso: string) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  if (Date.now() - d.getTime() < MS_DAY) return clockLabel(d);
  return d.toLocaleDateString(localeTag(), { day: "numeric", month: "short" });
}

function durationLabel(ms: number, t: TranslateFn) {
  if (!Number.isFinite(ms) || ms < 0) return "—";
  const min = t("admin.dashboard.unit_minutes");
  const minutes = Math.floor(ms / 60_000);
  if (minutes < 60) return `${minutes} ${min}`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24)
    return `${hours} ${t("admin.dashboard.unit_hours")} ${String(minutes % 60).padStart(2, "0")} ${min}`;
  const days = Math.floor(hours / 24);
  return `${days} ${countWord(t, "admin.dashboard.count.day", days)}`;
}

function userDisplayName(user: PanelUser) {
  if (user.name?.trim()) return user.name.trim();
  const local = user.email.split("@")[0];
  return local ? local.charAt(0).toUpperCase() + local.slice(1) : user.email;
}

function ArrowIcon({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.8}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
    >
      <path d="M5 12h13M13 6.5 18.5 12 13 17.5" />
    </svg>
  );
}

function revenueSeries(payments: AdminPayment[], range: Range) {
  const step = range === "24ч" ? MS_HOUR : MS_DAY;
  const count = range === "24ч" ? 24 : range === "7д" ? 7 : 30;

  const anchor = new Date();
  if (range === "24ч") anchor.setMinutes(0, 0, 0);
  else anchor.setHours(0, 0, 0, 0);

  const start = anchor.getTime() - (count - 1) * step;
  const buckets: { start: Date; total: number }[] = [];
  for (let i = 0; i < count; i++) {
    buckets.push({ start: new Date(start + i * step), total: 0 });
  }

  let sum = 0;
  let paidCount = 0;
  for (const p of payments) {
    if (!PAID_STATUSES.has(String(p.status).toLowerCase())) continue;
    const t = new Date(p.created_at).getTime();
    const idx = Math.floor((t - start) / step);
    if (idx < 0 || idx >= count) continue;
    const amount = Math.abs(Number(p.amount) || 0);
    buckets[idx].total += amount;
    sum += amount;
    paidCount += 1;
  }

  const peak = Math.max(1, ...buckets.map((b) => b.total));
  const axis = [
    buckets[0].start,
    buckets[Math.floor((count - 1) / 2)].start,
    buckets[count - 1].start,
  ].map((d) =>
    range === "24ч"
      ? clockLabel(d)
      : d.toLocaleDateString(localeTag(), { day: "numeric", month: "short" })
  );

  return { buckets, peak, sum, paidCount, axis };
}

type Capacity = { label: string; value: string; pct: number; warn: boolean };

function clusterCapacity(
  nodes: AdminDashboardNode[],
  t: TranslateFn
): Capacity[] {
  const cpuValues = nodes
    .map((n) => n.cpu_percent)
    .filter((v): v is number => typeof v === "number");
  const cpuAvg = cpuValues.length
    ? cpuValues.reduce((a, b) => a + b, 0) / cpuValues.length
    : null;

  let ramTotal = 0;
  let ramUsed = 0;
  let diskTotal = 0;
  let diskUsed = 0;
  for (const n of nodes) {
    const rt = parseBytes(n.ram_total);
    if (rt > 0) {
      ramTotal += rt;
      ramUsed += (rt * (n.ram_percent ?? 0)) / 100;
    }
    diskTotal += parseBytes(n.disk_total);
    diskUsed += parseBytes(n.disk_used);
  }

  const ramPct = pct(ramUsed, ramTotal);
  const diskPct = pct(diskUsed, diskTotal);

  return [
    {
      label: t("admin.dashboard.capacity.cpu"),
      value: cpuAvg === null ? "—" : `${Math.round(cpuAvg)}%`,
      pct: cpuAvg ?? 0,
      warn: (cpuAvg ?? 0) >= 85,
    },
    {
      label: t("admin.dashboard.capacity.ram"),
      value:
        ramTotal > 0
          ? `${bytesLabel(ramUsed, t)} / ${bytesLabel(ramTotal, t)}`
          : "—",
      pct: ramPct,
      warn: ramPct >= 85,
    },
    {
      label: t("admin.dashboard.capacity.disk"),
      value:
        diskTotal > 0
          ? `${bytesLabel(diskUsed, t)} / ${bytesLabel(diskTotal, t)}`
          : "—",
      pct: diskPct,
      warn: diskPct >= 85,
    },
  ];
}

function NodeMetric({ value, warn }: { value: number | undefined; warn: boolean }) {
  const known = typeof value === "number";
  return (
    <div>
      <div
        className={cn(
          "font-mono text-[12.5px]",
          !known
            ? "text-[var(--vx-ink-faint)]"
            : warn
              ? "text-[var(--vx-warn)]"
              : "text-[var(--vx-fg)]"
        )}
      >
        {known ? `${Math.round(value)}%` : "—"}
      </div>
      <div className="mt-1.5 h-1 overflow-hidden rounded-[2px] bg-[var(--vx-tint)]">
        <div
          className={cn("h-1", warn ? "bg-[var(--vx-warn)]" : "bg-[var(--vx-fg)]")}
          style={{ width: `${known ? value : 0}%` }}
        />
      </div>
    </div>
  );
}

export function AdminDashboardPageContent() {
  const t = useT();
  const [range, setRange] = useState<Range>("30д");

  const dashboardQuery = useQuery({
    queryKey: queryKeys.adminDashboard,
    queryFn: fetchAdminDashboard,
  });
  const nodesQuery = useQuery({
    queryKey: queryKeys.adminDashboardNodes,
    queryFn: fetchAdminDashboardNodes,
  });
  const supportQuery = useQuery({
    queryKey: queryKeys.adminSupport,
    queryFn: async () => (await fetchAdminSupport()).tickets ?? [],
  });
  const usersQuery = useQuery({ queryKey: queryKeys.users, queryFn: fetchUsers });
  const serversQuery = useQuery({ queryKey: queryKeys.servers, queryFn: fetchServers });
  const paymentsQuery = useQuery({
    queryKey: ["admin-billing", "dashboard"],
    queryFn: async () => (await fetchAdminBilling(500)).payments ?? [],
  });
  const readinessQuery = useQuery({
    queryKey: queryKeys.adminReadiness,
    queryFn: fetchAdminReadiness,
    staleTime: 5 * 60 * 1000,
  });
  const blockers = readinessQuery.data?.blockers ?? [];

  const logsQuery = useQuery({
    queryKey: ["admin-logs", "dashboard"],
    queryFn: async () => (await fetchAdminLogsFiltered({ limit: 6 })).logs ?? [],
  });

  const loading =
    dashboardQuery.isLoading ||
    nodesQuery.isLoading ||
    supportQuery.isLoading ||
    usersQuery.isLoading ||
    serversQuery.isLoading;

  const nodes = useMemo(() => nodesQuery.data ?? [], [nodesQuery.data]);
  const servers = useMemo(() => serversQuery.data ?? [], [serversQuery.data]);
  const users = useMemo(() => usersQuery.data ?? [], [usersQuery.data]);
  const tickets = useMemo(() => supportQuery.data ?? [], [supportQuery.data]);

  const stats = useMemo(() => {
    const dash = dashboardQuery.data;
    const now = Date.now();

    const nodesOnline = nodes.filter((n) => n.is_online).length;
    const serversRunning = servers.filter((s) => s.status === "running").length;
    const serversStopped = servers.filter((s) => s.status === "stopped").length;

    const newUsers7d = users.filter(
      (u) => now - new Date(u.created_at).getTime() < 7 * MS_DAY
    ).length;

    const openTickets = tickets.filter((t) => t.status === "open");
    const overdueTickets = openTickets.filter(
      (t) => now - new Date(t.created_at).getTime() > MS_DAY
    ).length;
    const closed24h = tickets.filter(
      (t) => t.status !== "open" && now - new Date(t.created_at).getTime() < MS_DAY
    ).length;

    const totalNodes = dash?.nodes ?? nodes.length;
    const totalServers = dash?.servers ?? servers.length;
    const totalUsers = dash?.users ?? users.length;
    const provInstalling = dash?.provisioning_installing ?? 0;
    const provFailed = dash?.provisioning_failed ?? 0;

    return {
      totalNodes,
      nodesOnline,
      nodesOffline: nodes.length - nodesOnline,
      totalServers,
      serversRunning,
      serversStopped,
      totalUsers,
      newUsers7d,
      openTickets: openTickets.length,
      overdueTickets,
      closed24h,
      queue: [...openTickets].sort(
        (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
      ),
      revenue24h: dash?.revenue_24h ?? 0,
      revenue7d: dash?.revenue_7d ?? 0,
      revenue30d: dash?.revenue_30d ?? 0,
      provInstalling,
      provFailed,
      provReady: Math.max(0, totalServers - provInstalling - provFailed),
      expiring: dash?.expiring_count ?? 0,
      staff: users.filter((u) => isStaffRole(u.role)),
      fresh: totalNodes === 0 && totalServers === 0,
    };
  }, [dashboardQuery.data, nodes, servers, users, tickets]);

  const revenue = useMemo(
    () => revenueSeries(paymentsQuery.data ?? [], range),
    [paymentsQuery.data, range]
  );

  // Без useMemo: t() стабильна, и мемо не пересчиталось бы при смене языка —
  // подписи ёмкости застыли бы на прежнем.
  const capacity = clusterCapacity(nodes, t);

  const rangeTotal =
    range === "24ч"
      ? stats.revenue24h
      : range === "7д"
        ? stats.revenue7d
        : stats.revenue30d;

  const offlineNode = nodes.find((n) => !n.is_online);
  const affectedServers = nodes
    .filter((n) => !n.is_online)
    .reduce((sum, n) => sum + n.servers_count, 0);

  const provRows = [
    {
      label: t("admin.dashboard.prov.ready"),
      value: stats.provReady,
      total: stats.totalServers,
      warn: false,
    },
    {
      label: t("admin.dashboard.prov.in_progress"),
      value: stats.provInstalling,
      total: stats.totalServers,
      warn: false,
    },
    {
      label: t("admin.dashboard.prov.failed"),
      value: stats.provFailed,
      total: stats.totalServers,
      warn: stats.provFailed > 0,
    },
  ];

  const kpis = [
    {
      label: t("admin.dashboard.kpi.revenue_30d"),
      icon: "ri-line-chart-line",
      value: money(stats.revenue30d, t),
      unit: "₽",
      delta: `${money(stats.revenue24h, t)} ₽`,
      warn: false,
      note: t("admin.dashboard.kpi.last_24h"),
    },
    {
      label: t("common.users"),
      icon: "ri-group-line",
      value: nf(stats.totalUsers),
      unit: "",
      delta: `+${stats.newUsers7d}`,
      warn: false,
      note: t("admin.dashboard.kpi.last_7d"),
    },
    {
      label: t("common.servers"),
      icon: "ri-server-line",
      value: nf(stats.totalServers),
      unit: "",
      delta: t("admin.dashboard.kpi.online", { count: stats.serversRunning }),
      warn: false,
      note: t("admin.dashboard.kpi.stopped", { count: stats.serversStopped }),
    },
    {
      label: t("admin.dashboard.kpi.tickets"),
      icon: "ri-customer-service-2-line",
      value: nf(stats.openTickets),
      unit: t("admin.dashboard.kpi.open"),
      delta:
        stats.overdueTickets > 0
          ? t("admin.dashboard.kpi.overdue", { count: stats.overdueTickets })
          : "—",
      warn: stats.overdueTickets > 0,
      note: t("admin.dashboard.kpi.total", { count: tickets.length }),
    },
  ];

  if (loading) {
    return (
      <PageShell variant="admin">
        <div className="font-panel space-y-5">
          <Skeleton className="h-[120px] w-full rounded-[20px]" />
          <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-6">
            <Skeleton className="h-[168px] rounded-[20px] lg:col-span-3" />
            <Skeleton className="h-[168px] rounded-[20px] lg:col-span-3" />
            <Skeleton className="h-[168px] rounded-[20px] lg:col-span-3" />
            <Skeleton className="h-[168px] rounded-[20px] lg:col-span-3" />
            <Skeleton className="h-[330px] rounded-[20px] lg:col-span-4" />
            <Skeleton className="h-[330px] rounded-[20px] lg:col-span-2" />
            <Skeleton className="h-[300px] rounded-[20px] lg:col-span-4" />
            <Skeleton className="h-[300px] rounded-[20px] lg:col-span-2" />
            <Skeleton className="h-[280px] rounded-[20px] lg:col-span-3" />
            <Skeleton className="h-[280px] rounded-[20px] lg:col-span-3" />
          </div>
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="admin">
      <div className="font-panel">
        <div className="flex flex-wrap items-end justify-between gap-[30px]">
          <div>
            <div className="font-mono text-xs tracking-[0.12em] text-[var(--vx-ink-faint)] uppercase">
              {new Date().toLocaleDateString(localeTag(), {
                day: "numeric",
                month: "long",
                year: "numeric",
              })}
            </div>
            <h1 className="mt-3 text-[34px] leading-[1.02] font-semibold tracking-[-0.035em] sm:text-[42px]">
              {t("admin.dashboard.title")}
            </h1>
            <p className="mt-3 text-[15px] text-muted-foreground">
              {stats.fresh
                ? t("admin.dashboard.fresh")
                : `${stats.totalNodes} ${countWord(t, "admin.dashboard.count.node", stats.totalNodes)} · ${nf(stats.totalServers)} ${countWord(t, "admin.dashboard.count.server", stats.totalServers)} · ${nf(stats.totalUsers)} ${countWord(t, "admin.dashboard.count.user", stats.totalUsers)}`}
            </p>
          </div>
          <div className="flex flex-wrap gap-2.5">
            <Link
              href="/admin/daemons"
              className="vx-btn-ghost flex items-center gap-[9px] rounded-full px-5 py-3 text-sm font-medium"
            >
              <i className="ri-terminal-box-line" />
              {t("admin.dashboard.daemon_logs")}
            </Link>
            <Link
              href="/admin/locations"
              className="vx-btn flex items-center gap-[9px] rounded-full px-[22px] py-3 text-sm font-semibold"
            >
              <i className="ri-add-line" />
              {t("admin.dashboard.add_node")}
            </Link>
          </div>
        </div>

        {offlineNode && (
          <div className="mt-[26px] flex flex-wrap items-center gap-4 rounded-2xl border border-[var(--vx-border-strong)] bg-[var(--vx-warn-tint)] px-5 py-4">
            <div className="flex size-[34px] flex-shrink-0 items-center justify-center rounded-[11px] bg-[rgba(232,160,60,0.13)] text-[var(--vx-warn)]">
              <i className="ri-error-warning-line text-[17px]" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="text-sm font-semibold">
                {stats.nodesOffline === 1
                  ? t("admin.dashboard.node_offline", { name: offlineNode.name })
                  : t("admin.dashboard.nodes_offline", {
                      count: stats.nodesOffline,
                      word: countWord(
                        t,
                        "admin.dashboard.count.node",
                        stats.nodesOffline
                      ),
                    })}
              </div>
              <div className="mt-[3px] text-[13px] text-muted-foreground">
                {affectedServers > 0
                  ? t("admin.dashboard.affected", {
                      count: nf(affectedServers),
                      word: countWord(
                        t,
                        "admin.dashboard.count.server",
                        affectedServers
                      ),
                    })
                  : t("admin.dashboard.daemon_unreachable")}
              </div>
            </div>
            <Link
              href="/admin/daemons"
              className="rounded-full border border-[var(--vx-border-strong)] px-[18px] py-[9px] text-[13px] font-semibold whitespace-nowrap text-[var(--vx-warn)] transition-colors hover:bg-[var(--vx-warn-tint)]"
            >
              {t("admin.dashboard.investigate")}
            </Link>
          </div>
        )}

        {blockers.length > 0 && (
          <div className="mt-[26px] rounded-2xl border border-[var(--vx-border-strong)] bg-[var(--vx-warn-tint)] px-5 py-4">
            <div className="flex flex-wrap items-center gap-4">
              <div className="flex size-[34px] flex-shrink-0 items-center justify-center rounded-[11px] bg-[rgba(232,160,60,0.13)] text-[var(--vx-warn)]">
                <i className="ri-shield-check-line text-[17px]" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="text-sm font-semibold">
                  {t("admin.dashboard.blockers_title", {
                    count: blockers.length,
                    word: countWord(
                      t,
                      "admin.dashboard.count.blocker",
                      blockers.length
                    ),
                  })}
                </div>
                <div className="mt-[3px] text-[13px] text-muted-foreground">
                  {t("admin.dashboard.blockers_hint")}
                </div>
              </div>
              <Link
                href="/admin/settings"
                className="rounded-full border border-[var(--vx-border-strong)] px-[18px] py-[9px] text-[13px] font-semibold whitespace-nowrap text-[var(--vx-warn)] transition-colors hover:bg-[rgba(232,160,60,0.13)]"
              >
                {t("common.settings")}
              </Link>
            </div>
            <ul className="mt-3 space-y-2 border-t border-[var(--vx-border-strong)] pt-3">
              {blockers.map((b) => (
                <li key={b.key} className="flex items-start gap-[13px]">
                  <span className="mt-1.5 size-[7px] flex-shrink-0 rounded-full bg-[var(--vx-warn)]" />
                  <div className="min-w-0 flex-1">
                    <div className="text-[13.5px] leading-[1.4] font-medium">
                      {t(`admin.dashboard.blocker_${b.key}`)}
                    </div>
                    <div className="mt-[3px] text-[13px] text-muted-foreground">
                      {t(`admin.dashboard.blocker_${b.key}_fallback`)}
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          </div>
        )}

        <div className="mt-5 grid grid-cols-1 gap-3.5 sm:grid-cols-2 lg:grid-cols-6">
          {kpis.map((k) => (
            <div key={k.label} className={cn(CARD, "px-6 py-[22px] lg:col-span-3")}>
              <div className="flex items-center justify-between">
                <span className="text-[13px] font-semibold tracking-[0.02em] text-muted-foreground">
                  {k.label}
                </span>
                <i className={cn(k.icon, "text-[17px] text-[var(--vx-ink-ghost)]")} />
              </div>
              <div className="mt-5 flex items-baseline gap-[9px]">
                <span className="text-[40px] leading-none font-semibold tracking-[-0.04em]">
                  {k.value}
                </span>
                {k.unit && (
                  <span className="text-sm font-medium text-[var(--vx-ink-faint)]">
                    {k.unit}
                  </span>
                )}
              </div>
              <div className="mt-[11px] flex items-center gap-2.5 text-[13px]">
                <span
                  className={cn(
                    "font-semibold",
                    k.warn ? "text-[var(--vx-warn)]" : "text-[var(--vx-fg)]"
                  )}
                >
                  {k.delta}
                </span>
                <span className="text-[var(--vx-ink-faint)]">{k.note}</span>
              </div>
            </div>
          ))}

          <div className={cn(CARD, "px-[26px] py-6 sm:col-span-2 lg:col-span-4")}>
            <div className="flex flex-wrap items-center gap-3.5">
              <span className="text-base font-semibold">
                {t("admin.dashboard.revenue")}
              </span>
              <span className="font-mono text-xs text-[var(--vx-ink-faint)]">
                {t("admin.dashboard.revenue_period", { value: nf(rangeTotal) })}
              </span>
              <div className="flex-1" />
              <div className="flex gap-1 rounded-full border border-[var(--vx-tint)] p-1">
                {RANGES.map((r) => (
                  <button
                    key={r.id}
                    type="button"
                    onClick={() => setRange(r.id)}
                    className={cn(
                      "rounded-full px-3.5 py-1.5 text-[12.5px] transition-colors",
                      r.id === range
                        ? "bg-[var(--vx-tint)] font-semibold text-[var(--vx-fg-strong)]"
                        : "font-medium text-[var(--vx-ink-faint)] hover:text-[var(--vx-fg)]"
                    )}
                  >
                    {t(r.labelKey)}
                  </button>
                ))}
              </div>
            </div>

            <div className="mt-6 flex h-[150px] items-end gap-[3px]">
              {revenue.buckets.map((b, i) => (
                <div
                  key={i}
                  className="flex h-full flex-1 flex-col justify-end"
                  title={`${nf(b.total)} ₽`}
                >
                  <div
                    className={cn(
                      "rounded-t-[3px]",
                      b.total > 0 ? "bg-[var(--vx-fg-strong)]" : "bg-[var(--vx-border-strong)]"
                    )}
                    style={{
                      height: b.total > 0 ? `${pct(b.total, revenue.peak)}%` : "2%",
                    }}
                  />
                </div>
              ))}
            </div>

            <div className="mt-3 flex justify-between font-mono text-[11px] text-[var(--vx-ink-ghost)]">
              <span>{revenue.axis[0]}</span>
              <span>{revenue.axis[1]}</span>
              <span>{revenue.axis[2]}</span>
            </div>

            <div className="mt-4 flex flex-wrap items-center gap-x-[18px] gap-y-2 border-t border-[var(--vx-panel-line)] pt-4">
              <div className="flex items-center gap-[7px] text-xs text-muted-foreground">
                <span className="size-2 rounded-[3px] bg-[var(--vx-fg-strong)]" />
                {t("admin.dashboard.successful_payments")}
              </div>
              <div className="flex-1" />
              <div className="text-[12.5px] text-[var(--vx-ink-faint)]">
                {t("admin.dashboard.payments_count")}{" "}
                <span className="font-semibold text-[var(--vx-fg)]">{revenue.paidCount}</span>
              </div>
              <div className="text-[12.5px] text-[var(--vx-ink-faint)]">
                {t("admin.dashboard.avg_payment")}{" "}
                <span className="font-semibold text-[var(--vx-fg)]">
                  {revenue.paidCount > 0
                    ? `${nf(revenue.sum / revenue.paidCount)} ₽`
                    : "—"}
                </span>
              </div>
            </div>
          </div>

          <div className={cn(CARD, "flex flex-col px-[26px] py-6 sm:col-span-2 lg:col-span-2")}>
            <div className="flex items-center justify-between">
              <span className="text-base font-semibold">
                {t("admin.dashboard.infrastructure")}
              </span>
              <i className="ri-hard-drive-3-line text-[17px] text-[var(--vx-ink-ghost)]" />
            </div>
            <div className="mt-5 flex items-baseline gap-2">
              <span className="text-[40px] leading-none font-semibold tracking-[-0.04em]">
                {stats.totalNodes > 0
                  ? `${Math.round(pct(stats.nodesOnline, stats.totalNodes))}%`
                  : "—"}
              </span>
              <span className="text-sm text-[var(--vx-ink-faint)]">
                {t("admin.dashboard.nodes_online_label")}
              </span>
            </div>

            <div className="mt-[22px] flex flex-col gap-3.5">
              {capacity.map((c) => (
                <div key={c.label}>
                  <div className="flex items-baseline justify-between text-[12.5px]">
                    <span className="text-muted-foreground">{c.label}</span>
                    <span className="font-mono text-[var(--vx-fg)]">{c.value}</span>
                  </div>
                  <div className="mt-2 h-[5px] overflow-hidden rounded-[3px] bg-[var(--vx-tint)]">
                    <div
                      className={cn("h-[5px]", c.warn ? "bg-[var(--vx-warn)]" : "bg-[var(--vx-fg-strong)]")}
                      style={{ width: `${c.pct}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>

            <div className="flex-1" />
            <Link
              href="/admin/locations"
              className="mt-[22px] flex items-center gap-[7px] text-[13px] font-semibold text-[var(--vx-fg-strong)]"
            >
              {t("admin.dashboard.all_nodes")}
              <ArrowIcon className="size-3.5" />
            </Link>
          </div>

          <div className={cn(CARD, "px-[26px] py-6 sm:col-span-2 lg:col-span-4")}>
            <div className="flex flex-wrap items-center gap-3.5">
              <span className="text-base font-semibold">
                {t("admin.dashboard.nodes")}
              </span>
              <span className="font-mono text-xs text-[var(--vx-ink-faint)]">
                {stats.totalNodes === 0
                  ? t("admin.dashboard.not_connected")
                  : `${stats.nodesOnline} online · ${stats.nodesOffline} offline`}
              </span>
              <div className="flex-1" />
              <Link
                href="/admin/locations"
                className="flex items-center gap-[7px] text-[13px] font-medium text-[var(--vx-fg-strong)]"
              >
                {t("admin.dashboard.manage")}
                <ArrowIcon className="size-3.5" />
              </Link>
            </div>

            {nodes.length === 0 ? (
              <div className="py-10 text-center">
                <div className="text-sm text-muted-foreground">
                  {t("admin.dashboard.nodes_empty")}
                </div>
                <div className="mt-[7px] text-[13px] text-[var(--vx-ink-ghost)]">
                  {t("admin.dashboard.nodes_empty_hint")}
                </div>
              </div>
            ) : (
              <div className="-mx-[26px] overflow-x-auto px-[26px]">
                <div className="min-w-[520px]">
                  <div
                    className={cn(
                      MONO_LABEL,
                      "mt-5 grid grid-cols-[1.5fr_1fr_1fr_1fr_0.9fr] gap-3 border-b border-[var(--vx-panel-line)] pb-2.5 tracking-[0.1em]"
                    )}
                  >
                    <span>{t("admin.dashboard.col_node")}</span>
                    <span>CPU</span>
                    <span>RAM</span>
                    <span>{t("admin.dashboard.col_disk")}</span>
                    <span className="text-right">{t("common.servers")}</span>
                  </div>
                  {nodes.map((n) => {
                    const diskTotal = parseBytes(n.disk_total);
                    const diskPct =
                      diskTotal > 0 ? pct(parseBytes(n.disk_used), diskTotal) : undefined;
                    const location =
                      [n.country, n.ip_address || n.fqdn].filter(Boolean).join(" · ") ||
                      n.code ||
                      "—";
                    return (
                      <Link
                        key={n.id}
                        href={`/admin/locations/${n.id}`}
                        className="grid grid-cols-[1.5fr_1fr_1fr_1fr_0.9fr] items-center gap-3 border-b border-[var(--vx-inset)] py-3.5 transition-colors hover:bg-[var(--vx-elevated)]"
                      >
                        <div className="flex min-w-0 items-center gap-2.5">
                          <span
                            className={cn(
                              "size-[7px] flex-shrink-0 rounded-full",
                              n.is_online ? "bg-[var(--vx-fg-strong)]" : "bg-[var(--vx-danger)]"
                            )}
                          />
                          <div className="min-w-0">
                            <div className="truncate text-[13.5px] font-semibold">
                              {n.name}
                            </div>
                            <div className="mt-0.5 truncate font-mono text-[11px] text-[var(--vx-faint)]">
                              {location}
                            </div>
                          </div>
                        </div>
                        <NodeMetric value={n.cpu_percent} warn={(n.cpu_percent ?? 0) >= 85} />
                        <NodeMetric value={n.ram_percent} warn={(n.ram_percent ?? 0) >= 85} />
                        <NodeMetric value={diskPct} warn={(diskPct ?? 0) >= 85} />
                        <div className="text-right font-mono text-[13px] text-[var(--vx-dim)]">
                          {n.servers_count}
                        </div>
                      </Link>
                    );
                  })}
                </div>
              </div>
            )}
          </div>

          <div className={cn(CARD, "flex flex-col px-[26px] py-6 sm:col-span-2 lg:col-span-2")}>
            <div className="flex items-center justify-between">
              <span className="text-base font-semibold">
                {t("admin.dashboard.provisioning")}
              </span>
              <span className="font-mono text-[11px] text-[var(--vx-ink-faint)]">
                {t("admin.dashboard.all_servers")}
              </span>
            </div>
            <div className="mt-5 flex flex-col gap-3">
              {provRows.map((p) => (
                <div key={p.label} className="flex items-center gap-3">
                  <span className="w-24 flex-shrink-0 text-[12.5px] text-muted-foreground">
                    {p.label}
                  </span>
                  <div className="h-2 flex-1 overflow-hidden rounded-full bg-[var(--vx-tint)]">
                    <div
                      className={cn("h-2", p.warn ? "bg-[var(--vx-danger)]" : "bg-[var(--vx-fg-strong)]")}
                      style={{ width: `${pct(p.value, p.total)}%` }}
                    />
                  </div>
                  <span className="w-[34px] text-right font-mono text-[13px] text-[var(--vx-fg)]">
                    {p.value}
                  </span>
                </div>
              ))}
            </div>
            <div className="flex-1" />
            <div className={cn(CARD_INNER, "mt-[22px] p-4")}>
              <div className="text-[12.5px] text-muted-foreground">
                {t("admin.dashboard.expiring")}
              </div>
              <div
                className={cn(
                  "mt-1.5 font-mono text-[22px]",
                  stats.expiring > 0 ? "text-[var(--vx-warn)]" : "text-[var(--vx-fg)]"
                )}
              >
                {stats.expiring}
              </div>
            </div>
          </div>

          <div className={cn(CARD, "px-[26px] py-6 sm:col-span-2 lg:col-span-6")}>
            <AdminLoadChart />
          </div>

          <div className={cn(CARD, "px-[26px] py-6 sm:col-span-2 lg:col-span-3")}>
            <div className="flex items-center">
              <span className="text-base font-semibold">
                {t("admin.dashboard.events")}
              </span>
              <div className="flex-1" />
              <Link
                href="/admin/logs"
                className="flex items-center gap-[7px] text-[13px] font-medium text-[var(--vx-fg-strong)]"
              >
                {t("admin.dashboard.journal")}
                <ArrowIcon className="size-3.5" />
              </Link>
            </div>
            <div className="mt-2">
              {(logsQuery.data ?? []).length === 0 ? (
                <div className="py-11 text-center">
                  <div className="text-sm text-muted-foreground">
                    {t("admin.dashboard.events_empty")}
                  </div>
                  <div className="mt-[7px] text-[13px] text-[var(--vx-ink-ghost)]">
                    {t("admin.dashboard.events_empty_hint")}
                  </div>
                </div>
              ) : (
                (logsQuery.data ?? []).map((e, i) => (
                  <div
                    key={`${e.created_at}-${i}`}
                    className="flex items-start gap-[13px] border-t border-[var(--vx-panel-line)] py-[13px]"
                  >
                    <span className="mt-1.5 size-[7px] flex-shrink-0 rounded-full bg-[var(--vx-faint)]" />
                    <div className="min-w-0 flex-1">
                      <div className="text-[13.5px] leading-[1.4] font-medium break-words">
                        {e.action}
                      </div>
                      <div className="mt-[3px] truncate font-mono text-[11px] text-[var(--vx-faint)]">
                        {e.resource || "—"}
                      </div>
                    </div>
                    <span className="font-mono text-[11px] whitespace-nowrap text-[var(--vx-ink-ghost)]">
                      {eventTime(e.created_at)}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>

          <div className={cn(CARD, "px-[26px] py-6 sm:col-span-2 lg:col-span-3")}>
            <div className="flex items-center gap-3">
              <span className="text-base font-semibold">
                {t("admin.dashboard.support_queue")}
              </span>
              <div className="flex-1" />
              <span className="font-mono text-[11px] text-[var(--vx-ink-faint)]">
                {stats.openTickets === 0
                  ? t("admin.dashboard.queue_empty_short")
                  : t("admin.dashboard.queue_open", {
                      count: stats.openTickets,
                    }) +
                    (stats.overdueTickets > 0
                      ? t("admin.dashboard.queue_overdue", {
                          count: stats.overdueTickets,
                        })
                      : "")}
              </span>
            </div>

            {stats.queue.length === 0 ? (
              <div className="mt-4 py-11 text-center">
                <div className="text-sm text-muted-foreground">
                  {t("admin.dashboard.tickets_empty")}
                </div>
                <div className="mt-[7px] text-[13px] text-[var(--vx-ink-ghost)]">
                  {t("admin.dashboard.tickets_empty_hint")}
                </div>
              </div>
            ) : (
              <div className="mt-4 flex flex-col gap-2.5">
                {/* Параметр назван ticket, а не t: имя t занято функцией перевода. */}
                {stats.queue.slice(0, 4).map((ticket) => {
                  const waited = Date.now() - new Date(ticket.created_at).getTime();
                  const priority =
                    waited > MS_DAY ? "P1" : waited > 4 * MS_HOUR ? "P2" : "P3";
                  const urgent = priority === "P1";
                  return (
                    <Link
                      key={ticket.id}
                      href={`/admin/support/${ticket.id}`}
                      className={cn(
                        "flex items-center gap-3.5 rounded-[14px] border p-4 transition-colors hover:border-[var(--vx-border-hover)]",
                        urgent
                          ? "border-[var(--vx-border-strong)] bg-[var(--vx-elevated)]"
                          : "border-[var(--vx-tint)] bg-[var(--vx-panel-card-2)]"
                      )}
                    >
                      <div
                        className={cn(
                          "flex size-[30px] flex-shrink-0 items-center justify-center rounded-[10px] font-mono text-[10.5px] font-medium",
                          urgent
                            ? "bg-[rgba(224,122,122,0.13)] text-[var(--vx-danger)]"
                            : "bg-[var(--vx-panel-tint)] text-muted-foreground"
                        )}
                      >
                        {priority}
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-[13.5px] font-semibold">
                          {ticket.subject}
                        </div>
                        <div className="mt-[3px] font-mono text-xs text-[var(--vx-ink-faint)]">
                          #{ticket.id.slice(0, 8)}
                        </div>
                      </div>
                      <span
                        className={cn(
                          "font-mono text-[11.5px] whitespace-nowrap",
                          waited > 4 * MS_HOUR
                            ? "text-[var(--vx-warn)]"
                            : "text-muted-foreground"
                        )}
                      >
                        {durationLabel(waited, t)}
                      </span>
                    </Link>
                  );
                })}
              </div>
            )}

            <div className="mt-[18px] flex gap-[26px] border-t border-[var(--vx-panel-line)] pt-4">
              <div>
                <div className={MONO_LABEL}>
                  {t("admin.dashboard.stat_open")}
                </div>
                <div className="mt-1.5 text-base font-semibold">{stats.openTickets}</div>
              </div>
              <div>
                <div className={MONO_LABEL}>
                  {t("admin.dashboard.stat_overdue")}
                </div>
                <div
                  className={cn(
                    "mt-1.5 text-base font-semibold",
                    stats.overdueTickets > 0 && "text-[var(--vx-warn)]"
                  )}
                >
                  {stats.overdueTickets}
                </div>
              </div>
              <div>
                <div className={MONO_LABEL}>
                  {t("admin.dashboard.stat_closed")}
                </div>
                <div className="mt-1.5 text-base font-semibold">{stats.closed24h}</div>
              </div>
            </div>
          </div>

          <div className={cn(CARD, "px-[26px] py-6 sm:col-span-2 lg:col-span-6")}>
            <div className="flex flex-wrap items-center gap-3.5">
              <span className="text-base font-semibold">
                {t("admin.dashboard.staff")}
              </span>
              <span className="font-mono text-xs text-[var(--vx-ink-faint)]">
                {stats.staff.length}{" "}
                {countWord(t, "admin.dashboard.count.admin", stats.staff.length)}
              </span>
              <div className="flex-1" />
              <Link
                href="/admin/users"
                className="vx-btn-ghost flex items-center gap-2 rounded-full px-4 py-[9px] text-[13px] font-medium"
              >
                <i className="ri-add-line" />
                {t("admin.dashboard.invite")}
              </Link>
            </div>

            {stats.staff.length === 0 ? (
              <div className="py-10 text-center text-sm text-muted-foreground">
                {t("admin.dashboard.staff_empty")}
              </div>
            ) : (
              <div className="mt-[18px] grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
                {stats.staff.map((u) => (
                  <Link
                    key={u.id}
                    href={`/admin/users/${u.id}`}
                    className={cn(
                      CARD_INNER,
                      "px-[18px] py-4 transition-colors hover:border-[var(--vx-border-hover)]"
                    )}
                  >
                    <div className="flex items-center gap-[11px]">
                      <div className="flex size-8 flex-shrink-0 items-center justify-center rounded-full bg-[var(--vx-tint)] text-xs font-semibold text-[var(--vx-fg-strong)] uppercase">
                        {userDisplayName(u).charAt(0)}
                      </div>
                      <div className="min-w-0">
                        <div className="truncate text-[13.5px] font-semibold">
                          {userDisplayName(u)}
                        </div>
                        <div className="mt-0.5 text-[11.5px] text-[var(--vx-ink-faint)]">
                          {roleLabel(u.role)}
                        </div>
                      </div>
                    </div>
                    <div className="mt-3.5 flex items-center justify-between border-t border-[var(--vx-panel-line)] pt-3">
                      <span className="truncate font-mono text-[11px] text-[var(--vx-faint)]">
                        {u.email}
                      </span>
                      <span
                        className={cn(
                          "ml-2 size-1.5 flex-shrink-0 rounded-full",
                          u.is_blocked ? "bg-[var(--vx-danger)]" : "bg-[var(--vx-fg-strong)]"
                        )}
                      />
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </PageShell>
  );
}
