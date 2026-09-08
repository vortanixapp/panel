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
import { PasswordInput } from "@/components/password-input";
import { resetPassword } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

// Схема собирается в рендере: на уровне модуля тексты ошибок застыли бы
// на языке, который был активен в момент загрузки бандла.
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
    <Card className="gap-4">
      <CardHeader>
        <CardTitle className="text-lg tracking-tight">{t("auth.reset.title")}</CardTitle>
        <CardDescription>
          {t("auth.reset.subtitle")}{" "}
          <Link href="/login" className="underline underline-offset-4 hover:text-primary">
            {t("auth.reset.login_link")}
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
              name="password"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("auth.reset.password_label")}</FormLabel>
                  <FormControl>
                    <PasswordInput autoComplete="new-password" disabled={loading} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="confirm"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("auth.reset.confirm_label")}</FormLabel>
                  <FormControl>
                    <PasswordInput autoComplete="new-password" disabled={loading} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button type="submit" disabled={loading || !token}>
              {loading && <Loader2 className="animate-spin" />}
              {t("auth.reset.submit")}
            </Button>
          </form>
        </Form>
      </CardContent>
    </Card>
  );
}
