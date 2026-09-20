"use client";

import { useEffect, useImperativeHandle, useRef, type Ref } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { consoleWsUrl, fetchConsoleTicket } from "@/lib/api";
import { t } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export type ConsoleStatus = "connecting" | "connected" | "disconnected" | "error";

export type ServerConsoleHandle = {
  send: (command: string) => boolean;
};

const RETRY_STEPS = [1000, 2000, 5000, 10000, 20000];

function consoleText(raw: string): string {
  if (!raw.startsWith('{"type":"error"')) return raw;
  try {
    const parsed = JSON.parse(raw) as { data?: string };
    return parsed.data ? `\r\n${parsed.data}\r\n` : raw;
  } catch {
    return raw;
  }
}

export function ServerConsole({
  serverId,
  className,
  onStatusChange,
  ref,
}: {
  serverId: string;
  className?: string;
  onStatusChange?: (status: ConsoleStatus) => void;
  ref?: Ref<ServerConsoleHandle>;
}) {
  const termRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const statusRef = useRef(onStatusChange);
  statusRef.current = onStatusChange;

  useImperativeHandle(ref, () => ({
    send(command: string) {
      const ws = wsRef.current;
      if (!ws || ws.readyState !== WebSocket.OPEN) return false;
      ws.send(JSON.stringify({ type: "input", data: `${command}\r` }));
      return true;
    },
  }));

  useEffect(() => {
    const host = termRef.current;
    if (!host) return;

    const term = new Terminal({
      theme: {
        background: "#08090a",
        foreground: "#b9babc",
        cursor: "#f1f2ed",
        selectionBackground: "#232426",
      },
      fontSize: 12,
      fontFamily: "var(--font-dm-mono), ui-monospace, monospace",
      scrollback: 5000,
      convertEol: true,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(host);

    const refit = () => {
      if (host.clientWidth < 2 || host.clientHeight < 2) return;
      try {
        fit.fit();
      } catch {
        return;
      }
    };
    refit();

    term.writeln(t("servers.console.connecting"));
    statusRef.current?.("connecting");

    let closed = false;
    let attempt = 0;
    let retry = 0;

    term.onData((data) => {
      const ws = wsRef.current;
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "input", data }));
      }
    });

    const scheduleRetry = () => {
      if (closed) return;
      const wait = RETRY_STEPS[Math.min(attempt, RETRY_STEPS.length - 1)];
      attempt += 1;
      retry = window.setTimeout(connect, wait);
    };

    const connect = async () => {
      if (closed) return;
      statusRef.current?.("connecting");
      let ticket = "";
      try {
        ticket = (await fetchConsoleTicket(serverId)).ticket;
      } catch (e) {
        if (closed) return;
        term.writeln(
          `\r\n${t("servers.console.error", {
            message: e instanceof Error ? e.message : "failed",
          })}\r\n`
        );
        statusRef.current?.("error");
        scheduleRetry();
        return;
      }
      if (closed) return;

      const ws = new WebSocket(consoleWsUrl(ticket));
      wsRef.current = ws;

      let openedAt = 0;
      ws.onopen = () => {
        openedAt = Date.now();
        term.writeln(`\r\n${t("servers.console.connected")}\r\n`);
        statusRef.current?.("connected");
        refit();
      };
      ws.onmessage = (ev) => {
        if (closed) return;
        const buffer = term.buffer.active;
        const atBottom = buffer.viewportY >= buffer.baseY;
        term.write(consoleText(String(ev.data)), () => {
          if (atBottom) term.scrollToBottom();
        });
      };
      ws.onclose = () => {
        if (closed || wsRef.current !== ws) return;
        wsRef.current = null;
        if (openedAt && Date.now() - openedAt > 10000) attempt = 0;
        term.writeln(`\r\n${t("servers.console.reconnecting")}\r\n`);
        statusRef.current?.("disconnected");
        scheduleRetry();
      };
      ws.onerror = () => {
        statusRef.current?.("error");
      };
    };

    void connect();

    const observer = new ResizeObserver(refit);
    observer.observe(host);
    window.addEventListener("orientationchange", refit);

    return () => {
      closed = true;
      window.clearTimeout(retry);
      observer.disconnect();
      window.removeEventListener("orientationchange", refit);
      const ws = wsRef.current;
      wsRef.current = null;
      ws?.close();
      term.dispose();
    };
  }, [serverId]);

  return (
    <div
      ref={termRef}
      className={cn(
        "vx-console touch-pan-y overflow-hidden rounded-[10px] border border-[var(--vx-inset)] bg-[var(--vx-code)] p-3.5",
        className ?? "h-[360px]"
      )}
    />
  );
}
