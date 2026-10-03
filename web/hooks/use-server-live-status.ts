"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";

import { fetchServerStatus } from "@/lib/api";
import { pollMs } from "@/lib/public-settings";
import { queryKeys } from "@/lib/query-keys";

export type ServerLiveStatus = Awaited<ReturnType<typeof fetchServerStatus>>;

const TRANSITIONAL = ["starting", "stopping", "restarting"];
const PROVISIONING = ["pending", "installing", "provisioning", "reinstalling", "updating"];

function isRunning(status: ServerLiveStatus): boolean {
  return (status.runtime_status || "").toLowerCase() === "running" || status.online === true;
}

export function mergeLiveStatus(
  prev: ServerLiveStatus | undefined,
  next: ServerLiveStatus
): ServerLiveStatus {
  if (!prev || !isRunning(next) || !isRunning(prev)) return next;
  const unanswered = next.answered === false;
  const emptyMap = !next.current_map && !!prev.current_map;
  if (!unanswered && !emptyMap) return next;
  if (unanswered) {
    return {
      ...next,
      max_players: prev.max_players || next.max_players,
      online_players: prev.online_players,
      players_online: prev.players_online,
      current_map: prev.current_map,
    };
  }
  return { ...next, current_map: prev.current_map };
}

export function useServerLiveStatus(
  id: string,
  runtimeStatus?: string,
  provisioningStatus?: string
) {
  const queryClient = useQueryClient();
  return useQuery({
    queryKey: queryKeys.serverStatus(id),
    queryFn: async () => {
      const next = await fetchServerStatus(id);
      const prev = queryClient.getQueryData<ServerLiveStatus>(queryKeys.serverStatus(id));
      return mergeLiveStatus(prev, next);
    },
    enabled: !!id,
    refetchInterval: (query) => {
      const runtime = (query.state.data?.runtime_status || runtimeStatus || "").toLowerCase();
      const prov = (provisioningStatus || "").toLowerCase();
      const busy = TRANSITIONAL.includes(runtime) || PROVISIONING.includes(prov);
      return busy ? pollMs(3000) : pollMs(10000);
    },
  });
}
