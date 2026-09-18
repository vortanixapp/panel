"use client";

import { useEffect, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { VxPageLoader } from "@/components/vx/loader";
import { useMe } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";
import { hasSession } from "@/lib/api";

export function EditorSessionGate({ children }: { children: ReactNode }) {
  const router = useRouter();
  const t = useT();
  const { data: me, isLoading, isError } = useMe();

  useEffect(() => {
    if (!hasSession() || isError) router.replace("/login");
  }, [isError, router]);

  if (!hasSession() || isLoading || !me) {
    return <VxPageLoader hint={t("layout.loader.checking_session")} />;
  }
  return <>{children}</>;
}
