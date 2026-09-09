"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { useT } from "@/hooks/use-translations";

export function UnauthorizedError({
  className,
}: React.HTMLAttributes<HTMLDivElement>) {
  const t = useT();
  const router = useRouter();

  return (
    <div className={cn("h-svh w-full", className)}>
      <div className="m-auto flex h-full w-full flex-col items-center justify-center gap-2">
        <h1 className="text-[7rem] font-bold leading-tight">401</h1>
        <span className="font-medium">{t("errors.401.title")}</span>
        <p className="text-center text-muted-foreground">
          {t("errors.401.desc")}
        </p>
        <div className="mt-6 flex gap-4">
          <Button variant="outline" onClick={() => router.back()}>
            {t("errors.back")}
          </Button>
          <Button asChild>
            <Link href="/login">{t("errors.sign_in")}</Link>
          </Button>
        </div>
      </div>
    </div>
  );
}
