import { BootstrappedGuard } from "@/components/auth/bootstrapped-guard";
import { LoginForm } from "@/components/auth/login-form";

export default function LoginPage() {
  return (
    <BootstrappedGuard>
      <LoginForm />
    </BootstrappedGuard>
  );
}
