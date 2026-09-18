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
import { LOCALE_COOKIE_NAME } from "@/lib/i18n";
import { loadServerI18n } from "@/lib/i18n-server";
import { loadServerSite } from "@/lib/site-server";
import { DEFAULT_BRAND_MARK_URL } from "@/lib/brand";
import { loadServerBrandName } from "@/lib/branding-server";
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
  const brand = await loadServerBrandName();
  return {
    title: { default: brand, template: `%s — ${brand}` },
    description: "Game hosting control panel",
    icons: {
      icon: [{ url: DEFAULT_BRAND_MARK_URL, type: "image/png" }],
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
  const cookieLocale = store.get(LOCALE_COOKIE_NAME)?.value;
  const [i18n, site] = await Promise.all([loadServerI18n(cookieLocale), loadServerSite()]);

  return (
    <html lang={i18n.locale} className="dark" suppressHydrationWarning>
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html: `(function(){try{var m=document.cookie.match(/(?:^|; )vite-ui-theme=([^;]*)/);var t=m?decodeURIComponent(m[1]):"dark";if(t==="system"){t=window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light";}var r=document.documentElement;r.classList.remove("light","dark");r.classList.add(t);}catch(e){}})();`,
          }}
        />
        <script src="/api/config.js" />
        <link
          rel="stylesheet"
          href="https://cdn.jsdelivr.net/npm/remixicon@3.5.0/fonts/remixicon.css"
        />
      </head>
      <body
        className={`${inter.variable} ${outfit.variable} ${dmMono.variable} ${manrope.variable} ${rubik.variable} ${montserrat.variable} ${ibmPlexSans.variable} font-sans`}
      >
        <Providers i18n={i18n} hasLocaleCookie={Boolean(cookieLocale)} site={site}>
          <NavigationProgress />
          {children}
        </Providers>
      </body>
    </html>
  );
}
