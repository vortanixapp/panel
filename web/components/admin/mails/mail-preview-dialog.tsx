"use client";

import { Loader2 } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useT } from "@/hooks/use-translations";

export type MailPreview = {
  subject: string;
  html: string;
  text: string;
};

type MailPreviewDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  preview: MailPreview | null;
  loading: boolean;
};

export function MailPreviewDialog({
  open,
  onOpenChange,
  preview,
  loading,
}: MailPreviewDialogProps) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[88vh] gap-4 overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("admin.mails.preview_title")}</DialogTitle>
          <DialogDescription>{preview?.subject ?? ""}</DialogDescription>
        </DialogHeader>

        {loading || !preview ? (
          <div className="flex h-48 items-center justify-center text-muted-foreground">
            <Loader2 className="size-5 animate-spin" />
          </div>
        ) : (
          <div className="space-y-4">
            <iframe
              title={preview.subject}
              sandbox=""
              srcDoc={preview.html}
              className="h-[420px] w-full rounded-xl border bg-white"
            />
            <div className="space-y-2">
              <div className="text-[13px] font-medium">
                {t("admin.mails.preview_text")}
              </div>
              <pre className="max-h-48 overflow-auto rounded-xl border bg-muted/40 p-3 text-[12px] whitespace-pre-wrap">
                {preview.text}
              </pre>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
