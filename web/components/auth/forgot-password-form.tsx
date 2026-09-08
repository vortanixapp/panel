"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { AlertCircle, Loader2 } from "lucide-react";
import { useState } from "react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { forgotPassword, tenantSlug } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

// Схема собирается в рендере: на уровне модуля текст ошибки застыл бы
// на языке, который был активен в момент загрузки бандла.
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

  if (sent) {
    return (
      <Card className="gap-4">
        <CardHeader>
          <CardTitle className="text-lg tracking-tight">
            {t("auth.forgot.sent_title")}
          </CardTitle>
          <CardDescription>{t("auth.forgot.sent_text")}</CardDescription>
        </CardHeader>
        <CardContent>
          <Button asChild variant="outline" className="w-full">
            <Link href="/login">{t("auth.back_to_login")}</Link>
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="gap-4">
      <CardHeader>
        <CardTitle className="text-lg tracking-tight">{t("auth.forgot.title")}</CardTitle>
        <CardDescription>
          {t("auth.forgot.subtitle")}{" "}
          <Link href="/login" className="underline underline-offset-4 hover:text-primary">
            {t("auth.forgot.login_link")}
          </Link>
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="grid gap-4">
            {error && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("common.email")}</FormLabel>
                  <FormControl>
                    <Input type="email" autoComplete="email" disabled={loading} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button type="submit" disabled={loading}>
              {loading && <Loader2 className="animate-spin" />}
              {t("common.send")}
            </Button>
          </form>
        </Form>
      </CardContent>
    </Card>
  );
}
