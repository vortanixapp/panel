"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { MoreHorizontal, Plus, RotateCw, Search, Square, Play } from "lucide-react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { fetchMyServers, powerServer, type DashboardServer } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";
import {
  canStartServer,
  canStopServer,
  filterAndSortServers,
  getServerStatus,
  getServerStatusDotClass,
  serverSortOptions,
  serverStatusFilterOptions,
  type ServerSortOption,
  type ServerStatusCategory,
} from "@/lib/server-status";

const COLUMNS =
  "grid-cols-[minmax(230px,1.4fr)_150px_minmax(190px,1fr)_180px_150px_196px]";

function meterTone(value: number) {
  if (value >= 85) return { bar: "bg-rose-500", text: "text-rose-500" };
  if (value >= 65) return { bar: "bg-amber-500", text: "text-amber-500" };
  return { bar: "bg-emerald-500", text: "text-emerald-500" };
}

function statusAccent(category: string) {
  if (["running", "active"].includes(category)) return "bg-emerald-500";
  if (["failed", "missing"].includes(category)) return "bg-rose-500";
  if (["installing", "reinstalling", "updating", "suspended"].includes(category))
    return "bg-amber-500";
  return "bg-muted-foreground";
}

export function MyServersPageContent() {
  const t = useT();
  const queryClient = useQueryClient();
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<ServerStatusCategory | "all">(
    "all"
  );
  const [gameFilter, setGameFilter] = useState("all");
  const [sort, setSort] = useState<ServerSortOption>("name-asc");

  const { data, isLoading } = useQuery({
    queryKey: ["my-servers"],
    queryFn: fetchMyServers,
    refetchInterval: 30_000,
  });

  const servers = useMemo(() => data?.servers ?? [], [data?.servers]);
  const total = data?.total ?? 0;
  const activeCount = data?.active_count ?? 0;
  const expiringSoon = data?.expiring_soon ?? 0;
  const failedCount = useMemo(
    () =>
      servers.filter((s) =>
        ["failed", "missing"].includes(getServerStatus(s).category)
      ).length,
    [servers]
  );

  const gameOptions = useMemo(() => {
    const names = new Set<string>();
    for (const s of servers) {
      if (s.game?.name) names.add(s.game.name);
    }
    return Array.from(names).sort((a, b) => a.localeCompare(b, "ru"));
  }, [servers]);

  const filteredServers = useMemo(
    () =>
      filterAndSortServers(servers, {
        search,
        statusFilter,
        gameFilter,
        sort,
      }),
    [servers, search, statusFilter, gameFilter, sort]
  );

  const hasActiveFilters =
    search.trim() !== "" || statusFilter !== "all" || gameFilter !== "all";

  function resetFilters() {
    setSearch("");
    setStatusFilter("all");
    setGameFilter("all");
    setSort("name-asc");
  }

  async function serverAction(id: string, action: string) {
    setActionLoading(id);
    try {
      await powerServer(id, action);
      toast.success(t("servers.list.command_sent", { action }));
      await queryClient.invalidateQueries({ queryKey: ["my-servers"] });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.error"));
    } finally {
      setActionLoading(null);
    }
  }

  if (isLoading) {
    return (
      <PageShell variant="user">
        <div className="w-full space-y-6">
          <Skeleton className="h-16 w-full rounded-2xl" />
          <Skeleton className="h-24 w-full rounded-2xl" />
          <Skeleton className="h-96 w-full rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="user">
      <div className="w-full space-y-6">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <div className="font-mono text-[11px] tracking-wider text-muted-foreground uppercase">
              {t("servers.list.breadcrumb")}
            </div>
            <h1 className="text-[27px] leading-none font-bold tracking-tight">
              {t("servers.list.title")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("servers.list.subtitle")}
            </p>
          </div>
          <Button asChild className="h-[38px] text-[13px]">
            <Link href="/rent-server">
              <Plus className="size-4" />
              {t("servers.list.rent")}
            </Link>
          </Button>
        </div>

        {servers.length === 0 ? (
          <div className="rounded-2xl border bg-card py-16 text-center">
            <h3 className="text-lg font-semibold">
              {t("servers.list.empty_title")}
            </h3>
            <p className="mt-2 text-sm text-muted-foreground">
              {t("servers.list.empty_hint")}
            </p>
            <Button asChild className="mt-6 h-[38px] text-[13px]">
              <Link href="/rent-server">
                <Plus className="size-4" />
                {t("servers.list.rent")}
              </Link>
            </Button>
          </div>
        ) : (
          <>
            <div className="grid gap-px overflow-hidden rounded-2xl border bg-border [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))]">
              <StatTile
                label={t("servers.list.stat_total")}
                value={total}
                note={t("servers.list.stat_total_note")}
              />
              <StatTile
                label={t("servers.list.stat_running")}
                value={activeCount}
                note={t("servers.list.stat_of_total", { total })}
                tone={activeCount ? "emerald" : undefined}
              />
              <StatTile
                label={t("servers.list.stat_expiring")}
                value={expiringSoon}
                note={t("servers.list.stat_expiring_note")}
                tone={expiringSoon ? "amber" : undefined}
              />
              <StatTile
                label={t("servers.list.stat_failed")}
                value={failedCount}
                note={t("servers.list.stat_failed_note")}
                tone={failedCount ? "rose" : undefined}
              />
            </div>

            <div className="flex flex-wrap items-center gap-2.5 rounded-xl border bg-card px-3.5 py-3">
              <div className="relative min-w-[200px] flex-1 basis-[260px]">
                <Search className="pointer-events-none absolute top-1/2 left-3 size-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder={t("servers.list.search_placeholder")}
                  className="h-9 rounded-lg pl-8.5 text-[13px] md:text-[13px]"
                />
              </div>
              <FilterSelect
                value={statusFilter}
                onChange={(v) =>
                  setStatusFilter(v as ServerStatusCategory | "all")
                }
                options={serverStatusFilterOptions(t)}
              />
              {gameOptions.length > 0 && (
                <FilterSelect
                  value={gameFilter}
                  onChange={setGameFilter}
                  options={[
                    { value: "all", label: t("servers.filter.game_all") },
                    ...gameOptions.map((name) => ({
                      value: name,
                      label: name,
                    })),
                  ]}
                />
              )}
              <FilterSelect
                value={sort}
                onChange={(v) => setSort(v as ServerSortOption)}
                options={serverSortOptions(t)}
              />
              {hasActiveFilters && (
                <Button
                  variant="outline"
                  className="h-9 rounded-lg text-[13px]"
                  onClick={resetFilters}
                >
                  {t("common.reset")}
                </Button>
              )}
              <span className="font-mono text-[11px] text-muted-foreground">
                {hasActiveFilters
                  ? t("servers.list.shown_count", {
                      shown: filteredServers.length,
                      total: servers.length,
                    })
                  : t("servers.list.total_count", { count: servers.length })}
              </span>
            </div>

            <div className="overflow-hidden rounded-2xl border bg-card">
              {filteredServers.length === 0 ? (
                <div className="flex flex-col items-center gap-2.5 px-6 py-14 text-center">
                  <span className="text-[15px] font-medium">
                    {t("servers.list.no_match_title")}
                  </span>
                  <span className="text-[13px] text-muted-foreground">
                    {t("servers.list.no_match_hint")}
                  </span>
                  <Button
                    variant="outline"
                    className="mt-2 h-[34px] text-[13px]"
                    onClick={resetFilters}
                  >
                    {t("servers.list.reset_filters")}
                  </Button>
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <div className="flex min-w-[1040px] flex-col">
                    <div
                      className={cn(
                        "grid gap-3.5 border-b bg-muted/40 py-3 pr-5 pl-6 font-mono text-[11px] tracking-wider text-muted-foreground uppercase",
                        COLUMNS
                      )}
                    >
                      <div>{t("common.server")}</div>
                      <div>{t("common.status")}</div>
                      <div>{t("servers.list.col_address")}</div>
                      <div>{t("servers.list.col_load")}</div>
                      <div>{t("servers.list.col_rent")}</div>
                      <div className="text-right">{t("common.actions")}</div>
                    </div>

                    {filteredServers.map((server) => (
                      <ServerRow
                        key={server.id}
                        server={server}
                        loading={actionLoading === server.id}
                        onAction={serverAction}
                      />
                    ))}
                  </div>
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </PageShell>
  );
}

function ServerRow({
  server,
  loading,
  onAction,
}: {
  server: DashboardServer;
  loading: boolean;
  onAction: (id: string, action: string) => void;
}) {
  const t = useT();
  const st = getServerStatus(server);
  const dot = getServerStatusDotClass(st);
  const expiresAt = server.expires_at ? new Date(server.expires_at) : null;
  const daysLeft = expiresAt
    ? Math.ceil((expiresAt.getTime() - Date.now()) / 86400000)
    : null;
  const expiresSoon = daysLeft !== null && daysLeft >= 0 && daysLeft <= 7;
  const addr =
    server.ip_address && server.port
      ? `${server.ip_address}:${server.port}`
      : server.ip_address || "—";
  const canStop = canStopServer(server);
  const canStart = canStartServer(server);

  return (
    <div
      className={cn(
        "relative grid items-center gap-3.5 border-b py-4 pr-5 pl-6 transition-colors hover:bg-muted/30",
        COLUMNS
      )}
    >
      <span
        className={cn(
          "absolute top-3 bottom-3 left-0 w-0.5 rounded-r-sm",
          statusAccent(st.category)
        )}
      />

      <div className="min-w-0 space-y-1.5">
        <div className="truncate text-[15px] font-medium">{server.name}</div>
        <div className="truncate text-xs text-muted-foreground">
          {server.game?.name || t("common.game")} ·{" "}
          {server.location?.name || t("common.location")}
        </div>
      </div>

      <div className="space-y-1.5">
        <span
          className={cn(
            "inline-flex items-center gap-2 text-[13px]",
            st.cls.includes("emerald")
              ? "text-emerald-500"
              : st.cls.includes("amber")
                ? "text-amber-500"
                : st.cls.includes("rose")
                  ? "text-rose-500"
                  : "text-muted-foreground"
          )}
        >
          <span className={cn("size-1.5 rounded-full", dot)} />
          {st.label}
        </span>
        {server.is_blocked && (
          <span className="block font-mono text-[11px] text-rose-500">
            {t("servers.list.blocked")}
          </span>
        )}
      </div>

      <div className="min-w-0 space-y-1.5">
        <div className="truncate font-mono text-[13px]">{addr}</div>
        <div className="truncate text-xs text-muted-foreground">
          {server.tariff?.name || t("servers.list.no_tariff")}
        </div>
      </div>

      <div className="space-y-2">
        <Meter label="CPU" value={server.cpu_percent} />
        <Meter label="RAM" value={server.ram_percent} />
      </div>

      <div className="space-y-1.5">
        <div className="font-mono text-[13px]">
          {expiresAt
            ? t("servers.list.until", {
                date: expiresAt.toLocaleDateString(localeTag()),
              })
            : t("servers.list.unlimited")}
        </div>
        {daysLeft !== null && (
          <div
            className={cn(
              "text-xs",
              expiresSoon ? "text-amber-500" : "text-muted-foreground"
            )}
          >
            {daysLeft < 0
              ? t("servers.list.expired")
              : expiresSoon
                ? t("servers.list.days_left", { days: daysLeft })
                : t("servers.list.days", { days: daysLeft })}
          </div>
        )}
      </div>

      <div className="flex items-center justify-end gap-1.5">
        <Button
          variant="outline"
          asChild
          className="h-[30px] rounded-lg px-3 text-xs font-medium"
        >
          <Link href={`/servers/${server.id}`}>
            {t("servers.list.manage")}
          </Link>
        </Button>
        <Button
          variant="outline"
          size="icon"
          className="size-[30px] rounded-lg"
          title={t("common.restart")}
          disabled={loading || !canStop}
          onClick={() => onAction(server.id, "restart")}
        >
          <RotateCw className="size-3.5" />
        </Button>
        {canStart ? (
          <Button
            variant="outline"
            size="icon"
            className="size-[30px] rounded-lg border-emerald-500/30 text-emerald-500 hover:text-emerald-500"
            title={t("common.start")}
            disabled={loading}
            onClick={() => onAction(server.id, "start")}
          >
            <Play className="size-3.5" />
          </Button>
        ) : (
          <Button
            variant="outline"
            size="icon"
            className="size-[30px] rounded-lg border-rose-500/30 text-rose-500 hover:text-rose-500"
            title={t("common.stop")}
            disabled={loading || !canStop}
            onClick={() => onAction(server.id, "stop")}
          >
            <Square className="size-3.5" />
          </Button>
        )}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="outline"
              size="icon"
              className="size-[30px] rounded-lg"
              title={t("common.more")}
            >
              <MoreHorizontal className="size-3.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem asChild>
              <Link href={`/servers/${server.id}/console`}>
                {t("server.tab.console")}
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href={`/servers/${server.id}/settings`}>
                {t("server.tab.settings")}
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href={`/servers/${server.id}/tariff`}>
                {t("servers.list.menu_tariff")}
              </Link>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}

function Meter({ label, value }: { label: string; value?: number }) {
  const known = typeof value === "number" && Number.isFinite(value);
  const pct = known ? Math.min(100, Math.max(0, value)) : 0;
  const tone = meterTone(pct);

  return (
    <div className="flex items-center gap-2">
      <span className="w-[26px] font-mono text-[10px] text-muted-foreground">
        {label}
      </span>
      <span className="h-[3px] flex-1 overflow-hidden rounded-sm bg-muted">
        {known && (
          <span
            className={cn("block h-full", tone.bar)}
            style={{ width: `${pct}%` }}
          />
        )}
      </span>
      <span className="w-[34px] text-right font-mono text-[11px] text-muted-foreground">
        {known ? `${Math.round(pct)}%` : "—"}
      </span>
    </div>
  );
}

function StatTile({
  label,
  value,
  note,
  tone,
}: {
  label: string;
  value: number;
  note: string;
  tone?: "emerald" | "amber" | "rose";
}) {
  return (
    <div className="flex flex-col gap-2 bg-card px-5 py-4">
      <span className="font-mono text-[10px] tracking-wider text-muted-foreground uppercase">
        {label}
      </span>
      <span className="flex items-baseline gap-2">
        <span
          className={cn(
            "text-[26px] leading-none font-semibold tracking-tight",
            tone === "emerald" && "text-emerald-500",
            tone === "amber" && "text-amber-500",
            tone === "rose" && "text-rose-500"
          )}
        >
          {value}
        </span>
        <span className="text-xs text-muted-foreground">{note}</span>
      </span>
    </div>
  );
}

function FilterSelect({
  value,
  onChange,
  options,
}: {
  value: string;
  onChange: (next: string) => void;
  options: readonly { value: string; label: string }[];
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="h-9 rounded-lg border border-input bg-transparent px-3 text-[13px] outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
    >
      {options.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  );
}
