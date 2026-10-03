"use client";

import { memo, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { useParams } from "next/navigation";
import { toast } from "sonner";
import {
  Btn,
  EmptyState,
  Panel,
  Toggle,
  VX_FAINT,
  VX_INPUT,
  VX_MUTED,
  VX_SELECT,
  VX_CODE,
} from "@/components/vx/panel-ui";
import { Skeleton } from "@/components/ui/skeleton";
import { useServerLogs } from "@/hooks/use-server-logs";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";
import {
  downloadTextFile,
  parseLogLines,
  splitByQuery,
  type LogLevel,
  type LogLine,
} from "@/features/servers/log-utils";

type LevelFilter = "all" | "warn" | "error";

function formatLogTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleTimeString(localeTag(), { hour12: false });
}

const TAIL_OPTIONS = [200, 500, 1000];
const BOTTOM_GAP = 24;

const LEVEL_TEXT: Record<LogLevel, string> = {
  error: "text-[var(--vx-danger)]",
  warn: "text-[var(--vx-warn)]",
  info: "text-[var(--vx-dim)]",
};

const LEVEL_ROW: Record<LogLevel, string> = {
  error: "bg-[rgba(224,122,122,0.07)]",
  warn: "bg-[rgba(232,160,60,0.06)]",
  info: "",
};

type LogRowProps = {
  line: LogLine;
  wrap: boolean;
  showTime: boolean;
  query: string;
};

function sameLogRow(prev: LogRowProps, next: LogRowProps) {
  return (
    prev.wrap === next.wrap &&
    prev.showTime === next.showTime &&
    prev.query === next.query &&
    prev.line.n === next.line.n &&
    prev.line.text === next.line.text &&
    prev.line.level === next.line.level &&
    prev.line.time === next.line.time
  );
}

const LogRow = memo(function LogRow({
  line,
  wrap,
  showTime,
  query,
}: LogRowProps) {
  return (
    <div className={cn("flex gap-3 px-3.5", LEVEL_ROW[line.level], LEVEL_TEXT[line.level])}>
      <span className={cn("w-9 shrink-0 text-right tabular-nums select-none", VX_FAINT)}>{line.n}</span>
      {showTime && line.time && (
        <span className={cn("shrink-0 tabular-nums select-none", VX_FAINT)} title={line.time}>
          {formatLogTime(line.time)}
        </span>
      )}
      <span className={cn("min-w-0 flex-1", wrap ? "break-all whitespace-pre-wrap" : "whitespace-pre")}>
        {splitByQuery(line.text, query).map((chunk, idx) =>
          chunk.match ? (
            <mark key={idx} className="rounded-[3px] bg-[rgba(232,200,60,0.35)] text-inherit">
              {chunk.part}
            </mark>
          ) : (
            <span key={idx}>{chunk.part}</span>
          )
        )}
      </span>
    </div>
  );
}, sameLogRow);

export function ServerLogsTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const [tail, setTail] = useState(500);
  const [live, setLive] = useState(true);
  const [wrap, setWrap] = useState(true);
  const [showTime, setShowTime] = useState(false);
  const [level, setLevel] = useState<LevelFilter>("all");
  const [search, setSearch] = useState("");
  const [follow, setFollow] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);

  const { data, isLoading, isError, error, refetch, isFetching, dataUpdatedAt } = useServerLogs(
    id,
    tail,
    live
  );

  const all = useMemo(() => parseLogLines(data), [data]);
  const query = search.trim();

  const counts = useMemo(
    () => ({
      warn: all.filter((line) => line.level === "warn").length,
      error: all.filter((line) => line.level === "error").length,
    }),
    [all]
  );

  const shown = useMemo(() => {
    const needle = query.toLowerCase();
    return all.filter((line) => {
      if (level === "error" && line.level !== "error") return false;
      if (level === "warn" && line.level === "info") return false;
      return !needle || line.text.toLowerCase().includes(needle);
    });
  }, [all, level, query]);

  const scrollToEnd = () => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  };

  useLayoutEffect(() => {
    if (follow) scrollToEnd();
  }, [shown, follow, wrap]);

  useEffect(() => {
    setFollow(true);
  }, [id, level, query]);

  const onScroll = () => {
    const el = scrollRef.current;
    if (!el) return;
    const atEnd = el.scrollHeight - el.scrollTop - el.clientHeight < BOTTOM_GAP;
    setFollow((prev) => (prev === atEnd ? prev : atEnd));
  };

  const visibleText = () =>
    shown
      .map((line) => (showTime && line.time ? `${line.time} ${line.text}` : line.text))
      .join("\n");

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(visibleText());
      toast.success(t("common.copied"));
    } catch {
      toast.error(t("common.copy_failed"));
    }
  };

  const download = () => {
    const stamp = new Date().toISOString().replace(/[:.]/g, "-").slice(0, 19);
    downloadTextFile(`server-${id.slice(0, 8)}-${stamp}.log`, `${visibleText()}\n`);
  };

  if (isLoading) return <Skeleton className="h-[420px] w-full rounded-[14px]" />;

  const updated = dataUpdatedAt
    ? new Date(dataUpdatedAt).toLocaleTimeString(localeTag(), { hour12: false })
    : "";

  const levelButtons: { key: LevelFilter; label: string }[] = [
    { key: "all", label: t("servers.logs.level_all") },
    { key: "warn", label: `${t("servers.logs.level_warn")} (${counts.warn + counts.error})` },
    { key: "error", label: `${t("servers.logs.level_error")} (${counts.error})` },
  ];

  return (
    <Panel
      title={t("servers.logs.title")}
      aside={
        <div className="flex items-center gap-3">
          <label className={cn("flex items-center gap-2 text-[12px]", VX_MUTED)}>
            <Toggle label={t("servers.logs.live")} checked={live} onChange={setLive} />
            {t("servers.logs.live")}
          </label>
          <Btn size="sm" onClick={() => void refetch()} disabled={isFetching}>
            {isFetching ? t("common.updating") : t("common.refresh")}
          </Btn>
        </div>
      }
    >
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <input
          className={cn(VX_INPUT, "max-w-[280px] min-w-[160px] flex-1")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t("servers.logs.search")}
          spellCheck={false}
        />
        <div className="flex items-center gap-1">
          {levelButtons.map((item) => (
            <Btn
              key={item.key}
              size="sm"
              tone={level === item.key ? "primary" : "default"}
              onClick={() => setLevel(item.key)}
            >
              {item.label}
            </Btn>
          ))}
        </div>
        <select
          className={VX_SELECT}
          value={tail}
          onChange={(e) => setTail(Number(e.target.value))}
          aria-label={t("servers.logs.tail")}
        >
          {TAIL_OPTIONS.map((n) => (
            <option key={n} value={n}>
              {t("servers.logs.tail_option", { count: n })}
            </option>
          ))}
        </select>
        <label className={cn("flex items-center gap-2 text-[12px]", VX_MUTED)}>
          <Toggle label={t("servers.logs.wrap")} checked={wrap} onChange={setWrap} />
          {t("servers.logs.wrap")}
        </label>
        <label className={cn("flex items-center gap-2 text-[12px]", VX_MUTED)}>
          <Toggle label={t("servers.logs.show_time")} checked={showTime} onChange={setShowTime} />
          {t("servers.logs.show_time")}
        </label>
        <div className="ml-auto flex items-center gap-1">
          <Btn size="sm" onClick={() => void copy()} disabled={shown.length === 0}>
            {t("common.copy")}
          </Btn>
          <Btn size="sm" onClick={download} disabled={shown.length === 0}>
            {t("common.download")}
          </Btn>
        </div>
      </div>

      {isError && all.length === 0 ? (
        <EmptyState>
          <div className="flex flex-col items-center gap-3">
            <span>{error instanceof Error && error.message ? error.message : t("servers.logs.load_failed")}</span>
            <Btn size="sm" onClick={() => void refetch()}>
              {t("servers.logs.retry")}
            </Btn>
          </div>
        </EmptyState>
      ) : all.length === 0 ? (
        <EmptyState>{t("servers.logs.empty")}</EmptyState>
      ) : (
        <>
          <div className="relative">
            <div
              ref={scrollRef}
              onScroll={onScroll}
              className={cn(
                "h-[calc(100vh-420px)] min-h-[320px] overflow-auto rounded-[10px] py-2 font-mono text-[12px] leading-[1.6]",
                VX_CODE
              )}
            >
              {shown.length === 0 ? (
                <div className={cn("px-4 py-10 text-center text-[12.5px]", VX_FAINT)}>
                  {t("servers.logs.nothing_found")}
                </div>
              ) : (
                <div className={cn(!wrap && "w-max min-w-full")}>
                  {shown.map((line) => (
                    <LogRow key={line.n} line={line} wrap={wrap} showTime={showTime} query={query} />
                  ))}
                </div>
              )}
            </div>
            {!follow && shown.length > 0 && (
              <Btn
                size="sm"
                tone="primary"
                className="absolute right-4 bottom-3 shadow-lg"
                onClick={() => {
                  setFollow(true);
                  scrollToEnd();
                }}
              >
                {t("servers.logs.to_end")}
              </Btn>
            )}
          </div>
          <div className={cn("mt-2.5 flex flex-wrap justify-between gap-2 text-[11.5px]", VX_FAINT)}>
            <span>
              {t("servers.logs.lines_count", { shown: shown.length, total: all.length })}
            </span>
            {updated && <span>{t("servers.logs.updated", { time: updated })}</span>}
          </div>
        </>
      )}
    </Panel>
  );
}
