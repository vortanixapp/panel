"use client";

import { localeTag, t } from "@/lib/i18n";
import { cn } from "@/lib/utils";

/**
 * Общий словарь поддержки: статусы, приоритеты и отделы.
 *
 * В макете статусы названы «Открыт», «В работе», «Ждёт вас», «Закрыт», а в базе
 * они хранятся как open / pending / answered / closed. Раскладка здесь одна на
 * все экраны: клиентский список, переписка и очередь админа обязаны называть
 * одно состояние одинаково, иначе клиент и оператор говорят о разном.
 */

export type SupportStatus = "open" | "pending" | "answered" | "closed";

type StatusMeta = { label: string; token: string; hint: string };

// Раскладка читается на уровне модуля, поэтому храним ключи: готовые подписи
// застыли бы на языке, который стоял в момент загрузки страницы.
const STATUS: Record<string, { labelKey: string; token: string; hintKey: string }> = {
  // «Открыт» — обращение создано, оператор его ещё не взял.
  open: {
    labelKey: "support.status.open",
    token: "--vx-warn",
    hintKey: "support.status.open_hint",
  },
  // «В работе» — оператор назначен и разбирается.
  pending: {
    labelKey: "support.status.pending",
    token: "--vx-info",
    hintKey: "support.status.pending_hint",
  },
  // «Ждёт вас» — поддержка ответила, ход за клиентом.
  answered: {
    labelKey: "support.status.answered",
    token: "--vx-ok",
    hintKey: "support.status.answered_hint",
  },
  closed: {
    labelKey: "support.status.closed",
    token: "--vx-ink-ghost",
    hintKey: "support.status.closed_hint",
  },
};

export function supportStatusMeta(status: string): StatusMeta {
  const meta = STATUS[status];
  if (!meta) return { label: status || "—", token: "--vx-ink-ghost", hint: "" };
  return { label: t(meta.labelKey), token: meta.token, hint: t(meta.hintKey) };
}

export function SupportStatusPill({
  status,
  className,
}: {
  status: string;
  className?: string;
}) {
  const meta = supportStatusMeta(status);
  return (
    <span
      className={cn(
        "rounded-full border px-[9px] py-[3px] text-[11.5px] whitespace-nowrap",
        className
      )}
      style={{
        color: `var(${meta.token})`,
        background: `color-mix(in srgb, var(${meta.token}) 12%, transparent)`,
        borderColor: `color-mix(in srgb, var(${meta.token}) 30%, transparent)`,
      }}
    >
      {meta.label}
    </span>
  );
}

// label — вычисляемое свойство, а не строка: список читается на уровне модуля,
// и готовый текст застыл бы на языке, который стоял при загрузке страницы.
// Форму {id, label, token} менять нельзя — её читают экраны очереди оператора.
export const SUPPORT_PRIORITIES = [
  {
    id: "low",
    token: "--vx-ink-ghost",
    get label() {
      return t("support.priority.low");
    },
  },
  {
    id: "normal",
    token: "--vx-info",
    get label() {
      return t("support.priority.normal");
    },
  },
  {
    id: "high",
    token: "--vx-warn",
    get label() {
      return t("support.priority.high");
    },
  },
  {
    id: "urgent",
    token: "--vx-danger",
    get label() {
      return t("support.priority.urgent");
    },
  },
] as const;

export function supportPriorityMeta(priority: string | undefined) {
  return (
    SUPPORT_PRIORITIES.find((p) => p.id === priority) ?? SUPPORT_PRIORITIES[1]
  );
}

/** Точка приоритета. Цветом, а не словом: в строке очереди слово «Обычный»
 *  занимает место, которое нужнее теме обращения. */
export function PriorityDot({ priority }: { priority: string | undefined }) {
  const meta = supportPriorityMeta(priority);
  return (
    <span
      className="inline-block h-2 w-2 flex-none rounded-full"
      style={{ background: `var(${meta.token})` }}
      title={t("support.priority.title", { label: meta.label })}
    />
  );
}

/** Отделы приходят с сервера и настраиваются для каждой панели. Этот список —
 *  только запасной, на случай недоступной формы: без него селект был бы пуст,
 *  и обращение нельзя было бы создать вовсе. */
// name — вычисляемое свойство по той же причине, что и у приоритетов:
// список читается на уровне модуля, а язык может смениться позже.
export const FALLBACK_DEPARTMENTS = [
  {
    id: "billing",
    get name() {
      return t("support.department.billing");
    },
  },
  {
    id: "technical",
    get name() {
      return t("support.department.technical");
    },
  },
  {
    id: "hosting",
    get name() {
      return t("support.department.hosting");
    },
  },
  {
    id: "other",
    get name() {
      return t("support.department.other");
    },
  },
];

export function departmentName(
  id: string | undefined,
  departments: { id: string; name: string }[]
): string {
  if (!id) return t("support.department.other");
  return departments.find((d) => d.id === id)?.name ?? id;
}

/** «2 ч назад», «3 дн назад». Для списка это полезнее точной даты: важно, как
 *  давно обращение ждёт, а не в какую минуту его создали. */
export function timeAgo(value: string | null | undefined): string {
  if (!value) return "—";
  const ts = new Date(value).getTime();
  if (Number.isNaN(ts)) return "—";

  const min = Math.floor((Date.now() - ts) / 60000);
  if (min < 1) return t("support.time.just_now");
  if (min < 60) return t("support.time.minutes_ago", { n: min });

  const hours = Math.floor(min / 60);
  if (hours < 24) return t("support.time.hours_ago", { n: hours });

  const days = Math.floor(hours / 24);
  if (days < 30) return t("support.time.days_ago", { n: days });

  return new Date(ts).toLocaleDateString(localeTag(), { day: "2-digit", month: "short" });
}

/** Сколько обращение уже ждёт. Очередь админа сортируется глазами именно по
 *  этому числу, поэтому оно же решает, подсвечивать ли строку как просроченную. */
export function waitingFor(value: string | null | undefined): {
  label: string;
  overdue: boolean;
} {
  if (!value) return { label: "—", overdue: false };
  const ts = new Date(value).getTime();
  if (Number.isNaN(ts)) return { label: "—", overdue: false };

  const min = Math.floor((Date.now() - ts) / 60000);
  const overdue = min >= 24 * 60;

  if (min < 60) return { label: t("support.time.minutes", { n: min }), overdue };
  const hours = Math.floor(min / 60);
  if (hours < 24) {
    return {
      label: t("support.time.hours_minutes", { h: hours, m: min % 60 }),
      overdue,
    };
  }
  const days = Math.floor(hours / 24);
  return { label: t("support.time.days", { n: days }), overdue };
}

/** «18 мин», «1 ч 20 мин». Часы отдельно от минут: «80 мин» читается хуже, а
 *  в панели времени ответа эта цифра — единственное, ради чего туда смотрят. */
export function formatMinutes(minutes: number): string {
  if (minutes < 1) return t("support.time.less_than_minute");
  if (minutes < 60) return t("support.time.minutes", { n: minutes });
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  return rest === 0
    ? t("support.time.hours", { n: hours })
    : t("support.time.hours_minutes", { h: hours, m: rest });
}
