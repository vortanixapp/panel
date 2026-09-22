import type { Metadata } from "next";
import { cookies } from "next/headers";
import {
  DM_Mono,
  IBM_Plex_Sans,
  Inter,
  Manrope,
  Montserrat,
  Outfit,
  Rubik,
} from "next/font/google";
import { USER_COOKIE } from "@/lib/accounts";
import { bootScript } from "@/lib/appearance";
import { DEFAULT_BRAND_MARK_URL } from "@/lib/brand";
import { brandTitle, loadServerBranding } from "@/lib/branding-server";
import { loadRequestI18n } from "@/lib/i18n-server";
import { serverRuntimeConfig } from "@/lib/runtime-config";
import { loadServerSite, viewerFromCookie } from "@/lib/site-server";
import { Providers } from "@/components/providers";
import { NavigationProgress } from "@/components/navigation-progress";
import "./globals.css";

const inter = Inter({
  subsets: ["latin", "cyrillic"],
  variable: "--font-inter",
});

const outfit = Outfit({
  subsets: ["latin", "latin-ext"],
  weight: ["300", "400", "500", "600", "700"],
  variable: "--font-outfit",
  display: "swap",
  fallback: ["system-ui", "sans-serif"],
  adjustFontFallback: false,
});

const dmMono = DM_Mono({
  subsets: ["latin"],
  weight: ["400", "500"],
  variable: "--font-dm-mono",
  display: "swap",
  fallback: ["monospace"],
  adjustFontFallback: false,
});

const manrope = Manrope({
  subsets: ["latin", "cyrillic"],
  variable: "--font-manrope",
  display: "swap",
  preload: false,
});

const rubik = Rubik({
  subsets: ["latin", "cyrillic"],
  variable: "--font-rubik",
  display: "swap",
  preload: false,
});

const montserrat = Montserrat({
  subsets: ["latin", "cyrillic"],
  variable: "--font-montserrat",
  display: "swap",
  preload: false,
});

const ibmPlexSans = IBM_Plex_Sans({
  subsets: ["latin", "cyrillic"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-ibm-plex-sans",
  display: "swap",
  preload: false,
});

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
        className={`${inter.variable} ${outfit.variable} ${dmMono.variable} ${manrope.variable} ${rubik.variable} ${montserrat.variable} ${ibmPlexSans.variable} font-sans`}
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
