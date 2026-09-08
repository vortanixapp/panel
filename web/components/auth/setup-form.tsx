"use client";

import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormDescription,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { PasswordInput } from "@/components/password-input";
import { AuthAside, AuthShell } from "@/components/auth/auth-shell";
import { bootstrapPanel, setTokens } from "@/lib/api";
import { postLoginPath } from "@/lib/auth-redirect";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

function buildSchema(t: TranslateFn) {
  return z.object({
    panelName: z.string().min(1, t("auth.setup.error_panel_required")),
    email: z.email({ error: t("auth.setup.error_email_invalid") }),
    password: z.string().min(8, t("auth.error.password_min")),
  });
}

type FormValues = z.infer<ReturnType<typeof buildSchema>>;

function buildAsideItems(t: TranslateFn) {
  return [
    {
      icon: "ri-shield-user-line",
      tone: "bg-emerald-500/10 text-emerald-500",
      title: t("auth.setup.aside_owner_title"),
      note: t("auth.setup.aside_owner_note"),
    },
    {
      icon: "ri-price-tag-3-line",
      tone: "bg-primary/10 text-primary",
      title: t("auth.setup.aside_name_title"),
      note: t("auth.setup.aside_name_note"),
    },
    {
      icon: "ri-server-line",
      tone: "bg-amber-500/10 text-amber-500",
      title: t("auth.setup.aside_local_title"),
      note: t("auth.setup.aside_local_note"),
    },
  ];
}

export function SetupForm() {
  const t = useT();
  const queryClient = useQueryClient();
  const [error, setError] = useState("");

  const form = useForm<FormValues>({
    resolver: zodResolver(buildSchema(t)),
    defaultValues: {
      panelName: "",
      email: "",
      password: "",
    },
  });

  const loading = form.formState.isSubmitting;

  async function onSubmit(values: FormValues) {
    setError("");
    try {
      const boot = await bootstrapPanel(values.email, values.password, values.panelName);
      queryClient.clear();
      setTokens(boot.access_token, boot.refresh_token);
      window.location.replace(postLoginPath("owner"));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.setup.failed"));
    }
  }

  return (
    <AuthShell
      title={t("auth.setup.title")}
      subtitle={t("auth.setup.subtitle")}
      aside={
        <AuthAside
          title={t("auth.setup.aside_title")}
          text={t("auth.setup.aside_text")}
          items={buildAsideItems(t)}
        />
      }
    >
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="grid gap-5">
          {error && (
            <div className="p-4 bg-rose-500/10 border border-rose-500/20 rounded-2xl">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 shrink-0 flex items-center justify-center bg-rose-500/20 text-rose-500 rounded-xl">
                  <i className="ri-error-warning-line text-xl" />
                </div>
                <div className="text-sm text-rose-400">{error}</div>
              </div>
            </div>
          )}

          <FormField
            control={form.control}
            name="panelName"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t("auth.setup.panel_label")}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={t("auth.setup.panel_placeholder")}
                    disabled={loading}
                    {...field}
                  />
                </FormControl>
                <FormDescription>{t("auth.setup.panel_hint")}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className="pt-1 border-t border-border" />

          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t("auth.setup.owner_email_label")}</FormLabel>
                <FormControl>
                  <Input type="email" autoComplete="email" disabled={loading} {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="password"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t("auth.setup.owner_password_label")}</FormLabel>
                <FormControl>
                  <PasswordInput autoComplete="new-password" disabled={loading} {...field} />
                </FormControl>
                <FormDescription>{t("auth.setup.password_hint")}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <Button type="submit" size="lg" className="mt-2 h-12 text-base" disabled={loading}>
            {loading && <Loader2 className="animate-spin" />}
            {loading ? t("auth.setup.submitting") : t("auth.setup.submit")}
          </Button>
        </form>
      </Form>
    </AuthShell>
  );
}
