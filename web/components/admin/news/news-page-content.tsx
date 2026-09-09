"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { NEWS_TAGS, NewsTag } from "@/components/user/news/news-parts";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import {
  createAdminNews,
  deleteAdminNews,
  deleteAdminNewsImage,
  fetchAdminNews,
  updateAdminNews,
  uploadAdminNewsImage,
  type AdminNewsItem,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

type NewsForm = {
  id?: string;
  slug: string;
  title: string;
  excerpt: string;
  body: string;
  published_at: string;
  active: boolean;
  tag: string;
  pinned: boolean;
};

const EMPTY: NewsForm = {
  slug: "",
  title: "",
  excerpt: "",
  body: "",
  published_at: "",
  active: true,
  tag: "update",
  pinned: false,
};

function toLocalInput(iso: string | null | undefined): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function NewsPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["admin-news"],
    queryFn: async () => (await fetchAdminNews()).news,
  });
  const items = data ?? [];

  const [form, setForm] = useState<NewsForm | null>(null);
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [editingImages, setEditingImages] = useState<AdminNewsItem["images"]>([]);

  const invalidate = () => qc.invalidateQueries({ queryKey: ["admin-news"] });

  const save = useMutation({
    mutationFn: async (f: NewsForm) => {
      const payload = {
        slug: f.slug.trim() || undefined,
        title: f.title,
        excerpt: f.excerpt,
        body: f.body,
        published_at: f.published_at,
        active: f.active,
        tag: f.tag,
        pinned: f.pinned,
      };
      const id = f.id ?? (await createAdminNews(payload)).id;
      if (f.id) await updateAdminNews(f.id, payload);
      if (id && imageFile) await uploadAdminNewsImage(id, imageFile);
    },
    onSuccess: () => {
      toast.success(form?.id ? t("admin.news.saved") : t("admin.news.created"));
      setForm(null);
      setImageFile(null);
      void invalidate();
    },
    onError: (e: Error) => toast.error(e.message || t("common.save_failed")),
  });

  const removeImage = useMutation({
    mutationFn: ({ newsId, imageId }: { newsId: string; imageId: string }) =>
      deleteAdminNewsImage(newsId, imageId),
    onSuccess: (_, vars) => {
      setEditingImages((prev) => prev.filter((i) => i.id !== vars.imageId));
      void invalidate();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const remove = useMutation({
    mutationFn: (id: string) => deleteAdminNews(id),
    onSuccess: () => {
      toast.success(t("admin.news.deleted"));
      void invalidate();
    },
    onError: (e: Error) => toast.error(e.message || t("common.delete_failed")),
  });

  function startEdit(n: AdminNewsItem) {
    setImageFile(null);
    setEditingImages(n.images ?? []);
    setForm({
      id: n.id,
      slug: n.slug,
      title: n.title,
      excerpt: n.excerpt ?? "",
      body: n.body ?? "",
      published_at: toLocalInput(n.published_at),
      active: n.active,
      tag: n.tag ?? "update",
      pinned: n.pinned ?? false,
    });
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form) return;
    if (form.title.trim().length < 2) {
      toast.error(t("admin.news.title_required"));
      return;
    }
    save.mutate(form);
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("admin.news.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.news.subtitle")}
          </p>
        </div>
        <Button
          onClick={() => {
            setForm({ ...EMPTY });
            setImageFile(null);
            setEditingImages([]);
          }}
        >
          + {t("common.create")}
        </Button>
      </div>

      {form && (
        <form onSubmit={onSubmit} className="mb-6 space-y-3 rounded-lg border bg-card p-4">
          <div className="grid gap-3 md:grid-cols-2">
            <div>
              <Label>{t("common.title")}</Label>
              <Input
                value={form.title}
                onChange={(e) => setForm({ ...form, title: e.target.value })}
                required
              />
            </div>
            <div>
              <Label>{t("admin.news.slug")}</Label>
              <Input
                value={form.slug}
                placeholder={t("admin.news.slug_placeholder")}
                onChange={(e) => setForm({ ...form, slug: e.target.value })}
              />
              <p className="mt-1 text-[11px] text-muted-foreground">
                {t("admin.news.slug_hint")}
              </p>
            </div>
          </div>

          <div>
            <Label>{t("admin.news.excerpt")}</Label>
            <Input
              value={form.excerpt}
              maxLength={255}
              onChange={(e) => setForm({ ...form, excerpt: e.target.value })}
            />
          </div>

          <div>
            <Label>{t("admin.news.body")}</Label>
            <Textarea
              value={form.body}
              onChange={(e) => setForm({ ...form, body: e.target.value })}
              className="min-h-[160px]"
            />
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            <div>
              <Label>{t("admin.news.published_at")}</Label>
              <Input
                type="datetime-local"
                value={form.published_at}
                onChange={(e) => setForm({ ...form, published_at: e.target.value })}
              />
              <p className="mt-1 text-[11px] text-muted-foreground">
                {t("admin.news.draft_hint")}
              </p>
            </div>
            <div>
              <Label>{t("admin.news.image")}</Label>
              <Input
                type="file"
                accept="image/*"
                onChange={(e) => setImageFile(e.target.files?.[0] ?? null)}
              />
              <p className="mt-1 text-[11px] text-muted-foreground">
                {t("admin.news.image_hint")}
              </p>
            </div>
          </div>

          {editingImages.length > 0 && form.id && (
            <div className="flex flex-wrap gap-3">
              {editingImages.map((img) => (
                <div key={img.id} className="relative">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img src={img.url} alt="" className="h-20 w-32 rounded object-cover" />
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    className="mt-1 w-full"
                    disabled={removeImage.isPending}
                    onClick={() => removeImage.mutate({ newsId: form.id!, imageId: img.id })}
                  >
                    {t("admin.news.remove_image")}
                  </Button>
                </div>
              ))}
            </div>
          )}

          <div className="space-y-1.5">
            <Label>{t("admin.news.tag")}</Label>
            <div className="flex flex-wrap gap-1.5">
              {NEWS_TAGS.map((tag) => (
                <button
                  key={tag.id}
                  type="button"
                  onClick={() => setForm({ ...form, tag: tag.id })}
                  aria-pressed={form.tag === tag.id}
                  className={cn(
                    "rounded-[9px] border px-3 py-1.5 text-[12.5px] transition-colors",
                    form.tag === tag.id
                      ? "border-ring bg-muted"
                      : "border-border hover:border-ring"
                  )}
                  style={
                    form.tag === tag.id ? { color: `var(${tag.token})` } : undefined
                  }
                >
                  {tag.label}
                </button>
              ))}
            </div>
            <p className="text-xs text-muted-foreground">
              {t("admin.news.tag_hint")}
            </p>
          </div>

          <div className="flex flex-wrap gap-x-6 gap-y-2">
            <label className="flex items-center gap-2 text-sm">
              <Switch checked={form.active} onCheckedChange={(v) => setForm({ ...form, active: v })} />
              {t("admin.news.show_to_clients")}
            </label>
            <label className="flex items-center gap-2 text-sm">
              <Switch checked={form.pinned} onCheckedChange={(v) => setForm({ ...form, pinned: v })} />
              {t("admin.news.pin")}
            </label>
          </div>

          <div className="flex gap-2">
            <Button type="submit" disabled={save.isPending}>
              {save.isPending
                ? t("common.saving")
                : form.id
                  ? t("common.save")
                  : t("common.create")}
            </Button>
            <Button type="button" variant="ghost" onClick={() => setForm(null)}>
              {t("common.cancel")}
            </Button>
          </div>
        </form>
      )}

      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
      ) : items.length === 0 ? (
        <div className="rounded-lg border bg-card p-8 text-center text-sm text-muted-foreground">
          {t("admin.news.empty")}
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border bg-card">
          {items.map((n) => (
            <div key={n.id} className="flex items-center gap-4 border-b p-3 last:border-0">
              {n.image ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={n.image} alt="" className="h-12 w-20 rounded object-cover" />
              ) : (
                <div className="h-12 w-20 rounded bg-muted" />
              )}
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium">{n.title}</span>
                  <span className="font-mono text-xs text-muted-foreground">{n.slug}</span>
                  <NewsTag tag={n.tag} />
                  {n.pinned && (
                    <Badge variant="outline">{t("admin.news.pinned")}</Badge>
                  )}
                  {!n.active && (
                    <Badge variant="secondary">{t("admin.news.hidden")}</Badge>
                  )}
                  {!n.published_at && (
                    <Badge variant="outline">{t("admin.news.draft")}</Badge>
                  )}
                </div>
                <div className="mt-0.5 text-xs text-muted-foreground">
                  {n.published_at
                    ? new Date(n.published_at).toLocaleString(localeTag())
                    : t("admin.news.not_published")}
                  {n.images?.length
                    ? t("admin.news.images_count", { count: n.images.length })
                    : ""}
                </div>
              </div>
              <div className="flex gap-2">
                <Button size="sm" variant="outline" onClick={() => startEdit(n)}>
                  {t("common.edit")}
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={remove.isPending}
                  onClick={() => {
                    if (
                      !confirm(
                        t("admin.news.delete_confirm", { title: n.title })
                      )
                    )
                      return;
                    remove.mutate(n.id);
                  }}
                >
                  {t("common.delete")}
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </PageShell>
  );
}
