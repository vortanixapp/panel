"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { ThemeProvider } from "@/context/theme-provider";
import { FontProvider } from "@/context/font-provider";
import { DirectionProvider } from "@/context/direction-provider";
import { BrandProvider } from "@/context/brand-provider";
import { TranslationsProvider } from "@/context/translations-provider";
import { Toaster } from "@/components/ui/sonner";
import { AuthSessionProvider } from "@/components/auth/auth-session-provider";
import { VxRouteProgress } from "@/components/vx/loader";

export function Providers({ children }: { children: React.ReactNode }) {
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
                <TranslationsProvider>{children}</TranslationsProvider>
              </AuthSessionProvider>
              <Toaster richColors closeButton />
            </BrandProvider>
          </DirectionProvider>
        </FontProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
