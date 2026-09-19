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
        title={<span className="font-mono text-[12.5px] break-all">{path}</span>}
        aside={
          <div className="grid w-full grid-cols-2 gap-2 sm:flex sm:w-auto sm:flex-wrap">
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
          role="button"
          tabIndex={0}
          className={cn(
            "cursor-pointer rounded-[10px] border border-dashed p-3.5 text-[12px] transition-colors hover:border-[var(--vx-border-hover)]",
            dragOver ? "border-[var(--vx-fg-strong)] bg-[var(--vx-veil)]" : cn("border-[var(--vx-border-2)]", VX_FAINT)
          )}
          onClick={() => {
            if (!uploadingName) filePickerRef.current?.click();
          }}
          onKeyDown={(e) => {
            if ((e.key === "Enter" || e.key === " ") && !uploadingName) {
              e.preventDefault();
              filePickerRef.current?.click();
            }
          }}
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
          <span className="hidden sm:inline">{t("servers.ftp.drop_hint")}</span>
          <span className="sm:hidden">{t("servers.ftp.tap_hint")}</span>
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
              const meta = file.is_dir ? t("servers.ftp.folder") : formatSize(file.size);
              return (
                <div
                  key={file.name}
                  className={cn(
                    "flex items-center gap-2.5 px-3 py-2 text-[12.5px] last:border-b-0 sm:gap-3 sm:px-3.5 sm:py-2.5",
                    VX_ROW_LINE
                  )}
                >
                  <i
                    className={cn(
                      file.is_dir ? "ri-folder-3-line" : "ri-file-3-line",
                      "flex-shrink-0 text-[15px]",
                      VX_FAINT
                    )}
                  />
                  <button
                    type="button"
                    onClick={() => (file.is_dir ? setPath(fullPath) : openFile(file.name))}
                    className="min-w-0 flex-1 py-0.5 text-left transition-colors hover:text-white"
                  >
                    <span className="line-clamp-2 text-[13px] break-all text-[var(--vx-fg)] sm:line-clamp-none sm:block sm:truncate sm:text-[12.5px] sm:break-normal">
                      {file.name}
                    </span>
                    <span className={cn("block font-mono text-[11px] sm:hidden", VX_FAINT)}>
                      {meta}
                    </span>
                  </button>
                  <span
                    className={cn(
                      "hidden w-[90px] flex-shrink-0 font-mono text-[11.5px] sm:block",
                      VX_FAINT
                    )}
                  >
                    {meta}
                  </span>
                  <span className="flex flex-shrink-0 items-center gap-1 sm:gap-1.5">
                    <Btn
                      size="sm"
                      tone="ghost"
                      className={cn(
                        "size-8 rounded-[8px] border-[var(--vx-border-2)] px-0 sm:h-[26px] sm:w-auto sm:rounded-[7px] sm:px-2.5 sm:text-[11.5px]",
                        file.is_dir && "hidden sm:inline-flex"
                      )}
                      disabled={file.is_dir || !!uploadingName}
                      aria-label={t("common.download")}
                      title={
                        file.is_dir
                          ? t("servers.ftp.dir_download_unsupported")
                          : t("common.download")
                      }
                      onClick={() => void onDownload(fullPath)}
                    >
                      <i className="ri-download-2-line text-[15px] sm:hidden" />
                      <span className="hidden sm:inline">{t("common.download")}</span>
                    </Btn>
                    <Btn
                      size="sm"
                      className="size-8 rounded-[8px] border-[rgba(224,122,122,0.3)] bg-transparent px-0 text-[var(--vx-danger)] sm:h-[26px] sm:w-auto sm:rounded-[7px] sm:px-2.5 sm:text-[11.5px]"
                      disabled={deleteMutation.isPending}
                      aria-label={t("common.delete")}
                      title={t("common.delete")}
                      onClick={() => setDeleteTarget(fullPath)}
                    >
                      <i className="ri-delete-bin-line text-[15px] sm:hidden" />
                      <span className="hidden sm:inline">{t("common.delete")}</span>
                    </Btn>
                  </span>
                </div>
              );
            })
          )}
        </div>
      </Panel>

      <Dialog open={editorPath !== null} onOpenChange={(open) => !open && setEditorPath(null)}>
        <DialogContent className="max-h-[90vh] overflow-y-auto p-4 sm:max-w-2xl sm:p-6">
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
