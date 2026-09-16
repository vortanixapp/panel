import { LandingBody } from "@/components/landing/landing-body";
import { LandingPublicLayout } from "@/components/landing/landing-public-layout";
import type { Metadata } from "next";
import { BRAND_NAME } from "@/lib/brand";

export const metadata: Metadata = {
  title: `${BRAND_NAME} — игровой хостинг`,
  description:
    "Серверы для Minecraft, CS2, Rust и ещё сорока игр. Запуск за минуту, ядра закреплены за вашим сервером, защита от DDoS на всех тарифах.",
};

export default function Home() {
  return (
    <LandingPublicLayout>
      <LandingBody />
    </LandingPublicLayout>
  );
}
