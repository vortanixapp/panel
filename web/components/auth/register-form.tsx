"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { BrandLogo } from "@/components/brand-logo";
import {
  fetchSocialProviders,
  register as registerUser,
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

// Схема собирается в рендере: на уровне модуля тексты ошибок застыли бы
// на языке, который был активен в момент загрузки бандла.
function buildSchema(t: TranslateFn) {
  return z
    .object({
      name: z.string().min(1, t("auth.error.name_required")),
      lastName: z.string().optional(),
      email: z.string().email(t("auth.error.email_invalid")),
      password: z.string().min(8, t("auth.error.password_min")),
      passwordConfirmation: z
        .string()
        .min(1, t("auth.error.password_confirm_required")),
    })
    .refine((data) => data.password === data.passwordConfirmation, {
      message: t("auth.error.passwords_mismatch"),
      path: ["passwordConfirmation"],
    });
}

type FormValues = z.infer<ReturnType<typeof buildSchema>>;

export function RegisterForm() {
  const t = useT();
  const router = useRouter();
  const [error, setError] = useState("");
  const [configured, setConfigured] = useState<string[]>([]);
  const [telegramBot, setTelegramBot] = useState("");
  const [loading, setLoading] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(buildSchema(t)),
    defaultValues: {
      name: "",
      lastName: "",
      email: "",
      password: "",
      passwordConfirmation: "",
    },
  });

  const onSubmit = handleSubmit(async (values) => {
    setError("");
    setLoading(true);
    try {
      const res = await registerUser(values.email, values.password, tenantSlug(), {
        name: values.name,
        lastName: values.lastName,
      });
      setTokens(res.access_token, res.refresh_token);
      router.push(postLoginPath(res.user?.role ?? "user"));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.register.failed"));
    } finally {
      setLoading(false);
    }
  });

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

  const handleTelegramAuth = async (tgUser: TelegramAuthUser) => {
    setError("");
    try {
      const res = await telegramLogin(tgUser, tenantSlug());
      if (!res.access_token) {
        throw new Error(t("auth.error.token_failed"));
      }
      setTokens(res.access_token, res.refresh_token ?? "");
      router.push(res.redirect ?? postLoginPath(res.user?.role ?? "user"));
    } catch (err) {
      setError(
        err instanceof Error ? err.message : t("auth.register.telegram_failed")
      );
    }
  };

  const handleSocial = async (provider: string) => {
    try {
      const res = await socialRedirect(provider, tenantSlug());
      const url = String(res?.url || "");
      if (url) {
        sessionStorage.setItem("social_auth_mode", "register");
        sessionStorage.setItem("social_auth_provider", provider);
        sessionStorage.removeItem("social_auth_link");
        window.location.href = url;
      }
    } catch {
      setError(t("auth.register.social_failed"));
    }
  };

  const socialButtons = SOCIAL_BUTTONS.filter((b) => configured.includes(b.key));
  const hasSocial = socialButtons.length > 0 || telegramBot !== "";

  return (
    <section className="relative min-h-screen flex items-center justify-center py-12 overflow-hidden bg-background">
      <div className="pointer-events-none absolute inset-0">
        <div className="absolute top-1/4 right-1/4 w-[500px] h-[500px] bg-emerald-500/10 rounded-full blur-[120px] animate-pulse" />
        <div
          className="absolute bottom-1/4 left-1/4 w-[400px] h-[400px] bg-primary/5 rounded-full blur-[100px] animate-pulse"
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
              {t("auth.register.hero_title")}
              <span className="block text-emerald-500">
                {t("auth.register.hero_title_accent")}
              </span>
            </h1>
            <p className="text-lg text-muted-foreground mb-10 max-w-md">
              {t("auth.register.hero_text")}
            </p>
            <div className="space-y-4">
              <div className="flex items-center gap-4 p-4 bg-card/50 backdrop-blur-sm border border-border rounded-2xl transition-all duration-300 hover:bg-card hover:border-emerald-500/30">
                <div className="w-12 h-12 flex items-center justify-center bg-emerald-500/10 text-emerald-500 rounded-xl">
                  <i className="ri-rocket-line text-xl" />
                </div>
                <div>
                  <div className="font-semibold text-foreground">
                    {t("auth.register.feature_fast_title")}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {t("auth.register.feature_fast_note")}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-4 p-4 bg-card/50 backdrop-blur-sm border border-border rounded-2xl transition-all duration-300 hover:bg-card hover:border-emerald-500/30">
                <div className="w-12 h-12 flex items-center justify-center bg-primary/10 text-primary rounded-xl">
                  <i className="ri-money-dollar-circle-line text-xl" />
                </div>
                <div>
                  <div className="font-semibold text-foreground">
                    {t("auth.register.feature_prepay_title")}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {t("auth.register.feature_prepay_note")}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-4 p-4 bg-card/50 backdrop-blur-sm border border-border rounded-2xl transition-all duration-300 hover:bg-card hover:border-emerald-500/30">
                <div className="w-12 h-12 flex items-center justify-center bg-amber-500/10 text-amber-500 rounded-xl">
                  <i className="ri-gamepad-line text-xl" />
                </div>
                <div>
                  <div className="font-semibold text-foreground">
                    {t("auth.register.feature_games_title")}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {t("auth.register.feature_games_note")}
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="relative">
            <div className="absolute -inset-4 bg-gradient-to-r from-emerald-500/20 via-primary/10 to-emerald-500/20 rounded-[2rem] blur-2xl opacity-50" />
            <div className="relative bg-card border border-border rounded-3xl shadow-2xl overflow-hidden">
              <div className="h-1.5 bg-gradient-to-r from-emerald-500/50 via-emerald-500 to-emerald-500/50" />
              <div className="p-8 sm:p-10">
                <div className="mb-8 flex items-center justify-center lg:hidden">
                  <BrandLogo size="md" className="h-9 max-w-[220px]" priority />
                </div>

                <div className="text-center mb-8">
                  <h2 className="text-2xl sm:text-3xl font-bold text-foreground mb-2">
                    {t("auth.register.title")}
                  </h2>
                  <p className="text-muted-foreground">{t("auth.register.subtitle")}</p>
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
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div>
                      <label
                        className="block text-sm font-medium text-foreground mb-2"
                        htmlFor="name"
                      >
                        <i className="ri-user-line mr-1.5 text-muted-foreground" />{" "}
                        {t("auth.register.name_label")}
                      </label>
                      <input
                        id="name"
                        type="text"
                        {...register("name")}
                        autoFocus
                        placeholder={t("auth.register.name_placeholder")}
                        className={`w-full px-4 py-3.5 bg-muted/50 border rounded-xl text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 transition-all duration-200 ${
                          errors.name ? "border-rose-500" : "border-border"
                        }`}
                      />
                      {errors.name && (
                        <p className="mt-1.5 text-xs text-rose-500">{errors.name.message}</p>
                      )}
                    </div>
                    <div>
                      <label
                        className="block text-sm font-medium text-foreground mb-2"
                        htmlFor="last_name"
                      >
                        <i className="ri-user-line mr-1.5 text-muted-foreground" />{" "}
                        {t("auth.register.last_name_label")}
                      </label>
                      <input
                        id="last_name"
                        type="text"
                        {...register("lastName")}
                        placeholder={t("auth.register.last_name_placeholder")}
                        className={`w-full px-4 py-3.5 bg-muted/50 border rounded-xl text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 transition-all duration-200 ${
                          errors.lastName ? "border-rose-500" : "border-border"
                        }`}
                      />
                      {errors.lastName && (
                        <p className="mt-1.5 text-xs text-rose-500">{errors.lastName.message}</p>
                      )}
                    </div>
                  </div>
                  <div>
                    <label
                      className="block text-sm font-medium text-foreground mb-2"
                      htmlFor="email"
                    >
                      <i className="ri-mail-line mr-1.5 text-muted-foreground" /> Email
                    </label>
                    <input
                      id="email"
                      type="email"
                      {...register("email")}
                      placeholder="your@email.com"
                      className={`w-full px-4 py-3.5 bg-muted/50 border rounded-xl text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 transition-all duration-200 ${
                        errors.email ? "border-rose-500" : "border-border"
                      }`}
                    />
                    {errors.email && (
                      <p className="mt-1.5 text-xs text-rose-500">{errors.email.message}</p>
                    )}
                  </div>
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div>
                      <label
                        className="block text-sm font-medium text-foreground mb-2"
                        htmlFor="password"
                      >
                        <i className="ri-lock-line mr-1.5 text-muted-foreground" />{" "}
                        {t("common.password")}
                      </label>
                      <input
                        id="password"
                        type="password"
                        {...register("password")}
                        placeholder="••••••••"
                        className={`w-full px-4 py-3.5 bg-muted/50 border rounded-xl text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 transition-all duration-200 ${
                          errors.password ? "border-rose-500" : "border-border"
                        }`}
                      />
                      {errors.password && (
                        <p className="mt-1.5 text-xs text-rose-500">{errors.password.message}</p>
                      )}
                    </div>
                    <div>
                      <label
                        className="block text-sm font-medium text-foreground mb-2"
                        htmlFor="password_confirmation"
                      >
                        <i className="ri-lock-check-line mr-1.5 text-muted-foreground" />{" "}
                        {t("auth.register.password_confirm_label")}
                      </label>
                      <input
                        id="password_confirmation"
                        type="password"
                        {...register("passwordConfirmation")}
                        placeholder="••••••••"
                        className={`w-full px-4 py-3.5 bg-muted/50 border rounded-xl text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 transition-all duration-200 ${
                          errors.passwordConfirmation ? "border-rose-500" : "border-border"
                        }`}
                      />
                      {errors.passwordConfirmation && (
                        <p className="mt-1.5 text-xs text-rose-500">
                          {errors.passwordConfirmation.message}
                        </p>
                      )}
                    </div>
                  </div>

                  <div className="p-4 bg-muted/30 border border-border rounded-2xl">
                    <div className="flex items-start gap-3">
                      <div className="w-8 h-8 flex items-center justify-center bg-emerald-500/10 text-emerald-500 rounded-lg flex-shrink-0 mt-0.5">
                        <i className="ri-shield-check-line text-sm" />
                      </div>
                      <p className="text-xs text-muted-foreground leading-relaxed">
                        {t("auth.register.terms_prefix")}{" "}
                        <a href="#" className="text-primary hover:underline">
                          {t("auth.register.terms_link")}
                        </a>{" "}
                        {t("auth.register.terms_and")}{" "}
                        <a href="#" className="text-primary hover:underline">
                          {t("auth.register.privacy_link")}
                        </a>{" "}
                        {t("auth.register.terms_suffix")}
                      </p>
                    </div>
                  </div>

                  <button
                    type="submit"
                    disabled={loading}
                    className="group relative w-full flex items-center justify-center gap-2 px-6 py-4 bg-emerald-500 text-white font-semibold rounded-xl overflow-hidden transition-all duration-300 hover:bg-emerald-600 hover:shadow-xl hover:shadow-emerald-500/25 hover:-translate-y-0.5 disabled:opacity-50"
                  >
                    <span className="absolute inset-0 bg-gradient-to-r from-white/0 via-white/20 to-white/0 -translate-x-full group-hover:translate-x-full transition-transform duration-700" />
                    <i className="ri-rocket-line relative" />
                    <span className="relative">
                      {loading
                        ? t("auth.register.submitting")
                        : t("auth.register.submit")}
                    </span>
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
                    {t("auth.register.have_account")}
                    <Link
                      href="/login"
                      className="font-semibold text-primary hover:text-primary/80 transition-colors ml-1"
                    >
                      {t("auth.register.login_link")}
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
