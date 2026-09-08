"use client";

import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Btn, EmptyState, Panel, VX_CODE } from "@/components/vx/panel-ui";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchServerLogs } from "@/lib/api";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

export function ServerLogsTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  // Журнал подтягивается сам: раньше единственным способом увидеть новое была
  // кнопка «Обновить», и вкладка выглядела застывшей.
  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: ["server-logs", id],
    queryFn: () => fetchServerLogs(id),
    enabled: !!id,
    refetchInterval: 10_000,
  });

  if (isLoading) return <Skeleton className="h-[420px] w-full rounded-[14px]" />;

  const lines = data?.lines ?? [];

  return (
    <Panel
      title={t("servers.logs.title")}
      aside={
        <Btn size="sm" onClick={() => refetch()} disabled={isFetching}>
          {isFetching ? t("common.updating") : t("common.refresh")}
        </Btn>
      }
    >
      {lines.length === 0 ? (
        <EmptyState>{t("servers.logs.empty")}</EmptyState>
      ) : (
        <pre
          className={cn(
            "m-0 max-h-[460px] overflow-auto rounded-[10px] p-3.5 font-mono text-[12px] leading-[1.6] whitespace-pre-wrap text-[var(--vx-dim)]",
            VX_CODE
          )}
        >
          {lines.join("\n")}
        </pre>
      )}
    </Panel>
  );
}
