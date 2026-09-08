"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowRight,
  BookOpen,
  Box,
  CheckCheck,
  CircleDot,
  CreditCard,
  Globe,
  Headphones,
  Loader,
  MessageSquare,
  Plus,
  Search,
  Server,
  User,
  type LucideIcon,
} from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchSupportCreateForm,
  fetchSupportTickets,
  type SupportTicket,
} from "@/lib/api";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import {
  StatCell,
  StatStrip,
  btnGhost,
  btnPrimary,
  fieldClass,
} from "@/components/user/panel-parts";
import {
  FALLBACK_DEPARTMENTS,
  PriorityDot,
  SupportStatusPill,
  departmentName,
  timeAgo,
} from "@/components/user/support/support-parts";

// Списки читаются на уровне модуля, поэтому храним ключи: готовые подписи
// застыли бы на языке, который стоял в момент загрузки страницы.
const FILTERS = [
  { id: "all", labelKey: "common.all" },
  { id: "open", labelKey: "support.filter.open" },
  { id: "pending", labelKey: "support.filter.pending" },
  { id: "answered", labelKey: "support.filter.answered" },
  { id: "closed", labelKey: "support.filter.closed" },
] as const;

type FilterId = (typeof FILTERS)[number]["id"];

/** Готовые запросы для пустой выдачи: подставляют слово в поиск, чтобы не
 *  гадать с формулировкой. Темы взяты те, с которыми обращаются чаще всего. */
const HINTS: { labelKey: string; queryKey: string; Icon: LucideIcon }[] = [
  { labelKey: "support.hint.server_label", queryKey: "support.hint.server_query", Icon: Server },
  { labelKey: "support.hint.billing_label", queryKey: "support.hint.billing_query", Icon: CreditCard },
  { labelKey: "support.hint.domain_label", queryKey: "support.hint.domain_query", Icon: Globe },
];

function TicketCard({
  ticket,
  departments,
}: {
  ticket: SupportTicket;
  departments: { id: string; name: string }[];
}) {
  return (
    <Link
      href={`/support/${ticket.id}`}
      className="flex flex-col gap-3 rounded-2xl border border-border bg-card px-5 py-[18px] transition-colors hover:border-[var(--vx-panel-hover-line)]"
    >
      <div className="flex flex-wrap items-center gap-[10px]">
        <PriorityDot priority={ticket.priority} />
        <span className="text-[15.5px] font-semibold">{ticket.subject}</span>
        <SupportStatusPill status={ticket.status} />
        <span className="ms-auto font-mono text-[11.5px] text-muted-foreground/70">
          #{String(ticket.id).slice(0, 8)}
        </span>
      </div>

      {ticket.last_message && (
        <div className="flex items-start gap-2.5 rounded-xl border border-border bg-background px-3 py-2.5">
          <span
            className={cn(
              "mt-0.5 flex h-5 w-5 flex-none items-center justify-center rounded-full",
              ticket.last_from_staff ? "bg-[var(--vx-info-tint)]" : "bg-muted"
            )}
          >
            {ticket.last_from_staff ? (
              <Headphones className="h-3 w-3" style={{ color: "var(--vx-info)" }} />
            ) : (
              <User className="h-3 w-3 text-muted-foreground" />
            )}
          </span>
          <span className="min-w-0">
            <span className="block text-[11.5px] text-muted-foreground">
              {ticket.last_from_staff ? t("support.chat.staff") : t("support.chat.you")} ·{" "}
              {timeAgo(ticket.last_message_at ?? ticket.created_at)}
            </span>
            <span className="mt-0.5 line-clamp-2 block text-[12.5px] text-[var(--vx-ink-dim)]">
              {ticket.last_message}
            </span>
          </span>
        </div>
      )}

      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[12.5px] text-muted-foreground">
        <span>{departmentName(ticket.category, departments)}</span>
        {ticket.service && (
          <>
            <span className="text-muted-foreground/50">·</span>
            <span className="flex items-center gap-1.5 truncate">
              <Box className="h-3.5 w-3.5 flex-none" />
              {ticket.service}
            </span>
          </>
        )}
        <span className="text-muted-foreground/50">·</span>
        <span className="flex items-center gap-1.5">
          <MessageSquare className="h-3.5 w-3.5" />
          {t("support.list.messages_short", { n: ticket.messages ?? 0 })}
        </span>
        <span className="text-muted-foreground/50">·</span>
        <span>{t("support.list.created_ago", { ago: timeAgo(ticket.created_at) })}</span>
        <span className="ms-auto hidden items-center gap-1.5 sm:flex">
          {t("common.open")}
          <ArrowRight className="h-3.5 w-3.5" />
        </span>
      </div>
    </Link>
  );
}

export function SupportPageContent() {
  // Карточка обращения зовёт t() напрямую, здесь хук нужен ради подписки:
  // без него страница не перерисуется при смене языка.
  useT();
  const [filter, setFilter] = useState<FilterId>("all");
  const [query, setQuery] = useState("");

  const { data: tickets = [], isLoading, isError } = useQuery({
    queryKey: queryKeys.supportTickets,
    queryFn: async () => (await fetchSupportTickets()).tickets,
  });

  // Отделы нужны, чтобы показать в карточке название, а не служебный код.
  const { data: form } = useQuery({
    queryKey: queryKeys.supportCreateForm,
    queryFn: fetchSupportCreateForm,
  });
  const departments = form?.departments?.length
    ? form.departments
    : FALLBACK_DEPARTMENTS;

  const counts = useMemo(() => {
    const by = (s: string) => tickets.filter((t) => t.status === s).length;
    return {
      all: tickets.length,
      open: by("open"),
      pending: by("pending"),
      answered: by("answered"),
      closed: by("closed"),
    };
  }, [tickets]);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return tickets.filter(
      (t) =>
        (filter === "all" || t.status === filter) &&
        (!q ||
          t.subject.toLowerCase().includes(q) ||
          String(t.id).toLowerCase().includes(q))
    );
  }, [tickets, filter, query]);

  if (isLoading) {
    return (
      <PageShell variant="user">
        <div className="flex flex-col gap-[22px]">
          <Skeleton className="h-12 w-72 rounded-xl" />
          <Skeleton className="h-[74px] rounded-[14px]" />
          <Skeleton className="h-24 rounded-2xl" />
          <Skeleton className="h-24 rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-[22px]">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div className="flex flex-col gap-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-[-0.02em]">
              {t("support.list.title")}
            </h1>
            <p className="text-[13.5px] text-muted-foreground">
              {t("support.list.subtitle")}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Link href="/support/kb" className={cn(btnGhost, "h-[38px]")}>
              <BookOpen className="h-3.5 w-3.5" />
              {t("support.kb.title")}
            </Link>
            <Link href="/support/create" className={cn(btnPrimary, "h-[38px] px-[18px]")}>
              <Plus className="h-[15px] w-[15px]" />
              {t("support.list.new_ticket")}
            </Link>
          </div>
        </div>

        {isError && (
          <p className="text-sm text-destructive">{t("support.list.load_failed")}</p>
        )}

        {tickets.length > 0 && (
          <StatStrip>
            <StatCell
              label={t("support.filter.open")}
              value={counts.open}
              tone={counts.open > 0 ? "warn" : undefined}
              icon={<CircleDot className="h-3.5 w-3.5" />}
              note={t("support.stats.open_note")}
            />
            <StatCell
              label={t("support.filter.pending")}
              value={counts.pending}
              icon={<Loader className="h-3.5 w-3.5" />}
              note={t("support.stats.pending_note")}
            />
            <StatCell
              label={t("support.filter.answered")}
              value={counts.answered}
              icon={<MessageSquare className="h-3.5 w-3.5" />}
              note={t("support.stats.answered_note")}
            />
            <StatCell
              label={t("support.filter.closed")}
              value={counts.closed}
              icon={<CheckCheck className="h-3.5 w-3.5" />}
              note={t("support.stats.closed_note")}
            />
          </StatStrip>
        )}

        {tickets.length > 0 && (
          <div className="flex flex-wrap items-center gap-2">
            <div className="no-scrollbar flex gap-1 overflow-x-auto rounded-xl border border-border bg-card p-1">
              {FILTERS.map((f) => (
                <button
                  key={f.id}
                  type="button"
                  onClick={() => setFilter(f.id)}
                  aria-pressed={filter === f.id}
                  className={cn(
                    "flex h-9 flex-none items-center gap-2 rounded-[9px] px-3.5 text-[13px] whitespace-nowrap transition-colors",
                    filter === f.id
                      ? "bg-accent font-medium text-accent-foreground"
                      : "text-muted-foreground hover:text-foreground"
                  )}
                >
                  {t(f.labelKey)}
                  <span className="font-mono text-[11.5px] text-muted-foreground/70">
                    {counts[f.id]}
                  </span>
                </button>
              ))}
            </div>

            <label className="relative ms-auto min-w-[220px] flex-1 sm:max-w-[280px] sm:flex-none">
              <Search className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <input
                type="search"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={t("support.list.search_placeholder")}
                className={cn(fieldClass, "ps-9")}
              />
            </label>
          </div>
        )}

        <div className="flex flex-col gap-2.5">
          {visible.map((t) => (
            <TicketCard key={t.id} ticket={t} departments={departments} />
          ))}
        </div>

        {/* Одно место и для пустого списка, и для пустой выборки — но текст
            разный: «ничего не найдено» при активном фильтре не значит, что
            обращений нет вовсе, и предлагать создать новое здесь неуместно. */}
        {visible.length === 0 && (
          <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-input bg-card px-5 py-12 text-center">
            <span className="flex h-[38px] w-[38px] items-center justify-center rounded-[10px] bg-muted text-muted-foreground">
              {tickets.length === 0 ? (
                <MessageSquare className="h-4 w-4" />
              ) : (
                <Search className="h-4 w-4" />
              )}
            </span>
            {tickets.length === 0 ? (
              <>
                <div className="text-sm font-medium">{t("support.list.empty_title")}</div>
                <div className="max-w-md text-[12.5px] text-muted-foreground">
                  {t("support.list.empty_text")}
                </div>
                <div className="mt-1 flex items-center gap-2">
                  <Link href="/support/kb" className={cn(btnGhost, "h-[34px]")}>
                    {t("support.kb.title")}
                  </Link>
                  <Link href="/support/create" className={cn(btnPrimary, "h-[34px]")}>
                    {t("support.list.create")}
                  </Link>
                </div>
              </>
            ) : (
              <>
                <div className="text-sm font-medium">{t("common.not_found")}</div>
                <div className="text-[12.5px] text-muted-foreground">
                  {t("support.list.not_found_text")}
                </div>
                {/* Подсказки — это заранее заданные запросы по темам, с
                    которыми обращаются чаще всего. Они не «умные»: просто
                    подставляют слово в поиск, чтобы не гадать с формулировкой. */}
                <div className="mt-1 flex flex-wrap justify-center gap-1.5">
                  {HINTS.map((h) => (
                    <button
                      key={h.queryKey}
                      type="button"
                      onClick={() => {
                        setFilter("all");
                        setQuery(t(h.queryKey));
                      }}
                      className="flex items-center gap-2 rounded-[9px] border border-border bg-background px-3 py-1.5 text-[12.5px] text-muted-foreground transition-colors hover:border-ring hover:text-foreground"
                    >
                      <h.Icon className="h-3.5 w-3.5" />
                      {t(h.labelKey)}
                    </button>
                  ))}
                </div>
                <button
                  type="button"
                  onClick={() => {
                    setFilter("all");
                    setQuery("");
                  }}
                  className={cn(btnGhost, "mt-1 h-[34px]")}
                >
                  {t("support.list.reset_filters")}
                </button>
              </>
            )}
          </div>
        )}
      </div>
    </PageShell>
  );
}
