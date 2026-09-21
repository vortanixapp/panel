"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { consoleWsUrl, fetchConsoleTicket } from "@/lib/api";
import type { ConsoleProfile } from "@/lib/api";
import {
  compileProfile,
  decodeFrame,
  parseLine,
  syntheticLine,
  type CompiledProfile,
  type ConsoleLine,
} from "@/lib/console/parse";

export type ConsoleStatus = "connecting" | "connected" | "disconnected" | "error";

const RETRY_STEPS = [1000, 2000, 5000, 10000, 20000];
const BUFFER_LIMIT = 4000;
const PENDING_LIMIT = 600;
const FLUSH_MS = 120;

export type ConsoleStream = {
  status: ConsoleStatus;
  lines: ConsoleLine[];
  seen: string[];
  ready: boolean;
  map: string;
  send: (command: string) => boolean;
};

type Derived = { seen: string[]; ready: boolean; map: string };

function derive(lines: ConsoleLine[]): Derived {
  const online = new Set<string>();
  let ready = false;
  let map = "";
  for (const line of lines) {
    switch (line.kind) {
      case "join":
        if (line.player) online.add(line.player);
        break;
      case "leave":
        if (line.player) online.delete(line.player);
        break;
      case "ready":
        ready = true;
        break;
      case "stop":
        ready = false;
        online.clear();
        break;
      case "notice":
        if (line.value === "waiting_start") {
          ready = false;
          online.clear();
        }
        break;
      case "map":
        if (line.value) map = line.value;
        break;
    }
  }
  return {
    seen: Array.from(online).sort((a, b) => a.localeCompare(b)),
    ready,
    map,
  };
}

function reparse(profile: CompiledProfile, line: ConsoleLine): ConsoleLine {
  if (line.kind === "input" || line.kind === "reply" || line.kind === "notice") return line;
  return parseLine(profile, line.raw, line.id, line.at);
}

function release(ws: WebSocket) {
  ws.onmessage = null;
  ws.onclose = null;
  ws.onerror = null;
  if (ws.readyState === WebSocket.CONNECTING) {
    ws.onopen = () => ws.close();
    return;
  }
  ws.onopen = null;
  ws.close();
}

export function useConsoleStream(serverId: string, profile: ConsoleProfile | undefined): ConsoleStream {
  const compiled = useMemo(() => compileProfile(profile), [profile]);
  const compiledRef = useRef(compiled);
  compiledRef.current = compiled;

  const [status, setStatus] = useState<ConsoleStatus>("connecting");
  const [lines, setLines] = useState<ConsoleLine[]>([]);

  const wsRef = useRef<WebSocket | null>(null);
  const counter = useRef(0);
  const pending = useRef<ConsoleLine[]>([]);
  const replace = useRef(false);
  const tail = useRef("");
  const flushTimer = useRef(0);
  const schedule = useRef<() => void>(() => {});

  useEffect(() => {
    pending.current = pending.current.map((line) => reparse(compiled, line));
    setLines((prev) => (prev.length === 0 ? prev : prev.map((line) => reparse(compiled, line))));
  }, [compiled]);

  const push = useCallback((line: ConsoleLine) => {
    pending.current.push(line);
    if (pending.current.length > PENDING_LIMIT) {
      pending.current = pending.current.slice(pending.current.length - PENDING_LIMIT);
    }
    schedule.current();
  }, []);

  const nextId = useCallback(() => {
    counter.current += 1;
    return counter.current;
  }, []);

  const send = useCallback(
    (command: string) => {
      const ws = wsRef.current;
      if (!ws || ws.readyState !== WebSocket.OPEN) return false;
      ws.send(JSON.stringify({ type: "input", data: command }));
      push(syntheticLine("input", command, "", nextId(), Date.now()));
      return true;
    },
    [push, nextId]
  );

  useEffect(() => {
    let closed = false;
    let attempt = 0;
    let retry = 0;
    let lastNotice = "";

    const flush = () => {
      flushTimer.current = 0;
      const batch = pending.current;
      const fresh = replace.current;
      if (batch.length === 0 && !fresh) return;
      pending.current = [];
      replace.current = false;
      setLines((prev) => {
        const next = fresh ? batch : prev.concat(batch);
        return next.length > BUFFER_LIMIT ? next.slice(next.length - BUFFER_LIMIT) : next;
      });
    };

    schedule.current = () => {
      if (!flushTimer.current) {
        flushTimer.current = window.setTimeout(flush, FLUSH_MS);
      }
    };

    const queue = (raw: string) => {
      lastNotice = "";
      push(parseLine(compiledRef.current, raw, nextId(), Date.now()));
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

    const notice = (code: string, detail: string) => {
      const key = `${code}|${detail}`;
      if (key === lastNotice) return;
      lastNotice = key;
      push(syntheticLine("notice", detail, code, nextId(), Date.now()));
    };

    const reply = (via: string, data: string) => {
      lastNotice = "";
      const rows = data.replace(/\r\n/g, "\n").split("\n");
      rows.forEach((row, index) => {
        push(syntheticLine("reply", row, index === 0 ? via : "", nextId(), Date.now()));
      });
    };

    const receive = (raw: string) => {
      const frame = decodeFrame(raw);
      if (!frame) {
        feed(raw);
        return;
      }
      if (frame.type === "output") {
        feed(frame.data);
      } else if (frame.type === "reply") {
        reply(frame.code, frame.data);
      } else {
        notice(frame.code, frame.data);
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
        tail.current = "";
        if (counter.current > 0) {
          replace.current = true;
          pending.current = [];
        }
        setStatus("connected");
      };
      ws.onmessage = (event) => {
        if (closed) return;
        receive(String(event.data));
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
      schedule.current = () => {};
      pending.current = [];
      replace.current = false;
      tail.current = "";
      const ws = wsRef.current;
      wsRef.current = null;
      if (ws) release(ws);
    };
  }, [serverId, push, nextId]);

  const derived = useMemo(() => derive(lines), [lines]);

  return { status, lines, seen: derived.seen, ready: derived.ready, map: derived.map, send };
}
