"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchAdminHostingAccountsList,
  suspendAdminHostingAccount,
  unsuspendAdminHostingAccount,
  type AdminHostingAccount,
} from "@/lib/api";
import { localeTag } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";

function fmtDate(iso: string | null): string {
  if (!iso) return "—";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? "—" : d.toLocaleDateString(localeTag());
}

// Словарь читается на уровне модуля, поэтому храним ключи, а не готовый текст:
// перевод, посчитанный при загрузке страницы, застыл бы на прежнем языке.
const STATUS_LABEL: Record<string, string> = {
  pending: "admin.hosting_accounts.status.pending",
  active: "admin.hosting.active",
  suspended: "admin.hosting_accounts.status.suspended",
  terminated: "admin.hosting_accounts.status.terminated",
  error: "admin.hosting_accounts.status.error",
};

export function HostingAccountsPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const [search, setSearch] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["admin-hosting-accounts"],
    queryFn: async () => (await fetchAdminHostingAccountsList()).accounts,
  });
  const accounts = data ?? [];

  const invalidate = () =>
    qc.invalidateQueries({ queryKey: ["admin-hosting-accounts"] });

  const suspendMut = useMutation({
    mutationFn: (a: AdminHostingAccount) =>
      suspendAdminHostingAccount(a.id, "suspended by admin"),
    onSuccess: () => {
      toast.success(t("admin.hosting_accounts.suspended"));
      invalidate();
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.hosting_accounts.request_failed")),
  });

  const unsuspendMut = useMutation({
    mutationFn: (a: AdminHostingAccount) => unsuspendAdminHostingAccount(a.id),
    onSuccess: () => {
      toast.success(t("admin.hosting_accounts.unsuspended"));
      invalidate();
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.hosting_accounts.request_failed")),
  });

  const visible = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return accounts;
    return accounts.filter((a) =>
      [a.username, a.user_email, a.primary_domain, a.plan, a.server].some((v) =>
        (v || "").toLowerCase().includes(q)
      )
    );
  }, [accounts, search]);

  const busy = suspendMut.isPending || unsuspendMut.isPending;

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("admin.hosting_accounts.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.hosting_accounts.subtitle")}
          </p>
        </div>
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t("admin.hosting_accounts.search_placeholder")}
          className="w-full sm:w-72"
        />
      </div>

      <div className="rounded-lg border bg-card">
        {isLoading ? (
          <div className="space-y-3 p-6">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : visible.length === 0 ? (
          <div className="p-10 text-center text-sm text-muted-foreground">
            {accounts.length === 0
              ? t("admin.hosting_accounts.empty")
              : t("common.not_found")}
          </div>
        ) : (
          <div className="divide-y">
            {visible.map((a) => {
              const expired =
                a.expires_at !== null && new Date(a.expires_at) < new Date();
              return (
                <div
                  key={a.id}
                  className="flex flex-wrap items-center justify-between gap-4 p-4"
                >
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium">{a.username}</span>
                      {a.primary_domain && (
                        <span className="font-mono text-xs text-muted-foreground">
                          {a.primary_domain}
                        </span>
                      )}
                    </div>
                    <div className="mt-1 text-xs text-muted-foreground">
                      {a.user_email || t("admin.hosting_accounts.no_user")} ·{" "}
                      {a.plan || t("admin.hosting_accounts.no_plan")} ·{" "}
                      {a.server || t("admin.hosting_accounts.no_server")} ·{" "}
                      {t("admin.hosting_accounts.until", {
                        date: fmtDate(a.expires_at),
                      })}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {a.status === "active" ? (
                      <Badge className="bg-emerald-500/10 text-emerald-600 ring-1 ring-inset ring-emerald-500/20">
                        {t(STATUS_LABEL.active)}
                      </Badge>
                    ) : (
                      <Badge variant="outline">
                        {STATUS_LABEL[a.status]
                          ? t(STATUS_LABEL[a.status])
                          : a.status}
                      </Badge>
                    )}
                    {expired && a.status === "active" && (
                      <Badge variant="outline">
                        {t("admin.hosting_accounts.expired")}
                      </Badge>
                    )}
                    {a.status === "suspended" ? (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => unsuspendMut.mutate(a)}
                        disabled={busy}
                      >
                        {t("admin.hosting_accounts.unsuspend")}
                      </Button>
                    ) : (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          if (
                            !confirm(
                              t("admin.hosting_accounts.suspend_confirm", {
                                name: a.username,
                              })
                            )
                          )
                            return;
                          suspendMut.mutate(a);
                        }}
                        disabled={busy || a.status !== "active"}
                      >
                        {t("admin.hosting_accounts.suspend")}
                      </Button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </PageShell>
  );
}
