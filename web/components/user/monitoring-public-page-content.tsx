"use client";

import { useMemo } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Skeleton } from "@/components/ui/skeleton";
import { GameIcon } from "@/components/user/monitoring-page-content";
import { IncidentsTab } from "@/components/user/monitoring-server-page-content";
import {
  MON,
  MON_BTN,
  MON_CARD,
  MON_INNER,
  StatusPill,
  areaPath,
  initials,
  normalizeIncidents,
  normalizePlayers,
  normalizeServerRow,
  normalizeSettings,
  normalizeUptimeDays,
  polyPoints,
  secondsToHms,
  uptimeText,
} from "@/components/user/monitoring/shared";
import {
  fetchPublicMonitoringServer,
  fetchPublicMonitoringStats,
  voteForMonitoringServer,
} from "@/lib/api";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

export function MonitoringPublicPageContent() {
  // Вложенные помощники зовут t() напрямую, здесь хук нужен ради подписки:
  // без него страница не перерисуется при смене языка.
  useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();

  const page = useQuery({
    queryKey: queryKeys.monitoringPublic(id ?? ""),
    queryFn: () => fetchPublicMonitoringServer(id!),
    enabled: !!id,
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
  });

  const stats = useQuery({
    queryKey: queryKeys.monitoringPublicStats(id ?? "", 1),
    queryFn: () => fetchPublicMonitoringStats(id!, 1),
    enabled: !!id && page.data?.show_chart === true,
  });

  const vote = useMutation({
    mutationFn: () => voteForMonitoringServer(id!),
    onSuccess: () => {
      toast.success(t("monitoring.vote.thanks"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.monitoringPublic(id ?? "") });
    },
    onError: () => toast.error(t("monitoring.vote.already")),
  });

  const chart = useMemo(() => {
    const series = stats.data?.series;
    const values = (series?.online ?? []).map((v) => Number(v ?? 0));
    const labels = series?.labels ?? [];
    const ticks =
      labels.length <= 6
        ? labels
        : labels.filter((_, i) => i % Math.floor(labels.length / 6) === 0).slice(0, 7);
    return { values, ticks, cap: series?.max_cap ?? 0 };
  }, [stats.data]);

  if (page.isLoading) {
    return (
      <PublicShell>
        <Skeleton className="h-56 w-full rounded-[16px]" />
        <Skeleton className="h-72 w-full rounded-[14px]" />
      </PublicShell>
    );
  }

  if (!page.data) {
    return (
      <PublicShell>
        <div className={cn("mx-auto max-w-[520px] px-8 py-10 text-center", MON_CARD)}>
          <div className="mx-auto mb-[18px] flex h-14 w-14 items-center justify-center rounded-[14px] bg-[var(--vx-inset)]">
            <i className="ri-eye-off-line text-2xl text-[var(--vx-muted)]" />
          </div>
          <h1 className="text-[18px] font-semibold">
            {t("monitoring.public.unavailable_title")}
          </h1>
          <p className="mt-2 text-[13px] text-[var(--vx-muted)]">
            {t("monitoring.public.unavailable_text")}
          </p>
        </div>
      </PublicShell>
    );
  }

  const s = normalizeServerRow(page.data.server);
  const settings = normalizeSettings(page.data.settings);
  const players = normalizePlayers(page.data.players);
  const peak = Number(page.data.peak_24h) || 0;
  const top = Math.max(chart.cap || 0, ...chart.values, 1);

  return (
    <PublicShell>
      <div className={cn("overflow-hidden rounded-[16px]", MON_CARD)}>
        <div className="flex flex-wrap items-center gap-6 bg-[var(--vx-elevated)] p-7">
          <GameIcon gameId={s.game_id} className="h-18 w-18 rounded-[16px] p-3" />
          <div className="min-w-[240px] flex-1">
            <div className="flex flex-wrap items-center gap-2.5">
              <h1 className="text-[28px] font-bold tracking-[-0.02em]">{s.name}</h1>
              <StatusPill status={s.status} />
            </div>
            {s.description && (
              <p className="mt-2 max-w-[560px] text-[13px] text-pretty text-[var(--vx-muted)]">
                {s.description}
              </p>
            )}
            {s.tags.length > 0 && (
              <div className="mt-3.5 flex flex-wrap gap-1.5">
                {s.tags.map((tag) => (
                  <span
                    key={tag}
                    className="rounded-full border border-[var(--vx-border)] bg-[var(--vx-inset)] px-2.5 py-1 text-[11px] text-[var(--vx-dim)]"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            )}
          </div>

          <div className="flex min-w-[232px] flex-col gap-2.5">
            <div className={cn("px-4 py-3.5 text-center", MON_INNER)}>
              <div className="text-[11px] uppercase tracking-[0.06em] text-[var(--vx-muted)]">
                {t("monitoring.public.online_now")}
              </div>
              <div className="mt-1.5 font-mono text-[30px] font-medium">
                {s.online} / {s.slots}
              </div>
            </div>

            {s.ip ? (
              <button
                type="button"
                onClick={() => {
                  void navigator.clipboard?.writeText(s.ip);
                  toast.success(t("monitoring.address_copied"));
                }}
                className="flex items-center justify-between gap-2 rounded-[12px] bg-[var(--vx-fg-strong)] px-3.5 py-3 font-mono text-[13px] font-medium text-[var(--vx-on-fill)] transition-colors hover:bg-white"
              >
                {s.ip}
                <i className="ri-file-copy-line text-[15px]" />
              </button>
            ) : null}

            <button
              type="button"
              onClick={() => vote.mutate()}
              disabled={vote.isPending}
              className={cn(MON_BTN, "justify-center")}
            >
              <i className="ri-thumb-up-line text-[15px]" />
              {t("monitoring.public.vote", { count: settings.votes })}
            </button>

            {(s.discord || s.website) && (
              <div className="flex gap-2">
                {s.discord && (
                  <ExternalLink href={s.discord} icon="ri-discord-line" label="Discord" />
                )}
                {s.website && (
                  <ExternalLink
                    href={s.website}
                    icon="ri-global-line"
                    label={t("monitoring.public.website")}
                  />
                )}
              </div>
            )}
          </div>
        </div>

        <div className="grid grid-cols-[repeat(auto-fit,minmax(140px,1fr))] gap-px border-t border-[var(--vx-border)] bg-[var(--vx-border)]">
          <PublicStat label={t("monitoring.col.uptime")} value={uptimeText(s.uptime)} />
          <PublicStat
            label={t("monitoring.col.ping")}
            value={s.ping ? t("monitoring.unit.ms", { value: s.ping }) : "—"}
          />
          {settings.show_version && (
            <PublicStat label={t("monitoring.stat.version")} value={s.version || "—"} />
          )}
          {settings.show_version && (
            <PublicStat label={t("monitoring.stat.map")} value={s.map || "—"} />
          )}
          <PublicStat label={t("monitoring.stat.region")} value={s.region || "—"} />
          <PublicStat label={t("monitoring.stat.peak_24h")} value={String(peak)} />
        </div>
      </div>

      <div
        className={cn(
          "grid gap-4",
          settings.show_chart && settings.show_players
            ? "lg:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)]"
            : "grid-cols-1"
        )}
      >
        {settings.show_chart && (
          <div className={cn("p-[18px]", MON_CARD)}>
            <div className="flex items-baseline gap-2.5">
              <span className="text-[14px] font-semibold">
                {t("monitoring.chart.online_24h")}
              </span>
              <span className="ml-auto text-[11px] text-[var(--vx-muted)]">
                {t("monitoring.chart.peak", { value: peak })}
              </span>
            </div>
            {stats.isLoading ? (
              <Skeleton className="mt-4 h-44 w-full" />
            ) : chart.values.length === 0 ? (
              <p className="py-14 text-center text-[12px] text-[var(--vx-muted)]">
                {t("monitoring.chart.collecting")}
              </p>
            ) : (
              <div className="mt-4">
                <svg
                  viewBox="0 0 600 160"
                  preserveAspectRatio="none"
                  className="block h-44 w-full"
                >
                  <line x1="0" y1="0.5" x2="600" y2="0.5" stroke="var(--vx-inset)" strokeWidth="1" />
                  <line x1="0" y1="80" x2="600" y2="80" stroke="var(--vx-inset)" strokeWidth="1" />
                  <line x1="0" y1="159.5" x2="600" y2="159.5" stroke="var(--vx-border)" strokeWidth="1" />
                  <path
                    d={areaPath(chart.values, 600, 160, top)}
                    fill="var(--vx-veil-strong)"
                  />
                  <polyline
                    points={polyPoints(chart.values, 600, 160, top)}
                    fill="none"
                    stroke="var(--vx-fg-strong)"
                    strokeWidth="1.75"
                    vectorEffect="non-scaling-stroke"
                  />
                </svg>
                <div className="mt-2 flex justify-between font-mono text-[10px] text-[var(--vx-faint)]">
                  {chart.ticks.map((l, i) => (
                    <span key={`${l}-${i}`}>{l}</span>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {settings.show_players && (
          <div className={cn("overflow-hidden", MON_CARD)}>
            <div className="flex items-center gap-2 border-b border-[var(--vx-border)] px-4 py-3.5">
              <span className="text-[14px] font-semibold">
                {t("monitoring.players.title")}
              </span>
              <span className="font-mono text-[12px] text-[var(--vx-muted)]">
                {s.online} / {s.slots}
              </span>
            </div>
            <div className="max-h-[236px] overflow-y-auto">
              {players.length === 0 ? (
                <p className="px-4 py-8 text-center text-[12px] text-[var(--vx-muted)]">
                  {t("monitoring.players.empty_public")}
                </p>
              ) : (
                players.map((p) => (
                  <div
                    key={p.name}
                    className="flex items-center gap-2.5 border-b border-[var(--vx-divider)] px-4 py-2.5"
                  >
                    <span className="flex h-6 w-6 items-center justify-center rounded-[6px] bg-[var(--vx-tint)] text-[10px] font-semibold text-[var(--vx-dim)]">
                      {initials(p.name)}
                    </span>
                    <span className="min-w-0 flex-1 truncate text-[13px]">{p.name}</span>
                    <span className="font-mono text-[11px] text-[var(--vx-muted)]">
                      {p.duration_sec ? secondsToHms(p.duration_sec) : "—"}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>
        )}
      </div>

      {settings.show_incidents && (
        <IncidentsTab
          uptime={s.uptime}
          days={normalizeUptimeDays(page.data.uptime_days)}
          incidents={normalizeIncidents(page.data.incidents)}
        />
      )}

      <div className={cn("flex flex-wrap items-center gap-4 p-[18px]", MON_CARD)}>
        <div className="min-w-[220px] flex-1">
          <div className="text-[14px] font-semibold">{t("monitoring.banner.title")}</div>
          <p className="mt-1 text-[12px] text-[var(--vx-muted)]">
            {t("monitoring.banner.hint")}
          </p>
        </div>
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={`${page.data.banner_url}/560x95.png`}
          alt={t("monitoring.banner.alt", { name: s.name })}
          className="h-auto max-w-full rounded-[8px] border border-[var(--vx-border)]"
        />
        <button
          type="button"
          className={MON_BTN}
          onClick={() => {
            void navigator.clipboard?.writeText(
              `<a href="${page.data!.public_url}"><img src="${page.data!.banner_url}/560x95.png" alt="${s.name}" /></a>`
            );
            toast.success(t("monitoring.banner.copied"));
          }}
        >
          {t("monitoring.banner.get_code")}
        </button>
      </div>
    </PublicShell>
  );
}

function PublicShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-[var(--vx-bg)] px-4 py-6 text-[var(--vx-fg)]">
      <div className="flex w-full flex-col gap-4">
        <Link href="/monitoring" className="text-[12px] text-[var(--vx-muted)] hover:text-white">
          {t("monitoring.back")}
        </Link>
        {children}
      </div>
    </div>
  );
}

function PublicStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-[var(--vx-card)] px-4 py-3.5">
      <div className="text-[11px] uppercase tracking-[0.06em] text-[var(--vx-muted)]">{label}</div>
      <div className="mt-1.5 font-mono text-[16px] font-medium">{value}</div>
    </div>
  );
}

function ExternalLink({ href, icon, label }: { href: string; icon: string; label: string }) {
  const url = /^https?:\/\//i.test(href) ? href : `https://${href}`;
  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer nofollow"
      className="flex flex-1 items-center justify-center gap-1.5 rounded-[10px] border border-[var(--vx-border)] bg-[var(--vx-bg)] py-2.5 text-[12px] transition-colors hover:bg-[var(--vx-tint)]"
      style={{ color: MON.fg }}
    >
      <i className={cn(icon, "text-[14px]")} />
      {label}
    </a>
  );
}
