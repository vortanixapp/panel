"use client";

import { useT } from "@/hooks/use-translations";
import { ContentSection } from "../components/content-section";
import { DisplayForm } from "./display-form";

export function SettingsDisplay() {
  const t = useT();
  return (
    <ContentSection
      title={t("settings.display.section_title")}
      desc={t("settings.display.section_desc")}
    >
      <DisplayForm />
    </ContentSection>
  );
}
