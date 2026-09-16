export type NotificationTarget =
  | { kind: "internal"; path: string }
  | { kind: "external"; url: string };

export function notificationTarget(href: string): NotificationTarget | null {
  const raw = href.trim();
  if (!raw) return null;
  if (raw.startsWith("/") && !raw.startsWith("//")) return { kind: "internal", path: raw };
  if (typeof window === "undefined") return null;
  try {
    const url = new URL(raw, window.location.origin);
    if (url.protocol !== "http:" && url.protocol !== "https:") return null;
    if (url.hostname === window.location.hostname) {
      return { kind: "internal", path: url.pathname + url.search + url.hash };
    }
    return { kind: "external", url: url.toString() };
  } catch {
    return null;
  }
}
