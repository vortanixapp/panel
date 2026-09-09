"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  ArrowLeft,
  CheckCheck,
  Headphones,
  MessageSquareOff,
  Paperclip,
  Send,
  User,
  UserPlus,
} from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { useT } from "@/hooks/use-translations";
import { SupportAttachmentChip } from "@/components/user/support/support-attachment";
import { formatSupportDate } from "@/components/user/support/support-utils";
import {
  closeSupportTicket,
  fetchAdminSupport,
  fetchSupportCreateForm,
  fetchSupportTicket,
  replySupportTicket,
  updateAdminSupportTicket,
  uploadSupportAttachment,
  type SupportMessage,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { btnGhost, btnPrimary, fieldClass } from "@/components/user/panel-parts";
import {
  FALLBACK_DEPARTMENTS,
  SUPPORT_PRIORITIES,
  SupportStatusPill,
  departmentName,
  supportStatusMeta,
  waitingFor,
} from "@/components/user/support/support-parts";

const CANNED = [
  {
    labelKey: "support.canned.accepted.label",
    textKey: "support.canned.accepted.text",
  },
  { labelKey: "support.canned.log.label", textKey: "support.canned.log.text" },
  {
    labelKey: "support.canned.solved.label",
    textKey: "support.canned.solved.text",
  },
  {
    labelKey: "support.canned.waiting.label",
    textKey: "support.canned.waiting.text",
  },
];

const STATUSES = [
  { id: "open", labelKey: "support.status.open" },
  { id: "pending", labelKey: "support.status.pending" },
  { id: "answered", labelKey: "support.status.answered_admin" },
  { id: "closed", labelKey: "support.status.closed" },
] as const;

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
    <div className={cn("flex", staff ? "justify-end" : "justify-start")}>
      <div
        className={cn(
          "flex max-w-[85%] flex-col gap-2 rounded-2xl border px-4 py-3 md:max-w-[72%]",
          staff ? "border-input bg-muted" : "border-border bg-card"
        )}
      >
        <div className="flex items-center gap-2">
          <span
            className={cn(
              "flex h-5 w-5 items-center justify-center rounded-full",
              staff ? "bg-background" : "bg-[var(--vx-info-tint)]"
            )}
          >
            {staff ? (
              <Headphones className="h-3 w-3 text-muted-foreground" />
            ) : (
              <User className="h-3 w-3" style={{ color: "var(--vx-info)" }} />
            )}
          </span>
          <span
            className="text-[12.5px] font-semibold"
            style={staff ? undefined : { color: "var(--vx-info)" }}
          >
            {staff ? t("support.chat.staff") : t("support.chat.client")}
          </span>
          <span className="ms-auto text-[11px] text-muted-foreground/70">
            {formatSupportDate(message.created_at)}
          </span>
        </div>

        <div className="text-sm leading-relaxed whitespace-pre-line">{message.body}</div>

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

export function AdminSupportTicketDetailContent() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();
  const [reply, setReply] = useState("");
  const chatRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { data: adminList = [] } = useQuery({
    queryKey: queryKeys.adminSupport,
    queryFn: async () => (await fetchAdminSupport()).tickets ?? [],
  });
  const ticket = adminList.find((t) => t.id === id);

  const { data: form } = useQuery({
    queryKey: queryKeys.supportCreateForm,
    queryFn: fetchSupportCreateForm,
  });
  const departments = form?.departments?.length ? form.departments : FALLBACK_DEPARTMENTS;

  const { data: thread, isLoading, isError } = useQuery({
    queryKey: queryKeys.supportTicket(id),
    queryFn: () => fetchSupportTicket(id),
    enabled: !!id,
    refetchInterval: 5000,
  });

  const messages = thread?.messages ?? [];
  const isClosed = ticket?.status === "closed";

  useEffect(() => {
    if (chatRef.current) chatRef.current.scrollTop = chatRef.current.scrollHeight;
  }, [messages.length]);

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: queryKeys.supportTicket(id) });
    void qc.invalidateQueries({ queryKey: queryKeys.adminSupport });
    void qc.invalidateQueries({ queryKey: ["admin-support"] });
  };

  const replyMut = useMutation({
    mutationFn: (body: string) => replySupportTicket(id, body),
    onSuccess: () => {
      setReply("");
      invalidate();
    },
    onError: (e: Error) =>
      toast.error(e.message || t("support.admin.reply_failed")),
  });

  const uploadMut = useMutation({
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
    onError: (e: Error) =>
      toast.error(e.message || t("support.ticket.upload_failed")),
  });

  const closeMut = useMutation({
    mutationFn: () => closeSupportTicket(id),
    onSuccess: () => {
      toast.success(t("support.ticket.closed_toast"));
      invalidate();
    },
    onError: (e: Error) =>
      toast.error(e.message || t("support.ticket.close_failed")),
  });

  const updateMut = useMutation({
    mutationFn: (data: Parameters<typeof updateAdminSupportTicket>[1]) =>
      updateAdminSupportTicket(id, data),
    onSuccess: () => {
      toast.success(t("support.queue.updated"));
      invalidate();
    },
    onError: (e: Error) =>
      toast.error(e.message || t("support.admin.update_failed")),
  });

  function sendReply(e?: React.FormEvent) {
    e?.preventDefault();
    const text = reply.trim();
    if (!text || replyMut.isPending) return;
    replyMut.mutate(text);
  }

  async function replyAndClose() {
    const text = reply.trim();
    if (!text) return;
    try {
      await replyMut.mutateAsync(text);
      await closeMut.mutateAsync();
    } catch {
    }
  }

  if (isLoading && !thread) {
    return (
      <PageShell variant="admin">
        <div className="flex flex-col gap-[22px]">
          <Skeleton className="h-12 w-80 rounded-xl" />
          <Skeleton className="h-96 rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  if (isError || !thread) {
    return (
      <PageShell variant="admin">
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <span className="flex h-14 w-14 items-center justify-center rounded-2xl bg-muted">
            <MessageSquareOff className="h-6 w-6 text-muted-foreground" />
          </span>
          <p className="text-sm text-muted-foreground">
            {t("support.ticket.not_found")}
          </p>
          <Link href="/admin/support" className={cn(btnGhost, "h-[34px]")}>
            {t("support.admin.back_to_queue")}
          </Link>
        </div>
      </PageShell>
    );
  }

  const status = ticket?.status ?? "open";
  const wait = waitingFor(ticket?.last_message_at ?? ticket?.created_at);
  const busy = replyMut.isPending || closeMut.isPending;

  return (
    <PageShell variant="admin">
      <div className="flex w-full flex-col gap-[22px]">
        <div className="flex flex-wrap items-center gap-[14px]">
          <Link
            href="/admin/support"
            className="flex h-[34px] w-[34px] flex-none items-center justify-center rounded-[9px] border border-input text-foreground transition-colors hover:bg-accent"
            aria-label={t("support.admin.back_aria")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2.5">
              <h1 className="text-[26px] leading-tight font-bold tracking-[-0.02em]">
                {ticket?.subject ??
                  t("support.ticket.fallback_subject", {
                    id: thread.ticket_id,
                  })}
              </h1>
              <SupportStatusPill status={status} />
            </div>
            <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-[12.5px] text-muted-foreground">
              <span className="font-mono">#{String(id).slice(0, 8)}</span>
              <span className="text-muted-foreground/50">·</span>
              <span>{ticket?.user_email || "—"}</span>
              {ticket?.awaiting_staff && status !== "closed" && (
                <>
                  <span className="text-muted-foreground/50">·</span>
                  <span className={cn(wait.overdue && "text-[var(--vx-danger)]")}>
                    {t("support.admin.waiting_for_reply", { time: wait.label })}
                  </span>
                </>
              )}
            </div>
          </div>
          {!ticket?.assigned_admin_email && (
            <button
              type="button"
              onClick={() => updateMut.mutate({ assign_to_me: true })}
              disabled={updateMut.isPending}
              className={cn(btnGhost, "ms-auto h-[38px] text-[13px]")}
            >
              <UserPlus className="h-3.5 w-3.5" />
              {t("support.admin.take")}
            </button>
          )}
        </div>

        <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,1fr)_300px]">
          <div className="flex flex-col gap-[14px] rounded-2xl border border-border bg-card px-[22px] py-5">
            <div className="flex items-center gap-2 text-[12.5px] text-muted-foreground">
              <span>{t("support.admin.thread")}</span>
              <span className="font-mono">{messages.length}</span>
            </div>

            <div ref={chatRef} className="flex max-h-[55vh] flex-col gap-3 overflow-y-auto pe-1">
              {messages.map((m) => (
                <MessageBubble key={m.id} message={m} ticketId={id} />
              ))}
            </div>

            <form onSubmit={sendReply} className="flex flex-col gap-2">
              <div className="flex flex-wrap gap-1.5">
                {CANNED.map((c) => (
                  <button
                    key={c.labelKey}
                    type="button"
                    onClick={() => setReply(t(c.textKey))}
                    className="rounded-[9px] border border-border bg-background px-3 py-1.5 text-[12px] text-muted-foreground transition-colors hover:border-ring hover:text-foreground"
                  >
                    {t(c.labelKey)}
                  </button>
                ))}
              </div>

              <textarea
                value={reply}
                onChange={(e) => setReply(e.target.value)}
                rows={4}
                placeholder={t("support.admin.reply_placeholder")}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) sendReply(e);
                }}
                className={cn(fieldClass, "h-auto resize-y py-2.5 leading-relaxed")}
              />

              <div className="flex flex-wrap items-center gap-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  multiple
                  className="hidden"
                  onChange={(e) => {
                    const files = Array.from(e.target.files ?? []);
                    if (files.length > 0) uploadMut.mutate(files);
                    e.target.value = "";
                  }}
                />
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={uploadMut.isPending}
                  className={cn(btnGhost, "h-[38px]")}
                >
                  <Paperclip className="h-3.5 w-3.5" />
                  {uploadMut.isPending
                    ? t("common.loading")
                    : t("support.ticket.file_button")}
                </button>

                {!isClosed && (
                  <button
                    type="button"
                    onClick={replyAndClose}
                    disabled={!reply.trim() || busy}
                    className={cn(btnGhost, "h-[38px]")}
                  >
                    <CheckCheck className="h-3.5 w-3.5" />
                    {t("support.admin.reply_and_close")}
                  </button>
                )}

                <button
                  type="submit"
                  disabled={!reply.trim() || busy}
                  className={cn(btnPrimary, "ms-auto h-[38px] px-[18px]")}
                >
                  <Send className="h-[15px] w-[15px]" />
                  {replyMut.isPending ? t("common.sending") : t("common.send")}
                </button>
              </div>
            </form>
          </div>

          <div className="flex flex-col gap-[14px] lg:sticky lg:top-[88px]">
            <div className="flex flex-col gap-[14px] rounded-2xl border border-border bg-card px-5 py-4">
              <span className="text-[15px] font-semibold">
                {t("support.admin.client_card")}
              </span>
              <div className="flex flex-col">
                {[
                  {
                    label: t("support.field.email"),
                    value: ticket?.user_email || "—",
                  },
                  {
                    label: t("support.field.department"),
                    value: departmentName(ticket?.category, departments),
                  },
                  {
                    label: t("support.field.created"),
                    value: formatSupportDate(ticket?.created_at),
                  },
                  {
                    label: t("support.field.messages"),
                    value: String(messages.length),
                  },
                  {
                    label: t("support.field.agent"),
                    value:
                      ticket?.assigned_admin_email ||
                      t("support.admin.unassigned"),
                  },
                ].map((r) => (
                  <div
                    key={r.label}
                    className="flex items-center justify-between gap-3 border-b border-border/60 py-[11px] text-[13px] last:border-0"
                  >
                    <span className="text-muted-foreground">{r.label}</span>
                    <span className="truncate">{r.value}</span>
                  </div>
                ))}
              </div>
            </div>

            <div className="flex flex-col gap-[14px] rounded-2xl border border-border bg-card px-5 py-4">
              <span className="text-[15px] font-semibold">
                {t("support.admin.manage")}
              </span>

              <label className="flex flex-col gap-[7px]">
                <span className="text-[12.5px] text-muted-foreground">
                  {t("common.status")}
                </span>
                <select
                  value={status}
                  onChange={(e) =>
                    updateMut.mutate({
                      status: e.target.value as (typeof STATUSES)[number]["id"],
                    })
                  }
                  className={cn(fieldClass, "px-2.5")}
                >
                  {STATUSES.map((s) => (
                    <option key={s.id} value={s.id}>
                      {t(s.labelKey)}
                    </option>
                  ))}
                </select>
                <span className="text-[11.5px] text-muted-foreground/70">
                  {supportStatusMeta(status).hint}
                </span>
              </label>

              <label className="flex flex-col gap-[7px]">
                <span className="text-[12.5px] text-muted-foreground">
                  {t("support.field.priority")}
                </span>
                <select
                  value={ticket?.priority ?? "normal"}
                  onChange={(e) =>
                    updateMut.mutate({
                      priority: e.target.value as "low" | "normal" | "high" | "urgent",
                    })
                  }
                  className={cn(fieldClass, "px-2.5")}
                >
                  {SUPPORT_PRIORITIES.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.label}
                    </option>
                  ))}
                </select>
              </label>

              {ticket?.assigned_admin_email ? (
                <div className="text-[12.5px] text-muted-foreground">
                  {t("support.admin.assigned_to", {
                    email: ticket.assigned_admin_email,
                  })}
                </div>
              ) : (
                <button
                  type="button"
                  onClick={() => updateMut.mutate({ assign_to_me: true })}
                  disabled={updateMut.isPending}
                  className={cn(btnGhost, "h-[38px] text-[13px]")}
                >
                  <UserPlus className="h-3.5 w-3.5" />
                  {t("support.admin.assign_to_me")}
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </PageShell>
  );
}
