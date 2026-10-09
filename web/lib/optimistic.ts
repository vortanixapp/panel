import type { QueryClient, QueryKey } from "@tanstack/react-query";
import type { DashboardData, DashboardServer } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

type Snapshot = [QueryKey, unknown][];

export type ServerPatch = Partial<DashboardServer> & Record<string, unknown>;

export const POWER_PATCH: Record<string, ServerPatch> = {
  start: { status: "starting", runtime_status: "starting" },
  restart: { status: "starting", runtime_status: "starting" },
  stop: { status: "stopping", runtime_status: "stopping" },
  kill: { status: "stopping", runtime_status: "stopping" },
};

function patchList<T extends { id: string }>(list: T[] | undefined, id: string, patch: ServerPatch) {
  if (!Array.isArray(list)) return list;
  return list.map((item) => (item.id === id ? { ...item, ...patch } : item));
}

function patchOne<T extends { id?: string }>(item: T | undefined, patch: ServerPatch) {
  if (!item || typeof item !== "object" || Array.isArray(item)) return item;
  return { ...item, ...patch };
}

export async function applyServerPatch(
  qc: QueryClient,
  id: string,
  patch: ServerPatch
): Promise<() => void> {
  const keys: QueryKey[] = [
    queryKeys.servers,
    queryKeys.adminServers,
    queryKeys.server(id),
    queryKeys.serverDetail(id),
    queryKeys.serverStatus(id),
    ["dashboard"],
    ["my-servers"],
  ];
  await Promise.all(keys.map((queryKey) => qc.cancelQueries({ queryKey, exact: true })));

  const snapshots: Snapshot = keys.map((queryKey) => [queryKey, qc.getQueryData(queryKey)]);

  qc.setQueryData<{ id: string }[]>(queryKeys.servers, (old) => patchList(old, id, patch));
  qc.setQueryData<{ id: string }[]>(queryKeys.adminServers, (old) => patchList(old, id, patch));
  qc.setQueryData(queryKeys.server(id), (old: { id?: string } | undefined) => patchOne(old, patch));
  qc.setQueryData(queryKeys.serverDetail(id), (old: { id?: string } | undefined) => patchOne(old, patch));
  qc.setQueryData(queryKeys.serverStatus(id), (old: Record<string, unknown> | undefined) =>
    patchOne(old, statusFields(patch))
  );
  qc.setQueryData<DashboardData>(["dashboard"], (old) =>
    old && Array.isArray(old.recent_servers)
      ? { ...old, recent_servers: patchList(old.recent_servers, id, patch) ?? old.recent_servers }
      : old
  );
  qc.setQueryData<{ servers: { id: string }[] }>(["my-servers"], (old) =>
    old && Array.isArray(old.servers) ? { ...old, servers: patchList(old.servers, id, patch) ?? old.servers } : old
  );

  return () => {
    for (const [queryKey, data] of snapshots) qc.setQueryData(queryKey, data);
  };
}

function statusFields(patch: ServerPatch): ServerPatch {
  const out: ServerPatch = {};
  if ("status" in patch) out.status = patch.status;
  if ("runtime_status" in patch) out.runtime_status = patch.runtime_status;
  return out;
}
