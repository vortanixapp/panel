export function safeUrl(value: string): boolean {
  if (!value) return true;
  if (value.length > 2000 || /[\\\x00\r\n\t <>"'`]/.test(value)) return false;
  if (value.startsWith("#")) return value.length > 1;
  if (value.startsWith("/")) return !value.startsWith("//");
  try {
    const url = new URL(value);
    if (url.protocol === "http:" || url.protocol === "https:") return Boolean(url.host) && !url.username && !url.password;
    if (url.protocol === "mailto:" || url.protocol === "tel:") return url.pathname.length > 0;
  } catch {
    return false;
  }
  return false;
}

export function safeImageUrl(value: string): boolean {
  if (!value) return true;
  if (/^branding\/[A-Za-z0-9._-]{1,120}$/.test(value)) return !value.includes("..");
  if (!safeUrl(value)) return false;
  return value.startsWith("/") || /^https?:\/\//i.test(value);
}

const VIDEO_HOSTS = new Set([
  "youtube.com",
  "www.youtube.com",
  "m.youtube.com",
  "youtu.be",
  "youtube-nocookie.com",
  "www.youtube-nocookie.com",
  "vk.com",
  "vkvideo.ru",
  "rutube.ru",
]);

export function safeVideoUrl(value: string): boolean {
  if (!value) return true;
  if (!safeUrl(value)) return false;
  try {
    const url = new URL(value);
    return url.protocol === "https:" && VIDEO_HOSTS.has(url.hostname.toLowerCase());
  } catch {
    return false;
  }
}
