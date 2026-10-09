import type { QueryClient } from "@tanstack/react-query";
import type { DashboardData, MetricPoint, PlayersPush, ServerPush } from "@/lib/api";
import type { ServerLiveStatus } from "@/hooks/use-server-live-status";
import { queryKeys } from "@/lib/query-keys";

const MIN_POINTS = 60;

function appendPoint(points: MetricPoint[] | undefined, push: ServerPush) {
  if (!points) return points;
  const last = points[points.length - 1];
  if (last && last.ts >= push.ts) return points;
  const point: MetricPoint = {
    ts: push.ts,
    cpu_pct: push.cpu_pct,
    mem_used_mb: push.mem_used_mb,
    mem_limit_mb: push.mem_limit_mb,
  };
  return [...points, point].slice(-Math.max(points.length, MIN_POINTS));
}

export function applyServerPush(qc: QueryClient, push: ServerPush) {
  const id = push.server_id;
  if (!id) return;

  qc.setQueryData<MetricPoint[]>(queryKeys.metrics(id), (prev) => appendPoint(prev, push));
  qc.setQueryData<MetricPoint[]>(["dashboard-spark", id], (prev) => appendPoint(prev, push));

  const ram = push.mem_limit_mb > 0 ? (push.mem_used_mb / push.mem_limit_mb) * 100 : undefined;
  qc.setQueryData<DashboardData>(["dashboard"], (prev) => {
    if (!prev || !prev.recent_servers.some((server) => server.id === id)) return prev;
    return {
      ...prev,
      recent_servers: prev.recent_servers.map((server) =>
        server.id === id
          ? {
              ...server,
              cpu_percent: push.cpu_pct,
              ram_percent: ram ?? server.ram_percent,
              mem_used_mb: push.mem_used_mb,
              mem_limit_mb: push.mem_limit_mb,
            }
          : server
      ),
    };
  });
}

export function applyPlayersPush(qc: QueryClient, push: PlayersPush) {
  const id = push.server_id;
  if (!id) return;

  qc.setQueryData<ServerLiveStatus>(queryKeys.serverStatus(id), (prev) =>
    prev
      ? ({
          ...prev,
          answered: true,
          online_players: push.online_players,
          max_players: push.max_players || prev.max_players,
          players_online: push.players_online,
          current_map: push.current_map || prev.current_map,
        } as ServerLiveStatus)
      : prev
  );

  qc.setQueryData<DashboardData>(["dashboard"], (prev) => {
    if (!prev || !prev.recent_servers.some((server) => server.id === id)) return prev;
    return {
      ...prev,
      recent_servers: prev.recent_servers.map((server) =>
        server.id === id
          ? { ...server, online_players: push.online_players, max_players: push.max_players || server.max_players }
          : server
      ),
    };
  });
}
