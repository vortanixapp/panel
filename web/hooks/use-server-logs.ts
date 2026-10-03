"use client";

import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";

import { fetchServerLogs } from "@/lib/api";
import { pollMs } from "@/lib/public-settings";
import { mergeLogs, type ServerLogStore } from "@/features/servers/log-utils";

export function useServerLogs(id: string, tail: number, live: boolean) {
  const queryClient = useQueryClient();
  const queryKey = ["server-logs", id, tail] as const;
  return useQuery({
    queryKey,
    queryFn: async () => {
      const prev = queryClient.getQueryData<ServerLogStore>(queryKey);
      const response = await fetchServerLogs(id, tail, prev?.lastTime || undefined);
      return mergeLogs(prev, response, tail);
    },
    enabled: !!id,
    placeholderData: keepPreviousData,
    refetchInterval: live ? pollMs(5_000) : false,
  });
}
