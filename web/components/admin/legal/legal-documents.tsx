"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { LegalText } from "@/components/legal/legal-text";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import {
  fetchAdminLegalDocuments,
  fetchAdminLegalDocumentVersion,
  fetchAdminLegalTemplate,
  publishAdminLegalDocument,
  type LegalKind,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";

const KEY = ["admin-legal-documents"];

function fmt(iso: string) {
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? iso : date.toLocaleString(localeTag());
}

export function LegalDocuments() {
  const t = useT();
  const qc = useQueryClient();
  const query = useQuery({ queryKey: KEY, queryFn: fetchAdminLegalDocuments });
  const [kind, setKind] = useState<LegalKind>("offer");
  const [loadedKind, setLoadedKind] = useState("");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [requires, setRequires] = useState(true);
  const [preview, setPreview] = useState(false);
  const [viewing, setViewing] = useState<{ version: number; title: string; body: string } | null>(null);

  const items = query.data?.documents ?? [];
  const item = items.find((doc) => doc.kind === kind);

  useEffect(() => {
    if (!item || loadedKind === kind) return;
    setTitle(item.current?.title ?? item.default_title);
    setBody(item.current?.body ?? "");
    setRequires(true);
    setPreview(false);
    setViewing(null);
    setLoadedKind(kind);
  }, [item, kind, loadedKind]);

  const publishMut = useMutation({
    mutationFn: () => publishAdminLegalDocument(kind, { title, body, requires_acceptance: requires }),
    onSuccess: (doc) => {
      toast.success(t("admin.legal.published", { version: doc.version }));
      void qc.invalidateQueries({ queryKey: KEY });
    },
    onError: (e: Error) => toast.error(e.message || t("admin.legal.publish_failed")),
  });

  const insertTemplate = async () => {
    if (body.trim() && !window.confirm(t("admin.legal.template_confirm"))) return;
    try {
      const template = await fetchAdminLegalTemplate(kind);
      setTitle(template.title);
      setBody(template.body);
      setPreview(false);
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : t("admin.legal.template_failed"));
    }
  };

  const openVersion = async (version: number) => {
    try {
      const doc = await fetchAdminLegalDocumentVersion(kind, version);
      setViewing({ version: doc.version, title: doc.title, body: doc.body ?? "" });
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : t("admin.legal.version_failed"));
    }
  };

  if (query.isLoading) {
    return <Skeleton className="h-96 w-full" />;
  }

  return (
    <div className="grid gap-4 lg:grid-cols-[280px_1fr]">
      <div className="space-y-2">
        {items.map((doc) => (
          <button
            key={doc.kind}
            type="button"
            onClick={() => setKind(doc.kind)}
            className={cn(
              "w-full rounded-lg border bg-card p-3 text-left transition-colors hover:bg-muted/50",
              kind === doc.kind && "border-primary"
            )}
          >
            <div className="text-sm font-medium">{t(`admin.legal.kind.${doc.kind}`)}</div>
            <div className="mt-1 text-xs text-muted-foreground">
              {doc.current
                ? t("admin.legal.current_version", {
                    version: doc.current.version,
                    date: fmt(doc.current.published_at),
                  })
                : t("admin.legal.not_published")}
            </div>
          </button>
        ))}
        <p className="px-1 text-xs text-muted-foreground">{t("admin.legal.lawyer_hint")}</p>
      </div>

      <div className="space-y-4">
        <div className="space-y-3 rounded-lg border bg-card p-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium">{t(`admin.legal.kind.${kind}`)}</span>
              {item?.current ? (
                <Badge variant="secondary">v{item.current.version}</Badge>
              ) : (
                <Badge variant="outline">{t("admin.legal.draft")}</Badge>
              )}
            </div>
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" onClick={() => void insertTemplate()}>
                {t("admin.legal.insert_template")}
              </Button>
              <Button variant="outline" size="sm" onClick={() => setPreview((v) => !v)}>
                {preview ? t("admin.legal.edit") : t("admin.legal.preview")}
              </Button>
              {item?.current ? (
                <Button variant="outline" size="sm" asChild>
                  <a href={`/legal/${kind}`} target="_blank" rel="noopener noreferrer">
                    {t("admin.legal.open_public")}
                  </a>
                </Button>
              ) : null}
            </div>
          </div>

          <div className="space-y-1">
            <Label>{t("admin.legal.title_label")}</Label>
            <Input value={title} onChange={(e) => setTitle(e.target.value)} />
          </div>

          {preview ? (
            <div className="max-h-[640px] overflow-y-auto rounded-md border p-4">
              <LegalText body={body} />
            </div>
          ) : (
            <div className="space-y-1">
              <Label>{t("admin.legal.body_label")}</Label>
              <Textarea
                value={body}
                onChange={(e) => setBody(e.target.value)}
                rows={24}
                className="font-mono text-xs"
              />
              <p className="text-xs text-muted-foreground">{t("admin.legal.markup_hint")}</p>
            </div>
          )}

          {item?.acceptance && item.current ? (
            <label className="flex items-start gap-2 text-sm">
              <Checkbox
                className="mt-0.5"
                checked={requires}
                onCheckedChange={(value) => setRequires(value === true)}
              />
              <span>
                {t("admin.legal.requires_acceptance")}
                <span className="mt-0.5 block text-xs text-muted-foreground">
                  {t("admin.legal.requires_acceptance_hint")}
                </span>
              </span>
            </label>
          ) : null}

          <Button
            disabled={publishMut.isPending || !body.trim()}
            onClick={() => {
              if (window.confirm(t("admin.legal.publish_confirm"))) publishMut.mutate();
            }}
          >
            {t("admin.legal.publish")}
          </Button>
        </div>

        {item && item.versions.length > 0 ? (
          <div className="rounded-lg border bg-card">
            <div className="border-b p-4 text-sm font-medium">{t("admin.legal.history")}</div>
            <div className="divide-y">
              {item.versions.map((version) => (
                <div
                  key={version.version}
                  className="flex flex-wrap items-center justify-between gap-2 px-4 py-2 text-sm"
                >
                  <span>
                    {t("admin.legal.version_line", {
                      version: version.version,
                      date: fmt(version.published_at),
                      author: version.published_by || "—",
                    })}
                    {version.requires_acceptance ? "" : ` · ${t("admin.legal.minor_edit")}`}
                  </span>
                  <Button variant="ghost" size="sm" onClick={() => void openVersion(version.version)}>
                    {t("admin.legal.open_version")}
                  </Button>
                </div>
              ))}
            </div>
          </div>
        ) : null}

        {viewing ? (
          <div className="rounded-lg border bg-card p-4">
            <div className="mb-3 flex items-center justify-between gap-2">
              <span className="text-sm font-medium">
                {viewing.title} · v{viewing.version}
              </span>
              <Button variant="ghost" size="sm" onClick={() => setViewing(null)}>
                {t("admin.legal.close")}
              </Button>
            </div>
            <div className="max-h-[480px] overflow-y-auto">
              <LegalText body={viewing.body} />
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}
