"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { useT } from "@/hooks/use-translations";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  runNodeBulkAction,
  setNodeMaintenance,
  type AdminLocationListItem,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export function MaintenanceDialog({
  location,
  open,
  onOpenChange,
}: {
  location: AdminLocationListItem;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const qc = useQueryClient();
  const enabled = Boolean(location.maintenance_mode);
  const [reason, setReason] = useState(location.maintenance_reason ?? "");
  const [until, setUntil] = useState(
    location.maintenance_until ? location.maintenance_until.slice(0, 16) : ""
  );
  const [notify, setNotify] = useState(true);
  const [bulkMessage, setBulkMessage] = useState("");
  const [extendDays, setExtendDays] = useState("1");

  const mut = useMutation({
    mutationFn: (turnOn: boolean) =>
      setNodeMaintenance(location.id, {
        enabled: turnOn,
        reason,
        until: until ? new Date(until).toISOString() : "",
        notify_owners: turnOn && notify,
      }),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: queryKeys.adminLocations });
      toast.success(
        res.maintenance_mode
          ? res.notified > 0
            ? t("admin.maintenance.enabled_notified", { count: res.notified })
            : t("admin.maintenance.enabled")
          : t("admin.maintenance.finished")
      );
      onOpenChange(false);
    },
    onError: (e: Error) => toast.error(e.message || t("common.save_failed")),
  });

  const bulkMut = useMutation({
    mutationFn: (action: "start" | "stop" | "restart" | "notify" | "extend") =>
      runNodeBulkAction(location.id, {
        action,
        message: bulkMessage,
        days: Number(extendDays) || 1,
        only_running: action === "restart",
      }),
    onSuccess: (res) => {
      toast.success(t("admin.maintenance.bulk_queued", { count: res.servers }));
      setBulkMessage("");
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.maintenance.bulk_failed")),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {t("admin.maintenance.title", { name: location.name })}
          </DialogTitle>
          <DialogDescription>
            {t("admin.maintenance.description", {
              count: location.servers_count,
            })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="space-y-1">
            <Label>{t("admin.maintenance.reason")}</Label>
            <Input
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder={t("admin.maintenance.reason_placeholder")}
            />
          </div>
          <div className="space-y-1">
            <Label>{t("admin.maintenance.until")}</Label>
            <Input
              type="datetime-local"
              value={until}
              onChange={(e) => setUntil(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.maintenance.until_hint")}
            </p>
          </div>
          {!enabled ? (
            <div className="flex items-start gap-2">
              <Checkbox
                id="notify-owners"
                checked={notify}
                onCheckedChange={(v) => setNotify(v === true)}
              />
              <Label htmlFor="notify-owners">
                {t("admin.maintenance.notify_owners")}
              </Label>
            </div>
          ) : null}
        </div>

        <div className="space-y-3 border-t pt-4">
          <div className="text-sm font-medium">
            {t("admin.maintenance.bulk_title")}
          </div>
          <p className="text-xs text-muted-foreground">
            {t("admin.maintenance.bulk_hint")}
          </p>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => bulkMut.mutate("start")}
              disabled={bulkMut.isPending}
            >
              {t("admin.maintenance.start_all")}
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => bulkMut.mutate("restart")}
              disabled={bulkMut.isPending}
            >
              {t("admin.maintenance.restart_running")}
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                if (!confirm(t("admin.maintenance.stop_all_confirm"))) return;
                bulkMut.mutate("stop");
              }}
              disabled={bulkMut.isPending}
            >
              {t("admin.maintenance.stop_all")}
            </Button>
          </div>
          <div className="flex flex-wrap items-end gap-2">
            <div className="flex-1 space-y-1">
              <Label className="text-xs">
                {t("admin.maintenance.message_label")}
              </Label>
              <Input
                value={bulkMessage}
                onChange={(e) => setBulkMessage(e.target.value)}
                placeholder={t("admin.maintenance.message_placeholder")}
              />
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => bulkMut.mutate("notify")}
              disabled={bulkMut.isPending || !bulkMessage.trim()}
            >
              {t("admin.maintenance.notify")}
            </Button>
          </div>
          <div className="flex flex-wrap items-end gap-2">
            <div className="space-y-1">
              <Label className="text-xs">
                {t("admin.maintenance.extend_days")}
              </Label>
              <Input
                type="number"
                min="1"
                max="365"
                value={extendDays}
                onChange={(e) => setExtendDays(e.target.value)}
                className="w-28"
              />
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                if (
                  !confirm(
                    t("admin.maintenance.extend_confirm", { days: extendDays })
                  )
                )
                  return;
                bulkMut.mutate("extend");
              }}
              disabled={bulkMut.isPending}
            >
              {t("admin.maintenance.extend")}
            </Button>
          </div>
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          {enabled ? (
            <>
              <Button
                variant="outline"
                onClick={() => mut.mutate(true)}
                disabled={mut.isPending}
              >
                {t("admin.maintenance.update_reason")}
              </Button>
              <Button onClick={() => mut.mutate(false)} disabled={mut.isPending}>
                {t("admin.maintenance.finish")}
              </Button>
            </>
          ) : (
            <Button onClick={() => mut.mutate(true)} disabled={mut.isPending}>
              {t("admin.maintenance.enable")}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
