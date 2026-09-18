export type AppearanceTheme = "dark" | "light" | "system";
export type AppearanceRadius = "standard" | "strict" | "soft";
export type AppearanceFont =
  | "default"
  | "inter"
  | "manrope"
  | "rubik"
  | "montserrat"
  | "ibm-plex-sans";

export type AppearanceAccent = {
  panel: string;
  panel_dark: string;
  landing: string;
  landing_dark: string;
};

export type PublicAppearance = {
  accent: AppearanceAccent;
  theme: AppearanceTheme;
  theme_locked: boolean;
  radius: AppearanceRadius;
  font_panel: AppearanceFont;
  font_landing: AppearanceFont;
  custom_css: string;
  links: Record<string, string>;
};

export type BrandingPayload = {
  brand_name: string;
  panel_name?: string;
  logo_url: string;
  logo_dark_url?: string;
  icon_url?: string;
  primary_color: string;
  user_menu_variant?: "default" | "screenshot";
  appearance?: PublicAppearance;
  whmcs?: {
    order_url: string;
    client_area_url: string;
    orders_only: boolean;
  } | null;
  legal?: {
    documents: { kind: string; version: number; title: string; published_at: string }[];
    company: {
      name: string;
      full_name: string;
      inn: string;
      ogrn: string;
      address: string;
      email: string;
      phone: string;
    };
    cookie_banner: boolean;
    registration: { terms: boolean; personal_data: boolean };
  } | null;
};

export const DEFAULT_PANEL_ACCENT = "#6366f1";
export const BRANDING_GLOBAL = "__VORTANIX_BRANDING__";

const STYLE_ID = "vx-appearance";
const CUSTOM_STYLE_ID = "vx-custom-css";
const HEX = /^#[0-9a-f]{6}$/i;

export const APPEARANCE_FONTS: {
  id: AppearanceFont;
  label: string;
  variable: string;
}[] = [
  { id: "default", label: "", variable: "" },
  { id: "inter", label: "Inter", variable: "--font-inter" },
  { id: "manrope", label: "Manrope", variable: "--font-manrope" },
  { id: "rubik", label: "Rubik", variable: "--font-rubik" },
  { id: "montserrat", label: "Montserrat", variable: "--font-montserrat" },
  { id: "ibm-plex-sans", label: "IBM Plex Sans", variable: "--font-ibm-plex-sans" },
];

const RADIUS_VARS: Record<string, string> = {
  strict: "--radius:0.375rem;--radius-2xl:0.625rem;--radius-3xl:0.875rem;",
  soft: "--radius:0.875rem;--radius-2xl:1.25rem;--radius-3xl:1.75rem;",
};

export function isHexColor(value: string): boolean {
  return HEX.test(value.trim());
}

export function readableTextOn(color: string): string {
  if (!HEX.test(color)) return "#ffffff";
  const channels = [1, 3, 5].map((i) => {
    const c = parseInt(color.slice(i, i + 2), 16) / 255;
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
  });
  const luminance =
    0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
  const contrastWithDark = (luminance + 0.05) / 0.0533;
  const contrastWithWhite = 1.05 / (luminance + 0.05);
  return contrastWithDark > contrastWithWhite ? "#0a0b0d" : "#ffffff";
}

function pickHex(value: string | undefined): string {
  return value && HEX.test(value.trim()) ? value.trim().toLowerCase() : "";
}

function accentVars(color: string): string {
  const fg = readableTextOn(color);
  return (
    `--primary:${color};--primary-foreground:${fg};--ring:${color};` +
    `--sidebar-primary:${color};--sidebar-primary-foreground:${fg};--sidebar-ring:${color};`
  );
}

function fontVariable(id: string | undefined): string {
  return APPEARANCE_FONTS.find((f) => f.id === id)?.variable ?? "";
}

export function appearanceCSS(a: PublicAppearance | undefined | null): string {
  if (!a) return "";
  const panel = pickHex(a.accent?.panel) || DEFAULT_PANEL_ACCENT;
  const panelDark = pickHex(a.accent?.panel_dark) || panel;
  const landing = pickHex(a.accent?.landing);
  const landingDark = pickHex(a.accent?.landing_dark) || landing;

  let css =
    `html:root{${accentVars(panel)}${RADIUS_VARS[a.radius] ?? ""}}` +
    `html.dark{${accentVars(panelDark)}}`;

  if (landing) {
    css +=
      `html .font-landing{${accentVars(landing)}--vx-landing-accent:${landing};--vx-landing-on-accent:${readableTextOn(landing)};}` +
      `html.dark .font-landing{${accentVars(landingDark)}--vx-landing-accent:${landingDark};--vx-landing-on-accent:${readableTextOn(landingDark)};}`;
  }

  const panelFont = fontVariable(a.font_panel);
  if (panelFont) {
    css +=
      `html body{--font-sans:var(${panelFont})}` +
      `html .font-panel{font-family:var(${panelFont});--font-sans:var(${panelFont})}`;
  }
  const landingFont = fontVariable(a.font_landing);
  if (landingFont) {
    css += `html .font-landing{font-family:var(${landingFont});--font-sans:var(${landingFont});--font-display:var(${landingFont})}`;
  }
  return css;
}

function putStyle(id: string, text: string) {
  const existing = document.getElementById(id);
  if (!text) {
    existing?.remove();
    return;
  }
  const el = existing ?? document.createElement("style");
  el.id = id;
  el.textContent = text;
  if (!existing) document.head.appendChild(el);
}

export function setFavicon(url: string) {
  if (!url) return;
  const links = document.querySelectorAll<HTMLLinkElement>("link[rel~='icon']");
  if (links.length === 0) {
    const link = document.createElement("link");
    link.rel = "icon";
    link.href = url;
    document.head.appendChild(link);
    return;
  }
  links.forEach((link) => link.setAttribute("href", url));
}

export function applyAppearance(branding: BrandingPayload | null) {
  if (typeof document === "undefined" || !branding) return;
  const a = branding.appearance;
  putStyle(STYLE_ID, appearanceCSS(a));
  putStyle(CUSTOM_STYLE_ID, a?.custom_css ?? "");
  if (a?.theme_locked) {
    document.documentElement.setAttribute("data-theme-locked", "true");
  } else {
    document.documentElement.removeAttribute("data-theme-locked");
  }
  if (branding.icon_url) setFavicon(branding.icon_url);
}

export function injectedBranding(): BrandingPayload | null {
  if (typeof window === "undefined") return null;
  const value = (window as unknown as Record<string, unknown>)[BRANDING_GLOBAL];
  return value && typeof value === "object" ? (value as BrandingPayload) : null;
}

export function injectedAppearance(): PublicAppearance | null {
  return injectedBranding()?.appearance ?? null;
}

export function rememberBranding(branding: BrandingPayload) {
  if (typeof window === "undefined") return;
  (window as unknown as Record<string, unknown>)[BRANDING_GLOBAL] = branding;
}

const BOOT =
  "(function(css,custom,theme,locked,icon){try{var d=document,r=d.documentElement;" +
  "function put(id,text){var el=d.getElementById(id);if(!text){if(el&&el.parentNode){el.parentNode.removeChild(el)}return}" +
  "if(!el){el=d.createElement('style');el.id=id;d.head.appendChild(el)}el.textContent=text}" +
  `put('${STYLE_ID}',css);put('${CUSTOM_STYLE_ID}',custom);` +
  "if(locked){r.setAttribute('data-theme-locked','true')}else{r.removeAttribute('data-theme-locked')}" +
  "var m=d.cookie.match(/(?:^|; )vite-ui-theme=([^;]*)/);var t=(!locked&&m)?decodeURIComponent(m[1]):theme;" +
  "if(t==='system'){t=window.matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light'}" +
  "if(t==='light'||t==='dark'){r.classList.remove('light','dark');r.classList.add(t)}" +
  "if(icon){var l=d.createElement('link');l.rel='icon';l.href=icon;d.head.appendChild(l)}" +
  "}catch(e){}})";

export function appearanceBootScript(branding: BrandingPayload): string {
  const a = branding.appearance;
  return (
    BOOT +
    "(" +
    [
      JSON.stringify(appearanceCSS(a)),
      JSON.stringify(a?.custom_css ?? ""),
      JSON.stringify(a?.theme ?? "dark"),
      a?.theme_locked ? "true" : "false",
      JSON.stringify(branding.icon_url ?? ""),
    ].join(",") +
    ");"
  );
}
