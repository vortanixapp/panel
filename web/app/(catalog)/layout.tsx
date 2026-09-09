import { DashboardLayout } from "@/components/layout/dashboard-layout";

export default function CatalogLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <DashboardLayout variant="user">{children}</DashboardLayout>;
}
