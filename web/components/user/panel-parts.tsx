"use client";

import type { ReactNode } from "react";
import { t } from "@/lib/i18n";
import { cn } from "@/lib/utils";

/**
 * Общие детали страниц кабинета: полоса метрик, статус-пилюля, полоса
 * заполнения и классы кнопок с полями. Из них собраны и хостинг, и поддержка —
 * держать копию на раздел значило бы разъехаться на первой же правке.
 *
 * Цвета берутся токенами, а не значениями из макета. Макет нарисован в тёмной
 * теме, и его палитра — это ровно тёмные токены проекта: #1c1d1f это --border,
 * #0c0d0f это --card, #f1f2ed это --primary, #8b8c8e это --muted-foreground.
 * Поэтому токены дают в тёмной теме попадание один в один, а светлая тема,
 * которой в макете нет, продолжает работать.
 */

/** Полоса метрик: карточки, сшитые линией в один блок.
 *
 * Приём из макета: контейнер красится цветом границы, а зазор в один пиксель
 * между карточками показывает его насквозь. Так разделители не удваиваются на
 * переносе строки, чего не добиться отдельными border у соседей. */
export function StatStrip({ children }: { children: ReactNode }) {
  return (
    <div className="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-px overflow-hidden rounded-[14px] border border-border bg-border">
      {children}
    </div>
  );
}

export function StatCell({
  label,
  value,
  tone,
  mono,
  icon,
  note,
}: {
  label: string;
  value: ReactNode;
  tone?: "warn";
  mono?: boolean;
  icon?: ReactNode;
  note?: string;
}) {
  return (
    <div className="bg-card px-5 py-[18px]">
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        {icon}
        {label}
      </div>
      <div
        className={cn(
          "mt-[7px] text-[22px] leading-none font-bold tracking-[-0.02em]",
          mono && "font-mono text-xl",
          tone === "warn" ? "text-[var(--vx-warn)]" : "text-foreground"
        )}
      >
        {value}
      </div>
      {note && <div className="mt-1 text-[11.5px] text-muted-foreground/70">{note}</div>}
    </div>
  );
}

/** Статус аккаунта.
 *
 * В макете состояний два — «в порядке» и «истекает». В данных их больше, и
 * сваливать «приостановлен» в ту же жёлтую пилюлю, что и «истекает», нельзя:
 * это разные новости для владельца. Ошибочные состояния получают красный. */
export function StatusPill({ status, label }: { status: string; label: string }) {
  const tone =
    status === "suspended" || status === "terminated" || status === "error"
      ? "danger"
      : status === "active"
        ? "info"
        : "warn";

  return (
    <span
      className="rounded-full border px-[9px] py-[3px] text-[11.5px] whitespace-nowrap"
      style={{
        color: `var(--vx-${tone})`,
        background: `var(--vx-${tone}-tint)`,
        borderColor: `color-mix(in srgb, var(--vx-${tone}) 30%, transparent)`,
      }}
    >
      {label}
    </span>
  );
}

/** Полоса заполнения.
 *
 * pct = null означает «доля неизвестна»: полосу тогда не рисуем вовсе. Пустая
 * или полная полоса читалась бы как измеренное значение, а по части лимитов
 * (занятое место на диске) сервер не отдаёт ничего. */
export function UsageBar({
  label,
  value,
  pct,
}: {
  label: string;
  value: ReactNode;
  pct: number | null;
}) {
  const warn = pct != null && pct > 85;

  return (
    <div>
      <div className="flex justify-between text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span className="font-mono text-foreground">{value}</span>
      </div>
      {pct == null ? (
        <div className="mt-2 h-1 rounded-full bg-border/60" />
      ) : (
        <div className="mt-2 h-1 overflow-hidden rounded-full bg-border">
          <span
            className={cn("block h-1", warn ? "bg-[var(--vx-warn)]" : "bg-primary")}
            style={{ width: `${Math.min(100, Math.max(0, pct))}%` }}
          />
        </div>
      )}
    </div>
  );
}

/** Кнопки макета: светлая заливка для главного действия, обводка для прочих. */
export const btnPrimary =
  "inline-flex items-center justify-center gap-2 rounded-[9px] bg-primary px-4 text-[13px] font-semibold text-primary-foreground transition-colors hover:bg-foreground disabled:opacity-50";

export const btnGhost =
  "inline-flex items-center justify-center gap-[7px] rounded-[9px] border border-input bg-transparent px-4 text-[12.5px] text-foreground transition-colors hover:bg-accent disabled:opacity-50";

export const fieldClass =
  "h-[38px] w-full rounded-[9px] border border-input bg-background px-3 text-[13px] text-foreground outline-none transition-colors placeholder:text-muted-foreground/70 focus:border-ring";

/** Сколько осталось до даты. null, если дата не задана или уже прошла. */
export function daysLeft(expiresAt: string | null | undefined): number | null {
  if (!expiresAt) return null;
  const ts = new Date(expiresAt).getTime();
  if (Number.isNaN(ts)) return null;
  const days = Math.ceil((ts - Date.now()) / 86_400_000);
  return days > 0 ? days : null;
}

export function pluralDays(n: number): string {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return t("dashboard.days_one", { count: n });
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) {
    return t("dashboard.days_few", { count: n });
  }
  return t("dashboard.days_many", { count: n });
}

/** Доля использования лимита. null, если лимит безлимитный или неизвестен —
 * делить на «безлимит» бессмысленно, полосе тогда нечего показывать. */
export function usagePct(used: number, limit: number | null | undefined): number | null {
  if (limit == null || limit <= 0) return null;
  return Math.round((used / limit) * 100);
}
