"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Trash2 } from "lucide-react";
import { Btn, EmptyState, Panel, VX_TEXTAREA } from "@/components/vx/panel-ui";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/servers/confirm-dialog";
import {
  createAdminServerNote,
  deleteAdminServerNote,
  fetchAdminServerNotes,
  type AdminServerNote,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

export function AdminServerNotesTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();

  const [body, setBody] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<AdminServerNote | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminServerNotes(id),
    queryFn: () => fetchAdminServerNotes(id),
    enabled: !!id,
  });

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: queryKeys.adminServerNotes(id) });
  }

  function reportError(err: unknown) {
    toast.error(err instanceof Error ? err.message : t("common.error"));
  }

  const createMutation = useMutation({
    mutationFn: () => createAdminServerNote(id, body.trim()),
    onSuccess: () => {
      setBody("");
      toast.success(t("servers.admin.notes.saved"));
      refresh();
    },
    onError: reportError,
  });

  const deleteMutation = useMutation({
    mutationFn: (noteId: string) => deleteAdminServerNote(id, noteId),
    onSuccess: () => {
      setDeleteTarget(null);
      toast.success(t("servers.admin.notes.deleted"));
      refresh();
    },
    onError: reportError,
  });

  const notes = data?.notes ?? [];

  return (
    <Panel title={t("servers.admin.notes.title")}>
      <p className="mb-3 text-[12.5px] text-[var(--vx-muted)]">
        {t("servers.admin.notes.hint")}
      </p>

      <form
        className="mb-5 flex flex-col gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          if (!body.trim()) return;
          createMutation.mutate();
        }}
      >
        <textarea
          className={cn(VX_TEXTAREA, "min-h-[90px]")}
          placeholder={t("servers.admin.notes.placeholder")}
          value={body}
          maxLength={4000}
          onChange={(e) => setBody(e.target.value)}
        />
        <div className="flex justify-end">
          <Btn
            type="submit"
            tone="primary"
            disabled={createMutation.isPending || !body.trim()}
          >
            {createMutation.isPending ? t("common.saving") : t("servers.admin.notes.add")}
          </Btn>
        </div>
      </form>

      {isLoading ? (
        <Skeleton className="h-[160px] w-full rounded-[10px]" />
      ) : notes.length === 0 ? (
        <EmptyState>{t("servers.admin.notes.empty")}</EmptyState>
      ) : (
        <div className="flex flex-col gap-2.5">
          {notes.map((note) => (
            <div
              key={note.id}
              className="rounded-[10px] border border-[var(--vx-border)] bg-[var(--vx-elevated)] px-3.5 py-3"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="text-[11.5px] text-[var(--vx-muted)]">
                  {note.author ?? t("servers.admin.audit.system")} ·{" "}
                  {new Date(note.created_at).toLocaleString(localeTag())}
                </div>
                <button
                  type="button"
                  title={t("common.delete")}
                  className="text-[var(--vx-faint)] transition-colors hover:text-[var(--vx-danger)]"
                  onClick={() => setDeleteTarget(note)}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
              <div className="mt-1.5 text-[13px] whitespace-pre-wrap">{note.body}</div>
            </div>
          ))}
        </div>
      )}

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
        title={t("servers.admin.notes.delete_title")}
        description={t("servers.admin.notes.delete_description")}
        confirmLabel={t("common.delete")}
        pending={deleteMutation.isPending}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
      />
    </Panel>
  );
}
