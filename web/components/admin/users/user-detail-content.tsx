"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { getAccessToken, setTokens } from "@/lib/api";
import { decodeAccess, markEnteredVia } from "@/lib/accounts";
import { isOwnerRole, roleLabel } from "@/lib/rbac";
import {
  useAdminUser,
  useCreateAdminUserWallet,
  useImpersonateAdminUser,
  useToggleAdminUserBlock,
} from "@/hooks/use-queries";
import { userDisplayName, userInitials } from "./users-page-content";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

const CURRENCIES = ["RUB", "USD", "EUR"];

const CURRENCY_SIGN: Record<string, string> = {
  RUB: "₽",
  USD: "$",
  EUR: "€",
};

function numberFormat(value: unknown): string {
  const num = Number(value);
  if (!Number.isFinite(num)) return "0,00";
  return new Intl.NumberFormat(localeTag(), {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(num);
}

function money(value: unknown, currency: string) {
  const cur = currency.toUpperCase();
  return `${numberFormat(value)} ${CURRENCY_SIGN[cur] ?? cur}`;
}

function formatDateTime(value?: string | null) {
  if (!value) return "—";
  try {
    return new Date(value)
      .toLocaleString(localeTag(), {
        day: "2-digit",
        month: "2-digit",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      })
      .replace(",", "");
  } catch {
    return value;
  }
}

export function UserDetailContent() {
  const t = useT();
  const params = useParams();
  const id = String(params?.id ?? "");
  const [newWalletCurrency, setNewWalletCurrency] = useState("RUB");

  const userQuery = useAdminUser(id);
  const toggleBlock = useToggleAdminUserBlock(id);
  const createWallet = useCreateAdminUserWallet(id);
  const impersonate = useImpersonateAdminUser();

  const data = userQuery.data;
  const user = data?.user;
  const servers = data?.servers ?? [];
  const wallets = data?.wallets ?? [];
  const defaultWallet = data?.defaultWallet ?? null;
  const transactions = data?.transactions ?? [];
  const sessions = data?.sessions ?? [];

  const existingCurrencies = wallets.map((w) => w.currency.toUpperCase());
  const availableCurrencies = CURRENCIES.filter(
    (c) => !existingCurrencies.includes(c)
  );

  const onToggleBlock = async () => {
    if (!user) return;
    if (
      !window.confirm(
        user.is_blocked
          ? t("admin.users.unblock_user_confirm", { email: user.email })
          : t("admin.users.block_user_confirm", { email: user.email })
      )
    )
      return;
    try {
      await toggleBlock.mutateAsync();
      toast.success(t("admin.locations.status_updated"));
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("common.error"));
    }
  };

  const onImpersonate = async () => {
    if (!user) return;
    if (
      !window.confirm(
        t("admin.users.impersonate_confirm", { email: user.email })
      )
    )
      return;
    try {
      const res = await impersonate.mutateAsync(id);
      const token = res.token ?? res.access_token;
      if (!res.ok && !token) {
        toast.error(res.message ?? t("admin.users.impersonate_failed"));
        return;
      }
      if (!token) {
        toast.error(t("admin.users.impersonate_failed"));
        return;
      }
      // Кто входит — запоминаем до подмены токенов: после неё активным будет
      // уже пользователь. Раньше здесь копились ключи vortanix_impersonator_*,
      // которые не читала ни одна строка кода: вернуться к себе было нельзя,
      // администратор выходил и входил заново.
      const admin = decodeAccess(getAccessToken() ?? "");
      setTokens(token, res.refresh_token ?? "");
      const entered = decodeAccess(token);
      if (admin && entered) {
        markEnteredVia(entered.user_id, { id: admin.user_id, email: admin.email });
      }
      window.location.href = "/dashboard";
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : t("admin.users.impersonate_failed")
      );
    }
  };

  const onCreateWallet = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createWallet.mutateAsync(newWalletCurrency);
      toast.success(t("admin.users.wallet_created"));
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : t("admin.users.wallet_failed")
      );
    }
  };

  if (userQuery.isLoading || !user) {
    return (
      <PageShell variant="admin">
        <div className="w-full space-y-4">
          <Skeleton className="h-16 w-full rounded-2xl" />
          <Skeleton className="h-24 w-full rounded-2xl" />
          <Skeleton className="h-96 w-full rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  const role = user.role || "user";
  const canImpersonate = role === "user";
  const verified = Boolean(user.email_verified_at);
  const blocked = Boolean(user.is_blocked);
  const displayName = userDisplayName(user);

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-6">
          <div className="flex items-center gap-4">
            <span className="flex size-12 shrink-0 items-center justify-center rounded-2xl border bg-muted font-mono text-[15px]">
              {userInitials(displayName)}
            </span>
            <div className="space-y-1.5">
              <div className="flex items-center gap-2.5">
                <span className="rounded-md border px-2 py-0.5 text-[11px]">
                  {roleLabel(role)}
                </span>
                <span className="font-mono text-xs text-muted-foreground">
                  #{id.slice(0, 8)}
                </span>
                {blocked && (
                  <span className="rounded-md border border-rose-500/40 px-2 py-0.5 text-[11px] text-rose-500">
                    {t("admin.users.status.blocked")}
                  </span>
                )}
              </div>
              <h1 className="text-2xl leading-none font-bold tracking-tight">
                {displayName}
              </h1>
              <p className="font-mono text-[13px] text-muted-foreground">
                {user.email}
              </p>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2.5">
            <Button variant="outline" asChild className="h-[38px] text-[13px]">
              <Link href="/admin/users">← {t("common.back")}</Link>
            </Button>
            <Button
              variant="outline"
              className="h-[38px] text-[13px]"
              onClick={onImpersonate}
              disabled={!canImpersonate || impersonate.isPending}
              title={
                canImpersonate
                  ? undefined
                  : isOwnerRole(role)
                    ? t("admin.users.impersonate_owner")
                    : t("admin.users.impersonate_only_users")
              }
            >
              {t("admin.users.impersonate")}
            </Button>
            {!isOwnerRole(role) && (
              <Button
                variant="outline"
                className={cn(
                  "h-[38px] text-[13px]",
                  blocked ? "text-emerald-500" : "text-amber-500"
                )}
                onClick={onToggleBlock}
                disabled={toggleBlock.isPending}
              >
                {blocked
                  ? t("admin.users.unblock")
                  : t("admin.users.block")}
              </Button>
            )}
            <Button asChild className="h-[38px] text-[13px]">
              <Link href={`/admin/users/${id}/edit`}>{t("common.edit")}</Link>
            </Button>
          </div>
        </div>

        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard label={t("admin.users.group")}>
            <span className="text-[15px] font-medium">{roleLabel(role)}</span>
          </StatCard>
          <StatCard label={t("common.email")}>
            <span
              className={cn(
                "inline-flex items-center gap-2 text-sm",
                verified ? "text-emerald-500" : "text-amber-500"
              )}
            >
              <span
                className={cn(
                  "size-1.5 rounded-full",
                  verified ? "bg-emerald-500" : "bg-amber-500"
                )}
              />
              {verified
                ? t("admin.users.verified")
                : t("admin.users.status.unverified")}
            </span>
          </StatCard>
          <StatCard label={t("admin.users.col_balance")}>
            <span className="font-mono text-lg">
              {defaultWallet
                ? money(defaultWallet.balance, defaultWallet.currency)
                : "—"}
            </span>
          </StatCard>
          <StatCard label={t("admin.users.registered")}>
            <span className="font-mono text-sm">
              {formatDateTime(user.created_at)}
            </span>
          </StatCard>
        </div>

        <section className="rounded-2xl border bg-card px-5 py-5 sm:px-6">
          <div className="mb-5 text-[15px] font-semibold">
            {t("admin.users.info_title")}
          </div>
          <div className="grid gap-x-6 gap-y-5 sm:grid-cols-2 lg:grid-cols-4">
            {[
              { label: t("admin.users.name"), value: user.name },
              { label: t("admin.users.last_name"), value: user.last_name },
              { label: t("admin.users.login"), value: user.public_id },
              { label: t("common.email"), value: user.email },
              { label: t("admin.users.phone"), value: user.phone },
              { label: "Telegram", value: user.telegram_id },
              { label: "Discord", value: user.discord_id },
              { label: "VK", value: user.vk_id },
            ].map((item) => (
              <div key={item.label} className="flex flex-col gap-1.5">
                <span className="text-xs text-muted-foreground">
                  {item.label}
                </span>
                <span
                  className={cn(
                    "truncate font-mono text-[13px]",
                    !item.value && "text-muted-foreground/50"
                  )}
                >
                  {item.value || "—"}
                </span>
              </div>
            ))}
          </div>
        </section>

        <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
          <TableCard
            title={t("admin.users.wallets")}
            action={
              availableCurrencies.length > 0 ? (
                <form onSubmit={onCreateWallet} className="flex items-center gap-2">
                  <select
                    value={newWalletCurrency}
                    onChange={(e) => setNewWalletCurrency(e.target.value)}
                    className="h-8 rounded-md border border-input bg-transparent px-2 font-mono text-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
                  >
                    {availableCurrencies.map((c) => (
                      <option key={c} value={c}>
                        {c}
                      </option>
                    ))}
                  </select>
                  <Button
                    type="submit"
                    variant="outline"
                    size="sm"
                    className="h-8 rounded-md text-xs"
                    disabled={createWallet.isPending}
                  >
                    {createWallet.isPending ? "…" : t("common.create")}
                  </Button>
                </form>
              ) : null
            }
            columns="grid-cols-[90px_1fr_120px]"
            head={[
              t("admin.users.col_currency"),
              t("admin.users.col_balance"),
              t("admin.users.col_default"),
            ]}
            empty={
              wallets.length === 0 ? t("admin.users.wallets_empty") : undefined
            }
          >
            {wallets.map((w) => (
              <Row key={w.id} columns="grid-cols-[90px_1fr_120px]">
                <div className="font-mono text-[13px] font-medium">
                  {w.currency.toUpperCase()}
                </div>
                <div className="font-mono text-[13px] text-muted-foreground">
                  {money(w.balance, w.currency)}
                </div>
                <div
                  className={cn(
                    "text-xs",
                    w.is_default
                      ? "text-emerald-500"
                      : "text-muted-foreground/50"
                  )}
                >
                  {w.is_default ? t("common.yes") : "—"}
                </div>
              </Row>
            ))}
          </TableCard>

          <TableCard
            title={t("admin.users.servers_title")}
            action={
              <span className="font-mono text-xs text-muted-foreground">
                {t("admin.users.servers_count", { count: servers.length })}
              </span>
            }
            columns="grid-cols-[100px_minmax(0,1fr)_minmax(0,1.2fr)_190px]"
            head={[
              t("common.id"),
              t("common.name"),
              t("common.game"),
              "IP",
            ]}
            empty={
              servers.length === 0 ? t("admin.users.servers_empty") : undefined
            }
          >
            {servers.map((srv) => (
              <Row
                key={srv.id}
                columns="grid-cols-[100px_minmax(0,1fr)_minmax(0,1.2fr)_190px]"
              >
                <div className="font-mono text-xs text-muted-foreground">
                  #{srv.id.slice(0, 8)}
                </div>
                <div className="truncate text-[13px] font-medium">
                  {srv.name || "—"}
                </div>
                <div className="truncate text-[13px] text-muted-foreground">
                  {srv.game?.name || "—"}
                </div>
                <div className="font-mono text-xs text-muted-foreground">
                  {srv.ip_address ? `${srv.ip_address}:${srv.port}` : "—"}
                </div>
              </Row>
            ))}
          </TableCard>
        </div>

        <TableCard
          title={t("admin.users.transactions_title")}
          action={
            <Button
              variant="outline"
              size="sm"
              asChild
              className="h-[30px] rounded-md text-xs"
            >
              <Link href="/admin/billing">
                {t("admin.users.all_transactions")}
              </Link>
            </Button>
          }
          columns="grid-cols-[150px_90px_80px_120px_minmax(0,1fr)]"
          head={[
            t("common.date"),
            t("common.type"),
            t("admin.users.col_currency"),
            t("common.amount"),
            t("common.description"),
          ]}
          empty={
            transactions.length === 0
              ? t("admin.users.transactions_empty")
              : undefined
          }
        >
          {/* Параметр назван tx, а не t: имя t занято функцией перевода. */}
          {transactions.map((tx) => {
            const credit = tx.type === "credit";
            return (
              <Row
                key={tx.id}
                columns="grid-cols-[150px_90px_80px_120px_minmax(0,1fr)]"
              >
                <div className="font-mono text-xs text-muted-foreground">
                  {formatDateTime(tx.created_at)}
                </div>
                <div>
                  <span
                    className={cn(
                      "inline-block rounded-md border px-2 py-0.5 font-mono text-[10px] tracking-wider uppercase",
                      credit
                        ? "border-emerald-500/40 text-emerald-500"
                        : "border-rose-500/40 text-rose-500"
                    )}
                  >
                    {tx.type}
                  </span>
                </div>
                <div className="font-mono text-xs text-muted-foreground">
                  {(tx.wallet?.currency || "RUB").toUpperCase()}
                </div>
                <div
                  className={cn(
                    "font-mono text-[13px]",
                    credit ? "text-emerald-500" : "text-rose-500"
                  )}
                >
                  {credit ? "+" : "−"}
                  {numberFormat(Math.abs(Number(tx.amount) || 0))}
                </div>
                <div className="truncate text-[13px] text-muted-foreground">
                  {tx.description}
                </div>
              </Row>
            );
          })}
        </TableCard>

        <TableCard
          title={t("admin.users.sessions_title")}
          action={
            <span className="font-mono text-xs text-muted-foreground">
              {t("admin.users.sessions_count", { count: sessions.length })}
            </span>
          }
          columns="grid-cols-[130px_minmax(0,1fr)_160px]"
          head={["IP", "User-Agent", t("admin.users.col_activity")]}
          headAlign={["", "", "text-right"]}
          empty={
            sessions.length === 0 ? t("admin.users.sessions_empty") : undefined
          }
        >
          {sessions.map((s) => (
            <Row key={s.id} columns="grid-cols-[130px_minmax(0,1fr)_160px]">
              <div className="font-mono text-xs">{s.ip_address || "—"}</div>
              <div
                className="truncate font-mono text-xs text-muted-foreground"
                title={s.user_agent}
              >
                {s.user_agent || "—"}
              </div>
              <div className="text-right font-mono text-xs text-muted-foreground">
                {formatDateTime(s.last_activity)}
              </div>
            </Row>
          ))}
        </TableCard>
      </div>
    </PageShell>
  );
}

function StatCard({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5 rounded-xl border bg-card px-4 py-3.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      {children}
    </div>
  );
}

function TableCard({
  title,
  action,
  columns,
  head,
  headAlign,
  empty,
  children,
}: {
  title: string;
  action?: React.ReactNode;
  columns: string;
  head: string[];
  headAlign?: string[];
  empty?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="overflow-hidden rounded-2xl border bg-card">
      <div className="flex items-center justify-between gap-3.5 px-6 py-4">
        <span className="text-[15px] font-semibold">{title}</span>
        {action}
      </div>
      <div className="overflow-x-auto">
        <div className="min-w-[560px]">
          <div
            className={cn(
              "grid gap-3 border-t bg-muted/40 px-6 py-2.5 font-mono text-[11px] tracking-wider text-muted-foreground uppercase",
              columns
            )}
          >
            {head.map((h, i) => (
              <div key={h} className={headAlign?.[i]}>
                {h}
              </div>
            ))}
          </div>
          {empty ? (
            <div className="border-t px-6 py-8 text-center text-sm text-muted-foreground">
              {empty}
            </div>
          ) : (
            children
          )}
        </div>
      </div>
    </section>
  );
}

function Row({
  columns,
  children,
}: {
  columns: string;
  children: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "grid items-center gap-3 border-t px-6 py-3.5 transition-colors hover:bg-muted/30",
        columns
      )}
    >
      {children}
    </div>
  );
}
