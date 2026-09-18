"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { ThemeProvider } from "@/context/theme-provider";
import { FontProvider } from "@/context/font-provider";
import { DirectionProvider } from "@/context/direction-provider";
import { BrandProvider } from "@/context/brand-provider";
import { LocaleProvider } from "@/context/locale-provider";
import { SiteProvider } from "@/context/site-provider";
import { EditorBridgeLoader } from "@/components/site/editor-bridge/bridge-loader";
import type { I18nPayload } from "@/lib/i18n";
import type { SiteDocument } from "@/lib/site/types";
import { Toaster } from "@/components/ui/sonner";
import { AuthSessionProvider } from "@/components/auth/auth-session-provider";
import { CookieBanner } from "@/components/legal/cookie-banner";
import { VxRouteProgress } from "@/components/vx/loader";

export function Providers({
  children,
  i18n,
  hasLocaleCookie,
  site,
}: {
  children: React.ReactNode;
  i18n: I18nPayload;
  hasLocaleCookie: boolean;
  site: SiteDocument;
}) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 5_000,
            retry: 1,
            refetchOnWindowFocus: true,
          },
        },
      })
  );

  return (
    <QueryClientProvider client={queryClient}>
      <LocaleProvider initial={i18n} hasCookie={hasLocaleCookie}>
        <ThemeProvider defaultTheme="dark">
          <FontProvider>
            <DirectionProvider>
              <BrandProvider>
                <SiteProvider initial={site}>
                  <EditorBridgeLoader />
                  <VxRouteProgress />
                  <AuthSessionProvider>{children}</AuthSessionProvider>
                  <Toaster richColors closeButton />
                  <CookieBanner />
                </SiteProvider>
              </BrandProvider>
            </DirectionProvider>
          </FontProvider>
        </ThemeProvider>
      </LocaleProvider>
    </QueryClientProvider>
  );
}
