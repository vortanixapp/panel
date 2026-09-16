"use client";

import { useState, type ReactNode } from "react";
import { ArrowUpRight, Check, CircleDot, Trash2 } from "lucide-react";
import type { PanelNotification } from "@/lib/api";
import { t } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import {
  TONE_ACCENT,
  TONE_ICON,
  fullTime,
  relativeTime,
} from "@/components/notifications/notification-utils";

type NotificationItemProps = {
  item: PanelNotification;
  onOpen: (item: PanelNotification) => void;
  onToggleRead: (item: PanelNotification) => void;
  onRemove: (item: PanelNotification) => void;
};

function ItemAction({
  label,
  onClick,
  children,
}: {
  label: string;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      onClick={(event) => {
        event.stopPropagation();
        onClick();
      }}
      className="flex size-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
    >
      {children}
    </button>
  );
}

function ItemActions({ item, onToggleRead, onRemove }: Omit<NotificationItemProps, "onOpen">) {
  return (
    <>
      <ItemAction
        label={item.unread ? t("notifications.mark_read") : t("notifications.mark_unread")}
        onClick={() => onToggleRead(item)}
      >
        {item.unread ? <Check className="size-3.5" /> : <CircleDot className="size-3.5" />}
      </ItemAction>
      <ItemAction label={t("notifications.remove")} onClick={() => onRemove(item)}>
        <Trash2 className="size-3.5" />
      </ItemAction>
    </>
  );
}

export function NotificationCompactItem({ item, onOpen, onToggleRead, onRemove }: NotificationItemProps) {
  const hasLink = Boolean(item.href && item.action);
  return (
    <div className="group relative flex gap-3 px-4 py-3 transition-colors hover:bg-accent/60 focus-within:bg-accent/60">
      <button
        type="button"
        onClick={() => onOpen(item)}
        aria-label={item.title}
        className="absolute inset-0 z-0 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
      />
      {item.unread && (
        <span
          aria-hidden
          className="pointer-events-none absolute top-[26px] left-1.5 size-1.5 rounded-full bg-primary"
        />
      )}
      <span
        className={cn(
          "pointer-events-none relative flex size-9 shrink-0 items-center justify-center rounded-[10px]",
          TONE_ICON[item.tone]
        )}
      >
        <i className={cn(item.icon, "text-[17px]")} />
      </span>
      <div className="pointer-events-none relative min-w-0 flex-1">
        <div className="flex items-start gap-2">
          <p
            className={cn(
              "min-w-0 flex-1 text-[13.5px] leading-snug",
              item.unread ? "font-semibold text-foreground" : "font-medium text-foreground/75"
            )}
          >
            {item.title}
          </p>
          <time
            dateTime={item.created_at}
            className="shrink-0 pt-px text-[11.5px] text-muted-foreground"
          >
            {relativeTime(item.created_at)}
          </time>
        </div>
        {item.body && (
          <p className="mt-0.5 line-clamp-2 text-[12.5px] leading-[1.45] text-muted-foreground">
            {item.body.replace(/\s*\n\s*/g, " ")}
          </p>
        )}
        <div className="mt-1.5 flex items-center gap-1.5 text-[11.5px] text-muted-foreground">
          <span className="truncate">{item.category}</span>
          {hasLink && (
            <>
              <span aria-hidden>·</span>
              <span className="truncate font-medium text-primary">{item.action}</span>
            </>
          )}
          <span className="pointer-events-auto relative z-10 ms-auto hidden gap-0.5 [@media(hover:none)]:flex">
            <ItemActions item={item} onToggleRead={onToggleRead} onRemove={onRemove} />
          </span>
        </div>
      </div>
      <div className="absolute top-2 right-2 z-10 flex gap-0.5 rounded-lg border border-border bg-popover p-0.5 opacity-0 shadow-sm transition-opacity group-hover:opacity-100 group-focus-within:opacity-100 [@media(hover:none)]:hidden">
        <ItemActions item={item} onToggleRead={onToggleRead} onRemove={onRemove} />
      </div>
    </div>
  );
}

const LONG_BODY = 240;

export function NotificationFullItem({ item, onOpen, onToggleRead, onRemove }: NotificationItemProps) {
  const [expanded, setExpanded] = useState(false);
  const long = item.body.length > LONG_BODY || item.body.split("\n").length > 4;
  const hasLink = Boolean(item.href && item.action);

  return (
    <div
      className={cn(
        "group relative flex gap-3.5 px-4 py-4 transition-colors sm:px-5",
        item.unread ? "bg-accent/35" : "hover:bg-accent/30"
      )}
    >
      {item.unread && (
        <span
          aria-hidden
          className={cn("absolute inset-y-3 left-0 w-[3px] rounded-r-full", TONE_ACCENT[item.tone])}
        />
      )}
      <span
        className={cn(
          "flex size-10 shrink-0 items-center justify-center rounded-xl",
          TONE_ICON[item.tone]
        )}
      >
        <i className={cn(item.icon, "text-[19px]")} />
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1 pe-16 sm:pe-0">
          <h3
            className={cn(
              "text-[14.5px] leading-snug",
              item.unread ? "font-semibold text-foreground" : "font-medium text-foreground/80"
            )}
          >
            {item.title}
          </h3>
          {item.unread && (
            <span className="rounded-full bg-primary/10 px-1.5 py-px text-[10.5px] font-semibold text-primary">
              {t("notifications.new")}
            </span>
          )}
        </div>
        {item.body && (
          <div className="mt-1">
            <p
              className={cn(
                "text-[13px] leading-[1.55] text-pretty whitespace-pre-line text-muted-foreground",
                long && !expanded && "line-clamp-3"
              )}
            >
              {item.body}
            </p>
            {long && (
              <button
                type="button"
                onClick={() => setExpanded((v) => !v)}
                className="mt-1 text-[12px] font-medium text-foreground/80 transition-colors hover:text-foreground"
              >
                {expanded ? t("notifications.collapse") : t("notifications.expand")}
              </button>
            )}
          </div>
        )}
        <div className="mt-2.5 flex flex-wrap items-center gap-x-3 gap-y-2 text-[12px] text-muted-foreground">
          <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground">
            {item.category}
          </span>
          <time dateTime={item.created_at} title={fullTime(item.created_at)}>
            {relativeTime(item.created_at)}
          </time>
          {hasLink && (
            <button
              type="button"
              onClick={() => onOpen(item)}
              className="group/link inline-flex items-center gap-1 font-medium text-primary"
            >
              {item.action}
              <ArrowUpRight className="size-3.5 transition-transform duration-200 group-hover/link:translate-x-0.5 group-hover/link:-translate-y-0.5" />
            </button>
          )}
        </div>
      </div>
      <div className="absolute top-3 right-3 flex gap-0.5 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100 sm:static sm:self-start [@media(hover:none)]:opacity-100">
        <ItemActions item={item} onToggleRead={onToggleRead} onRemove={onRemove} />
      </div>
    </div>
  );
}
