"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef, useState } from "react";
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
import { exchangeWhmcsSSO, setTokens } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

function WhmcsProgress() {
  const t = useT();
  return (
    <div className="flex min-h-svh flex-col items-center justify-center p-6">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{t("auth.whmcs.title")}</CardTitle>
          <CardDescription>{t("auth.whmcs.waiting")}</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center gap-2 text-muted-foreground">
            <Loader2 className="h-5 w-5 animate-spin" />
            <span>{t("auth.whmcs.progress")}</span>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function WhmcsSignInContent() {
  const t = useT();
  const router = useRouter();
  const params = useSearchParams();
  const [error, setError] = useState("");
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    const token = params.get("token")?.trim() ?? "";
    if (!token) {
      setError(t("auth.whmcs.missing_token"));
      return;
    }
    exchangeWhmcsSSO(token)
      .then((res) => {
        setTokens(res.access_token, res.refresh_token ?? "");
        router.replace(res.redirect || "/servers");
      })
      .catch((err: unknown) => {
        setError(err instanceof Error && err.message ? err.message : t("auth.whmcs.failed"));
      });
  }, [params, router, t]);

  if (!error) return <WhmcsProgress />;

  return (
    <div className="flex min-h-svh flex-col items-center justify-center p-6">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{t("auth.whmcs.error_title")}</CardTitle>
          <CardDescription>{error}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
          <Button asChild className="w-full">
            <Link href="/login">{t("auth.back_to_login")}</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

export default function WhmcsSignInPage() {
  return (
    <Suspense fallback={<WhmcsProgress />}>
      <WhmcsSignInContent />
    </Suspense>
  );
}
