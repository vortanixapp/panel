"use client";

import { useEffect, useMemo, useState } from "react";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import {
  CategoryBadge,
  Chip,
  EventIcon,
  ListEmpty,
  MON,
  MON_BTN,
  MON_BTN_PRIMARY,
  MON_CARD,
  StatCard,
  dayGroupTitle,
  dayKey,
  formatTime,
  plural,
  timeAgo,
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

// Списки читаются на уровне модуля, поэтому храним ключи: готовые подписи
// застыли бы на языке, который стоял в момент загрузки страницы.
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

function actionIcon(entry: ActivityEntry): { icon: string; tone: string } {
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

export function ActivityPageContent() {
  // Помощники модуля зовут t() напрямую, здесь хук нужен ради подписки:
  // без него страница не перерисуется при смене языка.
  useT();
  const [category, setCategory] = useState("all");
  const [range, setRange] = useState(1);
  const [query, setQuery] = useState("");
  const [limit, setLimit] = useState(PAGE_SIZE);
  const [exporting, setExporting] = useState(false);

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
      <div className="flex w-full flex-col gap-[18px]">
        <div className="flex flex-wrap items-end gap-4">
          <div className="min-w-[260px] flex-1">
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

        <div className="grid grid-cols-[repeat(auto-fit,minmax(190px,1fr))] gap-3">
          <StatCard
            label={t("dashboard.activity.stat_events")}
            icon="ri-file-list-3-line"
            value={String(stats?.events_24h ?? 0)}
            sub={t("dashboard.activity.stat_sub_total")}
          />
          <StatCard
            label={t("dashboard.activity.stat_server_actions")}
            icon="ri-server-line"
            value={String(stats?.server_actions ?? 0)}
            sub={t("dashboard.activity.for_24h")}
          />
          <StatCard
            label={t("dashboard.activity.stat_logins")}
            icon="ri-login-circle-line"
            value={String(stats?.logins ?? 0)}
            sub={t("dashboard.activity.for_24h")}
          />
          <StatCard
            label={t("dashboard.activity.stat_errors")}
            icon="ri-error-warning-line"
            value={String(stats?.errors_7d ?? 0)}
            sub={t("monitoring.kpi.for_7d")}
            color={stats?.errors_7d ? MON.warn : MON.ok}
          />
        </div>

        <div className={cn("overflow-hidden", MON_CARD)}>
          <div className="flex flex-wrap items-center gap-2.5 border-b border-[var(--vx-border)] px-4 py-3.5">
            <div className="flex h-[34px] w-[250px] items-center gap-2 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-bg)] px-2.5">
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
                  <i className="ri-close-line text-[14px] text-[var(--vx-muted)] hover:text-white" />
                </button>
              )}
            </div>

            <div className="flex gap-1 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-bg)] p-[3px]">
              {CATEGORIES.map((c) => (
                <Chip
                  key={c.key}
                  active={category === c.key}
                  onClick={() => {
                    setCategory(c.key);
                    setLimit(PAGE_SIZE);
                  }}
                >
                  {t(c.labelKey)}
                </Chip>
              ))}
            </div>

            <div className="ml-auto flex gap-1 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-bg)] p-[3px]">
              {RANGES.map((r) => (
                <Chip
                  key={r.value}
                  active={range === r.value}
                  onClick={() => {
                    setRange(r.value);
                    setLimit(PAGE_SIZE);
                  }}
                >
                  {t(r.labelKey)}
                </Chip>
              ))}
            </div>
          </div>

          {activity.isLoading ? (
            <div className="p-4">
              <Skeleton className="h-80 w-full rounded-[12px]" />
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
                <div key={group.key}>
                  <div className="flex items-center gap-2.5 border-b border-[var(--vx-border)] bg-[var(--vx-card-2)] px-4 py-2.5">
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
                  </div>
                  {group.items.map((row) => {
                    const { icon, tone } = actionIcon(row);
                    return (
                      <div
                        key={row.id}
                        className="flex items-start gap-3 border-b border-[var(--vx-divider)] px-4 py-3 transition-colors hover:bg-[var(--vx-card-2)]"
                      >
                        <EventIcon icon={icon} tone={tone} />
                        <div className="min-w-0 flex-1">
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="font-mono text-[13px] font-medium">{row.action}</span>
                            <CategoryBadge>{row.category}</CategoryBadge>
                          </div>
                          <div className="mt-[3px] text-[12px] text-[var(--vx-dim)]">
                            {row.description}
                          </div>
                          <div className="mt-1 flex flex-wrap items-center gap-2 font-mono text-[11px] text-[var(--vx-faint)]">
                            {[row.resource, row.user_email, row.ip]
                              .filter(Boolean)
                              .map((part, i, arr) => (
                                <span key={`${part}-${i}`} className="flex items-center gap-2">
                                  <span className="max-w-[280px] truncate">{part}</span>
                                  {i < arr.length - 1 && <span>·</span>}
                                </span>
                              ))}
                          </div>
                        </div>
                        <div className="shrink-0 text-right">
                          <div className="font-mono text-[12px] text-[var(--vx-dim)]">
                            {formatTime(row.created_at)}
                          </div>
                          <div className="mt-[3px] text-[11px] text-[var(--vx-faint)]">
                            {timeAgo(row.created_at)}
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              ))}

              <div className="flex items-center justify-between gap-3 px-4 py-3.5">
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
                    className="rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-bg)] px-3.5 py-2 text-[12px] transition-colors hover:bg-[var(--vx-tint)] disabled:opacity-55"
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

function useDebounced<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(id);
  }, [value, delay]);
  return debounced;
}
