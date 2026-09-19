"use client";

import { ClipboardCopy, Info, PencilLine, TriangleAlert } from "lucide-react";
import { toast } from "sonner";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { btnGhost, btnPrimary } from "@/components/user/panel-parts";
import { useT } from "@/hooks/use-translations";
import type { BugReportPrepared } from "@/lib/api";
import { cn } from "@/lib/utils";
import { copyText } from "./bug-report-utils";

function Notice({ tone, children }: { tone: "warn" | "info"; children: React.ReactNode }) {
  const Icon = tone === "warn" ? TriangleAlert : Info;
  return (
    <div
      className={cn(
        "flex items-start gap-2.5 rounded-xl border px-3.5 py-3 text-[12.5px] leading-relaxed",
        tone === "warn"
          ? "border-amber-500/30 bg-amber-500/[0.07] text-foreground"
          : "border-border bg-muted/40 text-muted-foreground"
      )}
    >
      <Icon
        className={cn("mt-0.5 h-4 w-4 flex-none", tone === "warn" ? "text-amber-600 dark:text-amber-400" : "")}
      />
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}

export function PreviewDialog({
  report,
  attachLogs,
  onClose,
  onOpened,
}: {
  report: BugReportPrepared | null;
  attachLogs: boolean;
  onClose: () => void;
  onOpened: (report: BugReportPrepared) => void;
}) {
  const t = useT();

  const copy = async (text: string, success: string) => {
    if (await copyText(text)) toast.success(success);
    else toast.error(t("admin.bug.copy_failed"));
  };

  const open = () => {
    if (!report) return;
    if (report.clipboard) {
      void copyText(report.clipboard).then((ok) => {
        if (!ok) toast.error(t("admin.bug.copy_failed"));
      });
    }
    onOpened(report);
  };

  return (
    <Dialog open={report !== null} onOpenChange={(value) => !value && onClose()}>
      {report && (
        <DialogContent className="flex max-h-[min(90vh,860px)] flex-col gap-0 p-0 sm:max-w-2xl">
          <DialogHeader className="border-b px-5 pt-5 pb-4 text-start sm:px-6">
            <DialogTitle className="pe-6">{t("admin.bug.preview_title")}</DialogTitle>
            <DialogDescription className="text-[12.5px] leading-relaxed">
              {t("admin.bug.preview_hint", { repo: report.repo })}
            </DialogDescription>
          </DialogHeader>

          <div className="min-h-0 flex-1 overflow-y-auto px-5 py-5 sm:px-6">
            <div className="rounded-xl border bg-background">
              <div className="border-b px-4 py-3">
                <h3 className="text-[15px] leading-snug font-semibold break-words">{report.title}</h3>
              </div>
              <dl className="flex flex-col divide-y">
                {report.fields.map((field) => (
                  <div key={field.id} className="flex flex-col gap-1.5 px-4 py-3">
                    <dt className="text-[11.5px] font-medium tracking-wide text-muted-foreground uppercase">
                      {t(`admin.bug.field.${field.id}`)}
                    </dt>
                    <dd className="min-w-0">
                      {field.id === "logs" ? (
                        <pre className="max-h-56 overflow-auto rounded-lg bg-muted/50 px-3 py-2.5 font-mono text-[11.5px] leading-relaxed">
                          {field.value}
                        </pre>
                      ) : (
                        <p className="text-[13px] leading-relaxed break-words whitespace-pre-wrap">{field.value}</p>
                      )}
                    </dd>
                  </div>
                ))}
              </dl>
            </div>

            <div className="mt-4 flex flex-col gap-2.5">
              {report.clipboard && (
                <Notice tone="warn">
                  <p>{t("admin.bug.clipboard_note")}</p>
                  <button
                    type="button"
                    onClick={() => void copy(report.clipboard, t("admin.bug.desc_copied"))}
                    className={cn(btnGhost, "mt-2 h-7 bg-background px-2.5 text-[12px]")}
                  >
                    <ClipboardCopy className="h-3.5 w-3.5" />
                    {t("admin.bug.copy_desc")}
                  </button>
                </Notice>
              )}
              {attachLogs && report.logs_total === 0 && <Notice tone="info">{t("admin.bug.logs_empty")}</Notice>}
              {report.logs_total > 0 && report.logs_lines === 0 && (
                <Notice tone="info">{t("admin.bug.logs_dropped")}</Notice>
              )}
              {report.logs_lines > 0 && report.logs_lines < report.logs_total && (
                <Notice tone="info">
                  {t("admin.bug.logs_trimmed", { lines: report.logs_lines, total: report.logs_total })}
                </Notice>
              )}
            </div>

            <ol className="mt-5 flex flex-col gap-2.5">
              {["admin.bug.step_open", "admin.bug.step_shots", "admin.bug.step_create"].map((key, index) => (
                <li key={key} className="flex items-start gap-3 text-[12.5px] leading-relaxed">
                  <span className="grid h-5 w-5 flex-none place-items-center rounded-full bg-muted font-mono text-[11px] font-semibold">
                    {index + 1}
                  </span>
                  <span className="pt-px">{t(key)}</span>
                </li>
              ))}
            </ol>
          </div>

          <DialogFooter className="gap-2 border-t px-5 py-4 sm:px-6">
            <button type="button" onClick={onClose} className={cn(btnGhost, "h-9 sm:me-auto")}>
              <PencilLine className="h-4 w-4" />
              {t("admin.bug.back_edit")}
            </button>
            <button
              type="button"
              onClick={() => void copy(report.text, t("admin.bug.copied"))}
              className={cn(btnGhost, "h-9")}
            >
              <ClipboardCopy className="h-4 w-4" />
              {t("admin.bug.copy_text")}
            </button>
            <a
              href={report.url}
              target="_blank"
              rel="noopener noreferrer"
              onClick={open}
              className={cn(btnPrimary, "h-9")}
            >
              <i className="ri-github-fill text-[16px] leading-none" />
              {t("admin.bug.open_github")}
            </a>
          </DialogFooter>
        </DialogContent>
      )}
    </Dialog>
  );
}
