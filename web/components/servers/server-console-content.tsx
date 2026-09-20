"use client";

import { useMemo, useRef, useState } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { SendHorizontal } from "lucide-react";

import { ServerTabShell } from "@/components/servers/server-tab-shell";
import { ConsoleLogView, countLines, type LogFilter } from "@/components/servers/console/console-log";
import { QuickCommands } from "@/components/servers/console/console-commands";
import { PlayersCard, StateCard, type ConsolePlayer, type StateRow } from "@/components/servers/console/console-side";
import {
  useConsoleStream,
  type ConsoleStatus,
} from "@/components/servers/console/use-console-stream";
import { Btn, Panel, VX_FAINT, VX_INPUT_MONO, VX_MUTED } from "@/components/vx/panel-ui";
import { useT } from "@/hooks/use-translations";
import { fetchServerConsoleProfile, fetchServerStatus } from "@/lib/api";
import type { TranslateFn } from "@/lib/i18n";
import type { PanelVariant } from "@/lib/panel-paths";
import { cn } from "@/lib/utils";

const STATUS_DOT: Record<ConsoleStatus, string> = {
  connecting: "bg-[var(--vx-warn)]",
  connected: "bg-[var(--vx-info)]",
  disconnected: "bg-[var(--vx-faint)]",
  error: "bg-[var(--vx-danger)]",
};

function statusText(t: TranslateFn): Record<ConsoleStatus, string> {
  return {
    connecting: t("servers.console.status_connecting"),
    connected: t("servers.console.status_connected"),
    disconnected: t("servers.console.status_disconnected"),
    error: t("servers.console.status_error"),
  };
}

export function ServerConsolePanel() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const [command, setCommand] = useState("");
  const [filter, setFilter] = useState<LogFilter>("all");
  const [query, setQuery] = useState("");
  const history = useRef<string[]>([]);
  const cursor = useRef(-1);
  const inputRef = useRef<HTMLInputElement>(null);

  const { data: meta } = useQuery({
    queryKey: ["server-console-profile", id],
    queryFn: () => fetchServerConsoleProfile(id),
    enabled: !!id,
    staleTime: 10 * 60_000,
  });

  const { data: live } = useQuery({
    queryKey: ["server-console-status", id],
    queryFn: () => fetchServerStatus(id),
    enabled: !!id,
    refetchInterval: 15_000,
  });

  const stream = useConsoleStream(id, meta?.profile);
  const counts = useMemo(() => countLines(stream.lines), [stream.lines]);

  const players = useMemo<ConsolePlayer[]>(() => {
    const out = new Map<string, ConsolePlayer>();
    for (const item of live?.players_online ?? []) {
      const name = String(item?.name ?? "").trim();
      if (name) out.set(name, { name, live: true });
    }
    for (const name of stream.seen) {
      if (!out.has(name)) out.set(name, { name, live: false });
    }
    return Array.from(out.values());
  }, [live?.players_online, stream.seen]);

  const rows = useMemo<StateRow[]>(() => {
    const running = live?.runtime_status === "running" || live?.online;
    const state = running
      ? stream.ready
        ? t("servers.console.state_ready")
        : t("servers.console.state_running")
      : t("servers.console.state_stopped");
    return [
      { label: t("servers.console.state_server"), value: state },
      { label: t("servers.console.state_game"), value: meta?.title ?? "" },
      { label: t("servers.console.state_map"), value: stream.map || live?.current_map || "" },
      { label: t("servers.console.state_uptime"), value: live?.uptime ?? "" },
      {
        label: t("servers.console.state_slots"),
        value: live?.max_players ? String(live.max_players) : "",
      },
    ];
  }, [live, stream.ready, stream.map, meta?.title, t]);

  const connected = stream.status === "connected";
  const note = meta?.profile?.note ?? "";
  const commands = meta?.profile?.commands ?? [];
  const names = players.map((player) => player.name);

  function run(value: string) {
    const text = value.trim();
    if (!text) return;
    if (!stream.send(text)) {
      toast.error(t("servers.console.not_connected"));
      return;
    }
    history.current = [text, ...history.current.filter((item) => item !== text)].slice(0, 30);
    cursor.current = -1;
  }

  function submit() {
    run(command);
    setCommand("");
  }

  function recall(step: number) {
    const items = history.current;
    if (items.length === 0) return;
    const next = Math.min(items.length - 1, Math.max(-1, cursor.current + step));
    cursor.current = next;
    setCommand(next < 0 ? "" : items[next]);
  }

  return (
    <Panel
      title={t("server.tab.console")}
      aside={
        <span className={cn("inline-flex items-center gap-[7px] font-mono text-[11px]", VX_MUTED)}>
          <span className={cn("h-1.5 w-1.5 rounded-full", STATUS_DOT[stream.status])} />
          {statusText(t)[stream.status]}
          {meta && !meta.tailored && (
            <span className={VX_FAINT}>· {t("servers.console.generic")}</span>
          )}
        </span>
      }
    >
      <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_260px]">
        <div className="flex min-w-0 flex-col gap-3">
          <ConsoleLogView
            lines={stream.lines}
            filter={filter}
            onFilter={setFilter}
            query={query}
            onQuery={setQuery}
            counts={counts}
            className="h-[min(58vh,460px)]"
          />

          <div className="flex gap-2">
            <input
              ref={inputRef}
              className={cn(VX_INPUT_MONO, "h-9 flex-1")}
              value={command}
              onChange={(event) => setCommand(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter") {
                  event.preventDefault();
                  submit();
                  return;
                }
                if (event.key === "ArrowUp") {
                  event.preventDefault();
                  recall(1);
                  return;
                }
                if (event.key === "ArrowDown") {
                  event.preventDefault();
                  recall(-1);
                }
              }}
              placeholder={t("servers.console.command_placeholder")}
            />
            <Btn tone="primary" className="h-9 px-[18px]" onClick={submit} disabled={!connected}>
              <SendHorizontal className="size-3.5" />
              {t("common.send")}
            </Btn>
          </div>
          {note && <p className={cn("text-[11.5px] leading-snug", VX_FAINT)}>{note}</p>}
        </div>

        <aside className="grid gap-3 md:grid-cols-2 xl:grid-cols-1">
          <PlayersCard
            players={players}
            max={live?.max_players ?? 0}
            onPick={(name) => {
              setCommand((prev) => (prev.trim() ? `${prev.trim()} ${name}` : name));
              inputRef.current?.focus();
            }}
          />
          <StateCard rows={rows} note={meta && !meta.tailored ? t("servers.console.generic_hint") : undefined} />
          <QuickCommands
            commands={commands}
            players={names}
            disabled={!connected}
            onRun={run}
          />
        </aside>
      </div>
    </Panel>
  );
}

export function ServerConsoleContent({ variant = "user" }: { variant?: PanelVariant }) {
  return (
    <ServerTabShell variant={variant} activeTab="console">
      <ServerConsolePanel />
    </ServerTabShell>
  );
}
