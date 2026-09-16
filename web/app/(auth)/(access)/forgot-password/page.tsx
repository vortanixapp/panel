import { BootstrappedGuard } from "@/components/auth/bootstrapped-guard";
import { ForgotPasswordForm } from "@/components/auth/forgot-password-form";

export default function ForgotPasswordPage() {
  return (
    <BootstrappedGuard loaderClassName="min-h-[420px]">
      <ForgotPasswordForm />
    </BootstrappedGuard>
  );
}
