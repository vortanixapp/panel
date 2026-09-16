"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { AuthError, AuthField, AuthHeading, AuthSubmit, AuthSwitch } from "@/components/auth/auth-kit";
import { adoptSession, challenge2FA } from "@/lib/api";
import { postLoginPath } from "@/lib/auth-redirect";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

function buildSchema(t: TranslateFn) {
  return z.object({
    code: z.string().min(6, t("auth.two_factor.code_min")).max(8),
  });
}

type FormValues = z.infer<ReturnType<typeof buildSchema>>;

export function TwoFactorChallengeForm() {
  const t = useT();
  const router = useRouter();
  const queryClient = useQueryClient();
  const searchParams = useSearchParams();
  const twoFactorToken = searchParams.get("token") ?? "";
  const [error, setError] = useState("");

  const form = useForm<FormValues>({
    resolver: zodResolver(buildSchema(t)),
    defaultValues: { code: "" },
  });

  const loading = form.formState.isSubmitting;

  async function onSubmit(values: FormValues) {
    if (!twoFactorToken) {
      setError(t("auth.two_factor.no_token"));
      return;
    }
    setError("");
    try {
      const res = await challenge2FA(twoFactorToken, values.code);
      queryClient.clear();
      adoptSession();
      router.push(postLoginPath(res.user?.role ?? "user"));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.two_factor.invalid_code"));
    }
  }

  return (
    <>
      <AuthHeading title={t("auth.two_factor.title")} subtitle={t("auth.two_factor.subtitle")} />
      <AuthError message={error} />
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-5" noValidate>
        <AuthField
          id="code"
          label={t("auth.two_factor.code_label")}
          inputMode="numeric"
          autoComplete="one-time-code"
          autoFocus
          maxLength={8}
          placeholder="000000"
          disabled={loading}
          error={form.formState.errors.code?.message}
          className="[&_input]:h-14 [&_input]:text-center [&_input]:font-mono [&_input]:text-[22px] [&_input]:tracking-[0.45em]"
          {...form.register("code")}
        />
        <div className="pt-1">
          <AuthSubmit loading={loading} disabled={!twoFactorToken}>
            {t("common.confirm")}
          </AuthSubmit>
        </div>
      </form>
      <AuthSwitch>
        <Link href="/login" className="font-medium text-foreground underline-offset-4 hover:underline">
          {t("common.back")}
        </Link>
      </AuthSwitch>
    </>
  );
}
