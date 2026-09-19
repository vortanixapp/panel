import { isStaffRole } from "@/lib/rbac";

const START_PAGES = new Set(["/dashboard", "/servers", "/billing", "/notifications", "/support"]);

export function postLoginPath(role: string, startPage?: string | null): string {
  if (startPage && START_PAGES.has(startPage)) return startPage;
  return isStaffRole(role) ? "/admin/dashboard" : "/dashboard";
}
