"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";
import { AnimatePresence, m, useMotionValueEvent, useScroll } from "motion/react";
import { ArrowRight, ArrowUpRight } from "lucide-react";
import { BrandLogo } from "@/components/brand-logo";
import { landingFontVariables } from "@/components/landing/fonts";
import { landingFooterCols, landingNav } from "@/components/landing/landing-content";
import { EASE_OUT, MotionRoot } from "@/components/landing/motion";
import { useBrand } from "@/context/brand-provider";
import { getAccessToken } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";
import { cn } from "@/lib/utils";

const ANCHOR_BLOCKS: Record<string, string> = {
  "#pricing": "pricing",
  "#faq": "faq",
  "#games": "games",
};

function anchorVisible(blocks: Record<string, boolean>, href: string) {
  const block = ANCHOR_BLOCKS[href];
  return !block || blocks[block] !== false;
}

function footerContacts(t: TranslateFn, links: Record<string, string>) {
  return [
    { key: "telegram", icon: "ri-telegram-line", label: "Telegram", href: links.telegram },
    { key: "discord", icon: "ri-discord-line", label: "Discord", href: links.discord },
    { key: "vk", icon: "", label: "VK", href: links.vk },
    {
      key: "support",
      icon: "ri-customer-service-2-line",
      label: t("landing.footer.support"),
      href: links.support,
    },
    {
      key: "email",
      icon: "ri-mail-line",
      label: links.email ?? "",
      href: links.email ? `mailto:${links.email}` : "",
    },
  ].filter((contact) => Boolean(contact.href));
}

export function LandingPublicLayout({ children }: { children: ReactNode }) {
  const t = useT();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [loggedIn, setLoggedIn] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const [hidden, setHidden] = useState(false);
  const [hovered, setHovered] = useState<string | null>(null);
  const pathname = usePathname();
  const { scrollY } = useScroll();
  const { name: appName, templateBlocks: blocks, links, legal } = useBrand();
  const publishedDocs = new Set((legal?.documents ?? []).map((doc) => doc.kind));
  const legalLinks = [
    { key: "offer", label: t("landing.footer.offer"), href: links.offer || (publishedDocs.has("offer") ? "/legal/offer" : "") },
    { key: "privacy", label: t("landing.footer.privacy"), href: links.privacy || (publishedDocs.has("privacy") ? "/legal/privacy" : "") },
    { key: "cookies", label: t("landing.footer.cookies"), href: publishedDocs.has("cookies") ? "/legal/cookies" : "" },
  ].filter((link) => link.href);
  const company = legal?.company;
  const requisites = company?.inn
    ? [
        company.full_name || company.name,
        `${t("landing.footer.inn")} ${company.inn}`,
        company.ogrn
          ? `${company.ogrn.length === 15 ? t("landing.footer.ogrnip") : t("landing.footer.ogrn")} ${company.ogrn}`
          : "",
        company.address,
        company.email,
        company.phone,
      ]
        .filter(Boolean)
        .join(", ")
    : "";

  const navItems = landingNav(t).filter((item) => anchorVisible(blocks, item.href));
  const footerCols = landingFooterCols(t)
    .map((col) => ({
      ...col,
      links: col.links.filter((link) => anchorVisible(blocks, link.href)),
    }))
    .filter((col) => col.links.length > 0);
  const contacts = footerContacts(t, links);

  useEffect(() => {
    setLoggedIn(!!getAccessToken());
  }, []);

  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (!mobileOpen) return;
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") setMobileOpen(false);
    };
    window.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = previous;
      window.removeEventListener("keydown", onKey);
    };
  }, [mobileOpen]);

  useMotionValueEvent(scrollY, "change", (latest) => {
    const previous = scrollY.getPrevious() ?? 0;
    setScrolled(latest > 8);
    if (latest < 120) {
      setHidden(false);
    } else if (Math.abs(latest - previous) > 4) {
      setHidden(latest > previous);
    }
  });

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

  const headerHidden = hidden && !mobileOpen;

  return (
    <MotionRoot>
      <div
        className={cn(
          landingFontVariables,
          "font-landing relative flex min-h-screen flex-col overflow-x-clip bg-background text-foreground antialiased"
        )}
      >
        <m.header
          initial={false}
          animate={{ y: headerHidden ? "-100%" : 0 }}
          transition={{ duration: 0.4, ease: EASE_OUT }}
          className={cn(
            "sticky top-0 z-50 border-b transition-[background-color,border-color,backdrop-filter] duration-300",
            scrolled || mobileOpen
              ? "border-border bg-background/85 backdrop-blur-xl"
              : "border-transparent bg-background"
          )}
        >
          <div className="mx-auto flex h-16 max-w-[1240px] items-center gap-8 px-5 sm:px-8">
            <Link
              href="/"
              className="flex shrink-0 items-center"
              aria-label={t("landing.header.home_aria")}
            >
              <BrandLogo size="sm" className="h-8 max-w-[176px]" priority />
            </Link>

            <nav
              className="hidden flex-1 items-center gap-0.5 lg:flex"
              onMouseLeave={() => setHovered(null)}
            >
              {navItems.map((item) => {
                const active = !item.href.startsWith("#") && pathname === item.href;
                const content = (
                  <>
                    {hovered === item.href && (
                      <m.span
                        layoutId="landing-nav-hover"
                        className="absolute inset-0 rounded-full bg-accent"
                        transition={{ type: "spring", stiffness: 500, damping: 38 }}
                      />
                    )}
                    <span className="relative">{item.label}</span>
                  </>
                );
                const className = cn(
                  "relative rounded-full px-3.5 py-2 text-[14px] font-medium whitespace-nowrap transition-colors",
                  active || hovered === item.href ? "text-foreground" : "text-muted-foreground"
                );
                return item.href.startsWith("#") ? (
                  <button
                    key={item.label}
                    type="button"
                    onMouseEnter={() => setHovered(item.href)}
                    onFocus={() => setHovered(item.href)}
                    onClick={() => handleNavClick(item.href)}
                    className={className}
                  >
                    {content}
                  </button>
                ) : (
                  <Link
                    key={item.label}
                    href={item.href}
                    onMouseEnter={() => setHovered(item.href)}
                    onFocus={() => setHovered(item.href)}
                    className={className}
                  >
                    {content}
                  </Link>
                );
              })}
            </nav>

            <div className="ml-auto flex shrink-0 items-center gap-2">
              <Link
                href={loggedIn ? "/dashboard" : "/login"}
                className="hidden rounded-full px-3.5 py-2 text-[14px] font-medium text-muted-foreground transition-colors hover:text-foreground sm:inline-flex"
              >
                {loggedIn ? t("landing.header.panel") : t("landing.header.sign_in")}
              </Link>
              <Link
                href={loggedIn ? "/dashboard" : "/register"}
                className="group hidden items-center gap-2 rounded-full bg-primary py-2 pr-2 pl-4 text-[14px] font-semibold text-primary-foreground transition-transform active:scale-[0.97] min-[430px]:inline-flex"
              >
                {loggedIn ? t("landing.header.to_panel") : t("landing.header.create_server")}
                <span className="flex size-6 items-center justify-center rounded-full bg-primary-foreground/15 transition-transform duration-300 group-hover:translate-x-0.5">
                  <ArrowRight className="size-3.5" />
                </span>
              </Link>
              <button
                type="button"
                onClick={() => setMobileOpen((open) => !open)}
                aria-expanded={mobileOpen}
                aria-label={t("landing.header.menu_aria")}
                className="relative flex size-10 items-center justify-center rounded-full transition-colors hover:bg-accent lg:hidden"
              >
                <m.span
                  className="absolute h-[1.5px] w-[18px] rounded-full bg-foreground"
                  animate={mobileOpen ? { rotate: 45, y: 0 } : { rotate: 0, y: -4 }}
                  transition={{ duration: 0.3, ease: EASE_OUT }}
                />
                <m.span
                  className="absolute h-[1.5px] w-[18px] rounded-full bg-foreground"
                  animate={mobileOpen ? { rotate: -45, y: 0 } : { rotate: 0, y: 4 }}
                  transition={{ duration: 0.3, ease: EASE_OUT }}
                />
              </button>
            </div>
          </div>

        </m.header>

          <AnimatePresence>
            {mobileOpen && (
              <m.div
                key="mobile-menu"
                initial={{ opacity: 0, clipPath: "inset(0 0 100% 0)" }}
                animate={{ opacity: 1, clipPath: "inset(0 0 0% 0)" }}
                exit={{ opacity: 0, clipPath: "inset(0 0 100% 0)" }}
                transition={{ duration: 0.45, ease: EASE_OUT }}
                className="fixed inset-x-0 top-16 bottom-0 z-40 overflow-y-auto bg-background lg:hidden"
              >
                <m.nav
                  initial="hidden"
                  animate="shown"
                  exit="hidden"
                  variants={{
                    hidden: { transition: { staggerChildren: 0.03, staggerDirection: -1 } },
                    shown: { transition: { staggerChildren: 0.06, delayChildren: 0.1 } },
                  }}
                  className="flex min-h-full flex-col px-5 pt-6 pb-10 sm:px-8"
                >
                  {navItems.map((item) => {
                    const className =
                      "vx-display flex w-full items-center justify-between border-b border-border py-4 text-left text-[1.7rem] font-semibold tracking-[-0.03em]";
                    return (
                      <m.div
                        key={item.label}
                        variants={{
                          hidden: { opacity: 0, y: 18 },
                          shown: { opacity: 1, y: 0, transition: { duration: 0.5, ease: EASE_OUT } },
                        }}
                      >
                        {item.href.startsWith("#") ? (
                          <button type="button" onClick={() => handleNavClick(item.href)} className={className}>
                            {item.label}
                            <ArrowRight className="size-5 text-muted-foreground" />
                          </button>
                        ) : (
                          <Link href={item.href} onClick={() => setMobileOpen(false)} className={className}>
                            {item.label}
                            <ArrowRight className="size-5 text-muted-foreground" />
                          </Link>
                        )}
                      </m.div>
                    );
                  })}
                  <m.div
                    variants={{
                      hidden: { opacity: 0, y: 18 },
                      shown: { opacity: 1, y: 0, transition: { duration: 0.5, ease: EASE_OUT } },
                    }}
                    className="mt-auto flex flex-col gap-3 pt-10"
                  >
                    {loggedIn ? (
                      <Link
                        href="/dashboard"
                        className="flex items-center justify-center rounded-full bg-primary px-5 py-3.5 text-[15px] font-semibold text-primary-foreground"
                      >
                        {t("landing.header.panel_full")}
                      </Link>
                    ) : (
                      <>
                        <Link
                          href="/register"
                          className="flex items-center justify-center rounded-full bg-primary px-5 py-3.5 text-[15px] font-semibold text-primary-foreground"
                        >
                          {t("landing.header.create_server")}
                        </Link>
                        <Link
                          href="/login"
                          className="flex items-center justify-center rounded-full border border-border px-5 py-3.5 text-[15px] font-medium"
                        >
                          {t("landing.header.sign_in")}
                        </Link>
                      </>
                    )}
                  </m.div>
                </m.nav>
              </m.div>
            )}
          </AnimatePresence>

        <main className="relative flex flex-1 flex-col">{children}</main>

        <footer className="relative overflow-hidden">
          <div className="mx-auto max-w-[1240px] px-5 pt-16 sm:px-8 sm:pt-20">
            <div className="grid gap-12 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)] lg:gap-16">
              <div>
                <BrandLogo size="sm" className="h-7 max-w-[160px]" />
                <p className="mt-5 max-w-[340px] text-[15px] leading-[1.6] text-muted-foreground">
                  {t("landing.footer.tagline")}
                </p>
                {contacts.length > 0 && (
                  <div className="mt-6 flex flex-wrap gap-2">
                    {contacts.map((contact) => (
                      <a
                        key={contact.key}
                        href={contact.href}
                        target={contact.key === "email" ? undefined : "_blank"}
                        rel="noopener noreferrer"
                        aria-label={contact.label}
                        title={contact.label}
                        className="inline-flex h-9 items-center gap-2 rounded-full border border-border px-3.5 text-[13px] text-muted-foreground transition-colors hover:border-foreground hover:text-foreground"
                      >
                        {contact.icon ? <i className={`${contact.icon} text-base`} /> : null}
                        <span className={cn(contact.icon && contact.key !== "email" && "hidden sm:inline")}>
                          {contact.label}
                        </span>
                      </a>
                    ))}
                  </div>
                )}
              </div>

              <div className="grid grid-cols-2 gap-x-6 gap-y-10 sm:grid-cols-4">
                {footerCols.map((col) => (
                  <div key={col.title}>
                    <div className="font-mono text-[12px] text-muted-foreground">{col.title}</div>
                    <ul className="mt-4 flex flex-col gap-2.5">
                      {col.links.map((link) => (
                        <li key={link.label}>
                          {link.href.startsWith("#") ? (
                            <button
                              type="button"
                              onClick={() => handleNavClick(link.href)}
                              className="text-left text-[14.5px] text-foreground/80 transition-colors hover:text-foreground"
                            >
                              {link.label}
                            </button>
                          ) : (
                            <Link
                              href={link.href}
                              className="group inline-flex items-center gap-1 text-[14.5px] text-foreground/80 transition-colors hover:text-foreground"
                            >
                              {link.label}
                              <ArrowUpRight className="size-3.5 -translate-x-1 opacity-0 transition-[opacity,transform] duration-300 group-hover:translate-x-0 group-hover:opacity-100" />
                            </Link>
                          )}
                        </li>
                      ))}
                    </ul>
                  </div>
                ))}
              </div>
            </div>

            <div className="mt-16 flex flex-col gap-4 border-t border-border py-6 text-[12.5px] text-muted-foreground sm:flex-row sm:flex-wrap sm:items-center sm:justify-between">
              <span>
                © {new Date().getFullYear()} {appName}. {t("landing.footer.rights")}
              </span>
              {legalLinks.length > 0 && (
                <div className="flex flex-wrap gap-x-5 gap-y-2">
                  {legalLinks.map((link) => (
                    <a key={link.key} href={link.href} className="transition-colors hover:text-foreground">
                      {link.label}
                    </a>
                  ))}
                </div>
              )}
              {requisites && <p className="w-full leading-relaxed">{requisites}</p>}
            </div>
          </div>
        </footer>
      </div>
    </MotionRoot>
  );
}
