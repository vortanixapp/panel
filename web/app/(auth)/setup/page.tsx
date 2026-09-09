import { SetupGuard } from "@/components/auth/setup-guard";
import { SetupForm } from "@/components/auth/setup-form";

export default function SetupPage() {
  return (
    <SetupGuard>
      <SetupForm />
    </SetupGuard>
  );
}
