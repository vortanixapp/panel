import type { Metadata } from "next";
import { MonitoringPublicPageContent } from "@/components/user/monitoring-public-page-content";

export const metadata: Metadata = {
  title: "Публичный мониторинг сервера",
  description: "Статус игрового сервера в реальном времени",
};

export default function MonitoringPublicPage() {
  return <MonitoringPublicPageContent />;
}
