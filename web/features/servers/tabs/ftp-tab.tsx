"use client";

import { useRef, useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Btn,
  EmptyState,
  Field,
  Panel,
  VX_FAINT,
  VX_INPUT,
  VX_MUTED,
  VX_ROW_LINE,
  VX_TEXTAREA,
} from "@/components/vx/panel-ui";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { FtpAccountsCard } from "@/features/servers/tabs/ftp-accounts-card";
import {
  deleteServerPath,
  downloadServerFile,
  listServerFiles,
  mkdirServerDir,
  readServerFile,
  uploadServerFile,
  writeServerFile,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

type FileRow = { name: string; is_dir: boolean; size: number };

function joinPath(base: string, name: string) {
  return base === "/" ? `/${name}` : `${base.replace(/\/$/, "")}/${name}`;
}

function formatSize(bytes: number) {
  if (!bytes || bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let size = bytes;
  let i = 0;
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024;
    i += 1;
  }
  return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

export function ServerFtpTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const [path, setPath] = useState("/");
  const [editorPath, setEditorPath] = useState<string | null>(null);
  const [editorContent, setEditorContent] = useState("");
  const [mkdirOpen, setMkdirOpen] = useState(false);
  const [mkdirName, setMkdirName] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploadingName, setUploadingName] = useState("");
  const [dragOver, setDragOver] = useState(false);
  const filePickerRef = useRef<HTMLInputElement | null>(null);

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: ["server-files", id, path],
    queryFn: () => listServerFiles(id, path),
    enabled: !!id,
  });

  const readMutation = useMutation({
    mutationFn: (filePath: string) => readServerFile(id, filePath),
    onSuccess: (res) => setEditorContent(res.content),
    onError: (err) => {
      toast.error(
        err instanceof Error ? err.message : t("servers.ftp.open_failed")
      );
      setEditorPath(null);
    },
  });

  const saveMutation = useMutation({
    mutationFn: () => writeServerFile(id, editorPath!, editorContent),
    onSuccess: () => {
      toast.success(t("servers.ftp.file_saved"));
      void queryClient.invalidateQueries({ queryKey: ["server-files", id] });
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.save_failed")),
  });

  const mkdirMutation = useMutation({
    mutationFn: (p: string) => mkdirServerDir(id, p),
    onSuccess: () => {
      toast.success(t("servers.ftp.dir_created"));
      setMkdirOpen(false);
      setMkdirName("");
      void queryClient.invalidateQueries({ queryKey: ["server-files", id] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.ftp.mkdir_error")
      ),
  });

  const deleteMutation = useMutation({
    mutationFn: (p: string) => deleteServerPath(id, p),
    onSuccess: () => {
      toast.success(t("servers.ftp.deleted"));
      setDeleteTarget(null);
      void queryClient.invalidateQueries({ queryKey: ["server-files", id] });
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.delete_failed")),
  });

  const files = (data?.files ?? []) as FileRow[];

  async function openFile(name: string) {
    const filePath = joinPath(path, name);
    setEditorPath(filePath);
    setEditorContent("");
    await readMutation.mutateAsync(filePath);
  }

  async function onUpload(file: File) {
    try {
      setUploadingName(file.name);
      setUploadProgress(0);
      await uploadServerFile(id, path, file, (pct) => setUploadProgress(pct));
      toast.success(t("servers.ftp.uploaded", { name: file.name }));
      void queryClient.invalidateQueries({ queryKey: ["server-files", id] });
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("servers.ftp.upload_error")
      );
    } finally {
      setUploadingName("");
      setUploadProgress(0);
      if (filePickerRef.current) filePickerRef.current.value = "";
    }
  }

  async function onDownload(filePath: string) {
    try {
      await downloadServerFile(id, filePath);
      toast.success(t("servers.ftp.downloaded"));
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("servers.ftp.download_error")
      );
    }
  }

  if (isLoading) return <Skeleton className="h-[420px] w-full rounded-[14px]" />;

  return (
    <>
      <FtpAccountsCard serverId={id} />
      <Panel
        title={<span className="font-mono text-[12.5px]">{path}</span>}
        aside={
          <div className="flex flex-wrap gap-2">
            <Btn
              size="sm"
              disabled={path === "/"}
              onClick={() => setPath(path.split("/").slice(0, -1).join("/") || "/")}
            >
              {t("servers.ftp.up")}
            </Btn>
            <Btn
              size="sm"
              onClick={() => {
                setMkdirName("");
                setMkdirOpen(true);
              }}
            >
              {t("servers.ftp.new_dir")}
            </Btn>
            <Btn size="sm" onClick={() => refetch()} disabled={isFetching}>
              {isFetching ? t("common.updating") : t("common.refresh")}
            </Btn>
            <Btn
              size="sm"
              tone="primary"
              onClick={() => filePickerRef.current?.click()}
              disabled={!!uploadingName}
            >
              {uploadingName ? t("common.loading") : t("common.upload")}
            </Btn>
          </div>
        }
      >
        <input
          ref={filePickerRef}
          type="file"
          className="hidden"
          onChange={(e) => {
            const file = e.target.files?.[0];
            if (file) void onUpload(file);
          }}
        />

        <div
          className={cn(
            "rounded-[10px] border border-dashed p-3.5 text-[12px] transition-colors",
            dragOver ? "border-[var(--vx-fg-strong)] bg-[var(--vx-veil)]" : cn("border-[var(--vx-border-2)]", VX_FAINT)
          )}
          onDragOver={(e) => {
            e.preventDefault();
            setDragOver(true);
          }}
          onDragLeave={() => setDragOver(false)}
          onDrop={(e) => {
            e.preventDefault();
            setDragOver(false);
            const file = e.dataTransfer.files?.[0];
            if (file) void onUpload(file);
          }}
        >
          {t("servers.ftp.drop_hint")}
          {uploadingName && (
            <div className="mt-2">
              <div className="mb-1 font-mono text-[11px]">
                {uploadingName}: {uploadProgress}%
              </div>
              <div className="h-1.5 overflow-hidden rounded-full bg-[var(--vx-border)]">
                <div
                  className="h-full bg-[var(--vx-fg-strong)] transition-[width]"
                  style={{ width: `${uploadProgress}%` }}
                />
              </div>
            </div>
          )}
        </div>

        <div className="mt-3.5 overflow-hidden rounded-[10px] border border-[var(--vx-inset)]">
          {files.length === 0 ? (
            <EmptyState>{t("servers.ftp.empty")}</EmptyState>
          ) : (
            files.map((file) => {
              const fullPath = joinPath(path, file.name);
              return (
                <div
                  key={file.name}
                  className={cn(
                    "grid grid-cols-[22px_minmax(0,1fr)_90px_auto] items-center gap-3 px-3.5 py-2.5 text-[12.5px] last:border-b-0",
                    VX_ROW_LINE
                  )}
                >
                  <span className={cn("font-mono", VX_FAINT)}>{file.is_dir ? "▸" : "·"}</span>
                  <button
                    type="button"
                    onClick={() => (file.is_dir ? setPath(fullPath) : openFile(file.name))}
                    className="truncate text-left text-[12.5px] text-[var(--vx-fg)] transition-colors hover:text-white"
                  >
                    {file.name}
                  </button>
                  <span className={cn("font-mono text-[11.5px]", VX_FAINT)}>
                    {file.is_dir ? t("servers.ftp.folder") : formatSize(file.size)}
                  </span>
                  <span className="flex justify-end gap-1.5">
                    <Btn
                      size="sm"
                      tone="ghost"
                      className="h-[26px] rounded-[7px] border-[var(--vx-border-2)] px-2.5 text-[11.5px]"
                      disabled={file.is_dir || !!uploadingName}
                      title={
                        file.is_dir
                          ? t("servers.ftp.dir_download_unsupported")
                          : t("common.download")
                      }
                      onClick={() => void onDownload(fullPath)}
                    >
                      {t("common.download")}
                    </Btn>
                    <Btn
                      size="sm"
                      className="h-[26px] rounded-[7px] border-[rgba(224,122,122,0.3)] bg-transparent px-2.5 text-[11.5px] text-[var(--vx-danger)]"
                      disabled={deleteMutation.isPending}
                      onClick={() => setDeleteTarget(fullPath)}
                    >
                      {t("common.delete")}
                    </Btn>
                  </span>
                </div>
              );
            })
          )}
        </div>
      </Panel>

      <Dialog open={editorPath !== null} onOpenChange={(open) => !open && setEditorPath(null)}>
        <DialogContent className="max-h-[85vh] max-w-2xl">
          <DialogHeader>
            <DialogTitle className="truncate font-mono text-[13px]">{editorPath}</DialogTitle>
          </DialogHeader>
          {readMutation.isPending ? (
            <Skeleton className="h-64 w-full" />
          ) : (
            <textarea
              value={editorContent}
              onChange={(e) => setEditorContent(e.target.value)}
              className={cn(VX_TEXTAREA, "min-h-[320px] resize-y")}
              spellCheck={false}
            />
          )}
          <DialogFooter>
            <Btn onClick={() => setEditorPath(null)}>{t("common.close")}</Btn>
            <Btn
              tone="primary"
              onClick={() => saveMutation.mutate()}
              disabled={saveMutation.isPending || readMutation.isPending}
            >
              {saveMutation.isPending ? t("common.saving") : t("common.save")}
            </Btn>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={mkdirOpen} onOpenChange={setMkdirOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("servers.ftp.mkdir_title", { path })}</DialogTitle>
          </DialogHeader>
          <Field label={t("servers.ftp.dir_name")}>
            <input
              className={VX_INPUT}
              value={mkdirName}
              onChange={(e) => setMkdirName(e.target.value)}
              placeholder="newdir"
              onKeyDown={(e) => {
                if (e.key === "Enter" && mkdirName.trim()) {
                  mkdirMutation.mutate(joinPath(path, mkdirName.trim()));
                }
              }}
            />
          </Field>
          <DialogFooter>
            <Btn onClick={() => setMkdirOpen(false)}>{t("common.cancel")}</Btn>
            <Btn
              tone="primary"
              onClick={() => mkdirMutation.mutate(joinPath(path, mkdirName.trim()))}
              disabled={mkdirMutation.isPending || !mkdirName.trim()}
            >
              {mkdirMutation.isPending
                ? t("common.creating")
                : t("common.create")}
            </Btn>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteTarget !== null} onOpenChange={(o) => !o && setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("servers.ftp.delete_title")}</DialogTitle>
          </DialogHeader>
          <p className={cn("font-mono text-[12.5px] break-all", VX_MUTED)}>{deleteTarget}</p>
          <DialogFooter>
            <Btn onClick={() => setDeleteTarget(null)}>{t("common.cancel")}</Btn>
            <Btn
              tone="danger"
              onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget)}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending
                ? t("common.deleting")
                : t("common.delete")}
            </Btn>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
