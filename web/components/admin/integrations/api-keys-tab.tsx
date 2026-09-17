"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus } from "lucide-react";
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
  createAdminAPIKey,
  fetchAdminAPIKeys,
  revokeAdminAPIKey,
  type AdminAPIKey,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";

const SCOPES_SHOWN = 3;

export function ApiKeysTab() {
  const t = useT();
  const qc = useQueryClient();

  const keysQuery = useQuery({
    queryKey: ["admin-api-keys"],
    queryFn: fetchAdminAPIKeys,
  });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [keyName, setKeyName] = useState("");
  const [keyScopes, setKeyScopes] = useState<string[]>([]);
  const [issuedKey, setIssuedKey] = useState("");
  const [revokeTarget, setRevokeTarget] = useState<AdminAPIKey | null>(null);

  const createKeyMut = useMutation({
    mutationFn: () => createAdminAPIKey({ name: keyName.trim(), scopes: keyScopes }),
    onSuccess: (res) => {
      setIssuedKey(res.key);
      setKeyName("");
      setKeyScopes([]);
      setDialogOpen(false);
      void qc.invalidateQueries({ queryKey: ["admin-api-keys"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.integrations.key_create_failed")),
  });

  const revokeKeyMut = useMutation({
    mutationFn: (id: string) => revokeAdminAPIKey(id),
    onSuccess: () => {
      toast.success(t("admin.integrations.key_revoked"));
      setRevokeTarget(null);
      void qc.invalidateQueries({ queryKey: ["admin-api-keys"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.integrations.revoke_failed")),
  });

  const keys = keysQuery.data?.keys ?? [];
  const allScopes = keysQuery.data?.all_scopes ?? [];
  const canCreate = keyName.trim() !== "" && keyScopes.length > 0;

  const toggleScope = (scope: string) =>
    setKeyScopes((prev) =>
      prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope]
    );

  return (
    <div className="flex flex-col gap-4">
      <TabHeader
        title={t("admin.integrations.keys_title")}
        description={t("admin.integrations.keys_hint")}
        action={
          <Button onClick={() => setDialogOpen(true)}>
            <Plus className="mr-1.5 size-4" />
            {t("admin.integrations.issue_key")}
          </Button>
        }
      />

      {issuedKey ? (
        <OneTimeSecret
          title={t("admin.integrations.key_once")}
          value={issuedKey}
          hint={
            <>
              {t("admin.integrations.key_hash_hint")}{" "}
              <code className="font-mono">X-API-Key</code>
            </>
          }
          onDismiss={() => setIssuedKey("")}
        />
      ) : null}

      <div className="overflow-hidden rounded-xl border bg-card">
        {keysQuery.isLoading ? (
          <div className="p-4">
            <Skeleton className="h-24 w-full" />
          </div>
        ) : keys.length === 0 ? (
          <EmptyBlock>{t("admin.integrations.keys_empty")}</EmptyBlock>
        ) : (
          <div className="overflow-x-auto">
            <table className="vx-tbl vx-tbl-flat w-full min-w-[760px] text-sm">
              <thead>
                <tr className="border-b text-left text-xs text-muted-foreground">
                  <th className="px-4 py-2.5 font-medium">{t("common.name")}</th>
                  <th className="px-4 py-2.5 font-medium">
                    {t("admin.integrations.col_scopes")}
                  </th>
                  <th className="px-4 py-2.5 font-medium">
                    {t("admin.integrations.col_created")}
                  </th>
                  <th className="px-4 py-2.5 font-medium">
                    {t("admin.integrations.col_used")}
                  </th>
                  <th className="px-4 py-2.5 font-medium">{t("common.status")}</th>
                  <th className="px-4 py-2.5" />
                </tr>
              </thead>
              <tbody>
                {keys.map((key) => (
                  <tr key={key.id} className="border-b align-top last:border-0">
                    <td className="px-4 py-3" data-cell="full">
                      <div className="font-medium">{key.name}</div>
                      <code className="font-mono text-xs text-muted-foreground">
                        {key.prefix}…
                      </code>
                    </td>
                    <td className="px-4 py-3" data-label={t("admin.integrations.col_scopes")}>
                      <div className="flex max-w-[300px] flex-wrap gap-1 max-md:justify-end">
                        {key.scopes.slice(0, SCOPES_SHOWN).map((scope) => (
                          <Badge
                            key={scope}
                            variant="outline"
                            className="font-mono text-[10px]"
                          >
                            {scope}
                          </Badge>
                        ))}
                        {key.scopes.length > SCOPES_SHOWN ? (
                          <span
                            className="px-1 text-xs text-muted-foreground"
                            title={key.scopes.slice(SCOPES_SHOWN).join(", ")}
                          >
                            +{key.scopes.length - SCOPES_SHOWN}
                          </span>
                        ) : null}
                      </div>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-muted-foreground" data-label={t("admin.integrations.col_created")}>
                      {fmtDateTime(key.created_at)}
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-muted-foreground" data-label={t("admin.integrations.col_used")}>
                      {fmtDateTime(key.last_used_at)}
                    </td>
                    <td className="px-4 py-3" data-label={t("common.status")}>
                      {key.active ? (
                        <Badge variant="secondary">
                          {t("admin.integrations.key_active")}
                        </Badge>
                      ) : (
                        <Badge variant="outline">
                          {t("admin.integrations.key_revoked_badge")}
                        </Badge>
                      )}
                    </td>
                    <td className="px-4 py-3 text-right" data-cell="actions">
                      {key.active ? (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => setRevokeTarget(key)}
                        >
                          {t("admin.integrations.revoke")}
                        </Button>
                      ) : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <p className="text-xs text-muted-foreground">
        {t("admin.integrations.key_scope_hint_before")}{" "}
        <code className="font-mono">/v1/admin/*</code>{" "}
        {t("admin.integrations.key_scope_hint_after")}
      </p>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>{t("admin.integrations.issue_key")}</DialogTitle>
            <DialogDescription>{t("admin.integrations.keys_hint")}</DialogDescription>
          </DialogHeader>
          <form
            className="flex flex-col gap-4"
            onSubmit={(e) => {
              e.preventDefault();
              if (canCreate) createKeyMut.mutate();
            }}
          >
            <div className="space-y-1.5">
              <Label htmlFor="api-key-name">{t("common.name")}</Label>
              <Input
                id="api-key-name"
                value={keyName}
                onChange={(e) => setKeyName(e.target.value)}
                placeholder={t("admin.integrations.key_name_placeholder")}
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <div className="flex items-center justify-between gap-3">
                <Label>{t("admin.integrations.key_scopes")}</Label>
                <span className="text-xs text-muted-foreground">
                  {keyScopes.length} / {allScopes.length}
                </span>
              </div>
              <div className="grid max-h-[320px] gap-1.5 overflow-y-auto rounded-lg border p-2 sm:grid-cols-2">
                {allScopes.map((scope) => (
                  <label
                    key={scope}
                    className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-xs hover:bg-muted"
                  >
                    <Checkbox
                      checked={keyScopes.includes(scope)}
                      onCheckedChange={() => toggleScope(scope)}
                    />
                    <span className="font-mono">{scope}</span>
                  </label>
                ))}
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                {t("common.cancel")}
              </Button>
              <Button type="submit" disabled={!canCreate || createKeyMut.isPending}>
                {t("admin.integrations.issue_key")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!revokeTarget}
        onOpenChange={(open) => {
          if (!open) setRevokeTarget(null);
        }}
        title={t("admin.integrations.revoke_confirm", { name: revokeTarget?.name ?? "" })}
        desc={t("admin.integrations.revoke_desc")}
        cancelBtnText={t("common.cancel")}
        confirmText={t("admin.integrations.revoke")}
        destructive
        isLoading={revokeKeyMut.isPending}
        handleConfirm={() => {
          if (revokeTarget) revokeKeyMut.mutate(revokeTarget.id);
        }}
      />
    </div>
  );
}
