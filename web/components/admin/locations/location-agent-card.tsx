"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { ArrowRight } from "lucide-react";
import { LabelChips, StatePill } from "@/components/admin/agents/agents-table";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Bar } from "@/components/vx/panel-ui";
import { fetchAgentView } from "@/lib/api";
import { diskUsedPct, formatAgo, formatDuration, pct } from "@/lib/agents";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

function Meter({ label, value }: { label: string; value: number | null }) {
  const warn = value != null && value > 85;
  return (
    <div className="grid gap-1.5">
      <div className="flex justify-between gap-2 text-[12px]">
        <span className="text-muted-foreground">{label}</span>
        <span className={cn("font-mono tabular-nums", warn && "text-amber-500")}>
          {value == null ? "—" : `${Math.round(value)}%`}
        </span>
      </div>
      <Bar pct={value ?? 0} barClassName={warn ? "bg-[var(--vx-warn)]" : undefined} />
    </div>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-4 border-b py-2 text-[13px] last:border-0">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate text-right font-mono">{children}</span>
    </div>
  );
}

export function LocationAgentCard({ locationId }: { locationId: string }) {
  const t = useT();
  const query = useQuery({
    queryKey: queryKeys.agent(locationId),
    queryFn: () => fetchAgentView(locationId),
    enabled: !!locationId,
    refetchInterval: 15000,
    retry: false,
  });
  const agent = query.data?.agent_view;
  const href = `/admin/daemons/${locationId}`;
  const facts = agent?.facts;
  const os = [facts?.os, facts?.kernel].filter(Boolean).join(" · ") || agent?.platform || "—";

  return (
    <section className="flex flex-col gap-4 rounded-2xl border bg-card px-5 py-5 sm:px-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-[15px] leading-none font-semibold">{t("admin.location.agent.title")}</h2>
        <Button asChild variant="outline" size="sm" className="h-8 text-xs">
          <Link href={href}>
            {t("admin.agents.menu.open")}
            <ArrowRight className="size-3.5" />
          </Link>
        </Button>
      </div>
      {query.isLoading ? (
        <Skeleton className="h-40 w-full rounded-xl" />
      ) : !agent ? (
        <p className="text-xs text-muted-foreground">{t("admin.location.agent.unavailable")}</p>
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-1.5">
            <StatePill state={agent.state} />
            <LabelChips labels={agent.labels} />
          </div>
          {agent.state === "never_connected" ? (
            <p className="text-xs text-muted-foreground">
              {t("admin.location.agent.never")}{" "}
              <Link href={`/admin/locations/${locationId}/setup`} className="underline underline-offset-4">
                {t("admin.locations.ssh_setup")}
              </Link>
            </p>
          ) : (
            <div className="grid grid-cols-3 gap-4">
              <Meter label="CPU" value={pct(agent.resources.cpu_percent)} />
              <Meter label="RAM" value={pct(agent.resources.ram_percent)} />
              <Meter label={t("admin.agents.disk_short")} value={diskUsedPct(agent)} />
            </div>
          )}
          <div className="flex flex-col">
            <Row label={t("admin.agents.col.version")}>
              {agent.version || "—"}
              {agent.outdated && agent.target_version ? ` → ${agent.target_version}` : ""}
            </Row>
            <Row label={t("admin.agents.col.uptime")}>{formatDuration(agent.uptime_sec)}</Row>
            <Row label={t("admin.location.agent.os")}>{os}</Row>
            <Row label={t("admin.agents.col.servers")}>
              {agent.servers.running} / {agent.servers.total}
            </Row>
            <Row label={t("admin.agents.col.contact")}>{formatAgo(agent.last_seen)}</Row>
          </div>
        </>
      )}
    </section>
  );
}
