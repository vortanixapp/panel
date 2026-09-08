"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { getAccessToken, getRefreshToken, clearAuth, ensureValidSession } from "@/lib/api";
import { useMe } from "@/hooks/use-queries";
import { queryKeys } from "@/lib/query-keys";
import { AuthenticatedLayout } from "@/components/layout/authenticated-layout";
import { useLiveSync } from "@/hooks/use-live-sync";
import type { PanelVariant } from "@/lib/panel-paths";
import { VxPageLoader } from "@/components/vx/loader";
import { LicenseBanner } from "@/components/license-banner";
import { useT } from "@/hooks/use-translations";

function hasStoredSession() {
  return !!(getAccessToken() || getRefreshToken());
}

export function DashboardLayout({
  children,
  variant = "user",
}: {
  children: React.ReactNode;
  variant?: PanelVariant;
}) {
  const t = useT();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { data: me, isError, isLoading, error } = useMe();

  useLiveSync();

  useEffect(() => {
    if (!hasStoredSession()) {
      router.replace("/login");
    }
  }, [router]);

  useEffect(() => {
    if (!isError) return;
    const message = error instanceof Error ? error.message : "";
    if (message === "Session expired" || !getRefreshToken()) {
      clearAuth();
      router.replace("/login");
      return;
    }
    void ensureValidSession().then((ok) => {
      if (ok) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.me });
      } else {
        clearAuth();
        router.replace("/login");
      }
    });
  }, [isError, error, router, queryClient]);

  if (!hasStoredSession() || isLoading || !me) {
    return <VxPageLoader hint={t("layout.loader.checking_session")} />;
  }

  return (
    <AuthenticatedLayout email={me.email} role={me.role} variant={variant}>
      <LicenseBanner />
      {children}
    </AuthenticatedLayout>
  );
}
