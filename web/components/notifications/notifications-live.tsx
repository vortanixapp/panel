"use client";

import { useEffect, useRef } from "react";
import { usePathname } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { setUnreadCount, useOpenNotification } from "@/hooks/use-notifications";
import { useMe } from "@/hooks/use-queries";
import { subscribeNotifications, type PanelNotification } from "@/lib/api";
import { adminTasksKey } from "@/hooks/use-admin-tasks";
import { browserNotificationsActive } from "@/lib/browser-notifications";
import { setLiveLinked } from "@/lib/live-link";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";

type LiveMessage =
  | { type: "hello"; unread: number }
  | { type: "sync"; unread: number }
  | { type: "notification"; item: PanelNotification; unread: number }
  | { type: "seen"; id: string }
  | { type: "invalidate"; topic: string }
  | { type: "link"; up: boolean };

const SEEN_WAIT_MS = 300;
const INVALIDATE_BATCH_MS = 250;

function clip(text: string, limit: number) {
  const flat = text.replace(/\s+/g, " ").trim();
  return flat.length > limit ? flat.slice(0, limit - 1) + "…" : flat;
}

function iconHref(): string | undefined {
  const link = document.querySelector<HTMLLinkElement>('link[rel~="icon"]');
  return link?.href || undefined;
}

export function NotificationsLive() {
  const qc = useQueryClient();
  const { data: me } = useMe();
  const accountId = me?.user_id ?? "";
  const pathname = usePathname();
  const open = useOpenNotification();

  const pathRef = useRef(pathname);
  const openRef = useRef(open);
  useEffect(() => {
    pathRef.current = pathname;
    openRef.current = open;
  });

  useEffect(() => {
    if (!accountId) return;

    const channel =
      typeof BroadcastChannel !== "undefined"
        ? new BroadcastChannel(`vx-notifications:${accountId}`)
        : null;
    const seen = new Set<string>();
    const topics = new Set<string>();
    let flushTimer: number | undefined;
    let helloSeen = false;

    const invalidateTopic = (topic: string) => {
      if (topic === "jobs") {
        void qc.invalidateQueries({ queryKey: adminTasksKey });
        void qc.invalidateQueries({ queryKey: ["admin-jobs"] });
        return;
      }
      const id = topic.startsWith("server:") ? topic.slice("server:".length) : "";
      if (!id) return;
      void qc.invalidateQueries({ queryKey: queryKeys.serverStatus(id) });
      void qc.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
      void qc.invalidateQueries({ queryKey: queryKeys.serverRuntime(id) });
      void qc.invalidateQueries({ queryKey: ["server-logs", id] });
      void qc.invalidateQueries({ queryKey: ["my-servers"] });
      void qc.invalidateQueries({ queryKey: queryKeys.adminServerCard(id) });
      void qc.invalidateQueries({ queryKey: queryKeys.adminServers, exact: true });
    };

    const queueTopic = (topic: string) => {
      topics.add(topic);
      if (flushTimer !== undefined) return;
      flushTimer = window.setTimeout(() => {
        flushTimer = undefined;
        const batch = [...topics];
        topics.clear();
        batch.forEach(invalidateTopic);
      }, INVALIDATE_BATCH_MS);
    };
    const visible = () => document.visibilityState === "visible";

    const showToast = (item: PanelNotification) => {
      const show =
        item.tone === "bad"
          ? toast.error
          : item.tone === "warn"
            ? toast.warning
            : item.tone === "ok"
              ? toast.success
              : toast.info;
      show(item.title, {
        id: `notification-${item.id}`,
        description: item.body ? clip(item.body, 140) : undefined,
        action: {
          label: item.href && item.action ? item.action : t("notifications.open"),
          onClick: () => {
            openRef.current(item.href ? item : { ...item, href: "/notifications" });
          },
        },
      });
    };

    const showSystem = (item: PanelNotification) => {
      try {
        const note = new window.Notification(item.title, {
          body: item.body ? clip(item.body, 180) : undefined,
          tag: item.id,
          icon: iconHref(),
        });
        note.onclick = () => {
          window.focus();
          openRef.current(item.href ? item : { ...item, href: "/notifications" });
          note.close();
        };
      } catch {}
    };

    const present = (item: PanelNotification) => {
      if (item.quiet) return;
      if (visible()) {
        channel?.postMessage({ type: "seen", id: item.id } satisfies LiveMessage);
        if (!pathRef.current.startsWith("/notifications")) showToast(item);
        return;
      }
      if (!browserNotificationsActive()) return;
      window.setTimeout(() => {
        if (seen.has(item.id) || visible()) return;
        showSystem(item);
      }, SEEN_WAIT_MS);
    };

    const handle = (message: LiveMessage) => {
      switch (message.type) {
        case "hello": {
          if (helloSeen) {
            invalidateTopic("jobs");
            void qc.invalidateQueries({ queryKey: queryKeys.servers });
            void qc.invalidateQueries({ queryKey: ["my-servers"] });
            void qc.invalidateQueries({ queryKey: queryKeys.adminServers });
          }
          helloSeen = true;
          const previous = qc.getQueryData<{ count: number }>(queryKeys.notificationsUnread)?.count;
          setUnreadCount(qc, message.unread);
          if (previous !== undefined && previous !== message.unread) {
            void qc.invalidateQueries({ queryKey: queryKeys.notificationsFeeds });
          }
          break;
        }
        case "sync":
          setUnreadCount(qc, message.unread);
          void qc.invalidateQueries({ queryKey: queryKeys.notificationsFeeds, refetchType: "none" });
          break;
        case "notification":
          setUnreadCount(qc, message.unread);
          void qc.invalidateQueries({ queryKey: queryKeys.notificationsFeeds });
          present(message.item);
          break;
        case "seen":
          seen.add(message.id);
          break;
        case "invalidate":
          queueTopic(message.topic);
          break;
        case "link":
          setLiveLinked(message.up);
          break;
      }
    };

    const onChannelMessage = (event: MessageEvent<LiveMessage>) => handle(event.data);
    channel?.addEventListener("message", onChannelMessage);

    const relay = (message: LiveMessage) => {
      handle(message);
      channel?.postMessage(message);
    };
    const startStream = () =>
      subscribeNotifications({
        onHello: (unread) => relay({ type: "hello", unread }),
        onSync: (unread) => relay({ type: "sync", unread }),
        onNotification: (item, unread) => relay({ type: "notification", item, unread }),
        onInvalidate: (topic) => relay({ type: "invalidate", topic }),
        onLink: (up) => relay({ type: "link", up }),
      });

    const abort = new AbortController();
    let stop: (() => void) | null = null;
    if (channel && typeof navigator !== "undefined" && navigator.locks) {
      navigator.locks
        .request(`vx-notifications-stream:${accountId}`, { signal: abort.signal }, () => {
          if (abort.signal.aborted) return Promise.resolve();
          stop = startStream();
          return new Promise<void>((resolve) => {
            abort.signal.addEventListener(
              "abort",
              () => {
                stop?.();
                resolve();
              },
              { once: true }
            );
          });
        })
        .catch(() => {});
    } else {
      stop = startStream();
    }

    return () => {
      abort.abort();
      stop?.();
      setLiveLinked(false);
      if (flushTimer !== undefined) window.clearTimeout(flushTimer);
      channel?.removeEventListener("message", onChannelMessage);
      channel?.close();
    };
  }, [accountId, qc]);

  return null;
}
