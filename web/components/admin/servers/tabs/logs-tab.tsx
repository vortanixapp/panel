"use client";

import { useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Download } from "lucide-react";
import {
  Btn,
  EmptyState,
  Panel,
  SubTabs,
  Toggle,
  VX_CODE,
  VX_INPUT,
  VX_SELECT,
} from "@/components/vx/panel-ui";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchServerInstallLog, fetchServerLogs } from "@/lib/api";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const TAILS = [200, 1000, 5000];

export function AdminServerLogsTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();

  const [source, setSource] = useState<"container" | "install">("container");
  const [search, setSearch] = useState("");
  const [tail, setTail] = useState(200);
  const [follow, setFollow] = useState(true);

  const containerQuery = useQuery({
    queryKey: ["server-logs", id, tail],
    queryFn: () => fetchServerLogs(id, tail),
    enabled: !!id && source === "container",
    refetchInterval: follow && source === "container" ? 10_000 : false,
  });

  const installQuery = useQuery({
    queryKey: ["server-install-log", id],
    queryFn: () => fetchServerInstallLog(id),
    enabled: !!id && source === "install",
    refetchInterval: follow && source === "install" ? 10_000 : false,
  });

  const active = source === "container" ? containerQuery : installQuery;
  const rawLines = useMemo(() => {
    if (source === "container") return containerQuery.data?.lines ?? [];
    return installQuery.data?.lines ?? [];
  }, [source, containerQuery.data, installQuery.data]);

  const lines = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (!needle) return rawLines;
    return rawLines.filter((line) => line.toLowerCase().includes(needle));
  }, [rawLines, search]);

  function download() {
    const blob = new Blob([rawLines.join("\n")], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${id}-${source}.log`;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  }

  return (
    <Panel
      title={t("servers.logs.title")}
      aside={
        <div className="flex items-center gap-2">
          <Btn size="sm" onClick={download} disabled={rawLines.length === 0}>
            <Download className="mr-1.5 inline h-3.5 w-3.5" />
            {t("servers.admin.logs.download")}
          </Btn>
          <Btn size="sm" onClick={() => active.refetch()} disabled={active.isFetching}>
            {active.isFetching ? t("common.updating") : t("common.refresh")}
          </Btn>
        </div>
      }
    >
      <SubTabs
        items={[
          { id: "container", title: t("servers.admin.logs.container") },
          { id: "install", title: t("servers.admin.logs.install") },
        ]}
        active={source}
        onSelect={(next) => setSource(next as "container" | "install")}
      />

      <div className="my-4 flex flex-wrap items-center gap-3">
        <input
          className={cn(VX_INPUT, "h-[32px] w-[240px]")}
          placeholder={t("servers.admin.logs.search")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        {source === "container" && (
          <select
            className={cn(VX_SELECT, "h-[32px]")}
            value={tail}
            onChange={(e) => setTail(Number(e.target.value))}
          >
            {TAILS.map((n) => (
              <option key={n} value={n}>
                {t("servers.admin.logs.tail", { count: n })}
              </option>
            ))}
          </select>
        )}
        <label className="flex items-center gap-2 text-[12px] text-[var(--vx-muted)]">
          <Toggle
            checked={follow}
            onChange={setFollow}
            label={t("servers.admin.logs.follow")}
          />
          {t("servers.admin.logs.follow")}
        </label>
        <span className="text-[11.5px] text-[var(--vx-faint)]">
          {t("servers.admin.logs.shown", { shown: lines.length, total: rawLines.length })}
        </span>
      </div>

      {active.isLoading ? (
        <Skeleton className="h-[420px] w-full rounded-[10px]" />
      ) : lines.length === 0 ? (
        <EmptyState>
          {rawLines.length === 0
            ? t("servers.logs.empty")
            : t("servers.admin.logs.no_matches")}
        </EmptyState>
      ) : (
        <pre
          className={cn(
            "m-0 max-h-[520px] overflow-auto rounded-[10px] p-3.5 font-mono text-[12px] leading-[1.6] whitespace-pre-wrap text-[var(--vx-dim)]",
            VX_CODE
          )}
        >
          {lines.join("\n")}
        </pre>
      )}
    </Panel>
  );
}
