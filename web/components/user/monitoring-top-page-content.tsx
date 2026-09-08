"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { GameIcon } from "@/components/user/monitoring-page-content";
import {
  Chip,
  MON,
  MON_CARD,
  MON_ROW,
  MON_TH,
  Sparkline,
  uptimeText,
} from "@/components/user/monitoring/shared";
import { fetchMonitoringTop } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

// Значение фильтра, а не подпись: подпись переводится, а сравнение
// с именем игры должно остаться независимым от языка.
const ALL_GAMES = "all";

export function MonitoringTopPageContent() {
  const t = useT();
  const [game, setGame] = useState(ALL_GAMES);

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.monitoringTop(),
    queryFn: () => fetchMonitoringTop(),
    refetchInterval: 60_000,
  });

  const items = (data?.items ?? [])
    .map((t) => ({ ...t, spark: Array.isArray(t.spark) ? t.spark : [], tags: t.tags ?? [] }))
    .filter((t) => game === ALL_GAMES || t.game_name === game);

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-5">
        <div>
          <Link href="/monitoring" className="text-[12px] text-[var(--vx-muted)] hover:text-white">
            {t("monitoring.back")}
          </Link>
          <h1 className="mt-2 text-[26px] font-bold tracking-[-0.02em]">
            {t("monitoring.top.title")}
          </h1>
          <p className="mt-1 text-[13px] text-[var(--vx-muted)]">
            {t("monitoring.top.subtitle")}
          </p>
        </div>

        <div className={cn("overflow-hidden", MON_CARD)}>
          <div className="flex flex-wrap items-center gap-2 border-b border-[var(--vx-border)] px-4 py-3">
            <Chip active={game === ALL_GAMES} onClick={() => setGame(ALL_GAMES)}>
              {t("monitoring.top.all_games")}
            </Chip>
            {(data?.games ?? []).map((g) => (
              <Chip key={g} active={game === g} onClick={() => setGame(g)}>
                {g}
              </Chip>
            ))}
          </div>

          {isLoading ? (
            <div className="p-4">
              <Skeleton className="h-64 w-full rounded-[12px]" />
            </div>
          ) : isError ? (
            <p className="px-4 py-12 text-center text-[13px] text-[var(--vx-danger)]">
              {t("monitoring.top.load_failed")}
            </p>
          ) : items.length === 0 ? (
            <div className="px-4 py-12 text-center">
              <div className="text-[14px] font-semibold">
                {t("monitoring.top.empty_title")}
              </div>
              <p className="mt-1.5 text-[12px] text-[var(--vx-muted)]">
                {t("monitoring.top.empty_text")}
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full border-collapse text-left">
                <thead>
                  <tr className="bg-[var(--vx-card-2)]">
                    <th className={cn(MON_TH, "w-14 px-4")}>#</th>
                    <th className={MON_TH}>{t("monitoring.col.server")}</th>
                    <th className={MON_TH}>{t("monitoring.col.game")}</th>
                    <th className={MON_TH}>{t("monitoring.col.online")}</th>
                    <th className={MON_TH}>{t("monitoring.col.day")}</th>
                    <th className={MON_TH}>{t("monitoring.col.uptime")}</th>
                    <th className={MON_TH}>{t("monitoring.col.votes")}</th>
                    <th className={cn(MON_TH, "px-4 text-right")}>
                      {t("monitoring.col.address")}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {items.map((t) => (
                    <tr key={t.server_id} className={MON_ROW}>
                      <td className="px-4 py-3">
                        <span
                          className="inline-flex h-[26px] w-[26px] items-center justify-center rounded-[8px] font-mono text-[12px]"
                          style={{
                            background: t.rank <= 3 ? "var(--vx-tint)" : "transparent",
                            color: t.rank <= 3 ? MON.fg : MON.mut,
                          }}
                        >
                          {t.rank}
                        </span>
                      </td>
                      <td className="p-3">
                        <div className="flex items-center gap-2.5">
                          <GameIcon gameId={t.game_id} className="h-[26px] w-[26px] p-1" />
                          <div>
                            <Link
                              href={`/monitoring/public/${t.server_id}`}
                              className="block text-[13px] font-semibold hover:text-white"
                            >
                              {t.name}
                            </Link>
                            <div className="text-[11px] text-[var(--vx-muted)]">{t.tagline}</div>
                          </div>
                        </div>
                      </td>
                      <td className="p-3 text-[12px] text-[var(--vx-dim)]">{t.game_name}</td>
                      <td className="p-3 font-mono text-[13px]">
                        {t.online} / {t.slots}
                      </td>
                      <td className="p-3">
                        <Sparkline
                          values={t.spark}
                          cap={t.slots}
                          filled={false}
                          stroke={MON.mut}
                          className="h-[26px] w-[96px]"
                        />
                      </td>
                      <td className="p-3 font-mono text-[12px] text-[var(--vx-dim)]">
                        {uptimeText(t.uptime)}
                      </td>
                      <td className="p-3 font-mono text-[12px] text-[var(--vx-muted)]">{t.votes}</td>
                      <td className="px-4 py-3 text-right font-mono text-[11px] text-[var(--vx-muted)]">
                        {t.ip || "—"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </PageShell>
  );
}
