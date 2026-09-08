"use client";

import { useT } from "@/hooks/use-translations";
import { ContentSection } from "../components/content-section";
import { NotificationsForm } from "./notifications-form";

export function SettingsNotifications() {
  const t = useT();
  return (
    <ContentSection
      title={t("settings.notifications.section_title")}
      desc={t("settings.notifications.section_desc")}
    >
      <NotificationsForm />
    </ContentSection>
  );
}
