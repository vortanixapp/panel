"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, CheckCircle2, CircleAlert, Copy, Download, KeyRound, Loader2, ShieldCheck } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { InputOTP, InputOTPGroup, InputOTPSlot } from "@/components/ui/input-otp";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { PasswordInput } from "@/components/password-input";
import { TelegramLoginButton } from "@/components/auth/telegram-login-button";
import { ACCOUNT_KEY, useAccountQuery } from "@/hooks/use-account";
import { useChangePassword } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";
import {
  disable2FA,
  enable2FA,
  fetchAccountLogins,
  fetchSocialProviders,
  generate2FA,
  regenerateRecoveryCodes,
  socialLinkRedirect,
  socialUnlink,
  telegramLinkAccount,
  TENANT_SLUG,
  type AccountUser,
  type TelegramAuthUser,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { errorText, Field, QueryState, SettingsSection, StatusDot, useCopy } from "./ui";

const PROVIDERS = [
  { key: "google", label: "Google", icon: "ri-google-fill" },
  { key: "discord", label: "Discord", icon: "ri-discord-fill" },
  { key: "vk", label: "VK", icon: "ri-vk-fill" },
  { key: "telegram", label: "Telegram", icon: "ri-telegram-fill" },
] as const;

const WEEK_MS = 7 * 24 * 60 * 60 * 1000;

function scrollToSection(id: string) {
  document.getElementById(id)?.scrollIntoView({ behavior: "smooth", block: "start" });
}

export function SecurityTab() {
  const q = useAccountQuery();
  return (
    <QueryState isLoading={q.isLoading} isError={q.isError} error={q.error} onRetry={() => void q.refetch()} rows={4}>
      {q.data && (
        <>
          <OverviewCard user={q.data.user} />
          <PasswordCard user={q.data.user} />
          <TwoFactorCard user={q.data.user} />
          <LinkedCard user={q.data.user} />
        </>
      )}
    </QueryState>
  );
}

function OverviewCard({ user }: { user: AccountUser }) {
  const t = useT();
  const logins = useQuery({ queryKey: ["account-logins", "summary"], queryFn: () => fetchAccountLogins() });
  const providers = useQuery({ queryKey: ["social-providers"], queryFn: fetchSocialProviders, staleTime: 300_000 });
  const since = Date.now() - WEEK_MS;
  const failed = (logins.data?.logins ?? []).filter(
    (l) => !l.success && new Date(l.created_at).getTime() >= since
  ).length;
  const socialAvailable =
    (providers.data?.providers?.length ?? 0) > 0 || Boolean(providers.data?.telegram?.enabled) || (user.linked_providers?.length ?? 0) > 0;

  type Item = { key: string; ok: boolean; title: string; detail: string; action?: React.ReactNode };
  const items: Item[] = [
    {
      key: "email",
      ok: Boolean(user.email_verified),
      title: t("settings.overview.email"),
      detail: user.email_verified ? t("settings.overview.email_ok") : t("settings.overview.email_bad"),
      action: !user.email_verified && (
        <Button asChild size="sm" variant="outline">
          <Link href="/settings?tab=contacts">{t("settings.overview.email_action")}</Link>
        </Button>
      ),
    },
    {
      key: "password",
      ok: Boolean(user.has_password),
      title: t("settings.overview.password"),
      detail: user.has_password ? t("settings.overview.password_ok") : t("settings.overview.password_bad"),
      action: !user.has_password && (
        <Button size="sm" variant="outline" onClick={() => scrollToSection("password")}>
          {t("settings.overview.password_action")}
        </Button>
      ),
    },
    {
      key: "twofa",
      ok: user.two_factor_enabled,
      title: t("settings.overview.twofa"),
      detail: user.two_factor_enabled ? t("settings.overview.twofa_ok") : t("settings.overview.twofa_bad"),
      action: !user.two_factor_enabled && (
        <Button size="sm" variant="outline" onClick={() => scrollToSection("twofa")}>
          {t("settings.overview.twofa_action")}
        </Button>
      ),
    },
  ];
  if (user.two_factor_enabled) {
    const left = user.recovery_codes_left ?? 0;
    items.push({
      key: "codes",
      ok: left > 2,
      title: t("settings.overview.codes"),
      detail: t("settings.overview.codes_left", { count: left }),
      action: left <= 2 && (
        <Button size="sm" variant="outline" onClick={() => scrollToSection("twofa")}>
          {t("settings.overview.codes_action")}
        </Button>
      ),
    });
  }
  if (socialAvailable) {
    const linked = user.linked_providers?.length ?? 0;
    items.push({
      key: "linked",
      ok: linked > 0,
      title: t("settings.overview.linked"),
      detail: linked > 0 ? t("settings.overview.linked_ok", { count: linked }) : t("settings.overview.linked_bad"),
      action: linked === 0 && (
        <Button size="sm" variant="outline" onClick={() => scrollToSection("linked")}>
          {t("settings.overview.linked_action")}
        </Button>
      ),
    });
  }
  items.push({
    key: "logins",
    ok: failed === 0,
    title: t("settings.overview.logins"),
    detail: failed === 0 ? t("settings.overview.logins_ok") : t("settings.overview.logins_bad", { count: failed }),
    action: failed > 0 && (
      <Button asChild size="sm" variant="outline">
        <Link href="/settings?tab=sessions">{t("settings.overview.logins_action")}</Link>
      </Button>
    ),
  });

  const score = Math.round((items.filter((i) => i.ok).length / items.length) * 100);
  const tone = score >= 80 ? "ok" : score >= 50 ? "warn" : "bad";

  return (
    <SettingsSection title={t("settings.overview.title")} description={t("settings.overview.hint")}>
      <div className="flex items-center gap-4">
        <div
          className="relative grid size-16 shrink-0 place-items-center rounded-full"
          style={{
            background: `conic-gradient(${tone === "ok" ? "rgb(16 185 129)" : tone === "warn" ? "rgb(245 158 11)" : "rgb(244 63 94)"} ${score * 3.6}deg, var(--muted) 0deg)`,
          }}
        >
          <div className="grid size-12 place-items-center rounded-full bg-card text-[15px] font-semibold">{score}%</div>
        </div>
        <div className="text-[13.5px]">
          <div className="font-medium">
            {tone === "ok"
              ? t("settings.overview.score_ok")
              : tone === "warn"
                ? t("settings.overview.score_warn")
                : t("settings.overview.score_bad")}
          </div>
          <div className="text-muted-foreground">{t("settings.overview.score_hint")}</div>
        </div>
      </div>
      <ul className="mt-5 divide-y divide-border rounded-xl border border-border">
        {items.map((item) => (
          <li key={item.key} className="flex flex-wrap items-center gap-3 px-4 py-3">
            {item.ok ? (
              <CheckCircle2 className="size-5 shrink-0 text-emerald-500" />
            ) : (
              <CircleAlert className="size-5 shrink-0 text-amber-500" />
            )}
            <div className="min-w-0 flex-1">
              <div className="text-[13.5px] font-medium">{item.title}</div>
              <div className="text-[12.5px] text-muted-foreground">{item.detail}</div>
            </div>
            {item.action}
          </li>
        ))}
      </ul>
    </SettingsSection>
  );
}

function passwordScore(value: string): number {
  if (!value) return 0;
  let score = 0;
  if (value.length >= 8) score++;
  if (value.length >= 12) score++;
  if (/[a-zа-яё]/.test(value) && /[A-ZА-ЯЁ]/.test(value)) score++;
  if (/\d/.test(value)) score++;
  if (/[^\p{L}\p{N}]/u.test(value)) score++;
  return value.length < 8 ? Math.min(score, 1) : score;
}

function PasswordCard({ user }: { user: AccountUser }) {
  const t = useT();
  const qc = useQueryClient();
  const change = useChangePassword();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const hasPassword = Boolean(user.has_password);
  const score = passwordScore(next);
  const labels = [
    t("settings.password.strength_0"),
    t("settings.password.strength_1"),
    t("settings.password.strength_2"),
    t("settings.password.strength_3"),
    t("settings.password.strength_4"),
    t("settings.password.strength_5"),
  ];

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (next.length < 8) {
      toast.error(t("settings.password.min"));
      return;
    }
    if (next !== confirm) {
      toast.error(t("settings.password.mismatch"));
      return;
    }
    if (hasPassword && next === current) {
      toast.error(t("settings.password.same"));
      return;
    }
    try {
      const res = await change.mutateAsync({ currentPassword: hasPassword ? current : "", newPassword: next });
      setCurrent("");
      setNext("");
      setConfirm("");
      void qc.invalidateQueries({ queryKey: ACCOUNT_KEY });
      void qc.invalidateQueries({ queryKey: ["account-sessions"] });
      toast.success(
        res.was_set
          ? t("settings.password.updated", { count: res.sessions_closed })
          : t("settings.password.set_done")
      );
    } catch (err) {
      toast.error(errorText(err, t("settings.password.failed")));
    }
  };

  return (
    <SettingsSection
      id="password"
      title={hasPassword ? t("settings.password.title") : t("settings.password.set_title")}
      description={hasPassword ? t("settings.password.hint") : t("settings.password.set_hint")}
    >
      <form className="grid max-w-xl gap-4" onSubmit={(e) => void submit(e)}>
        {hasPassword && (
          <Field label={t("settings.password.current")} htmlFor="pw-current">
            <PasswordInput id="pw-current" autoComplete="current-password" value={current} onChange={(e) => setCurrent(e.target.value)} />
          </Field>
        )}
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label={t("settings.password.new")} htmlFor="pw-new">
            <PasswordInput id="pw-new" autoComplete="new-password" value={next} onChange={(e) => setNext(e.target.value)} />
          </Field>
          <Field label={t("settings.password.confirm")} htmlFor="pw-confirm">
            <PasswordInput id="pw-confirm" autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)} />
          </Field>
        </div>
        {next && (
          <div className="space-y-1.5">
            <div className="grid grid-cols-5 gap-1">
              {[1, 2, 3, 4, 5].map((i) => (
                <span
                  key={i}
                  className={cn(
                    "h-1.5 rounded-full bg-muted",
                    i <= score && (score <= 2 ? "bg-rose-500" : score === 3 ? "bg-amber-500" : "bg-emerald-500")
                  )}
                />
              ))}
            </div>
            <p className="text-[12px] text-muted-foreground">
              {labels[score]} · {t("settings.password.tips")}
            </p>
          </div>
        )}
        <div>
          <Button type="submit" disabled={change.isPending || !next || !confirm || (hasPassword && !current)}>
            {change.isPending && <Loader2 className="size-4 animate-spin" />}
            {hasPassword ? t("settings.password.submit") : t("settings.password.set_submit")}
          </Button>
        </div>
      </form>
    </SettingsSection>
  );
}

function RecoveryCodesDialog({ codes, onClose }: { codes: string[]; onClose: () => void }) {
  const t = useT();
  const { copy } = useCopy();
  const [saved, setSaved] = useState(false);
  const text = codes.join("\n");
  const download = () => {
    const blob = new Blob([`${t("settings.twofa.codes_file_title")}\n\n${text}\n`], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "recovery-codes.txt";
    a.click();
    URL.revokeObjectURL(url);
  };
  return (
    <Dialog open={codes.length > 0} onOpenChange={(open) => !open && saved && onClose()}>
      <DialogContent showCloseButton={false} className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("settings.twofa.codes_title")}</DialogTitle>
          <DialogDescription>{t("settings.twofa.codes_hint")}</DialogDescription>
        </DialogHeader>
        <div className="grid grid-cols-2 gap-2 rounded-xl border border-border bg-muted/40 p-4 font-mono text-[14px]">
          {codes.map((code) => (
            <span key={code} className="text-center tracking-wider select-all">
              {code}
            </span>
          ))}
        </div>
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="outline" size="sm" onClick={() => void copy(text, "codes")}>
            <Copy className="size-3.5" />
            {t("settings.twofa.codes_copy")}
          </Button>
          <Button type="button" variant="outline" size="sm" onClick={download}>
            <Download className="size-3.5" />
            {t("settings.twofa.codes_download")}
          </Button>
        </div>
        <label className="flex items-start gap-2.5 text-[13px]">
          <Checkbox checked={saved} onCheckedChange={(v) => setSaved(v === true)} className="mt-0.5" />
          {t("settings.twofa.codes_saved")}
        </label>
        <DialogFooter>
          <Button type="button" disabled={!saved} onClick={onClose}>
            {t("settings.twofa.codes_done")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function CodeInput({ value, onChange, autoFocus }: { value: string; onChange: (v: string) => void; autoFocus?: boolean }) {
  return (
    <InputOTP maxLength={6} value={value} onChange={onChange} autoFocus={autoFocus} inputMode="numeric" pattern="^[0-9]*$">
      <InputOTPGroup>
        {[0, 1, 2, 3, 4, 5].map((i) => (
          <InputOTPSlot key={i} index={i} className="size-11 text-base" />
        ))}
      </InputOTPGroup>
    </InputOTP>
  );
}

function TwoFactorCard({ user }: { user: AccountUser }) {
  const t = useT();
  const qc = useQueryClient();
  const { copied, copy } = useCopy();
  const [setup, setSetup] = useState<{ secret: string; uri: string; qr?: string } | null>(null);
  const [code, setCode] = useState("");
  const [codes, setCodes] = useState<string[]>([]);
  const [disableOpen, setDisableOpen] = useState(false);
  const [regenOpen, setRegenOpen] = useState(false);
  const [password, setPassword] = useState("");
  const [factor, setFactor] = useState("");
  const refresh = () => qc.invalidateQueries({ queryKey: ACCOUNT_KEY });

  const start = useMutation({
    mutationFn: generate2FA,
    onSuccess: (res) => {
      setSetup(res);
      setCode("");
    },
    onError: (err) => toast.error(errorText(err, t("settings.twofa.create_failed"))),
  });

  const enable = useMutation({
    mutationFn: () => enable2FA(code),
    onSuccess: (res) => {
      setSetup(null);
      setCode("");
      setCodes(res.recovery_codes ?? []);
      void refresh();
      toast.success(t("settings.twofa.enabled"));
    },
    onError: (err) => toast.error(errorText(err, t("settings.twofa.enable_failed"))),
  });

  const disable = useMutation({
    mutationFn: () => disable2FA({ password: user.has_password ? password : undefined, code: factor.trim() }),
    onSuccess: () => {
      setDisableOpen(false);
      setPassword("");
      setFactor("");
      void refresh();
      toast.success(t("settings.twofa.disabled"));
    },
    onError: (err) => toast.error(errorText(err, t("settings.twofa.disable_failed"))),
  });

  const regen = useMutation({
    mutationFn: () => regenerateRecoveryCodes(factor.trim()),
    onSuccess: (res) => {
      setRegenOpen(false);
      setFactor("");
      setCodes(res.recovery_codes);
      void refresh();
    },
    onError: (err) => toast.error(errorText(err, t("settings.twofa.regen_failed"))),
  });

  const left = user.recovery_codes_left ?? 0;
  const locked = Boolean(user.two_factor_required);

  return (
    <SettingsSection
      id="twofa"
      title={t("settings.twofa.title")}
      description={t("settings.twofa.hint")}
      action={
        user.two_factor_enabled ? (
          <span className="flex items-center gap-2 text-[13px] font-medium text-emerald-600 dark:text-emerald-400">
            <ShieldCheck className="size-4" />
            {t("settings.twofa.on")}
          </span>
        ) : (
          <span className="flex items-center gap-2 text-[13px] text-muted-foreground">
            <StatusDot tone="off" />
            {t("settings.twofa.off")}
          </span>
        )
      }
    >
      {!user.two_factor_enabled && !setup && (
        <Button type="button" disabled={start.isPending} onClick={() => start.mutate()}>
          {start.isPending && <Loader2 className="size-4 animate-spin" />}
          {t("settings.twofa.enable")}
        </Button>
      )}

      {!user.two_factor_enabled && setup && (
        <div className="grid gap-6 rounded-xl border border-dashed border-border p-4 sm:p-5 md:grid-cols-[auto_minmax(0,1fr)]">
          <div className="flex flex-col items-center gap-2">
            {setup.qr ? (
              <img src={setup.qr} alt={t("settings.twofa.qr_alt")} className="size-44 rounded-lg bg-white p-2" />
            ) : null}
            <a href={setup.uri} className="text-[12.5px] text-primary hover:underline md:hidden">
              {t("settings.twofa.open_app")}
            </a>
          </div>
          <div className="min-w-0 space-y-4">
            <ol className="list-decimal space-y-1 pl-5 text-[13px] text-muted-foreground">
              <li>{t("settings.twofa.step_install")}</li>
              <li>{t("settings.twofa.step_scan")}</li>
              <li>{t("settings.twofa.step_code")}</li>
            </ol>
            <div className="space-y-1.5">
              <div className="text-[12px] text-muted-foreground">{t("settings.twofa.manual")}</div>
              <div className="flex items-center gap-2">
                <code className="min-w-0 flex-1 truncate rounded-lg bg-muted px-3 py-2 font-mono text-[12.5px] tracking-wider">
                  {setup.secret}
                </code>
                <Button type="button" size="icon" variant="outline" className="size-9 shrink-0" aria-label={t("settings.common.copy")} onClick={() => void copy(setup.secret, "secret")}>
                  {copied === "secret" ? <Check className="size-4" /> : <Copy className="size-4" />}
                </Button>
              </div>
            </div>
            <form
              className="space-y-3"
              onSubmit={(e) => {
                e.preventDefault();
                if (code.length === 6) enable.mutate();
              }}
            >
              <CodeInput value={code} onChange={setCode} autoFocus />
              <div className="flex flex-wrap gap-2">
                <Button type="submit" disabled={code.length !== 6 || enable.isPending}>
                  {enable.isPending && <Loader2 className="size-4 animate-spin" />}
                  {t("settings.twofa.confirm")}
                </Button>
                <Button type="button" variant="ghost" onClick={() => setSetup(null)}>
                  {t("common.cancel")}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}

      {user.two_factor_enabled && (
        <div className="space-y-4">
          <div className="flex flex-wrap items-center gap-3 rounded-xl border border-border px-4 py-3">
            <KeyRound className={cn("size-5 shrink-0", left > 2 ? "text-muted-foreground" : "text-amber-500")} />
            <div className="min-w-0 flex-1 text-[13px]">
              <div className="font-medium">{t("settings.twofa.codes_left", { count: left, total: 10 })}</div>
              <div className="text-muted-foreground">{t("settings.twofa.codes_left_hint")}</div>
            </div>
            <Button type="button" size="sm" variant="outline" onClick={() => setRegenOpen(true)}>
              {t("settings.twofa.regen")}
            </Button>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <Button type="button" variant="destructive" disabled={locked} onClick={() => setDisableOpen(true)}>
              {t("settings.twofa.disable")}
            </Button>
            {locked && <span className="text-[12.5px] text-muted-foreground">{t("settings.twofa.required")}</span>}
          </div>
        </div>
      )}

      <RecoveryCodesDialog codes={codes} onClose={() => setCodes([])} />

      <ConfirmDialog
        open={disableOpen}
        onOpenChange={(open) => {
          setDisableOpen(open);
          if (!open) {
            setPassword("");
            setFactor("");
          }
        }}
        title={t("settings.twofa.disable_title")}
        desc={t("settings.twofa.disable_hint")}
        destructive
        cancelBtnText={t("common.cancel")}
        confirmText={t("settings.twofa.disable")}
        isLoading={disable.isPending}
        disabled={!factor.trim() || (Boolean(user.has_password) && !password)}
        handleConfirm={() => disable.mutate()}
      >
        <div className="grid gap-3">
          {user.has_password && (
            <Field label={t("settings.password.current")} htmlFor="twofa-off-password">
              <PasswordInput id="twofa-off-password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />
            </Field>
          )}
          <Field label={t("settings.twofa.factor")} htmlFor="twofa-off-code" hint={t("settings.twofa.factor_hint")}>
            <Input id="twofa-off-code" autoComplete="one-time-code" placeholder="123456 / xxxx-xxxx" value={factor} onChange={(e) => setFactor(e.target.value)} />
          </Field>
        </div>
      </ConfirmDialog>

      <ConfirmDialog
        open={regenOpen}
        onOpenChange={(open) => {
          setRegenOpen(open);
          if (!open) setFactor("");
        }}
        title={t("settings.twofa.regen_title")}
        desc={t("settings.twofa.regen_hint")}
        cancelBtnText={t("common.cancel")}
        confirmText={t("settings.twofa.regen")}
        isLoading={regen.isPending}
        disabled={!factor.trim()}
        handleConfirm={() => regen.mutate()}
      >
        <Field label={t("settings.twofa.factor")} htmlFor="twofa-regen-code">
          <Input id="twofa-regen-code" autoComplete="one-time-code" placeholder="123456" value={factor} onChange={(e) => setFactor(e.target.value)} />
        </Field>
      </ConfirmDialog>
    </SettingsSection>
  );
}

function LinkedCard({ user }: { user: AccountUser }) {
  const t = useT();
  const qc = useQueryClient();
  const providers = useQuery({ queryKey: ["social-providers"], queryFn: fetchSocialProviders, staleTime: 300_000 });
  const [unlinkKey, setUnlinkKey] = useState<string | null>(null);
  const linked = useMemo(() => new Set(user.linked_providers ?? []), [user.linked_providers]);
  const configured = new Set(providers.data?.providers ?? []);
  const telegramBot = providers.data?.telegram?.bot_username ?? "";
  const telegramOn = Boolean(providers.data?.telegram?.enabled) && telegramBot !== "";
  if (telegramOn) configured.add("telegram");
  const shown = PROVIDERS.filter((p) => configured.has(p.key) || linked.has(p.key));
  const lastMethod = !user.has_password && linked.size <= 1;

  const unlink = useMutation({
    mutationFn: (key: string) => socialUnlink(key),
    onSuccess: () => {
      setUnlinkKey(null);
      void qc.invalidateQueries({ queryKey: ACCOUNT_KEY });
      toast.success(t("settings.linked.unlinked"));
    },
    onError: (err) => toast.error(errorText(err, t("settings.linked.unlink_failed"))),
  });

  const link = async (key: string) => {
    try {
      const res = await socialLinkRedirect(key, TENANT_SLUG);
      if (!res.url) {
        toast.error(t("settings.linked.unavailable"));
        return;
      }
      sessionStorage.setItem("social_auth_provider", key);
      sessionStorage.setItem("social_auth_link", "true");
      window.location.href = res.url;
    } catch (err) {
      toast.error(errorText(err, t("settings.linked.link_failed")));
    }
  };

  const linkTelegram = async (tgUser: TelegramAuthUser) => {
    try {
      await telegramLinkAccount(tgUser);
      void qc.invalidateQueries({ queryKey: ACCOUNT_KEY });
      toast.success(t("settings.linked.linked", { provider: "Telegram" }));
    } catch (err) {
      toast.error(errorText(err, t("settings.linked.link_failed")));
    }
  };

  const unlinkLabel = PROVIDERS.find((p) => p.key === unlinkKey)?.label ?? "";

  return (
    <SettingsSection id="linked" title={t("settings.linked.title")} description={t("settings.linked.hint")}>
      <QueryState isLoading={providers.isLoading} isError={false} onRetry={() => void providers.refetch()} rows={1}>
        {shown.length === 0 ? (
          <p className="rounded-xl border border-dashed border-border px-4 py-4 text-[13px] text-muted-foreground">
            {t("settings.linked.none")}
          </p>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            {shown.map((p) => {
              const isLinked = linked.has(p.key);
              const available = configured.has(p.key);
              return (
                <div key={p.key} className="flex items-center gap-3 rounded-xl border border-border px-4 py-3">
                  <i className={cn(p.icon, "text-xl", isLinked ? "text-foreground" : "text-muted-foreground")} />
                  <div className="min-w-0 flex-1">
                    <div className="text-[13.5px] font-medium">{p.label}</div>
                    <div className="text-[12px] text-muted-foreground">
                      {isLinked ? t("settings.linked.on") : t("settings.linked.off")}
                    </div>
                  </div>
                  {isLinked ? (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      disabled={lastMethod}
                      title={lastMethod ? t("settings.linked.last_method") : undefined}
                      onClick={() => setUnlinkKey(p.key)}
                    >
                      {t("settings.linked.unlink")}
                    </Button>
                  ) : p.key === "telegram" ? (
                    available && <TelegramLoginButton botUsername={telegramBot} buttonSize="small" onAuth={(u) => void linkTelegram(u)} />
                  ) : (
                    available && (
                      <Button type="button" size="sm" onClick={() => void link(p.key)}>
                        {t("settings.linked.link")}
                      </Button>
                    )
                  )}
                </div>
              );
            })}
          </div>
        )}
        {lastMethod && linked.size > 0 && (
          <p className="mt-3 text-[12.5px] text-muted-foreground">{t("settings.linked.last_method")}</p>
        )}
      </QueryState>

      <ConfirmDialog
        open={unlinkKey !== null}
        onOpenChange={(open) => !open && setUnlinkKey(null)}
        title={t("settings.linked.unlink_title", { provider: unlinkLabel })}
        desc={t("settings.linked.unlink_hint", { provider: unlinkLabel })}
        destructive
        cancelBtnText={t("common.cancel")}
        confirmText={t("settings.linked.unlink")}
        isLoading={unlink.isPending}
        handleConfirm={() => unlinkKey && unlink.mutate(unlinkKey)}
      />
    </SettingsSection>
  );
}
