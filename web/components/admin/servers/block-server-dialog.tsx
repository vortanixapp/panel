"use client";

import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useT } from "@/hooks/use-translations";

export function BlockServerDialog({
  open,
  serverName,
  count,
  pending,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  serverName?: string;
  count?: number;
  pending?: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (reason: string) => void;
}) {
  const t = useT();
  const [reason, setReason] = useState("");

  useEffect(() => {
    if (open) setReason("");
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("servers.admin.block.title")}</DialogTitle>
          <DialogDescription>
            {count && count > 1
              ? t("servers.admin.block.description_many", { count })
              : t("servers.admin.block.description", { name: serverName ?? "" })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-1.5">
          <Label htmlFor="block-reason">{t("servers.admin.block.reason")}</Label>
          <Textarea
            id="block-reason"
            rows={3}
            value={reason}
            placeholder={t("servers.admin.block.reason_placeholder")}
            onChange={(e) => setReason(e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            {t("servers.admin.block.reason_hint")}
          </p>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button
            variant="destructive"
            disabled={pending}
            onClick={() => onConfirm(reason.trim())}
          >
            {t("servers.admin.actions.block")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
