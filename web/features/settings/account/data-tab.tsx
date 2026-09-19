"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Download, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { PasswordInput } from "@/components/password-input";
import { useAccountQuery } from "@/hooks/use-account";
import { useT } from "@/hooks/use-translations";
import {
  deleteOwnAccount,
  downloadAccountData,
  fetchAccountDeleteCheck,
  fetchAccountIdentification,
  fetchAccountLegal,
  logout,
} from "@/lib/api";
import { formatAmount } from "@/lib/format";
import { errorText, Field, formatDateTime, QueryState, SettingsSection } from "./ui";

export function DataTab() {
  const account = useAccountQuery();
  const staff = Boolean(account.data?.user.staff);
  return (
    <>
      <ConsentsCard />
      <IdentificationCard />
      <ExportCard />
      {account.data && !staff && <DeleteCard email={account.data.user.email} />}
    </>
  );
}

function ConsentsCard() {
  const t = useT();
  const q = useQuery({ queryKey: ["account-legal"], queryFn: fetchAccountLegal });
  const consents = q.data?.consents ?? [];
  return (
    <SettingsSection title={t("settings.data.consents_title")} description={t("settings.data.consents_hint")}>
      <QueryState isLoading={q.isLoading} isError={q.isError} error={q.error} onRetry={() => void q.refetch()}>
        {consents.length === 0 ? (
          <p className="text-[13px] text-muted-foreground">{t("settings.data.consents_empty")}</p>
        ) : (
          <ul className="divide-y divide-border rounded-xl border border-border">
            {consents.map((consent) => (
              <li key={`${consent.kind}-${consent.version}-${consent.at}`} className="flex flex-wrap items-center justify-between gap-2 px-4 py-3 text-[13.5px]">
                <a href={`/legal/${consent.kind}`} target="_blank" rel="noopener noreferrer" className="font-medium hover:underline">
                  {consent.title || consent.kind}
                </a>
                <span className="text-[12px] text-muted-foreground">
                  {t(consent.action === "accepted" ? "settings.data.consent_accepted" : "settings.data.consent_withdrawn", {
                    version: consent.version,
                    date: formatDateTime(consent.at),
                  })}
                </span>
              </li>
            ))}
          </ul>
        )}
      </QueryState>
    </SettingsSection>
  );
}

function IdentificationCard() {
  const t = useT();
  const q = useQuery({ queryKey: ["account-identification"], queryFn: fetchAccountIdentification });
  const ident = q.data;
  return (
    <SettingsSection title={t("settings.data.identification_title")} description={t("settings.data.identification_hint")}>
      <QueryState isLoading={q.isLoading} isError={q.isError} error={q.error} onRetry={() => void q.refetch()} rows={1}>
        {ident && (
          <div className="flex flex-wrap items-center gap-3 rounded-xl border border-border px-4 py-3 text-[13.5px]">
            <i className={ident.identified ? "ri-verified-badge-line text-xl text-emerald-500" : "ri-shield-user-line text-xl text-muted-foreground"} />
            <div className="min-w-0 flex-1">
              <div className="font-medium">
                {ident.identified
                  ? t("settings.data.identified", { date: formatDateTime(ident.identified_at) })
                  : ident.required
                    ? t("settings.data.not_identified_required")
                    : t("settings.data.not_identified")}
              </div>
              {ident.identified && ident.method && (
                <div className="text-[12px] text-muted-foreground">{t("settings.data.identified_method", { method: ident.method })}</div>
              )}
              {!ident.identified && ident.methods.length > 0 && (
                <div className="text-[12px] text-muted-foreground">
                  {t("settings.data.identify_methods", { methods: ident.methods.join(", ") })}
                </div>
              )}
            </div>
            {!ident.identified && ident.methods.length > 0 && (
              <Button asChild size="sm" variant="outline">
                <Link href="/billing#topup">{t("settings.data.identify_action")}</Link>
              </Button>
            )}
          </div>
        )}
      </QueryState>
    </SettingsSection>
  );
}

function ExportCard() {
  const t = useT();
  const [busy, setBusy] = useState(false);
  const run = async () => {
    setBusy(true);
    try {
      await downloadAccountData();
    } catch (err) {
      toast.error(errorText(err, t("settings.data.export_failed")));
    } finally {
      setBusy(false);
    }
  };
  return (
    <SettingsSection
      title={t("settings.data.export_title")}
      description={t("settings.data.export_hint")}
      action={
        <Button type="button" variant="outline" disabled={busy} onClick={() => void run()}>
          {busy ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
          {t("settings.data.export")}
        </Button>
      }
    />
  );
}

function DeleteCard({ email }: { email: string }) {
  const t = useT();
  const router = useRouter();
  const check = useQuery({ queryKey: ["account-delete-check"], queryFn: fetchAccountDeleteCheck });
  const [open, setOpen] = useState(false);
  const [confirmEmail, setConfirmEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const requires = check.data?.requires ?? { password: false, two_factor: false };

  const remove = useMutation({
    mutationFn: () =>
      deleteOwnAccount({
        confirm_email: confirmEmail.trim(),
        password: requires.password ? password : undefined,
        code: requires.two_factor ? code.trim() : undefined,
      }),
    onSuccess: async () => {
      toast.success(t("settings.data.deleted"));
      await logout().catch(() => undefined);
      router.replace("/");
    },
    onError: (err) => toast.error(errorText(err, t("settings.data.delete_failed"))),
  });

  const blockerText = (b: NonNullable<typeof check.data>["blockers"][number]) => {
    switch (b.kind) {
      case "servers":
        return { text: t("settings.data.blocker_servers", { count: b.count ?? 0 }), href: "/servers" };
      case "hosting":
        return { text: t("settings.data.blocker_hosting", { count: b.count ?? 0 }), href: "/hosting/my" };
      case "refund":
        return { text: t("settings.data.blocker_refund"), href: "/billing" };
      case "balance":
        return {
          text: t("settings.data.blocker_balance", {
            amounts: (b.amounts ?? []).map((a) => `${formatAmount(a.amount)} ${a.currency}`).join(", "),
          }),
          href: "/billing",
        };
    }
  };

  const canSubmit =
    confirmEmail.trim().toLowerCase() === email.toLowerCase() &&
    (!requires.password || password.length > 0) &&
    (!requires.two_factor || code.trim().length >= 6);

  return (
    <SettingsSection danger title={t("settings.data.delete_title")} description={t("settings.data.delete_hint")}>
      <QueryState isLoading={check.isLoading} isError={check.isError} error={check.error} onRetry={() => void check.refetch()} rows={1}>
        {check.data && check.data.blockers.length > 0 ? (
          <div className="space-y-2">
            <p className="text-[13px] font-medium">{t("settings.data.blockers_title")}</p>
            <ul className="space-y-2">
              {check.data.blockers.map((b) => {
                const item = blockerText(b);
                return (
                  <li key={b.kind} className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-border px-4 py-2.5 text-[13px]">
                    <span>{item.text}</span>
                    <Link href={item.href} className="text-[12.5px] text-primary hover:underline">
                      {t("settings.data.blocker_open")}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        ) : (
          <Button type="button" variant="destructive" onClick={() => setOpen(true)}>
            {t("settings.data.delete")}
          </Button>
        )}
      </QueryState>

      <ConfirmDialog
        open={open}
        onOpenChange={(v) => {
          setOpen(v);
          if (!v) {
            setConfirmEmail("");
            setPassword("");
            setCode("");
          }
        }}
        title={t("settings.data.delete_confirm_title")}
        desc={t("settings.data.delete_confirm_hint")}
        destructive
        cancelBtnText={t("common.cancel")}
        confirmText={t("settings.data.delete")}
        isLoading={remove.isPending}
        disabled={!canSubmit}
        handleConfirm={() => remove.mutate()}
      >
        <div className="grid gap-3">
          <Field label={t("settings.data.delete_confirm_label", { email })} htmlFor="delete-email">
            <Input id="delete-email" type="email" autoComplete="off" value={confirmEmail} onChange={(e) => setConfirmEmail(e.target.value)} />
          </Field>
          {requires.password && (
            <Field label={t("settings.password.current")} htmlFor="delete-password">
              <PasswordInput id="delete-password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />
            </Field>
          )}
          {requires.two_factor && (
            <Field label={t("settings.twofa.factor")} htmlFor="delete-code">
              <Input id="delete-code" autoComplete="one-time-code" placeholder="123456 / xxxx-xxxx" value={code} onChange={(e) => setCode(e.target.value)} />
            </Field>
          )}
        </div>
      </ConfirmDialog>
    </SettingsSection>
  );
}
