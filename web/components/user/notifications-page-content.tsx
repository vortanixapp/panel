"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { AnimatePresence, m } from "motion/react";
import {
  BellOff,
  CheckCheck,
  Loader2,
  MonitorSmartphone,
  MoreHorizontal,
  RotateCw,
  Search,
  SlidersHorizontal,
  Trash2,
} from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { AnimatedNumber } from "@/components/vx/motion";
import { btnGhost, btnPrimary, fieldClass } from "@/components/user/panel-parts";
import { NotificationFullItem } from "@/components/notifications/notification-item";
import { groupByDay } from "@/components/notifications/notification-utils";
import { groupIcon } from "@/components/notifications/group-icons";
import {
  useNotificationActions,
  useNotificationPrefs,
  useNotificationsFeed,
  useOpenNotification,
  useUnreadCount,
} from "@/hooks/use-notifications";
import { useT } from "@/hooks/use-translations";
import type { NotificationGroup, NotificationPrefs } from "@/lib/api";
import {
  enableBrowserNotifications,
  setBrowserNotificationsEnabled,
  useBrowserNotifications,
} from "@/lib/browser-notifications";
import { cn } from "@/lib/utils";

const PAGE_SIZE = 30;

type ParamsPatch = { filter?: "all" | "unread"; group?: string; q?: string };

export function NotificationsPageContent() {
  const t = useT();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const unreadOnly = searchParams.get("filter") === "unread";
  const group = searchParams.get("group") ?? "";
  const q = searchParams.get("q") ?? "";
  const [search, setSearch] = useState(q);
  const [confirmClear, setConfirmClear] = useState(false);

  const setParams = useCallback(
    (patch: ParamsPatch) => {
      const params = new URLSearchParams(searchParams.toString());
      const apply = (key: string, value: string) => {
        if (value) params.set(key, value);
        else params.delete(key);
      };
      if (patch.filter !== undefined) apply("filter", patch.filter === "unread" ? "unread" : "");
      if (patch.group !== undefined) apply("group", patch.group);
      if (patch.q !== undefined) apply("q", patch.q);
      const query = params.toString();
      router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false });
    },
    [pathname, router, searchParams]
  );

  useEffect(() => {
    const value = search.trim();
    if (value === q) return;
    const timer = setTimeout(() => setParams({ q: value }), 300);
    return () => clearTimeout(timer);
  }, [search, q, setParams]);

  const unread = useUnreadCount();
  const feed = useNotificationsFeed({ group, unread: unreadOnly, q }, PAGE_SIZE);
  const actions = useNotificationActions();
  const open = useOpenNotification();
  const prefs = useNotificationPrefs();

  const items = feed.data?.pages.flatMap((page) => page.notifications) ?? [];
  const groups = feed.data?.pages[0]?.groups ?? [];
  const total = groups.reduce((sum, g) => sum + g.count, 0);
  const sections = groupByDay(items);
  const filtered = Boolean(group || q || unreadOnly);
  const activeGroup = groups.find((g) => g.id === group);

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
      { rootMargin: "400px" }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [hasNext]);

  const resetFilters = () => {
    setSearch("");
    router.replace(pathname, { scroll: false });
  };

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-[22px]">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div className="flex flex-col gap-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-[-0.02em]">
              {t("notifications.title")}
            </h1>
            <p className="text-[13.5px] text-muted-foreground">{t("notifications.subtitle")}</p>
          </div>
          <div className="flex items-center gap-2">
            <Link href="/settings?tab=notifications" className={cn(btnGhost, "h-[38px]")}>
              <SlidersHorizontal className="size-3.5" />
              <span className="max-sm:hidden">{t("notifications.configure")}</span>
            </Link>
            <button
              type="button"
              onClick={() => actions.readAll(group)}
              disabled={(activeGroup ? activeGroup.unread : unread) === 0 || actions.readAllPending}
              className={cn(btnPrimary, "h-[38px] px-[18px]")}
            >
              <CheckCheck className="size-4" />
              {activeGroup ? t("notifications.mark_group") : t("notifications.mark_all")}
            </button>
            <DropdownMenu modal={false}>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  aria-label={t("notifications.more_actions")}
                  className={cn(btnGhost, "h-[38px] w-[38px] px-0")}
                >
                  <MoreHorizontal className="size-4" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-56">
                <DropdownMenuItem
                  onSelect={() => setConfirmClear(true)}
                  disabled={total - groups.reduce((sum, g) => sum + g.unread, 0) === 0}
                  className="text-destructive focus:text-destructive"
                >
                  <Trash2 className="size-4" />
                  {t("notifications.clear_read")}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        <div className="grid items-start gap-[22px] lg:grid-cols-[minmax(0,1fr)_300px]">
          <div className="flex min-w-0 flex-col gap-4">
            <div className="flex flex-wrap items-center gap-2">
              <div className="flex gap-1 rounded-xl border border-border bg-card p-1">
                {(["all", "unread"] as const).map((id) => {
                  const active = (id === "unread") === unreadOnly;
                  const count = id === "unread" ? unread : total;
                  return (
                    <button
                      key={id}
                      type="button"
                      aria-pressed={active}
                      onClick={() => setParams({ filter: id })}
                      className={cn(
                        "relative flex h-9 items-center gap-2 rounded-[9px] px-3.5 text-[13px] whitespace-nowrap transition-colors",
                        active ? "font-medium text-accent-foreground" : "text-muted-foreground hover:text-foreground"
                      )}
                    >
                      {active && (
                        <m.span
                          layoutId="notifications-page-filter"
                          className="absolute inset-0 rounded-[9px] bg-accent"
                          transition={{ type: "spring", stiffness: 480, damping: 38 }}
                        />
                      )}
                      <span className="relative">
                        {id === "unread" ? t("notifications.tab_unread") : t("notifications.tab_all")}
                      </span>
                      <span className="relative font-mono text-[11.5px] text-muted-foreground/80 tabular-nums">
                        {count}
                      </span>
                    </button>
                  );
                })}
              </div>

              <label className="relative ms-auto min-w-[200px] flex-1 sm:max-w-[280px] sm:flex-none">
                <Search className="pointer-events-none absolute start-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <input
                  type="search"
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder={t("notifications.search_placeholder")}
                  className={cn(fieldClass, "ps-9")}
                />
              </label>
            </div>

            {groups.length > 1 && (
              <div className="no-scrollbar -mx-1 flex gap-1.5 overflow-x-auto px-1 pb-0.5">
                <GroupChip active={!group} onClick={() => setParams({ group: "" })}>
                  {t("notifications.all_categories")}
                </GroupChip>
                {groups.map((g) => {
                  const Icon = groupIcon(g.id);
                  return (
                    <GroupChip key={g.id} active={group === g.id} onClick={() => setParams({ group: g.id })}>
                      <Icon className="size-3.5" />
                      {g.label}
                      {g.unread > 0 && (
                        <span className="rounded-full bg-primary px-1.5 text-[10.5px] leading-4 font-semibold text-primary-foreground tabular-nums">
                          {g.unread}
                        </span>
                      )}
                    </GroupChip>
                  );
                })}
              </div>
            )}

            {feed.isLoading ? (
              <div className="flex flex-col gap-3">
                <Skeleton className="h-4 w-24 rounded-md" />
                <Skeleton className="h-[220px] rounded-2xl" />
                <Skeleton className="h-4 w-24 rounded-md" />
                <Skeleton className="h-[140px] rounded-2xl" />
              </div>
            ) : feed.isError ? (
              <EmptyCard
                icon={<RotateCw className="size-4" />}
                title={t("notifications.load_failed_title")}
                text={t("notifications.load_failed_text")}
              >
                <button type="button" onClick={() => void feed.refetch()} className={cn(btnGhost, "h-[34px]")}>
                  {t("notifications.retry")}
                </button>
              </EmptyCard>
            ) : items.length === 0 ? (
              total === 0 && !q ? (
                <EmptyCard
                  icon={<BellOff className="size-4" />}
                  title={t("notifications.empty_title")}
                  text={t("notifications.empty_page_text")}
                >
                  <Link href="/settings?tab=notifications" className={cn(btnGhost, "h-[34px]")}>
                    {t("notifications.configure")}
                  </Link>
                </EmptyCard>
              ) : unreadOnly && !q && !group ? (
                <EmptyCard
                  icon={<CheckCheck className="size-4" />}
                  title={t("notifications.all_read_title")}
                  text={t("notifications.all_read_text")}
                >
                  <button type="button" onClick={() => setParams({ filter: "all" })} className={cn(btnGhost, "h-[34px]")}>
                    {t("notifications.show_all")}
                  </button>
                </EmptyCard>
              ) : (
                <EmptyCard
                  icon={<Search className="size-4" />}
                  title={t("notifications.not_found_title")}
                  text={t("notifications.not_found_text")}
                >
                  {filtered && (
                    <button type="button" onClick={resetFilters} className={cn(btnGhost, "h-[34px]")}>
                      {t("notifications.reset_filters")}
                    </button>
                  )}
                </EmptyCard>
              )
            ) : (
              <div className="flex flex-col gap-5">
                {sections.map((section) => (
                  <section key={section.key} className="flex flex-col gap-2">
                    <h2 className="px-1 text-[12px] font-semibold tracking-[0.04em] text-muted-foreground uppercase">
                      {section.title}
                    </h2>
                    <div
                      data-spotlight
                      className="relative isolate overflow-hidden rounded-2xl border border-border bg-card"
                    >
                      <AnimatePresence initial={false}>
                        {section.items.map((item, index) => (
                          <m.div
                            key={item.id}
                            layout="position"
                            initial={{ opacity: 0, height: 0 }}
                            animate={{ opacity: 1, height: "auto" }}
                            exit={{ opacity: 0, height: 0 }}
                            transition={{ duration: 0.24, ease: [0.22, 1, 0.36, 1] }}
                            className={cn("overflow-hidden", index > 0 && "border-t border-border")}
                          >
                            <NotificationFullItem
                              item={item}
                              onOpen={(n) => void open(n)}
                              onToggleRead={(n) => (n.unread ? actions.markRead(n) : actions.markUnread(n))}
                              onRemove={actions.remove}
                            />
                          </m.div>
                        ))}
                      </AnimatePresence>
                    </div>
                  </section>
                ))}
                {hasNext && (
                  <div ref={sentinelRef} className="flex justify-center">
                    <button
                      type="button"
                      onClick={() => loadMore.current()}
                      disabled={feed.isFetchingNextPage}
                      className={cn(btnGhost, "h-[38px]")}
                    >
                      {feed.isFetchingNextPage && <Loader2 className="size-3.5 animate-spin" />}
                      {t("notifications.load_more")}
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>

          <aside className="flex flex-col gap-4 lg:sticky lg:top-20">
            <SummaryCard
              unread={unread}
              groups={groups}
              active={group}
              onPick={(id) => setParams({ group: id === group ? "" : id })}
            />
            <ChannelsCard prefs={prefs.data} loading={prefs.isLoading} />
            <BrowserCard />
          </aside>
        </div>
      </div>

      <ConfirmDialog
        open={confirmClear}
        onOpenChange={setConfirmClear}
        title={t("notifications.clear_read_title")}
        desc={t("notifications.clear_read_text")}
        confirmText={t("notifications.clear_read")}
        destructive
        isLoading={actions.clearReadPending}
        handleConfirm={() => {
          actions.clearRead();
          setConfirmClear(false);
        }}
      />
    </PageShell>
  );
}

function GroupChip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        "flex h-8 flex-none items-center gap-1.5 rounded-full border px-3 text-[12.5px] whitespace-nowrap transition-colors",
        active
          ? "border-foreground/20 bg-foreground text-background"
          : "border-border bg-card text-muted-foreground hover:border-ring hover:text-foreground"
      )}
    >
      {children}
    </button>
  );
}

function EmptyCard({
  icon,
  title,
  text,
  children,
}: {
  icon: ReactNode;
  title: string;
  text: string;
  children?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-input bg-card px-5 py-14 text-center">
      <span className="flex size-[38px] items-center justify-center rounded-[10px] bg-muted text-muted-foreground">
        {icon}
      </span>
      <div className="text-sm font-medium">{title}</div>
      <div className="max-w-md text-[12.5px] leading-[1.5] text-muted-foreground">{text}</div>
      {children && <div className="mt-1 flex items-center gap-2">{children}</div>}
    </div>
  );
}

function AsideCard({ title, children, action }: { title: string; children: ReactNode; action?: ReactNode }) {
  return (
    <div data-spotlight className="relative isolate rounded-2xl border border-border bg-card p-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <span className="text-[13px] font-semibold tracking-[0.02em] text-muted-foreground">{title}</span>
        {action}
      </div>
      {children}
    </div>
  );
}

function SummaryCard({
  unread,
  groups,
  active,
  onPick,
}: {
  unread: number;
  groups: NotificationGroup[];
  active: string;
  onPick: (id: string) => void;
}) {
  const t = useT();
  return (
    <AsideCard title={t("notifications.summary_title")}>
      <div className="flex items-baseline gap-2">
        <AnimatedNumber value={unread} className="text-[30px] leading-none font-semibold tracking-[-0.03em] tabular-nums" />
        <span className="text-[13px] text-muted-foreground">{t("notifications.summary_unread")}</span>
      </div>
      {groups.length > 0 ? (
        <div className="mt-4 flex flex-col gap-0.5">
          {groups.map((g) => {
            const Icon = groupIcon(g.id);
            const share = g.count > 0 ? Math.round((g.unread / g.count) * 100) : 0;
            return (
              <button
                key={g.id}
                type="button"
                onClick={() => onPick(g.id)}
                aria-pressed={active === g.id}
                className={cn(
                  "-mx-2 flex items-center gap-2.5 rounded-lg px-2 py-2 text-left transition-colors",
                  active === g.id ? "bg-accent" : "hover:bg-accent/60"
                )}
              >
                <Icon className="size-4 shrink-0 text-muted-foreground" />
                <span className="min-w-0 flex-1">
                  <span className="flex items-center justify-between gap-2 text-[13px]">
                    <span className="truncate">{g.label}</span>
                    <span className="font-mono text-[11.5px] text-muted-foreground tabular-nums">
                      {g.unread > 0 ? `${g.unread} / ${g.count}` : g.count}
                    </span>
                  </span>
                  <span className="mt-1.5 block h-1 overflow-hidden rounded-full bg-muted">
                    <span
                      className="vx-grow-x block h-full rounded-full bg-primary"
                      style={{ width: `${share}%` }}
                    />
                  </span>
                </span>
              </button>
            );
          })}
        </div>
      ) : (
        <p className="mt-3 text-[12.5px] text-muted-foreground">{t("notifications.summary_empty")}</p>
      )}
    </AsideCard>
  );
}

type ChannelState = "on" | "off" | "missing";

function channelStates(prefs: NotificationPrefs) {
  const c = prefs.channels;
  const telegram: ChannelState = !c.telegram_chat_id ? "missing" : c.telegram ? "on" : "off";
  const discord: ChannelState = !c.discord_webhook ? "missing" : c.discord ? "on" : "off";
  return [
    { id: "email", label: "Email", icon: "ri-mail-line", state: (c.email ? "on" : "off") as ChannelState },
    { id: "telegram", label: "Telegram", icon: "ri-telegram-line", state: telegram },
    { id: "discord", label: "Discord", icon: "ri-discord-line", state: discord },
  ];
}

function ChannelsCard({ prefs, loading }: { prefs?: NotificationPrefs; loading: boolean }) {
  const t = useT();
  return (
    <AsideCard
      title={t("notifications.channels_title")}
      action={
        <Link
          href="/settings?tab=notifications"
          className="text-[12.5px] font-medium text-primary transition-opacity hover:opacity-80"
        >
          {t("notifications.channels_configure")}
        </Link>
      }
    >
      {loading || !prefs ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-8 rounded-lg" />
          <Skeleton className="h-8 rounded-lg" />
          <Skeleton className="h-8 rounded-lg" />
        </div>
      ) : (
        <div className="flex flex-col gap-1">
          {channelStates(prefs).map((channel) => (
            <div key={channel.id} className="flex items-center gap-2.5 py-1.5">
              <span className="flex size-8 items-center justify-center rounded-lg bg-muted text-[15px] text-muted-foreground">
                <i className={channel.icon} />
              </span>
              <span className="flex-1 text-[13px]">{channel.label}</span>
              <span
                className={cn(
                  "inline-flex items-center gap-1.5 text-[12px]",
                  channel.state === "on" ? "text-[var(--vx-ok)]" : "text-muted-foreground"
                )}
              >
                <span
                  className={cn(
                    "size-1.5 rounded-full",
                    channel.state === "on" ? "bg-[var(--vx-ok)]" : "bg-muted-foreground/40"
                  )}
                />
                {t(`notifications.channel_state.${channel.state}`)}
              </span>
            </div>
          ))}
        </div>
      )}
    </AsideCard>
  );
}

function BrowserCard() {
  const t = useT();
  const state = useBrowserNotifications();
  if (!state.supported) return null;
  return (
    <AsideCard title={t("notifications.browser_title")}>
      <div className="flex items-start gap-3">
        <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
          <MonitorSmartphone className="size-4" />
        </span>
        <p className="flex-1 text-[12.5px] leading-[1.5] text-muted-foreground">
          {state.permission === "denied"
            ? t("notifications.browser_denied")
            : t("notifications.browser_text")}
        </p>
        <Switch
          checked={state.enabled}
          disabled={state.permission === "denied"}
          aria-label={t("notifications.browser_title")}
          onCheckedChange={(on) => {
            if (on) void enableBrowserNotifications();
            else setBrowserNotificationsEnabled(false);
          }}
        />
      </div>
    </AsideCard>
  );
}
