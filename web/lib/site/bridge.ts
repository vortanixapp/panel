import type { EditorMode } from "@/lib/site/store";
import type { SiteDocument, SiteZone, Viewer } from "@/lib/site/types";

export const EDIT_PARAM = "vx-edit";

export type BlockAction = "up" | "down" | "hide" | "show" | "duplicate" | "remove";

export type ParentMessage =
  | {
      type: "vx:state";
      document: SiteDocument;
      mode: EditorMode;
      viewer: Viewer | null;
      selected: string | null;
      locale: string;
    }
  | { type: "vx:navigate"; path: string }
  | { type: "vx:scroll"; block: string };

export type FrameMessage =
  | { type: "vx:ready"; path: string }
  | { type: "vx:path"; path: string }
  | { type: "vx:select"; page: string; zone: SiteZone; block: string }
  | { type: "vx:deselect" }
  | { type: "vx:select-section"; page: string; section: string }
  | { type: "vx:insert"; page: string; zone: SiteZone; index: number }
  | { type: "vx:block-action"; page: string; zone: SiteZone; block: string; action: BlockAction }
  | { type: "vx:text"; locale: string; key: string; value: string }
  | { type: "vx:field"; page: string; zone: SiteZone; block: string; path: string; locale: string; value: string }
  | { type: "vx:section"; page: string; section: string; hidden: boolean }
  | { type: "vx:inventory"; path: string; keys: string[]; sections: { id: string; hidden: boolean }[] }
  | { type: "vx:history"; action: "undo" | "redo" | "save" };

export function isMessage<T extends { type: string }>(data: unknown): data is T {
  return Boolean(data) && typeof data === "object" && typeof (data as { type?: unknown }).type === "string" && (data as { type: string }).type.startsWith("vx:");
}

export function editorPreviewUrl(path: string): string {
  const [base, hash = ""] = path.split("#");
  const sep = base.includes("?") ? "&" : "?";
  return `${base}${sep}${EDIT_PARAM}=1${hash ? `#${hash}` : ""}`;
}
