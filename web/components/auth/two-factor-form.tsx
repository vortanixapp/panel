"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch, setTokens } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useT } from "@/hooks/use-translations";

export function TwoFactorChallengeForm() {
  const t = useT();
  const router = useRouter();
  const [code, setCode] = useState("");
  const [error, setError] = useState("");

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      const res = await apiFetch<{ access_token: string; refresh_token: string }>(
        "/v1/auth/2fa/challenge",
        { method: "POST", body: JSON.stringify({ code }) }
      );
      setTokens(res.access_token, res.refresh_token);
      router.push("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.two_factor.invalid_code"));
    }
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>{t("auth.two_factor.title")}</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} className="space-y-4">
          <Input
            placeholder={t("auth.two_factor.code_placeholder")}
            value={code}
            onChange={(e) => setCode(e.target.value)}
            required
          />
          {error && <p className="text-sm text-destructive">{error}</p>}
          <Button type="submit" className="w-full">
            {t("common.confirm")}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
