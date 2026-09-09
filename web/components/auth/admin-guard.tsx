"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useMe } from "@/hooks/use-queries";
import { isStaffRole } from "@/lib/rbac";

export function AdminGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { data: me } = useMe();

  useEffect(() => {
    if (me && !isStaffRole(me.role)) {
      router.replace("/dashboard");
    }
  }, [me, router]);

  if (me && !isStaffRole(me.role)) {
    return null;
  }

  return <>{children}</>;
}
