"use client";

import { useState } from "react";
import { toast } from "sonner";

import { useT } from "@/hooks/use-translations";
import { downloadSupportAttachment, type SupportAttachment } from "@/lib/api";
import type { TranslateFn } from "@/lib/i18n";

// t передаём параметром: единицы измерения нужны на языке, выбранном сейчас,
// а функция объявлена вне компонента и своей подписки на смену языка не имеет.
function fmtSize(bytes: number, t: TranslateFn): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "";
  if (bytes < 1024) return t("support.attachment.size_b", { n: bytes });
  if (bytes < 1024 * 1024)
    return t("support.attachment.size_kb", { n: Math.round(bytes / 1024) });
  return t("support.attachment.size_mb", {
    n: (bytes / (1024 * 1024)).toFixed(1),
  });
}

export function SupportAttachmentChip({
  ticketId,
  messageId,
  attachment,
}: {
  ticketId: string;
  messageId: string;
  attachment: SupportAttachment;
}) {
  const t = useT();
  const [busy, setBusy] = useState(false);
  const size = fmtSize(attachment.size, t);

  if (!attachment.url) {
    return (
      <div className="mt-2 flex items-center gap-2 rounded-lg border border-dashed border-border px-3 py-2 text-xs text-muted-foreground">
        <i className="ri-file-forbid-line" />
        <span className="truncate">{attachment.name}</span>
        <span className="whitespace-nowrap">
          {t("support.attachment.missing")}
        </span>
      </div>
    );
  }

  return (
    <button
      type="button"
      disabled={busy}
      onClick={async () => {
        setBusy(true);
        try {
          // У новых вложений своя запись, у старых идентификатор — сообщение.
          await downloadSupportAttachment(
            ticketId,
            attachment.id ?? messageId,
            attachment.name
          );
        } catch (err) {
          toast.error(
            err instanceof Error
              ? err.message
              : t("support.attachment.download_failed")
          );
        } finally {
          setBusy(false);
        }
      }}
      className="mt-2 flex w-full items-center gap-2 rounded-lg border border-border bg-background/60 px-3 py-2 text-left text-xs transition-colors hover:bg-background disabled:opacity-60"
    >
      <i
        className={
          busy
            ? "ri-loader-4-line animate-spin"
            : attachment.content_type?.startsWith("image/")
              ? "ri-image-line"
              : "ri-attachment-2"
        }
      />
      <span className="min-w-0 flex-1 truncate font-medium">
        {attachment.name}
      </span>
      {size && (
        <span className="whitespace-nowrap text-muted-foreground">{size}</span>
      )}
    </button>
  );
}
