"use client";

import { useEffect, useRef, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, MailCheck, MailWarning, Send } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { PasswordInput } from "@/components/password-input";
import { ACCOUNT_KEY, useAccountQuery, useSetAccount } from "@/hooks/use-account";
import { useT } from "@/hooks/use-translations";
import {
  cancelEmailChange,
  changeAccountEmail,
  resendEmailVerification,
  updateAccount,
  type AccountUser,
} from "@/lib/api";
import { errorText, Field, FormActions, formatDateTime, QueryState, SettingsSection, useUnsavedGuard } from "./ui";

const RESEND_COOLDOWN = 60;

type ContactForm = { phone: string; telegram: string; discord: string; vk: string };

function contactsOf(user: AccountUser): ContactForm {
  return {
    phone: user.phone ?? "",
    telegram: user.contacts?.telegram ?? "",
    discord: user.contacts?.discord ?? "",
    vk: user.contacts?.vk ?? "",
  };
}

export function ContactsTab() {
  const q = useAccountQuery();
  return (
    <QueryState isLoading={q.isLoading} isError={q.isError} error={q.error} onRetry={() => void q.refetch()} rows={3}>
      {q.data && (
        <>
          <EmailCard user={q.data.user} />
          <ContactsCard user={q.data.user} />
        </>
      )}
    </QueryState>
  );
}

function useCooldown() {
  const [left, setLeft] = useState(0);
  useEffect(() => {
    if (left <= 0) return;
    const timer = window.setTimeout(() => setLeft((v) => v - 1), 1000);
    return () => window.clearTimeout(timer);
  }, [left]);
  return { left, start: () => setLeft(RESEND_COOLDOWN) };
}

function EmailCard({ user }: { user: AccountUser }) {
  const t = useT();
  const qc = useQueryClient();
  const setAccount = useSetAccount();
  const cooldown = useCooldown();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const emailRef = useRef<HTMLInputElement | null>(null);

  const resend = useMutation({
    mutationFn: resendEmailVerification,
    onSuccess: (res) => {
      cooldown.start();
      toast.success(res.message || t("settings.email.resent"));
    },
    onError: (err) => toast.error(errorText(err, t("settings.email.resend_failed"))),
  });

  const change = useMutation({
    mutationFn: () =>
      changeAccountEmail({
        email: email.trim(),
        current_password: user.has_password ? password : undefined,
      }),
    onSuccess: (res) => {
      setEmail("");
      setPassword("");
      void qc.invalidateQueries({ queryKey: ACCOUNT_KEY });
      toast.success(t("settings.email.change_sent", { email: res.pending_email }));
      if (res.confirm_url) {
        toast.info(t("settings.email.dev_link"), {
          action: { label: t("settings.email.open_link"), onClick: () => window.open(res.confirm_url, "_blank") },
        });
      }
    },
    onError: (err) => toast.error(errorText(err, t("settings.email.change_failed"))),
  });

  const cancel = useMutation({
    mutationFn: cancelEmailChange,
    onSuccess: (res) => {
      setAccount(res.user);
      toast.success(t("settings.email.pending_cancelled"));
    },
    onError: (err) => toast.error(errorText(err, t("common.error"))),
  });

  const validEmail = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim());
  const sameEmail = email.trim().toLowerCase() === user.email.toLowerCase();

  return (
    <SettingsSection title={t("settings.email.title")} description={t("settings.email.hint")}>
      <div className="flex flex-wrap items-center gap-3 rounded-xl border border-border px-4 py-3">
        {user.email_verified ? (
          <MailCheck className="size-5 shrink-0 text-emerald-500" />
        ) : (
          <MailWarning className="size-5 shrink-0 text-amber-500" />
        )}
        <div className="min-w-0 flex-1">
          <div className="truncate text-[14px] font-medium">{user.email}</div>
          <div className="text-[12px] text-muted-foreground">
            {user.email_verified
              ? t("settings.email.verified_at", { date: formatDateTime(user.email_verified_at) })
              : t("settings.email.unverified_hint")}
          </div>
        </div>
        {user.email_verified ? (
          <Badge variant="secondary" className="text-emerald-600 dark:text-emerald-400">
            {t("settings.email.verified")}
          </Badge>
        ) : (
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={resend.isPending || cooldown.left > 0}
            onClick={() => resend.mutate()}
          >
            {resend.isPending ? <Loader2 className="size-3.5 animate-spin" /> : <Send className="size-3.5" />}
            {cooldown.left > 0 ? t("settings.email.resend_in", { seconds: cooldown.left }) : t("settings.email.resend")}
          </Button>
        )}
      </div>

      {user.pending_email && (
        <div className="mt-3 flex flex-wrap items-center gap-3 rounded-xl border border-amber-500/30 bg-amber-500/5 px-4 py-3">
          <div className="min-w-0 flex-1 text-[13px]">
            <div className="font-medium">{t("settings.email.pending_title", { email: user.pending_email })}</div>
            <div className="text-muted-foreground">{t("settings.email.pending_hint")}</div>
          </div>
          <Button type="button" size="sm" variant="ghost" disabled={cancel.isPending} onClick={() => cancel.mutate()}>
            {t("settings.email.pending_cancel")}
          </Button>
        </div>
      )}

      <form
        className="mt-5"
        onSubmit={(e) => {
          e.preventDefault();
          if (!validEmail) {
            toast.error(t("settings.email.invalid"));
            emailRef.current?.focus();
            return;
          }
          if (sameEmail) {
            toast.error(t("settings.email.same"));
            return;
          }
          change.mutate();
        }}
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label={t("settings.email.new")} htmlFor="new-email">
            <Input
              ref={emailRef}
              id="new-email"
              type="email"
              inputMode="email"
              autoComplete="email"
              placeholder="name@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </Field>
          {user.has_password && (
            <Field label={t("settings.password.current")} htmlFor="email-password">
              <PasswordInput
                id="email-password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </Field>
          )}
        </div>
        <div className="mt-5">
          <Button
            type="submit"
            disabled={change.isPending || !email.trim() || (user.has_password && !password)}
          >
            {change.isPending && <Loader2 className="size-4 animate-spin" />}
            {t("settings.email.change")}
          </Button>
        </div>
      </form>
    </SettingsSection>
  );
}

function ContactsCard({ user }: { user: AccountUser }) {
  const t = useT();
  const setAccount = useSetAccount();
  const [form, setForm] = useState<ContactForm>(() => contactsOf(user));
  const [base, setBase] = useState<ContactForm>(() => contactsOf(user));
  const baseRef = useRef(base);
  baseRef.current = base;
  const userKey = JSON.stringify(contactsOf(user));
  const dirty = JSON.stringify(form) !== JSON.stringify(base);
  useUnsavedGuard(dirty);

  useEffect(() => {
    const next = JSON.parse(userKey) as ContactForm;
    setForm((current) => (JSON.stringify(current) === JSON.stringify(baseRef.current) ? next : current));
    setBase(next);
  }, [userKey]);

  const save = useMutation({
    mutationFn: () =>
      updateAccount({
        phone: form.phone.trim(),
        contacts: { telegram: form.telegram.trim(), discord: form.discord.trim(), vk: form.vk.trim() },
      }),
    onSuccess: (res) => {
      setAccount(res.user);
      toast.success(t("settings.contacts.saved"));
    },
    onError: (err) => toast.error(errorText(err, t("common.save_failed"))),
  });

  const fields: { key: keyof ContactForm; icon: string; label: string; placeholder: string; props?: React.InputHTMLAttributes<HTMLInputElement> }[] = [
    {
      key: "phone",
      icon: "ri-phone-line",
      label: t("settings.contacts.phone"),
      placeholder: "+7 900 000-00-00",
      props: { type: "tel", inputMode: "tel", autoComplete: "tel" },
    },
    { key: "telegram", icon: "ri-telegram-line", label: "Telegram", placeholder: "@username" },
    { key: "discord", icon: "ri-discord-line", label: "Discord", placeholder: "username" },
    { key: "vk", icon: "ri-vk-line", label: "VK", placeholder: "vk.com/id" },
  ];

  return (
    <SettingsSection title={t("settings.contacts.title")} description={t("settings.contacts.hint")}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (dirty) save.mutate();
        }}
      >
        <div className="grid gap-4 sm:grid-cols-2">
          {fields.map((field) => (
            <Field key={field.key} label={field.label} htmlFor={`contact-${field.key}`}>
              <div className="relative">
                <i className={`${field.icon} pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted-foreground`} />
                <Input
                  id={`contact-${field.key}`}
                  className="pl-9"
                  maxLength={64}
                  placeholder={field.placeholder}
                  value={form[field.key]}
                  onChange={(e) => setForm({ ...form, [field.key]: e.target.value })}
                  {...field.props}
                />
              </div>
            </Field>
          ))}
        </div>
        <FormActions dirty={dirty} pending={save.isPending} onReset={() => setForm(base)} />
      </form>
    </SettingsSection>
  );
}
