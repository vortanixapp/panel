import { DashboardLayout } from "@/components/layout/dashboard-layout";

export default function UserGroupLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <DashboardLayout variant="user">{children}</DashboardLayout>;
}
