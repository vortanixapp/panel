"use client";

import { useState } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { AlertTriangle, CheckCircle2, Info, XCircle } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Btn, EmptyState, Panel, SubTabs, VX_FAINT, VX_MUTED } from "@/components/vx/panel-ui";
import type { AgentTabProps } from "@/components/admin/agents/agent-shell";
import { fetchAgentEvents, type AgentEvent } from "@/lib/api";
import { formatDateTime } from "@/lib/agents";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const GROUPS = ["", "connection", "updates", "maintenance", "problems"] as const;

const LEVEL_ICON = {
  info: <Info className="h-4 w-4 text-[var(--vx-info)]" />,
  success: <CheckCircle2 className="h-4 w-4 text-[var(--vx-ok)]" />,
  warn: <AlertTriangle className="h-4 w-4 text-[var(--vx-warn)]" />,
  error: <XCircle className="h-4 w-4 text-[var(--vx-danger)]" />,
};

function detail(ev: AgentEvent): string {
  const d = ev.data ?? {};
  const pick = (k: string) => (d[k] == null || d[k] === "" ? null : String(d[k]));
  return [
    pick("version") ?? pick("to"),
    pick("target"),
    pick("stage"),
    pick("method"),
    pick("reason"),
    pick("remote_addr"),
    pick("server_id") ? `server ${String(d.server_id).slice(0, 8)}` : null,
    pick("exit_code") != null ? `exit ${d.exit_code}` : null,
    pick("error"),
  ]
    .filter(Boolean)
    .join(" · ");
}

export function AgentEventsTab({ id }: AgentTabProps) {
  const t = useT();
  const [group, setGroup] = useState<string>("");
  const eventsQuery = useInfiniteQuery({
    queryKey: queryKeys.agentEvents(id, group),
    queryFn: ({ pageParam }) => fetchAgentEvents(id, group, pageParam || undefined),
    initialPageParam: 0,
    getNextPageParam: (last) => (last.has_more ? last.next_before : undefined),
    refetchInterval: 30000,
  });
  const events = eventsQuery.data?.pages.flatMap((p) => p.events) ?? [];

  return (
    <Panel
      title={t("admin.agents.tab.events")}
      aside={
        <SubTabs
          items={GROUPS.map((g) => ({ id: g || "all", title: t(`admin.agents.events.group.${g || "all"}`) }))}
          active={group || "all"}
          onSelect={(v) => setGroup(v === "all" ? "" : v)}
        />
      }
    >
      {eventsQuery.isLoading ? (
        <Skeleton className="h-[240px] w-full rounded-[10px]" />
      ) : events.length === 0 ? (
        <EmptyState>{t("admin.agents.events.empty")}</EmptyState>
      ) : (
        <div className="grid">
          {events.map((ev) => {
            const text = detail(ev);
            return (
              <div key={ev.id} className="flex items-start gap-3 border-b border-[var(--vx-elevated)] py-2.5 last:border-b-0">
                <span className="mt-0.5 shrink-0">{LEVEL_ICON[ev.level] ?? LEVEL_ICON.info}</span>
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-baseline justify-between gap-x-3">
                    <span className="text-[12.5px] font-medium">{t(`admin.agents.event.${ev.kind}`)}</span>
                    <span className={cn("font-mono text-[11px]", VX_FAINT)}>{formatDateTime(ev.created_at)}</span>
                  </div>
                  {(text || ev.actor) && (
                    <div className={cn("mt-0.5 text-[12px] break-words", VX_MUTED)}>
                      {text}
                      {ev.actor ? `${text ? " · " : ""}${ev.actor}` : ""}
                    </div>
                  )}
                </div>
              </div>
            );
          })}
          {eventsQuery.hasNextPage && (
            <div className="pt-3 text-center">
              <Btn size="sm" disabled={eventsQuery.isFetchingNextPage} onClick={() => void eventsQuery.fetchNextPage()}>
                {t("admin.agents.events.more")}
              </Btn>
            </div>
          )}
        </div>
      )}
    </Panel>
  );
}
