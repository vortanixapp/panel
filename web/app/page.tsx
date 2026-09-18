import type { Metadata } from "next";
import { cookies } from "next/headers";
import { LandingBody } from "@/components/landing/landing-body";
import { LandingPublicLayout } from "@/components/landing/landing-public-layout";
import { loadServerBrandName } from "@/lib/branding-server";
import { LOCALE_COOKIE_NAME, translator } from "@/lib/i18n";
import { loadServerI18n } from "@/lib/i18n-server";

export async function generateMetadata(): Promise<Metadata> {
  const store = await cookies();
  const [brand, i18n] = await Promise.all([
    loadServerBrandName(),
    loadServerI18n(store.get(LOCALE_COOKIE_NAME)?.value),
  ]);
  const t = translator(i18n);
  return {
    title: { absolute: `${brand} — ${t("landing.meta.title")}` },
    description: t("landing.meta.description"),
  };
}

export default function Home() {
  return (
    <LandingPublicLayout>
      <LandingBody />
    </LandingPublicLayout>
  );
}
