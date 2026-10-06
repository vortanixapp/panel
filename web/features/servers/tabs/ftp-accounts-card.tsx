"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { KeyRound, Plus, RefreshCw, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Btn, EmptyState, Panel, VX_INPUT } from "@/components/vx/panel-ui";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import {
  createServerFtpAccount,
  deleteServerFtpAccount,
  fetchServerFtp,
  resetServerFtpPassword,
  type ServerFtpAccount,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { confirmAction } from "@/components/action-dialog";

import { pollMs } from "@/lib/public-settings";
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
        ? pollMs(2000)
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
    <Panel
      title={t("servers.ftp.card_title")}
      aside={
        data?.host ? (
          <span className="font-mono text-[12px] break-all text-[var(--vx-muted)] select-all">
            sftp://{data.host}:{data.port}
          </span>
        ) : null
      }
      flush
    >
      {isLoading ? (
        <div className="p-6">
          <Skeleton className="h-20 w-full" />
        </div>
      ) : accounts.length === 0 ? (
        <EmptyState>{t("servers.ftp.no_accounts")}</EmptyState>
      ) : (
        <div className="overflow-x-auto">
          <div className="min-w-[560px]">
            <div className="grid grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_110px_120px] gap-3 border-b border-[var(--vx-border)] px-6 py-3 text-[11px] tracking-[0.08em] text-[var(--vx-ghost)] uppercase">
              <span>{t("servers.ftp.col_login")}</span>
              <span>{t("common.password")}</span>
              <span>{t("common.status")}</span>
              <span className="text-right">{t("common.actions")}</span>
            </div>
            {accounts.map((a) => (
              <div
                key={a.id}
                className="grid grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_110px_120px] items-center gap-3 border-b border-[var(--vx-border)] px-6 py-[13px] text-[13px] last:border-b-0"
              >
                <span className="truncate font-mono text-[12.5px]">{a.username}</span>
                <span className="font-mono text-[12.5px]">
                  {revealed[a.id] ? (
                    <span className="break-all">{a.password}</span>
                  ) : (
                    <button
                      type="button"
                      className="text-[var(--srv-accent)] hover:opacity-80"
                      onClick={() => setRevealed((r) => ({ ...r, [a.id]: true }))}
                    >
                      {t("servers.ftp.reveal")}
                    </button>
                  )}
                </span>
                <span>
                  <span
                    className={cn(
                      "inline-block rounded-full px-[11px] py-[3px] text-[12px]",
                      a.status === "failed"
                        ? "bg-[var(--vx-danger-tint)] text-[var(--vx-danger)]"
                        : "bg-[var(--srv-accent-soft)] text-[var(--srv-accent)]"
                    )}
                  >
                    {t(`servers.ftp.status.${a.status}`)}
                  </span>
                  {a.status === "failed" && a.error_message ? (
                    <div className="mt-1 text-[11px] text-[var(--vx-danger)]">{a.error_message}</div>
                  ) : null}
                </span>
                <span className="flex justify-end gap-2">
                  <Btn
                    size="sm"
                    disabled={resetMut.isPending}
                    aria-label={t("servers.ftp.change_password")}
                    title={t("servers.ftp.change_password")}
                    onClick={() => resetMut.mutate(a.username)}
                  >
                    <KeyRound className="h-3 w-3" />
                  </Btn>
                  <Btn
                    size="sm"
                    tone="danger"
                    disabled={deleteMut.isPending}
                    aria-label={t("common.delete")}
                    title={t("common.delete")}
                    onClick={async () => {
                      if (
                        await confirmAction(
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
                  </Btn>
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="flex flex-col gap-2 border-t border-[var(--vx-border)] px-6 py-4 sm:flex-row sm:items-center">
        <input
          placeholder={t("servers.ftp.suffix_placeholder")}
          value={suffix}
          onChange={(e) => setSuffix(e.target.value)}
          className={cn(VX_INPUT, "h-[38px] rounded-[10px] sm:max-w-xs")}
          disabled={limitReached}
        />
        <Btn
          tone="primary"
          onClick={() => createMut.mutate()}
          disabled={createMut.isPending || limitReached}
        >
          <Plus className="h-4 w-4" />
          {t("servers.ftp.create_account")}
        </Btn>
        <Btn onClick={() => void invalidate()} aria-label={t("common.refresh")}>
          <RefreshCw className="h-4 w-4" />
        </Btn>
        {limitReached ? (
          <span className="text-[12px] text-[var(--vx-faint)]">
            {t("servers.ftp.limit_reached", { limit: data?.limit ?? 0 })}
          </span>
        ) : null}
      </div>

      <p className="border-t border-[var(--vx-border)] px-6 py-3.5 text-[11.5px] text-[var(--vx-faint)]">
        {t("servers.ftp.sftp_note_head")} <span className="font-mono">/data</span>
        {t("servers.ftp.sftp_note_tail")}
      </p>
    </Panel>
  );
}
