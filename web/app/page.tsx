import type { Metadata } from "next";
import { LandingBody } from "@/components/landing/landing-body";
import { LandingPublicLayout } from "@/components/landing/landing-public-layout";
import { brandTitle, loadServerBranding } from "@/lib/branding-server";
import { translator } from "@/lib/i18n";
import { loadRequestI18n } from "@/lib/i18n-server";

export async function generateMetadata(): Promise<Metadata> {
  const [branding, { i18n }] = await Promise.all([loadServerBranding(), loadRequestI18n()]);
  const t = translator(i18n);
  return {
    title: { absolute: `${brandTitle(branding)} — ${t("landing.meta.title")}` },
    description: branding?.site_description?.trim() || t("landing.meta.description"),
  };
}

export default function Home() {
  return (
    <LandingPublicLayout>
      <LandingBody />
    </LandingPublicLayout>
  );
}
