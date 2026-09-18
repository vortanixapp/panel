"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useTenantStatus } from "@/hooks/use-tenant-status";

export function BootstrappedGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { data } = useTenantStatus();

  useEffect(() => {
    if (data && !data.bootstrapped) {
      router.replace("/setup");
    }
  }, [data, router]);

  return <>{children}</>;
}
