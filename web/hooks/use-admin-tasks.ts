"use client";

import { useQuery } from "@tanstack/react-query";

import {
  fetchAdminTaskDetail,
  fetchAdminTaskEvents,
  fetchAdminTaskFeed,
  type AdminTask,
} from "@/lib/api";
import { pollMs } from "@/lib/public-settings";

export const adminTasksKey = ["admin-tasks"] as const;

export function useAdminTaskFeed(scope: "active" | "history", enabled: boolean, open: boolean) {
  return useQuery({
    queryKey: [...adminTasksKey, "feed", scope],
    queryFn: () => fetchAdminTaskFeed(scope),
    enabled,
    retry: false,
    refetchInterval: pollMs(open ? 2000 : 8000),
  });
}

export function useAdminTaskDetail(task: Pick<AdminTask, "source" | "id"> | null) {
  return useQuery({
    queryKey: [...adminTasksKey, "detail", task?.source, task?.id],
    queryFn: () => fetchAdminTaskDetail(task!.source, task!.id),
    enabled: !!task,
    refetchInterval: (query) => {
      const status = query.state.data?.task.status;
      return status === "queued" || status === "running" ? pollMs(2000) : false;
    },
  });
}

export function useAdminTaskEvents(enabled: boolean) {
  return useQuery({
    queryKey: [...adminTasksKey, "events"],
    queryFn: () => fetchAdminTaskEvents(),
    enabled,
    refetchInterval: pollMs(5000),
  });
}
