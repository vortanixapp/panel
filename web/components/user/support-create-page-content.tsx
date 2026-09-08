"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowLeft, BookOpen, FileText, Paperclip, Send, Timer, X } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import {
  createSupportTicket,
  fetchKBArticles,
  fetchSupportCreateForm,
  uploadSupportAttachment,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { btnGhost, btnPrimary, fieldClass } from "@/components/user/panel-parts";
import {
  FALLBACK_DEPARTMENTS,
  SUPPORT_PRIORITIES,
  departmentName,
  formatMinutes,
} from "@/components/user/support/support-parts";

const BODY_LIMIT = 4000;

/** Заготовки обращения.
 *
 * Половина тикетов приходит без единого факта: «не работает, помогите». Шаблон
 * подставляет список того, что мы всё равно спросим первым ответом, — так
 * разбор начинается сразу, а не со второго круга переписки. */
// Список читается на уровне модуля, поэтому храним ключи: и подпись кнопки,
// и подставляемый текст должны быть на языке, выбранном сейчас.
const TEMPLATES = [
  {
    id: "server",
    category: "technical",
    labelKey: "support.template.server.label",
    subjectKey: "support.template.server.subject",
    bodyKey: "support.template.server.body",
  },
  {
    id: "billing",
    category: "billing",
    labelKey: "support.template.billing.label",
    subjectKey: "support.template.billing.subject",
    bodyKey: "support.template.billing.body",
  },
  {
    id: "domain",
    category: "hosting",
    labelKey: "support.template.domain.label",
    subjectKey: "support.template.domain.subject",
    bodyKey: "support.template.domain.body",
  },
  {
    id: "account",
    category: "other",
    labelKey: "support.template.account.label",
    subjectKey: "support.template.account.subject",
    bodyKey: "support.template.account.body",
  },
];

export function SupportCreatePageContent() {
  const t = useT();
  const router = useRouter();
  const qc = useQueryClient();

  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [category, setCategory] = useState("");
  const [priority, setPriority] = useState("normal");
  const [service, setService] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { data: form } = useQuery({
    queryKey: queryKeys.supportCreateForm,
    queryFn: fetchSupportCreateForm,
  });
  const departments = form?.departments?.length
    ? form.departments
    : FALLBACK_DEPARTMENTS;
  const services = form?.services ?? [];
  const sla = form?.sla ?? [];

  // Подсказка из базы знаний по теме обращения. Ищем только когда есть что
  // искать: пустой запрос вернул бы весь список, а он здесь не к месту.
  const probe = subject.trim();
  const { data: kb } = useQuery({
    queryKey: queryKeys.kbArticles(undefined, probe),
    queryFn: () => fetchKBArticles({ q: probe }),
    enabled: probe.length >= 3,
  });
  const suggestions = (kb?.articles ?? []).slice(0, 3);

  // Файлы копятся в форме и уходят сразу после создания: ручка вложений
  // требует номер обращения, которого до отправки ещё нет. Ошибку загрузки не
  // превращаем в ошибку создания — обращение уже есть, и терять его из-за
  // неприложенного лога нельзя.
  const createMutation = useMutation({
    mutationFn: async () => {
      // Услуга приходит одной строкой «вид:номер»: в селекте это один пункт,
      // а серверу нужны обе части — по виду он знает, в какой таблице искать.
      const [serviceKind, serviceId] = service.split(":");
      const created = await createSupportTicket(subject.trim(), body.trim(), {
        category: category || departments[0]?.id,
        priority,
        service_kind: serviceKind || undefined,
        service_id: serviceId || undefined,
      });
      const failed: string[] = [];
      for (const file of files) {
        try {
          await uploadSupportAttachment(created.id, file);
        } catch {
          failed.push(file.name);
        }
      }
      return { id: created.id, failed };
    },
    onSuccess: ({ id, failed }) => {
      void qc.invalidateQueries({ queryKey: queryKeys.supportTickets });
      if (failed.length > 0) {
        toast.warning(
          t("support.create.created_with_failed", { files: failed.join(", ") })
        );
      } else {
        toast.success(t("support.create.created"));
      }
      router.push(`/support/${id}`);
    },
    onError: (err: Error) =>
      toast.error(err.message || t("support.create.create_failed")),
  });

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!subject.trim() || !body.trim()) return;
    createMutation.mutate();
  }

  // Параметр назван tpl, а не t: t — это функция перевода.
  function applyTemplate(tpl: (typeof TEMPLATES)[number]) {
    setSubject(t(tpl.subjectKey));
    setBody(t(tpl.bodyKey));
    // Отдел из шаблона ставим, только если он есть у этой панели: набор
    // настраивается, и чужой код молча уехал бы в «Другое».
    if (departments.some((d) => d.id === tpl.category)) setCategory(tpl.category);
  }

  return (
    <PageShell variant="user">
      <form onSubmit={submit} className="flex w-full flex-col gap-[22px]">
        <div className="flex items-center gap-[14px]">
          <Link
            href="/support"
            className="flex h-[34px] w-[34px] flex-none items-center justify-center rounded-[9px] border border-input text-foreground transition-colors hover:bg-accent"
            aria-label={t("support.aria.back_to_tickets")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div className="flex flex-col gap-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-[-0.02em]">
              {t("support.create.title")}
            </h1>
            <p className="text-[13.5px] text-muted-foreground">
              {t("support.create.subtitle")}
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,1fr)_320px]">
          <div className="flex flex-col gap-[14px] rounded-2xl border border-border bg-card px-[22px] py-5">
            <label className="flex flex-col gap-[7px]">
              <span className="text-[12.5px] text-muted-foreground">
                {t("support.create.subject")}
              </span>
              <input
                type="text"
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
                required
                placeholder={t("support.create.subject_placeholder")}
                className={fieldClass}
              />
            </label>

            {suggestions.length > 0 && (
              <div className="flex flex-col gap-2 rounded-xl border border-border bg-background p-3">
                <span className="text-[11.5px] text-muted-foreground">
                  {t("support.create.kb_hint")}
                </span>
                {suggestions.map((a) => (
                  <Link
                    key={a.id}
                    href={`/support/kb/${a.slug}`}
                    className="flex items-center gap-2 text-[13px] text-foreground hover:underline"
                  >
                    <BookOpen className="h-3.5 w-3.5 flex-none text-muted-foreground" />
                    <span className="truncate">{a.title}</span>
                  </Link>
                ))}
              </div>
            )}

            <div className="grid grid-cols-1 gap-[14px] sm:grid-cols-2">
              <label className="flex flex-col gap-[7px]">
                <span className="text-[12.5px] text-muted-foreground">
                  {t("support.field.department")}
                </span>
                <select
                  value={category || departments[0]?.id || ""}
                  onChange={(e) => setCategory(e.target.value)}
                  className={cn(fieldClass, "px-2.5")}
                >
                  {departments.map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.name}
                    </option>
                  ))}
                </select>
              </label>

              {services.length > 0 && (
                <label className="flex flex-col gap-[7px]">
                  <span className="text-[12.5px] text-muted-foreground">
                    {t("support.field.service")}
                  </span>
                  <select
                    value={service}
                    onChange={(e) => setService(e.target.value)}
                    className={cn(fieldClass, "px-2.5")}
                  >
                    <option value="">{t("support.create.service_none")}</option>
                    {services.map((sv) => (
                      <option key={`${sv.kind}:${sv.id}`} value={`${sv.kind}:${sv.id}`}>
                        {sv.label}
                      </option>
                    ))}
                  </select>
                </label>
              )}

              <div className="flex flex-col gap-[7px]">
                <span className="text-[12.5px] text-muted-foreground">
                  {t("support.field.priority")}
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {SUPPORT_PRIORITIES.map((p) => (
                    <button
                      key={p.id}
                      type="button"
                      onClick={() => setPriority(p.id)}
                      aria-pressed={priority === p.id}
                      className={cn(
                        "flex h-[38px] items-center gap-2 rounded-[9px] border px-3 text-[12.5px] transition-colors",
                        priority === p.id
                          ? "border-ring bg-muted"
                          : "border-border bg-background hover:border-ring"
                      )}
                    >
                      <span
                        className="h-2 w-2 rounded-full"
                        style={{ background: `var(${p.token})` }}
                      />
                      {p.label}
                    </button>
                  ))}
                </div>
              </div>
            </div>

            <label className="flex flex-col gap-[7px]">
              <span className="flex items-center justify-between text-[12.5px] text-muted-foreground">
                <span>{t("support.create.body")}</span>
                <span
                  className={cn(
                    "font-mono",
                    body.length > BODY_LIMIT && "text-destructive"
                  )}
                >
                  {body.length} / {BODY_LIMIT}
                </span>
              </span>
              <textarea
                value={body}
                onChange={(e) => setBody(e.target.value)}
                required
                rows={10}
                maxLength={BODY_LIMIT}
                placeholder={t("support.create.body_placeholder")}
                className={cn(fieldClass, "h-auto resize-y py-2.5 leading-relaxed")}
              />
            </label>

            <div className="flex flex-col gap-[7px]">
              <span className="text-[12.5px] text-muted-foreground">
                {t("support.create.attachments")}
              </span>
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                className="flex flex-col items-center gap-1.5 rounded-xl border border-dashed border-input bg-background px-4 py-6 text-center transition-colors hover:border-ring"
              >
                <Paperclip className="h-4 w-4 text-muted-foreground" />
                <span className="text-[13px]">{t("support.create.pick_files")}</span>
                <span className="text-[11.5px] text-muted-foreground/70">
                  {t("support.create.file_types")}
                </span>
              </button>
              <input
                ref={fileInputRef}
                type="file"
                multiple
                className="hidden"
                onChange={(e) => {
                  const picked = Array.from(e.target.files ?? []);
                  if (picked.length) setFiles((prev) => [...prev, ...picked]);
                  e.target.value = "";
                }}
              />
              {files.length > 0 && (
                <div className="flex flex-wrap gap-1.5">
                  {files.map((f, i) => (
                    <span
                      key={`${f.name}-${i}`}
                      className="flex items-center gap-2 rounded-[9px] border border-border bg-background px-2.5 py-1.5 text-[12px]"
                    >
                      <FileText className="h-3.5 w-3.5 text-muted-foreground" />
                      <span className="max-w-[180px] truncate">{f.name}</span>
                      <span className="text-muted-foreground/70">
                        {t("support.attachment.size_kb", {
                          n: Math.max(1, Math.round(f.size / 1024)),
                        })}
                      </span>
                      <button
                        type="button"
                        onClick={() => setFiles((prev) => prev.filter((_, j) => j !== i))}
                        aria-label={t("support.create.remove_file", { name: f.name })}
                        className="text-muted-foreground transition-colors hover:text-foreground"
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                    </span>
                  ))}
                </div>
              )}
            </div>

            <div className="flex items-center justify-end gap-2">
              <Link href="/support" className={cn(btnGhost, "h-[38px]")}>
                {t("common.cancel")}
              </Link>
              <button
                type="submit"
                disabled={
                  createMutation.isPending ||
                  !subject.trim() ||
                  !body.trim() ||
                  body.length > BODY_LIMIT
                }
                className={cn(btnPrimary, "h-[38px] px-[18px]")}
              >
                <Send className="h-[15px] w-[15px]" />
                {createMutation.isPending ? t("common.sending") : t("common.send")}
              </button>
            </div>
          </div>

          <div className="flex flex-col gap-[14px] lg:sticky lg:top-[88px]">
          {sla.length > 0 && (
            <div className="flex flex-col gap-[14px] rounded-2xl border border-border bg-card px-[22px] py-5">
              <div className="flex items-center gap-2 text-[15px] font-semibold">
                <Timer className="h-4 w-4 text-muted-foreground" />
                {t("support.create.sla_title")}
              </div>
              <div className="flex flex-col gap-2">
                {sla.map((row) => (
                  <div
                    key={row.category}
                    className="flex items-center justify-between gap-3 text-[13px]"
                  >
                    <span className="truncate text-muted-foreground">
                      {departmentName(row.category, departments)}
                    </span>
                    <span className="font-mono">{formatMinutes(row.minutes)}</span>
                  </div>
                ))}
              </div>
              {/* Это замер, а не обещание: считается по переписке за неделю. */}
              <p className="text-[11.5px] leading-relaxed text-muted-foreground/70">
                {t("support.create.sla_note")}
              </p>
            </div>
          )}

          <div className="flex flex-col gap-[14px] rounded-2xl border border-border bg-card px-[22px] py-5">
            <div className="flex flex-col gap-1">
              <span className="text-[15px] font-semibold">
                {t("support.create.templates_title")}
              </span>
              <span className="text-[12.5px] text-muted-foreground">
                {t("support.create.templates_note")}
              </span>
            </div>
            <div className="flex flex-col gap-1.5">
              {/* Параметр назван tpl, а не t: t — это функция перевода. */}
              {TEMPLATES.map((tpl) => (
                <button
                  key={tpl.id}
                  type="button"
                  onClick={() => applyTemplate(tpl)}
                  className="flex items-center gap-2.5 rounded-[9px] border border-border bg-background px-3 py-2.5 text-start text-[13px] transition-colors hover:border-ring"
                >
                  <FileText className="h-3.5 w-3.5 flex-none text-muted-foreground" />
                  {t(tpl.labelKey)}
                </button>
              ))}
            </div>

            <div className="h-px bg-border" />

            <Link
              href="/support/kb"
              className="flex items-center gap-2 text-[13px] text-muted-foreground transition-colors hover:text-foreground"
            >
              <BookOpen className="h-3.5 w-3.5" />
              {t("support.create.open_kb")}
            </Link>
          </div>
          </div>
        </div>
      </form>
    </PageShell>
  );
}
