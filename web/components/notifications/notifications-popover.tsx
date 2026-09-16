"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { AnimatePresence, m } from "motion/react";
import { ArrowRight, BellOff, CheckCheck, Inbox, Loader2, RotateCw, Settings2 } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import {
  useNotificationActions,
  useNotificationsFeed,
  useOpenNotification,
  useUnreadCount,
} from "@/hooks/use-notifications";
import { useT } from "@/hooks/use-translations";
import type { PanelNotification } from "@/lib/api";
import { cn } from "@/lib/utils";
import { NotificationCompactItem } from "@/components/notifications/notification-item";
import { groupByDay } from "@/components/notifications/notification-utils";

type Tab = "all" | "unread";

const PAGE_SIZE = 20;

export function NotificationsPopover({ onClose }: { onClose: () => void }) {
  const t = useT();
  const [tab, setTab] = useState<Tab>("all");
  const unread = useUnreadCount();
  const feed = useNotificationsFeed({ group: "", unread: tab === "unread", q: "" }, PAGE_SIZE);
  const actions = useNotificationActions();
  const open = useOpenNotification();

  const listRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const loadMore = useRef(() => {});
  loadMore.current = () => {
    if (feed.hasNextPage && !feed.isFetchingNextPage) void feed.fetchNextPage();
  };

  const hasNext = Boolean(feed.hasNextPage);
  useEffect(() => {
    const el = sentinelRef.current;
    if (!el || !hasNext) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) loadMore.current();
      },
      { root: listRef.current, rootMargin: "160px" }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [hasNext, tab]);

  const items = feed.data?.pages.flatMap((page) => page.notifications) ?? [];
  const total = feed.data?.pages[0]?.groups.reduce((sum, g) => sum + g.count, 0);
  const sections = groupByDay(items);

  const handleOpen = (item: PanelNotification) => {
    if (open(item)) onClose();
  };

  const tabs: { id: Tab; label: string; count?: number }[] = [
    { id: "all", label: t("notifications.tab_all"), count: total },
    { id: "unread", label: t("notifications.tab_unread"), count: unread },
  ];

  return (
    <div className="flex max-h-[min(580px,calc(100dvh-88px))] flex-col">
      <div className="flex items-center gap-1 px-4 pt-3.5 pb-2.5">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <h2 className="text-[15px] font-semibold tracking-[-0.01em]">
            {t("notifications.title")}
          </h2>
          <AnimatePresence initial={false}>
            {unread > 0 && (
              <m.span
                key="count"
                initial={{ opacity: 0, scale: 0.6 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.6 }}
                transition={{ type: "spring", stiffness: 500, damping: 28 }}
                className="rounded-full bg-primary px-1.5 py-px text-[11px] font-semibold text-primary-foreground tabular-nums"
              >
                {unread > 99 ? "99+" : unread}
              </m.span>
            )}
          </AnimatePresence>
        </div>
        <button
          type="button"
          onClick={() => actions.readAll()}
          disabled={unread === 0 || actions.readAllPending}
          title={t("notifications.mark_all")}
          aria-label={t("notifications.mark_all")}
          className="flex size-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground disabled:pointer-events-none disabled:opacity-40"
        >
          <CheckCheck className="size-4" />
        </button>
        <Link
          href="/settings?tab=notifications"
          onClick={onClose}
          title={t("notifications.settings")}
          aria-label={t("notifications.settings")}
          className="flex size-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <Settings2 className="size-4" />
        </Link>
      </div>

      <div className="px-4 pb-2">
        <div role="tablist" className="flex gap-1 rounded-[10px] bg-muted p-[3px]">
          {tabs.map((item) => (
            <button
              key={item.id}
              type="button"
              role="tab"
              aria-selected={tab === item.id}
              onClick={() => setTab(item.id)}
              className={cn(
                "relative flex h-7 flex-1 items-center justify-center gap-1.5 rounded-[8px] text-[12.5px] transition-colors",
                tab === item.id ? "font-medium text-foreground" : "text-muted-foreground hover:text-foreground"
              )}
            >
              {tab === item.id && (
                <m.span
                  layoutId="notifications-popover-tab"
                  className="absolute inset-0 rounded-[8px] bg-background shadow-sm"
                  transition={{ type: "spring", stiffness: 480, damping: 36 }}
                />
              )}
              <span className="relative">{item.label}</span>
              {item.count !== undefined && item.count > 0 && (
                <span className="relative font-mono text-[11px] text-muted-foreground tabular-nums">
                  {item.count}
                </span>
              )}
            </button>
          ))}
        </div>
      </div>

      <div
        ref={listRef}
        className="min-h-0 flex-1 overflow-y-auto overscroll-contain border-t border-border"
      >
        {feed.isLoading ? (
          <div className="flex flex-col gap-1 p-2">
            {[0, 1, 2, 3].map((i) => (
              <div key={i} className="flex gap-3 px-2 py-2.5">
                <Skeleton className="size-9 shrink-0 rounded-[10px]" />
                <div className="flex flex-1 flex-col gap-2 pt-0.5">
                  <Skeleton className="h-3.5 w-3/5" />
                  <Skeleton className="h-3 w-full" />
                </div>
              </div>
            ))}
          </div>
        ) : feed.isError ? (
          <PopoverState
            icon={<RotateCw className="size-4" />}
            title={t("notifications.load_failed_title")}
            text={t("notifications.load_failed_text")}
            action={
              <button
                type="button"
                onClick={() => void feed.refetch()}
                className="mt-1 rounded-lg border border-border px-3 py-1.5 text-[12.5px] font-medium transition-colors hover:bg-accent"
              >
                {t("notifications.retry")}
              </button>
            }
          />
        ) : items.length === 0 ? (
          tab === "unread" ? (
            <PopoverState
              icon={<CheckCheck className="size-4" />}
              title={t("notifications.all_read_title")}
              text={t("notifications.all_read_text")}
            />
          ) : (
            <PopoverState
              icon={<BellOff className="size-4" />}
              title={t("notifications.empty_title")}
              text={t("notifications.empty_text")}
            />
          )
        ) : (
          <>
            {sections.map((section) => (
              <section key={section.key}>
                <div className="sticky top-0 z-20 border-b border-border/60 bg-popover/95 px-4 py-1.5 text-[11px] font-semibold tracking-[0.04em] text-muted-foreground uppercase backdrop-blur supports-[backdrop-filter]:bg-popover/80">
                  {section.title}
                </div>
                <AnimatePresence initial={false}>
                  {section.items.map((item) => (
                    <m.div
                      key={item.id}
                      layout="position"
                      initial={{ opacity: 0, height: 0 }}
                      animate={{ opacity: 1, height: "auto" }}
                      exit={{ opacity: 0, height: 0 }}
                      transition={{ duration: 0.22, ease: [0.22, 1, 0.36, 1] }}
                      className="overflow-hidden"
                    >
                      <NotificationCompactItem
                        item={item}
                        onOpen={handleOpen}
                        onToggleRead={(n) => (n.unread ? actions.markRead(n) : actions.markUnread(n))}
                        onRemove={actions.remove}
                      />
                    </m.div>
                  ))}
                </AnimatePresence>
              </section>
            ))}
            {hasNext && (
              <div ref={sentinelRef} className="flex justify-center py-3 text-muted-foreground">
                <Loader2 className="size-4 animate-spin" />
              </div>
            )}
          </>
        )}
      </div>

      <div className="border-t border-border p-1.5">
        <Link
          href="/notifications"
          onClick={onClose}
          className="group flex h-9 items-center justify-center gap-1.5 rounded-lg text-[13px] font-medium text-foreground transition-colors hover:bg-accent"
        >
          <Inbox className="size-4 text-muted-foreground" />
          {t("notifications.view_all")}
          <ArrowRight className="size-3.5 transition-transform duration-200 group-hover:translate-x-0.5" />
        </Link>
      </div>
    </div>
  );
}

function PopoverState({
  icon,
  title,
  text,
  action,
}: {
  icon: React.ReactNode;
  title: string;
  text: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center gap-2 px-6 py-10 text-center">
      <span className="flex size-10 items-center justify-center rounded-xl bg-muted text-muted-foreground">
        {icon}
      </span>
      <div className="text-[13.5px] font-medium">{title}</div>
      <p className="max-w-[260px] text-[12.5px] leading-[1.5] text-muted-foreground">{text}</p>
      {action}
    </div>
  );
}
