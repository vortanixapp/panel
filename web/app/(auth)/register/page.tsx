import { BootstrappedGuard } from "@/components/auth/bootstrapped-guard";
import { RegisterForm } from "@/components/auth/register-form";

export default function RegisterPage() {
  return (
    <BootstrappedGuard>
      <RegisterForm />
    </BootstrappedGuard>
  );
}
