"use client";

import { useEffect, useMemo, useState } from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Chip,
  EventIcon,
  ListEmpty,
  MON,
  MON_BTN,
  MON_BTN_PRIMARY,
  MON_CARD,
  dayGroupTitle,
  dayKey,
  formatDateTime,
  formatTime,
  plural,
  timeAgo,
  type Tone,
} from "@/components/user/account/shared";
import {
  downloadActivityCsv,
  fetchActivity,
  type ActivityEntry,
  type ActivityQuery,
} from "@/lib/api";
import { t } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const PAGE_SIZE = 100;

const CATEGORIES = [
  { key: "all", labelKey: "common.all" },
  { key: "server", labelKey: "common.server" },
  { key: "billing", labelKey: "dashboard.activity.cat_billing" },
  { key: "auth", labelKey: "dashboard.activity.cat_auth" },
  { key: "support", labelKey: "dashboard.support.title" },
];

const RANGES = [
  { value: 1, labelKey: "dashboard.activity.range_24h" },
  { value: 7, labelKey: "dashboard.activity.range_7d" },
  { value: 30, labelKey: "dashboard.activity.range_30d" },
];

function actionIcon(entry: ActivityEntry): { icon: string; tone: Tone } {
  const a = entry.action;
  if (a.startsWith("server.power")) return { icon: "ri-play-circle-line", tone: "info" };
  if (a.startsWith("server.reinstall")) return { icon: "ri-refresh-line", tone: "warn" };
  if (a.startsWith("server.ftp")) return { icon: "ri-folder-user-line", tone: "info" };
  if (a.startsWith("server.settings")) return { icon: "ri-equalizer-line", tone: "info" };
  if (a.startsWith("server.delete")) return { icon: "ri-delete-bin-line", tone: "bad" };
  if (a.startsWith("server.")) return { icon: "ri-server-line", tone: "info" };
  if (a.startsWith("billing.topup") || a.startsWith("wallet."))
    return { icon: "ri-bank-card-line", tone: "ok" };
  if (a.startsWith("bonus.")) return { icon: "ri-gift-line", tone: "ok" };
  if (a.startsWith("billing.") || a.startsWith("invoice."))
    return { icon: "ri-receipt-line", tone: "ok" };
  if (a.startsWith("auth.login")) return { icon: "ri-login-circle-line", tone: "info" };
  if (a.startsWith("auth.2fa")) return { icon: "ri-shield-check-line", tone: "ok" };
  if (a.startsWith("auth.") || a.startsWith("account."))
    return { icon: "ri-user-settings-line", tone: "info" };
  if (a.startsWith("support.") || a.startsWith("ticket."))
    return { icon: "ri-customer-service-2-line", tone: "info" };
  if (a.endsWith(".error") || a.endsWith(".failed"))
    return { icon: "ri-error-warning-line", tone: "bad" };
  return { icon: "ri-file-list-3-line", tone: "info" };
}

function shortResource(resource: string): string {
  if (!resource) return "";
  const idx = resource.indexOf(":");
  if (idx < 0) return resource.length > 28 ? `${resource.slice(0, 28)}…` : resource;
  const kind = resource.slice(0, idx);
  const id = resource.slice(idx + 1);
  return id.length > 12 ? `${kind}:${id.slice(0, 8)}…` : resource;
}

function metaPairs(meta: Record<string, unknown> | undefined): [string, string][] {
  if (!meta) return [];
  return Object.entries(meta)
    .map(([key, value]): [string, string] => [
      key,
      value === null || value === undefined
        ? "—"
        : typeof value === "object"
          ? JSON.stringify(value)
          : String(value),
    ])
    .filter(([, value]) => value !== "" && value !== "—")
    .slice(0, 12);
}

export function ActivityPageContent() {
  useT();
  const [category, setCategory] = useState("all");
  const [range, setRange] = useState(1);
  const [query, setQuery] = useState("");
  const [limit, setLimit] = useState(PAGE_SIZE);
  const [exporting, setExporting] = useState(false);
  const [opened, setOpened] = useState<number | null>(null);

  const debouncedQuery = useDebounced(query, 350);

  const params: ActivityQuery = {
    q: debouncedQuery || undefined,
    category,
    range,
    limit,
  };

  const activity = useQuery({
    queryKey: ["activity", debouncedQuery, category, range, limit],
    queryFn: () => fetchActivity(params),
    placeholderData: keepPreviousData,
  });

  const rows = activity.data?.activity ?? [];
  const stats = activity.data?.stats;

  const groups = useMemo(() => {
    const map = new Map<string, ActivityEntry[]>();
    for (const row of rows) {
      const key = dayKey(row.created_at);
      const list = map.get(key);
      if (list) list.push(row);
      else map.set(key, [row]);
    }
    return Array.from(map.entries()).map(([key, items]) => ({
      key,
      title: dayGroupTitle(items[0].created_at),
      items,
    }));
  }, [rows]);

  const exportCsv = async () => {
    setExporting(true);
    try {
      await downloadActivityCsv({ ...params, limit: undefined });
      toast.success(t("dashboard.activity.exported"));
    } catch {
      toast.error(t("dashboard.activity.export_failed"));
    } finally {
      setExporting(false);
    }
  };

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-4">
        <div className="flex flex-wrap items-end gap-3">
          <div className="min-w-[240px] flex-1">
            <h1 className="text-[26px] font-bold tracking-[-0.02em]">
              {t("dashboard.activity.title")}
            </h1>
            <p className="mt-1 text-[13px] text-[var(--vx-muted)]">
              {t("dashboard.activity.subtitle")}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              className={MON_BTN}
              onClick={() => void exportCsv()}
              disabled={exporting || rows.length === 0}
            >
              <i className="ri-download-2-line text-[15px]" />
              {exporting
                ? t("dashboard.activity.exporting")
                : t("dashboard.activity.export_csv")}
            </button>
            <button
              type="button"
              className={MON_BTN_PRIMARY}
              onClick={() => void activity.refetch()}
              disabled={activity.isFetching}
            >
              <i
                className={cn("ri-refresh-line text-[15px]", activity.isFetching && "animate-spin")}
              />
              {t("common.refresh")}
            </button>
          </div>
        </div>

        <div
          className={cn(
            "grid grid-cols-2 gap-px overflow-clip bg-[var(--vx-border)] sm:grid-cols-4",
            MON_CARD
          )}
        >
          <SummaryCell
            label={t("dashboard.activity.stat_events")}
            icon="ri-pulse-line"
            tone="info"
            value={String(stats?.events_24h ?? 0)}
            sub={t("dashboard.activity.for_24h")}
          />
          <SummaryCell
            label={t("dashboard.activity.stat_server_actions")}
            icon="ri-server-line"
            tone="info"
            value={String(stats?.server_actions ?? 0)}
            sub={t("dashboard.activity.for_24h")}
          />
          <SummaryCell
            label={t("dashboard.activity.stat_logins")}
            icon="ri-login-circle-line"
            tone="ok"
            value={String(stats?.logins ?? 0)}
            sub={t("dashboard.activity.for_24h")}
          />
          <SummaryCell
            label={t("dashboard.activity.stat_errors")}
            icon="ri-error-warning-line"
            tone={stats?.errors_7d ? "bad" : "ok"}
            value={String(stats?.errors_7d ?? 0)}
            sub={t("monitoring.kpi.for_7d")}
            color={stats?.errors_7d ? MON.bad : MON.fg}
          />
        </div>

        <div className={cn("overflow-clip", MON_CARD)}>
          <div className="flex flex-wrap items-center gap-2 border-b border-[var(--vx-border)] px-3 py-2.5">
            <div className="flex h-9 min-w-[200px] flex-1 items-center gap-2 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-bg)] px-2.5 sm:max-w-[300px]">
              <i className="ri-search-line text-[14px] text-[var(--vx-muted)]" />
              <input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={t("dashboard.activity.search_placeholder")}
                className="min-w-0 flex-1 bg-transparent text-[12px] text-[var(--vx-fg)] outline-none placeholder:text-[var(--vx-faint)]"
              />
              {query && (
                <button
                  type="button"
                  onClick={() => setQuery("")}
                  aria-label={t("dashboard.activity.clear")}
                >
                  <i className="ri-close-line text-[14px] text-[var(--vx-muted)] hover:text-[var(--vx-fg)]" />
                </button>
              )}
            </div>

            <div className="flex max-w-full gap-1 overflow-x-auto rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-bg)] p-[3px]">
              {CATEGORIES.map((c) => (
                <Chip
                  key={c.key}
                  active={category === c.key}
                  className="shrink-0"
                  onClick={() => {
                    setCategory(c.key);
                    setLimit(PAGE_SIZE);
                    setOpened(null);
                  }}
                >
                  {t(c.labelKey)}
                </Chip>
              ))}
            </div>

            <div className="flex gap-1 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-bg)] p-[3px] sm:ml-auto">
              {RANGES.map((r) => (
                <Chip
                  key={r.value}
                  active={range === r.value}
                  className="shrink-0"
                  onClick={() => {
                    setRange(r.value);
                    setLimit(PAGE_SIZE);
                    setOpened(null);
                  }}
                >
                  {t(r.labelKey)}
                </Chip>
              ))}
            </div>
          </div>

          {activity.isLoading ? (
            <div className="p-3">
              <Skeleton className="h-80 w-full rounded-[10px]" />
            </div>
          ) : activity.isError ? (
            <ListEmpty
              icon="ri-error-warning-line"
              title={t("dashboard.activity.load_failed_title")}
              text={t("dashboard.activity.load_failed_text")}
            />
          ) : rows.length === 0 ? (
            <ListEmpty
              icon="ri-file-list-3-line"
              title={t("dashboard.activity.empty_title")}
              text={t("dashboard.activity.empty_text")}
            />
          ) : (
            <>
              {groups.map((group) => (
                <section key={group.key}>
                  <header className="sticky top-0 z-[1] flex items-center justify-between gap-2 border-y border-[var(--vx-border)] bg-[var(--vx-card-2)] px-3 py-2">
                    <span className="text-[12px] font-semibold">{group.title}</span>
                    <span className="font-mono text-[11px] text-[var(--vx-muted)]">
                      {group.items.length}{" "}
                      {plural(
                        group.items.length,
                        t("dashboard.activity.records_one"),
                        t("dashboard.activity.records_few"),
                        t("dashboard.activity.records_many")
                      )}
                    </span>
                  </header>
                  {group.items.map((row) => (
                    <ActivityRow
                      key={row.id}
                      row={row}
                      open={opened === row.id}
                      onToggle={() => setOpened((id) => (id === row.id ? null : row.id))}
                    />
                  ))}
                </section>
              ))}

              <div className="flex flex-wrap items-center justify-between gap-2.5 px-3 py-3">
                <span className="text-[12px] text-[var(--vx-muted)]">
                  {t("dashboard.activity.shown", {
                    shown: rows.length,
                    total: activity.data?.total ?? rows.length,
                    range: t(
                      RANGES.find((r) => r.value === range)?.labelKey ??
                        "dashboard.activity.range_24h"
                    ),
                  })}
                </span>
                {activity.data?.has_more && (
                  <button
                    type="button"
                    onClick={() => setLimit((n) => n + PAGE_SIZE)}
                    disabled={activity.isFetching}
                    className={cn(MON_BTN, "h-8")}
                  >
                    {activity.isFetching
                      ? t("common.loading")
                      : t("dashboard.activity.show_more", { count: PAGE_SIZE })}
                  </button>
                )}
              </div>
            </>
          )}
        </div>
      </div>
    </PageShell>
  );
}

function SummaryCell({
  label,
  icon,
  tone,
  value,
  sub,
  color = MON.fg,
}: {
  label: string;
  icon: string;
  tone: Tone;
  value: string;
  sub: string;
  color?: string;
}) {
  return (
    <div className="flex items-center gap-2.5 bg-[var(--vx-card)] px-3.5 py-3">
      <EventIcon icon={icon} tone={tone} size={32} />
      <div className="min-w-0">
        <div className="truncate text-[11px] tracking-[0.02em] text-[var(--vx-muted)]">{label}</div>
        <div className="mt-[1px] flex items-baseline gap-1.5">
          <span className="text-[20px] font-bold tracking-[-0.02em]" style={{ color }}>
            {value}
          </span>
          <span className="text-[11px] text-[var(--vx-faint)]">{sub}</span>
        </div>
      </div>
    </div>
  );
}

function ActivityRow({
  row,
  open,
  onToggle,
}: {
  row: ActivityEntry;
  open: boolean;
  onToggle: () => void;
}) {
  const { icon, tone } = actionIcon(row);
  const title = row.description || row.action;
  const resource = shortResource(row.resource);
  const meta = metaPairs(row.meta);

  return (
    <div className="border-b border-[var(--vx-divider)] last:border-b-0">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        className={cn(
          "flex w-full items-center gap-2.5 px-3 py-2.5 text-left transition-colors hover:bg-[var(--vx-card-2)]",
          open && "bg-[var(--vx-card-2)]"
        )}
      >
        <EventIcon icon={icon} tone={tone} size={30} />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-[13px] text-[var(--vx-fg)]">{title}</span>
          <span className="mt-[2px] flex items-center gap-1.5 font-mono text-[11px] text-[var(--vx-faint)]">
            <span className="truncate">{row.action}</span>
            {resource && (
              <>
                <span aria-hidden>·</span>
                <span className="truncate">{resource}</span>
              </>
            )}
          </span>
        </span>
        <span className="shrink-0 text-right">
          <span className="block font-mono text-[12px] text-[var(--vx-dim)]">
            {formatTime(row.created_at)}
          </span>
          <span className="block text-[11px] text-[var(--vx-faint)]">
            {timeAgo(row.created_at)}
          </span>
        </span>
        <i
          className={cn(
            "ri-arrow-down-s-line shrink-0 text-[16px] text-[var(--vx-faint)] transition-transform",
            open && "rotate-180"
          )}
        />
      </button>

      {open && (
        <dl className="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-x-5 gap-y-2.5 border-t border-[var(--vx-divider)] bg-[var(--vx-bg)] px-3 py-3">
          <DetailItem label={t("dashboard.activity.detail_action")} value={row.action} mono />
          <DetailItem label={t("dashboard.activity.detail_category")} value={row.category} />
          <DetailItem
            label={t("dashboard.activity.detail_time")}
            value={formatDateTime(row.created_at)}
          />
          {row.resource && (
            <DetailItem
              label={t("dashboard.activity.detail_resource")}
              value={row.resource}
              mono
            />
          )}
          {row.user_email && (
            <DetailItem label={t("dashboard.activity.detail_user")} value={row.user_email} mono />
          )}
          {row.ip && <DetailItem label={t("dashboard.activity.detail_ip")} value={row.ip} mono />}
          {meta.map(([key, value]) => (
            <DetailItem key={key} label={key} value={value} mono />
          ))}
        </dl>
      )}
    </div>
  );
}

function DetailItem({
  label,
  value,
  mono = false,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="min-w-0">
      <dt className="text-[11px] text-[var(--vx-faint)]">{label}</dt>
      <dd
        className={cn(
          "mt-[2px] break-all text-[12px] text-[var(--vx-dim)]",
          mono && "font-mono"
        )}
      >
        {value}
      </dd>
    </div>
  );
}

function useDebounced<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(id);
  }, [value, delay]);
  return debounced;
}
