import { AdminGuard } from "@/components/auth/admin-guard";
import { DashboardLayout } from "@/components/layout/dashboard-layout";

export default function AdminGroupLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AdminGuard>
      <DashboardLayout variant="admin">{children}</DashboardLayout>
    </AdminGuard>
  );
}
