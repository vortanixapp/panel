"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowLeft, Eye, Loader2, RotateCcw, Save, Send, Settings2 } from "lucide-react";

import { PageShell } from "@/components/layout/page-shell";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { btnGhost, btnPrimary, fieldClass } from "@/components/user/panel-parts";
import { useT } from "@/hooks/use-translations";
import {
  fetchMailTemplates,
  previewMailTemplate,
  resetMailTemplate,
  saveMailTemplate,
  sendMailTemplateTest,
  type MailTemplate,
  type MailTemplateField,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { MailLogTable } from "./mail-log-table";
import { MailPreviewDialog, type MailPreview } from "./mail-preview-dialog";

type Tab = "templates" | "log";

const NAMED_FIELDS = ["subject", "title", "body", "action"];

function draftOf(template: MailTemplate): Record<string, string> {
  const out: Record<string, string> = {};
  for (const field of template.fields) out[field.name] = field.value;
  return out;
}

function syncKey(locale: string, template: MailTemplate): string {
  return [locale, template.id, ...template.fields.map((f) => f.value)].join("|");
}

function matches(template: MailTemplate, query: string): boolean {
  if (!query) return true;
  const needle = query.toLowerCase();
  if (template.id.toLowerCase().includes(needle)) return true;
  if (template.name.toLowerCase().includes(needle)) return true;
  return template.fields.some((field) =>
    (field.value || field.default).toLowerCase().includes(needle)
  );
}

export function MailsPageContent() {
  const t = useT();
  const queryClient = useQueryClient();

  const [tab, setTab] = useState<Tab>("templates");
  const [locale, setLocale] = useState("");
  const [selectedId, setSelectedId] = useState("");
  const [draft, setDraft] = useState<Record<string, string>>({});
  const [synced, setSynced] = useState("");
  const [query, setQuery] = useState("");
  const [preview, setPreview] = useState<MailPreview | null>(null);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [resetOpen, setResetOpen] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-mail-templates", locale],
    queryFn: () => fetchMailTemplates(locale),
  });

  const groups = useMemo(() => data?.groups ?? [], [data]);
  const templates = useMemo(
    () => groups.flatMap((group) => group.templates),
    [groups]
  );
  const selected = templates.find((item) => item.id === selectedId) ?? null;

  useEffect(() => {
    if (data?.locale && !locale) setLocale(data.locale);
  }, [data?.locale, locale]);

  useEffect(() => {
    if (templates.length === 0) return;
    const current = templates.find((item) => item.id === selectedId) ?? templates[0];
    const key = syncKey(data?.locale ?? "", current);
    if (key === synced) return;
    setSelectedId(current.id);
    setDraft(draftOf(current));
    setSynced(key);
  }, [templates, selectedId, data?.locale, synced]);

  const dirty =
    selected !== null &&
    selected.fields.some((field) => (draft[field.name] ?? "") !== field.value);

  const save = useMutation({
    mutationFn: () => saveMailTemplate(selectedId, locale, draft),
    onSuccess: () => {
      toast.success(t("admin.mails.saved"));
      void queryClient.invalidateQueries({ queryKey: ["admin-mail-templates"] });
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const restore = useMutation({
    mutationFn: () => resetMailTemplate(selectedId, locale),
    onSuccess: () => {
      toast.success(t("admin.mails.reset_done"));
      void queryClient.invalidateQueries({ queryKey: ["admin-mail-templates"] });
    },
    onError: (err: Error) => toast.error(err.message),
  });

  const show = useMutation({
    mutationFn: () => previewMailTemplate(selectedId, locale, draft),
    onSuccess: (res) => setPreview(res),
    onError: (err: Error) => {
      setPreviewOpen(false);
      toast.error(err.message);
    },
  });

  const test = useMutation({
    mutationFn: () => sendMailTemplateTest(selectedId, locale, draft, ""),
    onSuccess: (res) => toast.success(t("admin.mails.test_sent", { email: res.to })),
    onError: (err: Error) => toast.error(err.message),
  });

  function openPreview() {
    setPreview(null);
    setPreviewOpen(true);
    show.mutate();
  }

  function pickTemplate(template: MailTemplate) {
    setSelectedId(template.id);
    setDraft(draftOf(template));
    setSynced(syncKey(data?.locale ?? "", template));
  }

  function copyParam(param: string) {
    void navigator.clipboard
      ?.writeText(`{${param}}`)
      .then(() => toast.success(t("admin.mails.param_copied")))
      .catch(() => undefined);
  }

  const mail = data?.mail;

  return (
    <PageShell variant="admin">
      <div className="mb-5 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("admin.mails.title")}</h1>
          <p className="text-[13px] text-muted-foreground">{t("admin.mails.subtitle")}</p>
        </div>
        <Link href="/admin" className={cn(btnGhost, "h-[38px]")}>
          <ArrowLeft className="size-4" />
          {t("layout.back_arrow")}
        </Link>
      </div>

      {mail && !mail.ready && (
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-[13px]">
          <span>
            {mail.mailer === "smtp"
              ? t("admin.mails.mail_off")
              : t("admin.mails.mail_mode", { mailer: mail.mailer })}
          </span>
          <Link href="/admin/settings?tab=mail" className={cn(btnGhost, "h-[34px]")}>
            <Settings2 className="size-3.5" />
            {t("admin.mails.settings")}
          </Link>
        </div>
      )}

      <div className="mb-4 flex gap-1.5">
        {(["templates", "log"] as Tab[]).map((item) => (
          <button
            key={item}
            type="button"
            onClick={() => setTab(item)}
            className={cn(
              "rounded-[9px] border px-3.5 py-2 text-[13px] transition-colors",
              tab === item
                ? "border-primary bg-primary/10 text-foreground"
                : "border-input text-muted-foreground hover:bg-accent"
            )}
          >
            {t(`admin.mails.tab_${item}`)}
          </button>
        ))}
      </div>

      {tab === "log" ? (
        <MailLogTable />
      ) : isLoading ? (
        <div className="grid gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
          <Skeleton className="h-[420px] w-full rounded-2xl" />
          <Skeleton className="h-[420px] w-full rounded-2xl" />
        </div>
      ) : (
        <div className="grid items-start gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
          <aside className="space-y-3 rounded-2xl border bg-card p-3">
            <div className="space-y-2">
              <select
                value={locale}
                onChange={(e) => setLocale(e.target.value)}
                aria-label={t("admin.mails.language")}
                className={fieldClass}
              >
                {(data?.languages ?? []).map((item) => (
                  <option key={item.code} value={item.code}>
                    {item.name}
                  </option>
                ))}
              </select>
              <input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={t("admin.mails.search")}
                className={fieldClass}
              />
            </div>

            <div className="max-h-[560px] space-y-3 overflow-y-auto pe-1">
              {groups.map((group) => {
                const items = group.templates.filter((item) => matches(item, query));
                if (items.length === 0) return null;
                return (
                  <div key={group.id} className="space-y-1">
                    <div className="px-1 text-[11.5px] tracking-wide text-muted-foreground uppercase">
                      {group.label}
                    </div>
                    {items.map((item) => (
                      <button
                        key={item.id}
                        type="button"
                        onClick={() => pickTemplate(item)}
                        className={cn(
                          "flex w-full items-center gap-2 rounded-[9px] px-2.5 py-2 text-start text-[13px] transition-colors",
                          item.id === selectedId
                            ? "bg-primary/10 text-foreground"
                            : "text-muted-foreground hover:bg-accent"
                        )}
                      >
                        <span className="min-w-0 flex-1 truncate">{item.name || item.id}</span>
                        {item.customized && (
                          <span className="shrink-0 rounded-full bg-muted px-1.5 py-0.5 text-[10.5px] text-muted-foreground">
                            {t("admin.mails.customized")}
                          </span>
                        )}
                      </button>
                    ))}
                  </div>
                );
              })}
              {templates.every((item) => !matches(item, query)) && (
                <div className="px-1 py-6 text-center text-[13px] text-muted-foreground">
                  {t("admin.mails.empty")}
                </div>
              )}
            </div>
          </aside>

          {selected && (
            <section className="space-y-4 rounded-2xl border bg-card px-5 py-5 sm:px-7 sm:py-6">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0 space-y-1">
                  <div className="text-[15px] leading-tight font-semibold">
                    {selected.name || selected.id}
                  </div>
                  <div className="font-mono text-[12px] text-muted-foreground">{selected.id}</div>
                </div>
                {mail?.from && (
                  <div className="text-[12px] text-muted-foreground">
                    {t("admin.mails.mail_from", { from: mail.from })}
                  </div>
                )}
              </div>

              <div className="space-y-4">
                {selected.fields.map((field) => (
                  <FieldRow
                    key={field.key}
                    field={field}
                    value={draft[field.name] ?? ""}
                    onChange={(value) => setDraft((prev) => ({ ...prev, [field.name]: value }))}
                  />
                ))}
              </div>

              {selected.params.length > 0 && (
                <div className="space-y-1.5 rounded-xl border bg-muted/30 p-3">
                  <div className="text-[12.5px] font-medium">{t("admin.mails.params")}</div>
                  <div className="flex flex-wrap gap-1.5">
                    {selected.params.map((param) => (
                      <button
                        key={param}
                        type="button"
                        onClick={() => copyParam(param)}
                        className="rounded-full border border-input px-2 py-0.5 font-mono text-[11.5px] transition-colors hover:bg-accent"
                      >
                        {`{${param}}`}
                      </button>
                    ))}
                  </div>
                  <p className="text-[11.5px] text-muted-foreground">
                    {t("admin.mails.params_hint")}
                  </p>
                </div>
              )}

              <div className="flex flex-wrap gap-2 border-t pt-4">
                <button
                  type="button"
                  onClick={() => save.mutate()}
                  disabled={!dirty || save.isPending}
                  className={cn(btnPrimary, "h-[38px]")}
                >
                  {save.isPending ? (
                    <Loader2 className="size-3.5 animate-spin" />
                  ) : (
                    <Save className="size-3.5" />
                  )}
                  {t("admin.mails.save")}
                </button>
                <button
                  type="button"
                  onClick={openPreview}
                  className={cn(btnGhost, "h-[38px]")}
                >
                  <Eye className="size-3.5" />
                  {t("admin.mails.preview")}
                </button>
                <button
                  type="button"
                  onClick={() => test.mutate()}
                  disabled={test.isPending || !mail?.ready}
                  className={cn(btnGhost, "h-[38px]")}
                >
                  {test.isPending ? (
                    <Loader2 className="size-3.5 animate-spin" />
                  ) : (
                    <Send className="size-3.5" />
                  )}
                  {t("admin.mails.send_test")}
                </button>
                {selected.customized && (
                  <button
                    type="button"
                    onClick={() => setResetOpen(true)}
                    disabled={restore.isPending}
                    className={cn(btnGhost, "h-[38px] ms-auto")}
                  >
                    <RotateCcw className="size-3.5" />
                    {t("admin.mails.reset")}
                  </button>
                )}
              </div>
            </section>
          )}
        </div>
      )}

      <MailPreviewDialog
        open={previewOpen}
        onOpenChange={setPreviewOpen}
        preview={preview}
        loading={show.isPending}
      />

      <ConfirmDialog
        open={resetOpen}
        onOpenChange={setResetOpen}
        destructive
        isLoading={restore.isPending}
        title={t("admin.mails.reset")}
        desc={t("admin.mails.reset_confirm", { name: selected?.name ?? selectedId })}
        confirmText={t("admin.mails.reset")}
        cancelBtnText={t("common.cancel")}
        handleConfirm={() => {
          setResetOpen(false);
          restore.mutate();
        }}
      />
    </PageShell>
  );
}

function FieldRow({
  field,
  value,
  onChange,
}: {
  field: MailTemplateField;
  value: string;
  onChange: (value: string) => void;
}) {
  const t = useT();

  return (
    <div className="space-y-1.5">
      <label className="text-[12.5px] font-medium" htmlFor={field.key}>
        {NAMED_FIELDS.includes(field.name) ? (
          t(`admin.mails.field_${field.name}`)
        ) : (
          <span className="font-mono text-[12px] font-normal text-muted-foreground">
            {field.name}
          </span>
        )}
      </label>
      {field.multiline ? (
        <textarea
          id={field.key}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={field.default}
          rows={6}
          className="w-full rounded-[9px] border border-input bg-background px-3 py-2 text-[13px] leading-relaxed outline-none transition-colors placeholder:text-muted-foreground/70 focus:border-ring"
        />
      ) : (
        <input
          id={field.key}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={field.default}
          className={fieldClass}
        />
      )}
      {field.default && (
        <p className="text-[11.5px] text-muted-foreground">
          {t("admin.mails.default_hint", { value: field.default })}
        </p>
      )}
    </div>
  );
}
