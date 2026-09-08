"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  createAdminAPIKey,
  createAdminWebhook,
  deleteAdminWebhook,
  fetchAdminAPIKeys,
  fetchAdminWebhooks,
  fetchWebhookDeliveries,
  revokeAdminAPIKey,
  testAdminWebhook,
  updateAdminWebhook,
  type AdminAPIKey,
  type AdminWebhook,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

function fmtDateTime(iso: string | null): string {
  if (!iso) return "—";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString(localeTag());
}

export function IntegrationsPageContent() {
  const t = useT();
  const qc = useQueryClient();

  const keysQuery = useQuery({
    queryKey: ["admin-api-keys"],
    queryFn: fetchAdminAPIKeys,
  });
  const hooksQuery = useQuery({
    queryKey: ["admin-webhooks"],
    queryFn: fetchAdminWebhooks,
  });

  const [keyName, setKeyName] = useState("");
  const [keyScopes, setKeyScopes] = useState<string[]>([]);
  const [issuedKey, setIssuedKey] = useState("");

  const [hookURL, setHookURL] = useState("");
  const [hookEvents, setHookEvents] = useState<string[]>([]);
  const [hookSecret, setHookSecret] = useState("");
  const [deliveriesFor, setDeliveriesFor] = useState<string | null>(null);

  const deliveriesQuery = useQuery({
    queryKey: ["webhook-deliveries", deliveriesFor],
    queryFn: () => fetchWebhookDeliveries(deliveriesFor as string),
    enabled: !!deliveriesFor,
    refetchInterval: 10_000,
  });

  const createKeyMut = useMutation({
    mutationFn: () => createAdminAPIKey({ name: keyName, scopes: keyScopes }),
    onSuccess: (res) => {
      setIssuedKey(res.key);
      setKeyName("");
      setKeyScopes([]);
      void qc.invalidateQueries({ queryKey: ["admin-api-keys"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.integrations.key_create_failed")),
  });

  const revokeKeyMut = useMutation({
    mutationFn: (id: string) => revokeAdminAPIKey(id),
    onSuccess: () => {
      toast.success(t("admin.integrations.key_revoked"));
      void qc.invalidateQueries({ queryKey: ["admin-api-keys"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.integrations.revoke_failed")),
  });

  const createHookMut = useMutation({
    mutationFn: () => createAdminWebhook({ url: hookURL, events: hookEvents }),
    onSuccess: (res) => {
      setHookSecret(res.secret);
      setHookURL("");
      setHookEvents([]);
      void qc.invalidateQueries({ queryKey: ["admin-webhooks"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.integrations.hook_create_failed")),
  });

  const hookActionMut = useMutation({
    mutationFn: async (action: { id: string; kind: "toggle" | "delete" | "test"; active?: boolean }) => {
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
      void qc.invalidateQueries({ queryKey: ["admin-webhooks"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.integrations.action_failed")),
  });

  const toggle = (list: string[], value: string) =>
    list.includes(value) ? list.filter((v) => v !== value) : [...list, value];

  return (
    <PageShell variant="admin">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">
          {t("admin.integrations.title")}
        </h1>
        <p className="text-sm text-muted-foreground">
          {t("admin.integrations.subtitle")}
        </p>
      </div>

      <div className="mb-8">
        <h2 className="mb-3 text-lg font-semibold">
          {t("admin.integrations.keys_title")}
        </h2>

        {issuedKey ? (
          <div className="mb-4 rounded-lg border border-amber-500/40 bg-amber-500/5 p-4">
            <div className="text-sm font-medium">
              {t("admin.integrations.key_once")}
            </div>
            <code className="mt-2 block break-all rounded bg-muted p-2 font-mono text-xs">
              {issuedKey}
            </code>
            <p className="mt-2 text-xs text-muted-foreground">
              {t("admin.integrations.key_hash_hint")}{" "}
              <code className="font-mono">X-API-Key</code>
            </p>
            <Button
              variant="outline"
              size="sm"
              className="mt-3"
              onClick={() => setIssuedKey("")}
            >
              {t("admin.integrations.copied")}
            </Button>
          </div>
        ) : null}

        <div className="mb-4 space-y-3 rounded-lg border bg-card p-4">
          <div className="flex flex-wrap items-end gap-3">
            <div className="flex-1 space-y-1">
              <Label>{t("common.name")}</Label>
              <Input
                value={keyName}
                onChange={(e) => setKeyName(e.target.value)}
                placeholder={t("admin.integrations.key_name_placeholder")}
              />
            </div>
            <Button
              onClick={() => createKeyMut.mutate()}
              disabled={createKeyMut.isPending || !keyName.trim() || keyScopes.length === 0}
            >
              {t("admin.integrations.issue_key")}
            </Button>
          </div>
          <div>
            <Label className="text-xs">{t("admin.integrations.key_scopes")}</Label>
            <div className="mt-2 flex flex-wrap gap-2">
              {(keysQuery.data?.all_scopes ?? []).map((scope) => (
                <label
                  key={scope}
                  className="flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs"
                >
                  <Checkbox
                    checked={keyScopes.includes(scope)}
                    onCheckedChange={() => setKeyScopes((prev) => toggle(prev, scope))}
                  />
                  <span className="font-mono">{scope}</span>
                </label>
              ))}
            </div>
            <p className="mt-2 text-xs text-muted-foreground">
              {t("admin.integrations.key_scope_hint_before")}{" "}
              <code className="font-mono">/v1/admin/*</code>{" "}
              {t("admin.integrations.key_scope_hint_after")}
            </p>
          </div>
        </div>

        <div className="rounded-lg border bg-card">
          {keysQuery.isLoading ? (
            <div className="p-4">
              <Skeleton className="h-16 w-full" />
            </div>
          ) : (keysQuery.data?.keys.length ?? 0) === 0 ? (
            <div className="p-8 text-center text-sm text-muted-foreground">
              {t("admin.integrations.keys_empty")}
            </div>
          ) : (
            <div className="divide-y">
              {(keysQuery.data?.keys ?? []).map((key: AdminAPIKey) => (
                <div
                  key={key.id}
                  className="flex flex-wrap items-center justify-between gap-3 p-4"
                >
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{key.name}</span>
                      <code className="font-mono text-xs text-muted-foreground">
                        {key.prefix}…
                      </code>
                      {key.active ? (
                        <Badge variant="secondary">
                          {t("admin.integrations.key_active")}
                        </Badge>
                      ) : (
                        <Badge variant="outline">
                          {t("admin.integrations.key_revoked_badge")}
                        </Badge>
                      )}
                    </div>
                    <div className="mt-1 text-xs text-muted-foreground">
                      {t("admin.integrations.key_meta", {
                        scopes: key.scopes.length,
                        created: fmtDateTime(key.created_at),
                        used: fmtDateTime(key.last_used_at),
                      })}
                    </div>
                  </div>
                  {key.active ? (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        if (
                          !confirm(
                            t("admin.integrations.revoke_confirm", {
                              name: key.name,
                            })
                          )
                        )
                          return;
                        revokeKeyMut.mutate(key.id);
                      }}
                    >
                      {t("admin.integrations.revoke")}
                    </Button>
                  ) : null}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      <div>
        <h2 className="mb-3 text-lg font-semibold">
          {t("admin.integrations.hooks_title")}
        </h2>

        {hookSecret ? (
          <div className="mb-4 rounded-lg border border-amber-500/40 bg-amber-500/5 p-4">
            <div className="text-sm font-medium">
              {t("admin.integrations.hook_secret")}
            </div>
            <code className="mt-2 block break-all rounded bg-muted p-2 font-mono text-xs">
              {hookSecret}
            </code>
            <p className="mt-2 text-xs text-muted-foreground">
              {t("admin.integrations.hook_sign_before")}{" "}
              <code className="font-mono">
                X-Vortanix-Signature: sha256=HMAC(secret, body)
              </code>
              . {t("admin.integrations.hook_sign_after")}
            </p>
            <Button
              variant="outline"
              size="sm"
              className="mt-3"
              onClick={() => setHookSecret("")}
            >
              {t("admin.integrations.copied")}
            </Button>
          </div>
        ) : null}

        <div className="mb-4 space-y-3 rounded-lg border bg-card p-4">
          <div className="flex flex-wrap items-end gap-3">
            <div className="flex-1 space-y-1">
              <Label>{t("admin.integrations.hook_url")}</Label>
              <Input
                value={hookURL}
                onChange={(e) => setHookURL(e.target.value)}
                placeholder="https://example.com/vortanix-hook"
              />
            </div>
            <Button
              onClick={() => createHookMut.mutate()}
              disabled={createHookMut.isPending || !hookURL.trim() || hookEvents.length === 0}
            >
              {t("common.add")}
            </Button>
          </div>
          <div>
            <Label className="text-xs">{t("admin.integrations.events")}</Label>
            <div className="mt-2 flex flex-wrap gap-2">
              {(hooksQuery.data?.events ?? []).map((event) => (
                <label
                  key={event.key}
                  className="flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs"
                >
                  <Checkbox
                    checked={hookEvents.includes(event.key)}
                    onCheckedChange={() => setHookEvents((prev) => toggle(prev, event.key))}
                  />
                  <span>{event.label}</span>
                </label>
              ))}
            </div>
          </div>
        </div>

        <div className="rounded-lg border bg-card">
          {hooksQuery.isLoading ? (
            <div className="p-4">
              <Skeleton className="h-16 w-full" />
            </div>
          ) : (hooksQuery.data?.webhooks.length ?? 0) === 0 ? (
            <div className="p-8 text-center text-sm text-muted-foreground">
              {t("admin.integrations.hooks_empty")}
            </div>
          ) : (
            <div className="divide-y">
              {(hooksQuery.data?.webhooks ?? []).map((hook: AdminWebhook) => (
                <div key={hook.id} className="p-4">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <code className="font-mono text-sm break-all">{hook.url}</code>
                        {hook.active ? (
                          <Badge variant="secondary">
                            {t("admin.integrations.hook_on")}
                          </Badge>
                        ) : (
                          <Badge variant="outline">
                            {t("admin.integrations.hook_off")}
                          </Badge>
                        )}
                        {hook.failed_count > 0 ? (
                          <Badge variant="destructive">
                            {t("admin.integrations.hook_errors", {
                              count: hook.failed_count,
                            })}
                          </Badge>
                        ) : null}
                      </div>
                      <div className="mt-1 text-xs text-muted-foreground">
                        {t("admin.integrations.hook_meta", {
                          events: hook.events.length,
                          sent: fmtDateTime(hook.last_sent_at),
                        })}
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() =>
                          hookActionMut.mutate({ id: hook.id, kind: "test" })
                        }
                      >
                        {t("admin.integrations.hook_test")}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() =>
                          setDeliveriesFor(deliveriesFor === hook.id ? null : hook.id)
                        }
                      >
                        {t("admin.integrations.hook_history")}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() =>
                          hookActionMut.mutate({
                            id: hook.id,
                            kind: "toggle",
                            active: hook.active,
                          })
                        }
                      >
                        {hook.active ? t("common.disable") : t("common.enable")}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          if (!confirm(t("admin.integrations.hook_delete_confirm")))
                            return;
                          hookActionMut.mutate({ id: hook.id, kind: "delete" });
                        }}
                      >
                        {t("common.delete")}
                      </Button>
                    </div>
                  </div>

                  {deliveriesFor === hook.id ? (
                    <div className="mt-3 rounded-lg border">
                      {(deliveriesQuery.data?.deliveries.length ?? 0) === 0 ? (
                        <div className="p-4 text-center text-xs text-muted-foreground">
                          {t("admin.integrations.no_deliveries")}
                        </div>
                      ) : (
                        <table className="w-full text-xs">
                          <thead>
                            <tr className="text-muted-foreground">
                              <th className="px-3 py-2 text-left font-normal">
                                {t("admin.integrations.col_event")}
                              </th>
                              <th className="px-3 py-2 text-left font-normal">
                                {t("common.status")}
                              </th>
                              <th className="px-3 py-2 text-left font-normal">
                                {t("admin.integrations.col_attempts")}
                              </th>
                              <th className="px-3 py-2 text-left font-normal">
                                {t("admin.integrations.col_when")}
                              </th>
                            </tr>
                          </thead>
                          <tbody>
                            {(deliveriesQuery.data?.deliveries ?? []).map((d, i) => (
                              <tr key={`${d.event}-${i}`} className="border-t">
                                <td className="px-3 py-2 font-mono">{d.event}</td>
                                <td className="px-3 py-2">
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
                                <td className="px-3 py-2">{d.attempts}</td>
                                <td className="px-3 py-2">{fmtDateTime(d.created_at)}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      )}
                    </div>
                  ) : null}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </PageShell>
  );
}
