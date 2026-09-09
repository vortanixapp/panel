"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  resendEmailVerification,
  verifyEmailURL,
  getAccessToken,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";

function VerifyEmailContent() {
  const t = useT();
  const params = useSearchParams();
  const verifyURL = params.get("verify_url")?.trim() ?? "";
  const [verifying, setVerifying] = useState(!!verifyURL);
  const [verified, setVerified] = useState(false);
  const [verifyError, setVerifyError] = useState("");
  const [sent, setSent] = useState(false);
  const [sending, setSending] = useState(false);
  const [loggedIn, setLoggedIn] = useState(false);

  useEffect(() => {
    setLoggedIn(!!getAccessToken());
  }, []);

  useEffect(() => {
    if (!verifyURL) return;
    let cancelled = false;
    (async () => {
      try {
        await verifyEmailURL(verifyURL);
        if (!cancelled) setVerified(true);
      } catch (err) {
        if (!cancelled) {
          setVerifyError(
            err instanceof Error ? err.message : t("auth.verify.failed")
          );
        }
      } finally {
        if (!cancelled) setVerifying(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [verifyURL]);

  async function resend() {
    setSending(true);
    try {
      await resendEmailVerification();
      setSent(true);
    } catch {
      setVerifyError(t("auth.verify.resend_failed"));
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="flex min-h-svh flex-col items-center justify-center p-6">
      <Card className="w-full max-w-md text-center">
        <CardHeader>
          <CardTitle>{t("auth.verify.title")}</CardTitle>
          <CardDescription>{t("auth.verify.subtitle")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {verifying && (
            <div className="flex items-center justify-center gap-2 text-sm text-sky-400">
              <Loader2 className="h-4 w-4 animate-spin" />
              {t("auth.verify.checking")}
            </div>
          )}
          {verified && (
            <p className="rounded-xl border border-emerald-500/20 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-500">
              {t("auth.verify.success")}
            </p>
          )}
          {verifyError && (
            <p className="rounded-xl border border-destructive/20 bg-destructive/10 px-4 py-3 text-sm text-destructive">
              {verifyError}
            </p>
          )}
          {sent && (
            <p className="rounded-xl border border-emerald-500/20 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-500">
              {t("auth.verify.resent")}
            </p>
          )}
          <Button
            className="w-full"
            onClick={resend}
            disabled={sending || !loggedIn}
          >
            {sending ? t("common.sending") : t("auth.verify.resend")}
          </Button>
          <Button asChild variant="outline" className="w-full">
            <Link href="/dashboard">{t("auth.verify.go_panel")}</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

function VerifyEmailFallback() {
  const t = useT();
  return (
    <div className="flex min-h-svh flex-col items-center justify-center p-6">
      <Card className="w-full max-w-md text-center">
        <CardHeader>
          <CardTitle>{t("auth.verify.title")}</CardTitle>
          <CardDescription>{t("common.loading")}</CardDescription>
        </CardHeader>
        <CardContent>
          <Loader2 className="mx-auto h-6 w-6 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    </div>
  );
}

export default function VerifyEmailPage() {
  return (
    <Suspense fallback={<VerifyEmailFallback />}>
      <VerifyEmailContent />
    </Suspense>
  );
}
