"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { consoleWsUrl, fetchConsoleTicket } from "@/lib/api";
import type { ConsoleProfile } from "@/lib/api";
import { compileProfile, parseLine, type ConsoleLine } from "@/lib/console/parse";

export type ConsoleStatus = "connecting" | "connected" | "disconnected" | "error";

const RETRY_STEPS = [1000, 2000, 5000, 10000, 20000];
const BUFFER_LIMIT = 4000;
const FLUSH_MS = 120;

export type ConsoleStream = {
  status: ConsoleStatus;
  lines: ConsoleLine[];
  seen: string[];
  ready: boolean;
  map: string;
  send: (command: string) => boolean;
  clear: () => void;
};

export function useConsoleStream(serverId: string, profile: ConsoleProfile | undefined): ConsoleStream {
  const compiled = useMemo(() => compileProfile(profile), [profile]);
  const compiledRef = useRef(compiled);
  compiledRef.current = compiled;

  const [status, setStatus] = useState<ConsoleStatus>("connecting");
  const [lines, setLines] = useState<ConsoleLine[]>([]);
  const [seen, setSeen] = useState<string[]>([]);
  const [ready, setReady] = useState(false);
  const [map, setMap] = useState("");

  const wsRef = useRef<WebSocket | null>(null);
  const counter = useRef(0);
  const pending = useRef<ConsoleLine[]>([]);
  const tail = useRef("");
  const flushTimer = useRef(0);

  const send = useCallback((command: string) => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return false;
    ws.send(JSON.stringify({ type: "input", data: `${command}\r` }));
    return true;
  }, []);

  const clear = useCallback(() => {
    pending.current = [];
    setLines([]);
  }, []);

  useEffect(() => {
    let closed = false;
    let attempt = 0;
    let retry = 0;

    const flush = () => {
      flushTimer.current = 0;
      const batch = pending.current;
      if (batch.length === 0) return;
      pending.current = [];

      setLines((prev) => {
        const next = prev.concat(batch);
        return next.length > BUFFER_LIMIT ? next.slice(next.length - BUFFER_LIMIT) : next;
      });

      const joined: string[] = [];
      const left: string[] = [];
      let sawReady = false;
      let sawStop = false;
      let nextMap = "";
      for (const line of batch) {
        if (line.kind === "join" && line.player) joined.push(line.player);
        if (line.kind === "leave" && line.player) left.push(line.player);
        if (line.kind === "ready") sawReady = true;
        if (line.kind === "stop") sawStop = true;
        if (line.kind === "map" && line.value) nextMap = line.value;
      }
      if (joined.length || left.length) {
        setSeen((prev) => {
          const set = new Set(prev);
          for (const name of joined) set.add(name);
          for (const name of left) set.delete(name);
          return Array.from(set).sort((a, b) => a.localeCompare(b));
        });
      }
      if (sawStop) {
        setReady(false);
        setSeen([]);
      } else if (sawReady) {
        setReady(true);
      }
      if (nextMap) setMap(nextMap);
    };

    const queue = (raw: string) => {
      counter.current += 1;
      pending.current.push(parseLine(compiledRef.current, raw, counter.current, Date.now()));
      if (pending.current.length > 400) {
        pending.current = pending.current.slice(pending.current.length - 400);
      }
      if (!flushTimer.current) {
        flushTimer.current = window.setTimeout(flush, FLUSH_MS);
      }
    };

    const feed = (chunk: string) => {
      const merged = tail.current + chunk.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
      const parts = merged.split("\n");
      tail.current = parts.pop() ?? "";
      for (const part of parts) queue(part);
      if (tail.current.length > 8000) {
        queue(tail.current);
        tail.current = "";
      }
    };

    const scheduleRetry = () => {
      if (closed) return;
      const wait = RETRY_STEPS[Math.min(attempt, RETRY_STEPS.length - 1)];
      attempt += 1;
      retry = window.setTimeout(connect, wait);
    };

    const connect = async () => {
      if (closed) return;
      setStatus("connecting");
      let ticket = "";
      try {
        ticket = (await fetchConsoleTicket(serverId)).ticket;
      } catch {
        if (closed) return;
        setStatus("error");
        scheduleRetry();
        return;
      }
      if (closed) return;

      const ws = new WebSocket(consoleWsUrl(ticket));
      wsRef.current = ws;
      let openedAt = 0;

      ws.onopen = () => {
        openedAt = Date.now();
        setStatus("connected");
      };
      ws.onmessage = (event) => {
        if (closed) return;
        feed(String(event.data));
      };
      ws.onclose = () => {
        if (closed || wsRef.current !== ws) return;
        wsRef.current = null;
        if (openedAt && Date.now() - openedAt > 10000) attempt = 0;
        setStatus("disconnected");
        scheduleRetry();
      };
      ws.onerror = () => setStatus("error");
    };

    void connect();

    return () => {
      closed = true;
      window.clearTimeout(retry);
      window.clearTimeout(flushTimer.current);
      flushTimer.current = 0;
      pending.current = [];
      tail.current = "";
      const ws = wsRef.current;
      wsRef.current = null;
      ws?.close();
    };
  }, [serverId]);

  return { status, lines, seen, ready, map, send, clear };
}
