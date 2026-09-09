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
    if (!termRef.current) return;

    const term = new Terminal({
      theme: {
        background: "#08090a",
        foreground: "#b9babc",
        cursor: "#f1f2ed",
        selectionBackground: "#232426",
      },
      fontSize: 12,
      fontFamily: "var(--font-dm-mono), ui-monospace, monospace",
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(termRef.current);
    fit.fit();
    term.writeln(t("servers.console.connecting"));
    statusRef.current?.("connecting");

    let closed = false;

    (async () => {
      try {
        const { ticket } = await fetchConsoleTicket(serverId);
        if (closed) return;
        const ws = new WebSocket(consoleWsUrl(ticket));
        wsRef.current = ws;

        ws.onopen = () => {
          term.writeln("\r\n[connected]\r\n");
          statusRef.current?.("connected");
        };
        ws.onmessage = (ev) => term.write(String(ev.data));
        ws.onclose = () => {
          term.writeln("\r\n[disconnected]\r\n");
          statusRef.current?.("disconnected");
        };
        ws.onerror = () => {
          term.writeln("\r\n[error]\r\n");
          statusRef.current?.("error");
        };

        term.onData((data) => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: "input", data }));
          }
        });
      } catch (e) {
        term.writeln(
          `\r\n${t("servers.console.error", {
            message: e instanceof Error ? e.message : "failed",
          })}\r\n`
        );
        statusRef.current?.("error");
      }
    })();

    const onResize = () => fit.fit();
    window.addEventListener("resize", onResize);

    return () => {
      closed = true;
      window.removeEventListener("resize", onResize);
      wsRef.current?.close();
      term.dispose();
    };
  }, [serverId]);

  return (
    <div
      ref={termRef}
      className={cn(
        "overflow-hidden rounded-[10px] border border-[var(--vx-inset)] bg-[var(--vx-code)] p-3.5",
        className ?? "h-[360px]"
      )}
    />
  );
}
