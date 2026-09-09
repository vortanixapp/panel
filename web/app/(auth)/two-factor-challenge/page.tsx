import { Suspense } from "react";
import { BootstrappedGuard } from "@/components/auth/bootstrapped-guard";
import { TwoFactorChallengeForm } from "@/components/auth/two-factor-challenge-form";
import { Skeleton } from "@/components/ui/skeleton";

export default function TwoFactorChallengePage() {
  return (
    <BootstrappedGuard>
      <Suspense fallback={<Skeleton className="h-64 w-full" />}>
        <TwoFactorChallengeForm />
      </Suspense>
    </BootstrappedGuard>
  );
}
