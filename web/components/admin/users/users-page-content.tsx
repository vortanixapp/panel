"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Ban, Eye, Pencil, Plus, Trash2, Users } from "lucide-react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import {
  deletePanelUser,
  togglePanelUserBlock,
  verifyPanelUserEmail,
  type PanelUser,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { isOwnerRole, isStaffRole, roleLabel } from "@/lib/rbac";
import { useAdminUsers } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

const COLUMNS =
  "grid-cols-[100px_minmax(200px,1.3fr)_minmax(200px,1.4fr)_110px_140px_130px_120px_150px]";

function formatDate(value: string) {
  try {
    return new Intl.DateTimeFormat(localeTag(), { dateStyle: "medium" }).format(
      new Date(value)
    );
  } catch {
    return value;
  }
}

export function userDisplayName(user: {
  name?: string | null;
  email: string;
}) {
  if (user.name?.trim()) return user.name.trim();
  const local = user.email.split("@")[0];
  return local ? local.charAt(0).toUpperCase() + local.slice(1) : user.email;
}

export function userInitials(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "??";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[1][0]).toUpperCase();
}

function isBlocked(user: PanelUser) {
  return user.is_blocked ?? user.status === "disabled";
}

export function UsersPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [roleFilter, setRoleFilter] = useState("");
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(searchInput);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchInput]);

  const usersQuery = useAdminUsers(page, search, roleFilter);

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ["admin-users"] });
    void qc.invalidateQueries({ queryKey: queryKeys.users });
  };

  const userAction = useMutation({
    mutationFn: async ({
      userId,
      action,
    }: {
      userId: string;
      action: "toggle-block" | "delete" | "verify-email";
    }) => {
      if (action === "delete") return deletePanelUser(userId);
      if (action === "verify-email") return verifyPanelUserEmail(userId);
      return togglePanelUserBlock(userId);
    },
    onSuccess: (_data, vars) => {
      if (vars.action === "delete") toast.success(t("admin.users.deleted"));
      else if (vars.action === "verify-email")
        toast.success(t("admin.users.email_verified"));
      else toast.success(t("admin.users.status_updated"));
      invalidate();
    },
    onError: (err) => {
      toast.error(
        err instanceof Error ? err.message : t("admin.users.action_failed")
      );
    },
    onSettled: () => setActionLoading(null),
  });

  const handleAction = (
    user: PanelUser,
    action: "toggle-block" | "delete" | "verify-email",
    confirmMsg?: string
  ) => {
    if (isOwnerRole(user.role)) return;
    if (confirmMsg && !window.confirm(confirmMsg)) return;
    setActionLoading(user.id);
    userAction.mutate({ userId: user.id, action });
  };

  const users = usersQuery.data?.users ?? [];
  const lastPage = usersQuery.data?.lastPage ?? 1;
  const counts = usersQuery.data?.counts;
  const perPage = 15;
  const from = users.length === 0 ? 0 : (page - 1) * perPage + 1;
  const to = users.length === 0 ? 0 : from + users.length - 1;
  const total = counts?.total ?? users.length;

  if (usersQuery.isLoading) {
    return (
      <PageShell variant="admin">
        <div className="w-full space-y-5">
          <Skeleton className="h-16 w-full rounded-2xl" />
          <Skeleton className="h-20 w-full rounded-2xl" />
          <Skeleton className="h-96 w-full rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {t("common.users")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("admin.users.subtitle")}
              {counts ? t("admin.users.total_suffix", { count: counts.total }) : ""}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2.5">
            <Input
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              placeholder={t("admin.users.search_placeholder")}
              className="h-[38px] w-[280px] rounded-lg text-[13px] md:text-[13px]"
            />
            <select
              value={roleFilter}
              onChange={(e) => {
                setRoleFilter(e.target.value);
                setPage(1);
              }}
              className="h-[38px] rounded-lg border border-input bg-transparent px-2.5 text-[13px] outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
            >
              <option value="">{t("admin.users.all_roles")}</option>
              <option value="user">{t("admin.users.role.user")}</option>
              <option value="support">{t("admin.users.role.support")}</option>
              <option value="admin">{t("admin.users.role.admin_short")}</option>
            </select>
            <Button asChild className="h-[38px] text-[13px]">
              <Link href="/admin/users/create">
                <Plus className="size-4" />
                {t("common.create")}
              </Link>
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap gap-2.5">
          <StatCard label={t("common.total")} value={counts?.total} />
          <StatCard
            label={t("admin.users.stat_active")}
            value={counts?.active}
            tone="emerald"
          />
          <StatCard
            label={t("admin.users.stat_unverified")}
            value={counts?.unverified}
            tone="amber"
          />
          <StatCard
            label={t("admin.users.stat_blocked")}
            value={counts?.blocked}
            tone="rose"
          />
        </div>

        {usersQuery.isError && (
          <p className="text-sm text-destructive">
            {t("admin.users.list_load_failed")}
          </p>
        )}

        <div className="overflow-hidden rounded-2xl border bg-card">
          {users.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <Users className="mb-3 size-10 text-muted-foreground" />
              <p className="text-sm text-muted-foreground">
                {t("admin.users.not_found")}
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <div className="flex min-w-[1180px] flex-col">
                <div
                  className={cn(
                    "grid gap-3 border-b bg-muted/40 px-5 py-3 font-mono text-[11px] tracking-wider text-muted-foreground uppercase",
                    COLUMNS
                  )}
                >
                  <div>{t("common.id")}</div>
                  <div>{t("common.user")}</div>
                  <div>{t("common.email")}</div>
                  <div>{t("admin.users.col_balance")}</div>
                  <div>{t("common.status")}</div>
                  <div>{t("admin.users.role")}</div>
                  <div>{t("common.date")}</div>
                  <div className="text-right">{t("common.actions")}</div>
                </div>

                {users.map((u) => {
                  const blocked = isBlocked(u);
                  const verified = Boolean(u.email_verified_at);
                  const busy = actionLoading === u.id && userAction.isPending;
                  const owner = isOwnerRole(u.role);
                  const name = userDisplayName(u);

                  return (
                    <div
                      key={u.id}
                      className={cn(
                        "grid items-center gap-3 border-b px-5 py-3.5 transition-colors hover:bg-muted/30",
                        COLUMNS
                      )}
                    >
                      <div className="font-mono text-xs text-muted-foreground">
                        #{u.id.slice(0, 8)}
                      </div>
                      <div className="flex min-w-0 items-center gap-3">
                        <span className="flex size-8 shrink-0 items-center justify-center rounded-lg border bg-muted font-mono text-xs">
                          {userInitials(name)}
                        </span>
                        <span className="truncate text-sm font-medium">
                          {name}
                        </span>
                      </div>
                      <div className="truncate font-mono text-xs text-muted-foreground">
                        {u.email}
                      </div>
                      <div className="font-mono text-[13px]">
                        {(u.balance ?? 0).toLocaleString(localeTag(), {
                          minimumFractionDigits: 2,
                          maximumFractionDigits: 2,
                        })}{" "}
                        ₽
                      </div>
                      <div>
                        <StatusDot
                          tone={
                            blocked ? "rose" : verified ? "emerald" : "amber"
                          }
                          label={
                            blocked
                              ? t("admin.users.status.blocked")
                              : verified
                                ? t("admin.users.status.active")
                                : t("admin.users.status.unverified")
                          }
                        />
                      </div>
                      <div>
                        <span
                          className={cn(
                            "inline-block rounded-md border px-2 py-0.5 text-[11px]",
                            isStaffRole(u.role)
                              ? "border-foreground/30"
                              : "text-muted-foreground"
                          )}
                        >
                          {roleLabel(u.role)}
                        </span>
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {formatDate(u.created_at)}
                      </div>
                      <div className="flex items-center justify-end gap-1.5">
                        <Button
                          variant="outline"
                          size="icon"
                          className="size-7 rounded-md"
                          asChild
                        >
                          <Link
                            href={`/admin/users/${u.id}`}
                            title={t("admin.locations.view")}
                          >
                            <Eye className="size-3.5" />
                          </Link>
                        </Button>
                        <Button
                          variant="outline"
                          size="icon"
                          className="size-7 rounded-md"
                          asChild
                        >
                          <Link
                            href={`/admin/users/${u.id}/edit`}
                            title={t("common.edit")}
                          >
                            <Pencil className="size-3.5" />
                          </Link>
                        </Button>
                        {!owner && (
                          <>
                            <Button
                              variant="outline"
                              size="icon"
                              className="size-7 rounded-md text-amber-500 hover:text-amber-500"
                              title={
                                blocked
                                  ? t("admin.users.unblock")
                                  : t("admin.users.block")
                              }
                              disabled={busy}
                              onClick={() =>
                                handleAction(
                                  u,
                                  "toggle-block",
                                  blocked
                                    ? t("admin.users.unblock_confirm")
                                    : t("admin.users.block_confirm")
                                )
                              }
                            >
                              <Ban className="size-3.5" />
                            </Button>
                            <Button
                              variant="outline"
                              size="icon"
                              className="size-7 rounded-md text-destructive hover:text-destructive"
                              title={t("common.delete")}
                              disabled={busy}
                              onClick={() =>
                                handleAction(
                                  u,
                                  "delete",
                                  t("admin.users.delete_confirm", {
                                    email: u.email,
                                  })
                                )
                              }
                            >
                              <Trash2 className="size-3.5" />
                            </Button>
                          </>
                        )}
                      </div>
                    </div>
                  );
                })}

                <div className="flex items-center justify-between gap-4 bg-muted/40 px-5 py-3.5">
                  <span className="text-xs text-muted-foreground">
                    {t("admin.tariffs.range", { from, to, total })}
                  </span>
                  <div className="flex items-center gap-1.5">
                    {lastPage > 1 && (
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-[30px] rounded-md text-xs"
                        disabled={page <= 1}
                        onClick={() => setPage((p) => Math.max(1, p - 1))}
                      >
                        {t("common.back")}
                      </Button>
                    )}
                    {Array.from({ length: lastPage }, (_, i) => i + 1)
                      .filter(
                        (p) =>
                          p === 1 || p === lastPage || Math.abs(p - page) <= 1
                      )
                      .map((p, idx, arr) => (
                        <span key={p} className="flex items-center gap-1.5">
                          {idx > 0 && arr[idx - 1] !== p - 1 && (
                            <span className="px-0.5 font-mono text-xs text-muted-foreground">
                              …
                            </span>
                          )}
                          <Button
                            variant={p === page ? "default" : "outline"}
                            size="sm"
                            className="h-[30px] min-w-[30px] rounded-md px-2 font-mono text-xs"
                            onClick={() => setPage(p)}
                          >
                            {p}
                          </Button>
                        </span>
                      ))}
                    {lastPage > 1 && (
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-[30px] rounded-md text-xs"
                        disabled={page >= lastPage}
                        onClick={() =>
                          setPage((p) => Math.min(lastPage, p + 1))
                        }
                      >
                        {t("common.next")}
                      </Button>
                    )}
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </PageShell>
  );
}

function StatCard({
  label,
  value,
  tone,
}: {
  label: string;
  value?: number;
  tone?: "emerald" | "amber" | "rose";
}) {
  return (
    <div className="flex min-w-[160px] flex-1 flex-col gap-1.5 rounded-xl border bg-card px-4 py-3.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span
        className={cn(
          "font-mono text-xl",
          tone === "emerald" && "text-emerald-500",
          tone === "amber" && "text-amber-500",
          tone === "rose" && "text-rose-500"
        )}
      >
        {value ?? "—"}
      </span>
    </div>
  );
}

function StatusDot({
  tone,
  label,
}: {
  tone: "emerald" | "amber" | "rose";
  label: string;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-2 text-xs",
        tone === "emerald" && "text-emerald-500",
        tone === "amber" && "text-amber-500",
        tone === "rose" && "text-rose-500"
      )}
    >
      <span
        className={cn(
          "size-1.5 rounded-full",
          tone === "emerald" && "bg-emerald-500",
          tone === "amber" && "bg-amber-500",
          tone === "rose" && "bg-rose-500"
        )}
      />
      {label}
    </span>
  );
}
