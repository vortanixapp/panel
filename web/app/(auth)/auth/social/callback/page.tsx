"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";
import { AlertCircle, Loader2 } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { setTokens, socialExchange } from "@/lib/api";
import { postLoginPath } from "@/lib/auth-redirect";
import { useT } from "@/hooks/use-translations";

function SocialCallbackContent() {
  const t = useT();
  const router = useRouter();
  const params = useSearchParams();
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    const oauthError = params.get("error")?.trim() ?? "";
    const code = params.get("code")?.trim() ?? "";
    const state = params.get("state")?.trim() ?? "";
    const provider =
      typeof window !== "undefined"
        ? sessionStorage.getItem("social_auth_provider")?.trim() ?? ""
        : "";
    const linkMode =
      typeof window !== "undefined" &&
      sessionStorage.getItem("social_auth_link") === "true";

    async function run() {
      if (oauthError) {
        if (!cancelled) setError(oauthError);
        return;
      }
      if (!provider || !code) {
        if (!cancelled) {
          setError(t("auth.social.missing_params"));
        }
        return;
      }
      try {
        const res = await socialExchange(provider, code, state, linkMode);
        if (linkMode) {
          router.replace(res.redirect ?? "/settings/account");
          return;
        }
        if (!res.access_token) {
          throw new Error(t("auth.error.token_failed"));
        }
        setTokens(res.access_token, res.refresh_token ?? "");
        router.replace(
          res.redirect ?? postLoginPath(res.user?.role ?? "user")
        );
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error ? err.message : t("auth.social.failed")
          );
        }
      } finally {
        sessionStorage.removeItem("social_auth_mode");
        sessionStorage.removeItem("social_auth_provider");
        sessionStorage.removeItem("social_auth_link");
      }
    }

    void run();
    return () => {
      cancelled = true;
    };
  }, [params, router]);

  return (
    <div className="flex min-h-svh flex-col items-center justify-center p-6">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>
            {error ? t("auth.social.error_title") : t("auth.social.title")}
          </CardTitle>
          <CardDescription>
            {error ? error : t("auth.social.waiting")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {error ? (
            <>
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
              <Button asChild className="w-full">
                <Link href="/login">{t("auth.back_to_login")}</Link>
              </Button>
            </>
          ) : (
            <div className="flex items-center justify-center gap-2 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin" />
              <span>{t("auth.social.progress")}</span>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function SocialCallbackFallback() {
  const t = useT();
  return (
    <div className="flex min-h-svh flex-col items-center justify-center p-6">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{t("auth.social.title")}</CardTitle>
          <CardDescription>{t("auth.social.waiting")}</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center gap-2 text-muted-foreground">
            <Loader2 className="h-5 w-5 animate-spin" />
            <span>{t("auth.social.progress")}</span>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default function SocialCallbackPage() {
  return (
    <Suspense fallback={<SocialCallbackFallback />}>
      <SocialCallbackContent />
    </Suspense>
  );
}
