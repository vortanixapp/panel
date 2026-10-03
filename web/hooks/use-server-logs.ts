"use client";

import { useRef } from "react";
import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";

import { fetchServerLogs } from "@/lib/api";
import { pollMs } from "@/lib/public-settings";
import { mergeLogs, type ServerLogStore } from "@/features/servers/log-utils";

const POLL_STEPS_MS = [5_000, 5_000, 10_000, 15_000, 30_000];

export function useServerLogs(id: string, tail: number, live: boolean) {
  const queryClient = useQueryClient();
  const queryKey = ["server-logs", id, tail] as const;
  const quietPolls = useRef(0);
  return useQuery({
    queryKey,
    queryFn: async () => {
      const prev = queryClient.getQueryData<ServerLogStore>(queryKey);
      const response = await fetchServerLogs(id, tail, prev?.lastTime || undefined);
      const next = mergeLogs(prev, response, tail);
      quietPolls.current = prev && next === prev ? quietPolls.current + 1 : 0;
      return next;
    },
    enabled: !!id,
    placeholderData: keepPreviousData,
    refetchInterval: live
      ? () => pollMs(POLL_STEPS_MS[Math.min(quietPolls.current, POLL_STEPS_MS.length - 1)])
      : false,
  });
}
