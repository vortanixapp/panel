"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { AuthError, AuthHeading, AuthSubmit, AuthSwitch, PasswordField } from "@/components/auth/auth-kit";
import { resetPassword } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

function buildSchema(t: TranslateFn) {
  return z
    .object({
      password: z.string().min(8, t("auth.error.password_min")),
      confirm: z.string(),
    })
    .refine((v) => v.password === v.confirm, {
      message: t("auth.error.passwords_mismatch"),
      path: ["confirm"],
    });
}

type FormValues = z.infer<ReturnType<typeof buildSchema>>;

export function ResetPasswordForm() {
  const t = useT();
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token") ?? "";
  const [error, setError] = useState("");

  const form = useForm<FormValues>({
    resolver: zodResolver(buildSchema(t)),
    defaultValues: { password: "", confirm: "" },
  });

  const loading = form.formState.isSubmitting;
  const password = form.watch("password");

  async function onSubmit(values: FormValues) {
    if (!token) {
      setError(t("auth.reset.no_token"));
      return;
    }
    setError("");
    try {
      await resetPassword(token, values.password);
      router.push("/login");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.reset.failed"));
    }
  }

  return (
    <>
      <AuthHeading title={t("auth.reset.title")} subtitle={t("auth.reset.subtitle")} />
      <AuthError message={error} />
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-5" noValidate>
        <PasswordField
          id="password"
          label={t("auth.reset.password_label")}
          autoComplete="new-password"
          autoFocus
          placeholder="••••••••"
          disabled={loading}
          strengthOf={password}
          error={form.formState.errors.password?.message}
          {...form.register("password")}
        />
        <PasswordField
          id="confirm"
          label={t("auth.reset.confirm_label")}
          autoComplete="new-password"
          placeholder="••••••••"
          disabled={loading}
          error={form.formState.errors.confirm?.message}
          {...form.register("confirm")}
        />
        <div className="pt-1">
          <AuthSubmit loading={loading} disabled={!token}>
            {t("auth.reset.submit")}
          </AuthSubmit>
        </div>
      </form>
      <AuthSwitch>
        <Link href="/login" className="font-medium text-foreground underline-offset-4 hover:underline">
          {t("auth.reset.login_link")}
        </Link>
      </AuthSwitch>
    </>
  );
}
