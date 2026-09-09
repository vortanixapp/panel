"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useTenantStatus } from "@/hooks/use-tenant-status";
import { VxPageLoader } from "@/components/vx/loader";
import { useT } from "@/hooks/use-translations";

export function SetupGuard({ children }: { children: React.ReactNode }) {
  const t = useT();
  const router = useRouter();
  const { data, isLoading, isError } = useTenantStatus();

  useEffect(() => {
    if (data?.bootstrapped) {
      router.replace("/login");
    }
  }, [data?.bootstrapped, router]);

  if (isLoading) {
    return <VxPageLoader label="Vortanix" hint={t("layout.loader.checking_session")} />;
  }

  if (isError || data?.bootstrapped) {
    return null;
  }

  return <>{children}</>;
}
