"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Shield, User } from "lucide-react";
import { cn } from "@/lib/utils";
import { isAdminRole } from "@/lib/rbac";
import { isAdminSection } from "@/lib/panel-paths";
import { useT } from "@/hooks/use-translations";

type SectionSwitcherProps = {
  role: string;
  className?: string;
};

export function SectionSwitcher({ role, className }: SectionSwitcherProps) {
  const t = useT();
  const pathname = usePathname();

  if (!isAdminRole(role)) return null;

  const inAdmin = isAdminSection(pathname);
  const targetLabel = inAdmin ? t("common.user") : t("layout.section.admin");

  return (
    <>
      <Link
        href={inAdmin ? "/dashboard" : "/admin/dashboard"}
        title={targetLabel}
        aria-label={targetLabel}
        className="hidden size-8 items-center justify-center self-center rounded-md border bg-muted/40 text-muted-foreground transition-colors hover:text-foreground group-data-[collapsible=icon]:flex"
      >
        {inAdmin ? <User className="size-4" /> : <Shield className="size-4" />}
      </Link>
      <div
        className={cn(
          "flex rounded-lg border bg-muted/40 p-1 text-xs font-medium group-data-[collapsible=icon]:hidden",
          className
        )}
      >
        <Link
          href="/dashboard"
          className={cn(
            "flex flex-1 items-center justify-center gap-1.5 rounded-md px-2 py-1.5 transition-colors",
            !inAdmin
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          )}
        >
          <User className="size-3.5" />
          {t("common.user")}
        </Link>
        <Link
          href="/admin/dashboard"
          className={cn(
            "flex flex-1 items-center justify-center gap-1.5 rounded-md px-2 py-1.5 transition-colors",
            inAdmin
              ? "bg-background text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground"
          )}
        >
          <Shield className="size-3.5" />
          {t("layout.section.admin")}
        </Link>
      </div>
    </>
  );
}
