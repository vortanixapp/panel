import type { Metadata } from "next";
import { cookies } from "next/headers";
import { USER_COOKIE } from "@/lib/accounts";
import { bootScript } from "@/lib/appearance";
import { DEFAULT_BRAND_MARK_URL } from "@/lib/brand";
import { brandTitle, loadServerBranding } from "@/lib/branding-server";
import { loadRequestI18n } from "@/lib/i18n-server";
import { serverRuntimeConfig } from "@/lib/runtime-config";
import { loadServerSite, viewerFromCookie } from "@/lib/site-server";
import { Providers } from "@/components/providers";
import { NavigationProgress } from "@/components/navigation-progress";
import "./fonts.css";
import "./globals.css";

export async function generateMetadata(): Promise<Metadata> {
  const branding = await loadServerBranding();
  const brand = brandTitle(branding);
  const description = branding?.site_description?.trim() || "Game hosting control panel";
  return {
    title: { default: brand, template: `%s — ${brand}` },
    description,
    openGraph: { type: "website", title: brand, description, siteName: brand },
    twitter: { card: "summary", title: brand, description },
    icons: {
      icon: [
        branding?.icon_url
          ? { url: branding.icon_url }
          : { url: DEFAULT_BRAND_MARK_URL, type: "image/png" },
      ],
      apple: [{ url: "/apple-icon.png", type: "image/png" }],
    },
  };
}

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const store = await cookies();
  const [{ i18n, hasCookie }, site, branding] = await Promise.all([
    loadRequestI18n(),
    loadServerSite(),
    loadServerBranding(),
  ]);
  const viewer = viewerFromCookie(store.get(USER_COOKIE)?.value);

  return (
    <html lang={i18n.locale} className="dark" suppressHydrationWarning>
      <head>
        <script
          dangerouslySetInnerHTML={{ __html: bootScript(serverRuntimeConfig(), branding) }}
        />
        <link
          rel="stylesheet"
          href="https://cdn.jsdelivr.net/npm/remixicon@3.5.0/fonts/remixicon.css"
        />
      </head>
      <body
        className="font-sans"
      >
        <Providers
          i18n={i18n}
          hasLocaleCookie={hasCookie}
          site={site}
          branding={branding}
          viewer={viewer}
        >
          <NavigationProgress />
          {children}
        </Providers>
      </body>
    </html>
  );
}
