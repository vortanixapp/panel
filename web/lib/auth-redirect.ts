import { isStaffRole } from "@/lib/rbac";

export function postLoginPath(role: string): string {
  return isStaffRole(role) ? "/admin/dashboard" : "/dashboard";
}
