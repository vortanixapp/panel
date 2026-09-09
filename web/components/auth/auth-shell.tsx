import type { ReactNode } from "react";

import { BrandLogo } from "@/components/brand-logo";

export function AuthShell({
  aside,
  title,
  subtitle,
  footer,
  children,
}: {
  aside?: ReactNode;
  title: string;
  subtitle?: string;
  footer?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="relative min-h-screen flex items-center justify-center py-12 overflow-hidden bg-background">
      <div className="pointer-events-none absolute inset-0">
        <div className="absolute top-1/4 left-1/4 w-[500px] h-[500px] bg-primary/10 rounded-full blur-[120px] animate-pulse" />
        <div
          className="absolute bottom-1/4 right-1/4 w-[400px] h-[400px] bg-primary/5 rounded-full blur-[100px] animate-pulse"
          style={{ animationDelay: "1s" }}
        />
      </div>
      <div
        className="pointer-events-none absolute inset-0 opacity-[0.02]"
        style={{
          backgroundImage:
            "linear-gradient(to right, currentColor 1px, transparent 1px), linear-gradient(to bottom, currentColor 1px, transparent 1px)",
          backgroundSize: "60px 60px",
        }}
      />

      <div className="relative w-full max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
        <div
          className={
            aside
              ? "grid lg:grid-cols-2 gap-12 lg:gap-20 items-center"
              : "mx-auto w-full max-w-lg"
          }
        >
          {aside && <div className="hidden lg:block">{aside}</div>}

          <div className="relative">
            <div className="absolute -inset-4 bg-gradient-to-r from-primary/20 via-primary/10 to-primary/20 rounded-[2rem] blur-2xl opacity-50" />
            <div className="relative bg-card border border-border rounded-3xl shadow-2xl overflow-hidden">
              <div className="h-1.5 bg-gradient-to-r from-primary/50 via-primary to-primary/50" />
              <div className="p-8 sm:p-10">
                <div className="mb-8 flex items-center justify-center lg:hidden">
                  <BrandLogo size="md" className="h-9 max-w-[220px]" priority />
                </div>

                <div className="text-center mb-8">
                  <h2 className="text-2xl sm:text-3xl font-bold text-foreground mb-2">{title}</h2>
                  {subtitle && <p className="text-muted-foreground">{subtitle}</p>}
                </div>

                {children}
              </div>
              {footer && (
                <div className="border-t border-border px-8 py-5 sm:px-10">{footer}</div>
              )}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

export function AuthAside({
  title,
  text,
  items,
}: {
  title: string;
  text: string;
  items: { icon: string; tone: string; title: string; note: string }[];
}) {
  return (
    <>
      <div className="mb-8">
        <BrandLogo size="lg" className="h-12 max-w-[264px]" priority />
      </div>
      <h1 className="text-4xl lg:text-5xl font-bold text-foreground mb-6 leading-tight">{title}</h1>
      <p className="text-lg text-muted-foreground mb-10 max-w-md">{text}</p>
      <div className="space-y-4">
        {items.map((item) => (
          <div
            key={item.title}
            className="flex items-center gap-4 p-4 bg-card/50 backdrop-blur-sm border border-border rounded-2xl transition-all duration-300 hover:bg-card hover:border-primary/30"
          >
            <div
              className={`w-12 h-12 flex items-center justify-center rounded-xl ${item.tone}`}
            >
              <i className={`${item.icon} text-xl`} />
            </div>
            <div>
              <div className="font-semibold text-foreground">{item.title}</div>
              <div className="text-sm text-muted-foreground">{item.note}</div>
            </div>
          </div>
        ))}
      </div>
    </>
  );
}
