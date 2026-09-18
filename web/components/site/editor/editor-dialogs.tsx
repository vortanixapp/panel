"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { History, Loader2, RotateCcw } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Textarea } from "@/components/ui/textarea";
import { useEditor } from "@/components/site/editor/editor-context";
import { localeTag } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { fetchTemplateVersions } from "@/lib/site/api";

function formatDate(iso: string | null | undefined): string {
  if (!iso) return "";
  try {
    return new Date(iso).toLocaleString(localeTag(), { dateStyle: "medium", timeStyle: "short" });
  } catch {
    return iso;
  }
}

export function PublishDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { editor, t } = useEditor();
  const queryClient = useQueryClient();
  const [note, setNote] = useState("");
  const texts = editor.status?.pending_texts ?? 0;

  const publish = async () => {
    try {
      await editor.publish(note.trim());
      setNote("");
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminTemplateVersions });
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminSiteMenu });
      toast.success(t("template.publish.done"));
      onClose();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("template.publish.failed"));
    }
  };

  return (
    <Dialog open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("template.publish.title")}</DialogTitle>
          <DialogDescription>{t("template.publish.hint")}</DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <ul className="space-y-1 rounded-lg border bg-muted/30 px-3 py-2.5 text-xs">
            <li>{editor.status?.changed || editor.dirty ? t("template.publish.structure") : t("template.publish.no_structure")}</li>
            <li>{t("template.publish.texts", { count: texts })}</li>
          </ul>
          <Textarea
            value={note}
            maxLength={300}
            rows={3}
            placeholder={t("template.publish.note")}
            onChange={(event) => setNote(event.target.value)}
            className="text-sm"
          />
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} disabled={editor.busy}>
            {t("common.cancel")}
          </Button>
          <Button type="button" onClick={() => void publish()} disabled={editor.busy}>
            {editor.busy ? <Loader2 className="size-4 animate-spin" /> : null}
            {t("template.publish.submit")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function VersionsSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { editor, t, select } = useEditor();
  const [armed, setArmed] = useState<number | null>(null);
  const query = useQuery({
    queryKey: queryKeys.adminTemplateVersions,
    queryFn: fetchTemplateVersions,
    enabled: open,
  });
  const items = query.data?.items ?? [];

  const restore = async (id: number) => {
    try {
      await editor.restore(id);
      select(null);
      setArmed(null);
      toast.success(t("template.versions.restored"));
      onClose();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("template.versions.failed"));
    }
  };

  return (
    <Sheet open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
      <SheetContent className="flex w-full flex-col gap-0 p-0 sm:max-w-md">
        <SheetHeader className="border-b">
          <SheetTitle>{t("template.versions.title")}</SheetTitle>
          <SheetDescription>{t("template.versions.hint")}</SheetDescription>
        </SheetHeader>
        <div className="min-h-0 flex-1 overflow-y-auto p-4">
          {query.isLoading ? (
            <div className="flex justify-center py-10">
              <Loader2 className="size-5 animate-spin text-muted-foreground" />
            </div>
          ) : items.length === 0 ? (
            <div className="flex flex-col items-center gap-2 py-10 text-center text-sm text-muted-foreground">
              <History className="size-6 opacity-60" />
              {t("template.versions.empty")}
            </div>
          ) : (
            <ol className="space-y-2">
              {items.map((version, index) => (
                <li key={version.id} className="rounded-xl border p-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="text-sm font-medium">
                        {formatDate(version.created_at)}
                        {index === 0 ? (
                          <span className="ml-2 rounded-full bg-primary/10 px-2 py-0.5 text-[10.5px] font-medium text-primary">
                            {t("template.versions.current")}
                          </span>
                        ) : null}
                      </div>
                      <div className="mt-0.5 truncate text-xs text-muted-foreground">
                        {version.author || t("template.versions.unknown")}
                        {version.texts > 0 ? ` · ${t("template.versions.texts", { count: version.texts })}` : ""}
                      </div>
                      {version.note ? <p className="mt-1.5 text-xs leading-relaxed">{version.note}</p> : null}
                    </div>
                    <Button
                      type="button"
                      size="sm"
                      variant={armed === version.id ? "default" : "outline"}
                      disabled={editor.busy}
                      onClick={() => (armed === version.id ? void restore(version.id) : setArmed(version.id))}
                      className="shrink-0"
                    >
                      <RotateCcw className="size-3.5" />
                      {armed === version.id ? t("template.versions.confirm") : t("template.versions.restore")}
                    </Button>
                  </div>
                </li>
              ))}
            </ol>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

export { formatDate };
