"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { fetchAccountIdentification } from "@/lib/api";
import { t } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

export function IdentificationNotice({ className }: { className?: string }) {
  useT();
  const { data } = useQuery({
    queryKey: ["account-identification"],
    queryFn: fetchAccountIdentification,
    staleTime: 60_000,
  });
  if (!data?.required || data.identified) return null;
  return (
    <div
      className={cn(
        "rounded-[16px] border border-[var(--vx-panel-line-strong)] bg-[var(--vx-elevated)] px-5 py-4 text-[13px]",
        className
      )}
    >
      <div className="font-semibold text-[var(--vx-warn)]">{t("billing.identification.title")}</div>
      <p className="mt-1 leading-[1.5] text-muted-foreground">
        {data.methods.length > 0
          ? t("billing.identification.text_methods", { methods: data.methods.join(", ") })
          : t("billing.identification.text")}
      </p>
      {data.methods.length > 0 && (
        <Link href="/billing" className="mt-2 inline-block text-primary hover:underline">
          {t("billing.identification.topup")}
        </Link>
      )}
    </div>
  );
}
