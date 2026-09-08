"use client";

import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PasswordInput } from "@/components/password-input";
import { Badge } from "@/components/ui/badge";
import { AppearanceForm } from "@/features/settings/appearance/appearance-form";
import { DisplayForm } from "@/features/settings/display/display-form";
import { NotificationsForm } from "@/features/settings/notifications/notifications-form";
import {
  changeAccountEmail,
  destroyAccountSession,
  disable2FA,
  enable2FA,
  fetchAccount,
  fetchAccountSessions,
  fetchSocialProviders,
  generate2FA,
  socialLinkRedirect,
  socialUnlink,
  telegramLinkAccount,
  TENANT_SLUG,
  updateAccount,
  uploadAccountAvatar,
  type TelegramAuthUser,
} from "@/lib/api";
import { TelegramLoginButton } from "@/components/auth/telegram-login-button";
import { useChangePassword } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";
import { browserLocale, DEFAULT_LOCALE, localeTag } from "@/lib/i18n";
import { setAccountPreferences, getAccountPreferences } from "@/lib/user-preferences";

const SOCIAL_PROVIDERS = [
  { key: "google", label: "Google", icon: "ri-google-line" },
  { key: "discord", label: "Discord", icon: "ri-discord-line" },
  { key: "vk", label: "VK", icon: "ri-vk-line" },
] as const;

// Список объявлен на уровне модуля: держим ключи, а не готовые подписи —
// иначе язык вкладок застыл бы на том, что стоял при загрузке страницы.
const ACCOUNT_TABS = [
  { id: "profile", labelKey: "settings.account.tab_profile", icon: "ri-user-line" },
  { id: "contacts", labelKey: "settings.account.tab_contacts", icon: "ri-at-line" },
  { id: "security", labelKey: "settings.account.tab_security", icon: "ri-lock-line" },
  { id: "2fa", labelKey: "settings.account.tab_twofa", icon: "ri-shield-keyhole-line" },
  { id: "sessions", labelKey: "settings.account.tab_sessions", icon: "ri-device-line" },
  { id: "appearance", labelKey: "settings.account.tab_appearance", icon: "ri-palette-line" },
  {
    id: "notifications",
    labelKey: "settings.account.tab_notifications",
    icon: "ri-notification-3-line",
  },
] as const;

type AccountTab = (typeof ACCOUNT_TABS)[number]["id"];

export function AccountSettings() {
  const t = useT();
  const qc = useQueryClient();
  const changePassword = useChangePassword();

  const { data, isLoading } = useQuery({
    queryKey: ["account"],
    queryFn: fetchAccount,
  });

  const user = data?.user;

  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [phone, setPhone] = useState("");
  const [locale, setLocale] = useState(DEFAULT_LOCALE as string);

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const [twoFaSecret, setTwoFaSecret] = useState("");
  const [twoFaUri, setTwoFaUri] = useState("");

  const [newEmail, setNewEmail] = useState("");
  const [emailPassword, setEmailPassword] = useState("");
  const [avatarUploading, setAvatarUploading] = useState(false);
  const [sessionAction, setSessionAction] = useState(false);
  // Вкладка берётся из адреса: ссылки «Настроить» со страницы оповещений и из
  // бокового меню вели на /settings/notifications, который просто перенаправлял
  // на /settings, и человек оказывался на «Аккаунте», где нужного ему нет.
  const searchParams = useSearchParams();
  const requestedTab = searchParams.get("tab");
  const [tab, setTab] = useState<AccountTab>(() =>
    ACCOUNT_TABS.some((t) => t.id === requestedTab) ? (requestedTab as AccountTab) : "profile"
  );
  useEffect(() => {
    if (requestedTab && ACCOUNT_TABS.some((t) => t.id === requestedTab)) {
      setTab(requestedTab as AccountTab);
    }
  }, [requestedTab]);

  const sessionsQuery = useQuery({
    queryKey: ["account-sessions"],
    queryFn: fetchAccountSessions,
  });

  const providersQuery = useQuery({
    queryKey: ["social-providers"],
    queryFn: fetchSocialProviders,
    staleTime: 5 * 60 * 1000,
  });

  useEffect(() => {
    if (!user) return;
    setFirstName(user.first_name ?? "");
    setLastName(user.last_name ?? "");
    setDisplayName(user.display_name ?? "");
    setPhone(user.phone ?? "");
    setLocale(user.locale || getAccountPreferences().language || browserLocale());
    setNewEmail(user.email ?? "");
  }, [user]);

  const reloadAccount = () => qc.invalidateQueries({ queryKey: ["account"] });

  const saveProfile = useMutation({
    mutationFn: () =>
      updateAccount({
        first_name: firstName || null,
        last_name: lastName || null,
        display_name: displayName || null,
        phone: phone || null,
        locale,
      }),
    onSuccess: () => {
      setAccountPreferences({ language: locale });
      toast.success(t("settings.account.profile_saved"));
      reloadAccount();
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.save_failed")),
  });

  async function handlePassword() {
    if (newPassword.length < 8) {
      toast.error(t("settings.account.password_min"));
      return;
    }
    if (newPassword !== confirmPassword) {
      toast.error(t("settings.account.password_mismatch"));
      return;
    }
    try {
      await changePassword.mutateAsync({
        currentPassword,
        newPassword,
      });
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      toast.success(t("settings.account.password_updated"));
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("settings.account.password_change_failed")
      );
    }
  }

  async function handleGenerate2FA() {
    try {
      const res = await generate2FA();
      setTwoFaSecret(res.secret);
      setTwoFaUri(res.uri);
      toast.success(t("settings.account.twofa_scan"));
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("settings.account.twofa_create_failed")
      );
    }
  }

  async function handleEnable2FA() {
    try {
      await enable2FA();
      toast.success(t("settings.account.twofa_enabled"));
      setTwoFaSecret("");
      setTwoFaUri("");
      reloadAccount();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("settings.account.twofa_enable_failed")
      );
    }
  }

  async function handleDisable2FA() {
    // Пароль спрашиваем здесь же: сервер без него откажет, а отдельного экрана
    // для одной строки заводить незачем.
    const password = window.prompt(t("settings.account.twofa_disable_password"));
    if (password === null) return;
    if (!password.trim()) {
      toast.error(t("settings.account.twofa_disable_password_required"));
      return;
    }
    try {
      await disable2FA(password);
      toast.success(t("settings.account.twofa_disabled"));
      reloadAccount();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("settings.account.twofa_disable_failed")
      );
    }
  }

  async function handleLink(provider: string) {
    try {
      const res = await socialLinkRedirect(provider, TENANT_SLUG);
      if (!res.url) {
        toast.error(t("settings.account.provider_not_configured"));
        return;
      }
      sessionStorage.setItem("social_auth_provider", provider);
      sessionStorage.setItem("social_auth_link", "true");
      window.location.href = res.url;
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("settings.account.link_failed"));
    }
  }

  async function handleTelegramAuth(tgUser: TelegramAuthUser) {
    try {
      await telegramLinkAccount(tgUser);
      toast.success(t("settings.account.telegram_linked"));
      reloadAccount();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("settings.account.telegram_link_failed")
      );
    }
  }

  async function handleUnlink(provider: string) {
    try {
      await socialUnlink(provider);
      toast.success(t("settings.account.social_linked"));
      reloadAccount();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("settings.account.unlink_failed"));
    }
  }

  async function handleAvatarChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 4 * 1024 * 1024) {
      toast.error(t("settings.account.avatar_too_big"));
      return;
    }
    setAvatarUploading(true);
    try {
      await uploadAccountAvatar(file);
      toast.success(t("settings.account.avatar_updated"));
      reloadAccount();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("settings.account.avatar_upload_failed")
      );
    } finally {
      setAvatarUploading(false);
      e.target.value = "";
    }
  }

  async function handleEmailChange() {
    if (!newEmail.includes("@")) {
      toast.error(t("settings.account.email_invalid"));
      return;
    }
    if (!emailPassword) {
      toast.error(t("settings.account.password_current_required"));
      return;
    }
    try {
      const res = await changeAccountEmail({
        email: newEmail,
        current_password: emailPassword,
      });
      setEmailPassword("");
      toast.success(t("settings.account.email_updated"));
      if (res.verification_url) {
        toast.info(t("settings.account.email_dev_link"));
      }
      reloadAccount();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("settings.account.email_change_failed")
      );
    }
  }

  async function handleDestroySession(sessionId?: string) {
    setSessionAction(true);
    try {
      await destroyAccountSession(
        sessionId ? { session_id: sessionId } : { all_others: true }
      );
      toast.success(
        sessionId
          ? t("settings.account.session_closed")
          : t("settings.account.sessions_others_closed")
      );
      qc.invalidateQueries({ queryKey: ["account-sessions"] });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.error"));
    } finally {
      setSessionAction(false);
    }
  }

  const linked = new Set(user?.linked_providers ?? []);
  const configured = new Set(providersQuery.data?.providers ?? []);
  const availableProviders = SOCIAL_PROVIDERS.filter((p) =>
    configured.has(p.key)
  );
  const telegramBot = providersQuery.data?.telegram?.bot_username ?? "";
  const telegramAvailable =
    Boolean(providersQuery.data?.telegram?.enabled) && telegramBot !== "";
  const telegramLinked = linked.has("telegram");
  const sessions = sessionsQuery.data?.sessions ?? [];
  const avatarInitial =
    (user?.display_name || user?.first_name || user?.email || "U")
      .charAt(0)
      .toUpperCase();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-16 text-muted-foreground">
        <Loader2 className="mr-2 h-5 w-5 animate-spin" />
        {t("settings.account.loading")}
      </div>
    );
  }

  return (
    <div className="space-y-[22px]">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div className="flex flex-col gap-1.5">
          <h1 className="text-[26px] font-bold leading-none tracking-[-0.02em]">
            {t("common.settings")}
          </h1>
          <p className="text-[13.5px] text-muted-foreground">
            {t("settings.account.subtitle")}
          </p>
        </div>
        <div className="flex items-center gap-2 text-[12.5px] text-muted-foreground">
          <i className="ri-check-line" />
          <span>{t("settings.account.save_hint")}</span>
        </div>
      </div>

      <div className="flex gap-1 overflow-x-auto rounded-xl border border-border bg-card p-1 [scrollbar-width:none]">
        {ACCOUNT_TABS.map((tabItem) => (
          <button
            key={tabItem.id}
            type="button"
            onClick={() => setTab(tabItem.id)}
            className={`flex h-9 shrink-0 items-center gap-[7px] whitespace-nowrap rounded-[9px] px-3.5 text-[13px] transition-colors ${
              tab === tabItem.id
                ? "bg-muted font-medium text-foreground"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <i className={tabItem.icon} />
            {t(tabItem.labelKey)}
          </button>
        ))}
      </div>

      <div>
        <div className="space-y-4">
        {tab === "profile" && (
          <div className="rounded-2xl border border-border bg-card p-5">
            <div className="mb-4 flex flex-wrap items-center gap-4">
              <div className="relative h-16 w-16 shrink-0 overflow-hidden rounded-full border border-border bg-muted">
                {user?.avatar_url ? (
                  <img
                    src={user.avatar_url}
                    alt="Avatar"
                    className="h-full w-full object-cover"
                  />
                ) : (
                  <span className="flex h-full w-full items-center justify-center text-lg font-semibold text-foreground">
                    {avatarInitial}
                  </span>
                )}
              </div>
              <div>
                <Label htmlFor="avatar" className="cursor-pointer">
                  <span className="inline-flex items-center rounded-md border border-border bg-muted px-3 py-1.5 text-xs font-medium hover:bg-muted/80">
                    {avatarUploading
                      ? t("common.loading")
                      : t("settings.account.avatar_upload")}
                  </span>
                  <input
                    id="avatar"
                    type="file"
                    accept="image/jpeg,image/png,image/webp"
                    className="sr-only"
                    disabled={avatarUploading}
                    onChange={handleAvatarChange}
                  />
                </Label>
                <p className="mt-1 text-xs text-muted-foreground">
                  {t("settings.account.avatar_hint")}
                </p>
              </div>
            </div>
            <div className="mb-4 flex flex-wrap items-center gap-2">
              <span className="text-sm text-muted-foreground">{user?.email}</span>
              {user?.email_verified ? (
                <Badge variant="secondary" className="text-emerald-600">
                  {t("settings.account.email_verified")}
                </Badge>
              ) : (
                <Badge variant="outline" className="text-amber-600">
                  {t("settings.account.email_unverified")}{" "}
                  <Link href="/verify-email" className="underline">
                    {t("settings.account.email_verify_link")}
                  </Link>
                </Badge>
              )}
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="firstName">{t("settings.account.first_name")}</Label>
                <Input
                  id="firstName"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="lastName">{t("settings.account.last_name")}</Label>
                <Input
                  id="lastName"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                />
              </div>
              <div className="space-y-2 sm:col-span-2">
                <Label htmlFor="displayName">{t("settings.account.display_name")}</Label>
                <Input
                  id="displayName"
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="phone">{t("settings.account.phone")}</Label>
                <Input
                  id="phone"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="locale">{t("settings.account.language")}</Label>
                <select
                  id="locale"
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  value={locale}
                  onChange={(e) => setLocale(e.target.value)}
                >
                  <option value="ru">{t("settings.account.language_ru")}</option>
                  <option value="en">{t("settings.account.language_en")}</option>
                </select>
              </div>
            </div>
            <Button
              className="mt-4"
              onClick={() => saveProfile.mutate()}
              disabled={saveProfile.isPending}
            >
              {saveProfile.isPending
                ? t("common.saving")
                : t("settings.account.save_profile")}
            </Button>
          </div>
        )}

        {tab === "contacts" && (
          <div className="rounded-2xl border border-border bg-card p-5">
            <h3 className="text-sm font-semibold">{t("settings.account.tab_contacts")}</h3>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("settings.account.contacts_hint")}
            </p>
            <div className="mt-4 grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="newEmail">{t("settings.account.new_email")}</Label>
                <Input
                  id="newEmail"
                  type="email"
                  value={newEmail}
                  onChange={(e) => setNewEmail(e.target.value)}
                  autoComplete="email"
                />
              </div>
              <div className="space-y-2">
                <Label>{t("settings.account.password_current")}</Label>
                <PasswordInput
                  value={emailPassword}
                  onChange={(e) => setEmailPassword(e.target.value)}
                  autoComplete="current-password"
                />
              </div>
              <div className="md:col-span-2">
                <Button onClick={handleEmailChange}>
                  {t("settings.account.update_email")}
                </Button>
              </div>
            </div>
          </div>
        )}

        {tab === "security" && (
          <div className="space-y-4">
          <div className="rounded-2xl border border-border bg-card p-5">
            <h3 className="text-sm font-semibold">{t("settings.account.social_title")}</h3>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("settings.account.social_hint")}
            </p>
            {providersQuery.isLoading ? (
              <div className="mt-4 flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                {t("settings.account.social_checking")}
              </div>
            ) : availableProviders.length === 0 && !telegramAvailable ? (
              <p className="mt-4 rounded-xl border border-dashed border-border bg-muted/30 p-4 text-sm text-muted-foreground">
                {t("settings.account.social_none")}
              </p>
            ) : (
              <div className="mt-4 grid gap-3 sm:grid-cols-3">
                {availableProviders.map((p) => (
                  <div
                    key={p.key}
                    className="flex items-center justify-between gap-2 rounded-xl border border-border p-3"
                  >
                    <span className="flex items-center gap-2 text-sm font-medium">
                      <i className={p.icon} />
                      {p.label}
                    </span>
                    {linked.has(p.key) ? (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleUnlink(p.key)}
                      >
                        {t("settings.account.social_unlink")}
                      </Button>
                    ) : (
                      <Button size="sm" onClick={() => handleLink(p.key)}>
                        {t("settings.account.social_link")}
                      </Button>
                    )}
                  </div>
                ))}

                {telegramAvailable && (
                  <div className="flex items-center justify-between gap-2 rounded-xl border border-border p-3">
                    <span className="flex items-center gap-2 text-sm font-medium">
                      <i className="ri-telegram-line" />
                      Telegram
                    </span>
                    {telegramLinked ? (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleUnlink("telegram")}
                      >
                        {t("settings.account.social_unlink")}
                      </Button>
                    ) : (
                      <TelegramLoginButton
                        botUsername={telegramBot}
                        buttonSize="small"
                        onAuth={handleTelegramAuth}
                      />
                    )}
                  </div>
                )}
              </div>
            )}
          </div>

          <div className="rounded-2xl border border-border bg-card p-5">
            <h3 className="text-sm font-semibold">
              {t("settings.account.password_section")}
            </h3>
            <div className="mt-4 grid max-w-md gap-3">
              <div className="space-y-2">
                <Label>{t("settings.account.password_current")}</Label>
                <PasswordInput
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  autoComplete="current-password"
                />
              </div>
              <div className="space-y-2">
                <Label>{t("settings.account.password_new")}</Label>
                <PasswordInput
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  autoComplete="new-password"
                />
              </div>
              <div className="space-y-2">
                <Label>{t("settings.account.password_confirm")}</Label>
                <PasswordInput
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  autoComplete="new-password"
                />
              </div>
              <Button
                onClick={handlePassword}
                disabled={changePassword.isPending}
              >
                {changePassword.isPending
                  ? t("common.saving")
                  : t("settings.account.password_update")}
              </Button>
            </div>
          </div>
          </div>
        )}

        {tab === "sessions" && (
          <div className="rounded-2xl border border-border bg-card p-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 className="text-sm font-semibold">
                  {t("settings.account.sessions_title")}
                </h3>
                <p className="mt-1 text-xs text-muted-foreground">
                  {t("settings.account.sessions_hint")}
                </p>
              </div>
              {sessions.length > 1 && (
                <Button
                  variant="outline"
                  size="sm"
                  disabled={sessionAction}
                  onClick={() => handleDestroySession()}
                >
                  {t("settings.account.sessions_close_others")}
                </Button>
              )}
            </div>

            {sessionsQuery.isLoading ? (
              <div className="flex items-center justify-center py-8 text-muted-foreground">
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                {t("settings.account.sessions_loading")}
              </div>
            ) : sessions.length === 0 ? (
              <p className="mt-4 rounded-xl border border-dashed border-border bg-muted/30 p-4 text-sm text-muted-foreground">
                {t("settings.account.sessions_empty")}
              </p>
            ) : (
              <div className="mt-4 space-y-2">
                {sessions.map((s) => (
                  <div
                    key={s.id}
                    className="rounded-xl border border-border bg-muted/30 px-4 py-3"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="truncate text-sm font-medium">
                        {(
                          s.user_agent || t("settings.account.session_unknown_device")
                        ).slice(0, 48)}
                      </span>
                      {s.is_current ? (
                        <Badge variant="secondary" className="text-emerald-600">
                          {t("settings.account.session_current")}
                        </Badge>
                      ) : (
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={sessionAction}
                          onClick={() => handleDestroySession(s.id)}
                        >
                          {t("settings.account.session_close")}
                        </Button>
                      )}
                    </div>
                    <div className="mt-1 flex flex-wrap justify-between gap-2 text-xs text-muted-foreground">
                      <span>IP: {s.ip_address || "—"}</span>
                      <span>
                        {t("settings.account.session_activity")}{" "}
                        {s.last_activity
                          ? new Date(s.last_activity).toLocaleString(localeTag())
                          : "—"}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {tab === "2fa" && (
          <div className="rounded-2xl border border-border bg-card p-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 className="text-sm font-semibold">
                  {t("settings.account.twofa_title")}
                </h3>
                <p className="mt-1 text-xs text-muted-foreground">
                  {user?.two_factor_enabled
                    ? t("settings.account.twofa_on_desc")
                    : t("settings.account.twofa_off_desc")}
                </p>
              </div>
              {user?.two_factor_enabled ? (
                <Button variant="destructive" onClick={handleDisable2FA}>
                  {t("settings.account.twofa_disable")}
                </Button>
              ) : (
                <Button onClick={handleGenerate2FA}>
                  {t("settings.account.twofa_setup")}
                </Button>
              )}
            </div>

            {twoFaSecret && !user?.two_factor_enabled && (
              <div className="mt-4 space-y-3 rounded-xl border border-dashed border-border bg-muted/30 p-4">
                <p className="text-sm">{t("settings.account.twofa_secret_label")}</p>
                <code className="block break-all rounded-lg bg-muted px-3 py-2 text-xs">
                  {twoFaSecret}
                </code>
                {twoFaUri && (
                  <p className="break-all text-xs text-muted-foreground">
                    URI: {twoFaUri}
                  </p>
                )}
                <Button onClick={handleEnable2FA}>
                  {t("settings.account.twofa_confirm")}
                </Button>
              </div>
            )}
          </div>
        )}

        {tab === "appearance" && (
          <div className="space-y-4">
            <AppearanceForm />
            <DisplayForm />
          </div>
        )}

        {tab === "notifications" && <NotificationsForm />}
        </div>
      </div>
    </div>
  );
}
