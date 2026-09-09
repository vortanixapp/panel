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
import { challenge2FA, setTokens } from "@/lib/api";
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
      setTokens(res.access_token, res.refresh_token);
      router.push(postLoginPath(res.user?.role ?? "user"));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.two_factor.invalid_code"));
    }
  }

  return (
    <Card className="gap-4">
      <CardHeader>
        <CardTitle className="text-lg tracking-tight">
          {t("auth.two_factor.title")}
        </CardTitle>
        <CardDescription>
          {t("auth.two_factor.subtitle")}{" "}
          <Link href="/login" className="underline underline-offset-4 hover:text-primary">
            {t("common.back")}
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
              name="code"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("auth.two_factor.code_label")}</FormLabel>
                  <FormControl>
                    <Input
                      inputMode="numeric"
                      autoComplete="one-time-code"
                      disabled={loading}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button type="submit" disabled={loading || !twoFactorToken}>
              {loading && <Loader2 className="animate-spin" />}
              {t("common.confirm")}
            </Button>
          </form>
        </Form>
      </CardContent>
    </Card>
  );
}
