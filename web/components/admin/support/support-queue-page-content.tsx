"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { AlarmClock, Search, UserPlus } from "lucide-react";

import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchAdminSupport,
  fetchSupportCreateForm,
  updateAdminSupportTicket,
  type AdminSupportTicket,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import {
  StatCell,
  StatStrip,
  btnGhost,
  fieldClass,
} from "@/components/user/panel-parts";
import {
  FALLBACK_DEPARTMENTS,
  PriorityDot,
  SupportStatusPill,
  departmentName,
  supportPriorityMeta,
  waitingFor,
} from "@/components/user/support/support-parts";
import { useT } from "@/hooks/use-translations";

// Подписи храним ключами: список читается на уровне модуля, и готовый текст
// застыл бы на языке, который стоял в момент загрузки страницы.
const STATUS_FILTER = [
  { value: "all", labelKey: "common.all" },
  { value: "open", labelKey: "admin.support.filter.open" },
  { value: "pending", labelKey: "admin.support.filter.pending" },
  { value: "answered", labelKey: "admin.support.filter.answered" },
  { value: "closed", labelKey: "admin.support.filter.closed" },
];

/** Ширины колонок держим в одном месте: заголовок и строки обязаны совпадать,
 *  а два одинаковых списка классов разъезжаются на первой же правке. */
const COLUMNS =
  "grid-cols-[92px_minmax(0,2.2fr)_minmax(0,1.4fr)_110px_120px_auto] gap-3";

function QueueHead() {
  const t = useT();
  return (
    <div
      className={cn(
        "hidden items-center px-5 py-2.5 text-[11.5px] text-muted-foreground lg:grid",
        COLUMNS
      )}
    >
      <span>{t("admin.support.col_ticket")}</span>
      <span>{t("admin.support.col_subject")}</span>
      <span>{t("admin.support.col_client")}</span>
      <span>{t("admin.support.col_priority")}</span>
      <span>{t("admin.support.col_waiting")}</span>
      <span />
    </div>
  );
}

function QueueRow({
  ticket,
  departments,
  onAssign,
  assigning,
}: {
  ticket: AdminSupportTicket;
  departments: { id: string; name: string }[];
  onAssign: () => void;
  assigning: boolean;
}) {
  const t = useT();
  // Ждём именно ответа оператора. Если последнее слово за поддержкой, счётчик
  // ожидания ни о чём не говорит: очередь стоит не у нас.
  const wait = waitingFor(ticket.last_message_at ?? ticket.created_at);
  const overdue = ticket.awaiting_staff && wait.overdue && ticket.status !== "closed";

  return (
    <div
      className={cn(
        "items-center border-t border-border px-5 py-3 transition-colors hover:bg-accent/40",
        "flex flex-col gap-2 lg:grid",
        COLUMNS,
        overdue && "bg-[color-mix(in_srgb,var(--vx-danger)_7%,transparent)]"
      )}
    >
      <span className="flex w-full items-center gap-2 lg:w-auto">
        <PriorityDot priority={ticket.priority} />
        <span className="font-mono text-[12px] text-muted-foreground">
          #{String(ticket.id).slice(0, 6)}
        </span>
      </span>

      <span className="w-full min-w-0">
        <Link
          href={`/admin/support/${ticket.id}`}
          className="block truncate text-[14px] font-medium hover:underline"
        >
          {ticket.subject}
        </Link>
        <span className="mt-1 flex flex-wrap items-center gap-2 text-[11.5px] text-muted-foreground">
          <SupportStatusPill status={ticket.status} />
          <span>{departmentName(ticket.category, departments)}</span>
          {ticket.service && (
            <>
              <span className="text-muted-foreground/50">·</span>
              <span className="truncate">{ticket.service}</span>
            </>
          )}
          <span className="text-muted-foreground/50">·</span>
          <span>
            {t("admin.support.messages", { count: ticket.messages })}
          </span>
        </span>
      </span>

      <span className="w-full min-w-0">
        <span className="block truncate text-[13px]">{ticket.user_email || "—"}</span>
        <span className="mt-0.5 block text-[11.5px] text-muted-foreground">
          {(ticket.user_tickets ?? 0) > 1
            ? t("admin.support.user_tickets", {
                count: ticket.user_tickets ?? 0,
              })
            : t("admin.support.first_ticket")}
        </span>
      </span>

      <span className="text-[12.5px] text-muted-foreground">
        {supportPriorityMeta(ticket.priority).label}
      </span>

      <span
        className={cn(
          "flex items-center gap-1.5 text-[12.5px]",
          overdue ? "text-[var(--vx-danger)]" : "text-muted-foreground"
        )}
      >
        {overdue && <AlarmClock className="h-3.5 w-3.5" />}
        {ticket.status === "closed"
          ? t("admin.support.closed_lc")
          : ticket.awaiting_staff
            ? wait.label
            : t("admin.support.client_turn")}
      </span>

      <span className="flex w-full justify-start lg:w-auto lg:justify-end">
        {ticket.assigned_admin_email ? (
          <span className="truncate text-[11.5px] text-muted-foreground">
            {ticket.assigned_admin_email}
          </span>
        ) : (
          <button
            type="button"
            onClick={onAssign}
            disabled={assigning}
            className={cn(btnGhost, "h-[30px] px-3 text-[12px]")}
          >
            <UserPlus className="h-3.5 w-3.5" />
            {t("admin.support.take")}
          </button>
        )}
      </span>
    </div>
  );
}

export function SupportQueuePageContent() {
  const t = useT();
  const qc = useQueryClient();
  const [status, setStatus] = useState("all");
  const [category, setCategory] = useState("");
  const [query, setQuery] = useState("");

  const { data: form } = useQuery({
    queryKey: queryKeys.supportCreateForm,
    queryFn: fetchSupportCreateForm,
  });
  const departments = form?.departments?.length
    ? form.departments
    : FALLBACK_DEPARTMENTS;

  const { data, isLoading } = useQuery({
    queryKey: ["admin-support", status, category],
    queryFn: async () =>
      (
        await fetchAdminSupport(
          status === "all" ? undefined : status,
          category || undefined
        )
      ).tickets,
  });
  const tickets = data ?? [];

  const updateMut = useMutation({
    mutationFn: ({
      id,
      ...payload
    }: { id: string } & Parameters<typeof updateAdminSupportTicket>[1]) =>
      updateAdminSupportTicket(id, payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["admin-support"] });
      toast.success(t("admin.support.updated"));
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.support.update_failed")),
  });

  const stats = useMemo(() => {
    const openish = tickets.filter((t) => t.status !== "closed");
    const overdue = openish.filter(
      (t) =>
        t.awaiting_staff && waitingFor(t.last_message_at ?? t.created_at).overdue
    );
    const unassigned = openish.filter((t) => !t.assigned_admin_email);
    return {
      queue: openish.length,
      overdue: overdue.length,
      unassigned: unassigned.length,
      closed: tickets.filter((t) => t.status === "closed").length,
    };
  }, [tickets]);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return tickets;
    return tickets.filter(
      (t) =>
        t.subject.toLowerCase().includes(q) ||
        (t.user_email ?? "").toLowerCase().includes(q) ||
        String(t.id).toLowerCase().includes(q)
    );
  }, [tickets, query]);

  return (
    <PageShell variant="admin">
      <div className="flex w-full flex-col gap-[22px]">
        <div className="flex flex-col gap-1.5">
          <h1 className="text-[26px] leading-none font-bold tracking-[-0.02em]">
            {t("admin.dashboard.support_queue")}
          </h1>
          <p className="text-[13.5px] text-muted-foreground">
            {t("admin.support.subtitle")}
          </p>
        </div>

        <StatStrip>
          <StatCell
            label={t("admin.jobs.summary.pending")}
            value={stats.queue}
          />
          <StatCell
            label={t("admin.support.stat_overdue")}
            value={stats.overdue}
            tone={stats.overdue > 0 ? "warn" : undefined}
          />
          <StatCell
            label={t("admin.support.stat_unassigned")}
            value={stats.unassigned}
            tone={stats.unassigned > 0 ? "warn" : undefined}
          />
          <StatCell
            label={t("admin.support.filter.closed")}
            value={stats.closed}
          />
        </StatStrip>

        <div className="flex flex-wrap items-center gap-2">
          <div className="no-scrollbar flex gap-1 overflow-x-auto rounded-xl border border-border bg-card p-1">
            {STATUS_FILTER.map((f) => (
              <button
                key={f.value}
                type="button"
                onClick={() => setStatus(f.value)}
                aria-pressed={status === f.value}
                className={cn(
                  "flex h-9 flex-none items-center rounded-[9px] px-3.5 text-[13px] whitespace-nowrap transition-colors",
                  status === f.value
                    ? "bg-accent font-medium text-accent-foreground"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                {t(f.labelKey)}
              </button>
            ))}
          </div>

          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className={cn(fieldClass, "w-auto min-w-[170px] px-2.5")}
          >
            <option value="">{t("admin.support.all_departments")}</option>
            {departments.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>

          <label className="relative ms-auto min-w-[220px] flex-1 sm:max-w-[280px] sm:flex-none">
            <Search className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={t("admin.support.search_placeholder")}
              className={cn(fieldClass, "ps-9")}
            />
          </label>
        </div>

        {isLoading ? (
          <div className="flex flex-col gap-2.5">
            <Skeleton className="h-24 rounded-2xl" />
            <Skeleton className="h-24 rounded-2xl" />
            <Skeleton className="h-24 rounded-2xl" />
          </div>
        ) : visible.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-input bg-card px-5 py-12 text-center">
            <p className="text-sm font-medium">
              {t("admin.support.empty")}
            </p>
            <p className="mt-1 text-[12.5px] text-muted-foreground">
              {t("admin.support.empty_hint")}
            </p>
          </div>
        ) : (
          <div className="overflow-hidden rounded-2xl border border-border bg-card">
            <QueueHead />
            {/* Параметр назван ticket, а не t: имя t занято функцией перевода. */}
            {visible.map((ticket) => (
              <QueueRow
                key={ticket.id}
                ticket={ticket}
                departments={departments}
                assigning={updateMut.isPending}
                onAssign={() =>
                  updateMut.mutate({ id: ticket.id, assign_to_me: true })
                }
              />
            ))}
          </div>
        )}
      </div>
    </PageShell>
  );
}
