"use client";

import { useT } from "@/hooks/use-translations";
import { ContentSection } from "../components/content-section";
import { AppearanceForm } from "./appearance-form";

export function SettingsAppearance() {
  const t = useT();
  return (
    <ContentSection
      title={t("settings.appearance.section_title")}
      desc={t("settings.appearance.section_desc")}
    >
      <AppearanceForm />
    </ContentSection>
  );
}
