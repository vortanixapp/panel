import { AuthLayout } from "@/components/auth/auth-layout";

export default function AccessLayout({ children }: { children: React.ReactNode }) {
  return <AuthLayout>{children}</AuthLayout>;
}
