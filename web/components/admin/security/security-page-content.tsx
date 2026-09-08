"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import {
  createIPBlock,
  deleteIPBlock,
  fetchAdminSettings,
  fetchIPBlocks,
  fetchLoginAttempts,
  updateAdminSettings,
  type IPBlock,
  type LoginAttempt,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

function fmtDateTime(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString(localeTag());
}

export function SecurityPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const [onlyFailed, setOnlyFailed] = useState(true);
  const [search, setSearch] = useState("");
  const [applied, setApplied] = useState("");
  const [blockIP, setBlockIP] = useState("");
  const [blockReason, setBlockReason] = useState("");
  const [blockHours, setBlockHours] = useState("24");

  const attemptsQuery = useQuery({
    queryKey: ["login-attempts", onlyFailed, applied],
    queryFn: () => fetchLoginAttempts({ failed: onlyFailed, search: applied || undefined }),
    refetchInterval: 30_000,
  });

  const blocksQuery = useQuery({
    queryKey: ["ip-blocks"],
    queryFn: fetchIPBlocks,
  });

  const settingsQuery = useQuery({
    queryKey: ["admin-settings-security"],
    queryFn: fetchAdminSettings,
  });
  const require2FA = settingsQuery.data?.values?.["security.staff_2fa_required"] === "1";

  const toggle2FAMut = useMutation({
    mutationFn: (enabled: boolean) =>
      updateAdminSettings({ "security.staff_2fa_required": enabled ? "1" : "0" }),
    onSuccess: (_res, enabled) => {
      toast.success(
        enabled
          ? t("admin.security.2fa_on")
          : t("admin.security.2fa_off")
      );
      void qc.invalidateQueries({ queryKey: ["admin-settings-security"] });
    },
    onError: (e: Error) => toast.error(e.message || t("common.save_failed")),
  });

  const blockMut = useMutation({
    mutationFn: () =>
      createIPBlock({
        ip: blockIP.trim(),
        reason: blockReason.trim(),
        hours: Number(blockHours) || 0,
      }),
    onSuccess: () => {
      toast.success(t("admin.security.blocked"));
      setBlockIP("");
      setBlockReason("");
      void qc.invalidateQueries({ queryKey: ["ip-blocks"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.security.block_failed")),
  });

  const unblockMut = useMutation({
    mutationFn: (id: string) => deleteIPBlock(id),
    onSuccess: () => {
      toast.success(t("admin.security.unblocked"));
      void qc.invalidateQueries({ queryKey: ["ip-blocks"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.security.unblock_failed")),
  });

  const attempts = attemptsQuery.data?.attempts ?? [];
  const suspicious = attemptsQuery.data?.suspicious ?? [];
  const blocks = blocksQuery.data?.blocks ?? [];

  return (
    <PageShell variant="admin">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">
          {t("admin.security.title")}
        </h1>
        <p className="text-sm text-muted-foreground">
          {t("admin.security.subtitle")}
        </p>
      </div>

      <div className="mb-6 rounded-lg border bg-card p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <div className="text-sm font-medium">
              {t("admin.security.require_2fa")}
            </div>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("admin.security.require_2fa_hint")}
            </p>
          </div>
          <Switch
            checked={require2FA}
            onCheckedChange={(v) => toggle2FAMut.mutate(v)}
            disabled={toggle2FAMut.isPending || settingsQuery.isLoading}
          />
        </div>
      </div>

      {suspicious.length > 0 ? (
        <div className="mb-6 rounded-lg border border-amber-500/40 bg-amber-500/5 p-4">
          <div className="mb-2 text-sm font-medium">
            {t("admin.security.suspicious_title")}
          </div>
          <div className="flex flex-wrap gap-2">
            {suspicious.map((s) => (
              <button
                key={s.ip}
                type="button"
                onClick={() => setBlockIP(s.ip)}
                className="rounded-md border px-2 py-1 font-mono text-xs hover:bg-accent"
                title={t("admin.security.suspicious_hint", {
                  when: fmtDateTime(s.last_at),
                })}
              >
                {s.ip} · {s.fails}
              </button>
            ))}
          </div>
        </div>
      ) : null}

      <div className="mb-6 rounded-lg border bg-card p-4">
        <div className="mb-3 text-sm font-medium">
          {t("admin.security.block_title")}
        </div>
        <div className="flex flex-wrap items-end gap-3">
          <div className="space-y-1">
            <Label>IP</Label>
            <Input
              value={blockIP}
              onChange={(e) => setBlockIP(e.target.value)}
              placeholder="203.0.113.10"
              className="w-44 font-mono"
            />
          </div>
          <div className="flex-1 space-y-1">
            <Label>{t("admin.maintenance.reason")}</Label>
            <Input
              value={blockReason}
              onChange={(e) => setBlockReason(e.target.value)}
              placeholder={t("admin.security.reason_placeholder")}
            />
          </div>
          <div className="space-y-1">
            <Label>{t("admin.security.hours")}</Label>
            <Input
              type="number"
              min="0"
              value={blockHours}
              onChange={(e) => setBlockHours(e.target.value)}
              className="w-32"
            />
          </div>
          <Button onClick={() => blockMut.mutate()} disabled={blockMut.isPending || !blockIP.trim()}>
            {t("admin.security.block")}
          </Button>
        </div>
      </div>

      <div className="mb-6 rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t("admin.security.blocks_title", { count: blocks.length })}
        </div>
        {blocksQuery.isLoading ? (
          <div className="p-4">
            <Skeleton className="h-12 w-full" />
          </div>
        ) : blocks.length === 0 ? (
          <div className="p-6 text-center text-sm text-muted-foreground">
            {t("admin.security.blocks_empty")}
          </div>
        ) : (
          <div className="divide-y">
            {blocks.map((b: IPBlock) => (
              <div key={b.id} className="flex flex-wrap items-center justify-between gap-3 p-3">
                <div>
                  <span className="font-mono text-sm">{b.ip}</span>
                  <span className="ml-2 text-xs text-muted-foreground">
                    {b.reason || t("admin.security.no_reason")} ·{" "}
                    {b.expires_at
                      ? t("admin.security.until", {
                          when: fmtDateTime(b.expires_at),
                        })
                      : t("admin.security.forever")}
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  {b.auto ? (
                    <Badge variant="outline">
                      {t("admin.security.auto")}
                    </Badge>
                  ) : (
                    <Badge variant="secondary">
                      {t("admin.security.manual")}
                    </Badge>
                  )}
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => unblockMut.mutate(b.id)}
                    disabled={unblockMut.isPending}
                  >
                    {t("admin.security.unblock")}
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="rounded-lg border bg-card">
        <div className="flex flex-wrap items-center gap-3 border-b px-4 py-3">
          <span className="text-sm font-medium">
            {t("admin.security.log_title")}
          </span>
          <label className="flex items-center gap-2 text-xs text-muted-foreground">
            <input
              type="checkbox"
              checked={onlyFailed}
              onChange={(e) => setOnlyFailed(e.target.checked)}
              className="size-3.5"
            />
            {t("admin.security.only_failed")}
          </label>
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") setApplied(search.trim());
            }}
            placeholder={t("admin.security.search_placeholder")}
            className="ml-auto w-64"
          />
          <Button variant="outline" size="sm" onClick={() => setApplied(search.trim())}>
            {t("common.search")}
          </Button>
        </div>

        {attemptsQuery.isLoading ? (
          <div className="p-4">
            <Skeleton className="h-24 w-full" />
          </div>
        ) : attempts.length === 0 ? (
          <div className="p-8 text-center text-sm text-muted-foreground">
            {t("admin.security.log_empty")}
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-xs text-muted-foreground">
                <th className="px-4 py-2 text-left font-normal">
                  {t("admin.integrations.col_when")}
                </th>
                <th className="px-4 py-2 text-left font-normal">
                  {t("common.email")}
                </th>
                <th className="px-4 py-2 text-left font-normal">IP</th>
                <th className="px-4 py-2 text-left font-normal">
                  {t("admin.security.col_result")}
                </th>
              </tr>
            </thead>
            <tbody>
              {attempts.map((a: LoginAttempt, i) => (
                <tr key={`${a.created_at}-${i}`} className="border-t">
                  <td className="px-4 py-2 whitespace-nowrap">{fmtDateTime(a.created_at)}</td>
                  <td className="px-4 py-2">{a.email || "—"}</td>
                  <td className="px-4 py-2 font-mono text-xs">{a.ip || "—"}</td>
                  <td className="px-4 py-2">
                    {a.success ? (
                      <Badge variant="secondary">
                        {t("admin.security.result_ok")}
                      </Badge>
                    ) : (
                      <span className="text-destructive">
                        {a.reason || t("admin.security.result_denied")}
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </PageShell>
  );
}
