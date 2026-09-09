import type { Metadata } from "next";
import { cookies } from "next/headers";
import { DM_Mono, Inter, Outfit, Sora, Space_Mono } from "next/font/google";
import { LOCALE_COOKIE_NAME, normalizeLocale } from "@/lib/i18n";
import { BRAND_NAME, DEFAULT_BRAND_MARK_URL } from "@/lib/brand";
import { serverRuntimeConfig } from "@/lib/runtime-config";
import { Providers } from "@/components/providers";
import { NavigationProgress } from "@/components/navigation-progress";
import "./globals.css";

const inter = Inter({
  subsets: ["latin", "cyrillic"],
  variable: "--font-inter",
});

const sora = Sora({
  subsets: ["latin", "latin-ext"],
  weight: ["300", "400", "500", "600", "700", "800"],
  variable: "--font-sora",
  display: "swap",
  fallback: ["system-ui", "sans-serif"],
  adjustFontFallback: false,
});

const spaceMono = Space_Mono({
  subsets: ["latin", "latin-ext"],
  weight: ["400", "700"],
  variable: "--font-space-mono",
  display: "swap",
  fallback: ["monospace"],
  adjustFontFallback: false,
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

export function generateMetadata(): Metadata {
  const brand = serverRuntimeConfig().brand_name || BRAND_NAME;
  return {
    title: `${brand} Panel`,
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
  const locale = normalizeLocale(cookieLocale);

  return (
    <html lang={locale} className="dark" suppressHydrationWarning>
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
      <body className={`${inter.variable} ${sora.variable} ${spaceMono.variable} ${outfit.variable} ${dmMono.variable} font-sans`}>
        <Providers
          initialLocale={locale}
          hasLocaleCookie={Boolean(cookieLocale)}
        >
          <NavigationProgress />
          {children}
        </Providers>
      </body>
    </html>
  );
}
