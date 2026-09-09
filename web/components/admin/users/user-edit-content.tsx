"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { togglePanelUserBlock, verifyPanelUserEmail } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { isOwnerRole } from "@/lib/rbac";
import { useAdminUser, useUpdateAdminUser } from "@/hooks/use-queries";
import { userDisplayName, userInitials } from "./users-page-content";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

const ROLES = [
  { id: "user", labelKey: "admin.users.role.user" },
  { id: "support", labelKey: "admin.users.role.support_full" },
  { id: "admin", labelKey: "admin.users.role.admin_short" },
];

const CURRENCIES = ["RUB", "USD", "EUR"] as const;

type WalletBalances = Record<string, number>;

type FormState = {
  name: string;
  last_name: string;
  public_id: string;
  email: string;
  phone: string;
  telegram_id: string;
  discord_id: string;
  vk_id: string;
  password: string;
  password_confirmation: string;
  role: string;
};

const emptyForm: FormState = {
  name: "",
  last_name: "",
  public_id: "",
  email: "",
  phone: "",
  telegram_id: "",
  discord_id: "",
  vk_id: "",
  password: "",
  password_confirmation: "",
  role: "user",
};

function formatDate(value?: string | null, withTime = false) {
  if (!value) return "—";
  try {
    return new Intl.DateTimeFormat(localeTag(), {
      dateStyle: "medium",
      ...(withTime ? { timeStyle: "short" } : {}),
    }).format(new Date(value));
  } catch {
    return value;
  }
}

export function UserEditContent() {
  const t = useT();
  const params = useParams();
  const id = String(params?.id ?? "");
  const router = useRouter();
  const qc = useQueryClient();
  const userQuery = useAdminUser(id);
  const saveMutation = useUpdateAdminUser(id);

  const [error, setError] = useState("");
  const [walletBalances, setWalletBalances] = useState<WalletBalances>({});
  const [f, setF] = useState<FormState>(emptyForm);
  const [baseline, setBaseline] = useState<{
    form: FormState;
    wallets: WalletBalances;
  }>({ form: emptyForm, wallets: {} });

  const set = <K extends keyof FormState>(k: K, v: FormState[K]) =>
    setF((p) => ({ ...p, [k]: v }));

  useEffect(() => {
    if (!userQuery.data) return;
    const u = userQuery.data.user;
    const wallets = userQuery.data.wallets ?? [];
    const nextForm: FormState = {
      name: u.name || "",
      last_name: u.last_name || "",
      public_id: u.public_id || "",
      email: u.email || "",
      phone: u.phone || "",
      telegram_id: u.telegram_id || "",
      discord_id: u.discord_id || "",
      vk_id: u.vk_id || "",
      password: "",
      password_confirmation: "",
      role: u.role || "user",
    };
    const balances: WalletBalances = {};
    wallets.forEach((wl) => {
      const cur = String(wl.currency || "").toUpperCase();
      if ((CURRENCIES as readonly string[]).includes(cur)) {
        balances[cur] = wl.balance ?? 0;
      }
    });
    setF(nextForm);
    setWalletBalances(balances);
    setBaseline({ form: nextForm, wallets: balances });
  }, [userQuery.data]);

  const dirty = useMemo(
    () =>
      JSON.stringify(f) !== JSON.stringify(baseline.form) ||
      JSON.stringify(walletBalances) !== JSON.stringify(baseline.wallets),
    [f, walletBalances, baseline]
  );

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: queryKeys.adminUser(id) });
    void qc.invalidateQueries({ queryKey: ["admin-users"] });
  };

  const blockMutation = useMutation({
    mutationFn: () => togglePanelUserBlock(id),
    onSuccess: () => {
      toast.success(t("admin.users.status_updated"));
      invalidate();
    },
    onError: (err: Error) =>
      toast.error(err.message || t("admin.psp.status_failed")),
  });

  const verifyMutation = useMutation({
    mutationFn: () => verifyPanelUserEmail(id),
    onSuccess: () => {
      toast.success(t("admin.users.email_verified"));
      invalidate();
    },
    onError: (err: Error) =>
      toast.error(err.message || t("admin.users.verify_failed")),
  });

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      const payload: Record<string, unknown> = {
        name: f.name,
        last_name: f.last_name,
        public_id: f.public_id,
        email: f.email,
        phone: f.phone,
        telegram_id: f.telegram_id,
        discord_id: f.discord_id,
        vk_id: f.vk_id,
        role: f.role,
        wallets: walletBalances,
      };
      if (f.password) {
        payload.password = f.password;
        payload.password_confirmation = f.password_confirmation;
      }
      await saveMutation.mutateAsync(payload);
      const savedForm = { ...f, password: "", password_confirmation: "" };
      setF(savedForm);
      setBaseline({ form: savedForm, wallets: walletBalances });
      toast.success(t("common.saved"));
    } catch (err) {
      const msg = err instanceof Error ? err.message : t("common.error");
      setError(msg);
      toast.error(msg);
    }
  };

  if (userQuery.isLoading && !userQuery.data) {
    return (
      <PageShell variant="admin">
        <div className="w-full space-y-5">
          <Skeleton className="h-16 w-full rounded-2xl" />
          <Skeleton className="h-96 w-full rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  const user = userQuery.data?.user;
  const wallets = userQuery.data?.wallets ?? [];
  const sessions = userQuery.data?.sessions ?? [];
  const lastLogin = sessions
    .map((s) => s.last_activity)
    .filter(Boolean)
    .sort()
    .at(-1);
  const blocked = user?.is_blocked ?? user?.status === "disabled";
  const verified = Boolean(user?.email_verified_at);
  const owner = isOwnerRole(user?.role);
  const displayName = userDisplayName({
    name: f.name || user?.name,
    email: f.email || user?.email || "",
  });

  return (
    <PageShell variant="admin">
      <form onSubmit={submit} className="w-full space-y-5">
        <div className="flex flex-wrap items-center justify-between gap-6">
          <div className="flex items-center gap-4">
            <span className="flex size-12 shrink-0 items-center justify-center rounded-2xl border bg-muted font-mono text-[15px]">
              {userInitials(displayName)}
            </span>
            <div className="space-y-1">
              <div className="flex items-center gap-2.5 text-xs text-muted-foreground">
                <Link href="/admin/users" className="hover:text-foreground">
                  {t("common.users")}
                </Link>
                <span>/</span>
                <span className="font-mono">#{id.slice(0, 8)}</span>
              </div>
              <h1 className="text-2xl leading-none font-bold tracking-tight">
                {displayName}
              </h1>
              <p className="text-[13px] text-muted-foreground">
                {t("admin.users.edit_subtitle")}
              </p>
            </div>
          </div>
          <Button
            type="button"
            variant="outline"
            asChild
            className="h-[38px] text-[13px]"
          >
            <Link href={`/admin/users/${id}`}>← {t("common.back")}</Link>
          </Button>
        </div>

        {error && (
          <div className="rounded-xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
            {error}
          </div>
        )}

        <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1.55fr)_minmax(280px,0.85fr)]">
          <div className="flex flex-col gap-4">
            <Card title={t("admin.users.profile")}>
              <div className="grid gap-4 sm:grid-cols-2">
                <Field label={t("admin.users.name")}>
                  <Input
                    value={f.name}
                    onChange={(e) => set("name", e.target.value)}
                    required
                    className={inputClass}
                  />
                </Field>
                <Field label={t("admin.users.last_name")}>
                  <Input
                    value={f.last_name}
                    onChange={(e) => set("last_name", e.target.value)}
                    className={inputClass}
                  />
                </Field>
                <Field label={t("admin.users.login")}>
                  <Input
                    value={f.public_id}
                    onChange={(e) => set("public_id", e.target.value)}
                    placeholder="viktor"
                    className={cn(inputClass, "font-mono")}
                  />
                </Field>
                <Field label={t("common.email")}>
                  <Input
                    type="email"
                    value={f.email}
                    onChange={(e) => set("email", e.target.value)}
                    required
                    className={cn(inputClass, "font-mono")}
                  />
                </Field>
              </div>
            </Card>

            <Card
              title={t("admin.users.contacts")}
              description={t("admin.users.contacts_hint")}
            >
              <div className="grid gap-4 sm:grid-cols-2">
                <Field label={t("admin.users.phone")}>
                  <Input
                    value={f.phone}
                    onChange={(e) => set("phone", e.target.value)}
                    placeholder="+7 900 000-00-00"
                    className={cn(inputClass, "font-mono")}
                  />
                </Field>
                <Field label="Telegram">
                  <Input
                    value={f.telegram_id}
                    onChange={(e) => set("telegram_id", e.target.value)}
                    placeholder="@viktor"
                    className={cn(inputClass, "font-mono")}
                  />
                </Field>
                <Field label="Discord">
                  <Input
                    value={f.discord_id}
                    onChange={(e) => set("discord_id", e.target.value)}
                    placeholder="viktor#0001"
                    className={cn(inputClass, "font-mono")}
                  />
                </Field>
                <Field label="VK">
                  <Input
                    value={f.vk_id}
                    onChange={(e) => set("vk_id", e.target.value)}
                    placeholder="id12345"
                    className={cn(inputClass, "font-mono")}
                  />
                </Field>
              </div>
            </Card>

            <Card
              title={t("common.password")}
              description={t("admin.users.password_hint")}
            >
              <div className="grid gap-4 sm:grid-cols-2">
                <Field label={t("admin.users.new_password")}>
                  <Input
                    type="password"
                    value={f.password}
                    onChange={(e) => set("password", e.target.value)}
                    autoComplete="new-password"
                    className={inputClass}
                  />
                </Field>
                <Field label={t("admin.users.password_confirm")}>
                  <Input
                    type="password"
                    value={f.password_confirmation}
                    onChange={(e) =>
                      set("password_confirmation", e.target.value)
                    }
                    autoComplete="new-password"
                    className={inputClass}
                  />
                </Field>
              </div>
            </Card>

            {wallets.length > 0 && (
              <Card
                title={t("admin.users.wallets")}
                action={
                  <span className="text-xs text-muted-foreground">
                    {t("admin.users.wallets_hint")}
                  </span>
                }
              >
                <div className="grid gap-4 sm:grid-cols-3">
                  {CURRENCIES.filter(
                    (cur) => walletBalances[cur] !== undefined
                  ).map((cur) => (
                    <Field
                      key={cur}
                      label={t("admin.users.balance_of", { currency: cur })}
                    >
                      <Input
                        type="number"
                        step="0.01"
                        min={0}
                        value={walletBalances[cur]}
                        onChange={(e) =>
                          setWalletBalances((p) => ({
                            ...p,
                            [cur]: +e.target.value,
                          }))
                        }
                        className={cn(inputClass, "font-mono")}
                      />
                    </Field>
                  ))}
                </div>
              </Card>
            )}
          </div>

          <div className="flex flex-col gap-4">
            <Card title={t("admin.users.access")}>
              <Field label={t("admin.users.group")}>
                <div className="flex flex-col gap-1 rounded-xl border bg-muted/30 p-1">
                  {ROLES.map((role) => (
                    <button
                      key={role.id}
                      type="button"
                      onClick={() => set("role", role.id)}
                      disabled={owner}
                      className={cn(
                        "h-[34px] rounded-lg px-3 text-left text-[13px] transition-colors disabled:opacity-60",
                        f.role === role.id
                          ? "bg-background font-medium shadow-xs"
                          : "text-muted-foreground hover:text-foreground"
                      )}
                    >
                      {t(role.labelKey)}
                    </button>
                  ))}
                </div>
              </Field>
              {owner && (
                <p className="text-xs text-muted-foreground">
                  {t("admin.users.owner_locked")}
                </p>
              )}
              <div className="flex items-center justify-between gap-4 rounded-xl border bg-muted/30 px-4 py-3.5">
                <div className="space-y-0.5">
                  <div className="text-[13px] font-medium">
                    {t("admin.users.blocking")}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {t("admin.users.blocking_hint")}
                  </div>
                </div>
                <Switch
                  checked={!!blocked}
                  disabled={owner || blockMutation.isPending}
                  onCheckedChange={() => blockMutation.mutate()}
                  className="data-[state=checked]:bg-destructive"
                />
              </div>
            </Card>

            <Card title={t("admin.users.account_state")}>
              <InfoRow
                label={t("common.email")}
                value={
                  verified
                    ? t("admin.users.verified")
                    : t("admin.users.status.unverified")
                }
                tone={verified ? undefined : "amber"}
              />
              <InfoRow
                label={t("admin.users.registered")}
                value={formatDate(user?.created_at)}
                mono
              />
              <InfoRow
                label={t("admin.users.last_login")}
                value={formatDate(lastLogin, true)}
                mono
              />
              {!verified && (
                <Button
                  type="button"
                  variant="outline"
                  className="h-9 text-[13px]"
                  onClick={() => verifyMutation.mutate()}
                  disabled={verifyMutation.isPending}
                >
                  {verifyMutation.isPending
                    ? t("admin.users.verifying")
                    : t("admin.users.verify_manual")}
                </Button>
              )}
            </Card>

            <div className="space-y-3 rounded-2xl border bg-muted/30 px-5 py-4">
              <div className="flex items-center gap-2.5">
                <span
                  className={cn(
                    "size-1.5 rounded-full",
                    dirty ? "bg-amber-500" : "bg-muted-foreground/40"
                  )}
                />
                <span className="text-[13px] text-muted-foreground">
                  {dirty
                    ? t("admin.tariff_form.dirty")
                    : t("admin.tariff_form.clean")}
                </span>
              </div>
              <div className="flex gap-2.5">
                <Button
                  type="button"
                  variant="outline"
                  className="h-[38px] flex-1 text-[13px]"
                  onClick={() => router.push(`/admin/users/${id}`)}
                  disabled={saveMutation.isPending}
                >
                  {t("common.cancel")}
                </Button>
                <Button
                  type="submit"
                  className="h-[38px] flex-[2] text-[13px]"
                  disabled={saveMutation.isPending}
                >
                  {saveMutation.isPending
                    ? t("common.saving")
                    : t("common.save_changes")}
                </Button>
              </div>
            </div>
          </div>
        </div>
      </form>
    </PageShell>
  );
}

const inputClass = "h-[38px] rounded-lg text-[13px] md:text-[13px]";

function Card({
  title,
  description,
  action,
  children,
}: {
  title: string;
  description?: string;
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-5 rounded-2xl border bg-card px-5 py-5 sm:px-6">
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <div className="space-y-1">
          <div className="text-[15px] leading-none font-semibold">{title}</div>
          {description && (
            <div className="text-xs text-muted-foreground">{description}</div>
          )}
        </div>
        {action}
      </div>
      {children}
    </section>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-[7px]">
      <Label className="text-xs font-normal text-muted-foreground">
        {label}
      </Label>
      {children}
    </div>
  );
}

function InfoRow({
  label,
  value,
  mono,
  tone,
}: {
  label: string;
  value: string;
  mono?: boolean;
  tone?: "amber";
}) {
  return (
    <div className="flex items-center justify-between gap-3.5 text-[13px]">
      <span className="text-muted-foreground">{label}</span>
      <span
        className={cn(mono && "font-mono", tone === "amber" && "text-amber-500")}
      >
        {value}
      </span>
    </div>
  );
}
