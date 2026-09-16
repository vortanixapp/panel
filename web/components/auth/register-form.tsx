"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
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
import { useBrand } from "@/context/brand-provider";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

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

const docLinkClass = "text-foreground underline underline-offset-4 decoration-foreground/30 transition-colors hover:decoration-foreground";

export function RegisterForm() {
  const t = useT();
  const router = useRouter();
  const [error, setError] = useState("");
  const [configured, setConfigured] = useState<string[]>([]);
  const [telegramBot, setTelegramBot] = useState("");
  const [loading, setLoading] = useState(false);
  const { legal } = useBrand();
  const [acceptTerms, setAcceptTerms] = useState(false);
  const [acceptPersonalData, setAcceptPersonalData] = useState(false);
  const needTerms = Boolean(legal?.registration.terms);
  const needPersonalData = Boolean(legal?.registration.personal_data);
  const publishedDocs = new Set((legal?.documents ?? []).map((doc) => doc.kind));

  const {
    register,
    handleSubmit,
    watch,
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

  const password = watch("password");

  const onSubmit = handleSubmit(async (values) => {
    setError("");
    if ((needTerms && !acceptTerms) || (needPersonalData && !acceptPersonalData)) {
      setError(t("auth.register.consents_required"));
      return;
    }
    setLoading(true);
    try {
      const res = await registerUser(
        values.email,
        values.password,
        tenantSlug(),
        { name: values.name, lastName: values.lastName },
        { terms: acceptTerms, personalData: acceptPersonalData }
      );
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

  return (
    <>
      <AuthHeading title={t("auth.register.title")} subtitle={t("auth.register.subtitle")} />

      <AuthError message={error} />

      <form onSubmit={onSubmit} className="flex flex-col gap-5" noValidate>
        <div className="grid gap-5 sm:grid-cols-2 sm:gap-3">
          <AuthField
            id="name"
            type="text"
            label={t("auth.register.name_label")}
            autoComplete="given-name"
            autoFocus
            placeholder={t("auth.register.name_placeholder")}
            error={errors.name?.message}
            {...register("name")}
          />
          <AuthField
            id="last_name"
            type="text"
            label={t("auth.register.last_name_label")}
            autoComplete="family-name"
            placeholder={t("auth.register.last_name_placeholder")}
            error={errors.lastName?.message}
            {...register("lastName")}
          />
        </div>
        <AuthField
          id="email"
          type="email"
          label={t("common.email")}
          autoComplete="email"
          placeholder="you@example.com"
          error={errors.email?.message}
          {...register("email")}
        />
        <PasswordField
          id="password"
          label={t("common.password")}
          autoComplete="new-password"
          placeholder="••••••••"
          strengthOf={password}
          error={errors.password?.message}
          {...register("password")}
        />
        <PasswordField
          id="password_confirmation"
          label={t("auth.register.password_confirm_label")}
          autoComplete="new-password"
          placeholder="••••••••"
          error={errors.passwordConfirmation?.message}
          {...register("passwordConfirmation")}
        />

        {(needTerms || needPersonalData) && (
          <div className="flex flex-col gap-3 pt-1">
            {needTerms && (
              <AuthCheckbox
                checked={acceptTerms}
                onChange={(e) => setAcceptTerms(e.target.checked)}
              >
                {t("auth.register.accept_terms")}{" "}
                {publishedDocs.has("offer") && (
                  <a href="/legal/offer" target="_blank" rel="noopener noreferrer" className={docLinkClass}>
                    {t("auth.register.terms_link")}
                  </a>
                )}
                {publishedDocs.has("offer") && publishedDocs.has("privacy")
                  ? ` ${t("auth.register.terms_and")} `
                  : null}
                {publishedDocs.has("privacy") && (
                  <a href="/legal/privacy" target="_blank" rel="noopener noreferrer" className={docLinkClass}>
                    {t("auth.register.privacy_link")}
                  </a>
                )}
              </AuthCheckbox>
            )}
            {needPersonalData && (
              <AuthCheckbox
                checked={acceptPersonalData}
                onChange={(e) => setAcceptPersonalData(e.target.checked)}
              >
                {t("auth.register.accept_personal_data")}{" "}
                <a href="/legal/consent" target="_blank" rel="noopener noreferrer" className={docLinkClass}>
                  {t("auth.register.consent_link")}
                </a>
              </AuthCheckbox>
            )}
          </div>
        )}

        <div className="pt-1">
          <AuthSubmit loading={loading}>
            {loading ? t("auth.register.submitting") : t("auth.register.submit")}
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
        {t("auth.register.have_account")}{" "}
        <Link href="/login" className="font-medium text-foreground underline-offset-4 hover:underline">
          {t("auth.register.login_link")}
        </Link>
      </AuthSwitch>
    </>
  );
}
