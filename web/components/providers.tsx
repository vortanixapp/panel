"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { ThemeProvider } from "@/context/theme-provider";
import { FontProvider } from "@/context/font-provider";
import { DirectionProvider } from "@/context/direction-provider";
import { BrandProvider } from "@/context/brand-provider";
import { TranslationsProvider } from "@/context/translations-provider";
import { LocaleProvider } from "@/context/locale-provider";
import type { Locale } from "@/lib/i18n";
import { Toaster } from "@/components/ui/sonner";
import { AuthSessionProvider } from "@/components/auth/auth-session-provider";
import { VxRouteProgress } from "@/components/vx/loader";

export function Providers({
  children,
  initialLocale,
  hasLocaleCookie,
}: {
  children: React.ReactNode;
  initialLocale: Locale;
  hasLocaleCookie: boolean;
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
      <ThemeProvider defaultTheme="dark">
        <FontProvider>
          <DirectionProvider>
            <BrandProvider>
              <VxRouteProgress />
              <AuthSessionProvider>
                <LocaleProvider
                  initialLocale={initialLocale}
                  hasCookie={hasLocaleCookie}
                >
                  <TranslationsProvider>{children}</TranslationsProvider>
                </LocaleProvider>
              </AuthSessionProvider>
              <Toaster richColors closeButton />
            </BrandProvider>
          </DirectionProvider>
        </FontProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
