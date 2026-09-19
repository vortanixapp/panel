"use client";

import { useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  createAdminSiteFile,
  deleteAdminSiteFile,
  fetchAdminSiteFiles,
  uploadAdminSiteFile,
} from "@/lib/api";
import { dateLocaleTag } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import { FieldGrid, SettingsCard, TextAreaField, TextField } from "./settings-ui";

function errorText(err: unknown, fallback: string) {
  return err instanceof Error && err.message ? err.message : fallback;
}

function fileDate(iso: string) {
  const d = new Date(iso);
  return Number.isNaN(d.getTime())
    ? "—"
    : d.toLocaleString(dateLocaleTag(), {
        day: "2-digit",
        month: "2-digit",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      });
}

export function SiteFilesCard() {
  const t = useT();
  const qc = useQueryClient();
  const pickerRef = useRef<HTMLInputElement | null>(null);
  const [manual, setManual] = useState(false);
  const [name, setName] = useState("");
  const [content, setContent] = useState("");

  const { data: files = [], isLoading } = useQuery({
    queryKey: queryKeys.adminSiteFiles,
    queryFn: fetchAdminSiteFiles,
  });

  const refresh = () => qc.invalidateQueries({ queryKey: queryKeys.adminSiteFiles });

  const uploadMut = useMutation({
    mutationFn: uploadAdminSiteFile,
    onSuccess: (file) => {
      toast.success(t("admin.settings.site_files.saved", { name: file.name }));
      void refresh();
    },
    onError: (err) => toast.error(errorText(err, t("admin.settings.site_files.save_failed"))),
    onSettled: () => {
      if (pickerRef.current) pickerRef.current.value = "";
    },
  });

  const createMut = useMutation({
    mutationFn: () => createAdminSiteFile(name.trim(), content),
    onSuccess: (file) => {
      toast.success(t("admin.settings.site_files.saved", { name: file.name }));
      setName("");
      setContent("");
      setManual(false);
      void refresh();
    },
    onError: (err) => toast.error(errorText(err, t("admin.settings.site_files.save_failed"))),
  });

  const deleteMut = useMutation({
    mutationFn: deleteAdminSiteFile,
    onSuccess: () => {
      toast.success(t("admin.settings.site_files.deleted"));
      void refresh();
    },
    onError: (err) => toast.error(errorText(err, t("common.delete_failed"))),
  });

  return (
    <SettingsCard
      title={t("admin.settings.site_files.title")}
      description={t("admin.settings.site_files.description")}
      action={
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" onClick={() => setManual((open) => !open)}>
            {t("admin.settings.site_files.create")}
          </Button>
          <Button
            size="sm"
            disabled={uploadMut.isPending}
            onClick={() => pickerRef.current?.click()}
          >
            {uploadMut.isPending ? t("common.loading") : t("admin.settings.site_files.upload")}
          </Button>
        </div>
      }
    >
      <input
        ref={pickerRef}
        type="file"
        accept=".html,.htm,.txt,.xml"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) uploadMut.mutate(file);
        }}
      />

      {manual && (
        <div className="mb-4 space-y-3 rounded-xl border p-4">
          <FieldGrid>
            <TextField
              mono
              span="full"
              label={t("admin.settings.site_files.name")}
              value={name}
              onChange={setName}
              placeholder="enot_9a3201b4.html"
            />
            <TextAreaField
              mono
              span="full"
              rows={5}
              label={t("admin.settings.site_files.content")}
              value={content}
              onChange={setContent}
            />
          </FieldGrid>
          <div className="flex flex-wrap justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => setManual(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              size="sm"
              disabled={createMut.isPending || !name.trim() || !content}
              onClick={() => createMut.mutate()}
            >
              {createMut.isPending ? t("common.saving") : t("admin.settings.site_files.save")}
            </Button>
          </div>
        </div>
      )}

      {isLoading ? (
        <Skeleton className="h-14 w-full" />
      ) : files.length === 0 ? (
        <p className="text-[13px] text-muted-foreground">{t("admin.settings.site_files.empty")}</p>
      ) : (
        <ul className="divide-y rounded-xl border">
          {files.map((file) => (
            <li key={file.name} className="flex flex-wrap items-center gap-x-3 gap-y-2 px-4 py-3">
              <div className="min-w-0 flex-1">
                <a
                  href={`/${file.name}`}
                  target="_blank"
                  rel="noreferrer"
                  className="font-mono text-[13px] break-all hover:underline"
                >
                  {`${window.location.origin}/${file.name}`}
                </a>
                <div className="mt-0.5 text-[11.5px] text-muted-foreground">
                  {t("admin.settings.site_files.meta", {
                    size: file.size,
                    date: fileDate(file.updated_at),
                  })}
                </div>
              </div>
              <Button
                variant="outline"
                size="sm"
                disabled={deleteMut.isPending}
                onClick={() => {
                  if (confirm(t("admin.settings.site_files.delete_confirm", { name: file.name }))) {
                    deleteMut.mutate(file.name);
                  }
                }}
              >
                {t("common.delete")}
              </Button>
            </li>
          ))}
        </ul>
      )}
    </SettingsCard>
  );
}
