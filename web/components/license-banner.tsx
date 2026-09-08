"use client";

import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, Lock } from "lucide-react";
import { fetchLicenseState, type LicenseStateResponse } from "@/lib/api";
import { localeTag } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";

export function LicenseBanner() {
  const t = useT();
  const { data } = useQuery<LicenseStateResponse>({
    queryKey: ["license-state"],
    queryFn: fetchLicenseState,
    refetchInterval: 60_000,
    retry: false,
  });

  if (!data || data.state === "active") {
    if (data?.legacy) {
      return (
        <Banner tone="info">
          {data.message}{" "}
          <a href="/admin/license" className="underline underline-offset-2">
            {t("layout.license.open_settings")}
          </a>
        </Banner>
      );
    }
    return null;
  }

  if (data.state === "read_only") {
    return (
      <Banner tone="danger" icon={<Lock className="size-4 shrink-0" />}>
        {t("layout.license.read_only", { message: data.message })}
      </Banner>
    );
  }

  return (
    <Banner tone="warn" icon={<AlertTriangle className="size-4 shrink-0" />}>
      {data.message}
      {data.grace_until && (
        <>
          {" "}
          {t("layout.license.grace_until", {
            date: formatDate(data.grace_until),
          })}
        </>
      )}
    </Banner>
  );
}

function Banner({
  tone,
  icon,
  children,
}: {
  tone: "info" | "warn" | "danger";
  icon?: React.ReactNode;
  children: React.ReactNode;
}) {
  const palette = {
    info: "border-sky-500/30 bg-sky-500/10 text-sky-200",
    warn: "border-amber-500/30 bg-amber-500/10 text-amber-200",
    danger: "border-red-500/30 bg-red-500/10 text-red-200",
  }[tone];

  return (
    <div
      role="status"
      className={`flex items-center gap-2 border-b px-4 py-2 text-sm ${palette}`}
    >
      {icon}
      <span>{children}</span>
    </div>
  );
}

function formatDate(value: string): string {
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return "—";
  // Дата обязана следовать языку интерфейса: под английским «5 октября»
  // посреди английской фразы читается как сбой, а не как локальный формат.
  return d.toLocaleString(localeTag(), {
    day: "numeric",
    month: "long",
    hour: "2-digit",
    minute: "2-digit",
  });
}
