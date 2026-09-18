import { EDIT_PARAM } from "@/lib/site/bridge";

let cached: boolean | null = null;

function detect(): boolean {
  if (typeof window === "undefined" || window.self === window.top) return false;
  try {
    const parent = window.parent.location;
    if (parent.origin !== window.location.origin) return false;
    if (!parent.pathname.startsWith("/admin/template")) return false;
  } catch {
    return false;
  }
  return new URLSearchParams(window.location.search).has(EDIT_PARAM);
}

export function isEditorFrame(): boolean {
  if (typeof window === "undefined") return false;
  if (cached === null) cached = detect();
  return cached;
}
