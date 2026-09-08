"use client";

import { useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import {
  CategoryBadge,
  Chip,
  EventIcon,
  ListEmpty,
  MON_BTN,
  MON_CARD,
  formatDateTime,
} from "@/components/user/account/shared";
import {
  clearReadNotifications,
  deleteNotification,
  fetchNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  updateNotificationChannels,
  type Notification,
  type NotificationChannels,
} from "@/lib/api";
import { t } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const PAGE = 30;

/** Фильтр «Все» и «Непрочитанные» — не разделы, поэтому идут отдельно.
 *  Подписи разделов приходят с сервера готовым текстом, а свои держим ключами:
 *  готовая строка застыла бы на языке, который стоял при загрузке страницы. */
type Filter = { id: string; label?: string; labelKey?: string; unreadOnly?: boolean };

const BASE_FILTERS: Filter[] = [
  { id: "", labelKey: "common.all" },
  { id: "unread", labelKey: "dashboard.notifications.filter_unread", unreadOnly: true },
];

export function NotificationsPageContent() {
  // Строки оповещений зовут t() напрямую, здесь хук нужен ради подписки:
  // без него страница не перерисуется при смене языка.
  useT();
  const queryClient = useQueryClient();
  const [filter, setFilter] = useState<Filter>(BASE_FILTERS[0]);
  const [offset, setOffset] = useState(0);

  // Фильтр и страница входят в ключ запроса: сервер отдаёт уже отобранное, и
  // менять выборку в браузере больше не нужно.
  const notifications = useQuery({
    queryKey: ["notifications", filter.id, offset],
    queryFn: () =>
      fetchNotifications({
        group: filter.unreadOnly ? undefined : filter.id || undefined,
        unread: filter.unreadOnly,
        offset,
        limit: PAGE,
      }),
    refetchInterval: 60_000,
    refetchIntervalInBackground: false,
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["notifications"] });
    void queryClient.invalidateQueries({ queryKey: ["notifications-unread"] });
  };

  const markOne = useMutation({ mutationFn: markNotificationRead, onSuccess: invalidate });
  const removeOne = useMutation({ mutationFn: deleteNotification, onSuccess: invalidate });

  const markAll = useMutation({
    mutationFn: markAllNotificationsRead,
    onSuccess: () => {
      invalidate();
      toast.success(t("dashboard.notifications.marked_all"));
    },
  });

  const clearRead = useMutation({
    mutationFn: clearReadNotifications,
    onSuccess: (res) => {
      invalidate();
      setOffset(0);
      toast.success(
        res.deleted > 0
          ? t("dashboard.notifications.cleared", { count: res.deleted })
          : t("dashboard.notifications.clear_empty")
      );
    },
    onError: () => toast.error(t("dashboard.notifications.clear_failed")),
  });

  const channels = useMutation({
    mutationFn: (payload: Partial<NotificationChannels>) => updateNotificationChannels(payload),
    onSuccess: () => {
      invalidate();
      toast.success(t("dashboard.notifications.channels_updated"));
    },
    onError: () => toast.error(t("dashboard.notifications.channels_failed")),
  });

  const data = notifications.data;
  const list = data?.notifications ?? [];
  const unread = data?.unread ?? 0;
  const filters: Filter[] = [
    ...BASE_FILTERS,
    ...(data?.groups ?? []).map((g) => ({ id: g.id, label: g.label })),
  ];

  const pick = (f: Filter) => {
    setFilter(f);
    setOffset(0);
  };

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-4">
        <div className="relative overflow-hidden rounded-[16px] border border-[var(--vx-border)] bg-[var(--vx-elevated)] p-[22px]">
          <div className="relative flex flex-wrap items-center gap-4">
            <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-[14px] bg-[var(--vx-fg-strong)] text-[var(--vx-on-fill)]">
              <i className="ri-notification-3-line text-[22px]" />
            </span>
            <div className="min-w-[200px] flex-1">
              <h1 className="text-[22px] font-bold tracking-[-0.01em]">
                {t("dashboard.notifications.title")}
              </h1>
              <p className="mt-1 text-[13px] text-[var(--vx-muted)]">
                {t("dashboard.notifications.subtitle")}
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                className={MON_BTN}
                onClick={() => void notifications.refetch()}
                disabled={notifications.isFetching}
              >
                <i
                  className={cn(
                    "ri-refresh-line text-[15px]",
                    notifications.isFetching && "animate-spin"
                  )}
                />
                {t("common.refresh")}
              </button>
              <button
                type="button"
                onClick={() => markAll.mutate()}
                disabled={unread === 0 || markAll.isPending}
                className={cn(
                  "flex h-9 items-center gap-1.5 rounded-[8px] px-3.5 text-[13px] font-semibold transition-colors",
                  unread > 0
                    ? "cursor-pointer bg-[var(--vx-fg-strong)] text-[var(--vx-on-fill)] hover:bg-white"
                    : "cursor-default bg-[var(--vx-tint)] text-[var(--vx-muted)]"
                )}
              >
                <i className="ri-check-double-line text-[15px]" />
                {t("dashboard.notifications.mark_all")}
              </button>
            </div>
          </div>
        </div>

        <div className={cn("overflow-hidden", MON_CARD)}>
          <div className="flex flex-wrap items-center gap-2.5 border-b border-[var(--vx-border)] px-4 py-3">
            <span className="text-[13px] text-[var(--vx-muted)]">
              {t("dashboard.notifications.unread_label")}{" "}
              <span className="font-semibold text-[var(--vx-fg)]">{unread}</span>
            </span>
            <button
              type="button"
              onClick={() => clearRead.mutate()}
              disabled={clearRead.isPending}
              className="text-[12px] text-[var(--vx-muted)] transition-colors hover:text-[var(--vx-fg)] disabled:opacity-55"
            >
              {t("dashboard.notifications.clear_read")}
            </button>
            <div className="ml-auto flex flex-wrap gap-1 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-bg)] p-[3px]">
              {filters.map((f) => (
                <Chip key={f.id || "all"} active={filter.id === f.id} onClick={() => pick(f)}>
                  {f.labelKey ? t(f.labelKey) : f.label}
                </Chip>
              ))}
            </div>
          </div>

          {notifications.isLoading ? (
            <div className="p-4">
              <Skeleton className="h-72 w-full rounded-[12px]" />
            </div>
          ) : notifications.isError || data?.error ? (
            <ListEmpty
              icon="ri-error-warning-line"
              title={t("dashboard.notifications.load_failed_title")}
              text={t("dashboard.notifications.load_failed_text")}
            />
          ) : list.length === 0 ? (
            <ListEmpty
              icon="ri-notification-off-line"
              title={
                filter.unreadOnly
                  ? t("dashboard.notifications.all_read_title")
                  : t("dashboard.notifications.empty_title")
              }
              text={
                filter.unreadOnly
                  ? t("dashboard.notifications.all_read_text")
                  : t("dashboard.notifications.empty_text")
              }
            />
          ) : (
            <>
              {list.map((n) => (
                <NotificationRow
                  key={n.id}
                  item={n}
                  onRead={() => markOne.mutate(n.id)}
                  onDelete={() => removeOne.mutate(n.id)}
                  busy={markOne.isPending || removeOne.isPending}
                />
              ))}
              {(data?.has_more || offset > 0) && (
                <div className="flex items-center justify-between gap-3 px-[18px] py-3.5">
                  <button
                    type="button"
                    className={MON_BTN}
                    disabled={offset === 0}
                    onClick={() => setOffset(Math.max(0, offset - PAGE))}
                  >
                    <i className="ri-arrow-left-s-line text-[15px]" />
                    {t("common.back")}
                  </button>
                  <button
                    type="button"
                    className={MON_BTN}
                    disabled={!data?.has_more}
                    onClick={() => setOffset(offset + PAGE)}
                  >
                    {t("dashboard.notifications.more")}
                    <i className="ri-arrow-right-s-line text-[15px]" />
                  </button>
                </div>
              )}
            </>
          )}
        </div>

        <div className={cn("flex flex-wrap items-center gap-3.5 px-[18px] py-4", MON_CARD)}>
          <div className="min-w-[220px] flex-1">
            <div className="text-[13px] font-semibold">
              {t("dashboard.notifications.channels_title")}
            </div>
            <p className="mt-1 text-[12px] text-[var(--vx-muted)]">
              {t("dashboard.notifications.channels_hint")}
            </p>
          </div>

          <ChannelToggle
            icon="ri-mail-line"
            label="E-mail"
            enabled={data?.channels?.email ?? true}
            disabled={channels.isPending}
            onToggle={(next) => channels.mutate({ email: next })}
          />
          {/* Telegram и Discord включаются только там, где можно сразу указать
              адрес: тумблер без него оставлял клиента в уверенности, что он
              подписался, а доставки не было. */}
          <ChannelState
            icon="ri-telegram-line"
            label="Telegram"
            enabled={data?.channels?.telegram ?? false}
            configured={Boolean(data?.channels?.telegram_chat_id)}
          />
          <ChannelState
            icon="ri-discord-line"
            label="Discord"
            enabled={data?.channels?.discord ?? false}
            configured={Boolean(data?.channels?.discord_webhook)}
          />

          <Link
            href="/settings?tab=notifications"
            className="text-[12px] text-[var(--vx-muted)] transition-colors hover:text-white"
          >
            {t("dashboard.notifications.configure")}
          </Link>
        </div>
      </div>
    </PageShell>
  );
}

function NotificationRow({
  item,
  onRead,
  onDelete,
  busy,
}: {
  item: Notification;
  onRead: () => void;
  onDelete: () => void;
  busy: boolean;
}) {
  return (
    <div
      className={cn(
        "group flex items-start gap-3.5 border-b border-[var(--vx-divider)] px-[18px] py-4 last:border-b-0",
        item.unread && "bg-[var(--vx-veil)]"
      )}
    >
      <EventIcon icon={item.icon} tone={item.tone} size={36} className="rounded-[10px]" />

      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-[14px] font-semibold">{item.title}</span>
          {item.unread && <span className="h-[7px] w-[7px] rounded-full bg-[var(--vx-danger)]" />}
          <CategoryBadge>{item.category}</CategoryBadge>
        </div>
        {item.body && (
          <p className="mt-1.5 text-[13px] leading-[1.5] text-pretty whitespace-pre-line text-[var(--vx-muted)]">
            {item.body}
          </p>
        )}
        <div className="mt-2 flex flex-wrap items-center gap-3">
          <span className="font-mono text-[11px] text-[var(--vx-faint)]">
            {formatDateTime(item.created_at)}
          </span>
          {item.action && item.href && (
            // Ссылка абсолютная — то же уведомление уходит письмом и в Telegram,
            // где относительный путь никуда не ведёт.
            <a
              href={item.href}
              className="text-[12px] text-[var(--vx-dim)] transition-colors hover:text-white"
            >
              {item.action} →
            </a>
          )}
        </div>
      </div>

      <div className="flex shrink-0 items-center gap-1.5">
        {item.unread && (
          <button
            type="button"
            onClick={onRead}
            disabled={busy}
            className="flex items-center gap-1.5 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-inset)] px-3 py-[7px] text-[12px] transition-colors hover:bg-[var(--vx-tint)] disabled:opacity-55"
          >
            <i className="ri-check-line text-[13px]" />
            {t("dashboard.notifications.read")}
          </button>
        )}
        <button
          type="button"
          onClick={onDelete}
          disabled={busy}
          title={t("dashboard.notifications.remove")}
          aria-label={t("dashboard.notifications.remove_aria")}
          className="rounded-[8px] px-2 py-[7px] text-[13px] text-[var(--vx-faint)] opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100 disabled:opacity-40"
        >
          <i className="ri-close-line" />
        </button>
      </div>
    </div>
  );
}

function ChannelToggle({
  icon,
  label,
  enabled,
  disabled,
  onToggle,
}: {
  icon: string;
  label: string;
  enabled: boolean;
  disabled?: boolean;
  onToggle: (next: boolean) => void;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={() => onToggle(!enabled)}
      className={cn(
        "flex items-center gap-2 rounded-[10px] border px-3 py-2 text-[12.5px] transition-colors disabled:opacity-55",
        enabled
          ? "border-[var(--vx-border-strong)] bg-[var(--vx-tint)] text-[var(--vx-fg)]"
          : "border-[var(--vx-border)] text-[var(--vx-muted)] hover:text-[var(--vx-fg)]"
      )}
    >
      <i className={cn(icon, "text-[15px]")} />
      {label}
      <i className={cn(enabled ? "ri-check-line" : "ri-close-line", "text-[13px]")} />
    </button>
  );
}

/**
 * Канал, который нельзя включить одним нажатием: без адреса доставки он не
 * работает, поэтому здесь только состояние и ссылка на настройку.
 */
function ChannelState({
  icon,
  label,
  enabled,
  configured,
}: {
  icon: string;
  label: string;
  enabled: boolean;
  configured: boolean;
}) {
  const on = enabled && configured;
  return (
    <span
      title={
        enabled && !configured
          ? t("dashboard.notifications.channel_not_configured")
          : undefined
      }
      className={cn(
        "flex items-center gap-2 rounded-[10px] border px-3 py-2 text-[12.5px]",
        on
          ? "border-[var(--vx-border-strong)] bg-[var(--vx-tint)] text-[var(--vx-fg)]"
          : "border-[var(--vx-border)] text-[var(--vx-muted)]"
      )}
    >
      <i className={cn(icon, "text-[15px]")} />
      {label}
      {enabled && !configured ? (
        <i className="ri-error-warning-line text-[13px] text-[var(--vx-warn)]" />
      ) : (
        <i className={cn(on ? "ri-check-line" : "ri-close-line", "text-[13px]")} />
      )}
    </span>
  );
}
