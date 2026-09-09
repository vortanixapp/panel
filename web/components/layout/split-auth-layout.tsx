import { BrandLogo } from "@/components/brand-logo";
import { DashboardPreview } from "@/components/landing/dashboard-preview";
import { cn } from "@/lib/utils";

export function SplitAuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative container grid min-h-svh max-w-none flex-col items-center justify-center lg:grid-cols-2 lg:px-0">
      <div className="landing-grid pointer-events-none absolute inset-0 lg:right-1/2" />
      <div className="relative z-10 w-full lg:p-8">
        <div className="mx-auto flex w-full max-w-sm flex-col justify-center space-y-2 py-8 sm:p-8">
          <div className="mb-4 flex items-center justify-center">
            <BrandLogo size="lg" />
          </div>
          {children}
        </div>
      </div>

      <div
        className={cn(
          "relative hidden min-h-svh items-center justify-center overflow-hidden bg-muted/30 p-8 lg:flex"
        )}
      >
        <DashboardPreview className="w-full max-w-2xl select-none shadow-none" />
        <div className="pointer-events-none absolute inset-0 bg-gradient-to-l from-background/10 via-transparent to-transparent" />
      </div>
    </div>
  );
}
