"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { useT } from "@/hooks/use-translations";
import {
  destroyAccountSession,
  fetchAccountLogins,
  fetchAccountSessions,
  logout,
  type AccountLogin,
  type AccountSession,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { errorText, formatDateTime, QueryState, SettingsSection } from "./ui";

type Pending = { kind: "one"; session: AccountSession } | { kind: "others" } | { kind: "all" } | { kind: "notme" } | null;

function deviceIcon(device: string | undefined) {
  const d = (device ?? "").toLowerCase();
  if (!d) return "ri-question-line";
  if (d.includes("android") || d.includes("ios")) return "ri-smartphone-line";
  return "ri-computer-line";
}

export function DevicesTab() {
  const t = useT();
  const qc = useQueryClient();
  const router = useRouter();
  const [pending, setPending] = useState<Pending>(null);
  const sessions = useQuery({ queryKey: ["account-sessions"], queryFn: fetchAccountSessions });
  const logins = useInfiniteQuery({
    queryKey: ["account-logins", "history"],
    queryFn: ({ pageParam }) => fetchAccountLogins(pageParam || undefined),
    initialPageParam: 0,
    getNextPageParam: (last) => (last.has_more ? last.next_before : undefined),
  });

  const destroy = useMutation({
    mutationFn: (p: NonNullable<Pending>) => {
      if (p.kind === "one") return destroyAccountSession({ session_id: p.session.id });
      if (p.kind === "all") return destroyAccountSession({ all: true });
      return destroyAccountSession({ all_others: true });
    },
    onSuccess: async (res, p) => {
      setPending(null);
      if (p.kind === "all") {
        await logout().catch(() => undefined);
        router.replace("/login");
        return;
      }
      void qc.invalidateQueries({ queryKey: ["account-sessions"] });
      if (p.kind === "notme") {
        toast.success(t("settings.devices.notme_done", { count: res.closed }));
        router.replace("/settings?tab=password");
        return;
      }
      toast.success(
        p.kind === "one" ? t("settings.devices.closed_one") : t("settings.devices.closed_others", { count: res.closed })
      );
    },
    onError: (err) => toast.error(errorText(err, t("common.error"))),
  });

  const list = sessions.data?.sessions ?? [];
  const others = list.filter((s) => !s.is_current).length;
  const loginRows: AccountLogin[] = logins.data?.pages.flatMap((p) => p.logins) ?? [];

  const confirmCopy = (() => {
    switch (pending?.kind) {
      case "one":
        return {
          title: t("settings.devices.close_one_title"),
          desc: t("settings.devices.close_one_hint", { device: pending.session.device || t("settings.devices.unknown") }),
          confirm: t("settings.devices.close"),
        };
      case "others":
        return { title: t("settings.devices.close_others_title"), desc: t("settings.devices.close_others_hint"), confirm: t("settings.devices.close_others") };
      case "all":
        return { title: t("settings.devices.close_all_title"), desc: t("settings.devices.close_all_hint"), confirm: t("settings.devices.close_all") };
      case "notme":
        return { title: t("settings.devices.notme_title"), desc: t("settings.devices.notme_hint"), confirm: t("settings.devices.notme_confirm") };
      default:
        return { title: "", desc: "", confirm: "" };
    }
  })();

  return (
    <>
      <SettingsSection
        title={t("settings.devices.sessions_title")}
        description={t("settings.devices.sessions_hint")}
        action={
          <>
            {others > 0 && (
              <Button type="button" size="sm" variant="outline" onClick={() => setPending({ kind: "others" })}>
                {t("settings.devices.close_others")}
              </Button>
            )}
            <Button type="button" size="sm" variant="ghost" className="text-destructive hover:text-destructive" onClick={() => setPending({ kind: "all" })}>
              {t("settings.devices.close_all")}
            </Button>
          </>
        }
      >
        <QueryState isLoading={sessions.isLoading} isError={sessions.isError} error={sessions.error} onRetry={() => void sessions.refetch()}>
          {list.length === 0 ? (
            <p className="text-[13px] text-muted-foreground">{t("settings.devices.sessions_empty")}</p>
          ) : (
            <ul className="divide-y divide-border rounded-xl border border-border">
              {list.map((s) => (
                <li key={s.id} className="flex flex-wrap items-center gap-3 px-4 py-3">
                  <i className={cn(deviceIcon(s.device), "text-xl text-muted-foreground")} />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="truncate text-[13.5px] font-medium">{s.device || t("settings.devices.unknown")}</span>
                      {s.is_current && (
                        <Badge variant="secondary" className="text-emerald-600 dark:text-emerald-400">
                          {t("settings.devices.current")}
                        </Badge>
                      )}
                    </div>
                    <div className="mt-0.5 flex flex-wrap gap-x-3 gap-y-0.5 text-[12px] text-muted-foreground">
                      <span>IP {s.ip_address || "—"}</span>
                      <span>{t("settings.devices.signed_in", { date: formatDateTime(s.created_at) })}</span>
                      <span>{t("settings.devices.active", { date: formatDateTime(s.last_activity) })}</span>
                    </div>
                  </div>
                  {!s.is_current && (
                    <Button type="button" size="sm" variant="outline" onClick={() => setPending({ kind: "one", session: s })}>
                      {t("settings.devices.close")}
                    </Button>
                  )}
                </li>
              ))}
            </ul>
          )}
        </QueryState>
      </SettingsSection>

      <SettingsSection
        title={t("settings.devices.history_title")}
        description={t("settings.devices.history_hint")}
        action={
          <>
            <Button type="button" size="sm" variant="outline" onClick={() => setPending({ kind: "notme" })}>
              {t("settings.devices.notme")}
            </Button>
            <Button asChild type="button" size="sm" variant="ghost">
              <Link href="/activity">{t("settings.devices.activity")}</Link>
            </Button>
          </>
        }
      >
        <QueryState isLoading={logins.isLoading} isError={logins.isError} error={logins.error} onRetry={() => void logins.refetch()} rows={3}>
          {loginRows.length === 0 ? (
            <p className="text-[13px] text-muted-foreground">{t("settings.devices.history_empty")}</p>
          ) : (
            <>
              <ul className="divide-y divide-border rounded-xl border border-border">
                {loginRows.map((row) => (
                  <li key={row.id} className="flex items-start gap-3 px-4 py-3">
                    <i
                      className={cn(
                        row.success ? "ri-login-circle-line text-emerald-500" : "ri-error-warning-line text-rose-500",
                        "mt-0.5 text-lg"
                      )}
                    />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-x-2 text-[13.5px]">
                        <span className="font-medium">
                          {row.success ? t("settings.devices.login_ok") : t("settings.devices.login_failed")}
                        </span>
                        <span className="text-muted-foreground">{row.device || t("settings.devices.unknown")}</span>
                      </div>
                      <div className="mt-0.5 flex flex-wrap gap-x-3 text-[12px] text-muted-foreground">
                        <span>{formatDateTime(row.created_at)}</span>
                        <span>IP {row.ip || "—"}</span>
                        {row.reason && <span>{row.reason}</span>}
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
              {logins.hasNextPage && (
                <div className="mt-3 flex justify-center">
                  <Button type="button" variant="outline" size="sm" disabled={logins.isFetchingNextPage} onClick={() => void logins.fetchNextPage()}>
                    {t("settings.devices.more")}
                  </Button>
                </div>
              )}
            </>
          )}
        </QueryState>
      </SettingsSection>

      <ConfirmDialog
        open={pending !== null}
        onOpenChange={(open) => !open && setPending(null)}
        title={confirmCopy.title}
        desc={confirmCopy.desc}
        destructive={pending?.kind !== "one"}
        cancelBtnText={t("common.cancel")}
        confirmText={confirmCopy.confirm}
        isLoading={destroy.isPending}
        handleConfirm={() => pending && destroy.mutate(pending)}
      />
    </>
  );
}
