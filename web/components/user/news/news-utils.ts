import DOMPurify from "isomorphic-dompurify";

import { localeTag, t } from "@/lib/i18n";

export function formatNewsDate(value: string | null | undefined) {
  if (!value) return t("news.date.unknown");
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return t("news.date.unknown");
  return d.toLocaleDateString(localeTag());
}

export function formatNewsDateTime(value: string | null | undefined) {
  if (!value) return t("news.date.publish_unknown");
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return t("news.date.publish_unknown");
  return d
    .toLocaleString(localeTag(), {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    })
    .replace(",", "");
}

export function hasHtmlMarkup(content: string) {
  return /<\/?[a-z][\s\S]*>/i.test(content);
}

export function sanitizeNewsHtml(html: string): string {
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: [
      "p", "br", "hr", "b", "strong", "i", "em", "u", "s", "code", "pre",
      "blockquote", "h1", "h2", "h3", "h4", "h5", "h6",
      "ul", "ol", "li", "a", "img", "table", "thead", "tbody", "tr", "th", "td",
      "span", "div",
    ],
    ALLOWED_ATTR: ["href", "title", "target", "rel", "src", "alt", "width", "height"],
    ALLOWED_URI_REGEXP: /^(?:https?:|mailto:|tel:|#|\/)/i,
    FORBID_TAGS: ["script", "style", "iframe", "object", "embed", "form", "input"],
    FORBID_ATTR: ["style", "srcset", "formaction"],
  });
}
