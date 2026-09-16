"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronDown, Plus } from "lucide-react";
import { toast } from "sonner";

import {
  EmptyBlock,
  OneTimeSecret,
  TabHeader,
  fmtDateTime,
} from "@/components/admin/integrations/integrations-ui";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Badge } from "@/components/ui/badge";
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
import { Skeleton } from "@/components/ui/skeleton";
import {
  createAdminWebhook,
  deleteAdminWebhook,
  fetchAdminWebhooks,
  fetchWebhookDeliveries,
  testAdminWebhook,
  updateAdminWebhook,
  type AdminWebhook,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

type HookAction = { id: string; kind: "toggle" | "delete" | "test"; active?: boolean };

export function WebhooksTab() {
  const t = useT();
  const qc = useQueryClient();

  const hooksQuery = useQuery({
    queryKey: ["admin-webhooks"],
    queryFn: fetchAdminWebhooks,
  });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [hookURL, setHookURL] = useState("");
  const [hookEvents, setHookEvents] = useState<string[]>([]);
  const [hookSecret, setHookSecret] = useState("");
  const [deliveriesFor, setDeliveriesFor] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AdminWebhook | null>(null);

  const deliveriesQuery = useQuery({
    queryKey: ["webhook-deliveries", deliveriesFor],
    queryFn: () => fetchWebhookDeliveries(deliveriesFor as string),
    enabled: !!deliveriesFor,
    refetchInterval: 10_000,
  });

  const createHookMut = useMutation({
    mutationFn: () => createAdminWebhook({ url: hookURL.trim(), events: hookEvents }),
    onSuccess: (res) => {
      setHookSecret(res.secret);
      setHookURL("");
      setHookEvents([]);
      setDialogOpen(false);
      void qc.invalidateQueries({ queryKey: ["admin-webhooks"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.integrations.hook_create_failed")),
  });

  const hookActionMut = useMutation({
    mutationFn: async (action: HookAction) => {
      if (action.kind === "delete") return deleteAdminWebhook(action.id);
      if (action.kind === "test") return testAdminWebhook(action.id);
      return updateAdminWebhook(action.id, { active: !action.active });
    },
    onSuccess: (_res, action) => {
      toast.success(
        action.kind === "delete"
          ? t("admin.integrations.hook_deleted")
          : action.kind === "test"
            ? t("admin.integrations.hook_test_queued")
            : t("common.saved")
      );
      if (action.kind === "delete") {
        setDeleteTarget(null);
        if (deliveriesFor === action.id) setDeliveriesFor(null);
      }
      void qc.invalidateQueries({ queryKey: ["admin-webhooks"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.integrations.action_failed")),
  });

  const hooks = hooksQuery.data?.webhooks ?? [];
  const events = hooksQuery.data?.events ?? [];
  const eventLabels = new Map(events.map((event) => [event.key, event.label]));
  const canCreate = hookURL.trim() !== "" && hookEvents.length > 0;

  const toggleEvent = (key: string) =>
    setHookEvents((prev) =>
      prev.includes(key) ? prev.filter((v) => v !== key) : [...prev, key]
    );

  return (
    <div className="flex flex-col gap-4">
      <TabHeader
        title={t("admin.integrations.hooks_title")}
        description={t("admin.integrations.hooks_hint")}
        action={
          <Button onClick={() => setDialogOpen(true)}>
            <Plus className="mr-1.5 size-4" />
            {t("admin.integrations.add_hook")}
          </Button>
        }
      />

      {hookSecret ? (
        <OneTimeSecret
          title={t("admin.integrations.hook_secret")}
          value={hookSecret}
          onDismiss={() => setHookSecret("")}
        />
      ) : null}

      <div className="overflow-hidden rounded-xl border bg-card">
        {hooksQuery.isLoading ? (
          <div className="p-4">
            <Skeleton className="h-24 w-full" />
          </div>
        ) : hooks.length === 0 ? (
          <EmptyBlock>{t("admin.integrations.hooks_empty")}</EmptyBlock>
        ) : (
          <div className="divide-y">
            {hooks.map((hook) => {
              const open = deliveriesFor === hook.id;
              return (
                <div key={hook.id} className="px-4 py-3.5">
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <code className="font-mono text-sm break-all">{hook.url}</code>
                        {hook.active ? (
                          <Badge variant="secondary">{t("admin.integrations.hook_on")}</Badge>
                        ) : (
                          <Badge variant="outline">{t("admin.integrations.hook_off")}</Badge>
                        )}
                        {hook.failed_count > 0 ? (
                          <Badge variant="destructive">
                            {t("admin.integrations.hook_errors", { count: hook.failed_count })}
                          </Badge>
                        ) : null}
                      </div>
                      <div className="mt-1.5 flex flex-wrap gap-1">
                        {hook.events.map((event) => (
                          <span
                            key={event}
                            className="rounded border px-1.5 py-0.5 text-[11px] text-muted-foreground"
                          >
                            {eventLabels.get(event) ?? event}
                          </span>
                        ))}
                      </div>
                      <div className="mt-1.5 text-xs text-muted-foreground">
                        {t("admin.integrations.hook_last_sent", {
                          sent: fmtDateTime(hook.last_sent_at),
                        })}
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={hookActionMut.isPending}
                        onClick={() => hookActionMut.mutate({ id: hook.id, kind: "test" })}
                      >
                        {t("admin.integrations.hook_test")}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setDeliveriesFor(open ? null : hook.id)}
                      >
                        {t("admin.integrations.hook_history")}
                        <ChevronDown
                          className={cn("ml-1 size-3.5 transition-transform", open && "rotate-180")}
                        />
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={hookActionMut.isPending}
                        onClick={() =>
                          hookActionMut.mutate({ id: hook.id, kind: "toggle", active: hook.active })
                        }
                      >
                        {hook.active ? t("common.disable") : t("common.enable")}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive"
                        onClick={() => setDeleteTarget(hook)}
                      >
                        {t("common.delete")}
                      </Button>
                    </div>
                  </div>

                  {open ? (
                    <div className="mt-3 overflow-hidden rounded-lg border">
                      {deliveriesQuery.isLoading ? (
                        <div className="p-3">
                          <Skeleton className="h-12 w-full" />
                        </div>
                      ) : (deliveriesQuery.data?.deliveries.length ?? 0) === 0 ? (
                        <div className="p-4 text-center text-xs text-muted-foreground">
                          {t("admin.integrations.no_deliveries")}
                        </div>
                      ) : (
                        <div className="overflow-x-auto">
                          <table className="w-full min-w-[520px] text-xs">
                            <thead>
                              <tr className="border-b bg-muted/30 text-left text-muted-foreground">
                                <th className="px-3 py-2 font-medium">
                                  {t("admin.integrations.col_event")}
                                </th>
                                <th className="px-3 py-2 font-medium">{t("common.status")}</th>
                                <th className="px-3 py-2 font-medium">
                                  {t("admin.integrations.col_attempts")}
                                </th>
                                <th className="px-3 py-2 font-medium">
                                  {t("admin.integrations.col_when")}
                                </th>
                              </tr>
                            </thead>
                            <tbody>
                              {(deliveriesQuery.data?.deliveries ?? []).map((d, i) => (
                                <tr key={`${d.event}-${i}`} className="border-b last:border-0">
                                  <td className="px-3 py-2 font-mono">{d.event}</td>
                                  <td
                                    className={cn(
                                      "px-3 py-2",
                                      d.status === "failed" && "text-destructive"
                                    )}
                                  >
                                    {d.status === "delivered"
                                      ? t("admin.integrations.delivered", {
                                          code: d.response_code ?? "—",
                                        })
                                      : d.status === "failed"
                                        ? t("admin.integrations.delivery_failed", {
                                            error: d.error ?? "—",
                                          })
                                        : t("admin.integrations.delivery_queued")}
                                  </td>
                                  <td className="px-3 py-2 tabular-nums">{d.attempts}</td>
                                  <td className="px-3 py-2 whitespace-nowrap">
                                    {fmtDateTime(d.created_at)}
                                  </td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      )}
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        )}
      </div>

      <div className="rounded-xl border bg-muted/20 px-4 py-3 text-xs leading-relaxed text-muted-foreground">
        <span className="font-medium text-foreground">
          {t("admin.integrations.sign_title")}.
        </span>{" "}
        {t("admin.integrations.hook_sign_before")}{" "}
        <code className="font-mono">X-Vortanix-Signature: sha256=HMAC(secret, body)</code>.{" "}
        {t("admin.integrations.hook_sign_after")}
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>{t("admin.integrations.add_hook")}</DialogTitle>
            <DialogDescription>{t("admin.integrations.hooks_hint")}</DialogDescription>
          </DialogHeader>
          <form
            className="flex flex-col gap-4"
            onSubmit={(e) => {
              e.preventDefault();
              if (canCreate) createHookMut.mutate();
            }}
          >
            <div className="space-y-1.5">
              <Label htmlFor="webhook-url">{t("admin.integrations.hook_url")}</Label>
              <Input
                id="webhook-url"
                type="url"
                value={hookURL}
                onChange={(e) => setHookURL(e.target.value)}
                placeholder="https://example.com/vortanix-hook"
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <div className="flex items-center justify-between gap-3">
                <Label>{t("admin.integrations.events")}</Label>
                <span className="text-xs text-muted-foreground">
                  {hookEvents.length} / {events.length}
                </span>
              </div>
              <div className="grid max-h-[320px] gap-1.5 overflow-y-auto rounded-lg border p-2 sm:grid-cols-2">
                {events.map((event) => (
                  <label
                    key={event.key}
                    className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-xs hover:bg-muted"
                  >
                    <Checkbox
                      checked={hookEvents.includes(event.key)}
                      onCheckedChange={() => toggleEvent(event.key)}
                    />
                    <span>{event.label}</span>
                  </label>
                ))}
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                {t("common.cancel")}
              </Button>
              <Button type="submit" disabled={!canCreate || createHookMut.isPending}>
                {t("common.add")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
        title={t("admin.integrations.hook_delete_confirm")}
        desc={deleteTarget?.url ?? ""}
        cancelBtnText={t("common.cancel")}
        confirmText={t("common.delete")}
        destructive
        isLoading={hookActionMut.isPending}
        handleConfirm={() => {
          if (deleteTarget) hookActionMut.mutate({ id: deleteTarget.id, kind: "delete" });
        }}
      />
    </div>
  );
}
