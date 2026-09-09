"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { useT } from "@/hooks/use-translations";

export function GeneralError({
  className,
  minimal = false,
}: React.HTMLAttributes<HTMLDivElement> & { minimal?: boolean }) {
  const t = useT();
  const router = useRouter();

  return (
    <div className={cn("h-svh w-full", className)}>
      <div className="m-auto flex h-full w-full flex-col items-center justify-center gap-2">
        {!minimal && (
          <h1 className="text-[7rem] font-bold leading-tight">500</h1>
        )}
        <span className="font-medium">{t("errors.500.title")}</span>
        <p className="text-center text-muted-foreground">
          {t("errors.500.desc")}
        </p>
        {!minimal && (
          <div className="mt-6 flex gap-4">
            <Button variant="outline" onClick={() => router.back()}>
              {t("errors.back")}
            </Button>
            <Button asChild>
              <Link href="/dashboard">{t("errors.to_home")}</Link>
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
