"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  apiFetch,
  createNode,
  createPanelUser,
  createServer,
  fetchAdminServers,
  deleteNode,
  deletePanelUser,
  deleteServer,
  changePassword,
  fetchMe,
  getAccessToken,
  getRefreshToken,
  fetchNodeInstall,
  fetchNodes,
  fetchServerDetail,
  fetchServerMetrics,
  fetchServers,
  fetchUsers,
  fetchAdminUsers,
  fetchAdminUser,
  fetchAdminUserCreateForm,
  updatePanelUser,
  togglePanelUserBlock,
  verifyPanelUserEmail,
  impersonatePanelUser,
  createAdminUserWallet,
  normalizeAdminUsersList,
  fetchAdminLocation,
  fetchAdminLocationEdit,
  fetchAdminLocationSetup,
  fetchAdminLocations,
  createAdminLocation,
  updateAdminLocation,
  toggleAdminLocation,
  deleteAdminLocation,
  runAdminLocationSetupStep,
  pullAdminLocationDaemon,
  refreshAdminLocationDaemon,
  restartAdminLocationDaemon,
  installAdminLocationDaemon,
  fetchAdminGames,
  fetchAdminGame,
  fetchAdminGameEdit,
  createAdminGame,
  updateAdminGame,
  toggleAdminGame,
  deleteAdminGame,
  createAdminGameVersion,
  deleteAdminGameVersion,
  powerServer,
  type Node,
  type DashboardServer,
  type Server,
  adminToggleServerBlock,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export function useMe() {
  return useQuery({
    queryKey: queryKeys.me,
    queryFn: fetchMe,
    enabled: typeof window !== "undefined" && !!(getAccessToken() || getRefreshToken()),
    retry: (failureCount, error) => {
      if (error instanceof Error && error.message === "Session expired") {
        return false;
      }
      return failureCount < 1;
    },
    staleTime: 60_000,
  });
}

export function useUsers() {
  return useQuery({
    queryKey: queryKeys.users,
    queryFn: fetchUsers,
  });
}

export function useCreateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      email: string;
      password: string;
      name?: string;
      password_confirmation?: string;
      role?: "user" | "admin" | "support";
    }) => createPanelUser(body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.users });
      void qc.invalidateQueries({ queryKey: ["admin-users"] });
    },
  });
}

export function useDeleteUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: deletePanelUser,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.users });
      void qc.invalidateQueries({ queryKey: ["admin-users"] });
    },
  });
}

export function useAdminUsers(page: number, q: string, role: string) {
  return useQuery({
    queryKey: queryKeys.adminUsers(page, q, role),
    queryFn: () => fetchAdminUsers({ page, q, role: role || undefined }),
    select: normalizeAdminUsersList,
  });
}

export function useAdminUser(id: string) {
  return useQuery({
    queryKey: queryKeys.adminUser(id),
    queryFn: () => fetchAdminUser(id),
    enabled: !!id,
  });
}

export function useAdminUserCreateForm() {
  return useQuery({
    queryKey: ["admin-user-create-form"],
    queryFn: fetchAdminUserCreateForm,
  });
}

export function useUpdateAdminUser(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: Record<string, unknown>) => updatePanelUser(id, body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.adminUser(id) });
      void qc.invalidateQueries({ queryKey: queryKeys.adminUserEdit(id) });
      void qc.invalidateQueries({ queryKey: ["admin-users"] });
      void qc.invalidateQueries({ queryKey: queryKeys.users });
    },
  });
}

export function useToggleAdminUserBlock(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => togglePanelUserBlock(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.adminUser(id) });
      void qc.invalidateQueries({ queryKey: ["admin-users"] });
      void qc.invalidateQueries({ queryKey: queryKeys.users });
    },
  });
}

export function useVerifyAdminUserEmail() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: verifyPanelUserEmail,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["admin-users"] });
      void qc.invalidateQueries({ queryKey: queryKeys.users });
    },
  });
}

export function useCreateAdminUserWallet(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (currency: string) => createAdminUserWallet(id, currency),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.adminUser(id) });
    },
  });
}

export function useImpersonateAdminUser() {
  return useMutation({
    mutationFn: impersonatePanelUser,
  });
}

export function useChangePassword() {
  return useMutation({
    mutationFn: ({
      currentPassword,
      newPassword,
    }: {
      currentPassword: string;
      newPassword: string;
    }) => changePassword(currentPassword, newPassword),
  });
}

export function useNodes(enabled = true) {
  return useQuery({
    queryKey: queryKeys.adminNodes,
    queryFn: fetchNodes,
    enabled,
  });
}

export function useServers() {
  return useQuery({
    queryKey: queryKeys.servers,
    queryFn: fetchServers,
  });
}

export function useAdminServers(enabled = true) {
  return useQuery({
    queryKey: queryKeys.adminServers,
    queryFn: async () => {
      const res = await fetchAdminServers();
      return res.servers;
    },
    enabled,
  });
}

export function useServer(id: string) {
  return useQuery({
    queryKey: queryKeys.server(id),
    queryFn: () => apiFetch<Server>(`/v1/servers/${id}`),
    enabled: !!id,
  });
}

const TRANSIENT_STATUSES = new Set([
  "starting",
  "stopping",
  "installing",
  "provisioning",
  "reinstalling",
  "updating",
  "pending",
]);

function isTransient(server?: DashboardServer): boolean {
  if (!server) return false;
  return (
    TRANSIENT_STATUSES.has((server.runtime_status || server.status || "").toLowerCase()) ||
    TRANSIENT_STATUSES.has((server.provisioning_status || "").toLowerCase())
  );
}

export function useServerDetail(id: string) {
  return useQuery({
    queryKey: queryKeys.serverDetail(id),
    queryFn: () => fetchServerDetail(id),
    enabled: !!id,
    refetchInterval: (query) => (isTransient(query.state.data) ? 2_000 : 10_000),
  });
}

export function useServerMetrics(id: string) {
  return useQuery({
    queryKey: queryKeys.metrics(id),
    queryFn: () => fetchServerMetrics(id),
    enabled: !!id,
    refetchInterval: 15_000,
  });
}

export function useCreateNode() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ name, fqdn }: { name: string; fqdn: string }) =>
      createNode(name, fqdn),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.adminLocations });
      void qc.invalidateQueries({ queryKey: queryKeys.nodes });
    },
  });
}

export function useDeleteNode() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteNode(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.adminLocations });
      void qc.invalidateQueries({ queryKey: queryKeys.nodes });
    },
  });
}

export function useNodeInstall() {
  return useMutation({
    mutationFn: (id: string) => fetchNodeInstall(id),
  });
}

export function useCreateServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      nodeId,
      name,
      gameId = "test",
    }: {
      nodeId: string;
      name: string;
      gameId?: string;
    }) => createServer(nodeId, name, gameId),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.servers }),
  });
}

export function useDeleteServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteServer(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.servers }),
  });
}

export function usePowerServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action }: { id: string; action: string }) =>
      powerServer(id, action),
    onMutate: async ({ id, action }) => {
      await qc.cancelQueries({ queryKey: queryKeys.servers });
      await qc.cancelQueries({ queryKey: queryKeys.adminServers });
      const prev = qc.getQueryData<Server[]>(queryKeys.servers);
      const prevAdmin = qc.getQueryData<any[]>(queryKeys.adminServers);
      const optimistic =
        action === "start" || action === "restart" ? "starting" : "stopping";
      qc.setQueryData<Server[]>(queryKeys.servers, (old) =>
        old?.map((s) => (s.id === id ? { ...s, status: optimistic } : s))
      );
      qc.setQueryData<any[]>(queryKeys.adminServers, (old) =>
        old?.map((s) => (s.id === id ? { ...s, status: optimistic } : s))
      );
      const prevOne = qc.getQueryData<Server>(queryKeys.server(id));
      qc.setQueryData<Server>(queryKeys.server(id), (old) =>
        old ? { ...old, status: optimistic } : old
      );
      const prevDetail = qc.getQueryData<DashboardServer>(queryKeys.serverDetail(id));
      qc.setQueryData<DashboardServer>(queryKeys.serverDetail(id), (old) =>
        old ? { ...old, status: optimistic, runtime_status: optimistic } : old
      );
      return { prev, prevAdmin, prevOne, prevDetail };
    },
    onError: (_err, { id }, ctx) => {
      if (ctx?.prev) qc.setQueryData(queryKeys.servers, ctx.prev);
      if (ctx?.prevAdmin) qc.setQueryData(queryKeys.adminServers, ctx.prevAdmin);
      if (ctx?.prevOne) qc.setQueryData(queryKeys.server(id), ctx.prevOne);
      if (ctx?.prevDetail) qc.setQueryData(queryKeys.serverDetail(id), ctx.prevDetail);
    },
    onSettled: (_data, _err, { id }) => {
      qc.invalidateQueries({ queryKey: queryKeys.servers });
      qc.invalidateQueries({ queryKey: queryKeys.adminServers });
      qc.invalidateQueries({ queryKey: ["my-servers"] });
      qc.invalidateQueries({ queryKey: queryKeys.server(id) });
      qc.invalidateQueries({ queryKey: queryKeys.serverStatus(id) });
      qc.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
    },
  });
}

export function useAdminToggleServerBlock() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      blocked,
      reason,
    }: {
      id: string;
      blocked: boolean;
      reason?: string;
    }) => adminToggleServerBlock(id, blocked, reason),
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.adminServers });
      void qc.invalidateQueries({ queryKey: queryKeys.servers });
    },
  });
}

export function patchNodeStatus(
  qc: ReturnType<typeof useQueryClient>,
  nodeId: string,
  status: string
) {
  qc.setQueryData<Node[]>(queryKeys.nodes, (old) =>
    old?.map((n) => (n.id === nodeId ? { ...n, status } : n))
  );
}

export function patchServerStatus(
  qc: ReturnType<typeof useQueryClient>,
  serverId: string,
  status: string
) {
  qc.setQueryData<Server[]>(queryKeys.servers, (old) =>
    old?.map((s) => (s.id === serverId ? { ...s, status } : s))
  );
  qc.setQueryData<Server>(queryKeys.server(serverId), (old) =>
    old ? { ...old, status } : old
  );
  qc.setQueryData<DashboardServer>(queryKeys.serverDetail(serverId), (old) =>
    old ? { ...old, status, runtime_status: status } : old
  );
  qc.setQueryData<DashboardServer[]>(["my-servers"], (old) =>
    old?.map((s) => (s.id === serverId ? { ...s, status, runtime_status: status } : s))
  );
}

export function patchInstallProgress(
  qc: ReturnType<typeof useQueryClient>,
  serverId: string,
  percent: number,
  stage: string,
  message: string
) {
  qc.setQueryData<{
    lines: string[];
    progress?: { percent?: number; stage?: string; message?: string; derived?: boolean };
  }>(["server-install-log", serverId], (old) =>
    old
      ? {
          ...old,
          progress: {
            ...old.progress,
            percent,
            stage: stage || old.progress?.stage,
            message: message || old.progress?.message,
            derived: false,
          },
        }
      : old
  );
}

export function useAdminLocations() {
  return useQuery({
    queryKey: queryKeys.adminLocations,
    queryFn: fetchAdminLocations,
  });
}

export function useAdminLocation(id: string) {
  return useQuery({
    queryKey: queryKeys.adminLocation(id),
    queryFn: () => fetchAdminLocation(id),
    enabled: !!id,
  });
}

export function useAdminLocationEdit(id: string) {
  return useQuery({
    queryKey: queryKeys.adminLocationEdit(id),
    queryFn: () => fetchAdminLocationEdit(id),
    enabled: !!id,
  });
}

export function useAdminLocationSetup(id: string) {
  return useQuery({
    queryKey: queryKeys.adminLocationSetup(id),
    queryFn: () => fetchAdminLocationSetup(id),
    enabled: !!id,
  });
}

export function useCreateAdminLocation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: createAdminLocation,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.adminLocations }),
  });
}

export function useUpdateAdminLocation(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Record<string, unknown>) => updateAdminLocation(id, data),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.adminLocations });
      void qc.invalidateQueries({ queryKey: queryKeys.adminLocation(id) });
      void qc.invalidateQueries({ queryKey: queryKeys.adminLocationEdit(id) });
    },
  });
}

export function useToggleAdminLocation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: toggleAdminLocation,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.adminLocations }),
  });
}

export function useDeleteAdminLocation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: deleteAdminLocation,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.adminLocations }),
  });
}

export function useRunAdminLocationSetupStep(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (step: string) => runAdminLocationSetupStep(id, step),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.adminLocationSetup(id) }),
  });
}

export function usePullAdminLocationDaemon(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => pullAdminLocationDaemon(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.adminLocation(id) }),
  });
}

export function useAdminGames() {
  return useQuery({
    queryKey: queryKeys.adminGames,
    queryFn: fetchAdminGames,
  });
}

export function useAdminGame(id: string) {
  return useQuery({
    queryKey: queryKeys.adminGame(id),
    queryFn: () => fetchAdminGame(id),
    enabled: !!id,
  });
}

export function useAdminGameEdit(id: string) {
  return useQuery({
    queryKey: queryKeys.adminGameEdit(id),
    queryFn: () => fetchAdminGameEdit(id),
    enabled: !!id,
  });
}

export function useCreateAdminGame() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: createAdminGame,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.adminGames }),
  });
}

export function useUpdateAdminGame(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Record<string, unknown>) => updateAdminGame(id, data),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.adminGames });
      void qc.invalidateQueries({ queryKey: queryKeys.adminGame(id) });
      void qc.invalidateQueries({ queryKey: queryKeys.adminGameEdit(id) });
    },
  });
}

export function useToggleAdminGame() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: toggleAdminGame,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.adminGames }),
  });
}

export function useDeleteAdminGame() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: deleteAdminGame,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.adminGames }),
  });
}

export function useCreateAdminGameVersion(gameId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: createAdminGameVersion.bind(null, gameId),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: queryKeys.adminGameEdit(gameId) }),
  });
}

export function useDeleteAdminGameVersion(gameId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (versionId: string) => deleteAdminGameVersion(gameId, versionId),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: queryKeys.adminGameEdit(gameId) }),
  });
}
