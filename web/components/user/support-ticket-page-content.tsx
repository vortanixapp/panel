"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  ArrowLeft,
  Bell,
  BellOff,
  BookOpen,
  CheckCheck,
  CircleX,
  Headphones,
  MessageSquareOff,
  Paperclip,
  Send,
  User,
} from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { useT } from "@/hooks/use-translations";
import { SupportAttachmentChip } from "@/components/user/support/support-attachment";
import { formatSupportDate } from "@/components/user/support/support-utils";
import {
  closeSupportTicket,
  fetchSupportCreateForm,
  fetchSupportTicket,
  fetchSupportTickets,
  replySupportTicket,
  setTicketNotify,
  uploadSupportAttachment,
  type SupportMessage,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { btnGhost, btnPrimary, fieldClass } from "@/components/user/panel-parts";
import {
  FALLBACK_DEPARTMENTS,
  SupportStatusPill,
  departmentName,
  supportPriorityMeta,
  supportStatusMeta,
} from "@/components/user/support/support-parts";

function MessageBubble({
  message,
  ticketId,
}: {
  message: SupportMessage;
  ticketId: string;
}) {
  const t = useT();
  const staff = message.is_staff;

  return (
    <div className={cn("flex", staff ? "justify-start" : "justify-end")}>
      <div
        className={cn(
          "flex max-w-[85%] flex-col gap-2 rounded-2xl border px-4 py-3 md:max-w-[72%]",
          staff ? "border-border bg-card" : "border-input bg-muted"
        )}
      >
        <div className="flex items-center gap-2">
          <span
            className={cn(
              "flex h-5 w-5 items-center justify-center rounded-full",
              staff ? "bg-[var(--vx-info-tint)]" : "bg-background"
            )}
          >
            {staff ? (
              <Headphones className="h-3 w-3" style={{ color: "var(--vx-info)" }} />
            ) : (
              <User className="h-3 w-3 text-muted-foreground" />
            )}
          </span>
          <span
            className="text-[12.5px] font-semibold"
            style={staff ? { color: "var(--vx-info)" } : undefined}
          >
            {staff ? t("support.chat.staff") : t("support.chat.you")}
          </span>
          <span className="ms-auto text-[11px] text-muted-foreground/70">
            {formatSupportDate(message.created_at)}
          </span>
        </div>

        <div className="text-sm leading-relaxed whitespace-pre-line">
          {message.body}
        </div>

        {(message.attachments ?? (message.attachment ? [message.attachment] : [])).map(
          (attachment, i) => (
            <SupportAttachmentChip
              key={attachment.id ?? `${message.id}-${i}`}
              ticketId={ticketId}
              messageId={message.id}
              attachment={attachment}
            />
          )
        )}
      </div>
    </div>
  );
}

export function SupportTicketPageContent() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();
  const [reply, setReply] = useState("");
  const chatRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { data: list = [] } = useQuery({
    queryKey: queryKeys.supportTickets,
    queryFn: async () => (await fetchSupportTickets()).tickets,
  });
  const ticketMeta = list.find((t) => t.id === id);

  const { data: form } = useQuery({
    queryKey: queryKeys.supportCreateForm,
    queryFn: fetchSupportCreateForm,
  });
  const departments = form?.departments?.length
    ? form.departments
    : FALLBACK_DEPARTMENTS;

  const { data: thread, isLoading, isError } = useQuery({
    queryKey: queryKeys.supportTicket(id),
    queryFn: () => fetchSupportTicket(id),
    enabled: !!id,
    refetchInterval: 5000,
    refetchIntervalInBackground: false,
  });

  const messages = thread?.messages ?? [];
  const isClosed = ticketMeta?.status === "closed";

  useEffect(() => {
    if (chatRef.current) chatRef.current.scrollTop = chatRef.current.scrollHeight;
  }, [messages.length]);

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: queryKeys.supportTicket(id) });
    void qc.invalidateQueries({ queryKey: queryKeys.supportTickets });
  };

  const replyMutation = useMutation({
    mutationFn: (body: string) => replySupportTicket(id, body),
    onSuccess: () => {
      setReply("");
      invalidate();
    },
    onError: (err: Error) =>
      toast.error(err.message || t("support.ticket.reply_failed")),
  });

  const uploadMutation = useMutation({
    mutationFn: (files: File[]) => uploadSupportAttachment(id, files),
    onSuccess: (res) => {
      const count = res.attachments?.length ?? 1;
      toast.success(
        count > 1
          ? t("support.ticket.attached_many", { n: count })
          : t("support.ticket.attached_one", { name: res.filename })
      );
      invalidate();
    },
    onError: (err: Error) =>
      toast.error(err.message || t("support.ticket.upload_failed")),
  });

  // Уведомления по обращению. Признак живёт на тикете, а не на пользователе:
  // отключить хотят один разговор, а не поддержку целиком.
  const notifyOn = ticketMeta?.notify ?? true;
  const notifyMutation = useMutation({
    mutationFn: (next: boolean) => setTicketNotify(id, next),
    onSuccess: (res) => {
      toast.success(
        res.notify
          ? t("support.ticket.notify_on")
          : t("support.ticket.notify_off")
      );
      invalidate();
    },
    onError: (err: Error) =>
      toast.error(err.message || t("support.ticket.notify_failed")),
  });

  const closeMutation = useMutation({
    mutationFn: () => closeSupportTicket(id),
    onSuccess: () => {
      invalidate();
      toast.success(t("support.ticket.closed_toast"));
    },
    onError: (err: Error) =>
      toast.error(err.message || t("support.ticket.close_failed")),
  });

  function sendReply(e?: React.FormEvent) {
    e?.preventDefault();
    const text = reply.trim();
    if (!text || replyMutation.isPending || isClosed) return;
    replyMutation.mutate(text);
  }

  if (isLoading && !thread) {
    return (
      <PageShell variant="user">
        <div className="flex flex-col gap-[22px]">
          <Skeleton className="h-12 w-80 rounded-xl" />
          <Skeleton className="h-96 rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  if (isError || !thread) {
    return (
      <PageShell variant="user">
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <span className="flex h-14 w-14 items-center justify-center rounded-2xl bg-muted">
            <MessageSquareOff className="h-6 w-6 text-muted-foreground" />
          </span>
          <p className="text-sm text-muted-foreground">
            {t("support.ticket.not_found")}
          </p>
          <Link href="/support" className={cn(btnGhost, "h-[34px]")}>
            {t("support.ticket.back_to_list")}
          </Link>
        </div>
      </PageShell>
    );
  }

  const subject =
    ticketMeta?.subject ??
    t("support.ticket.fallback_subject", { id: thread.ticket_id });
  const status = ticketMeta?.status ?? "open";
  const priority = supportPriorityMeta(ticketMeta?.priority);

  const meta = [
    {
      label: t("support.field.number"),
      value: `#${String(id).slice(0, 8)}`,
      mono: true,
    },
    { label: t("common.status"), value: supportStatusMeta(status).label },
    {
      label: t("support.field.department"),
      value: departmentName(ticketMeta?.category, departments),
    },
    { label: t("support.field.priority"), value: priority.label },
    {
      label: t("support.field.created"),
      value: formatSupportDate(ticketMeta?.created_at),
    },
    { label: t("support.field.messages"), value: String(messages.length) },
  ];
  if (ticketMeta?.service) {
    meta.splice(3, 0, {
      label: t("support.field.service"),
      value: ticketMeta.service,
    });
  }

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-[22px]">
        <div className="flex flex-wrap items-center gap-[14px]">
          <Link
            href="/support"
            className="flex h-[34px] w-[34px] flex-none items-center justify-center rounded-[9px] border border-input text-foreground transition-colors hover:bg-accent"
            aria-label={t("support.aria.back_to_tickets")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2.5">
              <h1 className="text-[26px] leading-tight font-bold tracking-[-0.02em]">
                {subject}
              </h1>
              <SupportStatusPill status={status} />
            </div>
            <p className="mt-1.5 text-[13.5px] text-muted-foreground">
              {departmentName(ticketMeta?.category, departments)} · {supportStatusMeta(status).hint}
            </p>
          </div>
          {!isClosed && (
            <button
              type="button"
              onClick={() => closeMutation.mutate()}
              disabled={closeMutation.isPending}
              className={cn(btnGhost, "ms-auto h-[38px] text-[13px]")}
            >
              <CheckCheck className="h-3.5 w-3.5" />
              {closeMutation.isPending
                ? t("support.ticket.closing")
                : t("support.ticket.resolved")}
            </button>
          )}
        </div>

        <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,1fr)_300px]">
          <div className="flex flex-col gap-[14px] rounded-2xl border border-border bg-card px-[22px] py-5">
            <div
              ref={chatRef}
              className="flex max-h-[60vh] flex-col gap-3 overflow-y-auto pe-1"
            >
              {messages.map((m) => (
                <MessageBubble key={m.id} message={m} ticketId={id} />
              ))}
            </div>

            {isClosed ? (
              <div className="rounded-xl border border-dashed border-input px-4 py-3 text-center text-[12.5px] text-muted-foreground">
                {t("support.ticket.closed_note")}
              </div>
            ) : (
              <form onSubmit={sendReply} className="flex flex-col gap-2">
                <textarea
                  value={reply}
                  onChange={(e) => setReply(e.target.value)}
                  rows={3}
                  placeholder={t("support.ticket.reply_placeholder")}
                  // Enter отправляет, перенос строки — Shift+Enter. Так быстрее
                  // в переписке, а многострочный ответ всё равно возможен.
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !e.shiftKey) sendReply(e);
                  }}
                  className={cn(fieldClass, "h-auto resize-y py-2.5 leading-relaxed")}
                />
                <div className="flex items-center gap-2">
                  <input
                    ref={fileInputRef}
                    type="file"
                    multiple
                    className="hidden"
                    onChange={(e) => {
                      const files = Array.from(e.target.files ?? []);
                      if (files.length > 0) uploadMutation.mutate(files);
                      e.target.value = "";
                    }}
                  />
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    disabled={uploadMutation.isPending}
                    className={cn(btnGhost, "h-[38px]")}
                  >
                    <Paperclip className="h-3.5 w-3.5" />
                    {uploadMutation.isPending
                      ? t("common.loading")
                      : t("support.ticket.file_button")}
                  </button>
                  <button
                    type="submit"
                    disabled={!reply.trim() || replyMutation.isPending}
                    className={cn(btnPrimary, "ms-auto h-[38px] px-[18px]")}
                  >
                    <Send className="h-[15px] w-[15px]" />
                    {replyMutation.isPending
                      ? t("common.sending")
                      : t("common.send")}
                  </button>
                </div>
              </form>
            )}
          </div>

          <div className="flex flex-col gap-[14px] lg:sticky lg:top-[88px]">
          <div className="flex flex-col gap-[14px] rounded-2xl border border-border bg-card px-[22px] py-5">
            <span className="text-[15px] font-semibold">
              {t("support.ticket.card_title")}
            </span>
            <div className="flex flex-col">
              {meta.map((m) => (
                <div
                  key={m.label}
                  className="flex items-center justify-between gap-3 border-b border-border/60 py-[11px] text-[13px] last:border-0"
                >
                  <span className="text-muted-foreground">{m.label}</span>
                  <span className={cn("truncate", m.mono && "font-mono")}>{m.value}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="flex flex-col gap-2 rounded-2xl border border-border bg-card px-[22px] py-5">
            <span className="text-[15px] font-semibold">
              {t("common.actions")}
            </span>
            <Link href="/support/kb" className={cn(btnGhost, "h-[38px] justify-start")}>
              <BookOpen className="h-3.5 w-3.5" />
              {t("support.ticket.related_articles")}
            </Link>
            <button
              type="button"
              onClick={() => notifyMutation.mutate(!notifyOn)}
              disabled={notifyMutation.isPending}
              className={cn(btnGhost, "h-[38px] justify-start")}
            >
              {notifyOn ? (
                <BellOff className="h-3.5 w-3.5" />
              ) : (
                <Bell className="h-3.5 w-3.5" />
              )}
              {notifyOn
                ? t("support.ticket.notify_disable")
                : t("support.ticket.notify_enable")}
            </button>
            {!isClosed && (
              <button
                type="button"
                onClick={() => closeMutation.mutate()}
                disabled={closeMutation.isPending}
                className={cn(btnGhost, "h-[38px] justify-start")}
              >
                <CircleX className="h-3.5 w-3.5" />
                {closeMutation.isPending
                  ? t("support.ticket.closing")
                  : t("support.ticket.close_action")}
              </button>
            )}
          </div>
          </div>
        </div>
      </div>
    </PageShell>
  );
}
