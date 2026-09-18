"use client";

import { useEffect } from "react";
import { notFound, useRouter } from "next/navigation";
import { LandingPublicLayout } from "@/components/landing/landing-public-layout";
import { DashboardLayout } from "@/components/layout/dashboard-layout";
import { PageShell } from "@/components/layout/page-shell";
import { SiteBlockList } from "@/components/site/block-renderer";
import { VxPageLoader } from "@/components/vx/loader";
import { useSite } from "@/context/site-provider";
import { useSiteText, useViewer } from "@/hooks/use-site";

export function CustomPageView({ slug }: { slug: string }) {
  const { document, refreshed, inEditor } = useSite();
  const viewer = useViewer();
  const router = useRouter();
  const text = useSiteText();
  const page = document.custom_pages?.find((item) => item.slug === slug);
  const available = Boolean(page && (!page.hidden || inEditor));
  const audience = page?.layout === "panel" ? "users" : (page?.audience ?? "");

  useEffect(() => {
    if (!available || inEditor || !viewer.ready) return;
    if (audience === "users" && !viewer.loggedIn) {
      router.replace("/login");
    } else if (audience === "guests" && viewer.loggedIn) {
      router.replace("/dashboard");
    }
  }, [available, inEditor, viewer.ready, viewer.loggedIn, audience, router]);

  if (!page || !available) {
    if (!refreshed) return <VxPageLoader />;
    notFound();
  }

  const key = `p:${page.id}`;

  if (page.layout === "panel") {
    return (
      <DashboardLayout variant="user">
        <PageShell variant="user">
          <h1 className="mb-6 text-2xl font-semibold tracking-tight">{text(page.title)}</h1>
          <SiteBlockList blocks={page.blocks} variant="panel" page={key} zone="blocks" className="flex flex-col gap-6" />
        </PageShell>
      </DashboardLayout>
    );
  }

  return (
    <LandingPublicLayout>
      <SiteBlockList blocks={page.blocks} variant="site" page={key} zone="blocks" />
    </LandingPublicLayout>
  );
}
