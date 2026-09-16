"use client";

import { useRouter } from "next/navigation";
import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type InfiniteData,
  type QueryClient,
} from "@tanstack/react-query";
import { toast } from "sonner";
import {
  clearReadNotifications,
  deleteNotification,
  fetchNotificationPrefs,
  fetchNotifications,
  fetchNotificationsUnreadCount,
  markAllNotificationsRead,
  markNotificationRead,
  markNotificationUnread,
  updateNotificationPrefs,
  type NotificationChange,
  type NotificationGroup,
  type NotificationPrefs,
  type NotificationPrefsUpdate,
  type NotificationsPage,
  type PanelNotification,
} from "@/lib/api";
import { t } from "@/lib/i18n";
import { notificationTarget } from "@/lib/notification-link";
import { queryKeys } from "@/lib/query-keys";

export type NotificationsFilter = {
  group: string;
  unread: boolean;
  q: string;
};

type Feed = InfiniteData<NotificationsPage, string>;

export function useNotificationsFeed(filter: NotificationsFilter, limit: number, enabled = true) {
  return useInfiniteQuery({
    queryKey: queryKeys.notificationsFeed(filter.group, filter.unread, filter.q, limit),
    queryFn: ({ pageParam }) =>
      fetchNotifications({
        group: filter.group || undefined,
        unread: filter.unread,
        q: filter.q || undefined,
        before: pageParam || undefined,
        limit,
      }),
    initialPageParam: "",
    getNextPageParam: (last) => last.next_cursor || undefined,
    enabled,
  });
}

export function useUnreadCount(): number {
  const query = useQuery({
    queryKey: queryKeys.notificationsUnread,
    queryFn: fetchNotificationsUnreadCount,
    refetchInterval: 120_000,
    refetchOnWindowFocus: true,
  });
  return query.data?.count ?? 0;
}

export function setUnreadCount(qc: QueryClient, count: number) {
  qc.setQueryData(queryKeys.notificationsUnread, { count: Math.max(0, count) });
}

function unreadCount(qc: QueryClient): number {
  return qc.getQueryData<{ count: number }>(queryKeys.notificationsUnread)?.count ?? 0;
}

function patchFeeds(qc: QueryClient, patch: (page: NotificationsPage) => NotificationsPage) {
  qc.setQueriesData<Feed>({ queryKey: queryKeys.notificationsFeeds }, (data) =>
    data ? { ...data, pages: data.pages.map(patch) } : data
  );
}

function shiftGroup(groups: NotificationGroup[], id: string, unread: number, count: number) {
  return groups.map((g) =>
    g.id === id
      ? { ...g, unread: Math.max(0, g.unread + unread), count: Math.max(0, g.count + count) }
      : g
  );
}

export function useNotificationActions() {
  const qc = useQueryClient();

  const settle = (res: NotificationChange) => {
    setUnreadCount(qc, res.unread);
    void qc.invalidateQueries({ queryKey: queryKeys.notificationsFeeds, refetchType: "none" });
  };

  const fail = () => {
    void qc.invalidateQueries({ queryKey: queryKeys.notificationsFeeds });
    void qc.invalidateQueries({ queryKey: queryKeys.notificationsUnread });
    toast.error(t("notifications.action_failed"));
  };

  const toggleRead = useMutation({
    mutationFn: ({ item, read }: { item: PanelNotification; read: boolean }) =>
      read ? markNotificationRead(item.id) : markNotificationUnread(item.id),
    onMutate: ({ item, read }) => {
      if (item.unread !== read) return;
      patchFeeds(qc, (page) => ({
        ...page,
        notifications: page.notifications.map((n) =>
          n.id === item.id ? { ...n, unread: !read } : n
        ),
        groups: shiftGroup(page.groups, item.group, read ? -1 : 1, 0),
      }));
      setUnreadCount(qc, unreadCount(qc) + (read ? -1 : 1));
    },
    onSuccess: settle,
    onError: fail,
  });

  const remove = useMutation({
    mutationFn: (item: PanelNotification) => deleteNotification(item.id),
    onMutate: (item) => {
      patchFeeds(qc, (page) => ({
        ...page,
        notifications: page.notifications.filter((n) => n.id !== item.id),
        groups: shiftGroup(page.groups, item.group, item.unread ? -1 : 0, -1).filter(
          (g) => g.count > 0
        ),
      }));
      if (item.unread) setUnreadCount(qc, unreadCount(qc) - 1);
    },
    onSuccess: settle,
    onError: fail,
  });

  const readAll = useMutation({
    mutationFn: (group: string) => markAllNotificationsRead(group || undefined),
    onMutate: (group) => {
      patchFeeds(qc, (page) => ({
        ...page,
        notifications: page.notifications.map((n) =>
          n.unread && (!group || n.group === group) ? { ...n, unread: false } : n
        ),
        groups: page.groups.map((g) => (!group || g.id === group ? { ...g, unread: 0 } : g)),
      }));
      if (!group) setUnreadCount(qc, 0);
    },
    onSuccess: settle,
    onError: fail,
  });

  const clearRead = useMutation({
    mutationFn: clearReadNotifications,
    onMutate: () => {
      patchFeeds(qc, (page) => ({
        ...page,
        notifications: page.notifications.filter((n) => n.unread),
        groups: page.groups.map((g) => ({ ...g, count: g.unread })).filter((g) => g.count > 0),
      }));
    },
    onSuccess: (res) => {
      settle(res);
      toast.success(
        res.affected > 0
          ? t("notifications.cleared", { count: res.affected })
          : t("notifications.clear_empty")
      );
    },
    onError: fail,
  });

  return {
    markRead: (item: PanelNotification) => {
      if (item.unread) toggleRead.mutate({ item, read: true });
    },
    markUnread: (item: PanelNotification) => {
      if (!item.unread) toggleRead.mutate({ item, read: false });
    },
    remove: (item: PanelNotification) => remove.mutate(item),
    readAll: (group = "") => readAll.mutate(group),
    clearRead: () => clearRead.mutate(),
    readAllPending: readAll.isPending,
    clearReadPending: clearRead.isPending,
  };
}

export function useOpenNotification() {
  const router = useRouter();
  const actions = useNotificationActions();
  return (item: PanelNotification): boolean => {
    actions.markRead(item);
    const target = notificationTarget(item.href);
    if (!target) return false;
    if (target.kind === "internal") router.push(target.path);
    else window.open(target.url, "_blank", "noopener,noreferrer");
    return true;
  };
}

export function useNotificationPrefs() {
  return useQuery({
    queryKey: queryKeys.notificationPrefs,
    queryFn: fetchNotificationPrefs,
    staleTime: 30_000,
  });
}

function applyPrefs(prev: NotificationPrefs, payload: NotificationPrefsUpdate): NotificationPrefs {
  const { routes, ...channels } = payload;
  return {
    ...prev,
    channels: { ...prev.channels, ...channels },
    groups: prev.groups.map((g) =>
      routes?.[g.id] ? { ...g, routes: { ...g.routes, ...routes[g.id] } } : g
    ),
  };
}

export function useUpdateNotificationPrefs() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: NotificationPrefsUpdate) => updateNotificationPrefs(payload),
    onMutate: async (payload) => {
      await qc.cancelQueries({ queryKey: queryKeys.notificationPrefs });
      const previous = qc.getQueryData<NotificationPrefs>(queryKeys.notificationPrefs);
      if (previous) qc.setQueryData(queryKeys.notificationPrefs, applyPrefs(previous, payload));
      return { previous };
    },
    onError: (_error, _payload, context) => {
      if (context?.previous) qc.setQueryData(queryKeys.notificationPrefs, context.previous);
    },
    onSuccess: (data) => {
      qc.setQueryData(queryKeys.notificationPrefs, data);
    },
  });
}

const PREFS_ERRORS = new Set([
  "invalid_chat_id",
  "invalid_webhook",
  "chat_id_required",
  "webhook_required",
  "channel_not_configured",
  "too_many_requests",
  "telegram_unavailable",
]);

export function notificationErrorText(error: unknown): string {
  const message = error instanceof Error ? error.message : "";
  if (PREFS_ERRORS.has(message)) return t(`notifications.error.${message}`);
  return message || t("notifications.action_failed");
}
