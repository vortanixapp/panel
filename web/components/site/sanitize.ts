import DOMPurify from "isomorphic-dompurify";

const VIDEO_EMBED =
  /^https:\/\/(www\.youtube-nocookie\.com\/embed\/|www\.youtube\.com\/embed\/|vk\.com\/video_ext\.php\?|vkvideo\.ru\/video_ext\.php\?|rutube\.ru\/play\/embed\/)/i;

const RICH_TAGS = [
  "p", "br", "hr", "b", "strong", "i", "em", "u", "s", "mark", "small", "sub", "sup", "code", "pre",
  "blockquote", "h2", "h3", "h4", "ul", "ol", "li", "a", "img", "figure", "figcaption",
  "table", "thead", "tbody", "tr", "th", "td", "span", "div",
];

function withLinkHook<T>(run: () => T): T {
  DOMPurify.addHook("afterSanitizeAttributes", (node) => {
    if (node.tagName === "A" && node.getAttribute("target") === "_blank") {
      node.setAttribute("rel", "noopener noreferrer");
    }
  });
  try {
    return run();
  } finally {
    DOMPurify.removeHook("afterSanitizeAttributes");
  }
}

export function sanitizeRichHtml(html: string): string {
  return withLinkHook(() =>
    DOMPurify.sanitize(html, {
      ALLOWED_TAGS: RICH_TAGS,
      ALLOWED_ATTR: ["href", "title", "target", "rel", "src", "alt", "width", "height"],
      FORBID_ATTR: ["style", "srcset", "formaction"],
    })
  );
}

export function sanitizeCustomHtml(html: string): string {
  DOMPurify.addHook("uponSanitizeElement", (node, data) => {
    if (data.tagName !== "iframe") return;
    const element = node as Element;
    if (!VIDEO_EMBED.test(element.getAttribute("src") ?? "")) {
      element.parentNode?.removeChild(element);
    }
  });
  try {
    return withLinkHook(() =>
      DOMPurify.sanitize(html, {
        ADD_TAGS: ["iframe"],
        ADD_ATTR: ["allow", "allowfullscreen", "frameborder", "referrerpolicy", "loading", "target"],
        FORBID_TAGS: ["script", "style", "object", "embed", "form", "input", "button", "textarea", "select", "link", "meta", "base"],
        FORBID_ATTR: ["srcset", "formaction", "action"],
      })
    );
  } finally {
    DOMPurify.removeHook("uponSanitizeElement");
  }
}

export function videoEmbedUrl(raw: string): string {
  let url: URL;
  try {
    url = new URL(raw);
  } catch {
    return "";
  }
  if (url.protocol !== "https:") return "";
  const host = url.hostname.toLowerCase();
  if (host === "youtu.be") {
    const id = url.pathname.slice(1).split("/")[0];
    return id ? `https://www.youtube-nocookie.com/embed/${encodeURIComponent(id)}` : "";
  }
  if (host.endsWith("youtube.com") || host.endsWith("youtube-nocookie.com")) {
    const parts = url.pathname.split("/").filter(Boolean);
    const id =
      url.searchParams.get("v") ??
      (parts[0] === "embed" || parts[0] === "shorts" || parts[0] === "live" ? parts[1] : undefined);
    return id ? `https://www.youtube-nocookie.com/embed/${encodeURIComponent(id)}` : "";
  }
  if (host === "vk.com" || host === "vkvideo.ru") {
    if (url.pathname === "/video_ext.php") return url.toString();
    const match = url.pathname.match(/video(-?\d+)_(\d+)/) ?? url.searchParams.get("z")?.match(/video(-?\d+)_(\d+)/);
    return match ? `https://vk.com/video_ext.php?oid=${match[1]}&id=${match[2]}&hd=2` : "";
  }
  if (host === "rutube.ru") {
    const match = url.pathname.match(/\/(?:video|play\/embed)\/([a-z0-9]+)/i);
    return match ? `https://rutube.ru/play/embed/${match[1]}` : "";
  }
  return "";
}
