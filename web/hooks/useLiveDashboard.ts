"use client";

import { useEffect, useRef } from "react";
import { CONSOLE_URL, hasSession } from "@/lib/api";

export type TenantEvent = {
  type: string;
  server_id?: string;
  node_id?: string;
  status?: string;
  metrics?: Record<string, number>;
  percent?: number;
  message?: string;
};

export function useLiveDashboard(onEvent: (ev: TenantEvent) => void) {
  const cb = useRef(onEvent);
  cb.current = onEvent;

  useEffect(() => {
    if (!hasSession()) return;

    const ws = new WebSocket(`${CONSOLE_URL}/v1/dashboard/stream`);

    ws.onmessage = (msg) => {
      try {
        cb.current(JSON.parse(msg.data) as TenantEvent);
      } catch {}
    };

    return () => ws.close();
  }, []);
}
