import { Suspense } from "react";
import { BootstrappedGuard } from "@/components/auth/bootstrapped-guard";
import { ResetPasswordForm } from "@/components/auth/reset-password-form";
import { Skeleton } from "@/components/ui/skeleton";

export default function ResetPasswordPage() {
  return (
    <BootstrappedGuard loaderClassName="min-h-[420px]">
      <Suspense fallback={<Skeleton className="h-64 w-full" />}>
        <ResetPasswordForm />
      </Suspense>
    </BootstrappedGuard>
  );
}
