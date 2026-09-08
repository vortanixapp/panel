import { LandingBody } from "@/components/landing/landing-body";
import { LandingPublicLayout } from "@/components/landing/landing-public-layout";
import type { Metadata } from "next";
import { BRAND_NAME } from "@/lib/brand";

export const metadata: Metadata = {
  title: `${BRAND_NAME} — игровой хостинг`,
  description:
    "Выделенные ядра, NVMe Gen4 и анти-DDoS L3–L7 на всех тарифах. Развёртывание сервера за 40 секунд.",
};

export default function Home() {
  return (
    <LandingPublicLayout>
      <LandingBody />
    </LandingPublicLayout>
  );
}
