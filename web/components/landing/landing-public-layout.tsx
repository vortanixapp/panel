"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";
import { BrandLogo } from "@/components/brand-logo";
import { useBrand } from "@/context/brand-provider";
import { getAccessToken } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { landingFooterCols, landingNav } from "@/components/landing/landing-content";

function navLinkClass(active: boolean) {
  return `px-3.5 py-2 text-[13.5px] font-medium rounded-lg transition-colors whitespace-nowrap ${
    active
      ? "text-foreground bg-[#151618]"
      : "text-muted-foreground hover:text-foreground hover:bg-[#151618]/70"
  }`;
}

function mobileNavLinkClass(active: boolean) {
  return `flex items-center gap-3 px-4 py-3 text-sm font-medium rounded-xl transition-colors ${
    active
      ? "bg-[#151618] text-foreground"
      : "text-muted-foreground hover:text-foreground hover:bg-muted"
  }`;
}

export function LandingPublicLayout({ children }: { children: ReactNode }) {
  const t = useT();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [loggedIn, setLoggedIn] = useState(false);
  const pathname = usePathname();
  const { name: appName } = useBrand();

  // Списки собираются на каждом рендере: подписи меню и подвала должны
  // меняться вместе с языком, а модульные константы застыли бы на первом.
  const navItems = landingNav(t);
  const footerCols = landingFooterCols(t);

  useEffect(() => {
    setLoggedIn(!!getAccessToken());
  }, []);

  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

  const scrollTo = (id: string) => {
    setMobileOpen(false);
    if (pathname !== "/") {
      window.location.href = `/#${id}`;
      return;
    }
    const isMobileViewport =
      typeof window !== "undefined" &&
      window.matchMedia("(max-width: 768px)").matches;
    document.getElementById(id)?.scrollIntoView({
      behavior: isMobileViewport ? "auto" : "smooth",
    });
  };

  const handleNavClick = (href: string) => {
    if (href.startsWith("#")) {
      scrollTo(href.slice(1));
    } else {
      setMobileOpen(false);
    }
  };

  return (
    <div className="font-landing relative flex min-h-screen flex-col overflow-x-hidden bg-background text-foreground antialiased">
      <header className="sticky top-0 z-50 flex h-16 items-center gap-8 border-b border-border bg-background/88 px-4 backdrop-blur-xl sm:px-10">
        <Link
          href="/"
          className="flex flex-shrink-0 items-center"
          aria-label={t("landing.header.home_aria")}
        >
          <BrandLogo size="sm" className="h-8 max-w-[176px]" priority />
        </Link>

        <nav className="no-scrollbar hidden flex-1 items-center gap-0.5 overflow-x-auto lg:flex">
          {navItems.map((item) =>
            item.href.startsWith("#") ? (
              <button
                key={item.label}
                type="button"
                onClick={() => handleNavClick(item.href)}
                className={navLinkClass(false)}
              >
                {item.label}
              </button>
            ) : (
              <Link
                key={item.label}
                href={item.href}
                className={navLinkClass(pathname === item.href)}
              >
                {item.label}
              </Link>
            )
          )}
        </nav>

        <div className="ml-auto flex flex-shrink-0 items-center gap-2.5">
          {loggedIn ? (
            <Link
              href="/dashboard"
              className="hidden px-2.5 py-2 text-[13.5px] font-medium text-muted-foreground transition-colors hover:text-foreground sm:inline-flex"
            >
              {t("landing.header.panel")}
            </Link>
          ) : (
            <Link
              href="/login"
              className="hidden px-2.5 py-2 text-[13.5px] font-medium text-muted-foreground transition-colors hover:text-foreground sm:inline-flex"
            >
              {t("landing.header.sign_in")}
            </Link>
          )}
          <Link
            href={loggedIn ? "/dashboard" : "/register"}
            className="vx-btn hidden rounded-lg px-4 py-2.5 text-[13.5px] font-semibold min-[430px]:inline-flex"
          >
            {loggedIn
              ? t("landing.header.to_panel")
              : t("landing.header.create_server")}
          </Link>
          <button
            type="button"
            onClick={() => setMobileOpen(!mobileOpen)}
            className="flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground lg:hidden"
            aria-label={t("landing.header.menu_aria")}
          >
            <i
              className={`text-xl transition-transform duration-200 ${mobileOpen ? "ri-close-line rotate-90" : "ri-menu-line"}`}
            />
          </button>
        </div>

        {mobileOpen && (
          <div className="absolute inset-x-0 top-16 border-b border-border bg-background/98 backdrop-blur-xl lg:hidden">
            <div className="space-y-1 p-4">
              {navItems.map((item) =>
                item.href.startsWith("#") ? (
                  <button
                    key={item.label}
                    type="button"
                    onClick={() => handleNavClick(item.href)}
                    className={`w-full text-left ${mobileNavLinkClass(false)}`}
                  >
                    {item.label}
                  </button>
                ) : (
                  <Link
                    key={item.label}
                    href={item.href}
                    className={mobileNavLinkClass(pathname === item.href)}
                  >
                    {item.label}
                  </Link>
                )
              )}
              <div className="my-2 h-px bg-border" />
              {loggedIn ? (
                <Link href="/dashboard" className={mobileNavLinkClass(false)}>
                  {t("landing.header.panel_full")}
                </Link>
              ) : (
                <>
                  <Link href="/login" className={mobileNavLinkClass(false)}>
                    {t("landing.header.sign_in")}
                  </Link>
                  <Link
                    href="/register"
                    className="vx-btn mt-1 flex items-center justify-center gap-2 rounded-xl px-4 py-3 text-sm font-semibold"
                  >
                    {t("landing.header.create_server")}
                  </Link>
                </>
              )}
            </div>
          </div>
        )}
      </header>

      <main className="relative z-10 flex flex-1 flex-col">{children}</main>

      <footer className="border-t border-border px-4 py-12 sm:px-10">
        <div className="mx-auto grid max-w-[1180px] gap-12 lg:grid-cols-[280px_1fr]">
          <div>
            <BrandLogo size="sm" className="h-7 max-w-[160px]" />
          </div>
          <div className="grid grid-cols-2 gap-6 sm:grid-cols-4">
            {footerCols.map((col) => (
              <div key={col.title}>
                <div className="font-mono text-[11px] tracking-[0.08em] text-[var(--vx-ink-faint)] uppercase">
                  {col.title}
                </div>
                <div className="mt-3.5 flex flex-col gap-2.5">
                  {col.links.map((link) =>
                    link.href.startsWith("#") ? (
                      <button
                        key={link.label}
                        type="button"
                        onClick={() => handleNavClick(link.href)}
                        className="text-left text-[13.5px] text-[var(--vx-ink-dim)] transition-colors hover:text-primary"
                      >
                        {link.label}
                      </button>
                    ) : (
                      <Link
                        key={link.label}
                        href={link.href}
                        className="text-[13.5px] text-[var(--vx-ink-dim)] transition-colors hover:text-primary"
                      >
                        {link.label}
                      </Link>
                    )
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
        <div className="mx-auto mt-10 max-w-[1180px] border-t border-border pt-6 text-xs text-muted-foreground/70">
          © {new Date().getFullYear()} {appName}. {t("landing.footer.rights")}
        </div>
      </footer>
    </div>
  );
}
