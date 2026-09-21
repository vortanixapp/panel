"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { Btn, Panel, VX_FAINT, VX_INPUT, VX_MUTED } from "@/components/vx/panel-ui";
import type { AgentTabProps } from "@/components/admin/agents/agent-shell";
import { apiErrorCode, fetchAgentLogs, type AgentLogLine } from "@/lib/api";
import { agentErrorText, formatDateTime } from "@/lib/agents";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const MAX_LINES = 5000;
const FOLLOW_MS = 3000;

export function AgentLogsTab({ id }: AgentTabProps) {
  const t = useT();
  const [lines, setLines] = useState<AgentLogLine[]>([]);
  const [source, setSource] = useState<"relay" | "ssh" | "">("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [follow, setFollow] = useState(true);
  const [search, setSearch] = useState("");
  const [older, setOlder] = useState(false);
  const cursor = useRef<string | undefined>(undefined);
  const first = useRef<string | undefined>(undefined);
  const box = useRef<HTMLDivElement>(null);
  const stick = useRef(true);

  const load = useCallback(async () => {
    try {
      const res = await fetchAgentLogs(id, { tail: 500 });
      const list = res.lines ?? (res.logs ?? []).map((text) => ({ text }));
      setLines(list);
      setSource(res.source);
      cursor.current = res.cursor;
      first.current = res.first;
      setError(res.error ?? "");
    } catch (err) {
      setError(agentErrorText(apiErrorCode(err), err instanceof Error ? err.message : ""));
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!follow || source !== "relay") return;
    const timer = setInterval(async () => {
      try {
        const res = await fetchAgentLogs(id, { cursor: cursor.current, tail: 1000 });
        const next = res.lines ?? [];
        if (res.cursor) cursor.current = res.cursor;
        if (next.length) setLines((prev) => [...prev, ...next].slice(-MAX_LINES));
      } catch {
        return;
      }
    }, FOLLOW_MS);
    return () => clearInterval(timer);
  }, [follow, id, source]);

  useEffect(() => {
    const el = box.current;
    if (el && stick.current) el.scrollTop = el.scrollHeight;
  }, [lines]);

  async function loadOlder() {
    if (!first.current) return;
    setOlder(true);
    try {
      const res = await fetchAgentLogs(id, { before: first.current, tail: 500 });
      const prev = res.lines ?? [];
      if (prev.length) {
        first.current = res.first ?? prev[0]?.ts;
        stick.current = false;
        setLines((cur) => [...prev, ...cur].slice(0, MAX_LINES));
      } else {
        toast.message(t("admin.agents.logs.no_older"));
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.error"));
    } finally {
      setOlder(false);
    }
  }

  const shown = useMemo(() => {
    const q = search.trim().toLowerCase();
    return q ? lines.filter((l) => l.text.toLowerCase().includes(q)) : lines;
  }, [lines, search]);

  const asText = () => lines.map((l) => (l.ts ? `${l.ts} ${l.text}` : l.text)).join("\n");

  async function copy() {
    try {
      await navigator.clipboard.writeText(asText());
      toast.success(t("admin.agents.logs.copied"));
    } catch {
      toast.error(t("common.copy_failed"));
    }
  }

  function download() {
    const blob = new Blob([asText()], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `vortanix-agent-${id.slice(0, 8)}.log`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <Panel
      title={
        <span className="inline-flex items-center gap-2">
          {t("admin.agents.tab.logs")}
          {source && (
            <span className={cn("rounded-full border border-[var(--vx-border-2)] px-2 py-px text-[10.5px] font-normal", VX_MUTED)}>
              {t(`admin.agents.logs.source.${source}`)}
            </span>
          )}
        </span>
      }
      aside={
        <div className="flex flex-wrap items-center gap-2">
          <input
            className={cn(VX_INPUT, "h-[30px] w-[180px]")}
            placeholder={t("admin.agents.logs.search")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          {source === "relay" && (
            <>
              <Btn size="sm" disabled={older || !first.current} onClick={() => void loadOlder()}>
                {t("admin.agents.logs.older")}
              </Btn>
              <Btn size="sm" onClick={() => setFollow((v) => !v)}>
                {follow ? t("admin.agents.logs.pause") : t("admin.agents.logs.follow")}
              </Btn>
            </>
          )}
          {source === "ssh" && (
            <Btn size="sm" onClick={() => void load()}>
              {t("common.refresh")}
            </Btn>
          )}
          <Btn size="sm" disabled={!lines.length} onClick={() => void copy()}>
            {t("admin.agents.logs.copy")}
          </Btn>
          <Btn size="sm" disabled={!lines.length} onClick={download}>
            {t("admin.agents.logs.download")}
          </Btn>
        </div>
      }
      flush
    >
      {error && <div className="px-[18px] pt-3 text-[12.5px] text-[var(--vx-danger)]">{error}</div>}
      <div
        ref={box}
        onScroll={(e) => {
          const el = e.currentTarget;
          stick.current = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
        }}
        className="m-3 h-[560px] overflow-auto rounded-[10px] border border-[var(--vx-inset)] bg-[var(--vx-code)] p-3 font-mono text-[11.5px] leading-[1.55]"
      >
        {loading ? (
          <div className={VX_FAINT}>{t("admin.agents.logs.loading")}</div>
        ) : shown.length === 0 ? (
          <div className={VX_FAINT}>{t("admin.agents.logs.empty")}</div>
        ) : (
          shown.map((l, i) => (
            <div key={i} className="whitespace-pre-wrap break-all">
              {l.ts && <span className={cn("mr-2 select-none", VX_FAINT)}>{formatDateTime(l.ts)}</span>}
              <span className={cn(/error|ошибк|failed|panic/i.test(l.text) && "text-[var(--vx-danger)]")}>{l.text}</span>
            </div>
          ))
        )}
      </div>
    </Panel>
  );
}
