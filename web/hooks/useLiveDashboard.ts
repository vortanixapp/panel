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

export function useLiveDashboard(onEvent: (ev: TenantEvent) => void, enabled: boolean) {
  const cb = useRef(onEvent);
  cb.current = onEvent;

  useEffect(() => {
    if (!enabled || !hasSession()) return;

    const ws = new WebSocket(`${CONSOLE_URL}/v1/dashboard/stream`);

    ws.onmessage = (msg) => {
      try {
        cb.current(JSON.parse(msg.data) as TenantEvent);
      } catch {}
    };

    return () => {
      ws.onmessage = null;
      if (ws.readyState === WebSocket.CONNECTING) {
        ws.onopen = () => ws.close();
        return;
      }
      ws.close();
    };
  }, [enabled]);
}
