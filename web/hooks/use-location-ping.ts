"use client";

import { useEffect, useMemo, useRef, useState } from "react";

export type PingTarget = {
  id: string;
  host: string;
  port: number;
};

export type PingState = Record<string, number | null>;

const ATTEMPTS = 3;
const TIMEOUT_MS = 2500;
const GAP_MS = 120;

function probeURL(host: string, port: number): string {
  const target = host.includes(":") ? `[${host}]` : host;
  return `wss://${target}:${port}`;
}

function measureOnce(host: string, port: number): Promise<number | null> {
  return new Promise((resolve) => {
    let socket: WebSocket;
    try {
      socket = new WebSocket(probeURL(host, port));
    } catch {
      resolve(null);
      return;
    }
    const started = performance.now();
    let done = false;
    const finish = (value: number | null) => {
      if (done) return;
      done = true;
      clearTimeout(timer);
      socket.onerror = null;
      socket.onclose = null;
      socket.onopen = null;
      try {
        socket.close();
      } catch {
        void 0;
      }
      resolve(value);
    };
    const timer = setTimeout(() => finish(null), TIMEOUT_MS);
    const elapsed = () => Math.max(1, Math.round((performance.now() - started) / 2));
    socket.onerror = () => finish(elapsed());
    socket.onclose = () => finish(elapsed());
    socket.onopen = () => finish(elapsed());
  });
}

function wait(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export function useLocationPing(targets: PingTarget[]): PingState {
  const [pings, setPings] = useState<PingState>({});
  const measured = useRef<Set<string>>(new Set());
  const signature = useMemo(
    () => targets.map((tg) => `${tg.id}@${tg.host}:${tg.port}`).join("|"),
    [targets]
  );
  const latest = useRef(targets);
  latest.current = targets;

  useEffect(() => {
    if (typeof window === "undefined" || !signature) return;
    let cancelled = false;

    const run = async () => {
      for (const target of latest.current) {
        if (cancelled) return;
        const key = `${target.id}@${target.host}:${target.port}`;
        if (measured.current.has(key)) continue;
        measured.current.add(key);
        let best: number | null = null;
        for (let attempt = 0; attempt < ATTEMPTS; attempt += 1) {
          const value = await measureOnce(target.host, target.port);
          if (cancelled) return;
          if (value !== null && (best === null || value < best)) best = value;
          if (attempt < ATTEMPTS - 1) await wait(GAP_MS);
        }
        if (cancelled) return;
        setPings((prev) => ({ ...prev, [target.id]: best }));
      }
    };

    void run();
    return () => {
      cancelled = true;
    };
  }, [signature]);

  return pings;
}
