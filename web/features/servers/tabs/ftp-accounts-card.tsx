"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { KeyRound, Plus, RefreshCw, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  createServerFtpAccount,
  deleteServerFtpAccount,
  fetchServerFtp,
  resetServerFtpPassword,
  type ServerFtpAccount,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";

export function FtpAccountsCard({ serverId }: { serverId: string }) {
  const t = useT();
  const qc = useQueryClient();
  const [suffix, setSuffix] = useState("");
  const [revealed, setRevealed] = useState<Record<string, boolean>>({});

  const { data, isLoading } = useQuery({
    queryKey: ["server-ftp", serverId],
    queryFn: () => fetchServerFtp(serverId),
    enabled: !!serverId,
    refetchInterval: (query) =>
      query.state.data?.accounts.some((a) => a.status === "pending" || a.status === "deleting")
        ? 2000
        : false,
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ["server-ftp", serverId] });

  const createMut = useMutation({
    mutationFn: () => createServerFtpAccount(serverId, suffix.trim()),
    onSuccess: () => {
      setSuffix("");
      toast.success(t("servers.ftp.account_creating"));
      void invalidate();
    },
    onError: (e) =>
      toast.error(
        e instanceof Error ? e.message : t("servers.ftp.account_create_failed")
      ),
  });

  const resetMut = useMutation({
    mutationFn: (username: string) => resetServerFtpPassword(serverId, username),
    onSuccess: () => {
      toast.success(t("servers.ftp.password_changing"));
      void invalidate();
    },
    onError: (e) =>
      toast.error(
        e instanceof Error ? e.message : t("servers.ftp.password_change_failed")
      ),
  });

  const deleteMut = useMutation({
    mutationFn: (username: string) => deleteServerFtpAccount(serverId, username),
    onSuccess: () => {
      toast.success(t("servers.ftp.account_deleting"));
      void invalidate();
    },
    onError: (e) =>
      toast.error(
        e instanceof Error ? e.message : t("servers.ftp.account_delete_failed")
      ),
  });

  const accounts = data?.accounts ?? [];
  const limitReached = !!data && accounts.length >= data.limit;

  return (
    <Card className="mb-6">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-base">{t("servers.ftp.card_title")}</CardTitle>
        {data?.host ? (
          <span className="font-mono text-xs text-muted-foreground">
            sftp://{data.host}:{data.port}
          </span>
        ) : null}
      </CardHeader>
      <CardContent className="space-y-4">
        {isLoading ? (
          <Skeleton className="h-20 w-full" />
        ) : accounts.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t("servers.ftp.no_accounts")}
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-border text-sm">
              <thead className="bg-muted/40">
                <tr>
                  <th className="px-3 py-2 text-left font-medium text-muted-foreground">
                    {t("servers.ftp.col_login")}
                  </th>
                  <th className="px-3 py-2 text-left font-medium text-muted-foreground">
                    {t("common.password")}
                  </th>
                  <th className="px-3 py-2 text-left font-medium text-muted-foreground">
                    {t("common.status")}
                  </th>
                  <th className="px-3 py-2 text-right font-medium text-muted-foreground">
                    {t("common.actions")}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {accounts.map((a) => (
                  <tr key={a.id}>
                    <td className="px-3 py-2 font-mono text-xs">{a.username}</td>
                    <td className="px-3 py-2 font-mono text-xs">
                      {revealed[a.id] ? (
                        <span className="break-all">{a.password}</span>
                      ) : (
                        <button
                          className="text-primary underline"
                          onClick={() => setRevealed((s) => ({ ...s, [a.id]: true }))}
                        >
                          {t("servers.ftp.reveal")}
                        </button>
                      )}
                    </td>
                    <td className="px-3 py-2 text-xs">
                      {t(`servers.ftp.status.${a.status}`)}
                      {a.status === "failed" && a.error_message ? (
                        <div className="mt-1 text-[11px] text-destructive">{a.error_message}</div>
                      ) : null}
                    </td>
                    <td className="px-3 py-2">
                      <div className="flex justify-end gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={resetMut.isPending}
                          onClick={() => resetMut.mutate(a.username)}
                        >
                          <KeyRound className="mr-1 h-3 w-3" />
                          {t("servers.ftp.change_password")}
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={deleteMut.isPending}
                          onClick={() => {
                            if (
                              confirm(
                                t("servers.ftp.account_delete_confirm", {
                                  name: a.username,
                                })
                              )
                            ) {
                              deleteMut.mutate(a.username);
                            }
                          }}
                        >
                          <Trash2 className="h-3 w-3" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <Input
            placeholder={t("servers.ftp.suffix_placeholder")}
            value={suffix}
            onChange={(e) => setSuffix(e.target.value)}
            className="sm:max-w-xs"
            disabled={limitReached}
          />
          <Button
            onClick={() => createMut.mutate()}
            disabled={createMut.isPending || limitReached}
          >
            <Plus className="mr-1 h-4 w-4" />
            {t("servers.ftp.create_account")}
          </Button>
          <Button variant="outline" size="icon" onClick={() => void invalidate()}>
            <RefreshCw className="h-4 w-4" />
          </Button>
          {limitReached ? (
            <span className="text-xs text-muted-foreground">
              {t("servers.ftp.limit_reached", { limit: data?.limit ?? 0 })}
            </span>
          ) : null}
        </div>

        <p className="text-[11px] text-muted-foreground">
          {t("servers.ftp.sftp_note_head")}{" "}
          <span className="font-mono">/data</span>
          {t("servers.ftp.sftp_note_tail")}
        </p>
      </CardContent>
    </Card>
  );
}
