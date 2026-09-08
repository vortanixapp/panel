import type { Metadata } from "next";
import { DM_Mono, Inter, Outfit, Sora, Space_Mono } from "next/font/google";
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

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="ru" className="dark" suppressHydrationWarning>
      <head>
        <script src="/api/config.js" />
        <link
          rel="stylesheet"
          href="https://cdn.jsdelivr.net/npm/remixicon@3.5.0/fonts/remixicon.css"
        />
      </head>
      <body className={`${inter.variable} ${sora.variable} ${spaceMono.variable} ${outfit.variable} ${dmMono.variable} font-sans`}>
        <Providers>
          <NavigationProgress />
          {children}
        </Providers>
      </body>
    </html>
  );
}
