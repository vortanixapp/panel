import { BootstrappedGuard } from "@/components/auth/bootstrapped-guard";
import { RegisterForm } from "@/components/auth/register-form";

export default function RegisterPage() {
  return (
    <BootstrappedGuard loaderClassName="min-h-[420px]">
      <RegisterForm />
    </BootstrappedGuard>
  );
}
