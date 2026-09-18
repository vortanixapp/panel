import type { SiteDocument } from "@/lib/site/types";

export function parseSitePayload(raw: unknown): SiteDocument {
  if (!raw || typeof raw !== "object") return {};
  const doc = (raw as { document?: unknown }).document;
  return doc && typeof doc === "object" && !Array.isArray(doc) ? (doc as SiteDocument) : {};
}
