"use client";

import { isEditorFrame } from "@/lib/site/frame";
import Link from "next/link";
import { useQueryClient } from "@tanstack/react-query";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  fetchSocialProviders,
  login,
  adoptSession,
  hasSession,
  socialRedirect,
  telegramLogin,
  tenantSlug,
  type TelegramAuthUser,
} from "@/lib/api";
import { postLoginPath } from "@/lib/auth-redirect";
import {
  AuthCheckbox,
  AuthError,
  AuthField,
  AuthHeading,
  AuthSocial,
  AuthSubmit,
  AuthSwitch,
  PasswordField,
  SOCIAL_BUTTONS,
} from "@/components/auth/auth-kit";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

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
    if (!addingAccount && hasSession() && !isEditorFrame()) router.replace("/dashboard");
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
      queryClient.clear();
      adoptSession();
      if (!hasSession()) {
        throw new Error(t("auth.login.no_token"));
      }
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
      queryClient.clear();
      adoptSession();
      if (!hasSession()) {
        throw new Error(t("auth.error.token_failed"));
      }
      router.push(res.redirect ?? postLoginPath(res.user?.role ?? "user"));
    } catch (err) {
      setError(
        err instanceof Error ? err.message : t("auth.login.telegram_failed")
      );
    }
  };

  const socialButtons = SOCIAL_BUTTONS.filter((b) => configured.includes(b.key));

  return (
    <>
      <AuthHeading
        title={addingAccount ? t("auth.login.add_account_title") : t("auth.login.title")}
        subtitle={addingAccount ? t("auth.login.add_account_subtitle") : t("auth.login.subtitle")}
      >
        {addingAccount && (
          <Link
            href="/dashboard"
            className="mt-4 inline-block text-[14px] text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline"
          >
            {t("auth.login.back_to_panel")}
          </Link>
        )}
      </AuthHeading>

      <AuthError message={error} />

      <form onSubmit={onSubmit} className="flex flex-col gap-5" noValidate>
        <AuthField
          id="email"
          type="email"
          label={t("common.email")}
          autoComplete="email"
          autoFocus
          placeholder="you@example.com"
          error={errors.email?.message}
          {...register("email")}
        />
        <PasswordField
          id="password"
          label={t("common.password")}
          autoComplete="current-password"
          placeholder="••••••••"
          error={errors.password?.message}
          aside={
            <Link
              href="/forgot-password"
              className="text-[13px] text-muted-foreground underline-offset-4 transition-colors hover:text-foreground hover:underline"
            >
              {t("auth.login.forgot")}
            </Link>
          }
          {...register("password")}
        />
        <AuthCheckbox {...register("remember")}>{t("auth.login.remember")}</AuthCheckbox>
        <div className="pt-1">
          <AuthSubmit loading={loading}>
            {loading ? t("auth.login.submitting") : t("auth.login.submit")}
          </AuthSubmit>
        </div>
      </form>

      <AuthSocial
        providers={socialButtons}
        telegramBot={telegramBot}
        onProvider={(provider) => void handleSocial(provider)}
        onTelegram={handleTelegramAuth}
      />

      <AuthSwitch>
        {t("auth.login.no_account")}{" "}
        <Link href="/register" className="font-medium text-foreground underline-offset-4 hover:underline">
          {t("auth.login.create_account")}
        </Link>
      </AuthSwitch>
    </>
  );
}
