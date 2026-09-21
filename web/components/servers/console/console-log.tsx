"use client";

import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { ArrowDown, Search } from "lucide-react";

import { VX_INPUT, VX_FAINT, VX_MUTED } from "@/components/vx/panel-ui";
import { useT } from "@/hooks/use-translations";
import type { ConsoleLine, LineKind } from "@/lib/console/parse";
import { cn } from "@/lib/utils";

export type LogFilter = "all" | "chat" | "players" | "problems";

const RENDER_LIMIT = 900;

const TONE: Record<LineKind, string> = {
  plain: "text-[var(--vx-dim)]",
  info: "text-[var(--vx-dim)]",
  warn: "text-[var(--vx-warn)]",
  error: "text-[var(--vx-danger)]",
  chat: "text-[var(--vx-fg)]",
  join: "text-[var(--vx-info)]",
  leave: "text-[var(--vx-muted)]",
  ready: "text-[var(--vx-info)]",
  stop: "text-[var(--vx-warn)]",
  map: "text-[var(--vx-violet)]",
  input: "text-[var(--vx-fg-strong)]",
  reply: "text-[var(--vx-fg)]",
  notice: "text-[var(--vx-warn)]",
};

const NOTICES = new Set([
  "waiting_start",
  "node_offline",
  "no_permission",
  "stream_unavailable",
  "input_stopped",
  "input_no_stdin",
  "input_failed",
]);

function noticeText(t: ReturnType<typeof useT>, line: ConsoleLine): string {
  if (NOTICES.has(line.value)) {
    return t(`servers.console.notice_${line.value}`, { detail: line.text });
  }
  return line.text || line.value;
}

function keep(line: ConsoleLine, filter: LogFilter): boolean {
  switch (filter) {
    case "chat":
      return line.kind === "chat";
    case "players":
      return line.kind === "join" || line.kind === "leave";
    case "problems":
      return line.kind === "error" || line.kind === "warn";
    default:
      return true;
  }
}

export function ConsoleLogView({
  lines,
  filter,
  onFilter,
  query,
  onQuery,
  counts,
  className,
}: {
  lines: ConsoleLine[];
  filter: LogFilter;
  onFilter: (next: LogFilter) => void;
  query: string;
  onQuery: (next: string) => void;
  counts: Record<LogFilter, number>;
  className?: string;
}) {
  const t = useT();
  const boxRef = useRef<HTMLDivElement>(null);
  const [stuck, setStuck] = useState(true);

  const needle = query.trim().toLowerCase();
  const shown = useMemo(() => {
    const out: ConsoleLine[] = [];
    for (let i = lines.length - 1; i >= 0; i--) {
      const line = lines[i];
      if (!keep(line, filter)) continue;
      if (needle && !line.raw.toLowerCase().includes(needle)) continue;
      out.push(line);
      if (out.length >= RENDER_LIMIT) break;
    }
    return out.reverse();
  }, [lines, filter, needle]);

  useLayoutEffect(() => {
    const box = boxRef.current;
    if (box && stuck) box.scrollTop = box.scrollHeight;
  }, [shown, stuck]);

  useEffect(() => {
    const box = boxRef.current;
    if (!box) return;
    const onScroll = () => {
      const gap = box.scrollHeight - box.scrollTop - box.clientHeight;
      setStuck(gap < 40);
    };
    box.addEventListener("scroll", onScroll, { passive: true });
    return () => box.removeEventListener("scroll", onScroll);
  }, []);

  const filters: LogFilter[] = ["all", "chat", "players", "problems"];

  return (
    <div className={cn("flex min-h-0 flex-col gap-2", className)}>
      <div className="flex flex-wrap items-center gap-2">
        <div className="flex flex-wrap gap-1">
          {filters.map((item) => (
            <button
              key={item}
              type="button"
              onClick={() => onFilter(item)}
              className={cn(
                "rounded-[8px] border px-2.5 py-1 text-[12px] transition-colors",
                filter === item
                  ? "border-[var(--vx-border-strong)] bg-[var(--vx-inset)] text-[var(--vx-fg)]"
                  : "border-[var(--vx-border-2)] text-[var(--vx-muted)] hover:border-[var(--vx-border-hover)]"
              )}
            >
              {t(`servers.console.filter_${item}`)}
              {counts[item] > 0 && (
                <span className={cn("ms-1.5 font-mono text-[10.5px]", VX_FAINT)}>{counts[item]}</span>
              )}
            </button>
          ))}
        </div>
        <div className="relative ms-auto min-w-[160px] flex-1 sm:max-w-[260px]">
          <Search className="absolute start-2.5 top-1/2 size-3.5 -translate-y-1/2 text-[var(--vx-ghost)]" />
          <input
            value={query}
            onChange={(event) => onQuery(event.target.value)}
            placeholder={t("servers.console.search")}
            className={cn(VX_INPUT, "ps-8")}
          />
        </div>
      </div>

      <div className="relative min-h-0 flex-1">
        <div
          ref={boxRef}
          className="vx-console-log h-full overflow-y-auto overscroll-contain rounded-[10px] border border-[var(--vx-inset)] bg-[var(--vx-code)] p-3 font-mono text-[12px] leading-[1.65]"
        >
          {shown.length === 0 ? (
            <div className={cn("py-6 text-center text-[12.5px]", VX_MUTED)}>
              {lines.length === 0 ? t("servers.console.waiting") : t("servers.console.nothing_found")}
            </div>
          ) : (
            shown.map((line) => (
              <div key={line.id} className="flex gap-2 whitespace-pre-wrap break-words">
                {line.time && (
                  <span className={cn("shrink-0 tabular-nums", VX_FAINT)}>{line.time}</span>
                )}
                <span className={cn("min-w-0 flex-1", TONE[line.kind])}>
                  {line.kind === "chat" && line.player ? (
                    <>
                      <span className="text-[var(--vx-info)]">{line.player}</span>
                      <span className={VX_FAINT}>: </span>
                      {line.text}
                    </>
                  ) : line.kind === "join" || line.kind === "leave" ? (
                    <>
                      <span className="font-semibold">{line.player}</span>{" "}
                      {t(`servers.console.event_${line.kind}`)}
                    </>
                  ) : line.kind === "input" ? (
                    <>
                      <span className="text-[var(--vx-info)]">› </span>
                      {line.text}
                    </>
                  ) : line.kind === "reply" ? (
                    <>
                      <span className={cn("me-2 inline-block w-9 text-[10.5px] uppercase", VX_FAINT)}>
                        {line.value}
                      </span>
                      {line.text}
                    </>
                  ) : line.kind === "notice" ? (
                    <span className="italic">— {noticeText(t, line)}</span>
                  ) : (
                    line.text
                  )}
                </span>
              </div>
            ))
          )}
        </div>

        {!stuck && (
          <button
            type="button"
            onClick={() => {
              const box = boxRef.current;
              if (box) box.scrollTop = box.scrollHeight;
              setStuck(true);
            }}
            className="absolute bottom-3 end-3 inline-flex items-center gap-1.5 rounded-full border border-[var(--vx-border-strong)] bg-[var(--vx-inset)] px-3 py-1.5 text-[12px] text-[var(--vx-fg)] shadow-lg"
          >
            <ArrowDown className="size-3.5" />
            {t("servers.console.to_bottom")}
          </button>
        )}
      </div>
    </div>
  );
}

export function countLines(lines: ConsoleLine[]): Record<LogFilter, number> {
  const counts: Record<LogFilter, number> = { all: lines.length, chat: 0, players: 0, problems: 0 };
  for (const line of lines) {
    if (line.kind === "chat") counts.chat += 1;
    if (line.kind === "join" || line.kind === "leave") counts.players += 1;
    if (line.kind === "error" || line.kind === "warn") counts.problems += 1;
  }
  return counts;
}
