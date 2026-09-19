"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Copy, Loader2, Plus, TriangleAlert } from "lucide-react";
import { toast } from "sonner";
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
import { useT } from "@/hooks/use-translations";
import { API_URL, createAPIToken, fetchAPITokens, revokeAPIToken, type PersonalToken } from "@/lib/api";
import { cn } from "@/lib/utils";
import { errorText, Field, formatDate, formatDateTime, QueryState, SettingsSection, useCopy } from "./ui";

const TOKENS_KEY = ["account-api-tokens"];
const EXPIRY_DAYS = [30, 90, 365, 0];
const DEFAULT_SCOPES = ["servers.read"];

const SCOPE_ROUTES: Record<string, string[]> = {
  "servers.read": [
    "GET /v1/servers",
    "GET /v1/servers/{id}",
    "GET /v1/servers/{id}/status",
    "GET /v1/servers/{id}/metrics",
    "GET /v1/servers/{id}/logs",
  ],
  "servers.power": ["POST /v1/servers/{id}/power"],
  "servers.console": ["POST /v1/servers/{id}/console-command"],
  "servers.files.read": [
    "POST /v1/servers/{id}/files/list",
    "POST /v1/servers/{id}/files/read",
    "GET /v1/servers/{id}/files/download",
  ],
  "servers.files.write": [
    "POST /v1/servers/{id}/files/write",
    "POST /v1/servers/{id}/files/upload",
    "POST /v1/servers/{id}/files/mkdir",
    "POST /v1/servers/{id}/files/delete",
  ],
  "servers.backups": [
    "GET /v1/servers/{id}/backups",
    "POST /v1/servers/{id}/backups",
    "POST /v1/servers/{id}/backups/restore",
    "DELETE /v1/servers/{id}/backups",
  ],
  "billing.read": ["GET /v1/billing"],
};

function useScopeText() {
  const t = useT();
  return (scope: string, kind: "scope" | "scope_hint") => {
    const key = `settings.tokens.${kind}.${scope}`;
    const text = t(key);
    return text === key ? (kind === "scope" ? scope : "") : text;
  };
}

function apiBase() {
  const base = API_URL || (typeof window !== "undefined" ? window.location.origin : "");
  return base.replace(/\/+$/, "");
}

export function TokensTab() {
  const t = useT();
  const qc = useQueryClient();
  const scopeText = useScopeText();
  const query = useQuery({ queryKey: TOKENS_KEY, queryFn: fetchAPITokens });
  const [creating, setCreating] = useState(false);
  const [revoking, setRevoking] = useState<PersonalToken | null>(null);

  const revoke = useMutation({
    mutationFn: (id: string) => revokeAPIToken(id),
    onSuccess: () => {
      setRevoking(null);
      toast.success(t("settings.tokens.revoked"));
      void qc.invalidateQueries({ queryKey: TOKENS_KEY });
    },
    onError: (err) => toast.error(errorText(err, t("common.error"))),
  });

  const tokens = query.data?.tokens ?? [];
  const scopes = query.data?.scopes ?? [];
  const limit = query.data?.limit ?? 10;
  const enabled = query.data?.enabled ?? true;
  const active = tokens.filter((token) => !token.expired).length;
  const full = active >= limit;

  return (
    <>
      <SettingsSection
        title={t("settings.tokens.title")}
        description={t("settings.tokens.hint")}
        action={
          <Button type="button" size="sm" disabled={!query.data || !enabled || full} onClick={() => setCreating(true)}>
            <Plus className="size-4" />
            {t("settings.tokens.create")}
          </Button>
        }
      >
        {query.data && !enabled && (
          <div className="mb-4 flex items-start gap-2.5 rounded-xl border border-amber-500/30 bg-amber-500/5 px-4 py-3 text-[13px] text-amber-700 dark:text-amber-400">
            <TriangleAlert className="mt-0.5 size-4 shrink-0" />
            <span>{t("settings.tokens.disabled")}</span>
          </div>
        )}
        <QueryState isLoading={query.isLoading} isError={query.isError} error={query.error} onRetry={() => void query.refetch()}>
          {tokens.length === 0 ? (
            <div className="flex flex-col items-center gap-2 rounded-xl border border-dashed border-border px-6 py-8 text-center">
              <i className="ri-code-s-slash-line text-2xl text-muted-foreground" />
              <div className="text-[14px] font-medium">{t("settings.tokens.empty_title")}</div>
              <p className="max-w-md text-[13px] text-muted-foreground">{t("settings.tokens.empty_hint")}</p>
            </div>
          ) : (
            <ul className="divide-y divide-border rounded-xl border border-border">
              {tokens.map((token) => (
                <li key={token.id} className="flex flex-wrap items-start gap-3 px-4 py-3.5">
                  <i
                    className={cn(
                      "ri-key-2-line mt-0.5 text-lg",
                      token.expired ? "text-muted-foreground/60" : "text-muted-foreground"
                    )}
                  />
                  <div className="min-w-0 flex-1 space-y-1.5">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className={cn("truncate text-[13.5px] font-medium", token.expired && "text-muted-foreground")}>
                        {token.name}
                      </span>
                      <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-[11.5px] text-muted-foreground">
                        {token.prefix}…
                      </code>
                      {token.expired && <Badge variant="outline">{t("settings.tokens.expired")}</Badge>}
                    </div>
                    <div className="flex flex-wrap gap-1">
                      {token.scopes.map((scope) => (
                        <Badge key={scope} variant="secondary" className="font-normal">
                          {scopeText(scope, "scope")}
                        </Badge>
                      ))}
                    </div>
                    <div className="flex flex-wrap gap-x-3 gap-y-0.5 text-[12px] text-muted-foreground">
                      <span>{t("settings.tokens.created_at", { date: formatDate(token.created_at) })}</span>
                      <span>
                        {token.last_used_at
                          ? token.last_used_ip
                            ? t("settings.tokens.used_from", { date: formatDateTime(token.last_used_at), ip: token.last_used_ip })
                            : t("settings.tokens.used_at", { date: formatDateTime(token.last_used_at) })
                          : t("settings.tokens.never_used")}
                      </span>
                      <span className={cn(token.expired && "text-rose-600 dark:text-rose-400")}>
                        {token.expires_at
                          ? token.expired
                            ? t("settings.tokens.expired_at", { date: formatDate(token.expires_at) })
                            : t("settings.tokens.expires", { date: formatDate(token.expires_at) })
                          : t("settings.tokens.no_expiry")}
                      </span>
                    </div>
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    className="text-destructive hover:text-destructive"
                    onClick={() => setRevoking(token)}
                  >
                    {token.expired ? t("settings.tokens.remove") : t("settings.tokens.revoke")}
                  </Button>
                </li>
              ))}
            </ul>
          )}
          {query.data && (
            <p className={cn("mt-3 text-[12px]", full ? "text-amber-600 dark:text-amber-500" : "text-muted-foreground")}>
              {full ? t("settings.tokens.limit_reached", { limit }) : t("settings.tokens.count", { count: active, limit })}
            </p>
          )}
        </QueryState>
      </SettingsSection>

      <UsageSection scopes={scopes.length > 0 ? scopes : Object.keys(SCOPE_ROUTES)} />

      <CreateTokenDialog
        open={creating}
        onOpenChange={setCreating}
        scopes={scopes}
        onCreated={() => void qc.invalidateQueries({ queryKey: TOKENS_KEY })}
      />

      <ConfirmDialog
        open={revoking !== null}
        onOpenChange={(open) => !open && setRevoking(null)}
        title={t("settings.tokens.revoke_title", { name: revoking?.name ?? "" })}
        desc={t("settings.tokens.revoke_hint")}
        destructive
        cancelBtnText={t("common.cancel")}
        confirmText={revoking?.expired ? t("settings.tokens.remove") : t("settings.tokens.revoke")}
        isLoading={revoke.isPending}
        handleConfirm={() => revoking && revoke.mutate(revoking.id)}
      />
    </>
  );
}

function CreateTokenDialog({
  open,
  onOpenChange,
  scopes,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  scopes: string[];
  onCreated: () => void;
}) {
  const t = useT();
  const scopeText = useScopeText();
  const { copied, copy } = useCopy();
  const [name, setName] = useState("");
  const [picked, setPicked] = useState<string[]>(DEFAULT_SCOPES);
  const [days, setDays] = useState(90);
  const [issued, setIssued] = useState<{ key: string; name: string } | null>(null);

  const reset = () => {
    setName("");
    setPicked(DEFAULT_SCOPES);
    setDays(90);
    setIssued(null);
  };

  const create = useMutation({
    mutationFn: () => createAPIToken({ name: name.trim(), scopes: picked, expires_in_days: days }),
    onSuccess: (res) => {
      setIssued({ key: res.key, name: res.token.name });
      onCreated();
    },
    onError: (err) => toast.error(errorText(err, t("settings.tokens.create_failed"))),
  });

  const close = (next: boolean) => {
    if (next) return;
    onOpenChange(false);
    window.setTimeout(reset, 200);
  };

  const toggle = (scope: string, on: boolean) =>
    setPicked((prev) => (on ? [...prev.filter((s) => s !== scope), scope] : prev.filter((s) => s !== scope)));

  const trimmed = name.trim();
  const valid = trimmed.length > 0 && trimmed.length <= 64 && picked.length > 0;

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent
        showCloseButton={!issued}
        className="max-h-[90svh] overflow-y-auto sm:max-w-xl"
        onInteractOutside={(e) => {
          if (issued) e.preventDefault();
        }}
        onEscapeKeyDown={(e) => {
          if (issued) e.preventDefault();
        }}
      >
        {issued ? (
          <>
            <DialogHeader>
              <DialogTitle>{t("settings.tokens.issued_title")}</DialogTitle>
              <DialogDescription>{t("settings.tokens.issued_hint")}</DialogDescription>
            </DialogHeader>
            <div className="space-y-2">
              <div className="text-[13px] font-medium">{issued.name}</div>
              <div className="flex items-stretch gap-2">
                <code className="min-w-0 flex-1 rounded-lg border border-border bg-muted/40 px-3 py-2.5 font-mono text-[13px] break-all select-all">
                  {issued.key}
                </code>
                <Button type="button" variant="outline" className="h-auto" onClick={() => void copy(issued.key, "key")}>
                  {copied === "key" ? <Check className="size-4" /> : <Copy className="size-4" />}
                  <span className="max-sm:sr-only">{t("settings.common.copy")}</span>
                </Button>
              </div>
            </div>
            <DialogFooter>
              <Button type="button" onClick={() => close(false)}>
                {t("settings.tokens.issued_done")}
              </Button>
            </DialogFooter>
          </>
        ) : (
          <form
            className="grid gap-5"
            onSubmit={(e) => {
              e.preventDefault();
              if (picked.length === 0) {
                toast.error(t("settings.tokens.pick_scope"));
                return;
              }
              if (valid && !create.isPending) create.mutate();
            }}
          >
            <DialogHeader>
              <DialogTitle>{t("settings.tokens.dialog_title")}</DialogTitle>
              <DialogDescription>{t("settings.tokens.dialog_hint")}</DialogDescription>
            </DialogHeader>

            <Field label={t("settings.tokens.name")} hint={t("settings.tokens.name_hint")} htmlFor="token-name">
              <Input
                id="token-name"
                value={name}
                maxLength={64}
                autoFocus
                placeholder={t("settings.tokens.name_placeholder")}
                onChange={(e) => setName(e.target.value)}
              />
            </Field>

            <div className="space-y-2">
              <div className="flex items-center justify-between gap-3">
                <span className="text-[13px] font-medium">{t("settings.tokens.scopes")}</span>
                <span className="text-[12px] text-muted-foreground">
                  {t("settings.tokens.scopes_selected", { count: picked.length })}
                </span>
              </div>
              <div className="grid gap-1.5 sm:grid-cols-2">
                {scopes.map((scope) => {
                  const on = picked.includes(scope);
                  return (
                    <label
                      key={scope}
                      className={cn(
                        "flex cursor-pointer items-start gap-2.5 rounded-lg border px-3 py-2.5 transition-colors",
                        on ? "border-primary/50 bg-primary/5" : "border-border hover:bg-muted/50"
                      )}
                    >
                      <Checkbox checked={on} onCheckedChange={(v) => toggle(scope, v === true)} className="mt-0.5" />
                      <span className="min-w-0">
                        <span className="block text-[13px] font-medium">{scopeText(scope, "scope")}</span>
                        <span className="block text-[12px] leading-snug text-muted-foreground">
                          {scopeText(scope, "scope_hint")}
                        </span>
                      </span>
                    </label>
                  );
                })}
              </div>
            </div>

            <div className="space-y-2">
              <span className="text-[13px] font-medium">{t("settings.tokens.expiry")}</span>
              <div role="radiogroup" aria-label={t("settings.tokens.expiry")} className="grid grid-cols-2 gap-1.5 sm:grid-cols-4">
                {EXPIRY_DAYS.map((value) => (
                  <button
                    key={value}
                    type="button"
                    role="radio"
                    aria-checked={days === value}
                    onClick={() => setDays(value)}
                    className={cn(
                      "h-9 rounded-lg border text-[13px] transition-colors",
                      days === value ? "border-primary bg-primary/5 font-medium" : "border-border hover:bg-muted/50"
                    )}
                  >
                    {t(`settings.tokens.days_${value}`)}
                  </button>
                ))}
              </div>
              {days === 0 && (
                <p className="text-[12px] text-amber-600 dark:text-amber-500">{t("settings.tokens.no_expiry_warning")}</p>
              )}
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => close(false)}>
                {t("common.cancel")}
              </Button>
              <Button type="submit" disabled={!valid || create.isPending}>
                {create.isPending && <Loader2 className="size-4 animate-spin" />}
                {t("settings.tokens.submit")}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}

function UsageSection({ scopes }: { scopes: string[] }) {
  const t = useT();
  const scopeText = useScopeText();
  const { copied, copy } = useCopy();
  const base = apiBase();
  const example = [
    `curl ${base}/v1/servers \\`,
    `  -H "X-API-Key: vtx_…"`,
    "",
    `curl -X POST ${base}/v1/servers/SERVER_ID/power \\`,
    `  -H "X-API-Key: vtx_…" \\`,
    `  -H "Content-Type: application/json" \\`,
    `  -d '{"action":"restart"}'`,
  ].join("\n");

  return (
    <SettingsSection title={t("settings.tokens.usage_title")} description={t("settings.tokens.usage_hint")}>
      <div className="space-y-5">
        <div className="space-y-2">
          <div className="flex items-center justify-between gap-3">
            <span className="text-[13px] font-medium">{t("settings.tokens.usage_example")}</span>
            <Button type="button" size="sm" variant="ghost" onClick={() => void copy(example, "example")}>
              {copied === "example" ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
              {t("settings.common.copy")}
            </Button>
          </div>
          <pre className="overflow-x-auto rounded-xl border border-border bg-muted/40 px-4 py-3 font-mono text-[12.5px] leading-relaxed">
            {example}
          </pre>
        </div>
        <div className="space-y-2">
          <span className="text-[13px] font-medium">{t("settings.tokens.usage_routes")}</span>
          <div className="divide-y divide-border rounded-xl border border-border">
            {scopes.map((scope) => (
              <div key={scope} className="grid gap-1.5 px-4 py-3 sm:grid-cols-[200px_1fr] sm:gap-4">
                <div>
                  <div className="text-[13px] font-medium">{scopeText(scope, "scope")}</div>
                  <code className="font-mono text-[11.5px] text-muted-foreground">{scope}</code>
                </div>
                <ul className="space-y-0.5 font-mono text-[12px] text-muted-foreground">
                  {(SCOPE_ROUTES[scope] ?? []).map((route) => (
                    <li key={route} className="break-all">
                      {route}
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>
      </div>
    </SettingsSection>
  );
}
