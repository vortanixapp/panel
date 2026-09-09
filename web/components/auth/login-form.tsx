"use client";

import Link from "next/link";
import { useQueryClient } from "@tanstack/react-query";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { BrandLogo } from "@/components/brand-logo";
import {
  fetchSocialProviders,
  login,
  getAccessToken,
  setTokens,
  socialRedirect,
  telegramLogin,
  tenantSlug,
  type TelegramAuthUser,
} from "@/lib/api";
import { postLoginPath } from "@/lib/auth-redirect";
import { TelegramLoginButton } from "@/components/auth/telegram-login-button";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

const SOCIAL_BUTTONS = [
  { key: "vk", label: "VK", icon: "ri-vk-line" },
  { key: "discord", label: "Discord", icon: "ri-discord-line" },
  { key: "google", label: "Google", icon: "ri-google-line" },
] as const;

function buildSchema(t: TranslateFn) {
  return z.object({
    email: z.string().email(t("auth.error.email_invalid")),
    password: z.string().min(1, t("auth.error.password_required")),
    remember: z.boolean(),
  });
}

type FormValues = z.infer<ReturnType<typeof buildSchema>>;

export function LoginForm() {
  const t = useT();
  const router = useRouter();
  const queryClient = useQueryClient();
  const searchParams = useSearchParams();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [configured, setConfigured] = useState<string[]>([]);
  const [telegramBot, setTelegramBot] = useState("");

  const addingAccount = String(searchParams.get("add") || "") === "1";
  const prefillEmail = String(searchParams.get("email") || "").trim();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(buildSchema(t)),
    defaultValues: { email: prefillEmail, password: "", remember: false },
  });

  useEffect(() => {
    if (!addingAccount && getAccessToken()) router.replace("/dashboard");
  }, [router, addingAccount]);

  useEffect(() => {
    let cancelled = false;
    fetchSocialProviders()
      .then((res) => {
        if (cancelled) return;
        setConfigured(res.providers ?? []);
        setTelegramBot(
          res.telegram?.enabled ? res.telegram?.bot_username ?? "" : ""
        );
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const oauthError = String(searchParams.get("error") || "").trim();
    const twoFactorRequired = String(searchParams.get("two_factor_required") || "") === "1";
    const twoFactorToken = String(searchParams.get("two_factor_token") || "").trim();

    if (oauthError) {
      setError(oauthError);
    }

    if (twoFactorRequired && twoFactorToken) {
      router.replace(`/two-factor-challenge?token=${encodeURIComponent(twoFactorToken)}`);
    }
  }, [searchParams, router]);

  const onSubmit = handleSubmit(async (values) => {
    setError("");
    setLoading(true);
    try {
      const res = await login(values.email, values.password, tenantSlug(), values.remember);
      if (res.requires_2fa && res.two_factor_token) {
        router.push(
          `/two-factor-challenge?token=${encodeURIComponent(res.two_factor_token)}`
        );
        return;
      }
      if (!res.access_token) {
        throw new Error(t("auth.login.no_token"));
      }
      queryClient.clear();
      setTokens(res.access_token, res.refresh_token ?? "");
      router.push(postLoginPath(res.user?.role ?? "user"));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.login.failed"));
    } finally {
      setLoading(false);
    }
  });

  const handleSocial = async (provider: string) => {
    try {
      const res = await socialRedirect(provider, tenantSlug());
      const url = String(res?.url || "");
      if (url) {
        sessionStorage.setItem("social_auth_mode", "login");
        sessionStorage.setItem("social_auth_provider", provider);
        sessionStorage.removeItem("social_auth_link");
        window.location.href = url;
      }
    } catch {
      setError(t("auth.login.social_failed"));
    }
  };

  const handleTelegramAuth = async (tgUser: TelegramAuthUser) => {
    setError("");
    try {
      const res = await telegramLogin(tgUser, tenantSlug());
      if (!res.access_token) {
        throw new Error(t("auth.error.token_failed"));
      }
      queryClient.clear();
      setTokens(res.access_token, res.refresh_token ?? "");
      router.push(res.redirect ?? postLoginPath(res.user?.role ?? "user"));
    } catch (err) {
      setError(
        err instanceof Error ? err.message : t("auth.login.telegram_failed")
      );
    }
  };

  const socialButtons = SOCIAL_BUTTONS.filter((b) => configured.includes(b.key));
  const hasSocial = socialButtons.length > 0 || telegramBot !== "";

  return (
    <section className="relative min-h-screen flex items-center justify-center py-12 overflow-hidden bg-background">
      <div className="pointer-events-none absolute inset-0">
        <div className="absolute top-1/4 left-1/4 w-[500px] h-[500px] bg-primary/10 rounded-full blur-[120px] animate-pulse" />
        <div
          className="absolute bottom-1/4 right-1/4 w-[400px] h-[400px] bg-primary/5 rounded-full blur-[100px] animate-pulse"
          style={{ animationDelay: "1s" }}
        />
      </div>
      <div
        className="pointer-events-none absolute inset-0 opacity-[0.02]"
        style={{
          backgroundImage:
            "linear-gradient(to right, currentColor 1px, transparent 1px), linear-gradient(to bottom, currentColor 1px, transparent 1px)",
          backgroundSize: "60px 60px",
        }}
      />

      <div className="relative w-full max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid lg:grid-cols-2 gap-12 lg:gap-20 items-center">
          <div className="hidden lg:block">
            <div className="mb-8">
              <BrandLogo size="lg" className="h-12 max-w-[264px]" priority />
            </div>
            <h1 className="text-4xl lg:text-5xl font-bold text-foreground mb-6 leading-tight">
              {t("auth.login.welcome")}
            </h1>
            <p className="text-lg text-muted-foreground mb-10 max-w-md">
              {t("auth.login.welcome_text")}
            </p>
            <div className="space-y-4">
              <div className="flex items-center gap-4 p-4 bg-card/50 backdrop-blur-sm border border-border rounded-2xl transition-all duration-300 hover:bg-card hover:border-primary/30">
                <div className="w-12 h-12 flex items-center justify-center bg-primary/10 text-primary rounded-xl">
                  <i className="ri-shield-check-line text-xl" />
                </div>
                <div>
                  <div className="font-semibold text-foreground">
                    {t("auth.login.feature_secure_title")}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {t("auth.login.feature_secure_note")}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-4 p-4 bg-card/50 backdrop-blur-sm border border-border rounded-2xl transition-all duration-300 hover:bg-card hover:border-primary/30">
                <div className="w-12 h-12 flex items-center justify-center bg-emerald-500/10 text-emerald-500 rounded-xl">
                  <i className="ri-dashboard-3-line text-xl" />
                </div>
                <div>
                  <div className="font-semibold text-foreground">
                    {t("auth.login.feature_panel_title")}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {t("auth.login.feature_panel_note")}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-4 p-4 bg-card/50 backdrop-blur-sm border border-border rounded-2xl transition-all duration-300 hover:bg-card hover:border-primary/30">
                <div className="w-12 h-12 flex items-center justify-center bg-amber-500/10 text-amber-500 rounded-xl">
                  <i className="ri-customer-service-2-line text-xl" />
                </div>
                <div>
                  <div className="font-semibold text-foreground">
                    {t("auth.login.feature_support_title")}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {t("auth.login.feature_support_note")}
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="relative">
            <div className="absolute -inset-4 bg-gradient-to-r from-primary/20 via-primary/10 to-primary/20 rounded-[2rem] blur-2xl opacity-50" />
            <div className="relative bg-card border border-border rounded-3xl shadow-2xl overflow-hidden">
              <div className="h-1.5 bg-gradient-to-r from-primary/50 via-primary to-primary/50" />
              <div className="p-8 sm:p-10">
                <div className="mb-8 flex items-center justify-center lg:hidden">
                  <BrandLogo size="md" className="h-9 max-w-[220px]" priority />
                </div>

                <div className="text-center mb-8">
                  <h2 className="text-2xl sm:text-3xl font-bold text-foreground mb-2">
                    {addingAccount
                      ? t("auth.login.add_account_title")
                      : t("auth.login.title")}
                  </h2>
                  <p className="text-muted-foreground">
                    {addingAccount
                      ? t("auth.login.add_account_subtitle")
                      : t("auth.login.subtitle")}
                  </p>
                  {addingAccount && (
                    <Link
                      href="/dashboard"
                      className="mt-3 inline-block text-sm text-muted-foreground underline-offset-4 hover:underline"
                    >
                      {t("auth.login.back_to_panel")}
                    </Link>
                  )}
                </div>

                {error && (
                  <div className="mb-6 p-4 bg-rose-500/10 border border-rose-500/20 rounded-2xl">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 flex items-center justify-center bg-rose-500/20 text-rose-500 rounded-xl">
                        <i className="ri-error-warning-line text-xl" />
                      </div>
                      <div className="text-sm text-rose-400">{error}</div>
                    </div>
                  </div>
                )}

                <form onSubmit={onSubmit} className="space-y-5" noValidate>
                  <div>
                    <label
                      className="block text-sm font-medium text-foreground mb-2"
                      htmlFor="email"
                    >
                      <i className="ri-mail-line mr-1.5 text-muted-foreground" />
                      Email
                    </label>
                    <input
                      id="email"
                      type="email"
                      {...register("email")}
                      autoFocus
                      placeholder="your@email.com"
                      className={`w-full px-4 py-3.5 bg-muted/50 border rounded-xl text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all duration-200 ${
                        errors.email ? "border-rose-500" : "border-border"
                      }`}
                    />
                    {errors.email && (
                      <p className="mt-1.5 text-xs text-rose-500">{errors.email.message}</p>
                    )}
                  </div>
                  <div>
                    <label
                      className="block text-sm font-medium text-foreground mb-2"
                      htmlFor="password"
                    >
                      <i className="ri-lock-line mr-1.5 text-muted-foreground" />
                      {t("common.password")}
                    </label>
                    <input
                      id="password"
                      type="password"
                      {...register("password")}
                      placeholder="••••••••"
                      className={`w-full px-4 py-3.5 bg-muted/50 border rounded-xl text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all duration-200 ${
                        errors.password ? "border-rose-500" : "border-border"
                      }`}
                    />
                    {errors.password && (
                      <p className="mt-1.5 text-xs text-rose-500">{errors.password.message}</p>
                    )}
                  </div>
                  <div className="flex items-center justify-between">
                    <label className="flex items-center gap-3 cursor-pointer group">
                      <input
                        {...register("remember")}
                        type="checkbox"
                        className="w-5 h-5 rounded-lg border-2 border-border bg-muted/50 text-primary focus:ring-primary/20 focus:ring-offset-0 transition-colors"
                      />
                      <span className="text-sm text-muted-foreground group-hover:text-foreground transition-colors">
                        {t("auth.login.remember")}
                      </span>
                    </label>
                    <Link
                      href="/forgot-password"
                      className="text-sm font-semibold text-primary hover:text-primary/80 transition-colors"
                    >
                      {t("auth.login.forgot")}
                    </Link>
                  </div>
                  <button
                    type="submit"
                    disabled={loading}
                    className="group relative w-full flex items-center justify-center gap-2 px-6 py-4 bg-primary text-primary-foreground font-semibold rounded-xl overflow-hidden transition-all duration-300 hover:shadow-xl hover:shadow-primary/25 hover:-translate-y-0.5 disabled:opacity-50"
                  >
                    <span className="absolute inset-0 bg-gradient-to-r from-white/0 via-white/20 to-white/0 -translate-x-full group-hover:translate-x-full transition-transform duration-700" />
                    <span className="relative">
                      {loading ? t("auth.login.submitting") : t("auth.login.submit")}
                    </span>
                    {!loading && <i className="ri-arrow-right-line relative" />}
                  </button>
                </form>

                {hasSocial && (
                  <>
                    <div className="relative my-8">
                      <div className="absolute inset-0 flex items-center">
                        <div className="w-full border-t border-border" />
                      </div>
                      <div className="relative flex justify-center text-xs uppercase">
                        <span className="bg-card px-4 text-muted-foreground">
                          {t("auth.or")}
                        </span>
                      </div>
                    </div>

                    <div className="grid gap-3 sm:grid-cols-2">
                      {socialButtons.map((b) => (
                        <button
                          key={b.key}
                          type="button"
                          onClick={() => handleSocial(b.key)}
                          className="inline-flex items-center justify-center gap-2 px-4 py-3 bg-muted/50 border border-border rounded-xl text-sm font-semibold text-foreground hover:bg-muted transition-colors"
                        >
                          <i className={b.icon} /> {b.label}
                        </button>
                      ))}
                    </div>

                    {telegramBot && (
                      <div className="mt-3 flex justify-center">
                        <TelegramLoginButton
                          botUsername={telegramBot}
                          onAuth={handleTelegramAuth}
                        />
                      </div>
                    )}
                  </>
                )}

                <div className="text-center mt-8">
                  <p className="text-muted-foreground">
                    {t("auth.login.no_account")}
                    <Link
                      href="/register"
                      className="font-semibold text-primary hover:text-primary/80 transition-colors ml-1"
                    >
                      {t("auth.login.create_account")}
                    </Link>
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
