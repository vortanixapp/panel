"use client";

import { useState } from "react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Btn, VX_INPUT } from "@/components/vx/panel-ui";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

/**
 * Подтверждение необратимого действия.
 *
 * Заменяет window.confirm. Нативное окно рисует браузер: оно появляется мгновенно
 * поверх всего, выглядит чужим и не даёт ни объяснить последствия, ни отличить
 * «удалить правило» от «стереть все данные сервера» — обе просьбы выглядят
 * одинаково, и вторую подтверждают не глядя.
 *
 * Для по-настоящему разрушительных действий предусмотрен ввод подтверждающего
 * слова: набрать имя сервера — это ровно та пауза, которой не хватает, чтобы
 * не переустановить чужой сервер по ошибке.
 */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  tone = "danger",
  requirePhrase,
  pending,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: React.ReactNode;
  confirmLabel: string;
  tone?: "danger" | "primary";
  /** Слово, которое нужно набрать, чтобы кнопка стала доступной. */
  requirePhrase?: string;
  pending?: boolean;
  onConfirm: () => void;
}) {
  const t = useT();
  const [typed, setTyped] = useState("");
  const ready = !requirePhrase || typed.trim() === requirePhrase.trim();

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) setTyped("");
        onOpenChange(next);
      }}
    >
      <DialogContent className="sm:max-w-[460px]">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription asChild>
            <div className="text-[13px] leading-[1.55] text-[var(--vx-muted)]">{description}</div>
          </DialogDescription>
        </DialogHeader>

        {requirePhrase && (
          <label className="grid gap-1.5 text-[11.5px] text-[var(--vx-muted)]">
            {t("servers.confirm.type_phrase", { phrase: requirePhrase })}
            <input
              className={VX_INPUT}
              value={typed}
              autoFocus
              spellCheck={false}
              onChange={(e) => setTyped(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && ready && !pending) onConfirm();
              }}
            />
          </label>
        )}

        <DialogFooter className="gap-2">
          <Btn tone="ghost" onClick={() => onOpenChange(false)} disabled={pending}>
            {t("common.cancel")}
          </Btn>
          <Btn
            tone={tone}
            disabled={!ready || pending}
            onClick={onConfirm}
            className={cn(pending && "cursor-progress")}
          >
            {pending ? t("common.processing") : confirmLabel}
          </Btn>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
