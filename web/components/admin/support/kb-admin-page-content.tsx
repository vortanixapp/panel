"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { BookOpen, Eye, EyeOff, Plus, Trash2 } from "lucide-react";

import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import {
  deleteAdminKBArticle,
  fetchAdminKBArticles,
  fetchSupportCreateForm,
  saveAdminKBArticle,
  type KBArticle,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { btnGhost, btnPrimary, fieldClass } from "@/components/user/panel-parts";
import {
  FALLBACK_DEPARTMENTS,
  departmentName,
} from "@/components/user/support/support-parts";
import { useT } from "@/hooks/use-translations";

type Draft = Partial<KBArticle>;

const EMPTY: Draft = {
  title: "",
  slug: "",
  excerpt: "",
  body: "",
  category: "",
  published: true,
  position: 0,
};

export function KBAdminPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const [draft, setDraft] = useState<Draft | null>(null);

  const { data: form } = useQuery({
    queryKey: queryKeys.supportCreateForm,
    queryFn: fetchSupportCreateForm,
  });
  const departments = form?.departments?.length
    ? form.departments
    : FALLBACK_DEPARTMENTS;

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminKbArticles,
    queryFn: async () => (await fetchAdminKBArticles()).articles,
  });
  const articles = data ?? [];

  const refresh = () => {
    void qc.invalidateQueries({ queryKey: queryKeys.adminKbArticles });
    void qc.invalidateQueries({ queryKey: ["kb-articles"] });
  };

  const saveMut = useMutation({
    mutationFn: (a: Draft) => saveAdminKBArticle(a),
    onSuccess: () => {
      toast.success(t("admin.kb.saved"));
      setDraft(null);
      refresh();
    },
    onError: (e: Error) => toast.error(e.message || t("common.save_failed")),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteAdminKBArticle(id),
    onSuccess: () => {
      toast.success(t("admin.kb.deleted"));
      refresh();
    },
    onError: (e: Error) => toast.error(e.message || t("common.delete_failed")),
  });

  const set = (patch: Draft) => setDraft((d) => ({ ...(d ?? EMPTY), ...patch }));

  return (
    <PageShell variant="admin">
      <div className="flex w-full flex-col gap-[22px]">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div className="flex flex-col gap-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-[-0.02em]">
              {t("admin.kb.title")}
            </h1>
            <p className="text-[13.5px] text-muted-foreground">
              {t("admin.kb.subtitle")}
            </p>
          </div>
          <button
            type="button"
            onClick={() => setDraft({ ...EMPTY, category: departments[0]?.id })}
            className={cn(btnPrimary, "h-[38px] px-[18px]")}
          >
            <Plus className="h-[15px] w-[15px]" />
            {t("admin.kb.new")}
          </button>
        </div>

        {draft && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              saveMut.mutate(draft);
            }}
            className="flex flex-col gap-[14px] rounded-2xl border border-ring bg-card px-[22px] py-5"
          >
            <span className="text-[15px] font-semibold">
              {draft.id ? t("admin.kb.edit") : t("admin.kb.new")}
            </span>

            <div className="grid grid-cols-1 gap-[14px] sm:grid-cols-2">
              <label className="flex flex-col gap-[7px]">
                <span className="text-[12.5px] text-muted-foreground">
                  {t("common.title")}
                </span>
                <input
                  type="text"
                  required
                  value={draft.title ?? ""}
                  onChange={(e) => set({ title: e.target.value })}
                  className={fieldClass}
                />
              </label>
              <label className="flex flex-col gap-[7px]">
                <span className="text-[12.5px] text-muted-foreground">
                  {t("admin.kb.slug")}
                </span>
                <input
                  type="text"
                  value={draft.slug ?? ""}
                  onChange={(e) => set({ slug: e.target.value })}
                  placeholder="server-wont-start"
                  className={cn(fieldClass, "font-mono")}
                />
                <span className="text-[11.5px] text-muted-foreground/70">
                  {t("admin.kb.slug_hint")}
                </span>
              </label>
            </div>

            <div className="grid grid-cols-1 gap-[14px] sm:grid-cols-[1fr_140px_120px]">
              <label className="flex flex-col gap-[7px]">
                <span className="text-[12.5px] text-muted-foreground">
                  {t("admin.kb.department")}
                </span>
                <select
                  value={draft.category ?? ""}
                  onChange={(e) => set({ category: e.target.value })}
                  className={cn(fieldClass, "px-2.5")}
                >
                  {departments.map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.name}
                    </option>
                  ))}
                </select>
              </label>
              <label className="flex flex-col gap-[7px]">
                <span className="text-[12.5px] text-muted-foreground">
                  {t("admin.kb.position")}
                </span>
                <input
                  type="number"
                  value={draft.position ?? 0}
                  onChange={(e) => set({ position: Number(e.target.value) })}
                  className={cn(fieldClass, "font-mono")}
                />
              </label>
              <label className="flex flex-col gap-[7px]">
                <span className="text-[12.5px] text-muted-foreground">
                  {t("common.show")}
                </span>
                <button
                  type="button"
                  onClick={() => set({ published: !(draft.published ?? true) })}
                  className={cn(
                    btnGhost,
                    "h-[38px]",
                    (draft.published ?? true) && "border-ring bg-muted"
                  )}
                >
                  {(draft.published ?? true) ? (
                    <>
                      <Eye className="h-3.5 w-3.5" /> {t("common.yes")}
                    </>
                  ) : (
                    <>
                      <EyeOff className="h-3.5 w-3.5" /> {t("common.no")}
                    </>
                  )}
                </button>
              </label>
            </div>

            <label className="flex flex-col gap-[7px]">
              <span className="text-[12.5px] text-muted-foreground">
                {t("admin.kb.excerpt")}
              </span>
              <input
                type="text"
                value={draft.excerpt ?? ""}
                onChange={(e) => set({ excerpt: e.target.value })}
                placeholder={t("admin.kb.excerpt_placeholder")}
                className={fieldClass}
              />
            </label>

            <label className="flex flex-col gap-[7px]">
              <span className="text-[12.5px] text-muted-foreground">
                {t("admin.news.body")}
              </span>
              <textarea
                rows={12}
                value={draft.body ?? ""}
                onChange={(e) => set({ body: e.target.value })}
                className={cn(fieldClass, "h-auto resize-y py-2.5 leading-relaxed")}
              />
            </label>

            <div className="flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={() => setDraft(null)}
                className={cn(btnGhost, "h-[38px]")}
              >
                {t("common.cancel")}
              </button>
              <button
                type="submit"
                disabled={saveMut.isPending || !draft.title?.trim()}
                className={cn(btnPrimary, "h-[38px] px-[18px]")}
              >
                {saveMut.isPending ? t("common.saving") : t("common.save")}
              </button>
            </div>
          </form>
        )}

        {isLoading ? (
          <div className="flex flex-col gap-2.5">
            <Skeleton className="h-20 rounded-2xl" />
            <Skeleton className="h-20 rounded-2xl" />
          </div>
        ) : articles.length === 0 ? (
          <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-input bg-card px-5 py-12 text-center">
            <span className="flex h-[38px] w-[38px] items-center justify-center rounded-[10px] bg-muted text-muted-foreground">
              <BookOpen className="h-4 w-4" />
            </span>
            <div className="text-sm font-medium">{t("admin.kb.empty")}</div>
            <div className="max-w-md text-[12.5px] text-muted-foreground">
              {t("admin.kb.empty_hint")}
            </div>
          </div>
        ) : (
          <div className="flex flex-col gap-2.5">
            {articles.map((a) => (
              <div
                key={a.id}
                className="flex flex-col gap-2 rounded-2xl border border-border bg-card px-5 py-[18px]"
              >
                <div className="flex flex-wrap items-center gap-[10px]">
                  <span className="text-[15.5px] font-semibold">{a.title}</span>
                  <span className="rounded-full border border-border px-[9px] py-[3px] text-[11.5px] text-muted-foreground">
                    {departmentName(a.category, departments)}
                  </span>
                  {!a.published && (
                    <span
                      className="rounded-full border px-[9px] py-[3px] text-[11.5px]"
                      style={{
                        color: "var(--vx-warn)",
                        background: "var(--vx-warn-tint)",
                        borderColor:
                          "color-mix(in srgb, var(--vx-warn) 30%, transparent)",
                      }}
                    >
                      {t("admin.kb.draft")}
                    </span>
                  )}
                  <span className="ms-auto flex items-center gap-3">
                    <span className="flex items-center gap-1.5 text-[11.5px] text-muted-foreground/70">
                      <Eye className="h-3.5 w-3.5" />
                      {a.views}
                    </span>
                    <button
                      type="button"
                      onClick={() => setDraft(a)}
                      className={cn(btnGhost, "h-[30px] px-3 text-[12px]")}
                    >
                      {t("admin.kb.edit_action")}
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        if (
                          confirm(
                            t("admin.kb.delete_confirm", { title: a.title })
                          )
                        ) {
                          deleteMut.mutate(a.id);
                        }
                      }}
                      aria-label={t("admin.kb.delete_aria", { title: a.title })}
                      className="flex h-7 w-7 items-center justify-center rounded-lg border border-input text-muted-foreground transition-colors hover:border-[var(--vx-danger)] hover:text-[var(--vx-danger)]"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </span>
                </div>
                <div className="flex flex-wrap items-center gap-x-3 text-[12.5px] text-muted-foreground">
                  <span className="font-mono">/{a.slug}</span>
                  {a.excerpt && (
                    <>
                      <span className="text-muted-foreground/50">·</span>
                      <span className="truncate">{a.excerpt}</span>
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </PageShell>
  );
}
