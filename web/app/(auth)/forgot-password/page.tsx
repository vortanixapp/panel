import { BootstrappedGuard } from "@/components/auth/bootstrapped-guard";
import { ForgotPasswordForm } from "@/components/auth/forgot-password-form";

export default function ForgotPasswordPage() {
  return (
    <BootstrappedGuard>
      <ForgotPasswordForm />
    </BootstrappedGuard>
  );
}
