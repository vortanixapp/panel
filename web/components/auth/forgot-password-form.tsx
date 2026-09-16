"use client";

import Link from "next/link";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { AnimatePresence, m } from "motion/react";
import { MailCheck } from "lucide-react";
import { AuthError, AuthField, AuthHeading, AuthSubmit, AuthSwitch } from "@/components/auth/auth-kit";
import { EASE_OUT } from "@/components/landing/motion";
import { forgotPassword, tenantSlug } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

function buildSchema(t: TranslateFn) {
  return z.object({
    email: z.email({ error: t("auth.error.email_invalid") }),
  });
}

type FormValues = z.infer<ReturnType<typeof buildSchema>>;

export function ForgotPasswordForm() {
  const t = useT();
  const [error, setError] = useState("");
  const [sent, setSent] = useState(false);

  const form = useForm<FormValues>({
    resolver: zodResolver(buildSchema(t)),
    defaultValues: { email: "" },
  });

  const loading = form.formState.isSubmitting;

  async function onSubmit(values: FormValues) {
    setError("");
    try {
      await forgotPassword(values.email, tenantSlug());
      setSent(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.forgot.send_failed"));
    }
  }

  return (
    <AnimatePresence mode="wait" initial={false}>
      {sent ? (
        <m.div
          key="sent"
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, ease: EASE_OUT }}
        >
          <m.div
            initial={{ scale: 0.6, rotate: -12, opacity: 0 }}
            animate={{ scale: 1, rotate: 0, opacity: 1 }}
            transition={{ type: "spring", stiffness: 260, damping: 16, delay: 0.1 }}
            className="mb-7 flex size-14 items-center justify-center rounded-2xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
          >
            <MailCheck className="size-6" />
          </m.div>
          <AuthHeading title={t("auth.forgot.sent_title")} subtitle={t("auth.forgot.sent_text")} />
          <Link
            href="/login"
            className="flex h-12 w-full items-center justify-center rounded-full border border-border text-[15px] font-medium transition-colors hover:border-foreground/40 hover:bg-accent"
          >
            {t("auth.back_to_login")}
          </Link>
        </m.div>
      ) : (
        <m.div key="form" exit={{ opacity: 0, y: -12 }} transition={{ duration: 0.25 }}>
          <AuthHeading title={t("auth.forgot.title")} subtitle={t("auth.forgot.subtitle")} />
          <AuthError message={error} />
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-5" noValidate>
            <AuthField
              id="email"
              type="email"
              label={t("common.email")}
              autoComplete="email"
              autoFocus
              placeholder="you@example.com"
              disabled={loading}
              error={form.formState.errors.email?.message}
              {...form.register("email")}
            />
            <div className="pt-1">
              <AuthSubmit loading={loading}>{t("common.send")}</AuthSubmit>
            </div>
          </form>
          <AuthSwitch>
            <Link href="/login" className="font-medium text-foreground underline-offset-4 hover:underline">
              {t("auth.back_to_login")}
            </Link>
          </AuthSwitch>
        </m.div>
      )}
    </AnimatePresence>
  );
}
